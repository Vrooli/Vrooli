// Package worldseats provides HTTP handlers for world seat position configuration.
// DOC: docs/reference/api-endpoints.md#world-seats
package worldseats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"prompt-manager/store"
)

// SeatPosition holds the offset and facing direction for a single seat.
type SeatPosition struct {
	Position [3]float64 `json:"position"`
	Rotation float64    `json:"rotation"`
}

// Config maps furniture type keys to their seat arrays.
type Config map[string][]SeatPosition

// knownTypes lists all valid furniture type keys.
var knownTypes = map[string]bool{
	"chair":        true,
	"bench":        true,
	"stool":        true,
	"armchair":     true,
	"desk":         true,
	"table":        true,
	"picnic-table": true,
	"coffee-table": true,
	"campfire":     true,
}

const (
	maxSeatsPerType = 20
	posMin          = -10.0
	posMax          = 10.0
)

const configFile = "world-seats.json"

func configPath(configDir string) string {
	return filepath.Join(configDir, configFile)
}

func validate(cfg Config) error {
	for key, seats := range cfg {
		if !knownTypes[key] {
			return fmt.Errorf("unknown furniture type: %s", key)
		}
		if len(seats) > maxSeatsPerType {
			return fmt.Errorf("%s: too many seats (%d > %d)", key, len(seats), maxSeatsPerType)
		}
		for i, seat := range seats {
			for axis := 0; axis < 3; axis++ {
				if seat.Position[axis] < posMin || seat.Position[axis] > posMax {
					return fmt.Errorf("%s seat %d: position[%d] out of range [%g, %g]", key, i, axis, posMin, posMax)
				}
			}
		}
	}
	return nil
}

// HandleGet returns an http.HandlerFunc that reads world-seats.json.
func HandleGet(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := configPath(configDir)

		var cfg Config
		if store.FileExists(path) {
			loaded, err := store.LoadJSON[Config](path)
			if err != nil {
				http.Error(w, "reading world-seats config: "+err.Error(), http.StatusInternalServerError)
				return
			}
			cfg = *loaded
		} else {
			cfg = Config{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}

// HandlePut returns an http.HandlerFunc that validates and writes world-seats.json.
func HandlePut(configDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := validate(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		path := configPath(configDir)
		if err := store.SaveJSON(path, &cfg); err != nil {
			http.Error(w, "saving world-seats config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}
