package distribution

import (
	"encoding/json"
	"net/http"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-ios/internal/distribution"
	"scenario-to-ios/internal/module"

	"github.com/gorilla/mux"
)

func Module(distributor distribution.Distributor) module.Module {
	return module.Module{Name: "ios-distribution", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/distribution", func(w http.ResponseWriter, req *http.Request) {
			result, err := distributor.Distribute(req.Context(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{ImmutableRef: "pending-artifact"}})
			if err != nil {
				writeJSON(w, http.StatusOK, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "reason": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, result)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
