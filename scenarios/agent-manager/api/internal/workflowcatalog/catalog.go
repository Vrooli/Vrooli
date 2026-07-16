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
	"sort"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
)

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
	if len(diagnostics) != 0 {
		sort.SliceStable(diagnostics, func(i, j int) bool {
			if diagnostics[i].Path == diagnostics[j].Path {
				return diagnostics[i].Code < diagnostics[j].Code
			}
			return diagnostics[i].Path < diagnostics[j].Path
		})
		return &Result{Definition: definition, Diagnostics: diagnostics}, nil
	}
	canonical, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("canonicalize workflow definition: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return &Result{Definition: definition, Canonical: canonical, Digest: "sha256:" + hex.EncodeToString(sum[:])}, nil
}

func validate(d *domain.WorkflowDefinition, lookup Lookup) []domain.WorkflowDiagnostic {
	var out []domain.WorkflowDiagnostic
	add := func(code, path, message string) {
		out = append(out, domain.WorkflowDiagnostic{Code: code, Path: path, Message: message})
	}
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
		validateNode(n, path, lookup, add)
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

func validateNode(n *domain.WorkflowNode, path string, lookup Lookup, add func(string, string, string)) {
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
		if strings.TrimSpace(n.Run.PromptTemplate) == "" {
			add("prompt", path+".run.promptTemplate", "prompt template is required")
		}
		validateResultSpec(&n.Run.ResultSpec, path+".run.resultSpec", add)
		bindings = n.Run.Bindings
	case domain.WorkflowNodeContinue:
		if n.Continue == nil {
			add("node_kind", path, "continue payload is required")
			return
		}
		if n.Continue.ConversationFromNode == "" {
			add("continuation", path+".continue.conversationFromNode", "explicit prior run node is required")
		}
		if strings.TrimSpace(n.Continue.PromptTemplate) == "" {
			add("prompt", path+".continue.promptTemplate", "prompt template is required")
		}
		validateResultSpec(&n.Continue.ResultSpec, path+".continue.resultSpec", add)
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
		if n.Wait.Signal == "" || n.Wait.TimeoutSeconds <= 0 {
			add("wait", path+".wait", "signal and finite timeoutSeconds are required")
		}
		if len(n.Wait.PayloadSchema) != 0 {
			n.Wait.PayloadSchema = normalizeSchema(n.Wait.PayloadSchema, path+".wait.payloadSchema", add)
		}
	case domain.WorkflowNodeBranch:
		if n.Branch == nil || strings.TrimSpace(n.Branch.Expression) == "" {
			add("branch", path+".branch.expression", "expression is required")
		}
	case domain.WorkflowNodeJoin:
		if n.Join == nil || (n.Join.Strategy != "all" && n.Join.Strategy != "any" && n.Join.Strategy != "quorum") {
			add("join", path+".join.strategy", "strategy must be all, any, or quorum")
		} else if n.Join.Strategy == "quorum" && n.Join.Quorum <= 0 {
			add("join", path+".join.quorum", "quorum strategy requires a positive quorum")
		}
	case domain.WorkflowNodeEnd:
		if n.End == nil || (n.End.Status != "succeeded" && n.End.Status != "failed") {
			add("end", path+".end.status", "status must be succeeded or failed")
		} else {
			bindings = n.End.Bindings
		}
	default:
		add("node_kind", path+".kind", "unsupported node kind")
	}
	validateBindings(bindings, path, add)
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
		case domain.WorkflowBindingInput, domain.WorkflowBindingAttempts, domain.WorkflowBindingRunResult, domain.WorkflowBindingStructured, domain.WorkflowBindingHandoff, domain.WorkflowBindingSignal, domain.WorkflowBindingCounter:
		default:
			add("binding_source", p+".source", "unsupported journal source")
		}
		if b.Limit <= 0 || b.Limit > MaxBindingLimit {
			add("binding_limit", p+".limit", "must be finite and within limit")
		}
		if b.MaxBytes <= 0 || b.MaxBytes > MaxBindingBytes {
			add("binding_size", p+".maxBytes", "must be finite and within limit")
		}
		if b.RenderAs != "json" && b.RenderAs != "text" {
			add("binding_render", p+".renderAs", "must be json or text")
		}
		if b.MissingPolicy != "error" && b.MissingPolicy != "omit" && b.MissingPolicy != "null" {
			add("binding_missing", p+".missingPolicy", "must be error, omit, or null")
		}
	}
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
