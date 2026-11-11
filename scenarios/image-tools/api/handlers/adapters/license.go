package adapters

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	internaladapters "image-tools/internal/adapters"
)

func licenseTermsHandler(registry *internaladapters.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(mux.Vars(r)["id"])
		adapter, ok := registry.ByID(id)
		if !ok || adapter.LicenseTermsVersion == "" {
			http.NotFound(w, r)
			return
		}
		terms, err := internaladapters.LicenseTerms(adapter.LicenseTermsVersion)
		if err != nil {
			http.Error(w, "license terms are unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(terms)
	}
}
