package jobs

import (
	"context"
	"encoding/json"
	"net/http"

	internaljobs "image-tools/internal/jobs"
)

type heavyJobSnapshotReader interface {
	HeavyJobSnapshot(context.Context) internaljobs.HeavyJobSnapshot
}

func newHeavyJobSnapshotHandler(manager any) http.Handler {
	reader, ok := manager.(heavyJobSnapshotReader)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !ok {
			http.Error(w, "heavy job telemetry is unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(reader.HeavyJobSnapshot(r.Context())); err != nil {
			// A response write failure cannot be repaired after headers are sent.
			return
		}
	})
}
