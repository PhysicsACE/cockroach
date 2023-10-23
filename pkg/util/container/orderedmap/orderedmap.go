// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sortedmap implements operations on a sortedmap
//
// To iterate over a sortedmap (where m is a *sortedmap):
//
//  for i, v := range m.Iterate(reverse) {
//      // do something with v.key or v.value
//  }
package orderedmap

import "fmt"
import "github.com/cockroachdb/cockroach/pkg/util/container/list"

type KvNode[K comparable, V any] struct {
	// The linked list node for this node that stores its value and
	// its prev and next pointers
	key K

	// The key that this node belongs to
	value V
}

type OrderedMap[K comparable, V any] struct {

	// Initial node to create the linked list of values
	hardroot *list.List[KvNode[K, V]]

	// Mapping between the key and its respective node
	kvStore map[K]*list.Element[KvNode[K, V]]

	// The number of keys stored in the object
	size int
}

// Init initializes or clears orderedmap e
func (e *OrderedMap[K, V]) Init() *OrderedMap[K, V] {
	e.hardroot = list.New[KvNode[K, V]]()
	e.kvStore = make(map[K]*list.Element[KvNode[K, V]])
	e.size = 0
	return e
}

//Return an initialized orderedmap with the speficied maps
func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] { return new(OrderedMap[K, V]).Init() }

// Get value of
func (e *OrderedMap[K, V]) Get(key K) (V, bool) {
	fmt.Println("map side", e.kvStore)
	if val, ok := e.kvStore[key]; ok {
		return val.Value.value, true
	} else {
		var res V 
		return res, false
	}
}

// Insert the key, value pair in the map
func (e *OrderedMap[K, V]) Insert(key K, value V) {
	if val, ok := e.kvStore[key]; ok {
		val.Value.value = value
		return
	}

	valueNode := e.hardroot.PushBack(KvNode[K, V]{value: value, key: key})
	e.kvStore[key] = valueNode
	e.size++
}

// Remove the key from the map if it exists
func (e *OrderedMap[K, V]) Remove(key K) {
	if node, ok := e.kvStore[key]; ok {
		e.hardroot.Remove(node)
		delete(e.kvStore, key)
		e.size--
	}
}

// Get the size of the map
func (e *OrderedMap[K, V]) Size() int {
	return e.size
}

// Pop and return the key, value pair based on LIFO insertion
func (e *OrderedMap[K, V]) popItem(last bool) *KvNode[K, V] {
	if e.hardroot.Len() == 0 {
		return &KvNode[K, V]{}
	}
	var popped KvNode[K, V]
	if last {
		lifoObject := e.hardroot.Back()
		popped = lifoObject.Value
		e.hardroot.Remove(lifoObject)
	} else {
		fifoObject := e.hardroot.Front()
		popped = fifoObject.Value
		e.hardroot.Remove(fifoObject)
	}
  delete(e.kvStore, popped.key)
  e.size--
	return &popped
}

// Get the first element
func (e *OrderedMap[K, V]) Front() *KvNode[K, V] {
	return &e.hardroot.Front().Value
}

// Get the last element
func (e *OrderedMap[K, V]) Back() *KvNode[K, V] {
	return &e.hardroot.Back().Value
}

// Return a slice to iterate over
func (e *OrderedMap[K, V]) Iterate(reverse bool) []*KvNode[K, V] {
	orderedKV := make([]*KvNode[K, V], 0)
	if reverse {
		for n := e.hardroot.Back(); n != nil; n = n.Prev() {
			orderedKV = append(orderedKV, &n.Value)
		}
	} else {
		for n := e.hardroot.Front(); n != nil; n = n.Next() {
			orderedKV = append(orderedKV, &n.Value)
		}
	}
	return orderedKV
}

// Remove a range with respect to a key, direction and inclusivity
func (e *OrderedMap[K, V]) RemoveRange(key K, left bool, inclusive bool) {
	if _, ok := e.kvStore[key]; !ok {
		return
	}

	if left {
		lifoObject := e.hardroot.Back()
		for {
			if lifoObject.Value.key == key {
				if inclusive {
					e.Remove(lifoObject.Value.key)
				}
				break
			}
			e.Remove(lifoObject.Value.key)
			lifoObject = e.hardroot.Back()
		}
	} else {
		fifoObject := e.hardroot.Front()
		for {
			if fifoObject.Value.key == key {
				if inclusive {
					e.Remove(fifoObject.Value.key)
				}
				break
			}
			e.Remove(fifoObject.Value.key)
			fifoObject = e.hardroot.Front()
		}
	}
}


