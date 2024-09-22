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
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	// "github.com/cockroachdb/cockroach/pkg/util/errorutil/unimplemented"
)

type DollarSymbol struct {}

type AtSymbol struct {}

type VariableExpr struct {
	name string
	quoted bool
}

type JsonPathExpr interface {
	Eval(*JsonPathEvalCtx) (tree.DJSON, error)
}

type JsonPathPredicate interface {
	EvalPredicate(*JsonPathEvalCtx) (tree.DBool, error)
}

type JsonPathString string
type JsonPathNumeric float64

type JsonPathNull struct {}
type JsonPathBool bool
type JsonPathList []JsonPathExpr

type Accessor interface {
	EvalAccess(*JsonPathEvalCtx) (tree.DJSON, error)
}

type DotAccessExpr struct {
	name JsonPathExpr
}

type SubscriptAccessExpr struct {
	begin JsonPathExpr
	end JsonPathExpr
	slice bool
}

type WildcardAccessExpr struct {}

type RecursiveWildcardAccessExpr struct {
	Begin JsonPathExpr
	End JsonPathExpr
}

type BinPredType int

const (
	eqBinOp BinPredType = iota
	neqBinOp
	gtBinOp
	gteBinOp
	ltBinOp
	lteBinOp
)
type BinaryPredicateExpr struct {
	PredicateType BinPredType
	left JsonPathExpr
	right JsonPathExpr
}

type BinaryExprType int

const (
	plusBinOp BinaryExprType = iota
	minusBinOp
	multBinOp
	divBinOp
	modBinOp
)

type BinaryOperatorExpr struct {
	OperatorType BinaryExprType
	left JsonPathExpr
	right JsonPathExpr
}

type BinLogicType int

const (
	andBinLogic BinLogicType = iota
	orBinLogic
)

type BinaryLogicExpr struct {
	LogicType BinLogicType
	left JsonPathExpr
	right JsonPathExpr
}

type ParenExpr struct {
	expr JsonPathExpr
}

type ParenPredExpr struct {
	expr JsonPathPredicate
}

type UranaryExprType int

const (
	uminus UranaryExprType = iota
	uplus
)

type UnaryOperatorExpr struct {
	OperatorType UranaryExprType
	expr JsonPathExpr
}

type Function int

const (
	TypeFunction Function = iota
	SizeFunction
	DoubleFunction
	CielingFunction
	FloorFunction
	AbsFunction
	DateTimeFunction
	KeyValueFunction

)
type FunctionExpr struct {
	FunctionType Function
	expr JsonPathExpr
}

type ExistsExpr struct {
	expr JsonPathExpr
}

type LikeRegexExpr struct {
	expr JsonPathExpr
	regex string
}

type StartsWithExpr struct {
	expr JsonPathExpr

}

type IsUnknownExpr struct {
	expr JsonPathExpr
}


