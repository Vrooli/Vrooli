package delivery

import (
	"encoding/json"
	"net/http"

	component "github.com/vrooli/vrooli/packages/desktop-component"
	internal "landing-page-business-suite-api/internal/delivery"
)

// Components publishes signed code metadata through the app's existing LPBS
// artifact store. Only explicitly free component assets use this public route.
func Components(bundleKey func() string, catalog *internal.CatalogService, storage *internal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rows, err := catalog.ComponentAssets(r.Context(), bundleKey())
		if err != nil {
			http.Error(w, "component catalog unavailable", http.StatusServiceUnavailable)
			return
		}
		entries := make([]component.SignedEntry, 0, len(rows))
		id := r.URL.Query().Get("id")
		for _, row := range rows {
			if row.RequiresEntitlement {
				continue
			}
			entry, err := internal.ComponentEntry(row)
			if err != nil {
				http.Error(w, "invalid component catalog entry", http.StatusServiceUnavailable)
				return
			}
			if id == "" {
				entries = append(entries, entry)
				continue
			}
			if entry.Entry.ID != id {
				continue
			}
			if row.ArtifactID == nil {
				http.Error(w, "component artifact missing", http.StatusNotFound)
				return
			}
			artifact, err := storage.GetArtifact(r.Context(), bundleKey(), *row.ArtifactID)
			if err != nil || artifact == nil || artifact.SHA256 != entry.Entry.SHA256 || artifact.SizeBytes != entry.Entry.Size {
				http.Error(w, "component artifact verification failed", http.StatusServiceUnavailable)
				return
			}
			target, err := storage.PresignGetArtifact(r.Context(), bundleKey(), *artifact)
			if err != nil {
				http.Error(w, "component storage unavailable", http.StatusServiceUnavailable)
				return
			}
			http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			return
		}
		if id != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entries)
	}
}
