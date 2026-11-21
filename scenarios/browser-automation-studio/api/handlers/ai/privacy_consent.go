package ai

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// PrivacyConsentHandler proxies consent lifecycle calls to AI Gateway while
// preserving the verified browser session token for Gateway verification.
func PrivacyConsentHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base, err := discovery.ResolveScenarioURLDefault(r.Context(), "ai-gateway")
		if err != nil {
			http.Error(w, "AI consent authority is unavailable", http.StatusServiceUnavailable)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), r.Method, strings.TrimRight(base, "/")+r.URL.RequestURI(), r.Body)
		if err != nil {
			http.Error(w, "AI consent request could not be prepared", http.StatusBadGateway)
			return
		}
		for _, name := range []string{"Authorization", "Cookie", "Content-Type"} {
			for _, value := range r.Header.Values(name) {
				request.Header.Add(name, value)
			}
		}
		response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
		if err != nil {
			http.Error(w, "AI consent authority is unavailable", http.StatusServiceUnavailable)
			return
		}
		defer response.Body.Close()
		for _, value := range response.Header.Values("Content-Type") {
			w.Header().Add("Content-Type", value)
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	})
}
