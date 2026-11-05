package ontology

import (
	"time"

	"tech-tree-designer/internal/module"
	ontologydomain "tech-tree-designer/internal/ontology"

	"github.com/gorilla/mux"

	ontologyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology/ontology_v1connect"
)

const timeFormat = time.RFC3339Nano

func Module(service *ontologydomain.Service) module.Module {
	h := NewHandler(service)
	return module.Module{
		Name: "ontology",
		Mount: func(r *mux.Router) {
			path, handler := ontologyconnect.NewOntologyServiceHandler(h)
			r.PathPrefix(path).Handler(handler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return ontologydomain.Schema() }
