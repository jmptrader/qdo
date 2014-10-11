package queue

import (
	"bytes"
	"fmt"
	"time"

	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
)

func NewScheduleQueue(ID string, config *Config, total *AtomicInt, db store.Store, notifySignal chan systemSignal, prefix, suffix []byte) *scheduleQueue {
	sq := &scheduleQueue{
		queueLine: queueLine{
			ID:           ID,
			Type:         "scheduler",
			config:       config,
			db:           db,
			notifySignal: notifySignal,
			prefix:       prefix,
			suffix:       suffix,
			total:        total,
		},
		readFreq: time.Second * 1,
	}

	return sq
}

type scheduleQueue struct {
	queueLine
	readFreq time.Duration
}

func (s *scheduleQueue) Run(fn func(*Task)) {
	for {
		select {
		case sig := <-s.notifySignal:
			if sig == stop {
				log.Infof("queue/%s/scheduler - stopping", s.ID)
				return
			}
		default:
		}

		//log.Debugf("queue/%s/scheduler: tick", s.ID)

		// Fetch all tasks scheduled earlier then right now, starting with the
		// oldest (lowest) possible item.
		stop := append(s.prefix, []byte(fmt.Sprintf("%d", time.Now().Unix()))...)
		iter := s.db.NewIterator(nil)
		for iter.Seek(s.prefix); iter.Valid(); iter.Next() {
			k := iter.Key()
			v := iter.Value()

			if bytes.Compare(k, stop) > 0 {
				// This might be a task sechduled in the future or some other
				// stored value. All scheduled tasks up until right now is read.
				break
			}
			//log.Debugf("queue/%s/scheduler - reading key %s", s.ID, k)

			fn(UnserializeTask(k, v))
			err = s.Delete(k)
			if err != nil {
				panic("2")
			}
		}
		iter.Close()
		time.Sleep(s.readFreq)
	}
}

func (s *scheduleQueue) Add(task *Task, scheduled int64) error {
	return s.add(task, scheduled)
}
