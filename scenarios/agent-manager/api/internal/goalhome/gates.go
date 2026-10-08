package goalhome

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Typed exit gates (DL-4). An admitted brief lists its gates as
//
//	- G1 test: cd ui && pnpm type-check
//	- G2 journey: J01,J02,J03 @ bas-goal
//	- G3 inventory: runtime_lines <= 253452 from 254178 (python3 docs/internal/refactor_inventory.py --no-git)
//	- G4 review: deletion-list
//	- G5 custom: offline archive playback — reason: no automated offline check
//
// and amends them only with a dated, reasoned line under ## Gate amendments:
//
//	- A1 2026-10-07T12:00:00Z G5 drop — reason: unverifiable; tool logged in WORKAROUNDS
//	- A2 2026-10-07T12:00:00Z G2 unverified — reason: shadow down; tool logged in WORKAROUNDS
//	- A3 2026-10-07T12:00:00Z G6 add journey: J05 @ bas-goal — reason: …
//
// Results are slice-log cells "gate G<n>=pass|fail|unverified [value]"; the
// latest cell wins, and the ACCEPTED line's gates= names gates the
// orchestrator re-ran itself. Evidence outside the admitted items is never read.

// Gate types.
const (
	GateTest      = "test"
	GateJourney   = "journey"
	GateInventory = "inventory"
	GateReview    = "review"
	GateCustom    = "custom"
)

// Gate states and the epoch's overall gate status.
const (
	GateMet     = "met"
	GateUnmet   = "unmet"
	GateDropped = "dropped"
	// GatesLegacy means the exit gate has no typed G<n> items; it is never checked.
	GatesLegacy = "legacy"
)

// Gate is one typed exit-gate item.
type Gate struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Spec string `json:"spec"`
	Line int    `json:"line"`
}

// Amendment is one dated, reasoned change to the admitted gates.
type Amendment struct {
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Gate   string    `json:"gate"`
	Action string    `json:"action"`
	Body   string    `json:"body,omitempty"`
	Reason string    `json:"reason"`
	Line   int       `json:"line"`
}

// GateResult is one gate judged against its latest recorded result.
type GateResult struct {
	Gate
	State     string    `json:"state"`
	Result    string    `json:"result,omitempty"`
	Value     string    `json:"value,omitempty"`
	Detail    string    `json:"detail"`
	AmendedBy []string  `json:"amended_by,omitempty"`
	Measured  *float64  `json:"measured,omitempty"`
	Yield     *Quantity `json:"yield,omitempty"`
}

// GateReport is the epoch's gate verdict.
type GateReport struct {
	GateStatus string       `json:"gate_status"`
	Gates      []GateResult `json:"gates"`
	Warnings   []Finding    `json:"warnings,omitempty"`
}

var (
	gateLinePattern      = regexp.MustCompile(`^- (G\d+) ([A-Za-z]+):\s*(.*)$`)
	amendmentLinePattern = regexp.MustCompile(`^- (A\d+) (\S+) (G\d+) ([A-Za-z]+)\b:?\s*(.*)$`)
	gateBodyPattern      = regexp.MustCompile(`^([A-Za-z]+):\s*(.+)$`)
	inventorySpecPattern = regexp.MustCompile(`^(.+?)\s*(<=|>=|≤|≥|<|>|=)\s*([+\-−]?\d[\d,_]*(?:\.\d+)?)(?:\s+from\s+([+\-−]?\d[\d,_]*(?:\.\d+)?))?`)
	journeyIDPattern     = regexp.MustCompile(`\bJ\d+\b`)
	journeyCountPattern  = regexp.MustCompile(`^(\d+)\s*/\s*(\d+)`)
	gateTypes            = map[string]bool{GateTest: true, GateJourney: true, GateInventory: true, GateReview: true, GateCustom: true}
	amendmentActions     = map[string]bool{"add": true, "drop": true, "change": true, "unverified": true}
)

// parseGateLine reads one non-blank line of ## Exit gate or ## Gate amendments.
func (e *Epoch) parseGateLine(text string, num int, exitGate bool) {
	if m := amendmentLinePattern.FindStringSubmatch(text); m != nil {
		at, _ := parseWhen(m[2])
		body, reason := splitReason(m[5])
		e.Amendments = append(e.Amendments, Amendment{ID: m[1], At: at, Gate: strings.ToUpper(m[3]), Action: strings.ToLower(m[4]), Body: body, Reason: reason, Line: num})
		return
	}
	if m := gateLinePattern.FindStringSubmatch(text); m != nil && exitGate {
		kind := strings.ToLower(m[2])
		if gateTypes[kind] {
			e.Gates = append(e.Gates, Gate{ID: strings.ToUpper(m[1]), Type: kind, Spec: strings.TrimSpace(m[3]), Line: num})
			return
		}
		e.GateNotes = append(e.GateNotes, Finding{Code: "untyped-gate", Line: num, Detail: fmt.Sprintf("%s has unknown type %q; use test, journey, inventory, review or custom", m[1], m[2])})
		return
	}
	code := "untyped-gate"
	if !exitGate {
		code = "gate-amendment-reason"
	}
	e.GateNotes = append(e.GateNotes, Finding{Code: code, Line: num, Detail: "not a typed item: " + truncate(text, 120)})
}

// splitReason splits "<body> — reason: <text>" at the last "reason:".
func splitReason(text string) (string, string) {
	idx := strings.LastIndex(strings.ToLower(text), "reason:")
	if idx < 0 {
		return strings.TrimSpace(text), ""
	}
	body := strings.TrimRight(strings.TrimSpace(text[:idx]), " —–-;,|")
	return body, strings.TrimSpace(text[idx+len("reason:"):])
}

// EvaluateGates judges the admitted gates, after amendments, against the
// recorded results. A legacy epoch (no typed items) reports GatesLegacy and
// one warning; it is never failed for its gate format.
func EvaluateGates(e *Epoch) GateReport {
	report := GateReport{Gates: []GateResult{}}
	var warnings []Finding
	gates := map[string]*GateResult{}
	var order []string
	for _, gate := range e.Gates {
		if _, dup := gates[gate.ID]; dup {
			warnings = append(warnings, Finding{Code: "duplicate-gate", Line: gate.Line, Detail: gate.ID + " is listed twice; the first item counts"})
			continue
		}
		gates[gate.ID] = &GateResult{Gate: gate}
		order = append(order, gate.ID)
		warnings = append(warnings, gateShapeWarnings(gate)...)
	}
	unverifiedBy := map[string]string{}
	for _, amendment := range e.Amendments {
		if amendment.At.IsZero() || amendment.Reason == "" || !amendmentActions[amendment.Action] {
			warnings = append(warnings, Finding{Code: "gate-amendment-reason", Line: amendment.Line, Detail: amendment.ID + " is not applied: an amendment needs an ISO date, add|drop|change|unverified and a reason"})
			continue
		}
		target := gates[amendment.Gate]
		switch amendment.Action {
		case "drop", "unverified":
			if target == nil {
				warnings = append(warnings, Finding{Code: "gate-amendment-reason", Line: amendment.Line, Detail: amendment.ID + " names " + amendment.Gate + ", which is not an admitted gate"})
				continue
			}
			if amendment.Action == "drop" {
				target.State = GateDropped
			} else {
				unverifiedBy[amendment.Gate] = amendment.ID
			}
		case "add", "change":
			m := gateBodyPattern.FindStringSubmatch(amendment.Body)
			if m == nil || !gateTypes[strings.ToLower(m[1])] {
				warnings = append(warnings, Finding{Code: "gate-amendment-reason", Line: amendment.Line, Detail: amendment.ID + " is not applied: " + amendment.Action + " needs \"<type>: <spec>\""})
				continue
			}
			gate := Gate{ID: amendment.Gate, Type: strings.ToLower(m[1]), Spec: strings.TrimSpace(m[2]), Line: amendment.Line}
			if target == nil {
				target = &GateResult{}
				gates[gate.ID] = target
				order = append(order, gate.ID)
			}
			target.Gate, target.State = gate, ""
			warnings = append(warnings, gateShapeWarnings(gate)...)
		}
		target.AmendedBy = append(target.AmendedBy, amendment.ID)
	}
	if len(order) == 0 {
		report.GateStatus = GatesLegacy
		detail := "no ## Exit gate section"
		if e.HasExitGate {
			detail = "the exit gate has no typed G<n> items (test, journey, inventory, review, custom); acceptance is not checked against it"
		}
		report.Warnings = []Finding{{Code: "legacy-gates", Detail: detail}}
		return report
	}
	for _, note := range e.GateNotes {
		warnings = append(warnings, note)
	}

	latest := map[string]GateCell{}
	for _, entry := range e.Log {
		for _, cell := range entry.Gates {
			latest[cell.Gate] = cell
		}
	}
	if accepted := e.LastAcceptance(); accepted != nil {
		for _, cell := range accepted.Results {
			latest[cell.Gate] = cell
		}
		for _, id := range accepted.Gates {
			cell := latest[id]
			cell.Gate, cell.Result = id, "pass"
			latest[id] = cell
		}
	}

	report.GateStatus = GateMet
	for _, id := range order {
		result := gates[id]
		if result.State == GateDropped {
			result.Detail = "dropped by amendment " + strings.Join(result.AmendedBy, ", ")
		} else {
			cell, recorded := latest[id]
			judgeGate(result, cell, recorded, unverifiedBy[id])
			if result.State == GateUnmet {
				report.GateStatus = GateUnmet
			}
		}
		report.Gates = append(report.Gates, *result)
	}
	report.Warnings = warnings
	return report
}

func gateShapeWarnings(gate Gate) []Finding {
	switch gate.Type {
	case GateInventory:
		if inventorySpecPattern.FindStringSubmatch(gate.Spec) == nil {
			return []Finding{{Code: "inventory-gate-shape", Line: gate.Line, Detail: gate.ID + " needs \"<metric> <op> <n> [from <baseline>]\""}}
		}
	case GateCustom:
		if !strings.Contains(strings.ToLower(gate.Spec), "reason:") {
			return []Finding{{Code: "custom-gate-reason", Line: gate.Line, Detail: gate.ID + " is custom without a stated reason"}}
		}
	}
	return nil
}

// judgeGate sets the state of one gate from its latest recorded result.
func judgeGate(result *GateResult, cell GateCell, recorded bool, unverifiedBy string) {
	result.State = GateUnmet
	if !recorded {
		if unverifiedBy != "" {
			result.State, result.Detail = GateMet, "recorded unverified by amendment "+unverifiedBy
			return
		}
		result.Detail = "no result recorded (slice-log cell gate " + result.ID + "=pass|fail|unverified [value])"
		return
	}
	result.Result, result.Value = cell.Result, cell.Value
	switch cell.Result {
	case "fail":
		result.Detail = "failed"
	case "unverified":
		if unverifiedBy != "" {
			result.State, result.Detail = GateMet, "unverified, accepted by amendment "+unverifiedBy
		} else {
			result.Detail = "unverified; it counts as met only after a dated \"unverified\" amendment with a reason"
		}
	case "pass":
		judgePass(result)
	default:
		result.Detail = "unknown result " + cell.Result + " (want pass, fail or unverified)"
	}
}

func judgePass(result *GateResult) {
	switch result.Type {
	case GateJourney:
		judgeJourney(result)
	case GateInventory:
		judgeInventory(result)
	default:
		result.State, result.Detail = GateMet, "passed"
	}
}

// judgeJourney requires the value to cover every listed journey: either the
// IDs themselves or a count such as 12/12.
func judgeJourney(result *GateResult) {
	spec, _, _ := strings.Cut(result.Spec, "@")
	want := journeyIDPattern.FindAllString(spec, -1)
	if m := journeyCountPattern.FindStringSubmatch(result.Value); m != nil {
		passed, total := leadingNumber(m[1]), leadingNumber(m[2])
		switch {
		case passed != total:
			result.Detail = fmt.Sprintf("%d of %d journeys passed", passed, total)
		case total < len(want):
			result.Detail = fmt.Sprintf("%d/%d covers fewer than the %d listed journeys", passed, total, len(want))
		default:
			result.State, result.Detail = GateMet, fmt.Sprintf("%d/%d journeys passed", passed, total)
		}
		return
	}
	got := map[string]bool{}
	for _, id := range journeyIDPattern.FindAllString(result.Value, -1) {
		got[id] = true
	}
	var missing []string
	for _, id := range want {
		if !got[id] {
			missing = append(missing, id)
		}
	}
	switch {
	case len(got) == 0:
		result.Detail = "a journey result needs a count such as 12/12 or the passing journey IDs"
	case len(missing) > 0:
		result.Detail = "no pass recorded for " + strings.Join(missing, ",")
	default:
		result.State, result.Detail = GateMet, "every listed journey passed"
	}
}

// judgeInventory compares the measured value with the bound; "pass" alone is
// not trusted.
func judgeInventory(result *GateResult) {
	m := inventorySpecPattern.FindStringSubmatch(result.Spec)
	if m == nil {
		result.Detail = "spec is not \"<metric> <op> <n>\""
		return
	}
	measured, _, ok := ParseNumber(result.Value)
	if !ok {
		result.Detail = "pass without a measured value; record gate " + result.ID + "=pass <value>"
		return
	}
	bound, _, _ := ParseNumber(m[3])
	result.Measured = &measured
	metric := strings.TrimSpace(m[1])
	if m[4] != "" {
		baseline, _, _ := ParseNumber(m[4])
		result.Yield = &Quantity{Value: measured - baseline, Unit: metric}
	}
	if compare(measured, m[2], bound) {
		result.State, result.Detail = GateMet, fmt.Sprintf("%s %s %s %s", metric, formatNumber(measured), m[2], formatNumber(bound))
		return
	}
	result.Detail = fmt.Sprintf("%s %s is not %s %s", metric, formatNumber(measured), m[2], formatNumber(bound))
}

func compare(value float64, op string, bound float64) bool {
	switch op {
	case "<=", "≤":
		return value <= bound
	case "<":
		return value < bound
	case ">=", "≥":
		return value >= bound
	case ">":
		return value > bound
	default:
		return value == bound
	}
}

func formatNumber(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "…"
}
