package jobs

import (
	"github.com/gorilla/mux"
	internal "prose-studio/internal/jobs"
	"prose-studio/internal/module"
)

func Module(service *internal.Service) module.Module {
	h := New(service)
	return module.Module{Name: "jobs", Mount: func(r *mux.Router) { r.PathPrefix("/api/v1/jobs").Handler(h) }, Endpoints: Endpoints}
}
