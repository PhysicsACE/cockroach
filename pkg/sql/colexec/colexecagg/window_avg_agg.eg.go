// Code generated by execgen; DO NOT EDIT.
// Copyright 2018 The Cockroach Authors.
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package colexecagg

import (
	"unsafe"

	"github.com/cockroachdb/apd/v3"
	"github.com/cockroachdb/cockroach/pkg/col/coldata"
	"github.com/cockroachdb/cockroach/pkg/sql/colexecerror"
	"github.com/cockroachdb/cockroach/pkg/sql/colmem"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
	"github.com/cockroachdb/cockroach/pkg/util/duration"
	"github.com/cockroachdb/errors"
)

// Workaround for bazel auto-generated code. goimports does not automatically
// pick up the right packages when run within the bazel sandbox.
var (
	_ tree.AggType
	_ apd.Context
	_ duration.Duration
)

func newAvgWindowAggAlloc(
	allocator *colmem.Allocator, t *types.T, allocSize int64,
) (aggregateFuncAlloc, error) {
	allocBase := aggAllocBase{allocator: allocator, allocSize: allocSize}
	switch t.Family() {
	case types.IntFamily:
		switch t.Width() {
		case 16:
			return &avgInt16WindowAggAlloc{aggAllocBase: allocBase}, nil
		case 32:
			return &avgInt32WindowAggAlloc{aggAllocBase: allocBase}, nil
		case -1:
		default:
			return &avgInt64WindowAggAlloc{aggAllocBase: allocBase}, nil
		}
	case types.DecimalFamily:
		switch t.Width() {
		case -1:
		default:
			return &avgDecimalWindowAggAlloc{aggAllocBase: allocBase}, nil
		}
	case types.FloatFamily:
		switch t.Width() {
		case -1:
		default:
			return &avgFloat64WindowAggAlloc{aggAllocBase: allocBase}, nil
		}
	case types.IntervalFamily:
		switch t.Width() {
		case -1:
		default:
			return &avgIntervalWindowAggAlloc{aggAllocBase: allocBase}, nil
		}
	}
	return nil, errors.Errorf("unsupported avg agg type %s", t.Name())
}

type avgInt16WindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum apd.Decimal
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgInt16WindowAgg{}

func (a *avgInt16WindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int16(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgInt16WindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Decimal()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {

		col[outputIdx].SetInt64(a.curCount)
		if _, err := tree.DecimalCtx.Quo(&col[outputIdx], &a.curSum, &col[outputIdx]); err != nil {
			colexecerror.ExpectedError(err)
		}
	}
}

func (a *avgInt16WindowAgg) Reset() {
	a.curSum = zeroDecimalValue
	a.curCount = 0
}

type avgInt16WindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgInt16WindowAgg
}

var _ aggregateFuncAlloc = &avgInt16WindowAggAlloc{}

const sizeOfAvgInt16WindowAgg = int64(unsafe.Sizeof(avgInt16WindowAgg{}))
const avgInt16WindowAggSliceOverhead = int64(unsafe.Sizeof([]avgInt16WindowAgg{}))

func (a *avgInt16WindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgInt16WindowAggSliceOverhead + sizeOfAvgInt16WindowAgg*a.allocSize)
		a.aggFuncs = make([]avgInt16WindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgInt16WindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int16(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}

type avgInt32WindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum apd.Decimal
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgInt32WindowAgg{}

func (a *avgInt32WindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int32(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgInt32WindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Decimal()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {

		col[outputIdx].SetInt64(a.curCount)
		if _, err := tree.DecimalCtx.Quo(&col[outputIdx], &a.curSum, &col[outputIdx]); err != nil {
			colexecerror.ExpectedError(err)
		}
	}
}

func (a *avgInt32WindowAgg) Reset() {
	a.curSum = zeroDecimalValue
	a.curCount = 0
}

type avgInt32WindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgInt32WindowAgg
}

var _ aggregateFuncAlloc = &avgInt32WindowAggAlloc{}

const sizeOfAvgInt32WindowAgg = int64(unsafe.Sizeof(avgInt32WindowAgg{}))
const avgInt32WindowAggSliceOverhead = int64(unsafe.Sizeof([]avgInt32WindowAgg{}))

func (a *avgInt32WindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgInt32WindowAggSliceOverhead + sizeOfAvgInt32WindowAgg*a.allocSize)
		a.aggFuncs = make([]avgInt32WindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgInt32WindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int32(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}

type avgInt64WindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum apd.Decimal
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgInt64WindowAgg{}

func (a *avgInt64WindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int64(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgInt64WindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Decimal()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {

		col[outputIdx].SetInt64(a.curCount)
		if _, err := tree.DecimalCtx.Quo(&col[outputIdx], &a.curSum, &col[outputIdx]); err != nil {
			colexecerror.ExpectedError(err)
		}
	}
}

func (a *avgInt64WindowAgg) Reset() {
	a.curSum = zeroDecimalValue
	a.curCount = 0
}

type avgInt64WindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgInt64WindowAgg
}

var _ aggregateFuncAlloc = &avgInt64WindowAggAlloc{}

const sizeOfAvgInt64WindowAgg = int64(unsafe.Sizeof(avgInt64WindowAgg{}))
const avgInt64WindowAggSliceOverhead = int64(unsafe.Sizeof([]avgInt64WindowAgg{}))

func (a *avgInt64WindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgInt64WindowAggSliceOverhead + sizeOfAvgInt64WindowAgg*a.allocSize)
		a.aggFuncs = make([]avgInt64WindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgInt64WindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Int64(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					var tmpDec apd.Decimal //gcassert:noescape
					tmpDec.SetInt64(int64(v))
					if _, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &tmpDec); err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}

type avgDecimalWindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum apd.Decimal
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgDecimalWindowAgg{}

func (a *avgDecimalWindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Decimal(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					_, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &v)
					if err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					_, err := tree.ExactCtx.Add(&a.curSum, &a.curSum, &v)
					if err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount++
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgDecimalWindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Decimal()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {

		col[outputIdx].SetInt64(a.curCount)
		if _, err := tree.DecimalCtx.Quo(&col[outputIdx], &a.curSum, &col[outputIdx]); err != nil {
			colexecerror.ExpectedError(err)
		}
	}
}

func (a *avgDecimalWindowAgg) Reset() {
	a.curSum = zeroDecimalValue
	a.curCount = 0
}

type avgDecimalWindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgDecimalWindowAgg
}

var _ aggregateFuncAlloc = &avgDecimalWindowAggAlloc{}

const sizeOfAvgDecimalWindowAgg = int64(unsafe.Sizeof(avgDecimalWindowAgg{}))
const avgDecimalWindowAggSliceOverhead = int64(unsafe.Sizeof([]avgDecimalWindowAgg{}))

func (a *avgDecimalWindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgDecimalWindowAggSliceOverhead + sizeOfAvgDecimalWindowAgg*a.allocSize)
		a.aggFuncs = make([]avgDecimalWindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgDecimalWindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	oldCurSumSize := a.curSum.Size()
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Decimal(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					_, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &v)
					if err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					_, err := tree.ExactCtx.Sub(&a.curSum, &a.curSum, &v)
					if err != nil {
						colexecerror.ExpectedError(err)
					}
				}

				a.curCount--
			}
		}
	}
	newCurSumSize := a.curSum.Size()
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}

type avgFloat64WindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum float64
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgFloat64WindowAgg{}

func (a *avgFloat64WindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	var oldCurSumSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Float64(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					a.curSum = float64(a.curSum) + float64(v)
				}

				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					a.curSum = float64(a.curSum) + float64(v)
				}

				a.curCount++
			}
		}
	}
	var newCurSumSize uintptr
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgFloat64WindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Float64()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {
		col[outputIdx] = a.curSum / float64(a.curCount)
	}
}

func (a *avgFloat64WindowAgg) Reset() {
	a.curSum = zeroFloat64Value
	a.curCount = 0
}

type avgFloat64WindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgFloat64WindowAgg
}

var _ aggregateFuncAlloc = &avgFloat64WindowAggAlloc{}

const sizeOfAvgFloat64WindowAgg = int64(unsafe.Sizeof(avgFloat64WindowAgg{}))
const avgFloat64WindowAggSliceOverhead = int64(unsafe.Sizeof([]avgFloat64WindowAgg{}))

func (a *avgFloat64WindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgFloat64WindowAggSliceOverhead + sizeOfAvgFloat64WindowAgg*a.allocSize)
		a.aggFuncs = make([]avgFloat64WindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgFloat64WindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	var oldCurSumSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Float64(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					a.curSum = float64(a.curSum) - float64(v)
				}

				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)

				{

					a.curSum = float64(a.curSum) - float64(v)
				}

				a.curCount--
			}
		}
	}
	var newCurSumSize uintptr
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}

type avgIntervalWindowAgg struct {
	unorderedAggregateFuncBase
	// curSum keeps track of the sum of elements belonging to the current group,
	// so we can index into the slice once per group, instead of on each
	// iteration.
	curSum duration.Duration
	// curCount keeps track of the number of non-null elements that we've seen
	// belonging to the current group.
	curCount int64
}

var _ AggregateFunc = &avgIntervalWindowAgg{}

func (a *avgIntervalWindowAgg) Compute(
	vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int, sel []int,
) {
	var oldCurSumSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Interval(), vec.Nulls()
	// Unnecessary memory accounting can have significant overhead for window
	// aggregate functions because Compute is called at least once for every row.
	// For this reason, we do not use PerformOperation here.
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)
				a.curSum = a.curSum.Add(v)
				a.curCount++
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)
				a.curSum = a.curSum.Add(v)
				a.curCount++
			}
		}
	}
	var newCurSumSize uintptr
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsageAfterAllocation(int64(newCurSumSize - oldCurSumSize))
	}
}

func (a *avgIntervalWindowAgg) Flush(outputIdx int) {
	// The aggregation is finished. Flush the last value. If we haven't found
	// any non-nulls for this group so far, the output for this group should be
	// NULL.
	col := a.vec.Interval()
	if a.curCount == 0 {
		a.nulls.SetNull(outputIdx)
	} else {
		col[outputIdx] = a.curSum.Div(int64(a.curCount))
	}
}

func (a *avgIntervalWindowAgg) Reset() {
	a.curSum = zeroIntervalValue
	a.curCount = 0
}

type avgIntervalWindowAggAlloc struct {
	aggAllocBase
	aggFuncs []avgIntervalWindowAgg
}

var _ aggregateFuncAlloc = &avgIntervalWindowAggAlloc{}

const sizeOfAvgIntervalWindowAgg = int64(unsafe.Sizeof(avgIntervalWindowAgg{}))
const avgIntervalWindowAggSliceOverhead = int64(unsafe.Sizeof([]avgIntervalWindowAgg{}))

func (a *avgIntervalWindowAggAlloc) newAggFunc() AggregateFunc {
	if len(a.aggFuncs) == 0 {
		a.allocator.AdjustMemoryUsage(avgIntervalWindowAggSliceOverhead + sizeOfAvgIntervalWindowAgg*a.allocSize)
		a.aggFuncs = make([]avgIntervalWindowAgg, a.allocSize)
	}
	f := &a.aggFuncs[0]
	f.allocator = a.allocator
	a.aggFuncs = a.aggFuncs[1:]
	return f
}

// Remove implements the slidingWindowAggregateFunc interface (see
// window_aggregator_tmpl.go).
func (a *avgIntervalWindowAgg) Remove(vecs []coldata.Vec, inputIdxs []uint32, startIdx, endIdx int) {
	var oldCurSumSize uintptr
	vec := vecs[inputIdxs[0]]
	col, nulls := vec.Interval(), vec.Nulls()
	_, _ = col.Get(endIdx-1), col.Get(startIdx)
	if nulls.MaybeHasNulls() {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = nulls.NullAt(i)
			if !isNull {
				//gcassert:bce
				v := col.Get(i)
				a.curSum = a.curSum.Sub(v)
				a.curCount--
			}
		}
	} else {
		for i := startIdx; i < endIdx; i++ {

			var isNull bool
			isNull = false
			if !isNull {
				//gcassert:bce
				v := col.Get(i)
				a.curSum = a.curSum.Sub(v)
				a.curCount--
			}
		}
	}
	var newCurSumSize uintptr
	if newCurSumSize != oldCurSumSize {
		a.allocator.AdjustMemoryUsage(int64(newCurSumSize - oldCurSumSize))
	}
}
