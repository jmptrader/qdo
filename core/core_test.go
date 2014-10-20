package core

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/borgenk/qdo/log"
	_ "github.com/borgenk/qdo/log/stdout"
	"github.com/borgenk/qdo/queue"
	"github.com/borgenk/qdo/store"
	_ "github.com/borgenk/qdo/store/leveldb"
)

var resultPortal chan string

type TestPayload struct {
	Value string
}

func testSetup() (*queue.Manager, error) {
	resultPortal = make(chan string)
	log.InitLog(log.New())

	err := os.RemoveAll("/tmp/qdotest")
	if err != nil {
		return nil, err
	}

	store, _ := store.GetStoreConstructor("leveldb", "/tmp/qdotest")
	manager, err := queue.StartManager(store)
	if err != nil {
		return nil, err
	}
	c := &queue.Config{
		MaxConcurrent: 5,
		MaxRate:       100,
		TaskTimeout:   1,
		TaskMaxTries:  1,
	}
	queue.AddQueue("test", c)
	time.Sleep(1 * time.Second)

	/*
	   http.HandleFunc("/", handler)
	   go http.ListenAndServe(":9999", nil)*/

	return manager, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t TestPayload
	err := decoder.Decode(&t)
	if err != nil {
		resultPortal <- ""
		return
	}
	resultPortal <- t.Value
}

func TestConveyorTaskProcess(t *testing.T) {
	_, err := testSetup()
	if err != nil {
		t.Error(err)
	}
	payload := TestPayload{Value: "12345"}
	p, err := json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}
	q, err := queue.GetQueue("test")
	if err != nil {
		t.Error(err)
	}
	if q == nil {
		t.Error(err)
	}
	q.AddTask("http://localhost:9999", string(p), 0)
	result := <-resultPortal
	if result != "12345" {
		t.Errorf("Expected result %s, got %s", "12345", result)
	}
}

// $ go test -c
// $ ./queue.test -test.run=20 -test.bench=BenchmarkQueueAddTask -test.benchtime=15s -test.cpuprofile=queue.prof
// $ go tool pprof queue.test queue.prof
// $ go tool pprof queue.test queue.prof --gif > queue.gif
// (pprof) web
// 37831 ns/op
// 36198
// 102578 ns/op
// 37725 ns/op
func BenchmarkQueueAddTask(b *testing.B) {
	b.StopTimer()
	_, err := testSetup()
	if err != nil {
		b.Error(err)
	}
	payload := TestPayload{Value: "12345"}
	p, err := json.Marshal(payload)
	if err != nil {
		b.Error(err)
	}
	q, err := queue.GetQueue("test")
	if err != nil {
		b.Error(err)
	}
	if q == nil {
		b.Error(err)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		q.AddTask("http://localhost:9999", string(p), 0)
	}

	b.StopTimer()
	//manager.Stop()
}
