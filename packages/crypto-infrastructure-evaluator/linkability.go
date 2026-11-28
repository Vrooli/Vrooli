package evaluator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	chainforensics "github.com/vrooli/chain-forensics"
)

// Deployment layers are the CI2 inventory boundary. Every layer must be
// represented, including layers with no observer or control in the scenario.
var requiredLayers = []string{"payment", "naming", "storage", "compute", "gateway", "accounts", "recovery"}

type LayerInventory struct {
	Layer     string   `json:"layer"`
	Observers []string `json:"observers"`
	Controls  []string `json:"controls"`
}

// AttackInput models fields available to one observer during a bounded window.
// Values are synthetic fixture values or captured fields; they are never
// interpreted as proof of a human identity.
type AttackInput struct {
	ID             string            `json:"id"`
	Adversary      string            `json:"adversary"`
	Layer          string            `json:"layer"`
	Vantage        string            `json:"vantage"`
	VantageKind    string            `json:"vantage_kind"`
	TimeWindow     TimeInterval      `json:"time_window"`
	ObservedFields map[string]string `json:"observed_fields"`
}

// TimeInterval is a closed UTC interval. Both endpoints must be RFC3339 UTC
// timestamps; offsets other than Z/UTC are rejected to keep fixtures explicit.
type TimeInterval struct {
	StartUTC string `json:"start_utc"`
	EndUTC   string `json:"end_utc"`
}

type Deployment struct {
	ID         string           `json:"id"`
	Layers     []LayerInventory `json:"layers"`
	Attacks    []AttackInput    `json:"attacks"`
	EvidenceID string           `json:"evidence_id"`
}

type AttackResult struct {
	ID                          string   `json:"id"`
	Adversary                   string   `json:"adversary"`
	Layer                       string   `json:"layer"`
	Vantage                     string   `json:"vantage"`
	VantageKind                 string   `json:"vantage_kind"`
	TimeWindow                  string   `json:"time_window"`
	ObservedFields              []string `json:"observed_fields"`
	MatchedOccurrences          []string `json:"matched_occurrences"`
	UnverifiableOccurrenceCount int      `json:"unverifiable_occurrence_count"`
	EvidenceSources             []string `json:"evidence_sources"`
	Outcome                     string   `json:"outcome"`
	ResidualUncertainty         string   `json:"residual_uncertainty"`
}

type LinkabilityReport struct {
	Version             string         `json:"version"`
	DeploymentID        string         `json:"deployment_id"`
	CaseID              string         `json:"case_id"`
	InventoryComplete   bool           `json:"inventory_complete"`
	LayerCount          int            `json:"layer_count"`
	ObserverCount       int            `json:"observer_count"`
	ControlCount        int            `json:"control_count"`
	AttackCount         int            `json:"attack_count"`
	MatchedAttackCount  int            `json:"matched_attack_count"`
	CoverageScore       float64        `json:"inventory_coverage_score"`
	Attacks             []AttackResult `json:"attacks"`
	ResidualUncertainty []string       `json:"residual_uncertainty"`
}

// SeededSuiteReport keeps evidence inventory and deployment attack results
// together while retaining distinct meanings for the two report families.
type SeededSuiteReport struct {
	CaseEvidence Report              `json:"case_evidence"`
	Deployments  []LinkabilityReport `json:"deployments"`
}

// EvaluateSeededSuite evaluates all three seeded deployment profiles against
// one frozen, validated CF6 case. It is deterministic for the same inputs.
func EvaluateSeededSuite(c chainforensics.Case) (SeededSuiteReport, error) {
	evidence, err := Evaluate(c)
	if err != nil {
		return SeededSuiteReport{}, err
	}
	deployments, err := SeededDeployments(c)
	if err != nil {
		return SeededSuiteReport{}, err
	}
	suite := SeededSuiteReport{CaseEvidence: evidence}
	for _, deployment := range deployments {
		report, err := EvaluateDeployment(c, deployment)
		if err != nil {
			return SeededSuiteReport{}, err
		}
		suite.Deployments = append(suite.Deployments, report)
	}
	return suite, nil
}

// EvaluateDeployment validates the full per-layer inventory before running
// reproducible exact-field joins against the shared CF6 occurrence model.
// The coverage score measures inventory completeness only; it is not a
// decentralization, anonymity, or untraceability score.
func EvaluateDeployment(c chainforensics.Case, d Deployment) (LinkabilityReport, error) {
	if err := chainforensics.Validate(c); err != nil {
		return LinkabilityReport{}, fmt.Errorf("validate chain-forensics case: %w", err)
	}
	if d.ID == "" || d.EvidenceID == "" {
		return LinkabilityReport{}, fmt.Errorf("deployment id and evidence_id are required")
	}
	layers := make(map[string]LayerInventory, len(d.Layers))
	for _, layer := range d.Layers {
		if _, ok := layers[layer.Layer]; ok || layer.Layer == "" {
			return LinkabilityReport{}, fmt.Errorf("duplicate or empty deployment layer %q", layer.Layer)
		}
		if layer.Observers == nil || layer.Controls == nil {
			return LinkabilityReport{}, fmt.Errorf("deployment %q layer %q must explicitly inventory observers and controls (empty lists are allowed)", d.ID, layer.Layer)
		}
		layers[layer.Layer] = layer
	}
	for _, name := range requiredLayers {
		if _, ok := layers[name]; !ok {
			return LinkabilityReport{}, fmt.Errorf("deployment %q missing required layer inventory %q", d.ID, name)
		}
	}
	if len(layers) != len(requiredLayers) {
		return LinkabilityReport{}, fmt.Errorf("deployment %q contains unsupported layer inventory", d.ID)
	}

	r := LinkabilityReport{Version: "ci2-linkability/v2", DeploymentID: d.ID, CaseID: c.ID, InventoryComplete: true, LayerCount: len(layers), AttackCount: len(d.Attacks)}
	for _, layer := range d.Layers {
		r.ObserverCount += len(layer.Observers)
		r.ControlCount += len(layer.Controls)
	}
	r.CoverageScore = float64(len(layers)) / float64(len(requiredLayers))
	for _, attack := range d.Attacks {
		if attack.ID == "" || attack.Adversary == "" || attack.Layer == "" || attack.Vantage == "" || len(attack.ObservedFields) == 0 {
			return LinkabilityReport{}, fmt.Errorf("deployment %q has incomplete attack model", d.ID)
		}
		start, err := parseUTC(attack.TimeWindow.StartUTC)
		if err != nil {
			return LinkabilityReport{}, fmt.Errorf("attack %q invalid start_utc: %w", attack.ID, err)
		}
		end, err := parseUTC(attack.TimeWindow.EndUTC)
		if err != nil {
			return LinkabilityReport{}, fmt.Errorf("attack %q invalid end_utc: %w", attack.ID, err)
		}
		if end.Before(start) {
			return LinkabilityReport{}, fmt.Errorf("attack %q time window ends before it starts", attack.ID)
		}
		layer, ok := layers[attack.Layer]
		if !ok {
			return LinkabilityReport{}, fmt.Errorf("attack %q references undeclared layer %q", attack.ID, attack.Layer)
		}
		if !contains(layer.Observers, attack.Adversary) {
			return LinkabilityReport{}, fmt.Errorf("attack %q adversary %q is not declared as an observer in layer %q", attack.ID, attack.Adversary, attack.Layer)
		}
		switch attack.VantageKind {
		case "observer":
			if !contains(layer.Observers, attack.Vantage) {
				return LinkabilityReport{}, fmt.Errorf("attack %q vantage %q is not declared as an observer in layer %q", attack.ID, attack.Vantage, attack.Layer)
			}
		case "control":
			if !contains(layer.Controls, attack.Vantage) {
				return LinkabilityReport{}, fmt.Errorf("attack %q vantage %q is not declared as a control in layer %q", attack.ID, attack.Vantage, attack.Layer)
			}
		default:
			return LinkabilityReport{}, fmt.Errorf("attack %q vantage_kind must be observer or control", attack.ID)
		}
		for name, value := range attack.ObservedFields {
			if name != "transaction" && name != "address" && name != "asset" && name != "atomic_quantity" {
				return LinkabilityReport{}, fmt.Errorf("attack %q uses unsupported observed field %q", attack.ID, name)
			}
			if strings.TrimSpace(value) == "" {
				return LinkabilityReport{}, fmt.Errorf("attack %q has empty observed field %q", attack.ID, name)
			}
		}
		result := AttackResult{ID: attack.ID, Adversary: attack.Adversary, Layer: attack.Layer, Vantage: attack.Vantage, VantageKind: attack.VantageKind, TimeWindow: attack.TimeWindow.StartUTC + "/" + attack.TimeWindow.EndUTC, EvidenceSources: []string{d.EvidenceID}, ResidualUncertainty: "A field match links this deployment observation to recorded chain activity only; it does not establish person, intent, control, or legal responsibility."}
		for name := range attack.ObservedFields {
			result.ObservedFields = append(result.ObservedFields, name)
		}
		sort.Strings(result.ObservedFields)
		for _, occurrence := range c.Occurrences {
			observedAt, parseErr := parseUTC(occurrence.Timestamp)
			if parseErr != nil {
				// An occurrence without a valid UTC timestamp cannot be tested
				// against this bounded window. Keep it out of both positive and
				// negative results, and report the excluded evidence explicitly.
				result.UnverifiableOccurrenceCount++
				continue
			}
			if !observedAt.Before(start) && !observedAt.After(end) && fieldsMatch(attack.ObservedFields, occurrence) {
				result.MatchedOccurrences = append(result.MatchedOccurrences, occurrence.ID)
				result.EvidenceSources = append(result.EvidenceSources, occurrence.SourceID)
			}
		}
		sort.Strings(result.MatchedOccurrences)
		result.EvidenceSources = uniqueSorted(result.EvidenceSources)
		if len(result.MatchedOccurrences) > 0 {
			result.Outcome = "exact_field_link_found"
			r.MatchedAttackCount++
		} else {
			result.Outcome = "no_exact_field_link_found"
			result.ResidualUncertainty = "No exact match was found for the tested fields and evidence window; untested fields, sources, or time windows may still link activity."
		}
		if result.UnverifiableOccurrenceCount > 0 {
			result.ResidualUncertainty += fmt.Sprintf(" %d case occurrences with missing or invalid UTC timestamps were excluded from this bounded window and could not be classified.", result.UnverifiableOccurrenceCount)
		}
		r.Attacks = append(r.Attacks, result)
	}
	r.ResidualUncertainty = []string{
		"Inventory coverage is not a privacy score and declared observer/control lists are scenario inputs, not independently measured facts.",
		"Exact-field joins cover only the supplied case, attack fields, and time windows; absence of a match does not establish unlinkability.",
		"Chain occurrence links do not establish human identity, intent, custody, legal responsibility, or complete observer visibility.",
	}
	return r, nil
}

// SeededDeployments creates deterministic teaching fixtures with a deliberate
// leak progression: exact transaction/address fields, coarse asset/quantity,
// and a supported synthetic address field absent from the case. Callers must provide a validated
// case with at least one occurrence so fixture identities derive from evidence.
func SeededDeployments(c chainforensics.Case) ([]Deployment, error) {
	if err := chainforensics.Validate(c); err != nil {
		return nil, fmt.Errorf("validate seed case: %w", err)
	}
	if len(c.Occurrences) == 0 {
		return nil, fmt.Errorf("seed case must contain at least one occurrence")
	}
	var o *chainforensics.Occurrence
	for i := range c.Occurrences {
		if _, err := parseUTC(c.Occurrences[i].Timestamp); err == nil {
			o = &c.Occurrences[i]
			break
		}
	}
	if o == nil {
		return nil, fmt.Errorf("seed case must contain at least one occurrence with a valid UTC timestamp")
	}
	base := func(id string) Deployment {
		d := Deployment{ID: id, EvidenceID: "seeded-fixture/v1"}
		for _, name := range requiredLayers {
			d.Layers = append(d.Layers, LayerInventory{Layer: name, Observers: []string{}, Controls: []string{}})
		}
		return d
	}
	central := base("centralized-baseline")
	central.Layers[0].Observers = []string{"payment-processor", "payment API records"}
	central.Layers[0].Controls = []string{"account-freeze", "payment-record-retention"}
	central.Attacks = []AttackInput{{ID: "central-exact-join", Adversary: "payment-processor", Layer: "payment", Vantage: "payment API records", VantageKind: "observer", TimeWindow: intervalFor(o.Timestamp), ObservedFields: map[string]string{"transaction": o.Transaction, "address": firstNonEmpty(o.From, o.To)}}}
	partial := base("partial-deployment")
	partial.Layers[1].Observers = []string{"cross-provider observer"}
	partial.Layers[3].Observers = []string{"compute-provider"}
	partial.Layers[1].Observers = append(partial.Layers[1].Observers, "naming and compute logs")
	partial.Attacks = []AttackInput{{ID: "partial-coarse-join", Adversary: "cross-provider observer", Layer: "naming", Vantage: "naming and compute logs", VantageKind: "observer", TimeWindow: intervalFor(o.Timestamp), ObservedFields: map[string]string{"asset": o.Asset, "atomic_quantity": o.Atomic}}}
	full := base("full-deployment")
	full.Layers[6].Observers = []string{"recovery custodian", "recovery inventory"}
	full.Attacks = []AttackInput{{ID: "full-negative-control", Adversary: "recovery custodian", Layer: "recovery", Vantage: "recovery inventory", VantageKind: "observer", TimeWindow: intervalFor(o.Timestamp), ObservedFields: map[string]string{"address": "synthetic-address-with-no-case-match"}}}
	return []Deployment{central, partial, full}, nil
}

func fieldsMatch(fields map[string]string, o chainforensics.Occurrence) bool {
	matched := 0
	for field, value := range fields {
		if value == "" {
			return false
		}
		var candidate string
		switch field {
		case "transaction":
			candidate = o.Transaction
		case "address":
			if value == o.From || value == o.To {
				matched++
				continue
			}
			return false
		case "asset":
			candidate = o.Asset
		case "atomic_quantity":
			candidate = o.Atomic
		}
		if candidate != value {
			return false
		}
		matched++
	}
	return matched == len(fields)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func intervalFor(timestamp string) TimeInterval {
	t, err := parseUTC(timestamp)
	if err != nil {
		return TimeInterval{}
	}
	return TimeInterval{StartUTC: t.Add(-time.Second).Format(time.RFC3339), EndUTC: t.Add(time.Second).Format(time.RFC3339)}
}

func parseUTC(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	_, offset := t.Zone()
	if offset != 0 || !strings.HasSuffix(value, "Z") {
		return time.Time{}, fmt.Errorf("timestamp must be UTC with Z suffix")
	}
	return t.UTC(), nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
