package setpoint

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Cells the control plane consumes, named once.
const (
	CellCPUPressure    = "substrate/SB14"
	CellStrandedMemory = "substrate/SB15"
	CellForkRate       = "substrate/SB16"
	CellMemoryPSI      = "substrate/SB20"
	CellCrashLoop      = "availability/A2"
)

// RelativePath is where the setpoint lives under a repository root.
const RelativePath = "scenarios/infrastructure-manager/setpoint/reliability-setpoint.json"

// PathEnv overrides the file location; RootEnv is the repository root.
const (
	PathEnv = "VROOLI_SETPOINT_PATH"
	RootEnv = "VROOLI_ROOT"
)

// FallbackPath is the Path a Setpoint reports when no file could be read.
const FallbackPath = "compiled fallback"

// DefaultPressureSustain is the window a pressure consumer uses for a cell
// whose authored sustain is not a duration. It equals the ratified 10m of the
// three host-pressure bars, so an unauthored window never fires faster.
const DefaultPressureSustain = 10 * time.Minute

// ReclaimMinimumSwapped is the least swapped memory a stranded process must
// hold before a reclaim considers it; one value, shared by every reclaimer.
const ReclaimMinimumSwapped = 500 * 1024 * 1024

//go:embed reliability-setpoint.schema.json
var schemaDocument []byte

// Bar is one cell of the setpoint, exactly as the file authors it. The JSON
// tags are the file's; infrastructure-manager serves the same shape.
type Bar struct {
	ID          string   `json:"id"`
	CellRef     string   `json:"cell_ref"`
	Projection  string   `json:"projection"`
	TargetKind  string   `json:"target_kind"`
	Deadband    string   `json:"deadband"`
	Sustain     string   `json:"sustain"`
	Actuator    string   `json:"actuator"`
	DecisionRef string   `json:"decision_ref"`
	Unit        string   `json:"unit,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	// Gradeable records whether the operator authored a threshold this bar
	// can be graded against; a prose-only deadband must say so.
	Gradeable          bool   `json:"gradeable"`
	NotGradeableReason string `json:"not_gradeable_reason,omitempty"`
	// Provisional marks a bar authored alongside its projection and not yet
	// ratified by an operator setpoint review.
	Provisional       bool   `json:"provisional,omitempty"`
	ProvisionalReason string `json:"provisional_reason,omitempty"`
	// RatificationNote records the evidence a setpoint review accepted when
	// it cleared Provisional; it travels beside the number on purpose.
	RatificationNote string `json:"ratification_note,omitempty"`
	// Window is Sustain parsed as a duration, or zero when the authored text
	// is not one ("one read", "2 windows"). Consumers grade against Window.
	Window time.Duration `json:"-"`
}

// Confidence is the file's confidence block.
type Confidence struct {
	Level      string `json:"level"`
	Rationale  string `json:"rationale"`
	RecordedOn string `json:"recorded_on"`
}

// Constants are the file's shared constants.
type Constants struct {
	ReadDeadlineSeconds   int  `json:"read_deadline_seconds"`
	SaturationWindowHours int  `json:"saturation_window_hours"`
	ShelfExpiryRequired   bool `json:"shelf_expiry_required"`
	RetentionMarginDays   int  `json:"retention_margin_days"`
}

// Unmapped is a target kind the file deliberately does not grade.
type Unmapped struct {
	TargetKind string `json:"target_kind"`
	Reason     string `json:"reason"`
	Revisit    string `json:"revisit,omitempty"`
}

// Document is the whole file, typed.
type Document struct {
	SchemaVersion string     `json:"schema_version"`
	Confidence    Confidence `json:"confidence"`
	Constants     Constants  `json:"constants"`
	Bars          []Bar      `json:"bars"`
	Unmapped      []Unmapped `json:"unmapped,omitempty"`
}

// Setpoint is the parsed file with its provenance and cell index.
type Setpoint struct {
	Document
	Path   string
	byCell map[string]Bar
}

// Bar returns the bar for a cell.
func (s Setpoint) Bar(cellRef string) (Bar, bool) {
	bar, ok := s.byCell[cellRef]
	return bar, ok
}

// Max returns the bar's Max for a cell, or the fallback when the cell is
// absent or has no Max. Consumers that need a number and must keep running
// use it; the source is visible through Path.
func (s Setpoint) Max(cellRef string, fallback float64) float64 {
	if bar, ok := s.byCell[cellRef]; ok && bar.Max != nil {
		return *bar.Max
	}
	return fallback
}

// Sustain returns the bar's authored window for a cell, or the fallback
// when the cell is absent or its sustain is not a duration.
func (s Setpoint) Sustain(cellRef string, fallback time.Duration) time.Duration {
	if bar, ok := s.byCell[cellRef]; ok && bar.Window > 0 {
		return bar.Window
	}
	return fallback
}

// Load reads and validates one setpoint file.
func Load(path string) (Setpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Setpoint{}, err
	}
	return Parse(path, data)
}

// Parse validates and parses setpoint bytes; path is recorded for
// provenance. Beyond the schema it enforces the invariants a bar file must
// hold to be graded: unique cell_refs, a reason on every ungradeable bar,
// and a unit plus a threshold on every gradeable one.
func Parse(path string, data []byte) (Setpoint, error) {
	if err := validateSchema(data); err != nil {
		return Setpoint{}, fmt.Errorf("setpoint %s: %w", path, err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Setpoint{}, fmt.Errorf("setpoint %s: %w", path, err)
	}
	if len(doc.Bars) == 0 {
		return Setpoint{}, fmt.Errorf("setpoint %s: bars is empty", path)
	}
	out := Setpoint{Document: doc, Path: path, byCell: make(map[string]Bar, len(doc.Bars))}
	for i := range out.Bars {
		bar := &out.Bars[i]
		if strings.TrimSpace(bar.ID) == "" || strings.TrimSpace(bar.CellRef) == "" || strings.TrimSpace(bar.DecisionRef) == "" {
			return Setpoint{}, fmt.Errorf("setpoint %s: bar %d requires id, cell_ref and decision_ref", path, i)
		}
		if _, dup := out.byCell[bar.CellRef]; dup {
			return Setpoint{}, fmt.Errorf("setpoint %s: two bars share cell_ref %s", path, bar.CellRef)
		}
		if !bar.Gradeable && strings.TrimSpace(bar.NotGradeableReason) == "" {
			return Setpoint{}, fmt.Errorf("setpoint %s: bar %q is not gradeable and states no reason", path, bar.ID)
		}
		if bar.Gradeable {
			if bar.Min == nil && bar.Max == nil {
				return Setpoint{}, fmt.Errorf("setpoint %s: bar %q is marked gradeable but authors no min or max", path, bar.ID)
			}
			if strings.TrimSpace(bar.Unit) == "" {
				return Setpoint{}, fmt.Errorf("setpoint %s: bar %q is marked gradeable but names no unit", path, bar.ID)
			}
		}
		bar.Window, _ = ParseSustain(bar.Sustain)
		out.byCell[bar.CellRef] = *bar
	}
	return out, nil
}

func validateSchema(data []byte) error {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("reliability-setpoint.schema.json", bytes.NewReader(schemaDocument)); err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	schema, err := compiler.Compile("reliability-setpoint.schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	return nil
}

// ParseSustain reads an authored sustain. It accepts Go durations ("10m",
// "24h") and day counts ("30d"); any other text is not a duration and
// returns false so the caller keeps the authored words.
func ParseSustain(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if strings.HasSuffix(raw, "d") {
		if days, err := strconv.Atoi(strings.TrimSuffix(raw, "d")); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour, true
		}
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d, true
	}
	return 0, false
}

// Resolve finds the setpoint for a process: PathEnv, then RootEnv, then the
// working directory. A readable file that fails validation is an error, not
// a fallback: a broken bar file must be visible. When no candidate exists,
// the compiled Fallback is returned with a nil error.
func Resolve(env []string, cwd string) (Setpoint, error) {
	for _, candidate := range Candidates(env, cwd) {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		return Load(candidate)
	}
	return Fallback(), nil
}

// Candidates lists the paths Resolve tries, in order.
func Candidates(env []string, cwd string) []string {
	var paths []string
	if configured := strings.TrimSpace(envValue(env, PathEnv)); configured != "" {
		paths = append(paths, configured)
	}
	if root := strings.TrimSpace(envValue(env, RootEnv)); root != "" {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(RelativePath)))
	}
	if cwd != "" {
		paths = append(paths, filepath.Join(cwd, filepath.FromSlash(RelativePath)))
	}
	return paths
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, key+"="); ok {
			return value
		}
	}
	return ""
}

// The compiled fallback bars mirror the ratified 2026-08-22 host-pressure
// bars and the supervisor's memory gate, so a host without a checkout still
// brakes at the same values. They are the only numbers outside the file.
const (
	fallbackCPUPressurePercent  = 50.0
	fallbackStrandedMemoryMB    = 17200.0
	fallbackForksPerSecond      = 200.0
	fallbackMemoryPSIPercent    = 10.0
	fallbackCrashLoopsPerHour   = 2700.0
	fallbackPressureSustainText = "10m"
	fallbackDecisionRef         = "compiled-fallback"
)

// Fallback is the one compiled bar set, used only when no file is present.
func Fallback() Setpoint {
	f := func(v float64) *float64 { return &v }
	bars := []Bar{
		{ID: "substrate-cpu-pressure", CellRef: CellCPUPressure, Projection: "substrate", DecisionRef: fallbackDecisionRef, Unit: "percent", Max: f(fallbackCPUPressurePercent), Window: DefaultPressureSustain, Sustain: fallbackPressureSustainText, Gradeable: true},
		{ID: "substrate-memory-swap-pressure", CellRef: CellStrandedMemory, Projection: "substrate", DecisionRef: fallbackDecisionRef, Unit: "megabytes of stranded memory", Max: f(fallbackStrandedMemoryMB), Window: DefaultPressureSustain, Sustain: fallbackPressureSustainText, Gradeable: true},
		{ID: "substrate-process-and-fork-growth", CellRef: CellForkRate, Projection: "substrate", DecisionRef: fallbackDecisionRef, Unit: "forks per second", Max: f(fallbackForksPerSecond), Window: DefaultPressureSustain, Sustain: fallbackPressureSustainText, Gradeable: true},
		{ID: "substrate-memory-psi-stall", CellRef: CellMemoryPSI, Projection: "substrate", DecisionRef: fallbackDecisionRef, Unit: "percent", Max: f(fallbackMemoryPSIPercent), Sustain: "one read", Gradeable: true},
		{ID: "availability-declared-crash-loop", CellRef: CellCrashLoop, Projection: "availability", DecisionRef: fallbackDecisionRef, Unit: "restarts per hour", Max: f(fallbackCrashLoopsPerHour), Window: time.Hour, Sustain: "1h", Gradeable: true},
	}
	out := Setpoint{Path: FallbackPath, Document: Document{SchemaVersion: "1.0.0", Bars: bars}, byCell: make(map[string]Bar, len(bars))}
	for _, bar := range bars {
		out.byCell[bar.CellRef] = bar
	}
	return out
}
