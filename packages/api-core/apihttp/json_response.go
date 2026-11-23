package apihttp

import (
	"encoding/json"
	"net/http"
)

// WriteJSONResponse writes a JSON response with the standard content type and
// status handling. The value remains owned by the caller so scenario-specific
// envelope fields stay at their API boundary.
func WriteJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// BearerTokenMiddleware requires an exact Authorization bearer value except
// on the supplied exact request paths. The caller controls the unauthorized
// response so it can preserve its API envelope.
func BearerTokenMiddleware(token string, unauthorized func(http.ResponseWriter, *http.Request), exemptPaths ...string) func(http.Handler) http.Handler {
	exempt := make(map[string]struct{}, len(exemptPaths))
	for _, path := range exemptPaths {
		exempt[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := exempt[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("Authorization") != "Bearer "+token {
				unauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
