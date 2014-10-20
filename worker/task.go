package worker

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/borgenk/qdo/config"
	"github.com/borgenk/qdo/log"
)

type Task struct {
	ID      string `json:"id"`
	Key     []byte `json:"key"`
	Target  string `json:"target"`
	Payload string `json:"payload"`
	Tries   int32  `json:"tries"`
	Delay   int32  `json:"delay"`
}

var (
	ErrClientBadRequest  = errors.New("Client error: bad request")
	ErrClientUnkonwn     = errors.New("Client error: unknown")
	ErrTaskInvalidTarget = errors.New("Task error: invalid task target")
	ErrTaskMaxTries      = errors.New("Task error: max tries reached")
)

// TASK(4)|tries(4)|delay(4)|sizeOfTarget(4)|target(x)|payload(x)
func (t *Task) Serialize() []byte {
	var (
		out          []byte
		tries        []byte = make([]byte, 4)
		delay        []byte = make([]byte, 4)
		sizeOfTarget []byte = make([]byte, 4)
	)

	binary.LittleEndian.PutUint32(tries, uint32(t.Tries))
	binary.LittleEndian.PutUint32(delay, uint32(t.Delay))
	binary.LittleEndian.PutUint32(sizeOfTarget, uint32(len(t.Target)))

	out = append([]byte("TASK"), tries...)
	out = append(out, delay...)
	out = append(out, sizeOfTarget...)
	out = append(out, []byte(t.Target)...)
	out = append(out, []byte(t.Payload)...)
	return out
}

func (t *Task) String() string {
	return "ID: " + t.ID + "\n" +
		"Target: " + t.Target + "\n" +
		"Tries: " + strconv.Itoa(int(t.Tries)) + "\n" +
		"Delay: " + strconv.Itoa(int(t.Delay)) + "\n" +
		"Payload: " + t.Payload
}

func (t *Task) Process(queueID *string, client *http.Client, config *Config, stats *Stats) error {
	log.Infof("queue/%s/task/%s - processing:\n%s", *queueID, t.ID, t.String())

	_, err := url.Parse(t.Target)
	if err != nil {
		// Assume invalid task, discard it.
		log.Error(fmt.Sprintf("queue/%s/task/%s - invalid target URL", *queueID, t.ID), err)
		return ErrTaskInvalidTarget
	}
	if config.TaskMaxTries > 0 && t.Tries >= config.TaskMaxTries {
		// Max tries reached.
		log.Infof("queue/%s/task/%s - max tries reached (%d)", *queueID, t.ID, t.Tries)
		return ErrTaskMaxTries
	}

	resp, err := client.Post(t.Target, "application/json", bytes.NewReader([]byte(t.Payload)))
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if err == nil && resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		// Task processed successfully.
		log.Infof("queue/%s/task/%s - completed successfully: %s", *queueID, t.ID, resp.Status)
		stats.TotalProcessedOK.Add(1)
		return nil
	} else if err == nil && resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		// Bad request, no point in retrying.
		log.Infof("queue/%s/task/%s - request invalid: %s", *queueID, t.ID, resp.Status)
		stats.TotalProcessedError.Add(1)
		return ErrClientBadRequest
	} else if err != nil {
		// Connection failure, retry later.
		log.Error(fmt.Sprintf("queue/%s/task/%s - connection failure", *queueID, t.ID), err)
		stats.TotalProcessedError.Add(1)
		return ErrClientUnkonwn
	} else {
		// Unknown error, retry later.
		log.Infof("queue/%s/task/%s - http status %d", *queueID, t.ID, resp.StatusCode)
		stats.TotalProcessedError.Add(1)
		return ErrClientUnkonwn
	}
}

func UnserializeTask(key, value []byte) *Task {
	task := &Task{}
	i := bytes.LastIndex(key, []byte(config.Prefix))
	ts := int32(binary.LittleEndian.Uint32(value[12:16]))

	task.ID = string(key[i+1 : len(key)])
	task.Key = key
	task.Tries = int32(binary.LittleEndian.Uint32(value[4:8]))
	task.Delay = int32(binary.LittleEndian.Uint32(value[8:12]))
	task.Target = string(value[16 : 16+ts])
	task.Payload = string(value[16+ts : len(value)])
	return task
}
