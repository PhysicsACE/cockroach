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
	"fmt"
	"strings"
	"regexp"

	"github.com/cockroachdb/errors"
	// "github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/util/json"
	// "github.com/cockroachdb/cockroach/pkg/util/errorutil/unimplemented"
)

type JsonPathNode interface {
	// Walk(JsonPathVisitor)
	// Format(*FmtCtx)
	// Encode([]byte) []byte
}

type DollarSymbol struct {}

type AtSymbol struct {}

type VariableExpr struct {
	name string
	quoted bool
}

type LastExpr struct {}

type JsonPathExpr interface {
	Eval(*JsonPathEvalCtx) (json.JSON, error)
}

type JsonPathPredicate interface {
	EvalPredicate(*JsonPathEvalCtx) (bool, error)
}

type JsonPathString struct {val string}
type JsonPathNumeric struct {val float64}

type JsonPathNull struct {}
type JsonPathBool struct {val bool}
type JsonPathList []JsonPathExpr

type Accessor interface {
	EvalAccess(*JsonPathEvalCtx, json.JSON) (json.JSON, error)
}

type DotAccessExpr struct {
	name string
	quoted bool
}

type Subscript struct {
	Begin JsonPathExpr
	End JsonPathExpr
	Slice bool
}

type SubscriptList []Subscript

type SubscriptAccessExpr struct {
	Subscripts SubscriptList
}

type WildcardAccessExpr struct {}

type RecursiveWildcardAccessExpr struct {
	levels Subscript
}

type AccessExpr struct {
	left JsonPathExpr
	right Accessor
}

type BinPredType int

const (
	EqBinOp BinPredType = iota
	NeqBinOp
	GtBinOp
	GteBinOp
	LtBinOp
	LteBinOp
)
type BinaryPredicateExpr struct {
	PredicateType BinPredType
	left JsonPathExpr
	right JsonPathExpr
}

type BinaryExprType int

const (
	PlusBinOp BinaryExprType = iota
	MinusBinOp
	MultBinOp
	DivBinOp
	ModBinOp
)

type BinaryOperatorExpr struct {
	OperatorType BinaryExprType
	left JsonPathExpr
	right JsonPathExpr
}

type BinLogicType int

const (
	AndBinLogic BinLogicType = iota
	OrBinLogic
)

type BinaryLogicExpr struct {
	LogicType BinLogicType
	left JsonPathPredicate
	right JsonPathPredicate
}

type NotPredicate struct {
	expr JsonPathPredicate
}

type ParenExpr struct {
	expr JsonPathExpr
}

type ParenPredExpr struct {
	expr JsonPathPredicate
}

type UranaryExprType int

const (
	Uminus UranaryExprType = iota
	Uplus
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
	arg string
}

type FilterExpr struct {
	pred JsonPathPredicate
}

type ExistsExpr struct {
	expr JsonPathExpr
}

type LikeRegexExpr struct {
	expr JsonPathExpr
	rawPattern string
	pattern *regexp.Regexp
	flag *string
}

type StartsWithExpr struct {
	left JsonPathExpr
	right JsonPathExpr
}

type IsUnknownExpr struct {
	expr JsonPathPredicate
}