// Instrument coverage: does each team declare the one scenario it reads for
// the state of its domain, or an honest, dated hole where that scenario is not
// yet built?
//
// The target model this sensor reads against is docs/agent-system/TARGET_MODEL.md.
// Two properties of the declaration are deliberate and easy to get wrong:
//
//  1. `status: "none"` is a *valid* value, not a failure. Four of six teams
//     have no instrument today, and a declaration that could only say "yes" is
//     a declaration nobody would file. What is out of band is an *undeclared*
//     hole — silence — because silence cannot be counted or aged.
//
//  2. Nothing here requires a team to have an instrument. The deadband is
//     "declared or dated", never "present". Enforcement would make the sensor
//     into a controller, which is the same boundary violation the target model
//     forbids for instruments themselves.
//
// DOC: docs/agent-system/TARGET_MODEL.md § The instrument: six invariants
package memberflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// OperatingRuleGroupInstrument groups the instrument-declaration rules.
const OperatingRuleGroupInstrument RuleGroup = "instrument"

// Instrument status vocabulary. A team is in band with any of the three so
// long as the declaration is coherent (see instrumentFindings).
const (
	// InstrumentStatusLive means one scenario answers "what is the state of my
	// domain and what should I do next" for this team.
	InstrumentStatusLive = "live"
	// InstrumentStatusPartial means the capability exists across more than one
	// scenario with no single address, or the board serves only some modes.
	InstrumentStatusPartial = "partial"
	// InstrumentStatusNone means no instrument exists. Valid, and requires a
	// dated gap marker so the hole can be aged rather than merely noticed.
	InstrumentStatusNone = "none"
)

// Instrument archetype vocabulary. The two shapes differ in which error term
// dominates, not in their cell schema.
const (
	// InstrumentArchetypeCoverageBoard fits a bounded supply whose denominator
	// can be honestly authored; the board returns ratios with confidence.
	InstrumentArchetypeCoverageBoard = "coverage-board"
	// InstrumentArchetypeProductionLedger fits unbounded output, where no
	// denominator is defensible; the board returns queue state and staleness.
	InstrumentArchetypeProductionLedger = "production-ledger"
)

// TeamInstrument is the `instrument` block on a team.json.
type TeamInstrument struct {
	Status    string `json:"status"`
	Scenario  string `json:"scenario,omitempty"`
	Archetype string `json:"archetype,omitempty"`
	// CoversScenarios names the scenarios whose capability this instrument
	// aggregates. It is the guard series for the orientation-cost band: a rise
	// in orientation cost is only a defect when this grew in the same cycle.
	// It replaced a proxy that counted operating-graph external nodes, which
	// resolved to `operator` and `vision-walk` rather than to scenarios.
	CoversScenarios []string `json:"coversScenarios,omitempty"`
	// GapMarker is a leading-YYYY-MM-DD rationale, required whenever status is
	// not live. It converts an absence into a dated, ageable one.
	GapMarker string `json:"gapMarker,omitempty"`
}

// InstrumentReading is one team's declaration plus why it is or is not in band.
type InstrumentReading struct {
	TeamID string `json:"teamId"`
	// Declared is false when team.json carries no instrument block at all.
	// That is the only silent state, and the only one the deadband fails on
	// without qualification.
	Declared   bool            `json:"declared"`
	Instrument *TeamInstrument `json:"instrument,omitempty"`
	// Findings names each way this declaration falls outside the deadband.
	// Empty means in band.
	Findings []string `json:"findings,omitempty"`
	// GapOpenedOn is the parsed leading date of GapMarker, so an intentionally
	// visible hole can be told apart from an overdue one.
	GapOpenedOn string `json:"gapOpenedOn,omitempty"`
	Reachable   *bool  `json:"reachable,omitempty"`
}

// InstrumentReachabilityChecker probes the declared instrument's scenario.
// It is injected so declaration validation remains deterministic and tests do
// not depend on the local scenario fleet.
type InstrumentReachabilityChecker interface {
	Check(context.Context, string) error
}

type scenarioInstrumentReachabilityChecker struct {
	resolver instrumentScenarioURLResolver
	client   *http.Client
}

type instrumentScenarioURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

func (c scenarioInstrumentReachabilityChecker) Check(ctx context.Context, scenario string) error {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, strings.TrimSpace(scenario))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("health returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// InBand reports whether this reading raises nothing.
func (r InstrumentReading) InBand() bool { return len(r.Findings) == 0 }

// InstrumentCoverageReport is the JSON shape for GET /instruments.
type InstrumentCoverageReport struct {
	Teams []InstrumentReading `json:"teams"`
	// Live, Partial and None count declared statuses; Undeclared counts teams
	// with no block. The four sum to len(Teams).
	Live       int `json:"live"`
	Partial    int `json:"partial"`
	None       int `json:"none"`
	Undeclared int `json:"undeclared"`
	// OutOfBand counts teams with at least one finding.
	OutOfBand int    `json:"outOfBand"`
	Note      string `json:"note"`
}

const instrumentReportNote = "The deadband is 'every team declares an instrument or carries a dated gap marker', never 'every team has one'. status=none with a dated gapMarker is in band."

// teamInstrumentFile is the minimal slice of team.json this parser reads.
type teamInstrumentFile struct {
	ID         string          `json:"id"`
	Instrument *TeamInstrument `json:"instrument,omitempty"`
}

// LoadTeamInstruments reads every team.json's instrument block, keyed by team
// id. A team with no block maps to nil, which the reading distinguishes from a
// block that declares `none`.
func LoadTeamInstruments(configDir string) (map[string]*TeamInstrument, error) {
	out := map[string]*TeamInstrument{}
	if strings.TrimSpace(configDir) == "" {
		return out, nil
	}
	teamsDir := filepath.Join(configDir, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("memberflow: read teams dir %q: %w", teamsDir, err)
	}
	for _, te := range entries {
		if !te.IsDir() || strings.HasPrefix(te.Name(), ".") {
			continue
		}
		path := filepath.Join(teamsDir, te.Name(), "team.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("memberflow: read %q: %w", path, err)
		}
		var tf teamInstrumentFile
		if err := json.Unmarshal(data, &tf); err != nil {
			return nil, fmt.Errorf("memberflow: parse %q: %w", path, err)
		}
		id := strings.TrimSpace(tf.ID)
		if id == "" {
			id = te.Name()
		}
		out[id] = tf.Instrument
	}
	return out, nil
}

// ComputeInstrumentCoverage reads every team's declaration and bands it.
func ComputeInstrumentCoverage(configDir string) (InstrumentCoverageReport, error) {
	return computeInstrumentCoverage(configDir, nil)
}

// ComputeInstrumentCoverageWithReachability preserves the declaration
// deadband while checking whether each declared live instrument answers.
func ComputeInstrumentCoverageWithReachability(configDir string, checker InstrumentReachabilityChecker) (InstrumentCoverageReport, error) {
	return computeInstrumentCoverage(configDir, checker)
}

func computeInstrumentCoverage(configDir string, checker InstrumentReachabilityChecker) (InstrumentCoverageReport, error) {
	report := InstrumentCoverageReport{Teams: []InstrumentReading{}, Note: instrumentReportNote}
	declared, err := LoadTeamInstruments(configDir)
	if err != nil {
		return report, err
	}
	for _, teamID := range sortedMapKeys(declared) {
		reading := bandInstrument(teamID, declared[teamID])
		if checker != nil && reading.Instrument != nil && reading.Instrument.Status == InstrumentStatusLive && strings.TrimSpace(reading.Instrument.Scenario) != "" {
			probeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := checker.Check(probeCtx, reading.Instrument.Scenario)
			cancel()
			reachable := err == nil
			reading.Reachable = &reachable
			if err != nil {
				reading.Findings = append(reading.Findings, fmt.Sprintf("declared instrument scenario %q is unavailable: %v", reading.Instrument.Scenario, err))
			}
		}
		switch {
		case !reading.Declared:
			report.Undeclared++
		case reading.Instrument.Status == InstrumentStatusLive:
			report.Live++
		case reading.Instrument.Status == InstrumentStatusPartial:
			report.Partial++
		case reading.Instrument.Status == InstrumentStatusNone:
			report.None++
		}
		if !reading.InBand() {
			report.OutOfBand++
		}
		report.Teams = append(report.Teams, reading)
	}
	return report, nil
}

// bandInstrument applies the deadband to one declaration.
//
// Each finding names a specific incoherence rather than a generic "invalid",
// because the actuator differs: a missing gap marker is a one-line edit by the
// team, while a live status with no scenario means the declaration and reality
// disagree and someone has to decide which is wrong.
func bandInstrument(teamID string, inst *TeamInstrument) InstrumentReading {
	reading := InstrumentReading{TeamID: teamID}
	if inst == nil {
		reading.Findings = append(reading.Findings,
			"no instrument block declared; declare one, using status \"none\" with a dated gapMarker if the team has no instrument yet")
		return reading
	}
	reading.Declared = true
	reading.Instrument = inst

	status := strings.TrimSpace(inst.Status)
	switch status {
	case InstrumentStatusLive, InstrumentStatusPartial, InstrumentStatusNone:
	case "":
		reading.Findings = append(reading.Findings, "instrument.status is empty; expected live, partial, or none")
	default:
		reading.Findings = append(reading.Findings,
			fmt.Sprintf("instrument.status %q is outside the vocabulary (live, partial, none)", status))
	}

	if status == InstrumentStatusLive && strings.TrimSpace(inst.Scenario) == "" {
		reading.Findings = append(reading.Findings, "instrument.status is live but no scenario is named")
	}

	// A hole must be dated. Without this the sensor can see that a team lacks
	// an instrument but not whether the absence is a fresh decision or a
	// three-year-old one, which is the difference between a plan and a rot.
	if status == InstrumentStatusNone || status == InstrumentStatusPartial {
		if strings.TrimSpace(inst.GapMarker) == "" {
			reading.Findings = append(reading.Findings,
				fmt.Sprintf("instrument.status is %q with no gapMarker; a hole must carry a leading YYYY-MM-DD rationale", status))
		}
	}
	if marker := strings.TrimSpace(inst.GapMarker); marker != "" {
		if opened, ok := instrumentGapDate(marker); ok {
			reading.GapOpenedOn = opened
		} else {
			reading.Findings = append(reading.Findings,
				"instrument.gapMarker does not start with a YYYY-MM-DD date, so the hole cannot be aged")
		}
	}

	if arch := strings.TrimSpace(inst.Archetype); arch != "" {
		if arch != InstrumentArchetypeCoverageBoard && arch != InstrumentArchetypeProductionLedger {
			reading.Findings = append(reading.Findings,
				fmt.Sprintf("instrument.archetype %q is outside the vocabulary (coverage-board, production-ledger)", arch))
		}
	}
	return reading
}

// instrumentGapDate lifts the leading YYYY-MM-DD out of a gap marker. It
// matches the convention the framework-health sensor map uses so a reader does
// not have to learn two marker formats.
func instrumentGapDate(marker string) (string, bool) {
	if len(marker) < 10 {
		return "", false
	}
	head := marker[:10]
	if head[4] != '-' || head[7] != '-' {
		return "", false
	}
	for i, c := range head {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return "", false
		}
	}
	return head, true
}

// GetInstruments handles GET /instruments.
func (h *Handlers) GetInstruments(w http.ResponseWriter, r *http.Request) {
	report, err := ComputeInstrumentCoverageWithReachability(h.configDir, h.instrumentProbe)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// instrumentCoveredScenarios projects the declared coversScenarios sets, keyed
// by team, for the orientation-cost guard series.
func instrumentCoveredScenarios(declared map[string]*TeamInstrument) map[string][]string {
	out := map[string][]string{}
	for teamID, inst := range declared {
		if inst == nil {
			continue
		}
		seen := map[string]bool{}
		var names []string
		for _, s := range inst.CoversScenarios {
			name := strings.TrimSpace(s)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
		sort.Strings(names)
		out[teamID] = names
	}
	return out
}
