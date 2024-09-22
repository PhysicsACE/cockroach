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

type jEntry struct {
	typ int
	offset int
}

func MakeJEntry(typ int, offset int) jEntry {
	return jEntry{typ: typ, offset: offset}
}

const (
	jsonpathNull = iota
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

func EncodeJsonPath(appendTo []byte, expr JsonPathExpr) ([]byte, error) {

}

func (d *DollarSymbol) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathRoot, 0)
	appendTo = append(appendTo, byte(jsonpathRoot))
}

func (d *AtSymbol) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathCurrent, 0)
	appendTo = append(appendTo, byte(jsonpathCurrent))
}

func (d *VariableExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathVariable, len(d.name))
	appendTo = append(appendTo, byte(jsonpathVariable))
	return jEntry, append(appendTo, []byte(d.name)...), nil
}

func (d *LastExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathLast, 0)
	appendTo = append(appendTo, byte(jsonpathLast))
}

func (d *JsonPathString) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(JsonpathString, len(d.val))
	appendTo = append(appendTo, byte(JsonpathString))
	return jEntry, append(appendTo, []byte(d.val)...), nil
}

func (d *JsonPathNumeric) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathNumeric, 0)
	appendTo = append(appendTo, byte(jsonpathNumeric))
	decOffset := len(appendTo)
	dec := apd.Decimal(d.val)
	appendTo = encoding.EncodeUntaggedDecimalValue(appendTo, &dec)
	lengthInBytes := len(appendTo) - decOffset
	if err := checkLength(lengthInBytes); err != nil {
		return jEntry{}, []byte{}, err
	}
	jEntry.offset = lengthInBytes
	return jEntry, appendTo, nil
}

func (d *JsonPathNull) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathNull, 0)
	appendTo = append(appendTo, byte(jsonpathNull))
}

func (d *JsonPathBool) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathBool, 0)
	appendTo = append(appendTo, byte(jsonpathBool))
}

func (d *DotAccessExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathKey, len(d.name))
	appendTo = append(appendTo, byte(jsonpathKey))
	return jEntry, append(appendTo, []byte(d.name)...), nil
}

func (d *Subscript) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathSubscript, 0)
	appendTo = append(appendTo, byte(jsonpathSubscript))
}

func (d *SubscriptAccessExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathIndexArray, 0)
	appendTo = append(appendTo, jsonpathIndexArray, 0, 0, 0)
	numElements := len(d.Subscripts)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(numElements), len(appendTo))
	offsetStarting := len(appendTo)
	for i := 0; i < numElements; i++ {
		appendTo = append(appendTo, 0, 0, 0, 0, 0, 0, 0, 0)
	}

	for i, subscript := range d.Subscripts {
		leftLen := len(appendTo)
		appendTo, err := subscript.Begin.Encode(appendTo)
		if err != nil {
			return nil, b, err
		}
		appendTo = encoding.PutUint32Ascending(appendTo, uint32(leftLen), offsetStarting + i*8)

		if subscript.Slice {
			rightLen := len(appendTo)
			appendTo, err := subscript.End.Encode(appendTo)
			if err != nil {
				return nil, b, err
			}
			appendTo = encoding.PutUint32Ascending(appendTo, uint32(rightLen), offsetStarting + i*8 + 4)
		}
	}

	return jEntry, b, nil
}

func (d *WildcardAccessExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathAnyKey, 0)
	appendTo = append(appendTo, byte(jsonpathAnyKey))
}

func (d *RecursiveWildcardAccessExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathAny, 0)
	appendTo = append(appendTo, byte(jsonpathAny))
}

func (d *AccessExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var rootEncoded []byte
	rootExpr, b, err := d.left.Encode(rootEncoded)
	if err != nil {
		return nil, b, err
	}
	nextOffset := len(b) - 4
	appendTo = append(appendTo, b...)
	// Add the next offset to the header of the root expression encoding
	accessorEntry, b, err := d.right.Encode(appendTo)
	if err != nil {
		return nil, b, err
	}
	return rootExpr, appendTo, nil
}

func (d *BinaryPredicateExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var jEntry jEntry
	switch d.PredicateType {
	case EqBinOp:
		jEntry = MakeJEntry(jsonpathEqual, 0)
		appendTo = append(appendTo, byte(jsonpathEqual))
	case NeqBinOp:
		jEntry = MakeJEntry(jsonpathNotEqual, 0)
		appendTo = append(appendTo, byte(jsonpathNotEqual))
	case LtBinOp:
		jEntry = MakeJEntry(jsonpathLess, 0)
		appendTo = append(appendTo, byte(jsonpathLess))
	case LteBinOp:
		jEntry = MakeJEntry(jsonpathLeq, 0)
		appendTo = append(appendTo, byte(jsonpathLeq))
	case GtBinOp:
		jEntry = MakeJEntry(jsonpathGreater, 0)
		appendTo = append(appendTo, byte(jsonpathGreater))
	case GteBinOp:
		jEntry = MakeJEntry(jsonpathGeq, 0)
		appendTo = append(appendTo, byte(jsonpathGeq))
	default:
		return nil, appendTo, fmt.Errorf("unsupported binary operator %s", d.PredicateType)
	}
}

func (d *BinaryOperatorExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var jEntry jEntry
	switch d.OperatorType {
	case PlusBinOp:
		jEntry = MakeJEntry(jsonpathAdd, 0)
		appendTo = append(appendTo, byte(jsonpathAdd))
	case MinusBinOp:
		jEntry = MakeJEntry(jsonpathSub, 0)
		appendTo = append(appendTo, byte(jsonpathSub))
	case MultBinOp:
		jEntry = MakeJEntry(jsonpathMult, 0)
		appendTo = append(appendTo, byte(jsonpathMult))
	case DivBinOp:
		jEntry = MakeJEntry(jsonpathDiv, 0)
		appendTo = append(appendTo, byte(jsonpathDiv))
	case ModBinOp:
		jEntry = MakeJEntry(jsonpathMod, 0)
		appendTo = append(appendTo, byte(jsonpathMod))
	default:
		return nil, appendTo, fmt.Errorf("unsupported binary operator %s", d.OperatorType)
	}
}

func (d *BinaryLogicExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var jEntry jEntry
	switch d.LogicType {
	case AndLogicOp:
		jEntry = MakeJEntry(jsonpathAnd, 0)
		appendTo = append(appendTo, byte(jsonpathAnd))
	case OrLogicOp:
		jEntry = MakeJEntry(jsonpathOr, 0)
		appendTo = append(appendTo, byte(jsonpathOr))
	default:
		return nil, appendTo, fmt.Errorf("unsupported binary operator %s", d.LogicType)
	}

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	leftEncoding := make([]byte, 0)
	leftEntry, b, err := d.left.Encode(leftEncoding)
	if err != nil {
		return nil, b, err
	}
	leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(8), encodingStartPosition)

	rightEncoding := make([]byte, 0)
	rightEntry, b, err := d.right.Encode(rightEncoding)
	if err != nil {
		return nil, b, err
	}
	rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(8+leftLen), encodingStartPosition+4)
	appendTo = append(appendTo, leftEncoding..., rightEncoding...)
	jEntry.offset = leftEntry.offset + rightEntry.offset
	return jEntry, b, nil
}

func (d *NotPredicate) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathNot, 0)
	appendTo = append(appendTo, byte(jsonpathNot))
}

func (d *ParenExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	return d.expr.Encode(appendTo)
}

func (d *ParenPredExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	return d.expr.Encode(appendTo)
}

func (d *UnaryOperatorExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var jEntry jEntry
	switch d.OperatorType {
	case Uminus:
		jEntry = MakeJEntry(jsonpathNeg, 0)
		appendTo = append(appendTo, byte(jsonpathNeg))
	case Uplus:
		jEntry = MakeJEntry(jsonpathPos, 0)
		appendTo = append(appendTo, byte(jsonpathPos))
	default:
		return nil, appendTo, fmt.Errorf("unsupported unary operator %s", d.OperatorType)
	}

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	subEncoding := make([]byte, 0)
	subEntry, b, err := d.expr.Encode(subEncoding)
	if err != nil {
		return nil, b, err
	}
	subLen := len(subEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(subLen), encodingStartPosition)
	appendTo = append(appendTo, subEncoding...)
	return jEntry, b, nil
}

func (d *FunctionExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	var jEntry jEntry
	switch d.FunctionType {
	case TypeFunction:
		jEntry = MakeJEntry(jsonpathType, 0)
		appendTo = append(appendTo, byte(jsonpathType))
	case SizeFunction:
		jEntry = MakeJEntry(jsonpathSize, 0)
		appendTo = append(appendTo, byte(jsonpathSize))
	case DoubleFunction:
		jEntry = MakeJEntry(jsonpathDouble, 0)
		appendTo = append(appendTo, byte(jsonpathDouble))
	case CielingFunction:
		jEntry = MakeJEntry(jsonpathCieling, 0)
		appendTo = append(appendTo, byte(jsonpathCieling))
	case FloorFunction:
		jEntry = MakeJEntry(jsonpathFloor, 0)
		appendTo = append(appendTo, byte(jsonpathFloor))
	case AbsFunction:
		jEntry = MakeJEntry(jsonpathAbs, 0)
		appendTo = append(appendTo, byte(jsonpathAbs))
	case DateTimeFunction:
		jEntry = MakeJEntry(jsonpathDateTime, 0)
		appendTo = append(appendTo, byte(jsonpathDateTime))
	case KeyValueFunction:
		jEntry = MakeJEntry(jsonpathKeyValue, 0)
		appendTo = append(appendTo, byte(jsonpathKeyValue))
	}
}

func (d *FilterExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathFilter, 0)
	appendTo = append(appendTo, byte(jsonpathFilter))

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	subEncoding := make([]byte, 0)
	subEntry, b, err := d.expr.Encode(subEncoding)
	if err != nil {
		return nil, b, err
	}
	subLen := len(subEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(subLen), encodingStartPosition)
	appendTo = append(appendTo, subEncoding...)
	return jEntry, b, nil
}

func (d *ExistsExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathExists, 0)
	appendTo = append(appendTo, byte(jsonpathExists))

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	subEncoding := make([]byte, 0)
	subEntry, b, err := d.expr.Encode(subEncoding)
	if err != nil {
		return nil, b, err
	}
	subLen := len(subEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(subLen), encodingStartPosition)
	appendTo = append(appendTo, subEncoding...)
	return jEntry, b, nil
}

func (d *LikeRegexExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathLikeRegex, 0)
	appendTo = append(appendTo, byte(jsonpathLikeRegex))

	appendTo = append(appendTo, 0, 0, 0)
	offsetPositiion := len(appendTo)
	appendTo = append(appendTo, 0, 0, 0, 0)
	patternEncoding := []byte(d.rawPattern)
	patternLength := len(patternEncoding)
	encodingStartPosition := len(appendTo)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(patternLength), encodingStartPosition)
	appendTo = append(appendTo, patternEncoding...)
	
	_, err := d.expr.Encode(patternEncoding)
	if err != nil {
		return appendTo, err
	}

	offsetLength := len(patternEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(offsetLength), offsetPositiion)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(patternLength), len(appendTo))
	appendTo = append(appendTo, patternEncoding...)

	return appendTo, nil
}

func (d *StartsWithExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathStartsWith, 0)
	appendTo = append(appendTo, byte(jsonpathStartsWith))

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	leftEncoding := make([]byte, 0)
	leftEntry, b, err := d.left.Encode(leftEncoding)
	if err != nil {
		return nil, b, err
	}
	leftLen := len(leftEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(8), encodingStartPosition)

	rightEncoding := make([]byte, 0)
	rightEntry, b, err := d.right.Encode(rightEncoding)
	if err != nil {
		return nil, b, err
	}
	rightLen := len(rightEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(8+leftLen), encodingStartPosition+4)
	appendTo = append(appendTo, leftEncoding..., rightEncoding...)
	return jEntry, b, nil
}

func (d *IsUnknownExpr) Encode(appendTo []byte) (e jEntry, b []byte, err error) {
	jEntry := MakeJEntry(jsonpathIsUnknown, 0)
	appendTo = append(appendTo, byte(jsonpathIsUnknown))

	appendTo = append(appendTo, 0, 0, 0)
	encodingStartPosition := len(appendTo)

	subEncoding := make([]byte, 0)
	subEntry, b, err := d.expr.Encode(subEncoding)
	if err != nil {
		return nil, b, err
	}
	subLen := len(subEncoding)
	appendTo = encoding.PutUint32Ascending(appendTo, uint32(subLen), encodingStartPosition)
	appendTo = append(appendTo, subEncoding...)
	return jEntry, b, nil
}

func DecodeJsonPath(encoded []byte) (JsonPathNode, error) {
	header := encoded[0:4]

	switch header[0] {
	case jsonpathNull:
		return jsonpathNull{}, nil
	case jsonpathString:
	case jsonpathNumeric 
	case jsonpathBool 
	case jsonpathAnd:
		node := BinaryLogicExpr{
			LogicType: AndBinLogic,
		}
		right := encoded[8:12]
		rightOffset := binary.LittleEndian.Uint32(right)
		leftEncoding := encoded[12:rightOffset]
		nextOffset := extractLink(header)
		var rightEncoding []byte
		if nextOffset == 0 {
			rightEncoding := encoded[rightOffset:]
		} else {
			rightEncoding = encoded[rightOffset:nextOffset]
		}

		leftExpr, err := DecodeJsonPath(leftEncoding)
		if err != nil {
			return nil, err
		}

		rightExpr, err := DecodeJsonPath(rightEncoding)
		if err != nil {
			return nil, err
		}

		node.left = leftExpr
		node.right = rightExpr
		return node, nil
	case jsonpathOr 
	case jsonpathNot:
		exprLength := encoded[4:8]
		exprOffset := binary.LittleEndian.Uint32(exprLength)
		exprEncoding := encoded[8:exprOffset]
		expr, err := DecodeJsonPath(exprEncoding)
		if err != nil {
			return nil, err
		}
		return NotPredicate{
			expr: expr,
		}, nil
	case jsonpathIsUnknown
	case jsonpathEqual
	case jsonpathNotEqual
	case jsonpathLess 
	case jsonpathGreater
	case jsonpathLeq
	case jsonpathGeq
	case jsonpathAdd 
	case jsonpathSub 
	case jsonpathMult
	case jsonpathDiv 
	case jsonpathMod 
	case jsonpathPlus
	case jsonpathMinus
	case jsonpathAnyArray
	case jsonpathAnyKey
	case jsonpathIndexArray
	case jsonpathAny 
	case jsonpathKey 
	case jsonpathCurrent:
		return AtSymbol{}, nil
	case jsonpathRoot:
		return DollarSymbol{}, nil
	case jsonpathVariable
	case jsonpathFilter
	case jsonpathExists
	case jsonpathType
	case jsonpathSize
	case jsonpathAbs
	case jsonpathFloor
	case jsonpathCeiling
	case jsonpathDouble
	case jsonpathDatetime
	case jsonpathKeyValue
	case jsonpathSubscript
	case jsonpathLast:
		return LastExpr{}, nil
	case jsonpathStartsWith
	case jsonpathLikeRegex
	case jsonpathBigInt
	case jsonpathBoolean
	case jsonpathDate
	case jsonpathDecimal
	case jsonpathInteger
	case jsonpathNumber 
	case jsonpathStringFunc
	case jsonpathTime 
	case jsonpathTimeTz
	case jsonpathTimestamp
	case jsonpathtimestampTz
	default:
		return nil, fmt.Errorf("unsupported jsonpath operator %d", header[0])
	}
}