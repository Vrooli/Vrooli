package models

import (
	"net/http"

	modelspb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/models"
	"google.golang.org/protobuf/proto"
	"music-tools/internal/httpx"

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
		r.HandleFunc("/api/v1/models/{id:.+}", func(w http.ResponseWriter, req *http.Request) {
			model, err := registry.Resolve(mux.Vars(req)["id"])
			if err != nil {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
				return
			}
			write(w, http.StatusOK, wire(model))
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func wire(model modeldomain.Model) *modelspb.Model {
	status := "available"
	if !model.Installed {
		status = "runtime_unverified"
	}
	out := &modelspb.Model{Id: model.ID, DisplayName: model.DisplayName, Variant: model.Variant, LicenseLane: string(model.Licence), Status: status, Revision: model.Revision, Bytes: model.Bytes, Engine: model.Engine,
		SungLyrics:     &modelspb.ModelCapability{Supported: model.SungLyrics.Supported, Reason: model.SungLyrics.Reason, Source: model.SungLyrics.Source},
		VoiceReference: &modelspb.ModelCapability{Supported: model.VoiceReference.Supported, Reason: model.VoiceReference.Reason, Source: model.VoiceReference.Source}}
	if runtime := model.Runtime; runtime != nil {
		out.CompositionControls = &modelspb.CompositionControls{MinDurationSeconds: runtime.MinDuration, MaxDurationSeconds: runtime.MaxDuration,
			DefaultSteps: int32(runtime.DefaultSteps), MinSteps: int32(runtime.MinSteps), MaxSteps: int32(runtime.MaxSteps),
			DefaultGuidance: runtime.DefaultGuidance, MinGuidance: runtime.MinGuidance, MaxGuidance: runtime.MaxGuidance, DurationControl: "upper_bound"}
	}
	return out
}

func write(w http.ResponseWriter, status int, value proto.Message) {
	httpx.WriteProto(w, status, value)
}
