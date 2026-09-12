// Package conformance defines the evidence document produced by one
// end-to-end dictation soak run.
//
// It exists because the previous evidence surface could not tell a
// measurement from a default. Reliability facts were spread across three
// artifacts written by three processes — a browser unit test, a Go ledger unit
// test, and a browser-automation click-through — each shallow-merging its keys
// into one shared JSON file behind its own environment variable. Nothing
// recorded which run produced which block, or whether the blocks came from the
// same code. Two of the fields that mattered most, "no committed segment was
// delivered twice" and "no turn ended without a reason", were literals in test
// source that nothing ever computed.
//
// The rules here follow from that:
//
//  1. An assertion is a tri-state. `not_measured` is a first-class outcome and
//     it FAILS the run. A harness that cannot observe something must say so and
//     go red, rather than emitting a zero that reads as success.
//  2. A required assertion that is absent fails exactly like one that is
//     `not_measured`. Silence is not a pass.
//  3. Every run carries its own identity, the identity of the code it
//     exercised, and the provider cell it ran against. Evidence from one cell
//     can never credit another.
package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Outcome is the tri-state result of one assertion.
type Outcome string

const (
	// OutcomePassed means the harness observed the property and it held.
	OutcomePassed Outcome = "passed"
	// OutcomeFailed means the harness observed the property and it did not hold.
	OutcomeFailed Outcome = "failed"
	// OutcomeNotMeasured means the harness did not observe the property at all.
	// This is a failure, not a neutral state: an unobserved reliability claim
	// is indistinguishable from a false one.
	OutcomeNotMeasured Outcome = "not_measured"
)

// Lane distinguishes what a run is entitled to claim.
type Lane string

const (
	// LaneAccelerated drives a virtual clock. It can prove invariants
	// (accounting, retention bounds, constant per-frame cost) but never that
	// the system keeps up with a human, because no real time passes.
	LaneAccelerated Lane = "accelerated"
	// LaneRealtime drives the capture path at wall-clock speed. Only this lane
	// may claim latency, lag, and duration-ladder credit.
	LaneRealtime Lane = "realtime"
)

// Cell identifies exactly what was under test. Evidence is keyed to a cell so
// a passing run on one engine can never credit another.
type Cell struct {
	EngineID string `json:"engineId"`
	ModelID  string `json:"modelId"`
	Strategy string `json:"strategy"`
	Policy   string `json:"policy"`
}

func (c Cell) valid() error {
	switch {
	case strings.TrimSpace(c.EngineID) == "":
		return fmt.Errorf("cell.engineId is required")
	case strings.TrimSpace(c.ModelID) == "":
		return fmt.Errorf("cell.modelId is required")
	case strings.TrimSpace(c.Strategy) == "":
		return fmt.Errorf("cell.strategy is required")
	}
	return nil
}

// Code fingerprints the software the run exercised. Without it a comparison
// between two runs cannot attribute a change to anything.
type Code struct {
	// CapturePackage digests the browser capture substrate actually loaded.
	CapturePackage string `json:"capturePackage"`
	// Server digests the audio-tools server binary that served the session.
	Server string `json:"server"`
}

func (c Code) valid() error {
	switch {
	case strings.TrimSpace(c.CapturePackage) == "":
		return fmt.Errorf("code.capturePackage fingerprint is required")
	case strings.TrimSpace(c.Server) == "":
		return fmt.Errorf("code.server fingerprint is required")
	}
	return nil
}

// Assertion is one reliability claim and the observation behind it.
type Assertion struct {
	Name    string  `json:"name"`
	Outcome Outcome `json:"outcome"`
	// Detail states what was observed, in the units of the thing measured.
	// For OutcomeNotMeasured it must state why the harness could not see it.
	Detail string `json:"detail"`
}

// Run is one soak execution, start to finish, observed by one process.
type Run struct {
	SchemaVersion int    `json:"schemaVersion"`
	RunID         string `json:"runId"`
	Lane          Lane   `json:"lane"`
	Profile       string `json:"profile"`
	// Shape identifies the browser session shape exercised by the run. It is
	// recorded in the evidence rather than inferred from an output filename.
	Shape string `json:"shape,omitempty"`
	// SimulatedMinutes is the accelerated virtual-corpus target. It is zero for
	// realtime runs, where wall-clock duration is the meaningful measure.
	SimulatedMinutes int `json:"simulatedMinutes,omitempty"`
	Turns            int `json:"turns,omitempty"`
	// Fault identifies a deterministic fault-matrix run. Fault runs have a
	// deliberately smaller assertion contract: an explicit terminal failure is
	// the expected outcome for some profiles, while recovery is expected for
	// others. They must not be judged as ordinary healthy-turn soaks.
	Fault      string      `json:"fault,omitempty"`
	Cell       Cell        `json:"cell"`
	Code       Code        `json:"code"`
	DurationMs int64       `json:"durationMs"`
	Assertions []Assertion `json:"assertions"`
}

// SchemaVersion is the current Run schema.
const SchemaVersion = 1

// Invariant assertions every lane must measure. These are properties of the
// pipeline, not of the machine, so a lane that cannot observe one of them is
// not a weaker lane — it is an incomplete instrument.
var InvariantAssertions = []string{
	"interval_accounting_exactly_once",
	"capture_signal_observed",
	"browser_retention_bounded",
	"server_retention_bounded",
	"per_frame_cost_constant",
	"zero_duplicate_committed_segments",
	"zero_silent_terminal_outcomes",
	"provider_cell_identity",
	"peak_memory_flat",
	"wire_rate_within_budget",
}

// RealtimeAssertions are the claims only wall-clock execution can support.
// The accelerated lane must not report them at all; reporting them there is a
// category error, because a virtual clock cannot fail to keep up.
var RealtimeAssertions = []string{
	"first_partial_latency_stable",
	"committed_text_lag_stable",
	"continuous_interim_text_visible",
}

// QualityAssertions are required on both lanes. Recognition quality is not a
// wall-clock claim, so an accelerated run must measure it from an independent
// reference just as a realtime run does.
var QualityAssertions = []string{
	"word_error_rate_stable",
	"punctuation_rate_recorded",
	"capitalisation_rate_recorded",
}

// AcceleratedAssertions are the virtual-clock-specific proof obligations.
var AcceleratedAssertions = []string{"accelerated_duration_target", "accelerated_wall_budget"}

// FaultAssertions are the obligations shared by every deterministic fault
// profile. They prove the injected fault was observed, committed segments
// remained idempotent, and the turn either recovered or ended explicitly.
var FaultAssertions = []string{
	"fault_profile_observed",
	"fault_interval_accounting",
	"fault_no_duplicate_committed_segments",
	"fault_recovered_or_terminal",
}

// RequiredAssertions returns the names a run on this lane must carry.
func RequiredAssertions(lane Lane) []string {
	required := append([]string(nil), InvariantAssertions...)
	required = append(required, QualityAssertions...)
	if lane == LaneRealtime {
		required = append(required, RealtimeAssertions...)
	} else if lane == LaneAccelerated {
		required = append(required, AcceleratedAssertions...)
	}
	return required
}

// RequiredAssertionsForRun returns the contract for the run shape. A fault
// profile is not a healthy-turn qualification and must not receive accidental
// quality or duration credit.
func RequiredAssertionsForRun(r Run) []string {
	if strings.TrimSpace(r.Fault) != "" {
		return append([]string(nil), FaultAssertions...)
	}
	return RequiredAssertions(r.Lane)
}

// Verdict is the decision derived from a run. Qualified is true only when
// every required assertion was measured and held.
type Verdict struct {
	Qualified bool     `json:"qualified"`
	Reasons   []string `json:"reasons"`
}

// Evaluate derives the verdict. It is deliberately unforgiving: the failure
// mode this package exists to prevent is a run that looks green because
// something was never looked at.
func (r Run) Evaluate() Verdict {
	var reasons []string

	if r.SchemaVersion != SchemaVersion {
		reasons = append(reasons, fmt.Sprintf("schemaVersion %d is not the supported version %d", r.SchemaVersion, SchemaVersion))
	}
	if strings.TrimSpace(r.RunID) == "" {
		reasons = append(reasons, "runId is required: evidence without a run identity cannot be compared or reproduced")
	}
	if r.Lane != LaneAccelerated && r.Lane != LaneRealtime {
		reasons = append(reasons, fmt.Sprintf("lane %q is not a recognised lane", r.Lane))
	}
	if err := r.Cell.valid(); err != nil {
		reasons = append(reasons, err.Error())
	}
	if err := r.Code.valid(); err != nil {
		reasons = append(reasons, err.Error())
	}

	seen := make(map[string]Assertion, len(r.Assertions))
	for _, a := range r.Assertions {
		if _, dup := seen[a.Name]; dup {
			reasons = append(reasons, fmt.Sprintf("assertion %q is reported more than once", a.Name))
			continue
		}
		seen[a.Name] = a
	}

	for _, name := range RequiredAssertionsForRun(r) {
		a, ok := seen[name]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("required assertion %q is absent; an unreported claim is not a passing one", name))
			continue
		}
		switch a.Outcome {
		case OutcomePassed:
		case OutcomeFailed:
			reasons = append(reasons, fmt.Sprintf("assertion %q failed: %s", name, detailOrPlaceholder(a.Detail)))
		case OutcomeNotMeasured:
			reasons = append(reasons, fmt.Sprintf("assertion %q was not measured: %s", name, detailOrPlaceholder(a.Detail)))
		default:
			reasons = append(reasons, fmt.Sprintf("assertion %q has unrecognised outcome %q", name, a.Outcome))
		}
	}

	// A realtime claim on the accelerated lane is not a bonus; a virtual clock
	// cannot fail to keep up, so reporting it would manufacture credit.
	if r.Lane == LaneAccelerated {
		for _, name := range RealtimeAssertions {
			if a, ok := seen[name]; ok && a.Outcome == OutcomePassed {
				reasons = append(reasons, fmt.Sprintf(
					"assertion %q was reported as passed on the accelerated lane; wall-clock claims are only earnable in real time", name))
			}
		}
	}

	sort.Strings(reasons)
	return Verdict{Qualified: len(reasons) == 0, Reasons: reasons}
}

func detailOrPlaceholder(detail string) string {
	if strings.TrimSpace(detail) == "" {
		return "no detail recorded"
	}
	return detail
}

// Measured builds a passed/failed assertion from an observation. Harnesses
// should use it rather than writing an Outcome by hand, so the outcome is
// always derived from a value rather than asserted.
func Measured(name string, held bool, detail string) Assertion {
	outcome := OutcomeFailed
	if held {
		outcome = OutcomePassed
	}
	return Assertion{Name: name, Outcome: outcome, Detail: detail}
}

// NotMeasured records that the harness could not observe a property. `why` is
// required: "not observable here" with no reason is how an instrument gap
// becomes permanent.
func NotMeasured(name, why string) Assertion {
	return Assertion{Name: name, Outcome: OutcomeNotMeasured, Detail: why}
}

// WriteJSON writes exactly one self-contained run document. The writer does
// not merge with an existing artifact: a run identity must own the complete
// observation set, including failed assertions.
func (r Run) WriteJSON(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("conformance: nil output writer")
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}
