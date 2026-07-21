// Package workflowcatalog validates and canonicalizes scenario-owned workflow
// definitions. It deliberately contains no execution logic.
package workflowcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"text/template/parse"

	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowexpr"
)

// sharedCELEnv is the single CEL environment the engine also evaluates against,
// built once. Sharing it guarantees a condition that compiles at registration
// time is a condition the runtime can evaluate.
var sharedCELEnv = sync.OnceValues(workflowexpr.NewEnv)

const (
	MaxDefinitionBytes = 256 << 10
	MaxSchemaBytes     = 32 << 10
	MaxBindingBytes    = 64 << 10
	MaxBindingLimit    = 1000
	MaxEdgeTraversals  = 10_000
	MaxWallTimeSeconds = 86_400
	MaxTurns           = 1_000
	MaxTokens          = 10_000_000
	MaxCostUSD         = 10_000
	MaxNodeAttempts    = 10_000
	MaxChildren        = 1_000
	MaxConcurrency     = 64
	MaxRecursion       = 16
	MaxRetries         = 100
	MaxWaitSeconds     = 86_400
)

var (
	identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	versionPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

type Lookup interface {
	ProfileExists(key string) bool
	RoleExists(key string) bool
}

type Result struct {
	Definition  domain.WorkflowDefinition   `json:"definition"`
	Canonical   []byte                      `json:"-"`
	Digest      string                      `json:"digest"`
	Diagnostics []domain.WorkflowDiagnostic `json:"diagnostics,omitempty"`
}

func Parse(data []byte, lookup Lookup) (*Result, error) {
	if len(data) == 0 || len(data) > MaxDefinitionBytes {
		return nil, fmt.Errorf("workflow definition must be between 1 and %d bytes", MaxDefinitionBytes)
	}
	var definition domain.WorkflowDefinition
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return nil, fmt.Errorf("decode workflow definition: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, fmt.Errorf("decode workflow definition: %w", err)
	}
	return Validate(definition, lookup)
}

func Validate(definition domain.WorkflowDefinition, lookup Lookup) (*Result, error) {
	diagnostics := validate(&definition, lookup)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
	// A blocking (error) diagnostic withholds the digest so nothing registers.
	// Warning-only definitions still canonicalize and earn a digest; the
	// warnings ride along on the result so callers can surface them.
	if domain.HasBlockingDiagnostic(diagnostics) {
		return &Result{Definition: definition, Diagnostics: diagnostics}, nil
	}
	canonical, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("canonicalize workflow definition: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return &Result{Definition: definition, Canonical: canonical, Digest: "sha256:" + hex.EncodeToString(sum[:]), Diagnostics: diagnostics}, nil
}

// Canonical shape of the implied end synthesized for single-node sugar. These
// constants are load-bearing for digest identity: an explicit definition that
// spells out the same end node, edge, and entryNode must marshal to identical
// bytes, so they must never drift.
const (
	sugarEndNodeID            = "end"
	sugarResultBindingName    = "result"
	sugarResultBindingBytes   = 16384
	sugarResultBindingSelect  = "$.value"
	sugarResultBindingOrder   = "desc"
	sugarResultBindingRender  = "json"
	sugarResultBindingMissing = "error"
)

// expandSingleNodeSugar canonicalizes the single-run-node authoring shorthand
// into the full form BEFORE anything downstream (validation and, critically, the
// digest) sees it. A definition whose only node is a run node may omit
// entryNode, edges, and the terminal end node; the expander synthesizes the
// entry, an implied end that maps the run node's structured result to
// output.result, and the single unconditional edge between them. Because the
// expansion is deterministic, an equivalent explicit definition canonicalizes to
// identical bytes and therefore an identical digest.
func expandSingleNodeSugar(d *domain.WorkflowDefinition) {
	if d.EntryNode != "" || len(d.Edges) != 0 || len(d.Nodes) != 1 {
		return
	}
	run := d.Nodes[0]
	if run.Kind != domain.WorkflowNodeRun || run.Run == nil || run.ID == sugarEndNodeID {
		return
	}
	d.EntryNode = run.ID
	d.Nodes = append(d.Nodes, domain.WorkflowNode{
		ID:   sugarEndNodeID,
		Kind: domain.WorkflowNodeEnd,
		End: &domain.WorkflowEndNode{
			Status: "succeeded",
			Bindings: []domain.WorkflowInputBinding{{
				Name:          sugarResultBindingName,
				Source:        domain.WorkflowBindingStructured,
				Selector:      sugarResultBindingSelect,
				Order:         sugarResultBindingOrder,
				Limit:         1,
				MaxBytes:      sugarResultBindingBytes,
				RenderAs:      sugarResultBindingRender,
				MissingPolicy: sugarResultBindingMissing,
			}},
		},
	})
	d.Edges = append(d.Edges, domain.WorkflowEdge{From: run.ID, To: sugarEndNodeID})
}

func validate(d *domain.WorkflowDefinition, lookup Lookup) []domain.WorkflowDiagnostic {
	expandSingleNodeSugar(d)
	var out []domain.WorkflowDiagnostic
	add := func(code, path, message string) {
		out = append(out, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message, Severity: domain.DiagnosticSeverityError})
	}
	warn := func(code, path, message string) {
		out = append(out, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message, Severity: domain.DiagnosticSeverityWarning})
	}
	celEnv, celErr := sharedCELEnv()
	if d.SchemaVersion != domain.WorkflowSchemaVersionV1 {
		add("schema_version", "schemaVersion", "must equal "+domain.WorkflowSchemaVersionV1)
	}
	if !identifierPattern.MatchString(d.Owner) {
		add("owner", "owner", "must be a canonical scenario slug")
	}
	if d.Key != d.Owner+"/"+strings.TrimPrefix(d.Key, d.Owner+"/") || !identifierPattern.MatchString(strings.TrimPrefix(d.Key, d.Owner+"/")) {
		add("key", "key", "must be owner/name with a canonical name")
	}
	if !versionPattern.MatchString(d.Version) {
		add("version", "version", "must be semantic version x.y.z")
	}
	d.InputSchema = normalizeSchema(d.InputSchema, "inputSchema", add)
	d.OutputSchema = normalizeSchema(d.OutputSchema, "outputSchema", add)
	if len(d.Nodes) == 0 {
		add("nodes", "nodes", "must contain at least one node")
	}
	if !positiveBudgets(d.Budgets) {
		add("budgets", "budgets", "all workflow budgets must be finite and positive")
	} else if !withinSafetyLimits(d.Budgets) {
		add("budget_ceiling", "budgets", "workflow budgets exceed an Agent Manager operator safety ceiling")
	}
	validateTriggerPolicy(d.Trigger, add)

	nodes := make(map[string]*domain.WorkflowNode, len(d.Nodes))
	for i := range d.Nodes {
		n := &d.Nodes[i]
		path := fmt.Sprintf("nodes[%d]", i)
		if !identifierPattern.MatchString(n.ID) {
			add("node_id", path+".id", "must be a canonical identifier")
		}
		if _, exists := nodes[n.ID]; exists {
			add("duplicate_node", path+".id", "node id must be unique")
		} else {
			nodes[n.ID] = n
		}
		validateNode(n, path, lookup, add, warn)
	}
	validateExperimentEvaluator(d, nodes, add)
	for i := range d.Nodes {
		n := &d.Nodes[i]
		if n.Wait == nil || n.Wait.OnTimeout == "" {
			continue
		}
		if _, ok := nodes[n.Wait.OnTimeout]; !ok {
			add("wait_timeout_target", fmt.Sprintf("nodes[%d].wait.onTimeout", i), "must name an existing node")
		}
	}
	if _, ok := nodes[d.EntryNode]; !ok {
		add("entry_node", "entryNode", "must name an existing node")
	}

	adj := make(map[string][]domain.WorkflowEdge)
	for i, edge := range d.Edges {
		path := fmt.Sprintf("edges[%d]", i)
		if _, ok := nodes[edge.From]; !ok {
			add("edge_from", path+".from", "must name an existing node")
		}
		if _, ok := nodes[edge.To]; !ok {
			add("edge_to", path+".to", "must name an existing node")
		}
		if edge.MaxTraversals < 0 || edge.MaxTraversals > MaxEdgeTraversals {
			add("edge_budget", path+".maxTraversals", fmt.Sprintf("must be between 0 and %d", MaxEdgeTraversals))
		}
		// Compile the condition the engine actually evaluates at this edge (an
		// empty condition is the unconditional fallback edge). Sharing the
		// engine's CEL environment means a condition that compiles here is one
		// the runtime can evaluate, and a syntax or type error is caught at
		// registration instead of mid-execution.
		if strings.TrimSpace(edge.Condition) != "" {
			if celErr != nil {
				add("edge_condition", path+".condition", "workflow expression environment unavailable: "+celErr.Error())
			} else if err := celEnv.Check(edge.Condition); err != nil {
				add("edge_condition", path+".condition", err.Error())
			}
		}
		adj[edge.From] = append(adj[edge.From], edge)
	}
	validateReachability(d.EntryNode, nodes, adj, add)
	validateContinuations(d.EntryNode, d.Nodes, nodes, adj, add)
	validateParallelBranches(nodes, adj, add)
	validateCycles(nodes, adj, add)
	for id, node := range nodes {
		if node.Kind == domain.WorkflowNodeEnd && len(adj[id]) != 0 {
			add("end_edge", "nodes."+id, "end nodes cannot have outgoing edges")
		}
		if node.Kind != domain.WorkflowNodeEnd && len(adj[id]) == 0 {
			add("dead_end", "nodes."+id, "non-end nodes require an outgoing edge")
		}
	}
	return out
}

func validateExperimentEvaluator(d *domain.WorkflowDefinition, nodes map[string]*domain.WorkflowNode, add func(string, string, string)) {
	armed := map[string]bool{}
	for _, node := range d.Nodes {
		if node.Run != nil && node.Run.PromptRef != nil && node.Run.PromptRef.ExperimentID != "" {
			armed[node.ID] = true
		}
		if node.Continue != nil && node.Continue.PromptRef != nil && node.Continue.PromptRef.ExperimentID != "" {
			armed[node.ID] = true
		}
	}
	if len(armed) == 0 {
		return
	}
	c := d.ExperimentEvaluator
	if c == nil {
		add("experiment_evaluator_required", "experimentEvaluator", "armed promptRefs require an explicit independent evaluator contract")
		return
	}
	if c.EvaluatorNodeID == "" || c.VerdictPointer == "" || c.RubricHash == "" || c.RubricAuthor == "" || c.EvaluatorPromptHash == "" || c.IndependenceDeclaration == "" {
		add("experiment_evaluator_incomplete", "experimentEvaluator", "evaluator node, verdict pointer, rubric provenance, and independence declaration are required")
	}
	if strings.EqualFold(strings.TrimSpace(c.RubricAuthor), "skill-optimizer") {
		add("experiment_evaluator_author", "experimentEvaluator.rubricAuthor", "rubric author must be outside skill-optimizer")
	}
	evaluator := nodes[c.EvaluatorNodeID]
	if evaluator == nil || (evaluator.Run == nil && evaluator.Continue == nil) {
		add("experiment_evaluator_node", "experimentEvaluator.evaluatorNodeId", "must name a run or continue evaluator node")
	} else {
		var result *domain.ResultSpec
		var profile, role string
		if evaluator.Run != nil {
			result, profile, role = evaluator.Run.ResultSpec, evaluator.Run.ProfileKey, evaluator.Run.RoleRef
		} else {
			result = evaluator.Continue.ResultSpec
		}
		if result == nil {
			add("experiment_evaluator_result", "experimentEvaluator.evaluatorNodeId", "evaluator must declare a structured result spec")
		}
		for treatmentID := range armed {
			if treatmentID == c.EvaluatorNodeID {
				add("experiment_evaluator_independence", "experimentEvaluator.evaluatorNodeId", "evaluator cannot be a treatment node")
			}
			treatment := nodes[treatmentID]
			if treatment != nil && treatment.Run != nil && ((profile != "" && profile == treatment.Run.ProfileKey) || (role != "" && role == treatment.Run.RoleRef)) {
				add("experiment_evaluator_independence", "experimentEvaluator.evaluatorNodeId", "evaluator profile or role must differ from treatment")
			}
		}
	}
	if len(c.AllowedVerdicts) == 0 || len(c.SuccessVerdicts) == 0 {
		add("experiment_evaluator_verdicts", "experimentEvaluator", "allowed and success verdict vocabularies are required")
	}
	allowed := map[string]bool{}
	for _, verdict := range c.AllowedVerdicts {
		if strings.TrimSpace(verdict) != "" {
			allowed[verdict] = true
		}
	}
	for _, verdict := range c.SuccessVerdicts {
		if !allowed[verdict] {
			add("experiment_evaluator_success_mapping", "experimentEvaluator.successVerdicts", "success verdict must be in allowedVerdicts")
		}
	}
	for treatmentID := range armed {
		if !containsString(c.TreatmentNodeIDs, treatmentID) {
			add("experiment_evaluator_treatment", "experimentEvaluator.treatmentNodeIds", "must include every armed treatment node")
		}
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateTriggerPolicy(policy domain.WorkflowTriggerPolicy, add func(string, string, string)) {
	seen := map[domain.WorkflowInitiator]bool{}
	for index, initiator := range policy.Initiators {
		path := fmt.Sprintf("trigger.initiators[%d]", index)
		switch initiator {
		case domain.WorkflowInitiatorHuman, domain.WorkflowInitiatorProgrammatic, domain.WorkflowInitiatorAgent:
		default:
			add("trigger_initiator", path, "must be human, programmatic, or agent")
		}
		if seen[initiator] {
			add("trigger_initiator", path, "must not repeat an initiator")
		}
		seen[initiator] = true
	}
	switch policy.SelfTrigger.Mode {
	case "", domain.WorkflowSelfTriggerDeny:
		if policy.SelfTrigger.MaxDepth != 0 {
			add("trigger_self", "trigger.selfTrigger.maxDepth", "maxDepth is allowed only when selfTrigger.mode is allow")
		}
	case domain.WorkflowSelfTriggerAllow:
		if policy.SelfTrigger.MaxDepth < 1 {
			add("trigger_self", "trigger.selfTrigger.maxDepth", "must be at least 1 when selfTrigger.mode is allow")
		}
	default:
		add("trigger_self", "trigger.selfTrigger.mode", "must be deny or allow")
	}
}

func validateParallelBranches(nodes map[string]*domain.WorkflowNode, adj map[string][]domain.WorkflowEdge, add func(string, string, string)) {
	for id, node := range nodes {
		if node.Branch == nil || !node.Branch.Parallel {
			continue
		}
		if len(adj[id]) < 2 {
			add("parallel_members", "nodes."+id, "parallel branch requires at least two members")
			continue
		}
		joinID := ""
		for _, edge := range adj[id] {
			member := nodes[edge.To]
			if member == nil || (member.Kind != domain.WorkflowNodeRun && member.Kind != domain.WorkflowNodeContinue && member.Kind != domain.WorkflowNodeChild) {
				add("parallel_member", "nodes."+id, "parallel members must be run, continue, or child_workflow nodes")
				continue
			}
			if len(adj[member.ID]) != 1 || adj[member.ID][0].Condition != "" {
				add("parallel_join", "nodes."+member.ID, "parallel member must have one unconditional edge to the join")
				continue
			}
			if joinID == "" {
				joinID = adj[member.ID][0].To
			} else if joinID != adj[member.ID][0].To {
				add("parallel_join", "nodes."+id, "all parallel members must converge on one join")
			}
		}
		join := nodes[joinID]
		if join == nil || join.Kind != domain.WorkflowNodeJoin {
			add("parallel_join", "nodes."+id, "parallel members must converge on a join node")
		} else if join.Join.Strategy == "quorum" && join.Join.Quorum > len(adj[id]) {
			add("parallel_quorum", "nodes."+joinID, "quorum cannot exceed parallel member count")
		}
	}
}

func normalizeSchema(raw json.RawMessage, path string, add func(string, string, string)) json.RawMessage {
	if len(raw) == 0 {
		add("schema_required", path, "JSON schema is required")
		return raw
	}
	if len(raw) > MaxSchemaBytes {
		add("schema_size", path, "schema exceeds size limit")
		return raw
	}
	normalized, err := structuredresult.NormalizeSpec(&domain.ResultSpec{Version: structuredresult.SpecVersionV1, Kind: domain.ResultSpecKindJSONSchema, Schema: raw, ExtractionMode: domain.StructuredExtractionDeterministic})
	if err != nil {
		add("schema_invalid", path, err.Error())
		return raw
	}
	return normalized.Schema
}

func validateNode(n *domain.WorkflowNode, path string, lookup Lookup, add, warn func(string, string, string)) {
	payloads := 0
	for _, present := range []bool{n.Run != nil, n.Continue != nil, n.Child != nil, n.Wait != nil, n.Branch != nil, n.Join != nil, n.End != nil} {
		if present {
			payloads++
		}
	}
	if payloads != 1 {
		add("node_payload", path, "exactly one kind-specific payload is required")
		return
	}
	bindings := []domain.WorkflowInputBinding(nil)
	switch n.Kind {
	case domain.WorkflowNodeRun:
		if n.Run == nil {
			add("node_kind", path, "run payload is required")
			return
		}
		if (n.Run.ProfileKey == "") == (n.Run.RoleRef == "") {
			add("run_target", path+".run", "exactly one of profileKey or roleRef is required")
		}
		if lookup != nil && n.Run.ProfileKey != "" && !lookup.ProfileExists(n.Run.ProfileKey) {
			add("profile_missing", path+".run.profileKey", "profile does not exist")
		}
		if lookup != nil && n.Run.RoleRef != "" && !lookup.RoleExists(n.Run.RoleRef) {
			add("role_missing", path+".run.roleRef", "portable role does not exist")
		}
		validatePromptSource(n.Run.PromptTemplate, n.Run.PromptRef, n.Run.PromptProvenance, n.Run.Bindings, path+".run", add, warn)
		validateScopePathTemplate(n.Run.ScopePathTemplate, n.Run.Bindings, path+".run.scopePathTemplate", add)
		validateResultSpec(&n.Run.ResultSpec, path+".run.resultSpec", add)
		validateAgentNodeLimits(n.Run.MaxTurns, n.Run.TimeoutSeconds, path+".run", add)
		bindings = n.Run.Bindings
	case domain.WorkflowNodeContinue:
		if n.Continue == nil {
			add("node_kind", path, "continue payload is required")
			return
		}
		if n.Continue.ConversationFromNode == "" {
			add("continuation", path+".continue.conversationFromNode", "explicit prior run node is required")
		}
		validatePromptSource(n.Continue.PromptTemplate, n.Continue.PromptRef, n.Continue.PromptProvenance, n.Continue.Bindings, path+".continue", add, warn)
		validateResultSpec(&n.Continue.ResultSpec, path+".continue.resultSpec", add)
		validateAgentNodeLimits(n.Continue.MaxTurns, n.Continue.TimeoutSeconds, path+".continue", add)
		bindings = n.Continue.Bindings
	case domain.WorkflowNodeChild:
		if n.Child == nil {
			add("node_kind", path, "childWorkflow payload is required")
			return
		}
		if n.Child.WorkflowKey == "" || n.Child.MaxDepth <= 0 {
			add("child", path+".childWorkflow", "workflowKey and finite maxDepth are required")
		}
		bindings = n.Child.Bindings
	case domain.WorkflowNodeWait:
		if n.Wait == nil {
			add("node_kind", path, "wait payload is required")
			return
		}
		if n.Wait.Signal == "" || n.Wait.TimeoutSeconds < 0 {
			add("wait", path+".wait", "signal is required and timeoutSeconds cannot be negative")
		}
		if n.Wait.TimeoutSeconds == 0 && n.Wait.OnTimeout != "" {
			add("wait_timeout", path+".wait.onTimeout", "onTimeout requires a bounded timeoutSeconds")
		}
		if len(n.Wait.PayloadSchema) != 0 {
			n.Wait.PayloadSchema = normalizeSchema(n.Wait.PayloadSchema, path+".wait.payloadSchema", add)
		}
	case domain.WorkflowNodeBranch:
		if n.Branch == nil {
			add("branch", path+".branch", "branch payload is required")
		}
	case domain.WorkflowNodeJoin:
		if n.Join == nil || (n.Join.Strategy != "all" && n.Join.Strategy != "any" && n.Join.Strategy != "quorum") {
			add("join", path+".join.strategy", "strategy must be all, any, or quorum")
		} else if n.Join.Strategy == "quorum" && n.Join.Quorum <= 0 {
			add("join", path+".join.quorum", "quorum strategy requires a positive quorum")
		}
	case domain.WorkflowNodeEnd:
		if n.End == nil || !slices.Contains([]string{"succeeded", "blocked", "abstained", "budget_exhausted", "failed"}, n.End.Status) {
			add("end", path+".end.status", "status must be succeeded, blocked, abstained, budget_exhausted, or failed")
		} else {
			bindings = n.End.Bindings
		}
	default:
		add("node_kind", path+".kind", "unsupported node kind")
	}
	validateBindings(bindings, path, add)
}

// validateScopePathTemplate uses the same constrained template dialect as a
// prompt, but accepts only bindings that this run node has declared. Scope
// rendering happens before task creation, so unresolved fields must be caught
// while reconciling the declaration.
func validateScopePathTemplate(source string, bindings []domain.WorkflowInputBinding, path string, add func(string, string, string)) {
	if strings.TrimSpace(source) == "" {
		return
	}
	tmpl, err := workflowexpr.ParsePrompt(source)
	if err != nil {
		add("scope_path_template", path, err.Error())
		return
	}
	declared := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		declared[b.Name] = true
	}
	refs := templateFieldRefs(tmpl.Tree)
	for ref := range refs {
		if !declared[ref] {
			add("scope_path_unbound", path, fmt.Sprintf("scope path references {{.%s}} but no binding declares it", ref))
		}
	}
}

func validateAgentNodeLimits(maxTurns, timeoutSeconds int, path string, add func(string, string, string)) {
	if maxTurns < 0 || maxTurns > MaxTurns {
		add("node_limit", path+".maxTurns", "must be zero or within the workflow safety ceiling")
	}
	if timeoutSeconds < 0 || timeoutSeconds > MaxWallTimeSeconds {
		add("node_limit", path+".timeoutSeconds", "must be zero or within the workflow safety ceiling")
	}
}

func validateResultSpec(spec **domain.ResultSpec, path string, add func(string, string, string)) {
	if *spec == nil {
		return
	}
	normalized, err := structuredresult.NormalizeSpec(*spec)
	if err != nil {
		add("result_spec", path, err.Error())
		return
	}
	*spec = normalized
}

func validateBindings(bindings []domain.WorkflowInputBinding, path string, add func(string, string, string)) {
	seen := map[string]bool{}
	for i, b := range bindings {
		p := fmt.Sprintf("%s.bindings[%d]", path, i)
		if !identifierPattern.MatchString(b.Name) || seen[b.Name] {
			add("binding_name", p+".name", "must be a unique canonical identifier")
		}
		seen[b.Name] = true
		switch b.Source {
		case domain.WorkflowBindingInput, domain.WorkflowBindingAttempts, domain.WorkflowBindingRunResult, domain.WorkflowBindingStructured, domain.WorkflowBindingHandoff, domain.WorkflowBindingSignal, domain.WorkflowBindingCounter, domain.WorkflowBindingChild:
		default:
			add("binding_source", p+".source", "unsupported journal source")
		}
		if b.Limit <= 0 || b.Limit > MaxBindingLimit {
			add("binding_limit", p+".limit", "must be finite and within limit")
		}
		if b.MaxBytes <= 0 || b.MaxBytes > MaxBindingBytes {
			add("binding_size", p+".maxBytes", "must be finite and within limit")
		}
		if !slices.Contains([]string{"text", "json", "json_pretty", "xml", "markdown", "fenced"}, b.RenderAs) {
			add("binding_render", p+".renderAs", "must be text, json, json_pretty, xml, markdown, or fenced")
		}
		if b.WrapTag != "" && (b.RenderAs != "xml" || !identifierPattern.MatchString(b.WrapTag)) {
			add("binding_wrap_tag", p+".wrapTag", "requires xml rendering and a canonical tag")
		}
		if b.Lang != "" && b.RenderAs != "fenced" {
			add("binding_lang", p+".lang", "requires fenced rendering")
		}
		if b.Overflow != "" && b.Overflow != "error" && b.Overflow != "truncate" {
			add("binding_overflow", p+".overflow", "must be error or truncate")
		}
		if b.ItemMaxBytes < 0 || b.ItemMaxBytes > b.MaxBytes {
			add("binding_item_size", p+".itemMaxBytes", "must be positive and no greater than maxBytes")
		}
		if b.ItemTag != "" && !identifierPattern.MatchString(b.ItemTag) {
			add("binding_item_tag", p+".itemTag", "must be a canonical tag")
		}
		if b.EvictionPolicy != "" && !slices.Contains([]string{"keep_last", "keep_first", "keep_ends"}, b.EvictionPolicy) {
			add("binding_eviction", p+".evictionPolicy", "must be keep_last, keep_first, or keep_ends")
		}
		if b.EvictionPolicy != "" && b.RenderAs != "xml" {
			add("binding_eviction", p+".evictionPolicy", "requires xml rendering")
		}
		if (b.ItemTag != "" || b.ItemMaxBytes != 0) && (b.RenderAs != "xml" || b.EvictionPolicy == "") {
			add("binding_item", p, "item presentation requires xml rendering with an evictionPolicy")
		}
		if b.KeepFirst < 0 || b.EvictionPolicy != "keep_ends" && b.KeepFirst != 0 {
			add("binding_keep_first", p+".keepFirst", "requires keep_ends and cannot be negative")
		}
		if b.MissingPolicy != "error" && b.MissingPolicy != "omit" && b.MissingPolicy != "null" {
			add("binding_missing", p+".missingPolicy", "must be error, omit, or null")
		}
	}
}

// validatePromptSource enforces that a run/continue node authors exactly one of
// an inline promptTemplate or a promptRef, and validates whichever is present.
// A promptRef is a pre-reconcile authoring form: reconcile resolves it into an
// embedded promptTemplate (plus pinned provenance) before this validator runs on
// the resolved definition, so by the time a definition is digested it always
// carries a concrete template. When a promptRef survives to validation (e.g. the
// pure `workflow validate` lint path, which does not reach prompt-manager), it is
// accepted structurally but its placeholders cannot be cross-checked until it
// resolves.
func validatePromptSource(promptTemplate string, ref *domain.WorkflowPromptRef, provenance *domain.WorkflowPromptSource, bindings []domain.WorkflowInputBinding, path string, add, warn func(string, string, string)) {
	hasTemplate := strings.TrimSpace(promptTemplate) != ""
	hasRef := ref != nil
	switch {
	case hasTemplate && hasRef:
		add("prompt", path, "exactly one of promptTemplate or promptRef is allowed")
	case !hasTemplate && !hasRef:
		add("prompt", path, "exactly one of promptTemplate or promptRef is required")
	case hasRef:
		if strings.TrimSpace(ref.SkillID) == "" {
			add("prompt_ref", path+".promptRef.skillId", "promptRef requires a skillId")
		}
	default:
		if provenance == nil {
			warn("inline_prompt", path+".promptTemplate", "inline promptTemplate is supported but promptRef is required for the mature workflow rung")
		}
		validatePromptTemplate(promptTemplate, bindings, path+".promptTemplate", add, warn)
	}
}

// validatePromptTemplate parses the prompt with the exact text/template dialect
// the binding renderer uses (missingkey=error) and cross-checks every top-level
// placeholder against the node's declared binding names. A placeholder with no
// backing binding is an error (the render would fail at runtime); a binding the
// prompt never references is a warning (harmless but likely a mistake). It
// asserts against declared binding names, not runtime presence, so a binding
// with an omit/null missing policy is still considered satisfied.
func validatePromptTemplate(source string, bindings []domain.WorkflowInputBinding, path string, add, warn func(string, string, string)) {
	tmpl, err := workflowexpr.ParsePrompt(source)
	if err != nil {
		add("prompt_template", path, err.Error())
		return
	}
	refs := templateFieldRefs(tmpl.Tree)
	declared := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		declared[b.Name] = true
	}
	unbound := make([]string, 0, len(refs))
	for ref := range refs {
		if !declared[ref] {
			unbound = append(unbound, ref)
		}
	}
	sort.Strings(unbound)
	for _, ref := range unbound {
		add("prompt_unbound", path, fmt.Sprintf("prompt references {{.%s}} but no binding declares it", ref))
	}
	for _, b := range bindings {
		if !refs[b.Name] {
			warn("prompt_unused_binding", path, fmt.Sprintf("binding %q is declared but never referenced by the prompt", b.Name))
		}
	}
}

// templateFieldRefs collects the set of top-level field names a parsed prompt
// references. A reference like {{.a.b}} contributes only its root field "a"
// because bindings render into a flat map keyed by binding name.
func templateFieldRefs(tree *parse.Tree) map[string]bool {
	refs := map[string]bool{}
	if tree == nil {
		return refs
	}
	var walk func(parse.Node)
	walkPipe := func(pipe *parse.PipeNode) {
		if pipe != nil {
			walk(pipe)
		}
	}
	walk = func(n parse.Node) {
		switch x := n.(type) {
		case *parse.ListNode:
			if x == nil {
				return
			}
			for _, c := range x.Nodes {
				walk(c)
			}
		case *parse.ActionNode:
			walkPipe(x.Pipe)
		case *parse.PipeNode:
			for _, cmd := range x.Cmds {
				walk(cmd)
			}
		case *parse.CommandNode:
			for _, arg := range x.Args {
				walk(arg)
			}
		case *parse.FieldNode:
			if len(x.Ident) > 0 {
				refs[x.Ident[0]] = true
			}
		case *parse.IfNode:
			walkPipe(x.Pipe)
			walk(x.List)
			walk(x.ElseList)
		case *parse.RangeNode:
			walkPipe(x.Pipe)
			walk(x.List)
			walk(x.ElseList)
		case *parse.WithNode:
			walkPipe(x.Pipe)
			walk(x.List)
			walk(x.ElseList)
		case *parse.TemplateNode:
			walkPipe(x.Pipe)
		}
	}
	walk(tree.Root)
	return refs
}

func positiveBudgets(b domain.WorkflowBudgets) bool {
	return b.WallTimeSeconds > 0 && b.MaxTurns > 0 && b.MaxTokens > 0 && b.MaxCostUSD > 0 && b.MaxNodeAttempts > 0 && b.MaxChildren > 0 && b.MaxConcurrency > 0 && b.MaxRecursion > 0 && b.MaxRetries > 0 && b.MaxWaitSeconds > 0
}

func withinSafetyLimits(b domain.WorkflowBudgets) bool {
	return b.WallTimeSeconds <= MaxWallTimeSeconds && b.MaxTurns <= MaxTurns && b.MaxTokens <= MaxTokens && b.MaxCostUSD <= MaxCostUSD && b.MaxNodeAttempts <= MaxNodeAttempts && b.MaxChildren <= MaxChildren && b.MaxConcurrency <= MaxConcurrency && b.MaxRecursion <= MaxRecursion && b.MaxRetries <= MaxRetries && b.MaxWaitSeconds <= MaxWaitSeconds
}

func validateReachability(entry string, nodes map[string]*domain.WorkflowNode, adj map[string][]domain.WorkflowEdge, add func(string, string, string)) {
	seen := map[string]bool{}
	queue := []string{entry}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		for _, edge := range adj[id] {
			queue = append(queue, edge.To)
		}
	}
	for id := range nodes {
		if !seen[id] {
			add("unreachable", "nodes."+id, "node is unreachable from entryNode")
		}
	}
}

func validateContinuations(entry string, list []domain.WorkflowNode, nodes map[string]*domain.WorkflowNode, adj map[string][]domain.WorkflowEdge, add func(string, string, string)) {
	reachable := func(from, target string) bool {
		seen := map[string]bool{}
		q := []string{from}
		for len(q) > 0 {
			id := q[0]
			q = q[1:]
			if id == target {
				return true
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			for _, e := range adj[id] {
				q = append(q, e.To)
			}
		}
		return false
	}
	for _, n := range list {
		if n.Kind != domain.WorkflowNodeContinue || n.Continue == nil {
			continue
		}
		source := nodes[n.Continue.ConversationFromNode]
		if source == nil || source.Kind != domain.WorkflowNodeRun {
			add("continuation_source", "nodes."+n.ID+".continue.conversationFromNode", "must name a run node")
		} else if !reachable(source.ID, n.ID) {
			add("continuation_order", "nodes."+n.ID+".continue.conversationFromNode", "run node must be an ancestor of continuation")
		} else {
			// A valid source dominates the continuation: removing it must make
			// the continuation unreachable from entry. This rejects forward or
			// optional-path selectors that merely become reachable through a cycle.
			seen := map[string]bool{}
			q := []string{entry}
			bypasses := false
			for len(q) > 0 {
				id := q[0]
				q = q[1:]
				if id == source.ID || seen[id] {
					continue
				}
				if id == n.ID {
					bypasses = true
					break
				}
				seen[id] = true
				for _, edge := range adj[id] {
					q = append(q, edge.To)
				}
			}
			if bypasses {
				add("continuation_order", "nodes."+n.ID+".continue.conversationFromNode", "run node must dominate every path to continuation")
			}
		}
	}
}

func validateCycles(nodes map[string]*domain.WorkflowNode, adj map[string][]domain.WorkflowEdge, add func(string, string, string)) {
	index := 0
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	stack := []string{}
	var visit func(string)
	visit = func(v string) {
		indices[v] = index
		low[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, e := range adj[v] {
			w := e.To
			if _, ok := indices[w]; !ok {
				visit(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && indices[w] < low[v] {
				low[v] = indices[w]
			}
		}
		if low[v] != indices[v] {
			return
		}
		component := map[string]bool{}
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			component[w] = true
			if w == v {
				break
			}
		}
		cyclic := len(component) > 1
		if !cyclic {
			for _, e := range adj[v] {
				if e.To == v {
					cyclic = true
				}
			}
		}
		if cyclic {
			for from := range component {
				for _, e := range adj[from] {
					if component[e.To] && e.MaxTraversals <= 0 {
						add("cycle_budget", "edges."+from+"->"+e.To, "every edge inside a cycle requires positive maxTraversals")
					}
				}
			}
		}
	}
	for id := range nodes {
		if _, ok := indices[id]; !ok {
			visit(id)
		}
	}
}
