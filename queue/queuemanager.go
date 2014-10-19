package queue

import (
	"crypto/sha1"
	"fmt"
	"net"
	stdhttp "net/http"
	"sync"
	"time"

	"github.com/borgenk/qdo/third_party/github.com/bmizerany/perks/quantile"

	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/store"
)

const (
	queueKey         string = "q"
	waitQueueKey     string = "w"
	scheduleQueueKey string = "s"
)

// NewQueue creates a new queue ready to handle tasks after running
// initialize on it self.
func NewQueue(queueID string, config *Config, db store.Store, wg *sync.WaitGroup) *QueueManager {
	q := &QueueManager{
		ID:        queueID,
		CreatedAt: time.Now(),
		Config:    config,
		Stats:     &Stats{},
		db:        db,
	}
	return q.Initialize(wg)
}

type QueueManager struct {
	ID                      string
	CreatedAt               time.Time
	Config                  *Config
	Stats                   *Stats
	StatsAddQuantile        *quantile.Stream
	StatsProcessingQuantile *quantile.Stream
	db                      store.Store
	httpClient              *stdhttp.Client
	newTaskID               chan string
	notifySignal            chan systemSignal
	mWaitGroup              *sync.WaitGroup
	qmWaitGroup             *sync.WaitGroup
	waitQueue               *waitQueue
	scheduleQueue           *scheduleQueue
}

func (q *QueueManager) Initialize(mWaitGroup *sync.WaitGroup) *QueueManager {
	// Signals events to queue lines (i.e. stop).
	q.notifySignal = make(chan systemSignal)

	// Mananger wait group.
	q.mWaitGroup = mWaitGroup

	// Queue manager wait group.
	q.qmWaitGroup = &sync.WaitGroup{}

	// Initialize internal queue lines.
	q.initInternalQueues()

	// Initialize HTTP client.
	q.initHTTPClient()

	// Initialize quantile stats.
	q.StatsAddQuantile = quantile.NewTargeted(0.50, 0.90, 0.99)
	q.StatsProcessingQuantile = quantile.NewTargeted(0.50, 0.90, 0.99)

	// Task ID generator.
	// http://blog.cloudflare.com/go-at-cloudflare
	q.newTaskID = make(chan string)
	go func() {
		h := sha1.New()
		c := []byte(time.Now().String())
		for {
			h.Write(c)
			q.newTaskID <- fmt.Sprintf("%x", h.Sum(nil))
		}
	}()

	// TODO: Fill stats (in waiting, in schedule, etc)

	return q
}

// initInternalQueues initializes the internal queue lines; wait and
// schedule queue.
func (q *QueueManager) initInternalQueues() {
	q.waitQueue = NewWaitQueue(q.ID, q.Config, &q.Stats.InQueue, q.db, q.notifySignal,
		[]byte(queueKey+prefix+q.ID+prefix+waitQueueKey+prefix),
		[]byte(queueKey+prefix+q.ID+prefix+waitQueueKey+suffix))

	q.scheduleQueue = NewScheduleQueue(q.ID, q.Config, &q.Stats.InScheduled, q.db, q.notifySignal,
		[]byte(queueKey+prefix+q.ID+prefix+scheduleQueueKey+prefix),
		[]byte(queueKey+prefix+q.ID+prefix+scheduleQueueKey+suffix))
}

// initHTTPClient initializes the HTTP client.
func (q *QueueManager) initHTTPClient() {
	transport := stdhttp.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, time.Duration(q.Config.TaskTimeout)*time.Second)
		},
		Proxy: stdhttp.ProxyFromEnvironment,
		ResponseHeaderTimeout: time.Duration(q.Config.TaskTimeout) * time.Second,
	}
	q.httpClient = &stdhttp.Client{
		Transport: &transport,
	}
}

func (q *QueueManager) Start() {
	defer q.mWaitGroup.Done()

	log.Infof("queue/%s - starting with %d worker(s)", q.ID, q.Config.MaxConcurrent)
	go q.waitQueue.Run(func(task *Task) {
		q.processTask(task)
	})
	go q.scheduleQueue.Run(func(task *Task) {
		q.rescheduleTask(task)
	})
	// Wait for all tasks currently processing to end.
	q.qmWaitGroup.Wait()
}

func (q *QueueManager) Stop() {
	if q.notifySignal == nil {
		panic("notifySignal not created")
	}
	q.notifySignal <- stop
	// Wait queue might be stuck on a "wait for new task signal".
	q.waitQueue.Trigger()
}

func (q *QueueManager) rescheduleTask(task *Task) {
	err = q.waitQueue.Add(task)
	if err != nil {
		panic("Unable to add task to wait queue")
	}
}

func (q *QueueManager) processTask(task *Task) {
	q.qmWaitGroup.Add(1)

	go func() {
		defer func() {
			<-q.waitQueue.notifyReady
			q.qmWaitGroup.Done()
		}()

		start := time.Now()

		k := task.Key

		err := task.Process(&q.ID, q.httpClient, q.Config, q.Stats)
		if err == ErrTaskMaxTries {
			// Do nothing.
		} else if err != nil {
			if task.Delay == 0 {
				task.Delay = 1
			}
			task.Delay = task.Delay * 2
			task.Tries = task.Tries + 1

			err = q.scheduleQueue.Add(task, int64(task.Delay)+time.Now().Unix())
			if err != nil {
				panic("Unable to add task to schedule queue")
			}
		} else if err == nil {
			elapsed := time.Since(start)
			q.StatsProcessingQuantile.Insert(float64(elapsed / time.Millisecond))
		}
		err = q.waitQueue.Delete(k)
		if err != nil {
			panic("Unable to delete task from wait queue")
		}
	}()
}

func (q *QueueManager) AddTask(target, payload string, scheduled int64) (*Task, error) {
	start := time.Now()
	task := &Task{
		ID:      <-q.newTaskID,
		Target:  target,
		Payload: payload,
		Tries:   0,
		Delay:   0,
	}
	if scheduled == 0 {
		// Normal task.
		err = q.waitQueue.Add(task)
		if err != nil {
			return nil, err
		}
	} else {
		// Delayed task.
		err = q.scheduleQueue.Add(task, scheduled)
		if err != nil {
			return nil, err
		}
	}
	q.Stats.TotalReceived.Add(1)
	elapsed := time.Since(start)
	q.StatsAddQuantile.Insert(float64(elapsed / time.Millisecond))
	return task, nil
}

func (q *QueueManager) GetTasks() (*[]Task, error) {
	return q.waitQueue.GetAll()
}

func (q *QueueManager) GetScheduledTasks() (*[]Task, error) {
	return q.scheduleQueue.GetAll()
}

func (q *QueueManager) Flush() error {
	return nil
}
