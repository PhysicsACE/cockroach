%{
package jsonpath

import (
    "strings"
    "regexp"
)

type jsonpathSymUnion struct {
  val interface{}
}

%}

%union {
  expr JsonPathExpr
  pred JsonPathPred
  vals []JsonPathNode
  regexp *regexp.Regexp
  ranges []Subscript
  rangeNode Subscript
  accessor Accessor
  accessorList []Accessor
  str string
}

%token <val> AND
%token <val> EQ EXISTS
%token <val> FALSE FLAG
%token <val> FUNC_DATETIME FUNC_KEYVALUE
%token <val> FUNC_TYPE FUNC_SIZE FUNC_DOUBLE FUNC_CEILING FUNC_FLOOR FUNC_ABS
%token <val> GTE
%token <str> IDENT IS
%token <val> LAST LAX LTE LIKE_REGEX
%token <val> NEQ
%token <expr> NUMBER NULL
%token <val> OR
%token <str> STR
%token <val> STRICT STARTS
%token <val> TRUE TO
%token <val> UNKNOWN UNOT
%token <val> WITH
%token <val> EOF

%type <expr> root
%type <expr> expr
%type <expr> at_sign
%type <expr> variable_expr
%type <expr> literal
%type <expr> accessor_expr
%type <accessor> accessor
%type <accessor> member_accessor
%type <accessor> member_accessor_wildcard
%type <accessor> array_accessor
%type <accessortList> subscript_list
%type <accessort> subscript
%type <accessor> wildcard_array_accessor
%type <accessor> item_method
%type <accessor> method
%type <accessor> filter_expr
%type <pred> predicate_primary
%type <pred> delimited_predicate
%type <pred> non_delimited_predicate
%type <pred> comparison_pred
%type <pred> exists_pred
%type <pred> like_regex_pred
%type <pred> starts_with_pred
%type <pred> unknown_pred
%type <str> like_regex_pattern
%type <str> like_regex_flag

%left OR
%left AND
%left EQ '>' '<' GTE LTE NEQ
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
        root: $2,
      }
    }
    | STRICT expr
    {
      yylex.(*tokenStream).root = Program{
        mode: modeStrict,
        root: $2,
      }
    }

expr:
    '(' expr ')'
    {
        $$.val = ParenExpr{expr: $2}
    }
    | expr '+' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: PlusBinOp, left: $1, right: $3}

    }
    | expr '-' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: MinusBinOp, left: $1, right: $3}

    }
    | expr '*' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: MultBinOp, left: $1, right: $3}

    }
    | expr '/' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: DivBinOp, left: $1, right: $3}

    }
    | expr '%' expr
    {
        $$.val = BinaryOperatorExpr{OperatorType: ModBinOp, left: $1, right: $3}

    }
    | '-' expr %prec UMINUS
    {
        $$.val = UnaryOperatorExpr{OperatorType: Uminus, expr: $2}
    }
    | '+' expr %prec UMINUS
    {
        $$.val = UnaryOperatorExpr{OperatorType: Uplus, expr: $2}
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
    | '@'
    {
        $$.val = AtSymbol{}
    }
    | LAST
    {
        $$.val = LastExpr{}
    }

accessor_expr:
    primary
    | accessor_expr accessor
    {
        $$.val = AccessExpr{left: $1, right: $2}
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
        $$.val = RecursiveWildcardAccessExpr{levels: $5}
    }

array_accessor:
    '[' subscript_list ']
    {
        $$.val = SubscriptAccessExpr{subscripts: $2}
    }

subscript_list:
    subscript
    {
        $$.val = []Subscript{$1}
    }
    | subscript_list ',' subscript
    {
        ##.val = append($1, $3)
    }

subscript:
    expr
    {
        $$.val = Subscript{Begin: $1}
    }
    | expr TO expr
    {
        $$.val = Subscript{Begin: $1, End: $3, Slice: true}
    }

wildcard_array_accessor:
    '[' '*' ']'
    {
        $$.val = WildcardAccessExpr{}
    }

item_method:
    '.' method
    {
        $$.val = $2
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
        $$.val = FilterExpr{pred: $3}
    }
    | '?' '(' expr ')'
    {
        yylex.(*tokenStream).err = fmt.Errorf("filter expressions cannot be raw JSON values")
        return 0
    }


predicate_primary:
    delimited_predicate
    | non_delimited_predicate

delimited_predicate:
    exists_pred
    | '(' predicate_primary ')'
    {
        $$.val = ParenPredExpr{expr: $2}
    }

non_delimited_predicate:
    comparison_pred
    | like_regex_pred
    | starts_with_pred
    | unknown_pred

exists_pred:
    EXISTS '(' expr ')'
    {
        $$.val = ExistsExpr{expr: $3}
    }

comparison_pred:
    expr EQ expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: EqBinOp, left: $1, right: $3}
    }
    | expr NEQ expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: NeqBinOp, left: $1, right: $3}
    }
    | expr '>' expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: GtBinOp, left: $1, right: $3}
    }
    | expr GTE expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: GteBinOp, left: $1, right: $3}
    }
    | expr '<' expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: LtBinOp, left: $1, right: $3}
    }
    | expr LTE expr
    {
        $$.val = BinaryPredicateExpr{PredicateType: LteBinOp, left: $1, right: $3}
    }
    | expr AND expr
    {
        $$.val = BinaryLogicExpr{LogicType: AndBinLogic, left: $1, right: $3}
    }
    | expr OR expr
    {
        $$.val = BinaryLogicExpr{LogicType: OrBinLogic, left: $1, right: $3}
    }
    | UNOT predicate_primary %prec UMINUS
    {
        $$.val = UnaryNot{expr: $2}
    }

like_regex_pred:
    expr LIKE_REGEX like_regex_pattern FLAG like_regex_flag
    {
        pattern, err := regexp.Compile($3)
        if err != nil {
            yylex.(*tokenStream).err = err
            return 1
        }
        $$.val = LikeRegexExpr{expr: $1, rawPattern: $3, pattern: pattern, flag: &$5}
    }
    | expr LIKE_REGEX like_regex_pattern
    {
        pattern, err := regexp.Compile($3)
        if err != nil {
            yylex.(*tokenStream).err = err
            return 1
        }
        $$.val = LikeRegexExpr{expr: $1, rawPattern: $3, pattern: pattern}
    }

like_regex_pattern: STR
like_regex_flag: STR

starts_with_pred:
    expr STARTS WITH expr
    {
        $$.val = StartsWithExpr{left: $1, right: $4}
    }

unknown_pred:
    '(' predicate_primary ')' IS UNKNOWN
    {
        $$.val = IsUnknownExpr{expr: $2}
    }

&&