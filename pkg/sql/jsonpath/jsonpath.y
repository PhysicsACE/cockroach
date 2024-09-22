%{
package jsonpath

import (
    "strings"

    "github.com/cockroachdb/cockroach/pkg/sql/scanner"
    "github.com/cockroachdb/errors"
)
%}

%{
func setErr(jsonpathlex jsonpathLexer, err error) int {
    jsonpathlex.(*lexer).setErr(err)
    return 1
}

func unimplemented(jsonpathlex jsonpathLexer, feature string) int {
    jsonpathlex.(*lexer).Unimplemented(feature)
    return 1
}

var _ scanner.ScanSymType = &jsonpathSymType{}

func (j *jsonpathSymType) ID() int32 {
    return j.id
}

func (j *jsonpathSymType) SetID(id int32) {
    j.id = id
}

func (j *jsonpathSymType) Pos() int32 {
return j.pos
}

func (j *jsonSymType) SetPos(pos int32) {
    j.pos = pos
}

func (j *jsonpathSymType) Str() string {
    return j.str
}

func (j *jsonpathSymType) SetStr(str string) {
    j.str = str
}

func (j *jsonpathSymType) UnionVal() interface{} {
    return j.union.val
}

func (j *jsonpathSymType) SetUnionVal(val interface{}) {
    j.union.val = val
}

func (j *jsonpathSymType) jsonpathScanSymType() {}

type jsonpathSymUnion struct {
    val interface{}
}

func (u *jsonpathSymUnion) dollar() *jsonpathtree.DollarSymbol {
    return u.val.(*jsonpathtree.DollarSymbol)
}

func (u *jsonpathSymUnion) atSymbol() *jsonpathtree.AtSymbol {
    return u.val.(*jsonpathtree.AtSymbol)
}

func (u *jsonpathSymUnion) variable() *jsonpathtree.VariableExpr {
    return u.val.(*jsonpathtree.VariableExpr)
}

func (u *jsonpathSymUnion) dotAccessor() *jsonpathtree.DotAccessor {
    return u.val.(*jsonpathtree.DotAccessor)
}

func (u *jsonpathSymUnion) subscriptAccessor() *jsonpathtree.SubscriptAccessExpr {
    return u.val.(*jsonpathtree.SubscriptAccessExpr)
}

func (u *jsonpathSymUnion) wildcard() *jsonpathtree.WildcardAccessExpr {
    return u.val.(*jsonpathtree.WildcardAccessExpr)
}

func (u *jsonpathSymUnion) recursiveWildcard() *jsonpathtree.RecursiveWildcardAccessExpr {
    return u.val.(*jsonpathtree.RecursiveWildcardAccessExpr)
}

func (u *jsonpathSymUnion) binaryPredicate() *jsonpathtree.BinaryPredicateExpr {
    return u.val.(*jsonpathtree.BinaryPredicateExpr)
}

func (u *jsonpathSymUnion) binaryOperator() *jsonpathtree.BinaryOperatorExpr {
    return u.val.(*jsonpathtree.BinaryOperatorExpr)
}

func (u *jsonpathSymUnion) binaryLogic() *jsonpathtree.BinaryLogicExpr {
    return u.val.(*jsonpathtree.BinaryLogicExpr)
}

func (u *jsonpathSymUnion) paren() *jsonpathtree.ParenExpr {
    return u.val.(*jsonpathtree.ParenExpr)
}

func (u *jsonpathSymUnion) parenPredicate() *jsonpathtree.ParenPredExpr {
    return u.val.(*jsonpathtree.ParenPredExpr)
}

func (u *jsonpathSymUnion) unaryOperator() *jsonpathtree.UnaryOperatorExpr {
    return u.val.(*jsonpathtree.UnaryOperatorExpr)
}

func (u *jsonpathSymUnion) function() *jsonpathtree.FunctionExpr {
    return u.val.(*jsonpathtree.FunctionExpr)
}

func (u *jsonpathSymUnion) exists() *jsonpathtree.ExistsExpr {
    return u.val.(*jsonpathtree.ExistsExpr)
}

func (u *jsonpathSymUnion) regex() *jsonpathtree.LikeRegexExpr {
    return u.val.(*jsonpathtree.LikeRegexExpr)
}

func (u *jsonpathSymUnion) startsWith() *jsonpathtree.StartsWithExpr {
    return u.val.(*jsonpathtree.StartsWithExpr)
}

func (u *jsonpathSymUnion) unknown() *jsonpathtree.IsUnknownExpr {
    return u.val.(*jsonpathtree.IsUnknownExpr)
}

%}

%token <str> IDENT UIDENT FCONST SCONST USCONST BCONST XCONST Op 
%token <str> EQUALS_GREATER LESS_EQUALS GREATER_EQUALS NOT_EQUALS 