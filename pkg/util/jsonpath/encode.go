// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package jsonpath

import (
	"github.com/cockroachdb/apd/v3"
	"github.com/cockroachdb/cockroach/pkg/util/encoding"
	"github.com/cockroachdb/errors"
)

type headerType uint32
const baseTag = 0x00000000
const incrementor = 0x01000000
const containerHeaderTypeMask = 0xFF000000
const containerLengthMask = 0x00FFFFFF

const (
	_            = iota
	jsonpathNull headerType = baseTag + ((iota - 1) * incrementor)
	JsonpathString
	jsonpathNumeric 
	jsonpathBool 
	jsonpathAnd
	jsonpathOr 
	jsonpathNot 
	jsonpathIsUnknown
	jsonpathEqual
	jsonpathNotEqual
	jsonpathLess 
	jsonpathGreater
	jsonpathLeq
	jsonpathGeq
	jsonpathAdd 
	jsonpathSub 
	jsonpathMult
	jsonpathDiv 
	jsonpathMod 
	jsonpathPlus
	jsonpathMinus
	jsonpathAnyArray
	jsonpathAnyKey
	jsonpathIndexArray
	jsonpathAny 
	jsonpathKey 
	jsonpathCurrent
	jsonpathRoot
	jsonpathVariable
	jsonpathFilter
	jsonpathExists
	jsonpathType
	jsonpathSize
	jsonpathAbs
	jsonpathFloor
	jsonpathCeiling
	jsonpathDouble
	jsonpathDatetime
	jsonpathKeyValue
	jsonpathSubscript
	jsonpathLast
	jsonpathStartsWith
	jsonpathLikeRegex
	jsonpathBigInt
	jsonpathBoolean
	jsonpathDate
	jsonpathDecimal
	jsonpathInteger
	jsonpathNumber 
	jsonpathStringFunc
	jsonpathTime 
	jsonpathTimeTz
	jsonpathTimestamp
	jsonpathtimestampTz
)

const jsonPathNextMask = 0x00FFFFFF
const maxByteLength = int(jsonPathNextMask)

func checkLength(length int) error {
	if length > maxByteLength {
		return errors.Newf("length %d is too large", errors.Safe(length))
	}
	return nil
}

func (d *DollarSymbol) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathRoot)
	return nil
}

func (d *AtSymbol) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathCurrent)
	return nil
}

func (d *VariableExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathVariable)
	length := uint32(len(d.name))
	appendTo = encoding.EncodeUint32Ascending(appendTo, length)
	appendTo = append(appendTo, []byte(d.name)...)
	return nil
}

func (d *LastExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathLast)
	return nil
}

func (d *JsonPathString) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathString)
	length := uint32(len(d.val))
	appendTo = encoding.EncodeUint32Ascending(appendTo, length)
	appendTo = append(appendTo, []byte(d.val)...)
	return nil
}

func (d *JsonPathNumeric) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathNumeric)
	decOffset := len(appendTo)
	dec := apd.Decimal(d.val)
	appendTo = encoding.EncodeUntaggedDecimalValue(appendTo, &dec)
	return nil
}

func (d *JsonPathNull) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathNull)
	return nil
}

func (d *JsonPathBool) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathBool)
	return nil
}

func (d *DotAccessExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathKey)
	length := uint32(len(d.name))
	appendTo = encoding.EncodeUint32Ascending(appendTo, length)
	appendTo = append(appendTo, []byte(d.name)...)
	return nil
}

func (d *Subscript) Encode(appendTo []byte) (err error) {
	return nil
}

func (d *SubscriptAccessExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathIndexArray)
	numElements := len(d.Subscripts)
	appendTo = encoding.EncodeUint32Ascending(appendTo, uint32(numElements))
	offsetStarting := len(appendTo)
	for i := 0; i < numElements; i++ {
		appendTo = append(appendTo, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	for i, subscript := range d.Subscripts {
		leftLen := len(appendTo)
		left_err := subscript.Begin.Encode(appendTo)
		if left_err != nil {
			return left_err
		}
		appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), offsetStarting + i * 8)

		if subscript.Slice {
			rightLen := len(appendTo)
			right_err := subscript.End.Encode(appendTo)
			if right_err != nil {
				return right_err
			}
			appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), offsetStarting + i * 8 + 4)
		}
	}

	return nil
}

func (d *WildcardAccessExpr) Encode(appendTo []byte) (err error) {
	return nil
}

func (d *RecursiveWildcardAccessExpr) Encode(appendTo []byte) (err error) {
	return nil
}

func (d *AccessExpr) Encode(appendTo []byte) (err error) {
	rootEncoded := []byte{}
	root_err := d.left.Encode(rootEncoded)
	if root_err != nil {
		return root_err
	}

	// nextOffset := len(rootEncoded)

	accessor_err := d.right.Encode(rootEncoded)
	if accessor_err != nil {
		return accessor_err
	}
	
	// add the next offset to the container header of the root expression
	appendTo = append(appendTo, rootEncoded...)
	return nil
}

func (d *BinaryPredicateExpr) Encode(appendTo []byte) (err error) {
	switch d.PredicateType {
	case EqBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathEqual)
	case NeqBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathNotEqual)
	case LtBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathLess)
	case LteBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathLeq)
	case GtBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathGreater)
	case GteBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathGeq)
	default:
		return fmt.Errorf("unsupported binary predicate type: %s", d.PredicateType)
	}

	encodingStart := len(appendTo)
	leftEncoding := []byte{}
	left_err := d.left.Encode(leftEncoding)
	if left_err != nil {
		return left_err
	}
	let leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), encodingStart)
	rightEncoding := []byte{}
	right_err := d.right.Encode(rightEncoding)
	if right_err != nil {
		return right_err
	}
	let rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), len(appendTo))
	appendTo = append(appendTo, leftEncoding..., rightEncoding...)
	return nil
}

func (d *BinaryOperatorExpr) Encode(appendTo []byte) (err error) {
	switch d.OperatorType {
	case PlusBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathAdd)
	case MinusBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathSub)
	case MultBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathMult)
	case DivBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathDiv)
	case ModBinOp:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathMod)
	default:
		return fmt.Errorf("unsupported binary operator type: %s", d.OperatorType)
	}

	encodingStart := len(appendTo)
	leftEncoding := []byte{}
	left_err := d.left.Encode(leftEncoding)
	if left_err != nil {
		return left_err
	}
	let leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), encodingStart)
	rightEncoding := []byte{}
	right_err := d.right.Encode(rightEncoding)
	if right_err != nil {
		return right_err
	}
	let rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), len(appendTo))
	appendTo = append(appendTo, leftEncoding..., rightEncoding...)
	return nil
}

func (d *BinaryLogicExpr) Encode(appendTo []byte) (err error) {
	switch d.LogicType {
	case AndBinLogic:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathAnd)
	case OrBinLogic:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathOr)
	default:
		return fmt.Errorf("unsupported binary logic type: %s", d.LogicType)
	}

	encodingStart := len(appendTo)
	leftEncoding := []byte{}
	left_err := d.left.Encode(leftEncoding)
	if left_err != nil {
		return left_err
	}
	let leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), encodingStart)
	rightEncoding := []byte{}
	right_err := d.right.Encode(rightEncoding)
	if right_err != nil {
		return right_err
	}
	let rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), len(appendTo))
	appendTo = append(appendTo, leftEncoding..., rightEncoding...)
	return nil
}

func (d *NotPredicate) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathNot)
	encodingStart := len(appendTo)
	predRes := []byte{}
	pred_err := d.expr.Encode(predRes)
	if pred_err != nil {
		return pred_err
	}
	let predResLen := len(predRes)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(predResLen), encodingStart)
	appendTo = append(appendTo, predRes...)
	return nil
}

func (d *ParenExpr) Encode(appendTo []byte) (err error) {
	return d.expr.Encode(appendTo)
}

func (d *ParenPredExpr) Encode(appendTo []byte) (err error) {
	return d.expr.Encode(appendTo)
}

func (d *UnaryOperatorExpr) Encode(appendTo []byte) (err error) {
	switch d.OperatorType {
	case Uminus:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathMinus)
	case Uplus:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathPlus)
	default:
		return fmt.Errorf("unsupported unary operator type: %s", d.OperatorType)
	}
	
	encodingStart := len(appendTo)
	exprEncoding := []byte{}
	expr_err := d.expr.Encode(exprEncoding)
	if expr_err != nil {
		return expr_err
	}
	let exprLen := len(exprEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(exprLen), encodingStart)
	appendTo = append(appendTo, exprEncoding...)
	return nil
}

func (d *FunctionExpr) Encode(appendTo []byte) (err error) {
	switch d.FunctionType {
	case TypeFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathType)
	case SizeFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathSize)
	case DoubleFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathDouble)
	case CielingFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathCieling)
	case FloorFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathFloor)
	case AbsFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathAbs)
	case DateTimeFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathDateTime)
	case KeyValueFunction:
		appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathKeyValue)
	default:
		return fmt.Errorf("unsupported function type: %s", d.FunctionType)
	}
}

func (d *FilterExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathFilter)
	encodingStart := len(appendTo)
	exprEncoding := []byte{}
	expr_err := d.expr.Encode(exprEncoding)
	if expr_err != nil {
		return expr_err
	}
	exprLen := len(exprEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(exprLen), encodingStart)
	appendTo = append(appendTo, exprEncoding...)
	return nil
}

func (d *ExistsExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathExists)
	encodingStart := len(appendTo)
	exprEncoding := []byte{}
	expr_err := d.expr.Encode(exprEncoding)
	if expr_err != nil {
		return expr_err
	}
	let exprLen := len(exprEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(exprLen), encodingStart)
	appendTo = append(appendTo, exprEncoding...)
	return nil
}

func (d *LikeRegexExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathLikeRegex)
	offsetPosition := len(appendTo)
	appendTo = append(appendTo, 0, 0, 0, 0)
	patternEncoding := []byte(d.rawPattern)
	patternLen := len(patternEncoding)
	encodingStartPosition := len(appendTo)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(patternLen), encodingStartPosition)
	appendTo = append(appendTo, patternEncoding...)
	
	exprErr := d.expr.Encode(patternEncoding)
	if exprErr != nil {
		return exprErr
	}

	encodingLength := len(patternEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(encodingLength), offsetPosition)
	appendTo = append(appendTo, patternEncoding...)
	return nil
}

func (d *StartsWithExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathStartsWith)
	encodingStart := len(appendTo)
	leftEncoding := []byte{}
	left_err := d.left.Encode(leftEncoding)
	if left_err != nil {
		return left_err
	}

	leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), encodingStart)
	
	rightEncoding := []byte{}
	right_err := d.right.Encode(rightEncoding)
	if right_err != nil {
		return right_err
	}
	rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), len(appendTo))
	appendTo = append(appendTo, leftEncoding...)
	appendTo = append(appendTo, rightEncoding...)
	return nil
}

func (d *IsUnknownExpr) Encode(appendTo []byte) (err error) {
	appendTo = encoding.EncodeUint32Ascending(appendTo, jsonpathIsUnknown)
	encodingStart := len(appendTo)
	exprEncoding := []byte{}
	expr_err := d.expr.Encode(exprEncoding)
	if expr_err != nil {
		return expr_err
	}
	let exprLen := len(exprEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(exprLen), encodingStart)
	appendTo = append(appendTo, exprEncoding...)
	return nil
}

func DecodeJsonPath(encoded []byte) ([]byte, JsonPathNode, error) {
	offset += 4
	encoded, containerHeader, decodeErr := encoding.DecodeUint32Ascending(encoded)
	if decodeErr != nil {
		return encoded, nil, decodeErr
	}

	containerLength := int(containerHeader & containerLengthMask)
	hasNext := containerLength > 0
	var res JsonPathNode
	
	switch containerHeader & containerHeaderTypeMask {
	case jsonpathNull:
		return JsonPathNull{}, nil
	case jsonpathString:
		encoded, stringLen, decodeErr := encoding.DecodeUint32Ascending(encoded)
		if decodeErr != nil {
			return encoded, nil, decodeErr
		}
		encodedString := encoded[:stringLen]
		return encoded[stringLen:], JsonPathString(encodedString), nil
	case jsonpathNumeric:
		encoded, numeric, err := encoding.DecodeUntaggedDecimalValue(encoded)
		if err != nil {
			return encoded, nil, err
		}
		baseNumeric, _ := numeric.Float64()
		return encoded, JsonPathNumeric{baseNumeric}, nil
	case jsonpathAnd:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryLogicExpr{BinLogicAnd, leftExpr, rightExpr}, nil
	case jsonpathOr:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryLogicExpr{BinLogicOr, leftExpr, rightExpr}, nil
	case jsonpathNot:
		encoded, len, decodeErr := encoding.DecodeUint32Ascending(encoded)
		if decodeErr != nil {
			return encoded, nil, decodeErr
		}

		_, expr, exprErr := DecodeJsonPath(encoded[:len])
		if exprErr != nil {
			return encoded, nil, exprErr
		}
		return encoded[len:], NotPredicate{expr}, nil
	case jsonpathIsUnknown:
		encoded, len, decodeErr := encoding.DecodeUint32Ascending(encoded)
		if decodeErr != nil {
			return encoded, nil, decodeErr
		}
		_, expr, exprErr := DecodeJsonPath(encoded[:len])
		if exprErr != nil {
			return encoded, nil, exprErr
		}
		return encoded[len:], IsUnknownExpr{expr}, nil
	case jsonpathEqual:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{EqBinOp, leftExpr, rightExpr}, nil
	case jsonpathNotEqual:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{NeqBinOp, leftExpr, rightExpr}, nil
	case jsonpathLess:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{LtBinOp, leftExpr, rightExpr}, nil
	case jsonpathGreater:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{GtBinOp, leftExpr, rightExpr}, nil
	case jsonpathLeq:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{LteBinOp, leftExpr, rightExpr}, nil
	case jsonpathGeq:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryPredicateExpr{GteBinOp, leftExpr, rightExpr}, nil
	case jsonpathAdd:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryOperatorExpr{PlusBinOp, leftExpr, rightExpr}, nil
	case jsonpathSub:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryOperatorExpr{MinusBinOp, leftExpr, rightExpr}, nil 
	case jsonpathMult:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryOperatorExpr{MultBinOp, leftExpr, rightExpr}, nil
	case jsonpathDiv:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryOperatorExpr{DivBinOp, leftExpr, rightExpr}, nil 
	case jsonpathMod:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}
		return encoded, BinaryOperatorExpr{ModBinOp, leftExpr, rightExpr}, nil 
	case jsonpathPlus:
		encoded, exprLen, exprErr := encoding.DecodeUint32Ascending(encoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		exprEncoded := encoded[:exprLen]
		_, exprExpr, exprErr := DecodeJsonPath(exprEncoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		return encoded[exprLen:], UnaryOperatorExpr{Uplus, exprExpr}, nil
	case jsonpathMinus:
		encoded, exprLen, exprErr := encoding.DecodeUint32Ascending(encoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		exprEncoded := encoded[:exprLen]
		_, exprExpr, exprErr := DecodeJsonPath(exprEncoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		return encoded[exprLen:], UnaryOperatorExpr{Uminus, exprExpr}, nil
	// case jsonpathAnyArray:
	// 	// TODO
	// case jsonpathAnyKey:
	// 	// TODO
	case jsonpathIndexArray:
		encoded, numElements, numElementsErr := encoding.DecodeUint32Ascending(encoded)
		if numElementsErr != nil {
			return encoded, nil, numElementsErr
		}

		subscriptLens := make([](uint32, uint32), numElements)
		for i := 0; i < int(numElements); i++ {
			encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
			if leftErr != nil {
				return encoded, nil, leftErr
			}

			encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
			if rightErr != nil {
				return encoded, nil, rightErr
			}

			subscriptLens[i] = (leftLen, rightLen)
		}

		subscripts := make([]Subscript, numElements)
		for index, (left, right) := range subscriptLens {
			leftEncoded := encoded[:left]
			_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
			if leftErr != nil {
				return encoded, nil, leftErr
			}

			if right == 0 {
				subscripts[index] = Subscript{
					Begin: leftExpr,
					End: nil,
					Slice: false,
				}
			} else {
				rightEncoded := encoded[left:left+right]
				_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
				if rightErr != nil {
					return encoded, nil, rightErr
				}

				subscripts[index] = Subscript{
					Begin: leftExpr,
					End: rightExpr,
					Slice: true,
				}
			}
		}

		return encoded, SubscriptAccessExpr{subscripts}, nil
	case jsonpathAny:
		// TODO 
	case jsonpathKey:
		encoded, keyLen, keyErr := encoding.DecodeUint32Ascending(encoded)
		if keyErr != nil {
			return encoded, nil, keyErr
		} 

		keyEncoded := encoded[:keyLen]
		return encoded, DotAccessExpr{keyEncoded}, nil
	case jsonpathCurrent:
		return AtSymbol{}, nil
	case jsonpathRoot:
		return DollarSymbol{}, nil
	case jsonpathVariable:
		encoded, keyLen, keyErr := encoding.DecodeUint32Ascending(encoded)
		if keyErr != nil {
			return encoded, nil, keyErr
		} 

		keyEncoded := encoded[:keyLen]
		return encoded, VariableExpr{keyEncoded}, nil

	case jsonpathFilter:
		encoded, exprLen, exprErr := encoding.DecodeUint32Ascending(encoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		exprEncoded := encoded[:exprLen]
		_, exprExpr, exprErr := DecodeJsonPath(exprEncoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		return encoded[exprLen:], FilterExpr{exprExpr}, nil
	case jsonpathExists:
		encoded, exprLen, exprErr := encoding.DecodeUint32Ascending(encoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		exprEncoded := encoded[:exprLen]
		_, exprExpr, exprErr := DecodeJsonPath(exprEncoded)
		if exprErr != nil {
			return encoded, nil, exprErr
		}

		return encoded[exprLen:], ExistsExpr{exprExpr}, nil
	case jsonpathType:
		return encoded, FunctionExpr(TypeFunction), nil
	case jsonpathSize:
		return encoded, FunctionExpr(SizeFunction), nil
	case jsonpathAbs:
		return encoded, FunctionExpr(AbsFunction), nil
	case jsonpathFloor:
		return encoded, FunctionExpr(FloorFunction), nil
	case jsonpathCeiling:
		return encoded, FunctionExpr(CielingFunction), nil
	case jsonpathDouble:
		return encoded, FunctionExpr(DoubleFunction), nil
	case jsonpathDatetime:
		return encoded, FunctionExpr(DateTimeFunction), nil
	case jsonpathKeyValue:
		return encoded, FunctionExpr(KeyValueFunction), nil
	case jsonpathSubscript:
		return encoded, nil, nil
	case jsonpathLast:
		return encoded, LastExpr{}, nil
	case jsonpathStartsWith:
		encoded, leftLen, leftErr := encoding.DecodeUint32Ascending(encoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		encoded, rightLen, rightErr := encoding.DecodeUint32Ascending(encoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		leftEncoded := encoded[:leftLen]
		rightEncoded := encoded[leftLen:leftLen+rightLen]

		_, leftExpr, leftErr := DecodeJsonPath(leftEncoded)
		if leftErr != nil {
			return encoded, nil, leftErr
		}

		_, rightExpr, rightErr := DecodeJsonPath(rightEncoded)
		if rightErr != nil {
			return encoded, nil, rightErr
		}

		return encoded[leftLen+rightLen:], StartsWithExpr{leftExpr, rightExpr}, nil
	// case jsonpathLikeRegex:
	// 	// TODO
	// case jsonpathBigInt:
	// 	// TODO
	// case jsonpathBoolean:
	// 	// TODO
	// case jsonpathDate:
	// 	// TODO
	// case jsonpathDecimal:
	// 	// TODO
	// case jsonpathInteger:
	// 	// TODO
	// case jsonpathNumber:
	// 	// TODO
	// case jsonpathStringFunc:
	// 	// TODO
	// case jsonpathTime:
	// 	// TODO
	// case jsonpathTimeTz:
	// 	// TODO
	// case jsonpathTimestamp:
	// 	// TODO
	// case jsonpathtimestampTz:
	// 	// TODO
	default:
		return nil, fmt.Errorf("unknown jsonpath container type %d", containerHeader & containerHeaderTypeMask)
	}

	if hasNext {
		encoded, next, nextErr := DecodeJsonPath(encoded)
		if nextErr != nil {
			return encoded, nil, nextErr
		}

		return encoded, AccessExpr(res, next), nil
	} else {
		return encoded, res, nil
	}
}