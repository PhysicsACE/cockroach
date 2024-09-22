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

type JsonPathIterator struct {
	iterable json.JSON

	idx uint32

	len uint32

	exausted bool
}

func NewJsonPathIterator(iterable json.JSON) *JsonPathIterator {
	if json.IsJsonArray(iterable) {
		jsonArray, _ := json.AsJsonArray(iterable)
		return &JsonPathIterator{
			iterable: jsonArray,
			len:      uint32(jsonArray.Len()),
			idx:      0,
		}
	}
	return &JsonPathIterator{
		iterable: iterable,
		len:      1,
		idx:      0,
	}
}

func (jpi *JsonPathIterator) Reset(iterable json.JSON) {
	jpi.iterable = iterable
	if json.IsJsonArray(iterable) {
		jsonArray, _ := json.AsJsonArray(iterable)
		jpi.len = uint32(jsonArray.Len())
	} else {
		jpi.len = 1
	}
	jpi.idx = 0
	jpi.exausted = false
}

func (jpi *JsonPathIterator) Exausted() bool {
	return jpi.exausted
}

func (jpi *JsonPathIterator) Len() uint32 {
	return jpi.len
}

func (jpi *JsonPathIterator) Next() (json.JSON, error) {
	if jpi.idx < jpi.len {
		if json.IsJsonArray(jpi.iterable) {
			jsonArray, _ := json.AsJsonArray(jpi.iterable)
			res := jsonArray[jpi.idx]
			jpi.idx++
			return res, nil
		} else {
			jpi.idx++
			jpi.exausted = true
			return jpi.iterable, nil
		}
	}
	
	return nil, nil
}