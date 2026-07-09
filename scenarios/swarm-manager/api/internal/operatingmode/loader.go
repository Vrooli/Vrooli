package operatingmode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file is the data-driven engine's loader: it turns an operating-mode data
// document (modes/<id>/mode.json) into the typed in-memory Definition the
// runtime consumes, and discovers modes from a directory. It is the sole source
// of Definitions — there are no hardcoded Go mode builders. Branching is loaded
// into the generic Guard graph (PhaseGraph.Guards), which the runtime router,
// the simulation walker, and the UI transition rendering all evaluate; the
// static Transitions adjacency is derived from the guard targets for ordering
// and reachability.

// ModeFileName is the fixed filename of a mode's data document within its
// folder under modes/.
const ModeFileName = "mode.json"

// modeDocument is the on-disk shape of mode.json (schema kind operating-mode).
type modeDocument struct {
	Kind          string             `json:"kind"`
	SchemaVersion string             `json:"schema_version,omitempty"`
	ID            string             `json:"id"`
	Label         string             `json:"label"`
	Description   string             `json:"description"`
	BestFor       []string           `json:"best_for"`
	NotFor        []string           `json:"not_for"`
	Tradeoffs     []string           `json:"tradeoffs"`
	WhenInDoubt   string             `json:"when_in_doubt_pick_instead,omitempty"`
	Target        targetDoc          `json:"target"`
	RunStrategy   runStrategyDoc     `json:"run_strategy"`
	PhaseGraph    *phaseGraphDoc     `json:"phase_graph,omitempty"`
	Prompt        *promptPolicyDoc   `json:"prompt,omitempty"`
	Artifact      *artifactPolicyDoc `json:"artifact,omitempty"`
	Profile       *profilePolicyDoc  `json:"profile,omitempty"`
	BacklogSync   *backlogSyncDoc    `json:"backlog_sync,omitempty"`
	Metrics       *metricsDoc        `json:"metrics,omitempty"`
	Lock          *lockDoc           `json:"lock,omitempty"`
	UI            *uiDoc             `json:"ui,omitempty"`
}

// targetDoc is the on-disk shape of the mode's declared unit of work. The
// nested plan_ref block is initiative-adapter configuration (the bound
// plan-manager plan contract), deliberately distinct from the plan-ref target
// kind.
type targetDoc struct {
	Kind    string            `json:"kind"`
	PlanRef *planRefPolicyDoc `json:"plan_ref,omitempty"`
}

type runStrategyDoc struct {
	Kind string `json:"kind"`
}

type promptPolicyDoc struct {
	CatalogPrefix string `json:"catalog_prefix"`
}

type artifactPolicyDoc struct {
	Root      string `json:"root"`
	RoundRoot string `json:"round_root,omitempty"`
}

type planRefPolicyDoc struct {
	Required bool   `json:"required,omitempty"`
	Role     string `json:"role"`
}

type profilePolicyDoc struct {
	DefaultProfileKey string `json:"default_profile_key,omitempty"`
}

type backlogSyncDoc struct {
	Capabilities       []string `json:"capabilities,omitempty"`
	RequiresRunID      bool     `json:"requires_run_id,omitempty"`
	RequiresMembership bool     `json:"requires_membership,omitempty"`
	EventSource        string   `json:"event_source,omitempty"`
	ApplyMode          string   `json:"apply_mode,omitempty"`
}

type metricsDoc struct {
	EventSource      string   `json:"event_source,omitempty"`
	AcceptedVerdicts []string `json:"accepted_verdicts,omitempty"`
}

type lockDoc struct {
	InitiativeExclusive bool `json:"initiative_exclusive,omitempty"`
}

type uiDoc struct {
	WorkspaceTabID string `json:"workspace_tab_id,omitempty"`
}

type phaseGraphDoc struct {
	StartPhase string     `json:"start_phase"`
	Terminal   []string   `json:"terminal,omitempty"`
	Phases     []phaseDoc `json:"phases"`
}

type phaseDoc struct {
	ID               string             `json:"id"`
	Kind             string             `json:"kind"`
	ExecutedBy       string             `json:"executed_by,omitempty"`
	ActivityPurpose  string             `json:"activity_purpose"`
	LockPurpose      string             `json:"lock_purpose,omitempty"`
	AutoStartAfter   []string           `json:"auto_start_after,omitempty"`
	WritesRepo       bool               `json:"writes_repo,omitempty"`
	RequiresCriteria bool               `json:"requires_criteria,omitempty"`
	Reads            []string           `json:"reads"`
	ProfileKey       string             `json:"profile_key,omitempty"`
	Prompt           *phasePromptDoc    `json:"prompt,omitempty"`
	DeclaredOutput   *declaredOutputDoc `json:"declared_output,omitempty"`
	OutputArtifacts  []artifactDefDoc   `json:"output_artifacts,omitempty"`
	ResultBindings   []resultBindingDoc `json:"result_bindings,omitempty"`
	Transitions      []transitionDoc    `json:"transitions,omitempty"`
	Metrics          *phaseMetricsDoc   `json:"metrics,omitempty"`
}

type phasePromptDoc struct {
	Template string `json:"template,omitempty"`
	Suffix   string `json:"suffix,omitempty"`
	Title    string `json:"title,omitempty"`
	Trigger  string `json:"trigger,omitempty"`
	Purpose  string `json:"purpose,omitempty"`
}

type declaredOutputDoc struct {
	EnvelopeKey              string           `json:"envelope_key,omitempty"`
	RequiresStructuredResult *bool            `json:"requires_structured_result,omitempty"`
	Fields                   []outputFieldDoc `json:"fields,omitempty"`
	Resolution               *resolutionDoc   `json:"resolution,omitempty"`
}

type outputFieldDoc struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Required    bool             `json:"required,omitempty"`
	Enum        []any            `json:"enum,omitempty"`
	Minimum     *float64         `json:"minimum,omitempty"`
	Maximum     *float64         `json:"maximum,omitempty"`
	MinLength   *int             `json:"min_length,omitempty"`
	MaxLength   *int             `json:"max_length,omitempty"`
	Description string           `json:"description,omitempty"`
	Fields      []outputFieldDoc `json:"fields,omitempty"`
}

type resolutionDoc struct {
	DetectTrueFinalMessage *bool `json:"detect_true_final_message,omitempty"`
	ScanLastNMessages      *int  `json:"scan_last_n_messages,omitempty"`
	AllowClassifier        *bool `json:"allow_classifier,omitempty"`
}

type artifactDefDoc struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type resultBindingDoc struct {
	Kind     string         `json:"kind"`
	Artifact artifactDefDoc `json:"artifact"`
}

// transitionDoc is one on-disk edge out of a phase: either a guarded edge
// (`when`/`to`) or a classified edge (`classify`/`routes`) — the schema's
// Transition oneOf. A classified edge declares the classification contract for
// the routing field and maps each enum value to its target phase(s); the
// loader expands it into ordinary eq-guards in declared enum order.
type transitionDoc struct {
	When     *Guard                     `json:"when,omitempty"`
	To       []string                   `json:"to,omitempty"`
	Classify *classificationContractDoc `json:"classify,omitempty"`
	Routes   map[string][]string        `json:"routes,omitempty"`
}

type classificationContractDoc struct {
	Field       string   `json:"field"`
	Enum        []string `json:"enum"`
	From        string   `json:"from,omitempty"`
	Description string   `json:"description,omitempty"`
}

type phaseMetricsDoc struct {
	CountsReplanSample     bool `json:"counts_replan_sample,omitempty"`
	CountsAcceptanceSample bool `json:"counts_acceptance_sample,omitempty"`
}

// LoadModeDefinition parses and validates a raw mode.json document, returning
// the typed Definition. It runs JSON-Schema validation first (structural), then
// derives the runtime Definition. Semantic cross-reference validation (across
// the full mode set) is applied by LoadModesFromDir / ValidateLoadedModes.
func LoadModeDefinition(raw []byte) (Definition, error) {
	if err := ValidateDocumentBytes(raw); err != nil {
		return Definition{}, err
	}
	var doc modeDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Definition{}, fmt.Errorf("decode mode document: %w", err)
	}
	if doc.Kind != DocumentKindMode {
		return Definition{}, fmt.Errorf("expected document kind %q, got %q", DocumentKindMode, doc.Kind)
	}
	return doc.toDefinition()
}

func (doc modeDocument) toDefinition() (Definition, error) {
	def := Definition{
		Mode:                   Mode(doc.ID),
		Label:                  doc.Label,
		Description:            doc.Description,
		BestFor:                append([]string(nil), doc.BestFor...),
		NotFor:                 append([]string(nil), doc.NotFor...),
		Tradeoffs:              append([]string(nil), doc.Tradeoffs...),
		WhenInDoubtPickInstead: Mode(doc.WhenInDoubt),
		Target:                 doc.targetPolicy(),
		RunStrategy:            RunStrategyPolicy{Kind: RunStrategyKind(doc.RunStrategy.Kind)},
		UI:                     UIPolicy{WorkspaceTabID: uiTabID(doc.UI)},
	}
	if !IsValidTargetKind(def.Target.Kind) {
		return Definition{}, fmt.Errorf("mode %q declares unknown target kind %q (want one of %s|%s|%s)", doc.ID, doc.Target.Kind, TargetPlanManagerPlan, TargetPlanRef, TargetInitiative)
	}

	def.Profile = ProfilePolicy{DefaultProfileKey: profileDefaultKey(doc.Profile)}
	def.BacklogSync = doc.backlogSyncPolicy()
	def.Metrics = MetricsPolicy{EventSource: metricsEventSource(doc)}
	if doc.Metrics != nil {
		def.Metrics.AcceptedVerdicts = append([]string(nil), doc.Metrics.AcceptedVerdicts...)
	}

	if !def.RunsModeRounds() {
		return def, nil
	}

	if doc.PhaseGraph == nil {
		return Definition{}, fmt.Errorf("mode %q runs mode rounds but declares no phase_graph", doc.ID)
	}
	if doc.Prompt != nil {
		def.Prompt = PromptPolicy{CatalogPrefix: doc.Prompt.CatalogPrefix}
	}
	def.Artifact = doc.artifactPolicy()
	if doc.Lock != nil {
		def.Lock = LockPolicy{InitiativeExclusive: doc.Lock.InitiativeExclusive}
	}

	graph, phaseProfiles, replanPhases, acceptancePhases, err := doc.buildPhaseGraph(def.Prompt.CatalogPrefix, def.Profile.DefaultProfileKey)
	if err != nil {
		return Definition{}, err
	}
	def.PhaseGraph = graph
	def.Profile.PhaseProfiles = phaseProfiles
	def.Metrics.ReplanSamplePhases = replanPhases
	def.Metrics.AcceptanceSamplePhases = acceptancePhases

	return def, nil
}

func (doc modeDocument) buildPhaseGraph(catalogPrefix, defaultProfileKey string) (PhaseGraph, map[Phase]string, []Phase, []Phase, error) {
	graph := PhaseGraph{
		StartPhase:  Phase(doc.PhaseGraph.StartPhase),
		Terminal:    toPhases(doc.PhaseGraph.Terminal),
		Transitions: map[Phase][]Phase{},
		Guards:      map[Phase][]GuardedTransition{},
		Phases:      map[Phase]PhaseDefinition{},
	}
	phaseProfiles := map[Phase]string{}
	var replanPhases, acceptancePhases []Phase

	for _, phaseDoc := range doc.PhaseGraph.Phases {
		phase := Phase(phaseDoc.ID)
		phaseDef, err := phaseDoc.toPhaseDefinition(catalogPrefix, defaultProfileKey)
		if err != nil {
			return PhaseGraph{}, nil, nil, nil, fmt.Errorf("mode %q phase %q: %w", doc.ID, phase, err)
		}

		adjacency, guards, classification, err := phaseDoc.transitions()
		if err != nil {
			return PhaseGraph{}, nil, nil, nil, fmt.Errorf("mode %q phase %q: %w", doc.ID, phase, err)
		}
		phaseDef.TransitionClassification = classification
		graph.Phases[phase] = phaseDef
		phaseProfiles[phase] = phaseDef.ProfileKey

		if len(adjacency) > 0 {
			graph.Transitions[phase] = adjacency
		}
		if len(guards) > 0 {
			graph.Guards[phase] = guards
		}

		if phaseDoc.Metrics != nil {
			if phaseDoc.Metrics.CountsReplanSample {
				replanPhases = append(replanPhases, phase)
			}
			if phaseDoc.Metrics.CountsAcceptanceSample {
				acceptancePhases = append(acceptancePhases, phase)
			}
		}
	}
	return graph, phaseProfiles, replanPhases, acceptancePhases, nil
}

func (p phaseDoc) toPhaseDefinition(catalogPrefix, defaultProfileKey string) (PhaseDefinition, error) {
	if executedBy := strings.TrimSpace(p.ExecutedBy); executedBy != "" {
		// A delegated phase's execution surface (prompt, reads, emits,
		// artifacts, purposes, profile) comes entirely from the sub-mode. The
		// schema already forbids declaring any of it; the loader re-checks so a
		// hand-built document cannot bypass the contract.
		if len(p.Reads) > 0 || p.Prompt != nil || p.DeclaredOutput != nil ||
			len(p.OutputArtifacts) > 0 || len(p.ResultBindings) > 0 || p.Metrics != nil ||
			strings.TrimSpace(p.ActivityPurpose) != "" || strings.TrimSpace(p.LockPurpose) != "" ||
			strings.TrimSpace(p.ProfileKey) != "" || p.WritesRepo || p.RequiresCriteria {
			return PhaseDefinition{}, fmt.Errorf("delegated phase (executed_by=%q) must not declare reads/prompt/declared_output/output_artifacts/result_bindings/metrics/purposes/profile/writes_repo/requires_criteria: the sub-mode owns the execution surface", executedBy)
		}
		return PhaseDefinition{
			Phase:          Phase(p.ID),
			Kind:           PhaseKind(p.Kind),
			ExecutedBy:     Mode(executedBy),
			AutoStartAfter: toPhases(p.AutoStartAfter),
		}, nil
	}
	suffix := strings.TrimSpace(promptSuffix(p))
	if suffix == "" {
		suffix = p.ID
	}
	catalogID := catalogPrefix + "-" + suffix

	lockPurpose := strings.TrimSpace(p.LockPurpose)
	if lockPurpose == "" {
		lockPurpose = p.ActivityPurpose
	}

	profileKey := strings.TrimSpace(p.ProfileKey)
	if profileKey == "" {
		profileKey = defaultProfileKey
	}

	outputArtifacts := mergeOutputArtifacts(p.outputArtifacts(), p.resultBindings())
	outputContract := p.outputContract(outputArtifacts)
	outputContract.RequiresPlanRef = p.hasRequiredField("plan_ref")

	return PhaseDefinition{
		Phase:           Phase(p.ID),
		Kind:            PhaseKind(p.Kind),
		AutoStartAfter:  toPhases(p.AutoStartAfter),
		Reads:           append([]string(nil), p.Reads...),
		ActivityPurpose: p.ActivityPurpose,
		LockPurpose:     lockPurpose,
		CatalogID:       catalogID,
		SkillID:         catalogID,
		PromptCatalog: PromptCatalogMetadata{
			Title:   defaultString(promptTitle(p), humanizeToken(p.ID)),
			Trigger: defaultString(promptTrigger(p), "Operator starts "+catalogPrefix+" "+suffix+" phase"),
			Purpose: defaultString(promptPurpose(p), "Run the "+p.ID+" phase."),
		},
		ProfileKey:       profileKey,
		WritesRepo:       p.WritesRepo,
		OutputArtifacts:  outputArtifacts,
		ResultBindings:   p.resultBindings(),
		OutputContract:   outputContract,
		DeclaredOutput:   p.declaredOutput(),
		RequiresCriteria: p.RequiresCriteria,
	}, nil
}

// declaredOutput converts the on-disk declared_output block into the typed
// DeclaredOutput, applying the schema defaults (envelope key, structured-result
// requirement, and resolution-ladder knobs) so downstream consumers never read
// a zero-value that means "disabled".
func (p phaseDoc) declaredOutput() *DeclaredOutput {
	if p.DeclaredOutput == nil {
		return nil
	}
	out := &DeclaredOutput{
		EnvelopeKey:              defaultString(p.DeclaredOutput.EnvelopeKey, resultEnvelopeKey),
		RequiresStructuredResult: p.requiresStructuredResult(),
		Fields:                   toOutputFields(p.DeclaredOutput.Fields),
		Resolution:               resolutionPolicy(p.DeclaredOutput.Resolution),
	}
	return out
}

func toOutputFields(fields []outputFieldDoc) []OutputField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]OutputField, 0, len(fields))
	for _, f := range fields {
		out = append(out, OutputField{
			Name:        f.Name,
			Type:        f.Type,
			Required:    f.Required,
			Enum:        append([]any(nil), f.Enum...),
			Minimum:     f.Minimum,
			Maximum:     f.Maximum,
			MinLength:   f.MinLength,
			MaxLength:   f.MaxLength,
			Description: f.Description,
			Fields:      toOutputFields(f.Fields),
		})
	}
	return out
}

// resolutionPolicy applies the schema's resolution defaults: true-final-message
// detection on, a 5-message scan window, and the L2 classifier allowed.
func resolutionPolicy(res *resolutionDoc) ResolutionPolicy {
	policy := ResolutionPolicy{
		DetectTrueFinalMessage: true,
		ScanLastNMessages:      5,
		AllowClassifier:        true,
	}
	if res == nil {
		return policy
	}
	if res.DetectTrueFinalMessage != nil {
		policy.DetectTrueFinalMessage = *res.DetectTrueFinalMessage
	}
	if res.ScanLastNMessages != nil {
		policy.ScanLastNMessages = *res.ScanLastNMessages
	}
	if res.AllowClassifier != nil {
		policy.AllowClassifier = *res.AllowClassifier
	}
	return policy
}

// outputContract derives the phase output contract from the declared output
// schema, unifying the older scattered requires_* flags into
// declared-field-driven derivation: a required field named progress/verdict/
// handoff/backlog_sync sets the matching contract flag.
func (p phaseDoc) outputContract(outputArtifacts []ArtifactDefinition) PhaseOutputContract {
	return PhaseOutputContract{
		RequiresStructuredResult: p.requiresStructuredResult(),
		RequiredArtifacts:        requiredArtifacts(outputArtifacts),
		RequiresProgress:         p.hasRequiredField("progress"),
		RequiresVerdict:          p.hasRequiredField("verdict"),
		RequiresHandoff:          p.hasRequiredField("handoff"),
		RequiresBacklogSync:      p.hasRequiredField("backlog_sync"),
	}
}

func (p phaseDoc) requiresStructuredResult() bool {
	if p.DeclaredOutput != nil && p.DeclaredOutput.RequiresStructuredResult != nil {
		return *p.DeclaredOutput.RequiresStructuredResult
	}
	// Initiative-scoped phases always require a structured result envelope; the
	// Go builders set this unconditionally.
	return true
}

func (p phaseDoc) hasRequiredField(name string) bool {
	if p.DeclaredOutput == nil {
		return false
	}
	for _, field := range p.DeclaredOutput.Fields {
		if field.Name == name && field.Required {
			return true
		}
	}
	return false
}

// transitions expands the phase's declared edges into the derived static
// adjacency, the ordered guard list, and (when a classified edge is declared)
// the phase's classification contract. A classified edge expands into one
// eq-guard per enum value, in declared enum order, at its declared position
// among the phase's other transitions — routing stays pure guard evaluation;
// classification only supplies the field's value at the edge.
func (p phaseDoc) transitions() ([]Phase, []GuardedTransition, *TransitionClassification, error) {
	if len(p.Transitions) == 0 {
		return nil, nil, nil, nil
	}
	var adjacency []Phase
	seen := map[Phase]struct{}{}
	guards := make([]GuardedTransition, 0, len(p.Transitions))
	var classification *TransitionClassification
	appendEdge := func(gt GuardedTransition) {
		guards = append(guards, gt)
		for _, target := range gt.To {
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			adjacency = append(adjacency, target)
		}
	}
	for i, t := range p.Transitions {
		if t.Classify != nil {
			if classification != nil {
				return nil, nil, nil, fmt.Errorf("transition[%d]: phase declares more than one classified transition; a phase owns at most one classification contract", i)
			}
			contract, expanded, err := expandClassifiedTransition(*t.Classify, t.Routes)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("transition[%d]: %w", i, err)
			}
			classification = contract
			for _, gt := range expanded {
				appendEdge(gt)
			}
			continue
		}
		if t.When == nil {
			return nil, nil, nil, fmt.Errorf("transition[%d]: requires either when/to or classify/routes", i)
		}
		appendEdge(GuardedTransition{When: *t.When, To: toPhases(t.To)})
	}
	return adjacency, guards, classification, nil
}

// expandClassifiedTransition validates a classified edge's declaration and
// expands its routes into eq-guards over the classification field, one per
// enum value in declared order. Every enum value must be routed (an empty
// target list is a guarded stop); routes may not name values outside the enum.
func expandClassifiedTransition(doc classificationContractDoc, routes map[string][]string) (*TransitionClassification, []GuardedTransition, error) {
	field := strings.TrimSpace(doc.Field)
	if !isValidFieldPath(field) {
		return nil, nil, fmt.Errorf("classify.field %q must be a dotted lowercase field-path", doc.Field)
	}
	if len(doc.Enum) == 0 {
		return nil, nil, fmt.Errorf("classify.enum must declare at least one routing value")
	}
	enumSet := make(map[string]struct{}, len(doc.Enum))
	for _, v := range doc.Enum {
		if strings.TrimSpace(v) == "" {
			return nil, nil, fmt.Errorf("classify.enum contains an empty value")
		}
		if _, dup := enumSet[v]; dup {
			return nil, nil, fmt.Errorf("classify.enum declares duplicate value %q", v)
		}
		enumSet[v] = struct{}{}
	}
	from := strings.TrimSpace(doc.From)
	if from != "" && !isValidFieldPath(from) {
		return nil, nil, fmt.Errorf("classify.from %q must be a dotted lowercase field-path", doc.From)
	}
	if len(routes) == 0 {
		return nil, nil, fmt.Errorf("classified transition requires a routes map")
	}
	for key := range routes {
		if _, ok := enumSet[key]; !ok {
			return nil, nil, fmt.Errorf("routes declares %q, which is not a classify.enum value", key)
		}
	}
	guards := make([]GuardedTransition, 0, len(doc.Enum))
	for _, v := range doc.Enum {
		targets, ok := routes[v]
		if !ok {
			return nil, nil, fmt.Errorf("routes is missing enum value %q; every routing value needs a route (empty = guarded stop)", v)
		}
		guards = append(guards, GuardedTransition{
			When: Guard{Op: GuardOpEq, Field: field, Value: v},
			To:   toPhases(targets),
		})
	}
	contract := &TransitionClassification{
		Field:       field,
		Enum:        append([]string(nil), doc.Enum...),
		From:        from,
		Description: strings.TrimSpace(doc.Description),
	}
	return contract, guards, nil
}

func (p phaseDoc) outputArtifacts() []ArtifactDefinition {
	if len(p.OutputArtifacts) == 0 {
		return nil
	}
	out := make([]ArtifactDefinition, 0, len(p.OutputArtifacts))
	for _, a := range p.OutputArtifacts {
		out = append(out, ArtifactDefinition(a))
	}
	return out
}

func (p phaseDoc) resultBindings() []ResultBinding {
	if len(p.ResultBindings) == 0 {
		return nil
	}
	out := make([]ResultBinding, 0, len(p.ResultBindings))
	for _, b := range p.ResultBindings {
		out = append(out, ResultBinding{
			Kind:     ResultBindingKind(b.Kind),
			Artifact: ArtifactDefinition(b.Artifact),
		})
	}
	return out
}

func (doc modeDocument) backlogSyncPolicy() BacklogSyncPolicy {
	policy := BacklogSyncPolicy{EventSource: backlogEventSource(doc)}
	if doc.BacklogSync == nil {
		return policy
	}
	for _, cap := range doc.BacklogSync.Capabilities {
		policy.Capabilities = append(policy.Capabilities, BacklogSyncCapability(cap))
	}
	policy.RequiresRunID = doc.BacklogSync.RequiresRunID
	policy.RequiresMembership = doc.BacklogSync.RequiresMembership
	policy.ApplyMode = BacklogSyncApplyMode(doc.BacklogSync.ApplyMode)
	return policy
}

func (doc modeDocument) artifactPolicy() ArtifactPolicy {
	if doc.Artifact == nil {
		return ArtifactPolicy{}
	}
	root := doc.Artifact.Root
	roundRoot := strings.TrimSpace(doc.Artifact.RoundRoot)
	if roundRoot == "" {
		roundRoot = strings.TrimRight(root, "/") + "/rounds"
	}
	return ArtifactPolicy{Root: root, RoundRoot: roundRoot}
}

func (doc modeDocument) targetPolicy() TargetPolicy {
	policy := TargetPolicy{Kind: TargetKind(doc.Target.Kind)}
	if doc.Target.PlanRef != nil {
		policy.PlanRef = PlanRefPolicy{
			Required: doc.Target.PlanRef.Required,
			Role:     strings.TrimSpace(doc.Target.PlanRef.Role),
		}
	}
	return policy
}

func uiTabID(ui *uiDoc) string {
	if ui == nil {
		return ""
	}
	return ui.WorkspaceTabID
}

func profileDefaultKey(profile *profilePolicyDoc) string {
	if profile == nil {
		return ""
	}
	return profile.DefaultProfileKey
}

// metricsEventSource / backlogEventSource default the event-source tag to the
// mode id when omitted, matching the Go builders (which pass string(mode)).
func metricsEventSource(doc modeDocument) string {
	if doc.Metrics != nil && strings.TrimSpace(doc.Metrics.EventSource) != "" {
		return doc.Metrics.EventSource
	}
	return doc.ID
}

func backlogEventSource(doc modeDocument) string {
	if doc.BacklogSync != nil && strings.TrimSpace(doc.BacklogSync.EventSource) != "" {
		return doc.BacklogSync.EventSource
	}
	return doc.ID
}

func promptSuffix(p phaseDoc) string {
	if p.Prompt == nil {
		return ""
	}
	return p.Prompt.Suffix
}

func promptTitle(p phaseDoc) string {
	if p.Prompt == nil {
		return ""
	}
	return p.Prompt.Title
}

func promptTrigger(p phaseDoc) string {
	if p.Prompt == nil {
		return ""
	}
	return p.Prompt.Trigger
}

func promptPurpose(p phaseDoc) string {
	if p.Prompt == nil {
		return ""
	}
	return p.Prompt.Purpose
}

func toPhases(ids []string) []Phase {
	if len(ids) == 0 {
		return nil
	}
	out := make([]Phase, 0, len(ids))
	for _, id := range ids {
		out = append(out, Phase(id))
	}
	return out
}

// LoadModesFromDir discovers every modes/<id>/mode.json under dir, loads each
// into a Definition, and validates the full set (structural + semantic
// cross-references). It is the data-backed replacement for the static Go
// registry map. A malformed or semantically-invalid mode fails the whole load
// with an actionable error naming the offending mode.
func LoadModesFromDir(dir string) (map[Mode]Definition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read modes dir %q: %w", dir, err)
	}
	defs := map[Mode]Definition{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modePath := filepath.Join(dir, entry.Name(), ModeFileName)
		raw, err := os.ReadFile(modePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %q: %w", modePath, err)
		}
		def, err := LoadModeDefinition(raw)
		if err != nil {
			return nil, fmt.Errorf("load mode %q: %w", entry.Name(), err)
		}
		if string(def.Mode) != entry.Name() {
			return nil, fmt.Errorf("mode folder %q declares mismatched id %q", entry.Name(), def.Mode)
		}
		if _, dup := defs[def.Mode]; dup {
			return nil, fmt.Errorf("duplicate mode id %q", def.Mode)
		}
		runs, err := LoadExampleRunsForMode(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load mode %q example-runs: %w", entry.Name(), err)
		}
		def.ExampleRuns = runs
		defs[def.Mode] = def
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("no modes discovered under %q", dir)
	}
	if err := ValidateLoadedModes(defs); err != nil {
		return nil, err
	}
	return defs, nil
}

// SortedModes returns the mode ids of a loaded set in stable order.
func SortedModes(defs map[Mode]Definition) []Mode {
	modes := make([]Mode, 0, len(defs))
	for mode := range defs {
		modes = append(modes, mode)
	}
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	return modes
}

// mergeOutputArtifacts folds a phase's result-binding artifacts into its
// declared output-artifact list, deduplicating by path and preferring the
// stronger (required, typed) declaration. The runtime treats output artifacts
// and progress-bound artifacts uniformly, so the loader flattens them here.
func mergeOutputArtifacts(artifacts []ArtifactDefinition, bindings []ResultBinding) []ArtifactDefinition {
	out := append([]ArtifactDefinition(nil), artifacts...)
	seen := map[string]int{}
	for i, artifact := range out {
		seen[strings.TrimSpace(artifact.Path)] = i
	}
	for _, binding := range bindings {
		path := strings.TrimSpace(binding.Artifact.Path)
		if path == "" {
			out = append(out, binding.Artifact)
			continue
		}
		if existing, ok := seen[path]; ok {
			out[existing] = mergeArtifactDefinition(out[existing], binding.Artifact)
			continue
		}
		seen[path] = len(out)
		out = append(out, binding.Artifact)
	}
	return out
}

func mergeArtifactDefinition(existing ArtifactDefinition, incoming ArtifactDefinition) ArtifactDefinition {
	if existing.ContentType == "" {
		existing.ContentType = incoming.ContentType
	}
	existing.Required = existing.Required || incoming.Required
	return existing
}

// humanizeToken turns a snake/kebab phase id into a Title Cased label, used to
// default a phase's prompt-catalog title when the mode data omits one.
func humanizeToken(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
