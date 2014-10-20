package worker

import (
	"bytes"
	"fmt"

	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
	"github.com/borgenk/qdo/config"
)

type queueLine struct {
	ID           string
	Type         string
	config       *Config
	db           store.Store
	notifySignal chan systemSignal
	prefix       []byte
	suffix       []byte
	total        *AtomicInt
}

// Key format: [line id] \x00 [key type] \x00 [timestamp] \x00 [task id]
func (q *queueLine) key(task *Task, timestamp int64) []byte {
	return append(q.prefix, []byte(fmt.Sprintf("%d%s%s", timestamp, config.Prefix, task.ID))...)
}

func (q *queueLine) add(task *Task, scheduled int64) error {
	log.Infof("queue/%s/%s/task/%s - adding", q.ID, q.Type, task.ID)

	task.Key = q.key(task, scheduled)
	err := q.db.Put(task.Key, task.Serialize())
	if err != nil {
		log.Error(fmt.Sprintf("queue/%s/%s/task/%s - adding failed", q.ID, q.Type, task.ID), err)
		return err
	}
	q.total.Add(1)
	return nil
}

func (q *queueLine) Get() {

}

func (q *queueLine) GetAll() (*[]Task, error) {
	i := 0
	limit := 100
	result := []Task{}

	iter := q.db.NewIterator(nil)
	defer iter.Close()
	for iter.Seek(q.prefix); iter.Valid(); iter.Next() {
		if bytes.Compare(iter.Key(), q.suffix) > 0 {
			break
		}

		task := UnserializeTask(iter.Key(), iter.Value())
		result = append(result, *task)

		i++
		if i >= limit {
			break
		}
	}
	return &result, nil
}

func (q *queueLine) Delete(key []byte) error {
	log.Debugf("queue/%s/%s/task/x - deleting", q.ID, q.Type)

	err := q.db.Delete(key)
	if err != nil {
		log.Error(fmt.Sprintf("queue/%s/%s/task/x - deleting", q.ID, q.Type), err)
		return err
	}
	q.total.Add(-1)
	return nil
}

func (q *queueLine) DeleteAll() {

}
