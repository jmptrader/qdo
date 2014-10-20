package worker

import (
	"strconv"
	"sync/atomic"
)

type AtomicInt int64

func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

func (i *AtomicInt) Set(n int64) {
	atomic.StoreInt64((*int64)(i), n)
}

func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}

type Stats struct {
	InQueue                   AtomicInt
	InScheduled               AtomicInt
	InProcessing              AtomicInt
	TotalReceived             AtomicInt
	TotalProcessedOK          AtomicInt
	TotalProcessedError       AtomicInt
	TotalProcessedRescheduled AtomicInt
}
