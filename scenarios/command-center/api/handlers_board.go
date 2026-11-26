package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/storage"
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

type invalidSavedBoardSettingsError struct{ cause error }

func (e *invalidSavedBoardSettingsError) Error() string { return e.cause.Error() }
func (e *invalidSavedBoardSettingsError) Unwrap() error { return e.cause }

func isInvalidSavedBoardSettings(err error) bool {
	var invalid *invalidSavedBoardSettingsError
	return errors.As(err, &invalid)
}

func boardSettingsPath() (string, error) {
	// An explicit config root is retained for isolated tests and controlled
	// deployments. Otherwise, persist operator settings in the lifecycle's
	// per-instance config namespace rather than beside source catalogs.
	if root := strings.TrimSpace(os.Getenv("COMMAND_CENTER_CONFIG_DIR")); root != "" {
		return filepath.Join(root, "board.json"), nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return "", fmt.Errorf("resolve board settings storage: %w", err)
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioName}, storage.ClassConfig, "board.json")
	if err != nil {
		return "", fmt.Errorf("resolve board settings path: %w", err)
	}
	return path, nil
}

func readBoardSettings() (BoardSettings, error) {
	settings := defaultBoardSettings()
	if err := ensureConfigRoot(); err != nil {
		return settings, err
	}
	path, err := boardSettingsPath()
	if err != nil {
		return settings, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && strings.TrimSpace(os.Getenv("COMMAND_CENTER_CONFIG_DIR")) == "" {
		namespace, nsErr := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: scenarioName})
		if nsErr != nil {
			return settings, nsErr
		}
		// The old source-tree file remains a live-instance migration fallback.
		// A shadow starts from safe defaults and never imports a live user's
		// saved dashboard selection.
		if namespace.IsLive() {
			raw, err = os.ReadFile(filepath.Join(catalogDir(), "board.json"))
		}
	}
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings, &invalidSavedBoardSettingsError{cause: err}
	}
	if settings.CycleSeconds < 5 {
		settings.CycleSeconds = 60
	}
	if strings.TrimSpace(settings.Transition) == "" {
		settings.Transition = "crossfade"
	}
	if err := validateBoardSettings(settings); err != nil {
		return BoardSettings{}, &invalidSavedBoardSettingsError{cause: fmt.Errorf("invalid saved board settings: %w", err)}
	}
	return settings, nil
}

func validateBoardSettings(settings BoardSettings) error {
	if settings.CycleSeconds < 5 || settings.CycleSeconds > 3600 {
		return fmt.Errorf("cycleSeconds must be between 5 and 3600")
	}
	if settings.Transition != "crossfade" && settings.Transition != "cut" {
		return fmt.Errorf("transition must be crossfade or cut")
	}
	seen := map[string]bool{}
	for _, room := range settings.Rooms {
		if room.ID == "" || filepath.Base(room.ID) != room.ID || strings.Contains(room.ID, "..") || seen[room.ID] {
			return fmt.Errorf("invalid or duplicate room id %q", room.ID)
		}
		seen[room.ID] = true
	}
	return nil
}

func writeBoardSettings(settings BoardSettings) error {
	if err := validateBoardSettings(settings); err != nil {
		return err
	}
	if err := ensureConfigRoot(); err != nil {
		return err
	}
	path, err := boardSettingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := storage.WriteFileAtomic(path, data, storage.DefaultFilePerm); err != nil {
		return err
	}
	return nil
}

func (s *Server) registerBoardRoutes() {
	s.router.HandleFunc("/api/v1/board", s.handleBoard).Methods("GET")
	s.router.HandleFunc("/api/v1/board-settings", s.handleBoardSettings).Methods("GET", "PUT")
	s.router.HandleFunc("/api/v1/rooms/{id}", requireConfigRoot(s.handleRoom)).Methods("GET")
	s.router.HandleFunc("/api/v1/focus", requireConfigRoot(s.handleFocus)).Methods("GET")
	s.router.HandleFunc("/api/v1/open-loop", requireConfigRoot(s.handleOpenLoop)).Methods("GET")
	s.router.HandleFunc("/api/v1/capabilities/describe", requireConfigRoot(s.handleDescribe)).Methods("GET")
}

// orderBoardRooms applies explicit enablement and ordering while retaining
// defaults for rooms omitted from an older partial settings file.
func orderBoardRooms(rooms []Room, settings []BoardRoomMode) []Room {
	if len(settings) == 0 {
		return rooms
	}
	byID := make(map[string]Room, len(rooms))
	for _, room := range rooms {
		byID[room.ID] = room
	}
	ordered := make([]Room, 0, len(rooms))
	configured := make(map[string]bool, len(settings))
	for _, mode := range settings {
		configured[mode.ID] = true
		if room, ok := byID[mode.ID]; ok && mode.Enabled {
			ordered = append(ordered, room)
		}
	}
	for _, room := range rooms {
		// Older partial settings may omit rooms; retain those defaults while
		// preserving an explicit disabled choice.
		if !configured[room.ID] {
			ordered = append(ordered, room)
		}
	}
	return ordered
}

func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	rooms := append([]Room(nil), s.registry.Rooms...)
	if len(rooms) == 0 {
		for id := range s.registry.Dashboards {
			rooms = append(rooms, Room{ID: id, Title: id})
		}
	}
	settings, err := readBoardSettings()
	if err != nil {
		writeConfigReadError(w, err)
		return
	}
	rooms = orderBoardRooms(rooms, settings.Rooms)
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
			if isConfigRootUnavailable(err) {
				writeConfigReadError(w, err)
				return
			}
			if isInvalidSavedBoardSettings(err) {
				writeError(w, http.StatusUnprocessableEntity, "board_settings_invalid", "saved board settings are invalid and need repair", nil)
				return
			}
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
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON value", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	if err := writeBoardSettings(settings); err != nil {
		if isConfigRootUnavailable(err) {
			writeConfigReadError(w, err)
			return
		}
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
	setpoint, err := s.readSetpoint()
	if err != nil {
		writeSetpointReadError(w, err)
		return
	}
	hide := r.URL.Query().Get("samples") == "hide"
	readings, sources := s.readings(r.Context(), entries)
	applySetpoint(readings, setpoint)
	if hide {
		stripSamples(readings)
	}
	room := roomByID(s.registry, id)
	resp := RoomReadings{Room: room, Readings: readings, Sources: sources}
	if room.Category == "panorama" {
		resp.Constellations = s.constellations(r.Context(), hide, setpoint)
	}
	writeJSON(w, 200, resp)
}

// constellations reads every room that is not itself a panorama. The room set
// is the registry's, so a new room appears in the panorama with no code change.
func (s *Server) constellations(ctx context.Context, hide bool, setpoint *Setpoint) []Constellation {
	out := []Constellation{}
	for _, room := range s.registry.Rooms {
		if room.Category == "panorama" {
			continue
		}
		readings, _ := s.readings(ctx, s.registry.Dashboard(room.ID))
		applySetpoint(readings, setpoint)
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
			// Only NOW cells can report a producer outage. Any non-valid trust
			// state is degraded, including a retained last-good CACHED value.
			if m.Coverage != CoverageNow || m.Trust == TrustValid {
				continue
			}
			owner := m.Source.Team
			if owner == "" {
				owner = "unknown-source"
			}
			// An outcome owner may publish through several independent source
			// surfaces, including multiple endpoints from one integration. Keep
			// one finding per endpoint so one failure cannot hide another.
			producer := first(m.Source.IntegrationID, string(m.UpstreamSource), m.Source.Binding)
			if producer == "" {
				producer = owner
			}
			source := m.UpstreamSource
			if source == "" {
				source = sourceFromBinding(m.Source.Binding)
			}
			endpoint := first(m.Source.Read, defaultPathFor(source))
			key := "source-unavailable:" + owner + ":" + producer + ":" + endpoint
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, FocusEntry{Kind: "source-unavailable", Owner: owner, MetricID: m.ID, Reason: "Source has no current trusted reading; a cached value is last-good only until the source recovers.", RankReason: "Sensor-channel integrity outranks coverage breadth."})
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
