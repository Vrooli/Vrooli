package distribution

import (
	"encoding/json"
	"net/http"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-android/internal/distribution"
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
