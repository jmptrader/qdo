package http

import (
	"fmt"
	"html/template"
	stdhttp "net/http"
	"time"

	"github.com/borgenk/qdo/third_party/github.com/elazarl/go-bindata-assetfs"
	"github.com/borgenk/qdo/third_party/github.com/gorilla/mux"
)

var router = mux.NewRouter()

func GetRouter() *mux.Router {
	return router
}

func RouterParam(r *stdhttp.Request, name string) string {
	return mux.Vars(r)[name]
}

var Templates = make(map[string]*template.Template)

var FuncMap = template.FuncMap{
	"ShortID": func(ID string) string { return ID[:7] },
	"eq": func(a, b interface{}) bool {
		return a == b
	},
}

func registerTemplate(name string, t *template.Template) {
	Templates[name] = t
}

func renderTemplate(w stdhttp.ResponseWriter, tmpl string, p interface{}) {
	err := Templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
	}
}

func Run(port int) {
	r := GetRouter()
	r.PathPrefix("/static/").Handler(
		stdhttp.StripPrefix("/static/",
			stdhttp.FileServer(
				&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "static"})))

	srv := &stdhttp.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     r,
		ReadTimeout: 30 * time.Second,
	}
	srv.ListenAndServe()
}
