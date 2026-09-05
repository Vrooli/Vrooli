package readiness

import (
	"encoding/json"
	"net/http"
)

func PrepareHandler(preparer *Preparer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if preparer == nil {
			http.Error(w, "readiness preparation is unavailable", http.StatusServiceUnavailable)
			return
		}
		var request PrepareRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid readiness preparation request", http.StatusBadRequest)
			return
		}
		decision, err := preparer.Prepare(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(decision)
	})
}
