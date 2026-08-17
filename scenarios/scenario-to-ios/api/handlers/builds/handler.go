package builds

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-ios/internal/builds"
	"scenario-to-ios/internal/module"
)

type request struct {
	SourceRef  string            `json:"source_ref"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// Module exposes project generation and native build operations.
func Module(builder builds.Builder) module.Module {
	return module.Module{Name: "ios-builds", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/generate", GenerateHandler(builder)).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/ios/build", BuildHandler(builder)).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func GenerateHandler(builder builds.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var input request
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeError(w, err)
			return
		}
		result, err := builder.Generate(req.Context(), deliveryramp.BuildRequest{SourceRef: input.SourceRef, Parameters: input.Parameters})
		write(w, result, err)
	}
}

func BuildHandler(builder builds.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var input request
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeError(w, err)
			return
		}
		result, err := builder.Build(req.Context(), deliveryramp.BuildRequest{SourceRef: input.SourceRef, Parameters: input.Parameters})
		if unavailable, ok := err.(builds.UnavailableError); ok {
			writeJSON(w, http.StatusOK, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "missing_capability": unavailable.MissingCapability, "next_action": unavailable.NextAction})
			return
		}
		write(w, result, err)
	}
}

func write(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusOK, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "reason": err.Error(), "missing_capability": "iOS build capability", "next_action": "inspect the error and provide the named capability"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
