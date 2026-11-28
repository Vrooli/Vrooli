// Package variant owns HTTP transport for variant selection and reads.
package variant

import (
	"net/http"

	domain "landing-page-business-suite-api/internal/experimentation"
)

// VariantResponse is the stable flat JSON shape consumed by the landing and
// administration UIs. Read and write endpoints intentionally share it.
type VariantResponse = Response

type Dependencies struct {
	Get        func(string) (any, error)
	List       func() any
	Slug       func(*http.Request) string
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

// NewReadDependencies composes the read transport around the experimentation
// store. HTTP serialization and logging remain injected at the application
// boundary, while selection and response mapping stay in this domain package.
func NewReadDependencies(store domain.ConfigStoreReader, pathPrefix string, writeJSON func(http.ResponseWriter, any), writeError func(http.ResponseWriter, int, string, string), log func(string, map[string]any)) Dependencies {
	return Dependencies{
		Get: func(slug string) (any, error) {
			snapshot, err := store.GetVariant(slug)
			if err != nil {
				return nil, err
			}
			return response(snapshot), nil
		},
		List: func() any {
			snapshots := store.ListVariants()
			variants := make([]VariantResponse, 0, len(snapshots))
			for _, snapshot := range snapshots {
				variants = append(variants, response(snapshot))
			}
			return variants
		},
		Slug:       func(r *http.Request) string { return r.URL.Path[len(pathPrefix):] },
		WriteJSON:  writeJSON,
		WriteError: writeError,
		Log:        log,
	}
}

func List(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
			return
		}
		deps.WriteJSON(w, map[string]any{"variants": deps.List()})
	}
}

func AdminGet(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			deps.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed.", "")
			return
		}
		slug := deps.Slug(r)
		if slug == "" || slug == "select" {
			deps.WriteError(w, http.StatusBadRequest, "Variant slug is required.", "validation")
			return
		}
		item, err := deps.Get(slug)
		if err != nil {
			deps.Log("variant_fetch_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, http.StatusNotFound, "Variant not found.", "not_found")
			return
		}
		deps.WriteJSON(w, item)
	}
}
