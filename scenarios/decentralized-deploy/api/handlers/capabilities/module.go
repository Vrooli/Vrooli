package capabilities

import (
	"decentralized-deploy/internal/capabilities"
	"decentralized-deploy/internal/module"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	deploymentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func Module(registry *capabilities.Registry) module.Module {
	return module.Module{
		Name: "capabilities",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/capabilities/describe", func(w http.ResponseWriter, req *http.Request) {
				if registry == nil {
					http.Error(w, "capabilities registry is not configured", http.StatusServiceUnavailable)
					return
				}
				data, err := registry.Describe(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				var report map[string]any
				if err := json.Unmarshal(data, &report); err != nil {
					http.Error(w, "capability report is not valid JSON", http.StatusInternalServerError)
					return
				}
				structure, err := structpb.NewStruct(report)
				if err != nil {
					http.Error(w, "capability report cannot be represented as a protobuf Struct", http.StatusInternalServerError)
					return
				}
				body, err := protojson.Marshal(&deploymentv1.CapabilitiesDescription{Report: structure})
				if err != nil {
					http.Error(w, "capability report could not be encoded", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(body)
			}).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
