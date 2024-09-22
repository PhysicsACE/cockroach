// Copyright 2015 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package sql

type tableFuncScanNode struct {
	FunctionType tree.FunctionType

	Input exec.Node

	InputRow tree.Datums

	SiblingNode planNode

	SiblingRow tree.Datums

	ChildNode planNode

	ChildRow tree.Datums

	DocumentExpr tree.TypedExpr

	Document tree.Datum

	DocumentPath *types.JsonPath

	Columns []tree.TableColumn

	ColumnRow tree.Datums

	Lateral bool

	NestedFound bool

	NestedIterator *jsonpath.JsonPathIterator

	OutputTypes []*types.T

	OutputCols []tree.Datum
}

func (tf *tableFuncScanNode) AddInput(inputNode exec.Node) {
	tf.Input = inputNode
}

func (tf *tableFuncScanNode) AddSibling(siblingNode exec.Node) {
	tf.SiblingNode = siblingNode
}

func (tf *tableFuncScanNode) AddChild(childNode exec.Node) {
	tf.ChildNode = childNode
}

func (tf *tableFuncScanNode) SetDocument(doc tree.Datum) {
	tf.Document = doc
}

func (tf *tableFuncScanNode) SetDocumentAndInitIterator(doc tree.Datum) {
	tf.Document = doc
	tf.NestedIterator = jsonpath.NewJsonPathIterator(doc.JSON)
}

func (tf *tableFuncScanNode) startExec(_ context.Context) error {
	tf.OutputCols = make([]tree.Datum, len(tf.OutputTypes))
}

func (tf *tableFuncScanNode) Next(params runParams) (bool, error) {
	fetchNext := true
	if tf.ChildNode != nil {
		if !tf.NestedFound {
			fetchNext = false
		}
	}

	if tf.NestedIterator != nil {
		fetchNext = false
	}

	if fetchNext {
		inputSource, err := tf.Input.Next(params)
		if err != nil {
			return false, err
		}

		tf.InputRow = tf.Input.Values()
		params.EvalContext().PushIVarContainer()

		// TODO: for now, assume we have the input row and have executed
		// the document expressio to get the root document and it has been
		// placed in tf.Document

		if tf.DocumentPath != nil {
			pathDocument, err := jsonpath.ExecJsonPath(tf.Document.JSON, tf.DocumentPath)
			if err != nil {
				return false, err
			}
			tf.Document = pathDocument
		}
	}

	if err := tf.ConstructNext(params); err != nil {
		return false, err
	}

	tf.outputCols[:len(tf.InputRow)] = tf.InputRow
	tf.outputCols[len(tf.InputRow):len(tf.InputRow)+len(tf.Columns)] = tf.ColumnRow
	if tf.ChildNode != nil {
		tf.outputCols[len(tf.InputRow)+len(tf.Columns):] = tf.ChildRow
	}

	return true, nil

}

func (tf *tableFuncScanNode) ConstructNext(params runParams) error {

	fromDatum := tf.Document
	refreshColummns := false
	tf.OutputCols = make([]tree.Datum, len(tf.OutputTypes))
	if tf.NestedIterator != nil {
		iterate := true
		if tf.ChildNode != nil {
			iterate !tf.NestedFound
		}
		if iterate {
			nextJSON := tf.NestedIterator.Next()
			if nextJSON == json.NullJSONValue {
				tf.NestedFound = false
				return nil
			}
			fromDatum = tree.NewDJSON(nextJSON)
			refreshColummns = true
		} else {
			currJSON := tf.NestedIterator.Current()
			fromDatum = tree.NewDJSON(currJSON)
			refreshColummns = false
		}

		if tf.Nestediterator.Exausted() {
			tf.NestedFound = false
		}
	}

	if refreshColummns {
		colOutput := make([]tree.Datum, len(tf.Columns))
		for i := 0; i < len(tf.Columns); i++ {
			colJSON, err := jsonpath.ExecJsonPath(
				fromDatum.JSON, tf.Columns[i].Path,
			)

			if err != nil {
				if tf.Columns[i].OnError != nil {
					if tf.Columns[i].OnError.Null {
						colOutput[i] = tree.DNull
					} else if tf.Columns[i].OnError.DefaultExpr != nil {
						// Can execute in empty eval context as default expression
						// cannot reference any outer columns
						// TODO: add the expression evaluation along with the
						// document expression logic
						continue
					} else {
						return false, err
					}
				}
			}

			if colJSON == json.NullJSONValue {
				if tf.Columns[i].OnEmpty != nil {
					if tf.Columns[i].OnEmpty.Null {
						colOutput[i] = tree.DNull
					} else if tf.Columns[i].OnEmpty.DefaultExpr != nil {
						// Can execute in empty eval context as default expression
						// cannot reference any outer columns
						// TODO: add the expression evaluation along with the
						// document expression logic
						continue
					} else {
						return false, err
					}
				}
			}

			if tf.Columns[i].Type.Family() == types.JsonFamily {
				colOutput[i] = tree.NewDJSON(colJSON)
			} else {
				// Cast the resulting json.JSON value to the specified type
				// for the column. If the casting is not possible, then
				// reperform the onError logic above. Otherwise, construct
				// the appropriate datum value and add it colOutput

				colOutput[i] = tree.NewDString(colJSON.String())
			}
		}

		tf.ColumnRow = colOutput
		tf.OutputCols = append(tf.OutputCols, tf.ColumnRow...)
	}
	

	if tf.ChildNode != nil {
		tf.ChildNode.SetDocumentAndInitIterator(tf.Document)
		if err := tf.ChildNode.ConstructNext(params); err != nil {
			return err
		}

		_, err := tf.ChildNode.Next(params)
		if err != nil {
			return err
		}

		tf.ChildRow = tf.ChildNode.Values()
		tf.OutputCols = append(tf.OutputCols, tf.ChildRow...)
	}

	if tf.SiblingNode != nil {
		tf.SiblingNode.SetDocumentAndInitIterator(tf.Document)
		if err := tf.SiblingNode.ConstructNext(params); err != nil {
			return err
		}

		_, err := tf.SiblingNode.Next(params)

		if err != nil {
			return err
		}

		tf.SiblingRow = tf.SiblingNode.Values()
		tf.OutputCols = append(tf.OutputCols, tf.SiblingRow...)
	}

	tf.OutputCols = colOutput
	return nil
}

func (tf *tableFuncScanNode) Close(_ context.Context) {}

func (tf *tableFuncScanNode) Values() tree.Datums {
	return tf.outputCols
}


