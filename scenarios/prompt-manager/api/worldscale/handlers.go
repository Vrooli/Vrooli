// Package worldscale provides HTTP handlers for world object scale configuration.
// DOC: docs/reference/api-endpoints.md#world-scale
// DOC: docs/internal/SEAMS.md#world-scale-config
package worldscale

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"prompt-manager/store"
)

// Config holds scale multipliers for each object category.
type Config struct {
	Agent      float64 `json:"agent"`
	Furniture  float64 `json:"furniture"`
	Decoration float64 `json:"decoration"`
	Overlay    float64 `json:"overlay"`
}

var defaultConfig = Config{
	Agent:      1.0,
	Furniture:  1.0,
	Decoration: 1.0,
	Overlay:    1.0,
}

const configFile = "world-scale.json"

func configPath(configDir string) string {
	return filepath.Join(configDir, configFile)
}

// HandleGet returns an http.HandlerFunc that reads world-scale.json.
// Returns default values if the file does not exist.
func HandleGet(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := configPath(configDir)

		var cfg Config
		if store.FileExists(path) {
			loaded, err := store.LoadJSON[Config](path)
			if err != nil {
				http.Error(w, "reading world-scale config: "+err.Error(), http.StatusInternalServerError)
				return
			}
			cfg = *loaded
		} else {
			cfg = defaultConfig
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}

// HandlePut returns an http.HandlerFunc that validates and writes world-scale.json.
func HandlePut(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate ranges
		for _, entry := range []struct {
			name string
			val  float64
		}{
			{"agent", cfg.Agent},
			{"furniture", cfg.Furniture},
			{"decoration", cfg.Decoration},
			{"overlay", cfg.Overlay},
		} {
			if entry.val < 0.1 || entry.val > 3.0 {
				http.Error(w, entry.name+" must be between 0.1 and 3.0", http.StatusBadRequest)
				return
			}
		}

		path := configPath(configDir)
		if err := store.SaveJSON(path, &cfg); err != nil {
			http.Error(w, "saving world-scale config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}
