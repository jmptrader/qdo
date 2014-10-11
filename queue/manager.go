package queue

import (
	"bytes"
	"errors"
	"sync"

	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
)

type systemSignal int

const (
	start  systemSignal = iota
	pause  systemSignal = iota
	resume systemSignal = iota
	stop   systemSignal = iota
)

const (
	prefix string = "\x00"
	suffix string = "\xff"
)

const (
	queueManagerKey string = "m"
)

type Manager struct {
	queues map[string]*QueueManager
	db     store.Store
	wg     *sync.WaitGroup
}

type Config struct {
	MaxConcurrent int32 `json:"max_concurrent"` // Number of simultaneous workers processing tasks.
	MaxRate       int32 `json:"max_rate"`       // Number of maxium task invocations from queue per second.
	TaskTimeout   int32 `json:"task_timeout"`   // Duration allowed per task to complete in seconds.
	TaskMaxTries  int32 `json:"task_max_tries"` // Number of tries per task before giving up. Set 0 for unlimited retries.
}

var (
	manager *Manager
	mu      sync.Mutex
	err     error
)

var (
	ErrQueueNotFound     = errors.New("Queue not found")
	ErrQueueAlreadyExist = errors.New("Queue already exist")
)

// StartManager starts the queue manager.
func StartManager(db store.Store) (*Manager, error) {
	manager = &Manager{
		queues: make(map[string]*QueueManager),
		wg:     &sync.WaitGroup{},
		db:     db,
	}
	err := manager.start()
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func (man *Manager) start() error {
	storedQueues, err := getAllStoredQueues()
	if err != nil {
		return err
	}
	for _, v := range *storedQueues {
		man.queues[v.ID] = &v
		man.queues[v.ID].db = man.db
		go man.queues[v.ID].Initialize(man.wg).Start()
		man.wg.Add(1)
	}
	return nil
}

// Stop waits for all active queues to stop before cleaning up resources and
// exiting.
func (man *Manager) Stop() {
	for _, queue := range manager.queues {
		go queue.Stop()
	}
	man.wg.Wait()
	man.db.Close()
}

// GetQueue returns a queue manager for a given id.
func GetQueue(queueID string) (*QueueManager, error) {
	mu.Lock()
	defer mu.Unlock()

	if manager == nil {
		return nil, errors.New("Manager not initialized")
	}
	res, ok := manager.queues[queueID]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return res, nil
}

// GetAllQueues returns all queue managers.
func GetAllQueues() ([]*QueueManager, error) {
	mu.Lock()
	defer mu.Unlock()

	if manager == nil {
		return nil, errors.New("Manager not initialized")
	}
	res := make([]*QueueManager, 0, len(manager.queues))
	for _, v := range manager.queues {
		res = append(res, v)
	}
	return res, nil
}

// getAllStoredQueues retrieves all stored queue managers.
func getAllStoredQueues() (*[]QueueManager, error) {
	res := []QueueManager{}
	iter := manager.db.NewIterator(nil)
	defer iter.Close()
	for iter.Seek([]byte(queueManagerKey + prefix)); iter.Valid(); iter.Next() {
		if bytes.Compare(iter.Key(), []byte(queueManagerKey+suffix)) > 0 {
			break
		}
		c := QueueManager{}
		err := GobDecode(iter.Value(), &c)
		if err != nil {
			panic("for now")
		}
		res = append(res, c)
	}
	return &res, nil
}

// AddQueue adds a new queue manager and starts it automatically. QueueID must
// be unique.
func AddQueue(queueID string, config *Config) error {
	mu.Lock()
	defer mu.Unlock()

	if manager == nil {
		return errors.New("Manager not initialized")
	}

	_, ok := manager.queues[queueID]
	if ok {
		return ErrQueueAlreadyExist
	}

	queue := NewQueue(queueID, config, manager.db, manager.wg)
	manager.queues[queueID] = queue

	b, err := GobEncode(queue)
	if err != nil {
		log.Error("", err)
		return err
	}
	err = manager.db.Put(getQueueManagerKey(queueID), b)
	if err != nil {
		log.Error("creating new queue failed", err)
		return err
	}

	go func() {
		manager.queues[queueID].Start()
	}()
	manager.wg.Add(1)
	return nil
}

// RemoveQueue stops and removes the queue. The queue will wait on
// running tasks to complete before shutting down.
func RemoveQueue(queueID string) error {
	mu.Lock()
	defer mu.Unlock()

	if manager == nil {
		return errors.New("Manager not initialized")
	}

	// Check if queue exist.
	_, ok := manager.queues[queueID]
	if !ok {
		return errors.New("Queue does not exist")
	}
	err := manager.db.Delete(getQueueManagerKey(queueID))
	if err != nil {
		return err
	}
	go func() {
		manager.queues[queueID].Stop()
		delete(manager.queues, queueID)
		// TODO: Clean up tasks.
	}()
	return nil
}

// getQueueManagerKey builds the queue manager key prefix.
func getQueueManagerKey(queueID string) []byte {
	return []byte(queueManagerKey + prefix + queueID)
}
