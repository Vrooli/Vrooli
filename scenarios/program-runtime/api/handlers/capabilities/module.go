package capabilities

import (
	"encoding/json"
	"net/http"

	"program-runtime/internal/capabilities"
	"program-runtime/internal/module"

	"github.com/gorilla/mux"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/capabilities"
	"google.golang.org/protobuf/encoding/protojson"
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
				var response capabilitiesv1.DescribeResponse
				if err := json.Unmarshal(data, &response); err != nil {
					http.Error(w, "capabilities registry returned an invalid typed payload", http.StatusInternalServerError)
					return
				}
				data, err = protojson.Marshal(&response)
				if err != nil {
					http.Error(w, "capabilities payload could not be encoded as protojson", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
			}).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
