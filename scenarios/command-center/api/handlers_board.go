package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type BoardShape struct {
	SchemaVersion string           `json:"schemaVersion"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	Rooms         []Room           `json:"rooms"`
	Denominator   map[string]any   `json:"denominator"`
	Sources       []map[string]any `json:"sources"`
}
type RoomReadings struct {
	Room     Room                      `json:"room"`
	Readings []MetricEntry             `json:"readings"`
	Sources  map[string]sourceMetadata `json:"sources"`
	// Constellations is set only for a panorama room: every other room with
	// all of its readings, so the whole board's coverage is drawn by the same
	// resolver that draws each room.
	Constellations []Constellation `json:"constellations,omitempty"`
}

// Constellation is one room as the panorama sees it.
type Constellation struct {
	Room     Room          `json:"room"`
	Readings []MetricEntry `json:"readings"`
}
type FocusEntry struct {
	Kind       string `json:"kind"`
	Owner      string `json:"owner"`
	Reason     string `json:"reason"`
	MetricID   string `json:"metricId,omitempty"`
	RankReason string `json:"rankReason"`
}
type FocusSurface struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Entries     []FocusEntry `json:"entries"`
}
type OpenLoop struct {
	GeneratedAt  time.Time        `json:"generatedAt"`
	Missing      []MetricEntry    `json:"missing"`
	Unregistered []MetricEntry    `json:"unregistered"`
	Self         []map[string]any `json:"self"`
}

type BoardSettings struct {
	CycleSeconds float64         `json:"cycleSeconds"`
	Transition   string          `json:"transition"`
	Rooms        []BoardRoomMode `json:"rooms"`
}

type BoardRoomMode struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

func defaultBoardSettings() BoardSettings {
	return BoardSettings{CycleSeconds: 60, Transition: "crossfade", Rooms: []BoardRoomMode{}}
}

func boardSettingsPath() string {
	return filepath.Join(catalogDir(), "board.json")
}

func readBoardSettings() (BoardSettings, error) {
	settings := defaultBoardSettings()
	raw, err := os.ReadFile(boardSettingsPath())
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings, err
	}
	if settings.CycleSeconds < 5 {
		settings.CycleSeconds = 60
	}
	if strings.TrimSpace(settings.Transition) == "" {
		settings.Transition = "crossfade"
	}
	return settings, nil
}

func writeBoardSettings(settings BoardSettings) error {
	if settings.CycleSeconds < 5 || settings.CycleSeconds > 3600 {
		return fmt.Errorf("cycleSeconds must be between 5 and 3600")
	}
	if strings.TrimSpace(settings.Transition) == "" {
		return fmt.Errorf("transition is required")
	}
	seen := map[string]bool{}
	for _, room := range settings.Rooms {
		if room.ID == "" || filepath.Base(room.ID) != room.ID || strings.Contains(room.ID, "..") || seen[room.ID] {
			return fmt.Errorf("invalid or duplicate room id %q", room.ID)
		}
		seen[room.ID] = true
	}
	dir := catalogDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".board-settings-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, boardSettingsPath())
}

func (s *Server) registerBoardRoutes() {
	s.router.HandleFunc("/api/v1/board", s.handleBoard).Methods("GET")
	s.router.HandleFunc("/api/v1/board-settings", s.handleBoardSettings).Methods("GET", "PUT")
	s.router.HandleFunc("/api/v1/rooms/{id}", s.handleRoom).Methods("GET")
	s.router.HandleFunc("/api/v1/focus", s.handleFocus).Methods("GET")
	s.router.HandleFunc("/api/v1/open-loop", s.handleOpenLoop).Methods("GET")
	s.router.HandleFunc("/api/v1/capabilities/describe", s.handleDescribe).Methods("GET")
}

func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	rooms := append([]Room(nil), s.registry.Rooms...)
	if len(rooms) == 0 {
		for id := range s.registry.Dashboards {
			rooms = append(rooms, Room{ID: id, Title: id})
		}
	}
	if settings, err := readBoardSettings(); err == nil && len(settings.Rooms) > 0 {
		byID := make(map[string]Room, len(rooms))
		for _, room := range rooms {
			byID[room.ID] = room
		}
		ordered := make([]Room, 0, len(rooms))
		for _, mode := range settings.Rooms {
			if mode.Enabled {
				if room, ok := byID[mode.ID]; ok {
					ordered = append(ordered, room)
					delete(byID, mode.ID)
				}
			}
		}
		for _, room := range rooms {
			if _, ok := byID[room.ID]; ok {
				ordered = append(ordered, room)
			}
		}
		rooms = ordered
	}
	sources := s.integrationSources(r.Context())
	confidence, rationale := "partial", "The denominator is derived from the checked-in outcome registry and is partial until the objective transmitter is readable."
	if s.objectivesAvailable(r.Context()) {
		confidence, rationale = "declared", "The denominator is joined to the objective set supplied by Prompt Manager and the checked-in outcome registry."
	}
	writeJSON(w, http.StatusOK, BoardShape{SchemaVersion: first(s.registry.SchemaVersion, s.registry.Version), GeneratedAt: time.Now().UTC(), Rooms: rooms, Denominator: map[string]any{"outcomeCategories": len(rooms), "confidence": confidence, "rationale": rationale}, Sources: sources})
}

func (s *Server) handleBoardSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		settings, err := readBoardSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "board_settings_read_failed", err.Error(), nil)
			return
		}
		if len(settings.Rooms) == 0 {
			for _, room := range s.registry.Rooms {
				settings.Rooms = append(settings.Rooms, BoardRoomMode{ID: room.ID, Enabled: true})
			}
		}
		writeJSON(w, http.StatusOK, settings)
		return
	}
	var settings BoardSettings
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	if err := writeBoardSettings(settings); err != nil {
		writeError(w, http.StatusBadRequest, "board_settings_write_failed", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	entries := s.registry.Dashboard(id)
	if entries == nil {
		writeError(w, 404, "room_not_found", "Unknown room: "+id, nil)
		return
	}
	hide := r.URL.Query().Get("samples") == "hide"
	readings, sources := s.readings(r.Context(), entries)
	if hide {
		stripSamples(readings)
	}
	room := roomByID(s.registry, id)
	resp := RoomReadings{Room: room, Readings: readings, Sources: sources}
	if room.Category == "panorama" {
		resp.Constellations = s.constellations(r.Context(), hide)
	}
	writeJSON(w, 200, resp)
}

// constellations reads every room that is not itself a panorama. The room set
// is the registry's, so a new room appears in the panorama with no code change.
func (s *Server) constellations(ctx context.Context, hide bool) []Constellation {
	out := []Constellation{}
	for _, room := range s.registry.Rooms {
		if room.Category == "panorama" {
			continue
		}
		readings, _ := s.readings(ctx, s.registry.Dashboard(room.ID))
		if hide {
			stripSamples(readings)
		}
		out = append(out, Constellation{Room: room, Readings: readings})
	}
	return out
}

func stripSamples(readings []MetricEntry) {
	for i := range readings {
		readings[i].Sample = nil
	}
}

func (s *Server) handleFocus(w http.ResponseWriter, r *http.Request) {
	entries := []FocusEntry{}
	seen := map[string]bool{}
	entries = append(entries, predictionFindings(s.registry)...)
	for _, room := range s.registry.Rooms {
		readings, _ := s.readings(r.Context(), s.registry.Dashboard(room.ID))
		for _, m := range readings {
			// Only a NOW cell has a sensor that can fail to answer; an
			// IN-REACH or MISSING cell is a coverage finding, ranked below.
			if m.Coverage != CoverageNow || (m.Trust != TrustUnavailable && m.Trust != TrustUntrusted) {
				continue
			}
			owner := m.Source.Team
			if owner == "" {
				owner = "unknown-source"
			}
			key := "source-unavailable:" + owner
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, FocusEntry{Kind: "source-unavailable", Owner: owner, MetricID: m.ID, Reason: "Source returned no trustworthy reading; investigate the source before expanding coverage.", RankReason: "Sensor-channel integrity outranks coverage breadth."})
		}
	}
	for _, m := range s.registry.Metrics {
		owner := ""
		if m.Owner != nil {
			owner = *m.Owner
		}
		if m.Coverage == CoverageMissing {
			kind := "no-pipeline"
			if m.Source.Team == "marketing-crew" || m.Source.Team == "scenario-qa" {
				kind = "no-instrument"
			}
			key := kind + ":" + owner
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, FocusEntry{Kind: kind, Owner: owner, MetricID: m.ID, Reason: m.WhatIsNeededString(), RankReason: "Coverage gap follows sensor integrity findings."})
		}
	}
	if len(entries) == 0 {
		for _, room := range s.registry.Rooms {
			if len(s.registry.Dashboard(room.ID)) == 0 {
				entries = append(entries, FocusEntry{Kind: "unregistered-outcome", Owner: "director-swarm", Reason: "Room has no registered readings.", RankReason: "Unregistered outcomes are ranked after source integrity."})
			}
		}
	}
	writeJSON(w, 200, FocusSurface{GeneratedAt: time.Now().UTC(), Entries: entries})
}

func predictionFindings(reg *Registry) []FocusEntry {
	rows, err := loadPredictions()
	if err != nil {
		return nil
	}
	known := map[string]bool{}
	for _, m := range reg.Metrics {
		known[m.ID] = true
	}
	out := []FocusEntry{}
	for _, p := range rows {
		if p.Verdict == EmpiricalUnmeasurable || (!known[p.MetricID] && !p.Horizon.After(time.Now())) {
			out = append(out, FocusEntry{Kind: "unregistered-outcome", Owner: "director-swarm", MetricID: p.MetricID, Reason: first(p.Reason, "Prediction horizon matured without a registered sensor."), RankReason: "Unmeasurable predictions are routed to the open-loop surface."})
		}
	}
	return out
}

func (s *Server) handleOpenLoop(w http.ResponseWriter, r *http.Request) {
	missing := []MetricEntry{}
	unreg := []MetricEntry{}
	for _, m := range s.registry.Metrics {
		if m.Coverage == CoverageMissing {
			m.GapOpenDays = daysOpen(m.FirstObservedMissing)
			missing = append(missing, m)
		}
	}
	self := []map[string]any{{"id": "command-center-instrument", "reason": "The board does not yet read its own render telemetry as an outcome.", "firstObservedMissing": "2026-09-01", "gapOpenDays": daysOpenString("2026-09-01")}}
	for _, room := range s.registry.Rooms {
		if len(s.registry.Dashboard(room.ID)) == 0 {
			date := "2026-09-01"
			age := daysOpenString(date)
			unreg = append(unreg, MetricEntry{ID: room.ID, Label: room.Title, Coverage: CoverageUnregistered, FirstObservedMissing: &date, GapOpenDays: &age, Empirical: EmpiricalNone, Trust: TrustUnavailable})
		}
	}
	writeJSON(w, 200, OpenLoop{GeneratedAt: time.Now().UTC(), Missing: missing, Unregistered: unreg, Self: self})
}

func (s *Server) handleDescribe(w http.ResponseWriter, r *http.Request) {
	b := s.registry
	writeJSON(w, 200, map[string]any{"name": "command-center", "schemaVersion": first(b.SchemaVersion, b.Version), "rooms": b.Rooms, "metrics": b.Metrics, "sources": s.integrationSources(r.Context())})
}

// integrationSources projects the shared operational registry directly. The
// metric registry remains the authority for authored coverage and binding
// semantics, but it must not manufacture lifecycle state or readable flags.
func (s *Server) integrationSources(ctx context.Context) []map[string]any {
	teams := s.teamDeclarations(ctx)
	states := s.integrationSnapshot(ctx, false).States
	teamByIntegration := map[string]string{}
	bindingByIntegration := map[string]string{}
	for _, metric := range s.registry.Metrics {
		integrationID := metric.Source.IntegrationID
		if integrationID == "" {
			integrationID = string(sourceFromBinding(metric.Source.Binding))
		}
		if integrationID == "" || integrationID == "none" {
			continue
		}
		if teamByIntegration[integrationID] == "" {
			teamByIntegration[integrationID] = metric.Source.Team
		}
		if bindingByIntegration[integrationID] == "" {
			bindingByIntegration[integrationID] = metric.Source.Binding
		}
	}
	out := []map[string]any{}
	for _, state := range states {
		integrationID := state.ID
		name := bindingByIntegration[integrationID]
		if name == "" {
			name = state.DependencySlug
		}
		team := teamByIntegration[integrationID]
		decl := teams[team]
		row := map[string]any{
			"name":                name,
			"integrationId":       integrationID,
			"team":                team,
			"instrumentStatus":    first(decl["status"], "partial"),
			"instrumentArchetype": first(decl["archetype"], "production-ledger"),
			"readable":            state.Status == capreg.StatusAvailable,
			"reason":              state.Message,
			"state":               state,
		}
		out = append(out, row)
	}
	return out
}

func roomByID(r *Registry, id string) Room {
	for _, v := range r.Rooms {
		if v.ID == id {
			return v
		}
	}
	return Room{ID: id, Title: id}
}

func (m MetricEntry) WhatIsNeededString() string {
	if m.WhatIsNeeded != nil {
		return *m.WhatIsNeeded
	}
	return "A conforming source pipeline is required."
}

func daysOpen(p *string) *int {
	if p == nil {
		return nil
	}
	n := daysOpenString(*p)
	return &n
}

func daysOpenString(p string) int {
	t, e := time.Parse("2006-01-02", p)
	if e != nil {
		return 0
	}
	d := int(time.Since(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// first returns the first non-empty string in order of preference.
func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (s *Server) teamDeclarations(ctx context.Context) map[string]map[string]string {
	if s.instrumentProvider == nil {
		return map[string]map[string]string{}
	}
	return s.instrumentProvider.Declarations(ctx)
}

func (s *Server) objectivesAvailable(ctx context.Context) bool {
	return s.objectiveProvider != nil && s.objectiveProvider.ObjectivesAvailable(ctx)
}
