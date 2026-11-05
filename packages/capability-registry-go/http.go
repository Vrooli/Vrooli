package capabilityregistry

import (
	"net/http"
)

// DescribeHandler exposes a registry using the shared capability describe
// response contract. Callers own route registration and method constraints.
func DescribeHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if registry == nil {
			http.Error(w, "capabilities registry is not configured", http.StatusServiceUnavailable)
			return
		}
		data, err := registry.Describe(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}
