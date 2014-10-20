package core

import (
	"bytes"
	"errors"
	"sync"

	"github.com/borgenk/qdo/config"
	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
	"github.com/borgenk/qdo/worker"
)

type Controller struct {
	queues map[string]*worker.QueueManager
	db     store.Store
	wg     *sync.WaitGroup
}

var (
	controller *Controller
	mu         sync.Mutex
)

var (
	ErrQueueNotFound     = errors.New("Queue not found")
	ErrQueueAlreadyExist = errors.New("Queue already exist")
	ErrControllerNotInit = errors.New("Controller not initialized")
)

// StartController starts the queue controller.
func StartController(db store.Store) (*Controller, error) {
	controller = &Controller{
		queues: make(map[string]*worker.QueueManager),
		wg:     &sync.WaitGroup{},
		db:     db,
	}
	err := controller.start()
	if err != nil {
		return nil, err
	}
	return controller, nil
}

func (c *Controller) start() error {
	storedQueues, err := getAllStoredQueues()
	if err != nil {
		return err
	}
	for _, v := range *storedQueues {
		c.queues[v.ID] = &v
		go c.queues[v.ID].Initialize(c.db, c.wg).Start()
		c.wg.Add(1)
	}
	return nil
}

// Stop waits for all active queues to stop before cleaning up resources and
// exiting.
func (c *Controller) Stop() {
	for _, queue := range controller.queues {
		go queue.Stop()
	}
	c.wg.Wait()
	c.db.Close()
}

// GetQueue returns a queue controller for a given id.
func GetQueue(queueID string) (*worker.QueueManager, error) {
	mu.Lock()
	defer mu.Unlock()

	if controller == nil {
		return nil, ErrControllerNotInit
	}
	res, ok := controller.queues[queueID]
	if !ok {
		return nil, ErrQueueNotFound
	}
	return res, nil
}

// GetAllQueues returns all queue managers.
func GetAllQueues() ([]*worker.QueueManager, error) {
	mu.Lock()
	defer mu.Unlock()

	if controller == nil {
		return nil, ErrControllerNotInit
	}
	res := make([]*worker.QueueManager, 0, len(controller.queues))
	for _, v := range controller.queues {
		res = append(res, v)
	}
	return res, nil
}

// getAllStoredQueues retrieves all stored queue managers.
func getAllStoredQueues() (*[]worker.QueueManager, error) {
	res := []worker.QueueManager{}
	iter := controller.db.NewIterator(nil)
	defer iter.Close()
	for iter.Seek([]byte(config.QueueManagerKey + config.Prefix)); iter.Valid(); iter.Next() {
		if bytes.Compare(iter.Key(), []byte(config.QueueManagerKey+config.Suffix)) > 0 {
			break
		}
		c := worker.QueueManager{}
		err := GobDecode(iter.Value(), &c)
		if err != nil {
			panic("for now")
		}
		res = append(res, c)
	}
	return &res, nil
}

// AddQueue adds a new queue controller and starts it automatically. QueueID must
// be unique.
func AddQueue(queueID string, config *worker.Config) error {
	mu.Lock()
	defer mu.Unlock()

	if controller == nil {
		return ErrControllerNotInit
	}

	_, ok := controller.queues[queueID]
	if ok {
		return ErrQueueAlreadyExist
	}

	queue := worker.NewQueue(queueID, config, controller.db, controller.wg)
	controller.queues[queueID] = queue

	b, err := GobEncode(queue)
	if err != nil {
		log.Error("", err)
		return err
	}
	err = controller.db.Put(getQueueManagerKey(queueID), b)
	if err != nil {
		log.Error("creating new queue failed", err)
		return err
	}

	go func() {
		controller.queues[queueID].Start()
	}()
	controller.wg.Add(1)
	return nil
}

// RemoveQueue stops and removes the queue. The queue will wait on
// running tasks to complete before shutting down.
func RemoveQueue(queueID string) error {
	mu.Lock()
	defer mu.Unlock()

	if controller == nil {
		return ErrControllerNotInit
	}

	// Check if queue exist.
	_, ok := controller.queues[queueID]
	if !ok {
		return ErrQueueNotFound
	}
	err := controller.db.Delete(getQueueManagerKey(queueID))
	if err != nil {
		return err
	}
	go func() {
		controller.queues[queueID].Stop()
		delete(controller.queues, queueID)
		// TODO: Clean up tasks.
	}()
	return nil
}

// getQueueManagerKey builds the queue controller key prefix.
func getQueueManagerKey(queueID string) []byte {
	return []byte(config.QueueManagerKey + config.Prefix + queueID)
}
