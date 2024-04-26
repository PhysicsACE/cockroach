// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package eval

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
	"github.com/cockroachdb/cockroach/pkg/util/intsets"
	"github.com/cockroachdb/cockroach/pkg/util/json"
	"github.com/cockroachdb/errors"
)

// This file houses the fetch and assignment executors for the various container types in CockroachDB

type executor func(ctx context.Context, e *evaluator, container tree.Datum, subscripts tree.ArraySubscripts, update tree.Datum, containerType *types.T) (tree.Datum, error)

func arrayUpdate(
	ctx context.Context,
	e *evaluator,
	container tree.Datum,
	subscripts tree.ArraySubscripts,
	update tree.Datum,
	containerType *types.T,
) (tree.Datum, error) {
	var res *tree.DArray
	if container == tree.DNull {
		res = tree.NewDArray(containerType.ArrayContents())
	} else {
		res = tree.MustBeDArray(container)
	}
	maximum := func(a int, b int) int {
		if a <= b {
			return b
		}
		return a
	}

	// minimum := func(a int, b int) int {
	// 	if a <= b {
	// 		return a
	// 	}
	// 	return b
	// }

	arr := tree.MustBeDArray(res)
	for _, s := range subscripts {
		var subscriptBeginIdx int
		var subscriptEndIdx int
		val := tree.MustBeDArray(update)
		if s.Slice {
			beginDatum, err := s.Begin.(tree.TypedExpr).Eval(ctx, e)
			if err != nil {
				return tree.DNull, err
			}
			if beginDatum == tree.DNull {
				subscriptBeginIdx = 1
			} else {
				subscriptBeginIdx = int(tree.MustBeDInt(beginDatum))
			}
			endDatum, err := s.End.(tree.TypedExpr).Eval(ctx, e)
			if err != nil {
				return tree.DNull, err
			}
			if endDatum == tree.DNull {
				subscriptEndIdx = arr.Len()
			} else {
				subscriptEndIdx = int(tree.MustBeDInt(endDatum))
			}

			// mutatedArray := tree.NewDArray(arr.ParamTyp)
			if subscriptEndIdx < subscriptBeginIdx || (subscriptBeginIdx < 0 && subscriptEndIdx < 0) {
				return tree.DNull, errors.AssertionFailedf("invalid subscript range")
			}

			var currIndicies intsets.Fast
			if arr.Len() > 0 {
				currIndicies.AddRange(1, arr.Len())
			}
			var updateIndicies intsets.Fast
			updateIndicies.AddRange(subscriptBeginIdx, subscriptEndIdx)
			if updateIndicies.Len() > val.Len() {
				return tree.DNull, errors.AssertionFailedf("source array too small")
			}
			mutatedArray := tree.NewDArray(arr.ParamTyp)
			for i := 1; i <= maximum(arr.Len(), subscriptEndIdx); i++ {
				if updateIndicies.Contains(i) {
					if err := mutatedArray.Append(val.Array[i - 1]); err != nil {
						return tree.DNull, err
					}
				} else if currIndicies.Contains(i) {
					if err := mutatedArray.Append(arr.Array[i - 1]); err != nil {
						return tree.DNull, err
					}
				} else {
					if err := mutatedArray.Append(tree.DNull); err != nil {
						return tree.DNull, err
					}
				}
			}
			arr = mutatedArray
			continue
		}

		beginDatum, err := s.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return tree.DNull, err
		}
		subscriptBeginIdx = int(tree.MustBeDInt(beginDatum))
		if arr.FirstIndex() == 0 {
			subscriptBeginIdx++
		}
		// Postgres extends the array and fills indicies with null if update subscript
		// is greater than the current length of the column array
		if subscriptBeginIdx < 1 {
			return tree.DNull, nil
		}
		var currIndicies intsets.Fast
		if arr.Len() > 0 {
			currIndicies.AddRange(1, arr.Len())
		}
		mutatedArray := tree.NewDArray(arr.ParamTyp)
		for i := 1; i <= maximum(arr.Len(), subscriptBeginIdx); i++ {
			if i == subscriptBeginIdx {
				if err := mutatedArray.Append(val); err != nil {
					return tree.DNull, err
				}
			} else if currIndicies.Contains(i) {
				if err := mutatedArray.Append(arr.Array[i - 1]); err != nil {
					return tree.DNull, err
				}
			} else {
				// For ARRAY updates, if the index is at an index greater than the current length,
				// then postgres will automatically extend to account for the new index assuming the
				// user if always correct. Thus, if the provided index is not inside the current length
				// and not the intended mutation, we add null values. 
				if err := mutatedArray.Append(tree.DNull); err != nil {
					return tree.DNull, err
				}
			}
		}
		arr = mutatedArray
	}

	return arr, nil

}

func jsonUpdate(
	ctx         context.Context,
	e           *evaluator,
	container   tree.Datum,
	subscripts  tree.ArraySubscripts,
	update      tree.Datum,
	_ *types.T,
) (tree.Datum, error) {
	v := tree.MustBeDJSON(update)
	to := v.JSON
	if container == tree.DNull {
		new, err := constructPath(ctx, e, subscripts, to)
		if err != nil {
			return tree.DNull, err
		}
		return tree.NewDJSON(new), nil
	}
	j := tree.MustBeDJSON(container)
	curr := j.JSON
	updated, err := setValKeyOrIdx(ctx, e, curr, subscripts[0], to)
	if err != nil {
		return tree.DNull, err
	}
	return tree.NewDJSON(updated), nil
}

func constructPath(
	ctx context.Context,
	e *evaluator,
	subscripts tree.ArraySubscripts,
	to json.JSON,
) (json.JSON, error) {
	subscript := subscripts[0]
	if subscript.Slice {
		return nil, errors.AssertionFailedf("unsupported subscription type, should have been rejected during planning")
	}
	field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
	if err != nil {
		return nil, err
	}
	var container json.JSON
	switch field.ResolvedType().Family() {
	case types.StringFamily:
		container = json.EmptyJSONObject()
	case types.ArrayFamily:
		container = json.EmptyJSONArray()
	default:
		return nil, errors.AssertionFailedf("unsupported subscription type, should have been rejected during planning")
	}
	return setValKeyOrIdx(ctx, e, container, subscript, to)
}

func setValKeyOrIdx(
	ctx context.Context,
	e *evaluator,
	container json.JSON,
	subscript *tree.ArraySubscript,
	to json.JSON,
) (json.JSON, error) {
	if json.IsEncoded(container) {
		n, err := json.GetJsonDecoded(container)
		if err != nil {
			return nil, err
		}
		return setValKeyOrIdx(ctx, e, n, subscript, to)
	}
	switch v := container.Type(); v {
		case json.ObjectJSONType:
			vObj := json.GetJsonObject(container)
			field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
			if err != nil {
				return nil, err
			}
			if field == tree.DNull {
				return nil, errors.AssertionFailedf("Subscription fields for JSON must be type int or string, aborting transaction")
			}
			return vObj.SetKey(string(tree.MustBeDString(field)), to, true)
		case json.ArrayJSONType:
			vArr := json.GetJsonArray(container)
			field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
			if err != nil {
				return nil, err
			}
			if field == tree.DNull {
				return nil, errors.AssertionFailedf("Subscription fields for Array must be type int, aborting transaction")
			}
			idx := int(tree.MustBeDInt(field))
			if idx < 0 {
				idx = len(vArr) + idx
				if idx < 0 {
					return nil, errors.AssertionFailedf("Path element is out of range, aborting transaction")
				}
			}
			var result []json.JSON
			if idx >= len(vArr) {
				result = make([]json.JSON, idx + 1)
				copy(result, vArr)
				for i := len(vArr); i < idx; i++ {
					result[i] = json.NullJSONValue
				}
				result[idx] = to
			} else {
				result = make([]json.JSON, len(vArr))
				copy(result, vArr)
				result[idx] = to
			}
			return json.ConvertToJsonArray(result), nil
		default:
			return nil, errors.AssertionFailedf("Cannot replace existing key")
	}
}

func FetchUpdateExecutor(typ *types.T) (executor, error) {
	switch typ.Family() {
	case types.ArrayFamily:
		return arrayUpdate, nil
	case types.JsonFamily:
		return jsonUpdate, nil
	}
	return nil, errors.AssertionFailedf("unsupported feature should have been rejected during planning")
}