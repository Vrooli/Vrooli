package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"command-center/upstream"
	"github.com/gorilla/mux"
)

var catalogNames = map[string]bool{"connectors": true, "signals": true, "rooms": true, "themes": true, "compositions": true, "readouts": true}

type connectorCatalogEntry struct {
	ID        string   `json:"id"`
	Transport string   `json:"transport"`
	Paths     []string `json:"paths"`
}

func connectorDescriptor(id string, resolve func() string, features map[string]string, auth func(*http.Request)) upstream.Descriptor {
	entry := connectorCatalogEntry{ID: id, Transport: "http-json"}
	if raw, err := os.ReadFile(filepath.Join(catalogDir(), "connectors", id+".json")); err == nil {
		_ = json.Unmarshal(raw, &entry)
	}
	transport := upstream.TransportHTTPJSON
	switch entry.Transport {
	case "connect":
		transport = upstream.TransportConnect
	case "graphql":
		transport = upstream.TransportGraphQL
	}
	return upstream.Descriptor{ID: id, Transport: transport, ResolveBase: resolve, Paths: entry.Paths, Features: features, Auth: auth}
}

func catalogDir() string {
	if configured := strings.TrimSpace(os.Getenv("COMMAND_CENTER_CONFIG_DIR")); configured != "" {
		return configured
	}
	return filepath.Join("..", "config")
}

func (s *Server) registerCatalogRoutes() {
	s.router.HandleFunc("/api/v1/catalogs", s.handleCatalogs).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/catalogs/{catalog}", s.handleCatalog).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/catalogs/{catalog}/{id}", s.handleCatalogEntry).Methods(http.MethodPut)
}

func (s *Server) handleCatalogs(w http.ResponseWriter, _ *http.Request) {
	result := map[string][]map[string]any{}
	for name := range catalogNames {
		result[name] = readCatalog(name)
	}
	writeJSON(w, http.StatusOK, result)
}

func readCatalog(name string) []map[string]any {
	entries := []map[string]any{}
	files, err := filepath.Glob(filepath.Join(catalogDir(), name, "*.json"))
	if err != nil {
		return entries
	}
	for _, file := range files {
		raw, readErr := os.ReadFile(file)
		if readErr != nil {
			continue
		}
		var value map[string]any
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		// A catalog entry is keyed by a string id. Generated sidecars (e.g. the
		// *.slots.json manifests beside a composition) carry no id and are not
		// entries; skip them so the catalog holds only addressable definitions.
		if id, ok := value["id"].(string); !ok || strings.TrimSpace(id) == "" {
			continue
		}
		entries = append(entries, value)
	}
	return entries
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["catalog"]
	if !catalogNames[name] {
		writeError(w, http.StatusNotFound, "catalog_not_found", "unknown catalog", nil)
		return
	}
	writeJSON(w, http.StatusOK, readCatalog(name))
}

func (s *Server) handleCatalogEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name, id := vars["catalog"], vars["id"]
	if !catalogNames[name] || filepath.Base(id) != id || strings.Contains(id, "..") {
		writeError(w, http.StatusBadRequest, "invalid_catalog_entry", "invalid catalog entry", nil)
		return
	}
	var value map[string]any
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	if value["id"] != id {
		writeError(w, http.StatusBadRequest, "id_mismatch", fmt.Sprintf("body id must be %q", id), nil)
		return
	}
	dir := filepath.Join(catalogDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_write_failed", err.Error(), nil)
		return
	}
	tmp, err := os.CreateTemp(dir, ".catalog-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_write_failed", err.Error(), nil)
		return
	}
	defer os.Remove(tmp.Name())
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		tmp.Close()
		writeError(w, http.StatusInternalServerError, "catalog_write_failed", err.Error(), nil)
		return
	}
	if err := tmp.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_write_failed", err.Error(), nil)
		return
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, id+".json")); err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_write_failed", err.Error(), nil)
		return
	}
	if name == "rooms" {
		if err := s.reloadRoomCatalog(); err != nil {
			writeError(w, http.StatusBadRequest, "catalog_invalid", err.Error(), nil)
			return
		}
	}
	if name == "signals" {
		if err := LoadSplitSignalCatalog(s.registry); err != nil {
			writeError(w, http.StatusBadRequest, "catalog_invalid", err.Error(), nil)
			return
		}
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) reloadRoomCatalog() error {
	rooms := make([]Room, 0)
	for _, entry := range readCatalog("rooms") {
		encoded, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		var room Room
		if err := json.Unmarshal(encoded, &room); err != nil {
			return err
		}
		rooms = append(rooms, room)
	}
	if len(rooms) > 0 {
		s.registry.Rooms = rooms
	}
	return nil
}

// LoadSplitRoomCatalog is the startup seam: the registry remains the
// compatibility envelope for readings, while editable room presets own the
// instance composition/theme/bind topology.
func LoadSplitRoomCatalog(reg *Registry) error {
	rooms := make([]Room, 0)
	for _, entry := range readCatalog("rooms") {
		encoded, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		var room Room
		if err := json.Unmarshal(encoded, &room); err != nil {
			return err
		}
		rooms = append(rooms, room)
	}
	if len(rooms) == 0 {
		return fmt.Errorf("no split room catalog entries found")
	}
	reg.Rooms = rooms
	return nil
}

// LoadSplitSignalCatalog overlays editable signal definitions onto the
// runtime registry while retaining producer observations and trend policy.
func LoadSplitSignalCatalog(reg *Registry) error {
	byID := map[string]MetricEntry{}
	for _, entry := range readCatalog("signals") {
		raw, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		var signal MetricEntry
		if err := json.Unmarshal(raw, &signal); err != nil {
			return err
		}
		byID[signal.ID] = signal
	}
	if len(byID) == 0 {
		return fmt.Errorf("no split signal catalog entries found")
	}
	for i := range reg.Metrics {
		if configured, ok := byID[reg.Metrics[i].ID]; ok {
			current := &reg.Metrics[i]
			current.Label, current.Description, current.Unit, current.Format, current.Kind = configured.Label, configured.Description, configured.Unit, configured.Format, configured.Kind
			current.Shape, current.Columns, current.Source, current.Coverage, current.Trust, current.Empirical = configured.Shape, configured.Columns, configured.Source, configured.Coverage, configured.Trust, configured.Empirical
			current.Value, current.Rows, current.Sample, current.Ladder = configured.Value, configured.Rows, configured.Sample, configured.Ladder
		}
	}
	return validateSignalShapes(reg)
}
