package worker

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
)

func NewWaitQueue(ID string, config *Config, total *AtomicInt, db store.Store, notifySignal chan systemSignal, prefix, suffix []byte) *waitQueue {
	var locker sync.Mutex

	wq := &waitQueue{
		queueLine: queueLine{
			ID:           ID,
			Type:         "waitinglist",
			config:       config,
			db:           db,
			notifySignal: notifySignal,
			prefix:       prefix,
			suffix:       suffix,
			total:        total,
		},
		notifyReady: make(chan int, config.MaxConcurrent),
		rewind:      sync.NewCond(&locker),
	}
	wq.counterTime.Set(0)
	wq.counter.Set(0)
	return wq
}

type waitQueue struct {
	queueLine
	notifyReady chan int
	rewind      *sync.Cond
	counterTime AtomicInt
	counter     AtomicInt
}

func (w *waitQueue) Run(fn func(*Task)) {
	var (
		iter store.Iterator
		k    []byte
		v    []byte
	)

	for {
		select {
		case <-w.notifySignal:
			return
		default:
		}

		iter = w.db.NewIterator(nil)

		if k != nil {
			// Seek to last read key.
			iter.Seek(k)
			if bytes.Compare(iter.Key(), k) != 0 {
				// Last read key is deleted and it's now pointing to next key,
				// go one step back so for loop goes into right position.
				iter.Prev()
			}
		} else {
			iter.Seek(w.prefix)
			iter.Prev()
		}

		for iter.Next() {
			select {
			case <-w.notifySignal:
				return
			default:
			}

			k = iter.Key()
			v = iter.Value()

			if bytes.Compare(k, w.suffix) > 0 {
				// End of list reached.
				break
			}

			//log.Debugf("queue/%s/waitinglist: reading key %s", w.ID, k)

			// Block until conveyor is ready to process next task.
			w.notifyReady <- 1

			fn(UnserializeTask(k, v))

			// Throttle task invocations per second (processing + maxRate for now).
			if w.config.MaxRate > 0 {
				time.Sleep(time.Duration(int64(time.Second) / int64(w.config.MaxRate)))
			}
		}

		//log.Debugf("queue/%s/waitinglist: waiting on signal", w.ID)

		iter.Close()

		w.rewind.L.Lock()
		w.rewind.Wait()
		w.rewind.L.Unlock()
	}
}

func (w *waitQueue) Add(task *Task) error {
	now := time.Now().Unix()
	if w.counter.Get() > 99999 {
		return errors.New("Too many tasks at once")
	}
	if now != w.counterTime.Get() {
		w.counterTime.Set(now)
		w.counter.Set(0)
	} else {
		w.counter.Add(1)
	}
	err := w.add(task, fmt.Sprintf("%d%05d", now, w.counter.Get()))
	if err != nil {
		return err
	}

	// Signal new task to queue reader.
	w.Trigger()
	return nil
}

func (w *waitQueue) Trigger() {
	w.rewind.L.Lock()
	w.rewind.Signal()
	w.rewind.L.Unlock()
}
