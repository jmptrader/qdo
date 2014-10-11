package http

import (
	"encoding/json"
	stdhttp "net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/borgenk/qdo/third_party/github.com/gorilla/mux"

	"github.com/borgenk/qdo/log"
	"github.com/borgenk/qdo/queue"
)

func init() {
	r := GetRouter()
	r.HandleFunc("/api/queue", getAllQueues).Methods("GET")
	r.HandleFunc("/api/queue", createQueue).Methods("POST")
	r.HandleFunc("/api/queue/{queue_id}", getQueue).Methods("GET")
	r.HandleFunc("/api/queue/{queue_id}", deleteQueue).Methods("DELETE")
	r.HandleFunc("/api/queue/{queue_id}/task", getAllTasks).Methods("GET")
	r.HandleFunc("/api/queue/{queue_id}/task", CreateTask).Methods("POST")
	r.HandleFunc("/api/queue/{queue_id}/task", deleteAllTasks).Methods("DELETE")
	r.HandleFunc("/api/queue/{queue_id}/stats", getStats).Methods("GET")
}

type jsonListResult struct {
	Object string      `json:"object"`
	URL    string      `json:"url"`
	Count  int         `json:"count"`
	Data   interface{} `json:"data"`
}

func JSONListResult(url string, count int, data interface{}) *jsonListResult {
	res := &jsonListResult{
		Object: "list",
		URL:    url,
		Count:  count,
		Data:   data,
	}
	return res
}

func ReturnJSON(w stdhttp.ResponseWriter, r *stdhttp.Request, resp interface{}) {
	if resp == nil {
		w.WriteHeader(stdhttp.StatusOK)
		return
	}
	pretty := r.FormValue("pretty")
	if pretty != "" {
		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
			return
		}
		w.Write(b)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(stdhttp.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// API handler for GET /api/queue.
func getAllQueues(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	res, err := queue.GetAllQueues()
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, JSONListResult("/api/queue", len(res), res))
}

// API handler for POST /api/queue.
func createQueue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var (
		v   int
		err error
	)
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	queueID := reg.ReplaceAllString(r.FormValue("queue_id"), "")
	queueID = strings.ToLower(strings.Trim(queueID, " "))
	if queueID == "" {
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	config := &queue.Config{}

	v, err = strconv.Atoi(r.FormValue("max_concurrent"))
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	config.MaxConcurrent = int32(v)
	v, err = strconv.Atoi(r.FormValue("max_rate"))
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	config.MaxRate = int32(v)
	v, err = strconv.Atoi(r.FormValue("task_timeout"))
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	config.TaskTimeout = int32(v)
	v, err = strconv.Atoi(r.FormValue("task_max_tries"))
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
	config.TaskMaxTries = int32(v)
	err = queue.AddQueue(queueID, config)
	if err != nil {
		log.Error("", err)
		stdhttp.Error(w, "", stdhttp.StatusBadRequest)
		return
	}
}

// API handler for GET /api/queue/{queue_id}.
func getQueue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]
	res, err := queue.GetQueue(queueID)
	if err != nil {
		stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
		return
	}
	if res == nil {
		stdhttp.Error(w, "", stdhttp.StatusNotFound)
		return
	}
	ReturnJSON(w, r, res)
}

// API handler for DELETE /api/queue/{queue_id}.
func deleteQueue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]
	err := queue.RemoveQueue(queueID)
	if err != nil {
		stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, nil)
}

// API handler for GET /api/queue/{queue_id}/task
func getAllTasks(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]
	q, err := queue.GetQueue(queueID)
	if err != nil {
		stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
		return
	}
	if q == nil {
		stdhttp.Error(w, "", stdhttp.StatusNotFound)
		return
	}
	res, err := q.GetTasks()
	if err != nil {
		stdhttp.Error(w, "could not fetch tasks", stdhttp.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, JSONListResult("/api/queue/"+queueID+"/task", len(*res), res))
}

// API handler for POST /api/queue/{queue_id}/task.
func CreateTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]

	q, err := queue.GetQueue(queueID)
	if err != nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
		return
	}
	if q == nil {
		stdhttp.Error(w, "queue id does not exsist", stdhttp.StatusBadRequest)
		return
	}
	scheduled := 0
	if r.FormValue("scheduled") != "" {
		scheduled, err = strconv.Atoi(r.FormValue("scheduled"))
		if err != nil || scheduled < 0 {
			stdhttp.Error(w, "value for scheduled is invalid", stdhttp.StatusBadRequest)
			return
		}
	}

	res, err := q.AddTask(r.FormValue("target"), r.FormValue("payload"),
		int64(scheduled))
	if err != nil {
		stdhttp.Error(w, "could not add task to database", stdhttp.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, res)
}

// API handler for DELETE /api/queue/{queue_id}/task.
func deleteAllTasks(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]

	q, err := queue.GetQueue(queueID)
	if err != nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
		return
	}
	if q == nil {
		stdhttp.Error(w, "queue id does not exsist", stdhttp.StatusBadRequest)
		return
	}
	err = q.Flush()
	if err != nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, nil)
}

type StatsResponse struct {
	Object                    string `json:"object"`
	InQueue                   int64  `json:"in_queue"`
	InProcessing              int64  `json:"in_processing"`
	InScheduled               int64  `json:"in_scheduled"`
	TotalProcessed            int64  `json:"total_processed"`
	TotalProcessedOK          int64  `json:"total_processed_ok"`
	TotalProcessedError       int64  `json:"total_processed_error"`
	TotalProcessedRescheduled int64  `json:"total_processed_rescheduled"`
	TotalTime                 int64  `json:"total_time"`
	TimeLastOK                int64  `json:"time_last_ok"`
}

func (s *StatsResponse) Get(q *queue.QueueManager) {
	s.InQueue = q.Stats.InQueue.Get()
	s.InProcessing = q.Stats.InProcessing.Get()
	s.InScheduled = q.Stats.InScheduled.Get()
	s.TotalProcessed = q.Stats.TotalProcessed.Get()
	s.TotalProcessedOK = q.Stats.TotalProcessedOK.Get()
	s.TotalProcessedError = q.Stats.TotalProcessedError.Get()
	s.TotalProcessedRescheduled = q.Stats.TotalProcessedRescheduled.Get()
	s.TotalTime = q.Stats.TotalTime.Get()
	s.TimeLastOK = q.Stats.TimeLastOK.Get()
}

// API handler for GET /api/queue/{queue_id}/stats
func getStats(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]

	q, err := queue.GetQueue(queueID)
	if err != nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
		return
	}
	if q == nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusNotFound)
		return
	}

	statsResp := &StatsResponse{}
	statsResp.Get(q)
	ReturnJSON(w, r, statsResp)
}
