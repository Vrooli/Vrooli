package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

func (s *Server) registerBoardRoutes() {
	s.router.HandleFunc("/api/v1/board", s.handleBoard).Methods("GET")
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
	sources := s.sourceDeclarations()
	writeJSON(w, http.StatusOK, BoardShape{SchemaVersion: first(s.registry.SchemaVersion, s.registry.Version), GeneratedAt: time.Now().UTC(), Rooms: rooms, Denominator: map[string]any{"outcomeCategories": len(rooms), "confidence": "partial", "rationale": "The denominator is derived from the checked-in outcome registry and is partial until the objective transmitter is readable."}, Sources: sources})
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	entries := s.registry.Dashboard(id)
	if entries == nil {
		writeError(w, 404, "room_not_found", "Unknown room: "+id, nil)
		return
	}
	readings, sources := s.readings(r.Context(), entries)
	if r.URL.Query().Get("samples") == "hide" {
		for i := range readings {
			readings[i].Sample = nil
		}
	}
	writeJSON(w, 200, RoomReadings{Room: roomByID(s.registry, id), Readings: readings, Sources: sources})
}

func (s *Server) handleFocus(w http.ResponseWriter, r *http.Request) {
	entries := []FocusEntry{}
	seen := map[string]bool{}
	for _, p := range predictionFindings(s.registry) {
		entries = append(entries, p)
	}
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
	writeJSON(w, 200, map[string]any{"name": "command-center", "schemaVersion": first(b.SchemaVersion, b.Version), "rooms": b.Rooms, "metrics": b.Metrics, "sources": s.sourceDeclarations()})
}

func (s *Server) sourceDeclarations() []map[string]any {
	seen := map[string]bool{}
	teams := loadTeamDeclarations()
	out := []map[string]any{}
	for _, m := range s.registry.Metrics {
		name := m.Source.Binding
		if name == "" {
			name = string(m.UpstreamSource)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		decl := teams[m.Source.Team]
		status := first(m.Source.InstrumentStatus, decl["status"])
		status = first(status, "partial")
		archetype := first(m.Source.InstrumentArchetype, decl["archetype"])
		out = append(out, map[string]any{"name": name, "team": m.Source.Team, "instrumentStatus": status, "instrumentArchetype": archetype, "readable": name != "none", "reason": map[bool]string{true: "", false: "No instrument surface declared"}[name != "none"]})
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

func loadTeamDeclarations() map[string]map[string]string {
	out := map[string]map[string]string{}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = "../../"
	}
	matches, _ := filepath.Glob(filepath.Join(root, "scenarios/prompt-manager/store/teams/*/team.json"))
	for _, p := range matches {
		var v struct {
			Instrument struct {
				Status    string `json:"status"`
				Archetype string `json:"archetype"`
			} `json:"instrument"`
		}
		if b, e := os.ReadFile(p); e == nil && json.Unmarshal(b, &v) == nil {
			out[filepath.Base(filepath.Dir(p))] = map[string]string{"status": v.Instrument.Status, "archetype": v.Instrument.Archetype}
		}
	}
	return out
}

var _ = strings.TrimSpace
