package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Coverage string

const (
	CoverageNow          Coverage = "NOW"
	CoverageInReach      Coverage = "IN-REACH"
	CoverageMissing      Coverage = "MISSING"
	CoverageUnregistered Coverage = "UNREGISTERED"
)

type Trust string

const (
	TrustValid       Trust = "VALID"
	TrustCached      Trust = "CACHED"
	TrustUnavailable Trust = "UNAVAILABLE"
	TrustUntrusted   Trust = "UNTRUSTED"
)

type Empirical string

const (
	EmpiricalNone         Empirical = "NONE"
	EmpiricalPending      Empirical = "PENDING"
	EmpiricalHit          Empirical = "HIT"
	EmpiricalMiss         Empirical = "MISS"
	EmpiricalUnmeasurable Empirical = "UNMEASURABLE"
)

type UpstreamSource string

const (
	SourceSwarm  UpstreamSource = "swarm"
	SourceVrooli UpstreamSource = "vrooli"
	SourceLPBS   UpstreamSource = "lpbs"
	SourceOffer  UpstreamSource = "offer-desk"
	SourceDeploy UpstreamSource = "deployment-manager"
	SourceNone   UpstreamSource = "none"
)

type DataSourceStatus string

const (
	StatusLive    DataSourceStatus = "live"
	StatusPartial DataSourceStatus = "partial"
	StatusGap     DataSourceStatus = "gap"
)

type Sample struct {
	Value  any        `json:"value"`
	Series []any      `json:"series"`
	Rows   []PanelRow `json:"rows,omitempty"`
	Basis  string     `json:"basis"`
}
type SourceBinding struct {
	Team                string `json:"team"`
	Binding             string `json:"binding"`
	Read                string `json:"read"`
	Select              string `json:"select"`
	IntegrationID       string `json:"integrationId,omitempty"`
	FeatureID           string `json:"featureId,omitempty"`
	ContractVersion     string `json:"contractVersion,omitempty"`
	Selector            string `json:"selector,omitempty"`
	ExpectedUnit        string `json:"expectedUnit,omitempty"`
	SourceTimePolicy    string `json:"sourceTimePolicy,omitempty"`
	Origin              string `json:"origin,omitempty"`
	TTLSeconds          int    `json:"ttlSeconds"`
	InstrumentStatus    string `json:"instrumentStatus,omitempty"`
	InstrumentArchetype string `json:"instrumentArchetype,omitempty"`
}
type OriginSpec struct {
	Mode        string `json:"mode"`
	Environment string `json:"environment"`
	Display     string `json:"display"`
}
type Target struct {
	Direction string `json:"direction"`
	Bar       any    `json:"bar"`
	BarRef    string `json:"barRef,omitempty"`
}
type Prediction struct {
	Target                  any    `json:"target"`
	Direction               string `json:"direction"`
	RemainingHorizonSeconds int    `json:"remainingHorizonSeconds"`
}

type MetricEntry struct {
	ID                   string           `json:"id"`
	Label                string           `json:"label"`
	Kind                 string           `json:"kind,omitempty"`
	Description          string           `json:"description,omitempty"`
	Unit                 string           `json:"unit,omitempty"`
	Format               string           `json:"format,omitempty"`
	Tags                 []string         `json:"tags,omitempty"`
	Source               SourceBinding    `json:"source"`
	Coverage             Coverage         `json:"coverage"`
	Trust                Trust            `json:"trust"`
	TrustReason          string           `json:"trustReason,omitempty"`
	Empirical            Empirical        `json:"empirical"`
	Value                any              `json:"value"`
	Rows                 []PanelRow       `json:"rows,omitempty"`
	ObservedAt           *time.Time       `json:"observedAt"`
	TTLSeconds           int              `json:"ttlSeconds"`
	Target               *Target          `json:"target"`
	Owner                *string          `json:"owner"`
	WhatIsNeeded         *string          `json:"whatIsNeeded"`
	FirstObservedMissing *string          `json:"firstObservedMissing"`
	GapOpenDays          *int             `json:"gapOpenDays"`
	Sample               *Sample          `json:"sample"`
	Prediction           *Prediction      `json:"prediction"`
	DataSource           DataSourceStatus `json:"dataSource,omitempty"`
	UpstreamSource       UpstreamSource   `json:"upstreamSource,omitempty"`
	Origin               string           `json:"origin"`
	OriginEnv            string           `json:"origin_env"`
	OriginDisplay        string           `json:"origin_display"`
}
type Room struct {
	ID            string              `json:"id"`
	Title         string              `json:"title"`
	Category      string              `json:"category,omitempty"`
	ObjectiveRefs []string            `json:"objectiveRefs,omitempty"`
	Select        map[string][]string `json:"select,omitempty"`
	Composition   string              `json:"composition,omitempty"`
	Theme         string              `json:"theme,omitempty"`
	MetricIDs     []string            `json:"metricIds,omitempty"`
	Beats         []Beat              `json:"beats,omitempty"`
}

type Beat struct {
	Hero         string  `json:"hero"`
	Composition  string  `json:"composition,omitempty"`
	DwellSeconds float64 `json:"dwellSeconds,omitempty"`
}
type Tombstone struct {
	ID           string `json:"id"`
	Retired      string `json:"retired"`
	Reason       string `json:"reason"`
	SupersededBy string `json:"supersededBy,omitempty"`
}
type Registry struct {
	SchemaRef     string                   `json:"$schema,omitempty"`
	SchemaVersion string                   `json:"schemaVersion,omitempty"`
	Version       string                   `json:"version,omitempty"`
	Rooms         []Room                   `json:"rooms,omitempty"`
	Metrics       []MetricEntry            `json:"metrics,omitempty"`
	Tombstones    []Tombstone              `json:"tombstones,omitempty"`
	Origins       map[string]OriginSpec    `json:"origins,omitempty"`
	Dashboards    map[string][]MetricEntry `json:"dashboards,omitempty"`
}

func validCoverage(v Coverage) bool {
	return v == CoverageNow || v == CoverageInReach || v == CoverageMissing || v == CoverageUnregistered
}

func validEmpirical(v Empirical) bool {
	return v == EmpiricalNone || v == EmpiricalPending || v == EmpiricalHit || v == EmpiricalMiss || v == EmpiricalUnmeasurable
}

func LoadRegistry(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open registry: %w", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var reg Registry
	if err := dec.Decode(&reg); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if reg.SchemaVersion == "" && reg.Version == "" {
		return nil, fmt.Errorf("registry is missing 'schemaVersion'")
	}
	if len(reg.Metrics) == 0 && len(reg.Dashboards) == 0 {
		return nil, fmt.Errorf("registry is missing metrics")
	}
	if len(reg.Metrics) == 0 {
		for id, entries := range reg.Dashboards {
			for i := range entries {
				e := &entries[i]
				e.Coverage = coverageFromLegacy(e.DataSource)
				e.Source = SourceBinding{Binding: string(e.UpstreamSource), Read: defaultPathFor(e.UpstreamSource), TTLSeconds: 60}
				if e.Coverage != CoverageNow {
					e.Sample = &Sample{Value: 0, Series: []any{0}, Basis: "legacy compatibility sample"}
				}
				reg.Metrics = append(reg.Metrics, *e)
			}
			reg.Dashboards[id] = entries
		}
	}
	for i := range reg.Metrics {
		m := &reg.Metrics[i]
		if !validCoverage(m.Coverage) && m.Coverage != "" {
			return nil, fmt.Errorf("metric %s has invalid coverage %q", m.ID, m.Coverage)
		}
		if m.Coverage == "" {
			m.Coverage = CoverageNow
		}
		if m.Empirical == "" {
			m.Empirical = EmpiricalNone
		}
		if m.TTLSeconds == 0 {
			m.TTLSeconds = m.Source.TTLSeconds
		}
		if m.Source.IntegrationID == "" {
			m.Source.IntegrationID = strings.TrimPrefix(m.Source.Binding, "scenario:")
			m.Source.IntegrationID = strings.TrimPrefix(m.Source.IntegrationID, "resource:")
		}
		if m.Source.FeatureID == "" {
			m.Source.FeatureID = first(m.Source.Select, m.ID)
		}
		if m.Source.Selector == "" {
			m.Source.Selector = first(m.Source.Select, m.ID)
		}
		if m.Source.ContractVersion == "" {
			m.Source.ContractVersion = "legacy.v1"
		}
		if m.Source.ExpectedUnit == "" {
			m.Source.ExpectedUnit = m.Unit
		}
		if m.Source.SourceTimePolicy == "" {
			m.Source.SourceTimePolicy = "producer_required"
		}
		if m.Source.Origin != "" {
			if _, ok := reg.Origins[m.Source.Origin]; !ok {
				return nil, fmt.Errorf("metric %s names unknown origin %q", m.ID, m.Source.Origin)
			}
		}
		if err := validateMetricBinding(*m); err != nil {
			return nil, err
		}
		if m.TTLSeconds == 0 {
			m.TTLSeconds = 60
		}
		if m.Coverage != CoverageNow && m.Sample == nil {
			return nil, fmt.Errorf("metric %s missing sample", m.ID)
		}
	}
	if len(reg.Rooms) == 0 {
		for id := range reg.Dashboards {
			reg.Rooms = append(reg.Rooms, Room{ID: id, Title: id})
		}
	}
	return &reg, nil
}

func validateMetricBinding(m MetricEntry) error {
	s := m.Source
	if strings.TrimSpace(s.IntegrationID) == "" {
		return fmt.Errorf("metric %s binding is missing integrationId", m.ID)
	}
	if strings.TrimSpace(s.FeatureID) == "" {
		return fmt.Errorf("metric %s binding is missing featureId", m.ID)
	}
	if strings.TrimSpace(s.ContractVersion) == "" {
		return fmt.Errorf("metric %s binding is missing contractVersion", m.ID)
	}
	if strings.TrimSpace(s.Selector) == "" {
		return fmt.Errorf("metric %s binding is missing selector", m.ID)
	}
	if s.FeatureID != s.Selector {
		return fmt.Errorf("metric %s binding featureId %q differs from selector %q", m.ID, s.FeatureID, s.Selector)
	}
	if s.ExpectedUnit != m.Unit {
		return fmt.Errorf("metric %s binding expectedUnit %q differs from unit %q", m.ID, s.ExpectedUnit, m.Unit)
	}
	if s.SourceTimePolicy != "producer_required" {
		return fmt.Errorf("metric %s has unsupported sourceTimePolicy %q", m.ID, s.SourceTimePolicy)
	}
	return nil
}

func coverageFromLegacy(v DataSourceStatus) Coverage {
	switch v {
	case StatusLive:
		return CoverageNow
	case StatusPartial:
		return CoverageInReach
	default:
		return CoverageMissing
	}
}

func (r *Registry) Dashboard(id string) []MetricEntry {
	if r == nil {
		return nil
	}
	if len(r.Dashboards) > 0 {
		return r.Dashboards[id]
	}
	for _, room := range r.Rooms {
		if room.ID == id {
			var out []MetricEntry
			if len(room.MetricIDs) == 0 {
				return out
			}
			for _, m := range r.Metrics {
				if contains(room.MetricIDs, m.ID) {
					out = append(out, m)
				}
			}
			return out
		}
	}
	return nil
}

func (r *Registry) GapsByDashboard() map[string][]MetricEntry {
	out := map[string][]MetricEntry{}
	if len(r.Rooms) == 0 {
		for id := range r.Dashboards {
			var gaps []MetricEntry
			for _, m := range r.Dashboards[id] {
				if effectiveCoverage(m) != CoverageNow {
					gaps = append(gaps, m)
				}
			}
			if len(gaps) > 0 {
				out[id] = gaps
			}
		}
		return out
	}
	for _, room := range r.Rooms {
		var gaps []MetricEntry
		for _, m := range r.Dashboard(room.ID) {
			if effectiveCoverage(m) != CoverageNow {
				gaps = append(gaps, m)
			}
		}
		if len(gaps) > 0 {
			out[room.ID] = gaps
		}
	}
	return out
}

func effectiveCoverage(m MetricEntry) Coverage {
	if m.Coverage != "" {
		return m.Coverage
	}
	return coverageFromLegacy(m.DataSource)
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
