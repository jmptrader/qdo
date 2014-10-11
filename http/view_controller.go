package http

import (
	"fmt"
	"html/template"
	stdhttp "net/http"

	"github.com/borgenk/qdo/third_party/github.com/gorilla/mux"

	"github.com/borgenk/qdo/queue"
)

var templateList = []string{
	"dashboard.html",
	"queue_view.html",
	"queue_create.html",
}

func init() {
	r := GetRouter()
	r.HandleFunc("/", viewDashboard).Methods("GET")
	r.HandleFunc("/queue", viewDashboard).Methods("GET")
	r.HandleFunc("/queue/new", viewQueueCreate).Methods("GET", "POST")
	r.HandleFunc("/queue/{queue_id}", viewQueue).Methods("GET")
	r.HandleFunc("/queue/{queue_id}/{type}", viewQueue).Methods("GET")

	for _, v := range templateList {
		a, err := Asset("template/" + v)
		if err != nil {
			panic("template data not found")
		}
		b, err := Asset("template/layout.html")
		if err != nil {
			panic("template data not found")
		}
		c, err := Asset("template/top.html")
		if err != nil {
			panic("template data not found")
		}

		t := template.New(v)
		t.Funcs(FuncMap)
		t.Parse(string(a))
		t.Parse(string(b))
		t.Parse(string(c))
		registerTemplate(v, t)
	}
}

type Header struct {
	Title string
}

type Page struct {
	Header Header
	Title  string
	Result interface{}
}

type QueueRow struct {
	ID              string
	Status          string
	TasksProcessing int64
	TasksWaiting    int64
	TasksScheduled  int64
}

func viewDashboard(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	queues, err := queue.GetAllQueues()
	if err != nil {
		stdhttp.Error(w, "", stdhttp.StatusInternalServerError)
		return
	}

	res := []QueueRow{}
	for _, v := range queues {
		q := QueueRow{
			ID:              v.ID,
			Status:          "Active",
			TasksProcessing: v.Stats.InProcessing.Get(),
			TasksWaiting:    v.Stats.InQueue.Get(),
			TasksScheduled:  v.Stats.InScheduled.Get(),
		}
		res = append(res, q)
	}
	p := &Page{
		Header: Header{
			Title: "Dashboard | QDo",
		},
		Title:  "Dashboard",
		Result: res,
	}
	renderTemplate(w, "dashboard.html", p)
}

type QueueView struct {
	Q     *queue.QueueManager
	Type  string
	Stats *StatsResponse
	Tasks *[]queue.Task
}

func viewQueue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	vars := mux.Vars(r)
	queueID := vars["queue_id"]
	q, err := queue.GetQueue(queueID)
	if err != nil {
		return
	}
	qView := &QueueView{
		Q:    q,
		Type: vars["type"],
	}

	qView.Stats = &StatsResponse{}
	qView.Stats.Get(q)

	if vars["type"] == "" {
		qView.Tasks, err = q.GetTasks()
		if err != nil {
			return
		}
	} else {
		qView.Tasks, err = q.GetScheduledTasks()
		if err != nil {
			return
		}
	}
	h := Header{
		Title: fmt.Sprintf("%s | QDo", q.ID),
	}
	p := &Page{
		Header: h,
		Title:  q.ID,
		Result: qView,
	}
	renderTemplate(w, "queue_view.html", p)
}

func viewQueueCreate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method == "POST" {
		createQueue(w, r)
		stdhttp.Redirect(w, r, "/", 301)
		return
	}
	h := Header{
		Title: "QDo",
	}
	p := &Page{
		Header: h,
		Title:  "Create queue",
	}
	renderTemplate(w, "queue_create.html", p)
}
