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

type executor func(ctx context.Context, e *evaluator, container tree.Datum, subscripts tree.ArraySubscripts, update tree.Datum) (tree.Datum, error)

func arrayUpdate(
	ctx context.Context,
	e *evaluator,
	container tree.Datum,
	subscripts tree.ArraySubscripts,
	update tree.Datum,
) (tree.Datum, error) {
	res := tree.MustBeDArray(container)
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
			return mutatedArray, nil
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
				// then postgres will automatically extend to account for the new index assuing the
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
	e   *evaluator,
	container   tree.Datum,
	subscripts  tree.ArraySubscripts,
	update      tree.Datum,
) (tree.Datum, error) {
	j := tree.MustBeDJSON(container)
	curr := j.JSON
	val, err := update.Eval(ctx, e)
	if err != nil {
		return nil, err
	}
	v := tree.MustBeDJSON(val)
	to := v.JSON
	return recursiveUpdate(ctx, e, curr, subscripts, to)
}

func recursiveUpdate(
	ctx context.Context,
	e *evaluator,
	container json.JSON,
	subscripts tree.ArraySubscripts,
	to json.JSON,
) (tree.Datum, error) {
	switch len(subscripts) {
	case 0:
		return tree.NewDJSON(container), nil
	case 1:
		return tree.NewDJSON(container), nil
	default:
		currSub := subscripts[0]
		if currSub.Slice {
			return tree.DNull, errors.AssertionFailedf("jsonb subscripts do not support slices")
		}
		field, err := currSub.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return tree.DNull, err
		}
		if field == tree.DNull {
			return tree.DNull, nil
		}
		curr := container
		switch field.ResolvedType().Family() {
		case types.StringFamily:
			if curr, err = curr.FetchValKeyOrIdx(string(tree.MustBeDString(field))); err != nil {
				return tree.DNull, err
			}
		case types.IntFamily:
			if curr, err = curr.FetchValIdx(int(tree.MustBeDInt(field))); err != nil {
				return tree.DNull, err
			}
		default:
			return tree.DNull, errors.AssertionFailedf("unsupported subscription type, should have been rejected during planning")
		}
		if curr == nil {
			return tree.NewDJSON(container), nil
		}
		sub, err := recursiveUpdate(ctx, e, curr, subscripts[1:], to)
		if err != nil {
			return tree.DNull, err
		}
		newJSON, err := setValKeyOrIdx(ctx, e, container, subscripts[0], tree.MustBeDJSON(sub).JSON)
		if err != nil {
			return tree.DNull, err
		}
		return tree.NewDJSON(newJSON), nil
	}
}

func setValKeyOrIdx(
	ctx context.Context,
	e *evaluator,
	container json.JSON,
	subscript *tree.ArraySubscript,
	to json.JSON,
) (json.JSON, error) {
	switch v := container.Type(); v {
	case json.ObjectJSONType:
		vObj := json.GetJsonObject(container)
		field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return nil, err
		}
		if field == tree.DNull {
			return container, nil
		}
		return vObj.SetKey(string(tree.MustBeDString(field)), to, true)
	case json.ArrayJSONType:
		vArr := json.GetJsonArray(container)
		field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return nil, err
		}
		if field == tree.DNull {
			return container, nil
		}
		idx := int(tree.MustBeDInt(field))
		if idx < 0 {
			idx = len(vArr) + idx
		}
		var result []json.JSON
		if idx < 0 {
			result = make([]json.JSON, len(vArr)+1)
			copy(result[1:], vArr)
			result[0] = to
		} else if idx >= len(vArr) {
			result = make([]json.JSON, len(vArr)+1)
			copy(result, vArr)
			result[len(result)-1] = to
		} else {
			result = make([]json.JSON, len(vArr))
			copy(result, vArr)
			result[idx] = to
		}
		return json.ConvertToJsonArray(result), nil
	}
	return container, nil
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