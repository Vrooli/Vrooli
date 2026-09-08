// Package testquality defines scoped test-quality claims independently of
// framework evaluation, execution scheduling, and presentation.
package testquality

import (
	"encoding/json"
	"fmt"
	"sort"
)

const SchemaVersion = "test-quality/v1"

type Status string

const (
	Violation     Status = "violation"
	CheckedClean  Status = "checked_clean"
	Unknown       Status = "unknown"
	NotApplicable Status = "not_applicable"
)

// NormalizeStatus preserves uncertainty for missing and future values.
func NormalizeStatus(s Status) Status {
	switch s {
	case Violation, CheckedClean, Unknown, NotApplicable:
		return s
	default:
		return Unknown
	}
}

type Reason string

const (
	ReasonNone               Reason = "none"
	MissingAnalysis          Reason = "missing-analysis"
	UnsupportedAdapter       Reason = "unsupported-adapter"
	UnsupportedVersion       Reason = "unsupported-version"
	UnsupportedTestKind      Reason = "unsupported-test-kind"
	MissingInput             Reason = "missing-input"
	ParseFailure             Reason = "parse-failure"
	ExternalHelperUnresolved Reason = "external-helper-unresolved"
	ResolutionLimit          Reason = "resolution-limit"
	BuildContextUnavailable  Reason = "build-context-unavailable"
	NotExecuted              Reason = "not-executed"
	Skipped                  Reason = "skipped"
	OwnerUnavailable         Reason = "owner-unavailable"
	StaleEvidence            Reason = "stale-evidence"
	UnrecognizedValue        Reason = "unrecognized-value"
)

func Reasons() []Reason {
	return []Reason{ReasonNone, MissingAnalysis, UnsupportedAdapter, UnsupportedVersion,
		UnsupportedTestKind, MissingInput, ParseFailure, ExternalHelperUnresolved,
		ResolutionLimit, BuildContextUnavailable, NotExecuted, Skipped,
		OwnerUnavailable, StaleEvidence, UnrecognizedValue}
}

func NormalizeReason(r Reason) Reason {
	if r == "" {
		return MissingAnalysis
	}
	for _, known := range Reasons() {
		if r == known {
			return r
		}
	}
	return UnrecognizedValue
}

// ReasonGuidance describes how to obtain interpretable evidence, never how to
// turn an unknown into a pass without performing the missing assessment.
func ReasonGuidance(r Reason) string {
	switch r {
	case ReasonNone, "":
		return ""
	case NotExecuted:
		return "Request execution through Unit Health or Test Genie; static analysis does not supply runtime evidence."
	case Skipped:
		return "Inspect the native skip or expected-failure declaration and its rationale; run the test when applicable. A skip is not a pass."
	case UnsupportedAdapter, UnsupportedVersion, UnsupportedTestKind:
		return "Check the rule support profile and installed tool version. Use a supported profile or calibrate the missing profile before relying on this check."
	case OwnerUnavailable:
		return "Check the owning provider's health and retry the assessment after it is available; do not infer a clean result from missing evidence."
	case StaleEvidence:
		return "Obtain a fresh assessment for the current source and configuration; retain the original run identity for historical evidence."
	case ParseFailure:
		return "Inspect the retained native diagnostics and source location; repair malformed input or report an unsupported parser case, then reassess."
	case ExternalHelperUnresolved, ResolutionLimit:
		return "Inspect the referenced helper and resolution limits. Supply supported helper evidence or extend the owner adapter with calibrated fixtures."
	case BuildContextUnavailable:
		return "Supply the matching module, build tags and toolchain context before interpreting source observations."
	default:
		return "Inspect the report schema, source location and required evidence. Obtain a compatible complete report; unknown evidence is not a clean assessment."
	}
}

type EvidenceKind string

const (
	Static   EvidenceKind = "static"
	Runtime  EvidenceKind = "runtime"
	Registry EvidenceKind = "registry"
)

type Enforcement string

const (
	Advisory Enforcement = "advisory"
	Blocking Enforcement = "blocking"
)

type Severity string

const (
	Info    Severity = "info"
	Warning Severity = "warning"
	Error   Severity = "error"
)

type Target struct {
	// Empty retains historical per-test scope. File checks never invent test IDs.
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace"`
	File      string `json:"file"`
	TestID    string `json:"testId"`
}

type Location struct {
	Line      int `json:"line"`
	Column    int `json:"column"`
	EndLine   int `json:"endLine,omitempty"`
	EndColumn int `json:"endColumn,omitempty"`
}

type Result struct {
	RuntimeObservation *RuntimeObservation `json:"runtimeObservation,omitempty"`
	Diagnostics        []Diagnostic        `json:"diagnostics,omitempty"`
	RuleID             string              `json:"ruleId"`
	RuleVersion        string              `json:"ruleVersion"`
	Target             Target              `json:"target"`
	TestKind           string              `json:"testKind"`
	SupportProfile     string              `json:"supportProfile"`
	Status             Status              `json:"status"`
	Reason             Reason              `json:"reasonCode"`
	EvidenceKind       EvidenceKind        `json:"evidenceKind"`
	Severity           Severity            `json:"severity"`
	Enforcement        Enforcement         `json:"enforcement"`
	Location           Location            `json:"location"`
	EvidenceRefs       []string            `json:"evidenceRefs,omitempty"`
	Limitations        []string            `json:"limitations,omitempty"`
}

// RuntimeObservation preserves the native final result separately from assertion
// quality. RetryCount is aggregate metadata, never a fabricated attempt history.
type RuntimeObservation struct {
	RunID        string  `json:"runId"`
	State        string  `json:"state"`
	Seed         *string `json:"seed,omitempty"`
	RetryCount   *int    `json:"retryCount,omitempty"`
	RetryOrdinal *int    `json:"retryOrdinal,omitempty"`
}

// Diagnostic retains native detail without multiplying a file's denominator.
type Diagnostic struct {
	RuleID    string   `json:"nativeRuleId"`
	MessageID string   `json:"messageId"`
	Message   string   `json:"message"`
	Severity  int      `json:"nativeSeverity"`
	Location  Location `json:"location"`
}

// Normalized returns a copy; it never rewrites persisted historical evidence.
func (r Result) Normalized() Result {
	if r.RuntimeObservation != nil {
		observation := *r.RuntimeObservation
		r.RuntimeObservation = &observation
		switch observation.State {
		case "pass", "fail", "skip", "todo", "run", "only":
		default:
			observation.State = "unknown"
		}
		if observation.RetryCount != nil {
			count := *observation.RetryCount
			observation.RetryCount = &count
			if count < 0 {
				observation.RetryCount = nil
			}
		}
		if observation.RetryOrdinal != nil {
			ordinal := *observation.RetryOrdinal
			observation.RetryOrdinal = &ordinal
			if ordinal < 0 {
				observation.RetryOrdinal = nil
			}
		}
		if observation.Seed != nil {
			seed := *observation.Seed
			observation.Seed = &seed
		}
		if r.EvidenceKind != Runtime {
			r.RuntimeObservation = nil
		}
	}
	originalStatus := r.Status
	r.Status = NormalizeStatus(r.Status)
	r.Reason = NormalizeReason(r.Reason)
	if originalStatus == "" {
		r.Reason = MissingAnalysis
	} else if r.Status != originalStatus {
		r.Reason = UnrecognizedValue
	}
	if r.Status == Unknown && r.Reason == ReasonNone {
		r.Reason = MissingInput
	}
	if (r.Status == CheckedClean || r.Status == Violation) && (r.Reason == MissingAnalysis || r.Reason == UnrecognizedValue) {
		r.Status = Unknown
	}
	return r
}

func (r *Result) UnmarshalJSON(data []byte) error {
	type wire Result
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = Result(decoded).Normalized()
	return nil
}

// Coverage counts rule/test opportunities, not assertions or passing tests.
// Discovered = Assessed + Unknown + NotApplicable; Assessed includes violations.
type Coverage struct {
	Scope          string `json:"scope,omitempty"`
	RuleID         string `json:"ruleId"`
	SupportProfile string `json:"supportProfile"`
	Discovered     int    `json:"discovered"`
	Assessed       int    `json:"assessed"`
	Unknown        int    `json:"unknown"`
	NotApplicable  int    `json:"notApplicable"`
}

type Report struct {
	SchemaVersion  string     `json:"schemaVersion"`
	CatalogVersion string     `json:"catalogVersion"`
	Results        []Result   `json:"results"`
	Coverage       []Coverage `json:"coverage"`
	TotalResults   int        `json:"totalResults"`
	Truncated      bool       `json:"truncated"`
	DetailsRef     string     `json:"detailsRef,omitempty"`
	// Non-empty when the projection cannot be interpreted as current analysis.
	UnavailableReason Reason `json:"unavailableReason,omitempty"`
	// Missing collectors do not invalidate independently observed rows. Counts
	// cover reported rows only and do not establish complete test discovery.
	IncompleteReasons []Reason `json:"incompleteReasons,omitempty"`
}

func (r *Report) MarkIncomplete(reason Reason) {
	if reason == "" || reason == ReasonNone {
		return
	}
	if len(r.Results) == 0 {
		if r.UnavailableReason == "" || r.UnavailableReason == ReasonNone {
			r.UnavailableReason = NormalizeReason(reason)
		}
	}
	reason = NormalizeReason(reason)
	for _, existing := range r.IncompleteReasons {
		if existing == reason {
			return
		}
	}
	r.IncompleteReasons = append(r.IncompleteReasons, reason)
}

// Normalized preserves historical identity while invalidating claims whose
// schema or denominators this reader cannot interpret. It never edits storage.
func (r Report) Normalized() Report {
	r.Results = append([]Result(nil), r.Results...)
	r.Coverage = append([]Coverage(nil), r.Coverage...)
	r.IncompleteReasons = append([]Reason(nil), r.IncompleteReasons...)
	for i, reason := range r.IncompleteReasons {
		r.IncompleteReasons[i] = NormalizeReason(reason)
	}
	switch {
	case r.SchemaVersion == "":
		r.UnavailableReason = MissingAnalysis
	case r.SchemaVersion != SchemaVersion:
		r.UnavailableReason = UnsupportedVersion
	case r.UnavailableReason != "" && r.UnavailableReason != ReasonNone:
		r.UnavailableReason = NormalizeReason(r.UnavailableReason)
	case r.Validate() != nil:
		r.UnavailableReason = MissingInput
	}
	for i, result := range r.Results {
		result = result.Normalized()
		result.EvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		result.Limitations = append([]string(nil), result.Limitations...)
		if r.UnavailableReason != "" && r.UnavailableReason != ReasonNone {
			result.Status, result.Reason = Unknown, r.UnavailableReason
			result.Enforcement = Advisory
		}
		r.Results[i] = result
	}
	if r.UnavailableReason != "" && r.UnavailableReason != ReasonNone {
		// Do not reinterpret unknown-schema or inconsistent counts as coverage.
		r.Coverage = nil
	}
	return r
}

func (r *Report) UnmarshalJSON(data []byte) error {
	type wire Report
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = Report(decoded).Normalized()
	return nil
}

// Validate checks projection integrity, not behavioral adequacy.
func (r Report) Validate() error {
	if r.SchemaVersion != SchemaVersion || r.CatalogVersion == "" {
		return fmt.Errorf("unsupported or missing report version")
	}
	if r.TotalResults < len(r.Results) || r.TotalResults < 0 {
		return fmt.Errorf("invalid total result count")
	}
	if r.Truncated != (r.TotalResults > len(r.Results)) || (r.Truncated && r.DetailsRef == "") {
		return fmt.Errorf("invalid result bounds or missing details reference")
	}
	total := 0
	seen := map[[3]string]bool{}
	for _, c := range r.Coverage {
		key := [3]string{c.RuleID, c.SupportProfile, c.Scope}
		if c.Scope != "" && c.Scope != "file" {
			return fmt.Errorf("unknown coverage scope")
		}
		if c.RuleID == "" || c.SupportProfile == "" || seen[key] {
			return fmt.Errorf("missing or duplicate coverage identity")
		}
		seen[key] = true
		if c.Discovered < 0 || c.Assessed < 0 || c.Unknown < 0 || c.NotApplicable < 0 || c.Assessed > c.Discovered || c.Unknown > c.Discovered-c.Assessed || c.NotApplicable != c.Discovered-c.Assessed-c.Unknown {
			return fmt.Errorf("invalid coverage partition")
		}
		if c.Discovered > r.TotalResults-total {
			return fmt.Errorf("coverage exceeds total results")
		}
		total += c.Discovered
	}
	if total != r.TotalResults {
		return fmt.Errorf("coverage denominator differs from total results")
	}
	// Visible rows are lower bounds for truncated projections and exact counts
	// for complete ones. A future/unknown row cannot count as assessed.
	visible, err := BuildReport(r.CatalogVersion, r.Results, max(1, len(r.Results)), "")
	if err != nil {
		return err
	}
	for _, v := range visible.Coverage {
		found := false
		for _, c := range r.Coverage {
			if c.RuleID != v.RuleID || c.SupportProfile != v.SupportProfile || c.Scope != v.Scope {
				continue
			}
			found = true
			if v.Discovered > c.Discovered || v.Assessed > c.Assessed || v.Unknown > c.Unknown || v.NotApplicable > c.NotApplicable {
				return fmt.Errorf("visible rows exceed claimed status counts")
			}
			if !r.Truncated && v != c {
				return fmt.Errorf("complete rows disagree with coverage")
			}
		}
		if !found {
			return fmt.Errorf("visible result has no coverage denominator")
		}
	}
	return nil
}

// BuildReport calculates denominators before bounding output. The caller owns
// storage/retrieval of the complete details when a payload must be truncated.
func BuildReport(catalogVersion string, results []Result, limit int, detailsRef string) (Report, error) {
	if catalogVersion == "" || limit < 1 {
		return Report{}, fmt.Errorf("catalog version and positive result limit required")
	}
	if len(results) > limit && detailsRef == "" {
		return Report{}, fmt.Errorf("truncated results require a details reference")
	}
	out := Report{SchemaVersion: SchemaVersion, CatalogVersion: catalogVersion,
		TotalResults: len(results), Truncated: len(results) > limit, DetailsRef: detailsRef,
		Results: []Result{}, Coverage: []Coverage{}}
	type coverageKey struct{ rule, profile, scope string }
	type resultKey struct {
		rule, version, profile string
		target                 Target
	}
	counts := map[coverageKey]Coverage{}
	seen := map[resultKey]bool{}
	versions := map[coverageKey]string{}
	for _, raw := range results {
		r := raw.Normalized()
		validScope := (r.Target.Scope == "" && r.Target.TestID != "") || (r.Target.Scope == "file" && r.Target.TestID == "")
		if r.RuleID == "" || r.RuleVersion == "" || r.SupportProfile == "" || !validScope || r.Target.Workspace == "" || r.Target.File == "" {
			return Report{}, fmt.Errorf("rule/version, support profile and complete test identity required")
		}
		key := resultKey{r.RuleID, r.RuleVersion, r.SupportProfile, r.Target}
		if seen[key] {
			return Report{}, fmt.Errorf("duplicate quality result for %s/%s", r.RuleID, r.Target.TestID)
		}
		seen[key] = true
		ck := coverageKey{r.RuleID, r.SupportProfile, r.Target.Scope}
		if version := versions[ck]; version != "" && version != r.RuleVersion {
			return Report{}, fmt.Errorf("mixed rule versions in one coverage denominator")
		}
		versions[ck] = r.RuleVersion
		c := counts[ck]
		c.RuleID, c.SupportProfile = r.RuleID, r.SupportProfile
		c.Scope = r.Target.Scope
		c.Discovered++
		switch r.Status {
		case Violation, CheckedClean:
			c.Assessed++
		case Unknown:
			c.Unknown++
		case NotApplicable:
			c.NotApplicable++
		}
		counts[ck] = c
		if len(out.Results) < limit {
			r.Diagnostics = append([]Diagnostic(nil), r.Diagnostics...)
			r.EvidenceRefs = append([]string(nil), r.EvidenceRefs...)
			r.Limitations = append([]string(nil), r.Limitations...)
			out.Results = append(out.Results, r)
		}
	}
	for _, c := range counts {
		out.Coverage = append(out.Coverage, c)
	}
	sort.Slice(out.Coverage, func(i, j int) bool {
		a, b := out.Coverage[i], out.Coverage[j]
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		if a.SupportProfile != b.SupportProfile {
			return a.SupportProfile < b.SupportProfile
		}
		return a.Scope < b.Scope
	})
	return out, nil
}
