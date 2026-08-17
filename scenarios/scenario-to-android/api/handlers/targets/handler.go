package targets

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	"scenario-to-android/internal/targets"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	prober := targets.Prober{Devices: targets.NewDeviceControlInventory()}
	var bridgeSources []deliveryramp.BridgeSource
	if bridge := validationmatrix.NewClientFromEnv(); bridge != nil {
		bridgeSources = append(bridgeSources, bridge)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	inventory, err := deliveryramp.Discover(ctx, prober, bridgeSources...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(inventory)
}
