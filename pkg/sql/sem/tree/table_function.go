// Copyright 2017 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package tree

type FunctionType int

const (
	JSONTABLE FunctionType = iota
	XMLTABLE 
)

type TableFunc struct {
	// The type of table function to execution. Postgres currently supports
	// json_table and xml_table
	FunctionType FunctionType
	// The name of the table function
	Alias tree.Name
	// The document to execute the table function on
	Document Expr
	// Additional path to access data in the document
	DocumentPath string
	// The relational columns to generate from the document
	Columns []TableColumn
	// Path to nest the results of the table function
	NestedPath []string
	// Additional columns to construct from the nested path
	NestedColumns []TableColumns
	// Whether to laterally join the results of the table function to the
	// rest of the FROM clause
	Lateral bool
}

func (node *TableFunc) Format(ctx *FmtCtx) {
	if node == nil {
		return nil
	}
	if node.Lateral {
		ctx.WriteString("LATERAL ")
	}

	switch node.FunctionType {
	case JSONTABLE:
		ctx.WriteString("json_table")
	case XMLTABLE:
		ctx.WriteString("xml_table")
	}
	ctx.WriteString("(")
	ctx.FormatNode(node.Document)
	ctx.WriteString(", ")

	if node.DocumentPath != "" {
		ctx.WriteString(node.DocumentPath)
		ctx.WriteString(", ")
	}

	if len(node.Columns) > 0 {
		ctx.WriteString("COLUMNS(")
		for i, col := range node.Columns {
			if i > 0 {
				ctx.WriteString(", ")
			}
			ctx.WriteString(col.Name)
			ctx.WriteString(" ")
			ctx.FormatTypeReference(col.Type)
		}
		ctx.WriteString(")")
	}
	ctx.WriteString(")")
}

type TableColumn struct {
	// Name of the column
	Name string
	// Type of the column
	Type *types.T
	// If the column is ordinal
	Ordinality bool
	// OnEmpty clause
	OnEmpty ExceptionHandler
	// OnError clause
	OnError ExceptionHandler
	// Path to access data for the column
	Path string
	// Exists query for the column. If path exists in the document, the return
	// type must be an int (1 for success and 0 for failure)
	Exists bool
}

type TableColumns []TableColumn

type ExceptionHandler struct {
	// Return null datum in the event of empty of failure
	Null bool
	// Default expression to evaluate in the event of an error or missing path
	DefaultExpr Expr
	// If empty or error, halt execution and return error
	Error bool
}