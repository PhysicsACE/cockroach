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

	// "github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgcode"
	// "github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	// "github.com/cockroachdb/cockroach/pkg/sql/sem/tree/treecmp"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
	// "github.com/cockroachdb/cockroach/pkg/util/intsets"
	"github.com/cockroachdb/cockroach/pkg/util/json"
	"github.com/cockroachdb/errors"
)

type execFunc func(context.Context, *evaluator, tree.Datum, tree.ArraySubscripts, tree.TypedExpr) (tree.Datum, error)

// func arrayUpdate(ctx context.Context, e *evaluator, container tree.Datum, subscripts tree.ArraySubscripts, value tree.TypedExpr) (tree.Datum, error) {
// 	res := tree.MustBeDArray(container)
// 	val := value.(tree.TypedExpr).Eval(ctx, e)
// 	maximum := func(a int, b int) int {
// 		if a <= b {
// 			return b
// 		}
// 		return a
// 	}

// 	minimum := func(a int, b int) int {
// 		if a <= b {
// 			return a
// 		}
// 		return b
// 	}
// 	for _, s := range subscripts {
// 		if s.Slice {
// 			var subscriptBeginIdx int
// 			var subscriptEndIdx int
// 			arr := tree.MustBeDArray(res)
// 			val = tree.MustBeDArray(val)

// 			beginDatum, err := s.Begin.(tree.TypedExpr).Eval(ctx, e)
// 			if err != nil {
// 				return tree.DNull, err
// 			}
// 			if beginDatum == tree.DNull {
// 				subscriptBeginIdx = 1
// 			} else {
// 				subscriptBeginIdx = int(tree.MustBeDInt(beginDatum))
// 			}
// 			endDatum, err := s.End.(tree.TypedExpr).Eval(ctx, e)
// 			if err != nil {
// 				return tree.DNull, err
// 			}
// 			if endDatum == tree.DNull {
// 				subscriptEndIdx = arr.Len()
// 			} else {
// 				subscriptEndIdx = int(tree.MustBeDInt(endDatum))
// 			}

// 			mutatedArray := tree.NewDArray(arr.ParamTyp)
// 			if subscriptEndIdx < subscriptBeginIdx || (subscriptBeginIdx < 0 && subscriptEndIdx < 0) {
// 				return tree.DNull
// 			}

// 			var currIndicies intsets.Fast
// 			if arr.Len() > 0 {
// 				currIndicies.AddRange(1, arr.Len())
// 			}
// 			var updateIndicies intsets.Fast
// 			updateIndicies.AddRange(subscriptBeginIdx, subscriptEndIdx)
// 			if updateIndicies.Len() > val.Len() {
// 				return tree.DNull, errors.AssertionFailedf("source array too small")
// 			}
// 			mutatedArray := tree.NewDArray(arr.ParamTyp)
// 			for i := 1; i <= maximum(arr.Len(), subscriptEndIdx); i++ {
// 				if updateIndicies.Contains(i) {
// 					if err := mutatedArray.Append(val.Array[i - 1]); err != nil {
// 						return tree.DNull
// 					}
// 				} else if currIndicies.Contains(i) {
// 					if err := mutatedArray.Append(arr.Array[i - 1]); err != nil {
// 						return tree.DNull
// 					}
// 				} else {
// 					if err := mutatedArray.Append(tree.DNull); err != nil {
// 						return tree.DNull
// 					}
// 				}
// 			}
// 			return mutatedArray
// 		}

// 		beginDatum, err := s.Begin.(tree.TypedExpr).Eval(ctx, e)
// 		if err != nil {
// 			return tree.DNull
// 		}
// 		subscriptBeginIdx = int(tree.MustBeDInt(beginDatum))
// 		if arr.FirstIndex() == 0 {
// 			subscriptBeginIdx++
// 		}
// 		// Postgres extends the array and fills indicies with null if update subscript
// 		// is greater than the current length of the column array
// 		if subscriptBeginIdx < 1 {
// 			return tree.DNull
// 		}
// 		var currIndicies intsets.Fast
// 		if arr.Len() > 0 {
// 			currIndicies.AddRange(1, arr.Len())
// 		}
// 		mutatedArray := tree.NewDArray(arr.ParamTyp)
// 		for i := 1; i <= maximum(arr.Len(), subscriptBeginIdx); i++ {
// 			if i == subscriptBeginIdx {
// 				if err := mutatedArray.Append(val); err != nil {
// 					return tree.DNull
// 				}
// 			} else if currIndicies.Contains(i) {
// 				if err := mutatedArray.Append(arr.Array[i - 1]); err != nil {
// 					return tree.DNull
// 				}
// 			} else {
// 				// For ARRAY updates, if the index is at an index greater than the current length,
// 				// then postgres will automatically extend to account for the new index assuing the
// 				// user if always correct. Thus, if the provided index is not inside the current length
// 				// and not the intended mutation, we add null values. 
// 				if err := mutatedArray.Append(tree.DNull); err != nil {
// 					return tree.DNull
// 				}
// 			}
// 		return mutatedArray
// 	}
// }

func jsonUpdate(ctx context.Context, e *evaluator, container tree.Datum, subscripts tree.ArraySubscripts, value tree.TypedExpr) (tree.Datum, error) {
	j := tree.MustBeDJSON(container)
	curr := j.JSON
	val, err := value.Eval(ctx, e)
	if err != nil {
		return tree.DNull, err
	}
	v := tree.MustBeDJSON(val)
	to := v.JSON
	return recursiveUpdate(curr, subscripts, to, ctx, e)
}

func setValKeyOrIdx(j json.JSON, subscript *tree.ArraySubscript, to json.JSON, ctx context.Context, e *evaluator) (json.JSON, error) {
	switch v := j.Type(); v {
	case json.ObjectJSONType:
		vObj := json.GetJsonObject(j)
		field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return nil, err
		}
		if field == tree.DNull {
			return j, nil
		}
		return vObj.SetKey(string(tree.MustBeDString(field)), to, true)
	case json.ArrayJSONType:
		vArr := json.GetJsonArray(j)
		field, err := subscript.Begin.(tree.TypedExpr).Eval(ctx, e)
		if err != nil {
			return nil, err
		}
		if field == tree.DNull {
			return j, nil
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
	return j, nil
}

func recursiveUpdate(container json.JSON, subscripts tree.ArraySubscripts, value json.JSON, ctx context.Context, e *evaluator) (tree.Datum, error) {
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
		sub, err := recursiveUpdate(curr, subscripts[1:], value, ctx, e)
		if err != nil {
			return tree.DNull, err
		}
		newJSON, err := setValKeyOrIdx(container, subscripts[0], tree.MustBeDJSON(sub).JSON, ctx, e)
		if err != nil {
			return tree.DNull, err
		}
		return tree.NewDJSON(newJSON), nil
	}
}

func GetAssignExecutor(container tree.Datum) (execFunc, error) {
	switch container.ResolvedType().Family() {
	case types.JsonFamily:
		return jsonUpdate, nil
	default:
		return nil, errors.AssertionFailedf("Unsupported feature should have been rejected during planning")
	}
}

// func GetFetchExecutor(
// 	ctx context.Context, e *evaluator, container tree.Datum,
// ) (tree.Datum, error) {
// 	switch container.ResolvedType().Family() {
// 	default:
// 		return nil, errors.AssertionFailedf("Unsupported feature should have been rejected during planning")
// 	}
// }

// type SubscriptionRoutine struct {
// 	// The datum value of the column
// 	Expr      tree.Datum
// 	// All subscription paths that need to be updated
// 	Paths     []tree.ArraySubscripts
// 	// Associated values for paths for update
// 	Values    []tree.TypedExpr
// 	// The function that performs the update for a given path and value pair
// 	// Each supported container type should implement it's own executor function
// 	// to support subscription updates inside UPDATE statements
// 	Executor  execFunc
// 	// Context required to perform expression evaluations
// 	context   context.Context
// 	// Evaluator required to perform expression evaluations
// 	e         *evaluator
// }

// func (r *SubscriptionRoutine) init(ctx context.Context, e *evaluator, expr tree.Datum, paths []tree.ArraySubscripts, values []tree.TypedExpr) {
// 	r.context = ctx
// 	r.e = e 
// 	r.Paths = paths
// 	r.Values = values

// 	switch container.ResolvedType().Family() {
// 	case types.JsonFamily:
// 		r.Expr = tree.MustBeDJSON(expr)
// 		r.Executor = jsonUpdate
// 	default:
// 		return
// 	}
// }

// func (r *SubscriptionRoutine) GetResult() tree.Datum {
// 	return r.Expr
// }

// // Since the same column can be mutated multiple times with an update statement,
// // we iterate over the specified paths and their respective values compute all mutations
// // for a given container column before updating its value in the new row. We do this
// // by incrementally applying the individual updates and updating the Expr field so that
// // by the end of the iterations, the value of Expr represents that final updated container
// func (r *SubscriptionRoutine) Execute() (tree.Datum, error) {
// 	for i := range len(r.Paths) {
// 		if r.Expr, err = r.Executor(r.context, r.e, r.Expr, r.Paths[i], r.Values[i]); err != nil {
// 			return nil, err
// 		}
// 	}
// 	return r.Expr, nil
// }