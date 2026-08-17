package targets

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-ios/internal/module"
	"scenario-to-ios/internal/targets"
)

// Module exposes Apple target probing.
func Module(prober targets.Prober) module.Module {
	return module.Module{Name: "ios-targets", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/targets", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := prober.Probe(req.Context(), deliveryramp.ProbeRequest{})
			write(w, inventory, err)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func write(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "reason": err.Error(), "missing_capability": "iOS target capability", "next_action": "inspect the error and provide the named capability"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
