package distribution

import (
	"encoding/json"
	"net/http"

	"scenario-to-android/internal/distribution"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	result, err := (distribution.Distributor{}).Distribute(r.Context(), deliveryramp.DistributionRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
