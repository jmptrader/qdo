package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/borgenk/qdo/third_party/github.com/gorilla/mux"

	"github.com/borgenk/qdo/lib/log"
	"github.com/borgenk/qdo/lib/queue"
)

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

func ReturnJSON(w http.ResponseWriter, r *http.Request, resp interface{}) {
	pretty := r.FormValue("pretty")
	if pretty != "" {
		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		w.Write(b)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func getStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conveyorID := vars["conveyor_id"]

	conveyor, err := queue.GetConveyor(conveyorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conveyor == nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	res, err := conveyor.Stats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, res)
}

func getAllTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conveyorID := vars["conveyor_id"]

	res, err := queue.GetAllTasks(conveyorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, JSONListResult("/api/conveyor/"+conveyorID+"/task", len(res), res))
}

func createTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conveyorID := vars["conveyor_id"]

	conveyor, err := queue.GetConveyor(conveyorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conveyor == nil {
		http.Error(w, "conveyor id does not exsist", http.StatusBadRequest)
		return
	}
	res, err := conveyor.AddTask(r.FormValue("target"), r.FormValue("payload"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ReturnJSON(w, r, res)
}

// List all active conveyors.
// API handler for GET /api/conveyor.
func getAllConveyor(w http.ResponseWriter, r *http.Request) {
	res := queue.GetAllConveyor()
	ReturnJSON(w, r, JSONListResult("/api/conveyor", len(res), res))
}

// Creates a new conveyor.
// API handler for POST /api/conveyor.
func createConveyor(w http.ResponseWriter, r *http.Request) {
	conveyorID := r.FormValue("conveyor_id")

	config := &queue.Config{}
	var (
		v   int
		err error
	)
	v, err = strconv.Atoi(r.FormValue("workers"))
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	config.NWorker = int32(v)

	v, err = strconv.Atoi(r.FormValue("throttle"))
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	config.Throttle = int32(v)

	v, err = strconv.Atoi(r.FormValue("task_t_limit"))
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	config.TaskTLimit = int32(v)

	v, err = strconv.Atoi(r.FormValue("task_max_tries"))
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	config.TaskMaxTries = int32(v)

	v, err = strconv.Atoi(r.FormValue("log_size"))
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	config.LogSize = int32(v)

	err = queue.AddConveyor(conveyorID, config)
	if err != nil {
		log.Error("", err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
}

func updateConveyor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["conveyor_id"]
	conv, err := queue.GetConveyor(id)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	if conv == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	paused := r.FormValue("paused")
	if paused == "true" {
		conv.Pause()
	} else if paused == "false" {
		conv.Resume()
	}
	ReturnJSON(w, r, conv)
}

// List all active conveyors.
// API handler for GET /api/conveyor/{conveyor_id}.
func getConveyor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["conveyor_id"]
	res, err := queue.GetConveyor(id)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	if res == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	ReturnJSON(w, r, res)
}
