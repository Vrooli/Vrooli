package operatingmode

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// The resolution ladder makes a phase's structured output robust to real,
// nondeterministic model output. It replaces the old fail-on-malformed path
// (validateParseStatus): instead of rejecting any output that is not a single
// clean envelope, it climbs four rungs to either recover the declared result or
// abstain honestly, never fabricating fields.
//
//   - L0 (true-final-message detection): agents often emit a "final" answer and
//     then a subagent appends more, so the chronologically-last message is not
//     the real result. L0 scans the last N messages (newest→older) and picks the
//     first that yields a contract-satisfying envelope.
//   - L1 (deterministic extraction): parse the declared envelope out of a
//     message (whole body or a fenced ```json block) and validate it against the
//     phase's declared output schema (type/enum/bounds/required).
//   - L2 (classifier fallback): when no message extracts cleanly, reconstruct the
//     declared scalar/enum fields from the raw text via the FieldClassifier,
//     abstaining per-field rather than guessing. Object fields (handoff,
//     backlog_sync) are never fabricated.
//   - L3 (contract validation): the chosen/reconstructed result is validated
//     against the declared contract; unsatisfied required fields → abstain.

// ResolutionOutcome classifies whether, and how, the ladder produced a
// contract-satisfying structured result from a phase's raw agent output.
type ResolutionOutcome string

const (
	// ResolutionResolved: the chronologically-last message parsed cleanly and
	// satisfied the declared contract — the common happy path, no recovery.
	ResolutionResolved ResolutionOutcome = "resolved"
	// ResolutionRecovered: a valid result was recovered from output the old
	// path would have rejected — an earlier message when the last was subagent
	// noise (L0), or an LLM reconstruction (L2).
	ResolutionRecovered ResolutionOutcome = "recovered"
	// ResolutionAbstained: no message and no classifier could produce a
	// contract-satisfying result. The ladder abstains rather than guessing.
	ResolutionAbstained ResolutionOutcome = "abstained"
	// ResolutionNotRequired: the phase does not require a structured result; any
	// best-effort parse is applied and the outcome is never a failure.
	ResolutionNotRequired ResolutionOutcome = "not_required"
)

// ResolutionLayer names the ladder rung that produced (or last touched) the
// result, for the round's durable resolution record.
type ResolutionLayer string

const (
	ResolutionLayerNone       ResolutionLayer = ""
	ResolutionLayerFinalMsg   ResolutionLayer = "L0_true_final_message"
	ResolutionLayerExtract    ResolutionLayer = "L1_deterministic_extraction"
	ResolutionLayerClassifier ResolutionLayer = "L2_classifier"
)

// ResolvedPhaseResult is the outcome of running the resolution ladder over a
// completed phase's agent output.
type ResolvedPhaseResult struct {
	Result             PhaseResult
	Outcome            ResolutionOutcome
	Layer              ResolutionLayer
	ChosenMessageIndex int // index into the candidate messages, -1 when none matched
	MessagesScanned    int
	Missing            []string // declared required fields still unresolved
	Violations         []string // declared contract violations found
	Notes              []string // human-facing diagnostics
	// Envelope is the raw structured-result envelope the winning message
	// decoded to (or the classifier reconstructed), as a generic map. Unlike
	// Result it preserves fields the typed PhaseResult does not model — the
	// classification-on-transition ladder reads emitted/inline routing fields
	// from here.
	Envelope map[string]any
}

// Resolved reports whether the ladder produced a usable structured result
// (clean or recovered), as opposed to an abstain. Not-required phases are
// treated as resolved (nothing was owed).
func (r ResolvedPhaseResult) Resolved() bool {
	return r.Outcome == ResolutionResolved || r.Outcome == ResolutionRecovered || r.Outcome == ResolutionNotRequired
}

// PhaseResolutionRecord is the durable, operator-facing summary of how a round's
// structured output was resolved. It is stored on the round payload so the CLI
// and UI can show which ladder rung resolved the round (or why it abstained)
// without re-running resolution.
type PhaseResolutionRecord struct {
	Outcome            ResolutionOutcome `json:"outcome"`
	Layer              ResolutionLayer   `json:"layer,omitempty"`
	ChosenMessageIndex int               `json:"chosen_message_index"`
	MessagesScanned    int               `json:"messages_scanned,omitempty"`
	Missing            []string          `json:"missing,omitempty"`
	Violations         []string          `json:"violations,omitempty"`
	Notes              []string          `json:"notes,omitempty"`
	// ClassifiedField / ClassifiedValue mark a record produced by a
	// transition-owned classification (classification-on-transition) rather
	// than a phase-output resolution: ClassifiedField names the routing field
	// the ladder derived at the edge, and ClassifiedValue carries the derived
	// value the route guards matched on. Both empty on phase-output records.
	ClassifiedField string `json:"classified_field,omitempty"`
	ClassifiedValue string `json:"classified_value,omitempty"`
}

// Record projects the ladder outcome into its durable payload record.
func (r ResolvedPhaseResult) Record() PhaseResolutionRecord {
	return PhaseResolutionRecord{
		Outcome:            r.Outcome,
		Layer:              r.Layer,
		ChosenMessageIndex: r.ChosenMessageIndex,
		MessagesScanned:    r.MessagesScanned,
		Missing:            append([]string(nil), r.Missing...),
		Violations:         append([]string(nil), r.Violations...),
		Notes:              append([]string(nil), r.Notes...),
	}
}

// AbstainReason renders a single-line, operator-facing explanation of an
// abstain for the round's error/status surfaces: the ladder resolved no
// contract-satisfying result and lists what was missing.
func (r ResolvedPhaseResult) AbstainReason() string {
	reason := "resolution abstained: no contract-satisfying structured result could be resolved from the agent output"
	if len(r.Missing) > 0 {
		reason += " (unresolved: " + strings.Join(r.Missing, ", ") + ")"
	}
	if len(r.Violations) > 0 {
		reason += " (violations: " + strings.Join(r.Violations, ", ") + ")"
	}
	return reason
}

// envelopeAttempt is one candidate message's parse result: whether a declared
// envelope was found, the typed result the runtime consumes, and the generic
// envelope map used for declarative field validation and guard-style lookups.
type envelopeAttempt struct {
	ok       bool
	result   PhaseResult
	envelope map[string]any
}

// resolvePhaseOutput runs the resolution ladder for a phase over the ordered
// agent messages (oldest→newest). classifier may be nil (L2 disabled). It never
// panics and never returns an error: a phase that cannot be resolved abstains.
func resolvePhaseOutput(ctx context.Context, phaseDef PhaseDefinition, messages []string, classifier FieldClassifier) ResolvedPhaseResult {
	declared := effectiveDeclaredOutput(phaseDef)
	messages = nonEmptyMessages(messages)

	// Phases that do not require a structured result never abstain: parse the
	// newest message best-effort (for any artifacts/handoff it carries) and pass.
	if !declared.RequiresStructuredResult {
		res := ResolvedPhaseResult{Outcome: ResolutionNotRequired, Layer: ResolutionLayerNone, ChosenMessageIndex: -1}
		if len(messages) > 0 {
			if att := extractDeclared(messages[len(messages)-1], declared); att.ok {
				res.Result = att.result
				res.Envelope = att.envelope
				res.ChosenMessageIndex = len(messages) - 1
			}
		}
		return res
	}

	if len(messages) == 0 {
		return ResolvedPhaseResult{
			Outcome:            ResolutionAbstained,
			ChosenMessageIndex: -1,
			Missing:            requiredOrStructured(declared),
			Notes:              []string{"agent produced no output"},
		}
	}

	// L0 + L1: scan the last N messages newest→older; the first that parses and
	// satisfies the declared contract wins. When that message is not the
	// chronologically-last, the last message was subagent noise → recovered.
	order := scanOrder(len(messages), scanWindow(declared))
	var bestPartial *envelopeAttempt
	var partialMissing, partialViolations []string
	for pos, idx := range order {
		att := extractDeclared(messages[idx], declared)
		if !att.ok {
			continue
		}
		missing, violations := evaluateContract(declared, att)
		if len(missing) == 0 && len(violations) == 0 {
			outcome := ResolutionResolved
			layer := ResolutionLayerExtract
			if idx != len(messages)-1 {
				outcome = ResolutionRecovered
				layer = ResolutionLayerFinalMsg
			}
			return ResolvedPhaseResult{
				Result:             att.result,
				Outcome:            outcome,
				Layer:              layer,
				ChosenMessageIndex: idx,
				MessagesScanned:    pos + 1,
				Envelope:           att.envelope,
			}
		}
		if bestPartial == nil {
			a := att
			bestPartial = &a
			partialMissing, partialViolations = missing, violations
		}
	}

	// L2: reconstruct declared scalar/enum fields from the newest raw message.
	if declared.Resolution.AllowClassifier && classifier != nil {
		raw := messages[len(messages)-1]
		base := PhaseResult{}
		if bestPartial != nil {
			base = bestPartial.result
		}
		reconstructed, notes := classifyDeclaredFields(ctx, classifier, declared, raw, base)
		envelope := resultEnvelopeMap(reconstructed)
		missing, violations := validateDeclaredOutput(declared, envelope)
		if declared.RequiresStructuredResult && len(requiredFieldNames(declared)) == 0 && !hasPhaseResultContent(reconstructed) {
			missing = append(missing, "structured result content")
		}
		if len(missing) == 0 && len(violations) == 0 {
			return ResolvedPhaseResult{
				Result:             reconstructed,
				Outcome:            ResolutionRecovered,
				Layer:              ResolutionLayerClassifier,
				ChosenMessageIndex: len(messages) - 1,
				Notes:              notes,
				Envelope:           envelope,
			}
		}
		return ResolvedPhaseResult{
			Result:             reconstructed,
			Outcome:            ResolutionAbstained,
			Layer:              ResolutionLayerClassifier,
			ChosenMessageIndex: len(messages) - 1,
			Missing:            missing,
			Violations:         violations,
			Notes:              append(notes, "classifier could not reconstruct all required fields"),
			Envelope:           envelope,
		}
	}

	// No clean message and no classifier: abstain with the richest diagnostics.
	res := ResolvedPhaseResult{
		Outcome:            ResolutionAbstained,
		ChosenMessageIndex: len(messages) - 1,
		Missing:            requiredOrStructured(declared),
	}
	if bestPartial != nil {
		res.Result = bestPartial.result
		res.Envelope = bestPartial.envelope
		res.Layer = ResolutionLayerExtract
		res.Missing = partialMissing
		res.Violations = partialViolations
		if len(res.Missing) == 0 && len(res.Violations) == 0 {
			res.Missing = requiredOrStructured(declared)
		}
	}
	return res
}

// evaluateContract validates one parsed envelope against the declared schema and
// folds in the emptiness rule: a phase that requires a structured result but
// declares no specific required fields still needs a non-empty result body
// (preserving the old PhaseResultParseEmpty behavior for investigate/plan-style
// phases).
func evaluateContract(declared *DeclaredOutput, att envelopeAttempt) (missing, violations []string) {
	missing, violations = validateDeclaredOutput(declared, att.envelope)
	if declared.RequiresStructuredResult && len(requiredFieldNames(declared)) == 0 && !hasPhaseResultContent(att.result) {
		missing = append(missing, "structured result content")
	}
	return missing, violations
}

// extractDeclared finds and decodes the declared envelope out of one message. It
// tries the whole trimmed body first, then any fenced ```json blocks, returning
// the first body that carries the declared envelope key and decodes into a valid
// PhaseResult (a malformed progress decision disqualifies a candidate).
func extractDeclared(message string, declared *DeclaredOutput) envelopeAttempt {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return envelopeAttempt{}
	}
	for _, body := range candidateBodies(trimmed) {
		var top map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &top); err != nil {
			continue
		}
		payload, ok := top[declared.EnvelopeKey]
		if !ok {
			continue
		}
		var result PhaseResult
		if err := json.Unmarshal(payload, &result); err != nil {
			continue
		}
		if result.Progress != nil {
			if err := result.Progress.Validate(); err != nil {
				continue
			}
		}
		var envelope map[string]any
		_ = json.Unmarshal(payload, &envelope)
		return envelopeAttempt{ok: true, result: result, envelope: envelope}
	}
	return envelopeAttempt{}
}

// candidateBodies returns the JSON bodies to try for one message: the whole
// trimmed message, then each fenced ```json block. The whole body is tried first
// so a bare-envelope message resolves without depending on fences.
func candidateBodies(trimmed string) []string {
	bodies := []string{trimmed}
	for _, match := range fencedJSONBlockRE.FindAllStringSubmatch(trimmed, -1) {
		if len(match) > 1 {
			if body := strings.TrimSpace(match[1]); body != "" {
				bodies = append(bodies, body)
			}
		}
	}
	return bodies
}

// classifyDeclaredFields asks the classifier to reconstruct each declared
// leaf scalar/enum field that is not already present on the base result, folding
// found values back into a copy of base. Object-typed fields (handoff,
// backlog_sync) are never reconstructed — they stay whatever the base parse
// recovered (usually nil), so a missing required object drives an honest abstain
// instead of a fabricated plan.
func classifyDeclaredFields(ctx context.Context, classifier FieldClassifier, declared *DeclaredOutput, raw string, base PhaseResult) (PhaseResult, []string) {
	result := base
	notes := make([]string, 0, 4)
	baseEnvelope := resultEnvelopeMap(base)
	lookup := NewMapFieldLookup(baseEnvelope)
	for _, leaf := range declaredScalarLeaves(declared.Fields, "") {
		if v, present := lookup.Lookup(leaf.path); present && v != nil {
			continue // already recovered by L1
		}
		res, err := classifier.ClassifyField(ctx, ClassifyFieldRequest{
			RawOutput:   raw,
			FieldPath:   leaf.path,
			FieldType:   leaf.field.Type,
			Description: leaf.field.Description,
			Enum:        enumStrings(leaf.field.Enum),
		})
		if err != nil {
			notes = append(notes, "classify "+leaf.path+": "+err.Error())
			continue
		}
		if !res.Found || strings.TrimSpace(res.Value) == "" {
			notes = append(notes, "classifier abstained on "+leaf.path)
			continue
		}
		if applyClassifiedField(&result, leaf.path, res.Value, leaf.field.Type) {
			notes = append(notes, "classifier resolved "+leaf.path+" = "+res.Value)
		}
	}
	return result, notes
}

// scalarLeaf pairs a declared scalar/enum field with its dotted path.
type scalarLeaf struct {
	path  string
	field OutputField
}

// declaredScalarLeaves returns the declared leaf fields that the classifier can
// reconstruct: string/number/integer/boolean/enum fields. Object and array
// fields are excluded — the classifier never fabricates structured payloads.
func declaredScalarLeaves(fields []OutputField, prefix string) []scalarLeaf {
	var leaves []scalarLeaf
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if isScalarType(field.Type) || len(field.Enum) > 0 {
			leaves = append(leaves, scalarLeaf{path: path, field: field})
		}
		leaves = append(leaves, declaredScalarLeaves(field.Fields, path)...)
	}
	return leaves
}

// applyClassifiedField folds a single classifier-recovered value into the result
// by its declared dotted path. Runtime-known fields still hydrate their typed
// slots, while any other declared scalar is preserved in ExtraFields so L3
// validation, payload storage, and operator surfaces do not silently drop data
// the mode declared.
func applyClassifiedField(result *PhaseResult, path, value, fieldType string) bool {
	switch path {
	case "verdict":
		result.Verdict = value
		return true
	case "replan_needed":
		result.ReplanNeeded = parseBool(value)
		return true
	case "progress.decision":
		decision := ProgressDecision(strings.TrimSpace(value))
		if decision.Validate() != nil {
			return false
		}
		if result.Progress == nil {
			result.Progress = &ProgressState{}
		}
		result.Progress.Decision = decision
		return true
	default:
		return setExtraScalarField(result, path, parseClassifiedScalar(value, fieldType))
	}
}

// resultEnvelopeMap round-trips a PhaseResult through JSON into the generic
// envelope map the declarative validator and guard lookups consume, so a
// classifier-reconstructed result validates through exactly the same path as a
// parsed one.
func resultEnvelopeMap(result PhaseResult) map[string]any {
	data, err := json.Marshal(result)
	if err != nil {
		return map[string]any{}
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		return map[string]any{}
	}
	deepMergeMap(envelope, result.ExtraFields)
	return envelope
}

func parseClassifiedScalar(value, fieldType string) any {
	switch strings.TrimSpace(fieldType) {
	case "boolean", "bool":
		return parseBool(value)
	case "integer":
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil {
			return float64(n)
		}
	case "number":
		n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			return n
		}
	}
	return value
}

func setExtraScalarField(result *PhaseResult, path string, value any) bool {
	segments := strings.Split(strings.TrimSpace(path), ".")
	if len(segments) == 0 || segments[0] == "" {
		return false
	}
	if result.ExtraFields == nil {
		result.ExtraFields = map[string]any{}
	}
	dst := result.ExtraFields
	for _, segment := range segments[:len(segments)-1] {
		if strings.TrimSpace(segment) == "" {
			return false
		}
		next, _ := dst[segment].(map[string]any)
		if next == nil {
			next = map[string]any{}
			dst[segment] = next
		}
		dst = next
	}
	leaf := strings.TrimSpace(segments[len(segments)-1])
	if leaf == "" {
		return false
	}
	dst[leaf] = value
	return true
}

// effectiveDeclaredOutput returns the phase's declared output schema, synthesising
// a default (from the phase's structured-result contract) when the phase declares
// none, so the ladder always has an envelope key and resolution policy to work
// with.
func effectiveDeclaredOutput(phaseDef PhaseDefinition) *DeclaredOutput {
	if phaseDef.DeclaredOutput != nil {
		d := *phaseDef.DeclaredOutput
		if strings.TrimSpace(d.EnvelopeKey) == "" {
			d.EnvelopeKey = resultEnvelopeKey
		}
		if d.Resolution.ScanLastNMessages == 0 && !d.Resolution.DetectTrueFinalMessage {
			d.Resolution = defaultResolutionPolicy()
		}
		return &d
	}
	return &DeclaredOutput{
		EnvelopeKey:              resultEnvelopeKey,
		RequiresStructuredResult: phaseDef.OutputContract.RequiresStructuredResult,
		Resolution:               defaultResolutionPolicy(),
	}
}

func defaultResolutionPolicy() ResolutionPolicy {
	return ResolutionPolicy{DetectTrueFinalMessage: true, ScanLastNMessages: 5, AllowClassifier: true}
}

// scanWindow is how many trailing messages L0 considers. A mode that opts out of
// true-final-message detection (or declares a non-positive window) collapses to
// examining only the last message.
func scanWindow(declared *DeclaredOutput) int {
	if !declared.Resolution.DetectTrueFinalMessage {
		return 1
	}
	if declared.Resolution.ScanLastNMessages <= 0 {
		return 1
	}
	return declared.Resolution.ScanLastNMessages
}

// scanOrder returns up to window indices into a length-n message slice, newest
// first, so L0 examines the most recent candidate before older ones.
func scanOrder(n, window int) []int {
	if window > n {
		window = n
	}
	order := make([]int, 0, window)
	for i := 0; i < window; i++ {
		order = append(order, n-1-i)
	}
	return order
}

// requiredOrStructured describes what an abstain is missing: the declared
// required field names, or a single "structured result" marker when the phase
// requires an envelope but declares no specific fields.
func requiredOrStructured(declared *DeclaredOutput) []string {
	names := requiredFieldNames(declared)
	if len(names) == 0 && declared.RequiresStructuredResult {
		return []string{"structured result"}
	}
	return names
}

func nonEmptyMessages(messages []string) []string {
	out := make([]string, 0, len(messages))
	for _, m := range messages {
		if strings.TrimSpace(m) != "" {
			out = append(out, m)
		}
	}
	return out
}

func isScalarType(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "string", "number", "integer", "boolean", "bool":
		return true
	default:
		return false
	}
}

func enumStrings(enum []any) []string {
	if len(enum) == 0 {
		return nil
	}
	out := make([]string, 0, len(enum))
	for _, v := range enum {
		out = append(out, renderGuardValue(v))
	}
	return out
}

func parseBool(value string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && b
}
