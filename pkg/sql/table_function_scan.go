// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package sql

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/util/jsonpath"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/eval"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
)

type tableFuncScanNode struct {
	tableFuncScan *tree.TableFunc
	resultColumns colinfo.ResultColumns
	currentRow    tree.Datums

	// If the json_table expression contains nested columns, they are stored
	// as a child node to the current node. The results of the child node are then
	// outer/cross joined with the results of this node to produce the final result
	child tableFuncScanNode

	// Given the scenario that a json_table expression contains multiple nested
	// columns, which are on the same nesting level, they are considered as sibling
	// nodes and thier results are Unioned to produce the final result which will
	// then be provided to parent nodes to construct the final results
	sibling tableFuncScanNode
}

func (n *tableFuncScanNode) startExec(params runParams) error {
	n.typs = planTypes(n.tableFuncScan)

	return nil
}

func (n *tableFuncScanNode) Next(params runParams) (bool, error) {
	// Evaluate the document expression to fetch the json document to perform
	// the json_table operation on 
	res, err := eval.EvalExpr(params.ctx, params.EvalContext(), n.tableFuncScan.Document)
	if err != nil {
		return false, err
	}

	jsonDocument := tree.MustBeDJSON(res)

	// If additonal path is provided, execute it on the json document
	if n.tableFuncScan.DocumentPath != nil {
		pathExec, err = jsonpath.ExecPath(jsonDocument.JSON, n.tableFuncScan.DocumentPath)
		if err != nil {
			return false, err
		}
		jsonDocument = tree.NewDJSON(pathExec)
	}

	// For each column to be constructed, executed its respective path on the 
	// root json document and then perform the necessary casting to construct
	// the required Datums

	return false, nil
}

func (n *tableFuncScanNode) Values() tree.Datums {
	return n.currentRow
}

func (n *tableFuncScanNode) Close(ctx context.Context) {
	n.rows.Close(ctx)
}
