// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package optbuilder

func (b *Builder) buildTableFunction(
	tableFunc *tree.TableFunc, inScope *scope,
) *scope {
	if tableFunc == nil {
		return inScope
	}

	outScope = inScope.push()
	outScope.appendColumnsFromScope(inScope)

	tableFuncPrivate, err := ConstructRelationalTableFunc(tableFunc, outScope)
	if err != nil {
		panic(err)
	}

	tableFuncPrivate.Input = inScope.expr
	outScope.expr = b.factory.ConstructTableFuncScan(tableFuncPrivate)
	return outScope
}

func (b *Builder) ConstructRelationalTableFunc(
	tableFunc *tree.TableFunc,
	inScope *scope,
) (*memo.TableFuncPrivate, error) {
	// Constuct the table function relational expression while adding
	// the scope columns that will be constucted by the table function

	tableFuncPrivate := &memo.TableFuncPrivate{}

	if tableFunc.Document == nil {
		typedDocExpr, err := tableFunc.Document.TypeCheck(b.ctx, b.semaCtx, &types.Jsonb)
		if err != nil {
			panic(err)
		}

		tableFuncPrivate.Document = typedDocExpr
	}

	md := b.factory.Metadata()
	cols := make(opt.ColList, 0, len(tableFunc.Columns))
	for i, col := range tableFunc.Columns {
		tableFuncPrivate.Columns = append(tableFuncPrivate.Columns, col)
		if col.OnEmpty != nil {
			if col.OnEmpty.DefaultExpr == nil {
				typedDefault, err := col.OnEmpty.DefaultExpr.TypeCheck(b.ctx, b.semaCtx, col.Type)
				if err != nil {
					panic(err)
				}
				tableFuncPrivate.Columns[i].OnEmpty.DefaultExpr = typedDefault
				table
			}
		}

		if col.OnError != nil {
			if col.OnError.DefaultExpr == nil {
				typedDefault, err := col.OnError.DefaultExpr.TypeCheck(b.ctx, b.semaCtx, col.Type)
				if err != nil {
					panic(err)
				}
				tableFuncPrivate.Columns[i].OnError.DefaultExpr = typedDefault
			}
		}

		col := md.AddColumn(col.Name, col.Type)
		cols = append(cols, col)
	}

	inScope.cols = make([]scopeColumn, len(cols))
	for i, col := range cols {
		columnMeta := md.ColumnMeta(col)
		inScope.cols[i] = scopeColumn{
			id:    columnMeta,
			name:  scopeColName(tree.Name(col.Name)),
			table: tree.TableName{
				objName: tree.objName{
					ObjectName: tableFunc.Name,
				}
			},
			typ:   tableFunc.Columns[i].Type,
		}
	}

	var nested *memo.RelExpr
	if len(tableFunc.NestedPath) > 0 {
		for i, col := range tableFunc.NestedColumns {
			childTableFunc := &tree.TableFunc{
				FunctionType: tree.JSONTABLE,
				DocumentPath: tableFunc.NestedPath[i],
				Columns:      tableFunc.NestedColumns[i],
			}

			childExpr, err := ConstructRelationalTableFunc(childTableFunc, inScope)
			if err != nil {
				panic(err)
			}

			if nested == nil {
				nested = childExpr
			} else {
				childExpr.ChildNode = nested
				nested = childExpr
			}
		}
	}

	tableFuncPrivate.ChildNode = nested

	return tableFuncPrivate, nil
}