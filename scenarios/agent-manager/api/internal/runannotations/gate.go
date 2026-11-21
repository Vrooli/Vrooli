package runannotations

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// This file owns the accuracy gate. A label is proven for one observation only
// when a hold-out measurement exists for that observation's exact
// (definition hash, model, resource role) and clears the label's targets. A
// proof for one model never carries to another, and a proof for an earlier
// definition never carries to a changed one.

// Gate splits recorded in a measurement.
const (
	// SplitHoldout is a measurement on gold rows disjoint from every row used
	// to fit the calibration. Only a hold-out measurement can prove a label.
	SplitHoldout = "holdout"
	// SplitTuning is a measurement on the rows the calibration was fitted on.
	// It is evidence of fit, never proof.
	SplitTuning = "tuning"
	// SplitLegacy marks a measurement written before gates were per model. Its
	// model is unknown, so it is shown but can never prove anything.
	SplitLegacy = "legacy"
)

// RuleModel is the model name of deterministic, signal and composite
// observations: no inference ran, so the rule itself is what a measurement
// proves.
const RuleModel = "rule"

// Gate holds the quality targets a label must clear and the measurements that
// say whether it did.
type Gate struct {
	Precision    float64       `json:"precision"`
	Recall       float64       `json:"recall"`
	MinGold      int           `json:"min_gold"`
	Measurements []Measurement `json:"measurements,omitempty"`
	// Measured is the pre-per-model single measurement. It is read once at pack
	// load and migrated into Measurements; it is never written back.
	Measured *legacyMeasurement `json:"measured,omitempty"`
}

// legacyMeasurement is the pre-C1 gate.measured block.
type legacyMeasurement struct {
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	Gold           int     `json:"gold"`
	Threshold      float64 `json:"threshold,omitempty"`
	DefinitionHash string  `json:"definition_hash,omitempty"`
}

// Measurement is one measured result for one (definition hash, model, resource
// role). Temperature and Threshold are the calibration the measurement was
// taken at; they are applied at read time to that model's observations only.
type Measurement struct {
	DefinitionHash string  `json:"definition_hash"`
	Model          string  `json:"model"`
	ResourceRole   string  `json:"resource_role"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	Gold           int     `json:"gold"`
	Positives      int     `json:"positives"`
	Threshold      float64 `json:"threshold"`
	Temperature    float64 `json:"temperature"`
	ECE            float64 `json:"ece"`
	Brier          float64 `json:"brier"`
	Split          string  `json:"split"`
	GoldFile       string  `json:"gold_file,omitempty"`
	MeasuredAt     string  `json:"measured_at,omitempty"`
}

// Gate states, ordered from weakest to strongest evidence.
const (
	GateUnmeasured       = "unmeasured"
	GateLegacy           = "legacy_unattributed"
	GateTuningOnly       = "tuning_only"
	GateInsufficientGold = "insufficient_gold"
	GateUnproven         = "unproven"
	GateProven           = "proven"
	// GateDerived is a label-level state of a composite over other labels: it
	// has no gold of its own, and each observation of it is proven exactly when
	// every observation it read is proven for its own model.
	GateDerived = "derived"
)

var gateRank = map[string]int{
	GateUnmeasured: 0, GateLegacy: 1, GateTuningOnly: 2, GateInsufficientGold: 3, GateUnproven: 4, GateProven: 5,
}

// GateStatus is the gate object consumers read: the measurement that decided
// the state, the label's targets, and why the state is what it is.
type GateStatus struct {
	State           string  `json:"state"`
	Precision       float64 `json:"precision"`
	Recall          float64 `json:"recall"`
	Gold            int     `json:"gold"`
	MinGold         int     `json:"min_gold"`
	Reason          string  `json:"reason"`
	TargetPrecision float64 `json:"target_precision"`
	TargetRecall    float64 `json:"target_recall"`
	// Model and ResourceRole name who the status is for. They are empty for a
	// label-level summary that spans several models.
	Model        string  `json:"model,omitempty"`
	ResourceRole string  `json:"resource_role,omitempty"`
	Split        string  `json:"split,omitempty"`
	Positives    int     `json:"positives,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	ECE          float64 `json:"ece,omitempty"`
	Brier        float64 `json:"brier,omitempty"`
	GoldFile     string  `json:"gold_file,omitempty"`
	MeasuredAt   string  `json:"measured_at,omitempty"`
	// Inputs is what a derived status was derived from: the proof key of each
	// observation a composite read and that key's own state, or, on a
	// label-level status, the labels it reads.
	Inputs []InputStatus `json:"inputs,omitempty"`
}

// InputKey names one observation a composite read by what proves it: the
// label, its definition hash, and the model and resource role that answered.
type InputKey struct {
	Label          string `json:"label"`
	DefinitionHash string `json:"definition_hash,omitempty"`
	Model          string `json:"model,omitempty"`
	ResourceRole   string `json:"resource_role,omitempty"`
}

// InputStatus is an input's key with the gate state of that key now.
type InputStatus struct {
	InputKey
	State  string `json:"state"`
	Proven bool   `json:"proven"`
}

// isRuleMethod reports whether an observation came from a rule, a signal or a
// composite: no inference ran, so no model is what a measurement proves.
func isRuleMethod(method string) bool {
	switch strings.ToLower(method) {
	case string(MethodDeterministic), string(MethodSignal), string(MethodComposite):
		return true
	}
	return false
}

// ProofKey is what an observation was produced by, for gate purposes. A rule
// observation (deterministic, signal, composite) is keyed RuleModel with no
// resource role; a decision observation is keyed by the answering model and
// the resource role that served it.
func ProofKey(annotation Annotation) (model, resourceRole string) {
	if isRuleMethod(annotation.Method) {
		return RuleModel, ""
	}
	return annotation.Model, annotation.ModelVersion
}

// StatusFor is the gate status of one observation, the single place a proof is
// decided. An observation of a rule or a model is proven by its label's
// measurement for its own key. An observation of a composite over labels is
// derived: proven only when every observation it read is proven for its own
// key, and unproven when any input is not, whatever the others say.
func (cp *CompiledPack) StatusFor(compiled *compiledLabel, row Annotation) GateStatus {
	gate := compiled.label.Gate
	if !compiled.derived() {
		model, role := ProofKey(row)
		return gate.Status(row.DefinitionHash, model, role)
	}
	status := GateStatus{
		State: GateUnproven, MinGold: gate.MinGold, TargetPrecision: gate.Precision, TargetRecall: gate.Recall,
		Model: RuleModel,
	}
	if len(row.Value.Inputs) == 0 {
		status.Reason = "the composite read no label observation, so there is nothing to derive a proof from"
		return status
	}
	status.State = GateProven
	for _, key := range row.Value.Inputs {
		input := InputStatus{InputKey: key, State: GateUnmeasured}
		if label, ok := cp.labels[key.Label]; ok {
			input.State = label.label.Gate.Status(key.DefinitionHash, key.Model, key.ResourceRole).State
		}
		input.Proven = input.State == GateProven
		status.Inputs = append(status.Inputs, input)
		if !input.Proven && status.State == GateProven {
			status.State = GateUnproven
			status.Reason = fmt.Sprintf("input %s is %s", describeInput(key), input.State)
		}
	}
	return status
}

// LabelStatusFor is the gate status of a label across the models it was
// observed with: the strongest key for a label with gold, and the derived state
// for a composite over labels.
func (cp *CompiledPack) LabelStatusFor(compiled *compiledLabel, keys []ModelKey) GateStatus {
	if !compiled.derived() {
		return compiled.label.Gate.Best(compiled.hash, keys)
	}
	gate := compiled.label.Gate
	status := GateStatus{
		State: GateDerived, MinGold: gate.MinGold, TargetPrecision: gate.Precision, TargetRecall: gate.Recall,
		Model:  RuleModel,
		Reason: "derived: an observation is proven when every observation it read is proven for its own model",
	}
	for _, id := range append(append([]string(nil), compiled.refs.Labels...), compiled.refs.Messages...) {
		input := InputStatus{InputKey: InputKey{Label: id}, State: GateUnmeasured}
		if label, ok := cp.labels[id]; ok {
			input.State = cp.LabelStatusFor(label, nil).State
		}
		input.Proven = input.State == GateProven
		status.Inputs = append(status.Inputs, input)
	}
	return status
}

// matches reports whether a measurement belongs to exactly this key.
func (m Measurement) matches(hash, model, role string) bool {
	return m.DefinitionHash == hash && m.Model == model && m.ResourceRole == role
}

// forKey returns the strongest measurement for one key: a hold-out beats a
// tuning measurement, and the newest measurement wins within a split.
func (g Gate) forKey(hash, model, role string) (Measurement, bool) {
	if model == "" {
		return Measurement{}, false
	}
	var best Measurement
	found := false
	for _, candidate := range g.Measurements {
		if !candidate.matches(hash, model, role) || candidate.Split == SplitLegacy {
			continue
		}
		if !found || splitRank(candidate.Split) > splitRank(best.Split) ||
			(candidate.Split == best.Split && candidate.MeasuredAt > best.MeasuredAt) {
			best, found = candidate, true
		}
	}
	return best, found
}

func splitRank(split string) int {
	switch split {
	case SplitHoldout:
		return 2
	case SplitTuning:
		return 1
	default:
		return 0
	}
}

// Status is the gate status of one (definition hash, model, resource role).
func (g Gate) Status(hash, model, role string) GateStatus {
	status := GateStatus{
		State: GateUnmeasured, MinGold: g.MinGold, TargetPrecision: g.Precision, TargetRecall: g.Recall,
		Model: model, ResourceRole: role,
	}
	measurement, ok := g.forKey(hash, model, role)
	if !ok {
		status.Reason = "no measurement for this model and definition"
		switch {
		case model == "":
			status.Reason = "the observation names no model, so no measurement can prove it"
		case g.hasLegacy(hash):
			status.State = GateLegacy
			status.Reason = "measured before gates were per model; the model is unknown, so it proves nothing. Re-run the benchmark for this model"
		case g.hasOtherModels(hash):
			status.Reason = "measured for other models or resource roles only; a proof never carries to another"
		case g.hasOtherHashes(hash):
			status.Reason = "measurements belong to an earlier definition revision; re-run the benchmark"
		}
		return status
	}
	status.Precision, status.Recall, status.Gold = measurement.Precision, measurement.Recall, measurement.Gold
	status.Split, status.Positives = measurement.Split, measurement.Positives
	status.Threshold, status.Temperature = measurement.Threshold, measurement.Temperature
	status.ECE, status.Brier = measurement.ECE, measurement.Brier
	status.GoldFile, status.MeasuredAt = measurement.GoldFile, measurement.MeasuredAt
	switch {
	case measurement.Split != SplitHoldout:
		status.State = GateTuningOnly
		status.Reason = "measured on the rows the calibration was fitted on; run the benchmark on a hold-out file with --fit-from"
	case measurement.Gold < g.MinGold:
		status.State = GateInsufficientGold
		status.Reason = fmt.Sprintf("hold-out has %d gold row(s); the gate needs %d", measurement.Gold, g.MinGold)
	case measurement.Precision < g.Precision || measurement.Recall < g.Recall:
		status.State = GateUnproven
		status.Reason = "hold-out precision or recall is below target"
	default:
		status.State = GateProven
	}
	return status
}

// Proven reports whether the key's hold-out measurement clears every target.
func (g Gate) Proven(hash, model, role string) bool {
	return g.Status(hash, model, role).State == GateProven
}

// Calibration returns the temperature and threshold to apply to one model's
// observations at read time. Zero threshold means none was measured.
func (g Gate) Calibration(hash, model, role string) (temperature, threshold float64) {
	measurement, ok := g.forKey(hash, model, role)
	if !ok {
		return 1, 0
	}
	temperature = measurement.Temperature
	if temperature <= 0 {
		temperature = 1
	}
	return temperature, measurement.Threshold
}

func (g Gate) hasLegacy(hash string) bool {
	for _, m := range g.Measurements {
		if m.Split == SplitLegacy && m.DefinitionHash == hash {
			return true
		}
	}
	return false
}

func (g Gate) hasOtherModels(hash string) bool {
	for _, m := range g.Measurements {
		if m.DefinitionHash == hash && m.Split != SplitLegacy {
			return true
		}
	}
	return false
}

func (g Gate) hasOtherHashes(hash string) bool {
	for _, m := range g.Measurements {
		if m.DefinitionHash != hash {
			return true
		}
	}
	return false
}

// Best summarizes a label across the models it was observed with. It returns
// the strongest per-key status, so one proven model makes the label provable
// without hiding the others (callers list per-model status separately). With no
// keys it summarizes the label's measurements for the current definition.
func (g Gate) Best(hash string, keys []ModelKey) GateStatus {
	if len(keys) == 0 {
		keys = g.measuredKeys(hash)
	}
	best := g.Status(hash, "", "")
	best.Reason = "no model has been measured for this definition"
	if g.hasLegacy(hash) {
		best.State = GateLegacy
		best.Reason = "measured before gates were per model; re-run the benchmark per model"
	} else if g.hasOtherHashes(hash) && !g.hasOtherModels(hash) {
		best.Reason = "measurements belong to an earlier definition revision; re-run the benchmark"
	}
	for _, key := range keys {
		status := g.Status(hash, key.Model, key.ResourceRole)
		if gateRank[status.State] > gateRank[best.State] ||
			(gateRank[status.State] == gateRank[best.State] && best.Model == "") {
			best = status
		}
	}
	return best
}

// ModelKey is one (model, resource role) a label has observations for.
type ModelKey struct {
	Model        string `json:"model"`
	ResourceRole string `json:"resource_role"`
}

func (g Gate) measuredKeys(hash string) []ModelKey {
	seen := map[ModelKey]bool{}
	var keys []ModelKey
	for _, m := range g.Measurements {
		key := ModelKey{Model: m.Model, ResourceRole: m.ResourceRole}
		if m.DefinitionHash != hash || m.Split == SplitLegacy || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Model != keys[j].Model {
			return keys[i].Model < keys[j].Model
		}
		return keys[i].ResourceRole < keys[j].ResourceRole
	})
	return keys
}

// Upsert records a measurement, replacing one with the same key and split.
func (g *Gate) Upsert(measurement Measurement) {
	for index, existing := range g.Measurements {
		if existing.matches(measurement.DefinitionHash, measurement.Model, measurement.ResourceRole) && existing.Split == measurement.Split {
			g.Measurements[index] = measurement
			return
		}
	}
	g.Measurements = append(g.Measurements, measurement)
}

// migrateLegacy folds the pre-per-model measured block into Measurements. The
// model is unknown, so the entry is shown but never proves.
func (g *Gate) migrateLegacy() {
	if g.Measured == nil {
		return
	}
	legacy := g.Measured
	g.Measured = nil
	g.Measurements = append(g.Measurements, Measurement{
		DefinitionHash: legacy.DefinitionHash, Precision: legacy.Precision, Recall: legacy.Recall,
		Gold: legacy.Gold, Threshold: legacy.Threshold, Temperature: 1, Split: SplitLegacy,
	})
}

// validate checks measurement values. Hash and model rules differ by split: a
// legacy entry has no model by definition.
func (g Gate) validate(label string, add func(string, ...any)) {
	for index, m := range g.Measurements {
		where := fmt.Sprintf("%s: gate measurement %d", label, index)
		if m.Precision < 0 || m.Precision > 1 || m.Recall < 0 || m.Recall > 1 || m.Gold < 0 || m.Positives < 0 || m.Positives > m.Gold {
			add("%s: values are out of range", where)
		}
		if m.Threshold < 0 || m.Threshold > 1 || m.ECE < 0 || m.ECE > 1 || m.Brier < 0 || m.Brier > 1 {
			add("%s: threshold, ece and brier must be in [0,1]", where)
		}
		if m.Temperature < 0 {
			add("%s: temperature must not be negative", where)
		}
		switch m.Split {
		case SplitHoldout, SplitTuning:
			if strings.TrimSpace(m.Model) == "" {
				add("%s: model is required (use %q for rules)", where, RuleModel)
			}
		case SplitLegacy:
		default:
			add("%s: split %q must be holdout or tuning", where, m.Split)
		}
		if len(m.DefinitionHash) != 64 {
			add("%s: definition_hash must be 64 hex characters", where)
		} else if _, err := hex.DecodeString(m.DefinitionHash); err != nil {
			add("%s: definition_hash is not hex", where)
		}
		if m.MeasuredAt != "" {
			if _, err := time.Parse(time.RFC3339, m.MeasuredAt); err != nil {
				add("%s: measured_at must be RFC3339", where)
			}
		}
	}
}
