// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package jsonpathtree

import (
	"strings"
	"strconv"
	"math"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/cockroach/pkg/util/json"
)

type ExecCtx struct {
	// The json document on which to execute the json path query
	rootJson json.JSON
	// Stack of executed jsonpath expressions to resolve @ expressions in the query
	execScope []json.JSON
	// Optional varialbles provided by user for substitutions
	vars json.JSON
	// Whether to execute the query in lax mode (dealing with errors)
	isLax bool
}

func ExecJsonPath(root json.JSON, expr JsonPathExpr, vars json.JSON, isLax bool) (json.JSON, error) {
	execScope := make([]json.JSON, 0)
	ctx := ExecCtx{
		rootJson: root,
		execScope: execScope,
		vars: vars,
		isLax: isLax,
	}

	res, err := root.Eval(*ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *DollarSymbol) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	return ctx.rootJson, nil
}

func (d *AtSymbol) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	if len(ctx.execScope) == 0 {
		return nil, errors.New("jsonpath: @ can only be used in an expression with an array")
	}
	return ctx.execScope[len(ctx.execScope) - 1], nil
}

func (d *VariableExpr) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	res, _ := ctx.vars.FetchValKey(d.name)
	if res == nil {
		return json.NullJSONValue, nil
	}

	return res, nil
}

func (d *DotAccessExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {
	name, err := d.name.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return j.FetchVal(name.val), nil
}

func (d *SubscriptAccessExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {
	if d.Asterisk {
		return j, nil
	}

	res := make([]json.JSON, 0)
	for _, subscript := range d.Subscripts {

		if subscript.Slice {
			end, err := subscript.End.Eval(ctx)
			if err != nil {
				return nil, err
			}

			begin, err := subscript.Begin.Eval(ctx)
			if err != nil {
				return nil, err
			}

			endNumeric, success := json.AsJsonNumber(end)
			if !success {
				return nil, errors.New("jsonpath: subscript end is not a number")
			}

			beginNumeric, success := json.AsJsonNumber(begin)
			if !success {
				return nil, errors.New("jsonpath: subscript begin is not a number")
			}

			endIdx, err := endNumeric.Int64()
			if err != nil {
				return nil, err
			}

			beginIdx, err := beginNumeric.Int64()
			if err != nil {
				return nil, err
			}


			// for i := beginIdx; i <= endIdx; i++ {

		}


		begin, err := subscript.Begin.Eval(ctx)
		if err != nil {
			return nil, err
		}
	}

	begin, err := d.begin.Eval(ctx)
	if err != nil {
		return nil, err
	}

	end, err := d.end.Eval(ctx)
	if err != nil {
		return nil, err
	}
}

func (d *WildcardAccessExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {
	return j, nil
}

func (d *RecursiveWildcardAccessExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {
	return j, nil
}

func (d *FilterExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {

	if json.IsJsonArray(j) {
		arr, _ := json.AsJsonArray(currRoot)
		res := make([]json.JSON, 0)
		for _, j := range arr {
			ctx.execScope = append(ctx.execScope, j)
			predRes, err := d.pred.EvalPredicate(ctx)
			ctx.execScope = ctx.execScope[:len(ctx.execScope) - 1]
			if err != nil {
				return nil, err
			}
			if predRes {
				res = append(res, j)
			}
		}

		switch res.len() {
		case 0:
			return json.NullJSONValue, nil
		case 1:
			return res[0], nil
		default:
			return res, nil
		}
	}

	ctx.execScope = append(ctx.execScope, j)
	predRes, err := d.pred.EvalPredicate(ctx)
	ctx.execScope = ctx.execScope[:len(ctx.execScope) - 1]
	if err != nil {
		return nil, err
	}
	if predRes {
		return currRoot, nil
	}
	return json.NullJSONValue, nil
}

func (d *BinaryOperatorExpr) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	left, err := d.left.Eval(ctx)
	if err != nil {
		return nil, err
	}
	right, err := d.right.Eval(ctx)
	if err != nil {
		return nil, err
	}

	left_num, success := json.AsJsonNumber(left)
	if !success {
		return nil, errors.New("jsonpath: left operand is not a number")
	}
	right_num, success := json.AsJsonNumber(right)
	if !success {
		return nil, errors.New("jsonpath: right operand is not a number")
	}

	switch d.OperatorType {
	case PlusBinOp:
		return left_num.Add(right_num)
	case MinusBinOp:
		return left_num.Sub(right_num)
	case MultBinOp:
		return left_num.Mul(right_num)
	case DivBinOp:
		return left_num.Div(right_num)
	case ModBinOp:
		return left_num.Mod(right_num)
	}
}

func (d *BinaryLogicExpr) EvalPredicate(ctx *JsonPathEvalCtx) (json.JSON, error) {
	left, err := d.left.EvalPredicate(ctx)
	if err != nil {
		return nil, err
	}
	right, err := d.right.EvalPredicate(ctx)
	if err != nil {
		return nil, err
	}

	swtich d.LogicType {
	case AndBinLogic:
		return (left && right), nil
	case OrBinLogic:
		return (left || right), nil
	default:
		return nil, errors.New("jsonpath: unknown logic type")
	}
}

func (d *NotPredicate) EvalPredicate(ctx *JsonPathEvalCtx) (json.JSON, error) {
	predRes, err := d.expr.EvalPredicate(ctx)
	if err != nil {
		return nil, err
	}
	return !predRes, nil
}

func (d *ParenExpr) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	res, err := d.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *UnaryOperatorExpr) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	res, err := d.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}

	switch d.OperatorType {
	case Uminus:
		if 
	case Uplus:
	default:
		return nil, errors.New("jsonpath: unknown unary operator")
	}
}

func (d *FunctionExpr) EvalAccess(ctx *JsonPathEvalCtx, j json.JSON) (json.JSON, error) {
	swtich d.FunctionType {
	case TypeFunction:
		return json.NewJsonString(j.Type().String()), nil
	case SizeFunction:
		if json.IsJsonArray(j) {
			arr, _ := json.AsJsonArray(j)
			return json.FromInt(arr.len()), nil
		}
		return json.FromInt(1), nil
	case DoubleFunction:
		if json.IsJsonNumber(j) {
			return j, nil
		} else if json.IsJsonString(j) {
			res, err := strconv.ParseFloat(j.String(), 64)
			if err != nil {
				return nil, err
			}
			return json.FromFloat64(res), nil
		}
	case CielingFunction:
		if json.IsJsonNumber(j) {
			res, _ := json.AsJsonNumber(j)
			return json.FromFloat64(math.Ceil(res.Float64())), nil
		}
		return nil, errors.New("jsonpath: Ceiling function must be used on a number")
	case FloorFunction:
		if json.IsJsonNumber(j) {
			res, _ := json.AsJsonNumber(j)
			return json.FromFloat64(math.Floor(res.Float64())), nil
		}
		return nil, errors.New("jsonpath: Floor function must be used on a number")
	case AbsFunction:
		if json.IsJsonNumber(j) {
			res, _ := json.AsJsonNumber(j)
			return json.FromFloat64(math.Abs(res.Float64())), nil
		}
		return nil, errors.New("jsonpath: Abs function must be used on a number")
	case DateTimeFunction:
		return nil, errors.New("jsonpath: DateTime function is not implemented as of now")
	case KeyValueFunction:
		if json.IsJsonObject(j) {
			jsonObj, _ := json.AsJsonObject(j)
			resBuilder := json.NewObjectBuilder(len(jsonObj))

		}
	default:
		return nil, errors.New("jsonpath: unknown function")
	}
}

func (d *ExistsExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	res, err := d.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	
	if res == json.NullJSONValue {
		return false, nil
	}

	return true, nil
}

func (d *LikeRegexExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	expr, err := d.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	exprStr, success := json.AsJsonString(expr)
	if !success {
		return nil, errors.New("jsonpath: expression is not a string")
	}
	return d.pattern.Match([]byte(exprStr)), nil
}

func (d *StartsWithExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	left, err := d.left.Eval(ctx)
	if err != nil {
		return nil, err
	}
	right, err := d.right.Eval(ctx)
	if err != nil {
		return nil, err
	}

	leftStr, success := json.AsJsonString(left)
	if !success {
		return nil, errors.New("jsonpath: left operand is not a string")
	}
	rightStr, success := json.AsJsonString(right)
	if !success {
		return nil, errors.New("jsonpath: right operand is not a string")
	}

	return strings.HasPrefix(leftStr, rightStr), nil
}

func (d *IsUnknownExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	res, err := d.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if res == json.NullJSONValue {
		return true, nil
	}

	return false, nil
}

func (d *JsonPathString) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	return json.NewJsonString(d.val), nil
}

func (d *JsonPathNumeric) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	res, err := json.FromFloat64(d.val)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *JsonPathNull) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	return json.NullJSONValue, nil
}

func (d *JsonPathBool) Eval(ctx *JsonPathEvalCtx) (json.JSON, error) {
	if d.val {
		return json.TrueJSONValue, nil
	}
	return json.FalseJSONValue, nil
}

func(d *BinaryPredicateExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	left, err := d.left.EvalPredicate(ctx)
	if err != nil {
		return false, err
	}
	right, err := d.right.EvalPredicate(ctx)
	if err != nil {
		return false, err
	}

	cmpRes, err := left.Compare(right)
	if err != nil {
		return false, err
	}

	switch d.PredicateType {
	case EqBinOp:
		return cmpRes == 0, nil
	case NeqBinOp:
		return cmpRes != 0, nil
	case GtBinOp:
		return cmpRes > 0, nil
	case GteBinOp:
		return cmpRes >= 0, nil
	case LtBinOp:
		return cmpRes < 0, nil
	case LteBinOp:
		return cmpRes <= 0, nil
	default:
		return false, errors.New("jsonpath: unknown predicate")
	}
}

func (d *ParenPredExpr) EvalPredicate(ctx *JsonPathEvalCtx) (bool, error) {
	predRes, err := d.expr.EvalPredicate(ctx)
	if err != nil {
		return false, err
	}
	return predRes, nil
}



