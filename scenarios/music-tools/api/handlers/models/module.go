package models

import (
	"net/http"

	modelspb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/models"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	modeldomain "music-tools/internal/models"
	"music-tools/internal/module"

	"github.com/gorilla/mux"
)

func Module(registry *modeldomain.Registry) module.Module {
	return module.Module{Name: "models", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/models", func(w http.ResponseWriter, req *http.Request) {
			response := &modelspb.ListModelsResponse{}
			for _, model := range registry.List() {
				response.Models = append(response.Models, wire(model))
			}
			write(w, http.StatusOK, response)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/models/{id}", func(w http.ResponseWriter, req *http.Request) {
			model, err := registry.Resolve(mux.Vars(req)["id"])
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			write(w, http.StatusOK, wire(model))
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func wire(model modeldomain.Model) *modelspb.Model {
	status := "available"
	if !model.Installed {
		status = "not_installed"
	}
	return &modelspb.Model{Id: model.ID, DisplayName: model.DisplayName, Variant: model.Variant, LicenseLane: string(model.Licence), Status: status, Revision: model.Revision, Bytes: model.Bytes}
}

func write(w http.ResponseWriter, status int, value proto.Message) {
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(value)
	if err != nil {
		http.Error(w, "model response encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
