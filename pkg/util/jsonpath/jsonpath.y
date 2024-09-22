%{
package jsonpath

import (
    "strings"
    "regexp"

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

func (u *jsonpathSymUnion) expr() JsonPathExpr {
    if expr, ok := u.val.(JsonPathExpr); ok {
        return expr
    }
    return nil
}

func (u *jsonpathSymUnion) accessor() Accessor {
    if accessor, ok := u.val.(Accessor); ok {
        return accessor
    }
    return nil
}

func (u *jsonpathSymUnion)

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
%token <str> EQUALS LESS_EQUALS GREATER_EQUALS NOT_EQUALS 
%token <str> AND EXISTS FALSE FLAG FUNC_DATETIME FUNC_KEYVALUE 
%token <str> FUNC_TYPE FUNC_SIZE FUNC_DOUBLE FUNC_CEILING FUNC_FLOOR
%token <str> FUNC_ABS IS LAST LAX LIKE_REGEX NUMBER NULL OR STR STRICT
%token <str> STARTS TRUE TO UNKNOWN UNOT WITH EOF

%union {
    str string
    regexp *regexp.Regexp
    union jsonpathSymUnion
}

%type <JsonPathExpr> root
%type <JsonPathExpr> expr
%type <JsonPathExpr> at_sign
%type <JsonPathExpr> variable_expr
%type <JsonPathExpr> literal
%type <JsonPathExpr> accessor_expr
%type <Accessor> accessor
%type <Accessor> member_accessor
%type <Accessor> member_accessor_wildcard
%type <Accessor> array_accessor
%type <SubscriptList> subscript_list
%type <Subscript> subscript
%type <Accessor> wildcard_array_accessor
%type <Accessor> item_method
%type <Accessor> method
%type <Accessor> filter_expr
%type <JsonPathPredicate> predicate_primary
%type <JsonPathPredicate> delimited_predicate
%type <JsonPathPredicate> non_delimited_predicate
%type <JsonPathPredicate> comparison_pred
%type <JsonPathPredicate> exists_pred
%type <JsonPathPredicate> like_regex_pred
%type <JsonPathPredicate> starts_with_pred
%type <JsonPathPredicate> unknown_pred
%type <str> like_regex_pattern
%type <str> like_regex_flag

%left OR
%left AND
%left EQUALS '>' '<' GREATER_EQUALS LESS_EQUALS NOT_EQUALS
%left '+' '-'
%left '*' '/' '%'
%left UMINUS
%left '.'

%%

root:
    LAX expr
    {
        yylex.(*tokenStream).root = Program{
            mode: modeLax,
            root: $2.expr(),
        }
    }
    | STRICT expr
    {
        yylex.(*tokenStream).root = Program{
            mode: modeStrict,
            root: $2.expr(),
        }
    }

expr:
    '(' expr ')'
    {
        $$.val = ParenExpr{expr: $2.expr()}
    }
    | expr '+' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: PlusBinOp, left: $1.expr(), right: $3.expr()}

    }
    | expr '-' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: MinusBinOp, left: $1.expr(), right: $3.expr()}
        
    }
    | expr '*' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: MultBinOp, left: $1.expr(), right: $3.expr()}
        
    }
    | expr '/' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: DivBinOp, left: $1.expr(), right: $3.expr()}
        
    }
    | expr '%' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: ModBinOp, left: $1.expr(), right: $3.expr()}
        
    }
    | '-' expr %prec UMINUS
    {
        $$.val = UnaryOperatorExpr{OperatorType: Uminus, expr: $2.expr()}
    }
    | '+' expr %prec UMINUS
    {
        $$.val = UnaryOperatorExpr{OperatorType: Uplus, expr: $2.expr()}
    }
    | accessor_expr

primary:
    literal
    | variable

literal:
    NUMBER
    | TRUE
    {
        $$.val = JsonPathBool{val: true}
    }
    | FALSE 
    {
        $$.val = JsonPathBool{val: false}
    }
    | NULL 
    {
        $$.val = JsonPathNull{}
    }
    | STR 
    {
        $$.val = JsonPathString{val: $1}
    }

variable:
    IDENT
    {
        $$.val = VariableExpr{name: $1}
    }
    | LAST
    {
        $$.val = LastExpr{}
    }

accessor_expr:
    primary
    | accessor_expr accessor
    {
        $$.val = AccessExpr{left: $1.expr(), right: $2.accessor()}
    }

accessor:
    member_accessor
    | member_accessor_wildcard
    | array_accessor
    | wildcard_array_accessor
    | filter_expr
    | item_method

member_accessor:
    '.' IDENT 
    {
        $$.val = DotAccessExpr{name: $2}
    }
    | '.' STR 
    {
        $$.val = DotAccessExpr{name: $2, quoted: true}
    }

member_accessor_wildcard:
    '.' '*'
    {
        $$.val = WildcardAccessExpr{}
    }
    | '.' '*' '*'
    {
        $$.val = RecursiveWildcardAccessExpr{}
    }
    | '.' '*' '*' '{' subscript '}'
    {
        $$.val = RecursiveWildcardAccessExpr{levels: $5.subscript()}
    }

array_accessor:
    '[' subscript_list ']
    {
        $$.val = SubscriptAccessExpr{subscripts: $2.subscript_list()}
    }

subscript_list:
    subscript
    {
        $$.val = []SubscriptList{$1.subscript()}
    }
    | subscript_list ',' subscript
    {
        ##.val = append($1.subscript_list(), $3.subscript())
    }

subscript:
    expr
    {
        $$.val = Subscript{Begin: $1.expr()}
    }
    | expr TO expr
    {
        $$.val = Subscript{Begin: $1.expr(), End: $3.expr(), Slice: true}
    }

wildcard_array_accessor:
    '[' '*' ']'
    {
        $$.val = WildcardAccessExpr{}
    }

item_method:
    '.' method
    {
        $$.val = $2.accessor()
    }

method:
    FUNC_TYPE '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: TypeFunction}
    }
    | FUNC_SIZE '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: SizeFunction}
    }
    | FUNC_DOUBLE '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: DoubleFunction}
    }
    | FUNC_CEILING '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: CielingFunction}
    }
    | FUNC_FLOOR '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: FloorFunction}
    }
    | FUNC_ABS '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: AbsFunction}
    }
    | FUNC_DATETIME '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: DateTimeFunction}
    }
    | FUNC_DATETIME '(' STR ')'
    {
        $$.val = FunctionExpr{FunctionType: SizeFunction, arg: $3}
    }
    | FUNC_KEYVALUE '(' ')'
    {
        $$.val = FunctionExpr{FunctionType: KeyValueFunction}
    }

filter_expr:
    '?' '(' predicate_primary ')'
    {
        $$.val = FilterExpr{pred: $3.predicate()}
    }
    | '?' '(' expr ')'
    {
        yylex.(*tokenStream).err = fmt.Errorf("filter expressions cannot be raw JSON values")
        return 0
    }

predicate_primary:
    delimited_predicate
    {
        $$.val = $1.predicate()
    }
    | non_delimited_predicate
    {
        $$.val = $1.predicate()
    }

delimited_predicate:
    exists_pred
    {
        $$.val = $1.predicate()
    }
    | '(' predicate_primary ')'
    {
        $$.val = ParenPredExpr{expr: $2.predicate()}
    }

non_delimited_predicate:
    comparison_pred
    {
        $$.val = $1.predicate()
    }
    | like_regex_pred
    {
        $$.val = $1.predicate()
    }
    | starts_with_pred
    {
        $$.val = $1.predicate()
    }
    | unknown_pred
    {
        $$.val = $1.predicate()
    }

exists_pred:
    EXISTS '(' expr ')'
    {
        $$.val = ExistsExpr{expr: $3.expr()}
    }

comparison_pred:
    expr EQUALS expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: EqBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr NOT_EQUALS expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: NeqBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr '>' expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: GtBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr GREATER_EQUALS expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: GteBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr '<' expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: LtBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr LESS_EQUALS expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: LteBinOp, left: $1.expr(), right: $3.expr()}
    }
    | expr AND expr
    {
        $$.val = BinaryLogicExpr{LogicType: AndBinLogic, left: $1.expr(), right: $3.expr()}
    }
    | expr OR expr
    {
        $$.val = BinaryLogicExpr{LogicType: OrBinLogic, left: $1.expr(), right: $3.expr()}
    }
    | UNOT predicate_primary %prec UMINUS
    {
        $$.val = UnaryNot{expr: $2.predicate()}
    }

like_regex_pred:
    expr LIKE_REGEX like_regex_pattern FLAG like_regex_flag
    {
        pattern, err := regexp.Compile($3)
        if err != nil {
            yylex.(*tokenStream).err = err
            return 1
        }
        $$.val = LikeRegexExpr{expr: $1.expr(), rawPattern: $3, pattern: pattern, flag: &$5}
    }
    | expr LIKE_REGEX like_regex_pattern
    {
        pattern, err := regexp.Compile($3)
        if err != nil {
            yylex.(*tokenStream).err = err
            return 1
        }
        $$.val = LikeRegexExpr{expr: $1.expr(), rawPattern: $3, pattern: pattern}
    }

like_regex_pattern: STR

like_regex_flag: STR

starts_with_pred:
    expr STARTS WITH expr
    {
        $$.val = StartsWithExpr{left: $1.expr(), right: $4.expr()}
    }

unknown_pred:
    '(' predicate_primary ')' IS UNKNOWN
    {
        $$.val = IsUnknownExpr{expr: $2.predicate()}
    }

    






