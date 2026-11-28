package releases

import (
	"context"
	"encoding/json"
	"net/http"

	"scenario-to-ios/internal/module"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"

	"github.com/gorilla/mux"
)

type Surface struct {
	Probe        func(context.Context) (deliveryramp.Inventory, error)
	ChapterCount int
}

func Module(matrixHandlers []*validationmatrix.Handler, surface Surface) module.Module {
	return module.Module{Name: "ios-releases", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/matrix", func(w http.ResponseWriter, req *http.Request) {
			inventory, err := surface.Probe(req.Context())
			if err != nil {
				writeJSON(w, http.StatusOK, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "reason": err.Error()})
				return
			}
			available := 0
			for _, target := range inventory.Targets {
				if target.Available {
					available++
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{"disposition": deliveryramp.DispositionUnavailable, "target_count": len(inventory.Targets), "available_targets": available, "chapter_count": surface.ChapterCount, "missing_capability": "iOS validation target", "next_action": "register a macOS bridge, then create a matrix run for the selected scenario"})
		}).Methods(http.MethodGet)
		for _, handler := range matrixHandlers {
			if handler != nil {
				handler.RegisterRoutes(r)
			}
		}
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
