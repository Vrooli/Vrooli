// Package memberflow handlers for per-member topics declarations and the
// derived graph / drain-status endpoints.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package memberflow

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for member-flow operations.
type Handlers struct {
	storeDir       string
	knowledgeQuery KnowledgeQuery
	agingOpts      InboxAgingOptions
}

// NewHandlers constructs handlers rooted at the given store directory.
// storeDir should be the absolute path to scenarios/prompt-manager/store/.
//
// Use SetKnowledgeQuery to enable stalled_drain / piling_inbox warnings;
// without it the API returns the pure-Go validation result.
func NewHandlers(storeDir string) *Handlers {
	return &Handlers{storeDir: storeDir}
}

// SetKnowledgeQuery installs a backend used to compute inbox-aging warnings.
// Pass nil to disable.
func (h *Handlers) SetKnowledgeQuery(q KnowledgeQuery, opts InboxAgingOptions) {
	h.knowledgeQuery = q
	h.agingOpts = opts
}

// MemberTopicsResponse is the JSON shape for a single member's declarations.
type MemberTopicsResponse struct {
	Team   string `json:"team"`
	Member string `json:"member"`
	Exists bool   `json:"exists"`
	Topics Topics `json:"topics"`
}

// TeamTopicsResponse aggregates every member's declarations for one team.
type TeamTopicsResponse struct {
	Team    string                 `json:"team"`
	Members []MemberTopicsResponse `json:"members"`
}

// GetMember handles GET /teams/{id}/members/{agentId}/topics.
func (h *Handlers) GetMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	team := vars["id"]
	member := vars["agentId"]
	if team == "" || member == "" {
		writeJSONError(w, http.StatusBadRequest, "team and member are required")
		return
	}
	mt, err := LoadMember(h.storeDir, team, member)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, MemberTopicsResponse{
		Team:   team,
		Member: member,
		Exists: mt.Exists,
		Topics: mt.Topics,
	})
}

// PutMember handles PUT /teams/{id}/members/{agentId}/topics.
// The request body is a Topics document; the handler validates and writes it
// to disk. Empty body == empty Topics ({}), which is a valid "no flow" state.
func (h *Handlers) PutMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	team := vars["id"]
	member := vars["agentId"]
	if team == "" || member == "" {
		writeJSONError(w, http.StatusBadRequest, "team and member are required")
		return
	}

	var t Topics
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
	}
	if err := WriteMember(h.storeDir, team, member, t); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, MemberTopicsResponse{
		Team:   team,
		Member: member,
		Exists: true,
		Topics: t,
	})
}

// GetTeam handles GET /teams/{id}/topics — aggregates all members of one team.
func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	team := mux.Vars(r)["id"]
	if team == "" {
		writeJSONError(w, http.StatusBadRequest, "team is required")
		return
	}
	all, err := LoadTeam(h.storeDir, team)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]MemberTopicsResponse, 0, len(all))
	for _, m := range all {
		out = append(out, MemberTopicsResponse{
			Team:   m.Ref.Team,
			Member: m.Ref.Member,
			Exists: m.Exists,
			Topics: m.Topics,
		})
	}
	writeJSON(w, http.StatusOK, TeamTopicsResponse{Team: team, Members: out})
}

// GraphResponse is the directed-graph view of all member declarations across
// teams. Phase 2 returns nodes and edges only; Phase 3 layers validation
// results on top via /topics/validate.
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode is a single addressable element in the topic flow graph. Member
// nodes carry a Ref; boundary nodes (external producers, decision queues, PoR
// sinks) carry a synthetic Ref with Team="<external>".
type GraphNode struct {
	Kind   string    `json:"kind"` // "member" | "external" | "decision" | "por_file" | "capability_gap" | "skill_proposal" | "backlog" | "knowledge_sink"
	ID     string    `json:"id"`   // unique within the response
	Ref    MemberRef `json:"ref,omitempty"`
	Label  string    `json:"label,omitempty"`
	Topics Topics    `json:"topics,omitempty"` // populated for "member" nodes
}

// GraphEdge is a directed flow from source node to destination node, carrying
// the topic prefix that connects them.
type GraphEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Prefix string `json:"prefix"`
	Kind   string `json:"kind"` // "intake" | "output" | "decision_owned" | "decision_consumed" | "external_producer" | "capability_gap"
}

// GraphWithValidationResponse pairs the directed graph with cross-graph
// validation findings.
type GraphWithValidationResponse struct {
	GraphResponse
	Validation ValidationResult `json:"validation"`
}

// GetGraph handles GET /topics/graph — returns the full directed graph
// derived from all members' topics.json plus cross-graph validation findings.
//
// Optional query params:
//   - team=<name>: filter graph + validation to one team's members
func (h *Handlers) GetGraph(w http.ResponseWriter, r *http.Request) {
	team := r.URL.Query().Get("team")

	all, err := LoadAll(h.storeDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Validation runs against the *full* set so cross-team references
	// resolve correctly, then findings are filtered down to the requested
	// team for the response.
	skillIDs, _ := LoadSkillIDs(h.storeDir)
	skillPaths, _ := LoadSkillPaths(h.storeDir)
	repoRoot := h.repoRoot()
	taxonomies, _ := LoadAllTaxonomies(repoRoot)
	val := Validate(all, ValidationOptions{
		RepoRoot:   repoRoot,
		StoreDir:   h.storeDir,
		SkillIDs:   skillIDs,
		SkillPaths: skillPaths,
		Taxonomies: taxonomies,
	})

	// Layer inbox-aging warnings if a knowledge query is wired in.
	if h.knowledgeQuery != nil {
		extra := EnrichWithDrainStatus(all, h.knowledgeQuery, h.agingOpts)
		val = MergeFindings(val, extra)

		// Cross-check entry topic keys against declared prefixes.
		mismatchExtra := EnrichWithKeyPrefixMismatch(all, h.knowledgeQuery)
		val = MergeFindings(val, mismatchExtra)
	}

	if team != "" {
		filtered := make([]MemberTopics, 0, len(all))
		for _, m := range all {
			if m.Ref.Team == team {
				filtered = append(filtered, m)
			}
		}
		all = filtered

		filteredFindings := make([]Finding, 0, len(val.Findings))
		errs, warns := 0, 0
		for _, f := range val.Findings {
			if f.Member.Team == team {
				filteredFindings = append(filteredFindings, f)
				switch f.Severity {
				case SeverityError:
					errs++
				case SeverityWarning:
					warns++
				}
			}
		}
		val = ValidationResult{Findings: filteredFindings, Errors: errs, Warnings: warns}
	}

	resp := GraphWithValidationResponse{
		GraphResponse: buildGraph(all),
		Validation:    val,
	}
	writeJSON(w, http.StatusOK, resp)
}

// repoRoot resolves the repository root from storeDir
// (.../scenarios/prompt-manager/store -> repo root).
func (h *Handlers) repoRoot() string {
	// store -> prompt-manager -> scenarios -> repo
	return filepath.Clean(filepath.Join(h.storeDir, "..", "..", ".."))
}

// buildGraph turns a flat slice of member declarations into a GraphResponse.
// Pulled out for unit-testability; no I/O.
func buildGraph(members []MemberTopics) GraphResponse {
	resp := GraphResponse{}
	nodeIDs := map[string]bool{}

	addNode := func(n GraphNode) {
		if !nodeIDs[n.ID] {
			resp.Nodes = append(resp.Nodes, n)
			nodeIDs[n.ID] = true
		}
	}

	for _, m := range members {
		memberID := "member:" + m.Ref.String()
		addNode(GraphNode{Kind: "member", ID: memberID, Ref: m.Ref, Label: m.Ref.Member, Topics: m.Topics})

		// External producers feed the member's intake.
		for _, p := range m.Topics.ExternalProducers {
			extID := "external:" + p
			addNode(GraphNode{Kind: "external", ID: extID, Label: p})
			resp.Edges = append(resp.Edges, GraphEdge{From: extID, To: memberID, Prefix: "", Kind: "external_producer"})
		}

		// Intake edges: source members (or external producer) -> this member.
		// At graph-build time we render each intake claim as a "demand"
		// edge from a synthetic prefix node to this member; the actual
		// producer match happens during validation (Phase 3).
		for _, e := range m.Topics.Intake {
			prefixID := "prefix:" + e.Prefix
			addNode(GraphNode{Kind: "knowledge_sink", ID: prefixID, Label: e.Prefix})
			resp.Edges = append(resp.Edges, GraphEdge{From: prefixID, To: memberID, Prefix: e.Prefix, Kind: "intake"})
		}

		// Output edges: this member -> destination prefix (knowledge sink,
		// PoR file, decision queue, etc.).
		for _, e := range m.Topics.Output {
			var destKind, destID string
			switch e.DestinationKind {
			case DestinationPORFile:
				destKind = "por_file"
				if e.DestinationPath != nil {
					destID = "por:" + *e.DestinationPath
				} else {
					destID = "por:<missing>"
				}
				addNode(GraphNode{Kind: destKind, ID: destID, Label: stringPtr(e.DestinationPath)})
			case DestinationDecision:
				destKind = "decision"
				destID = "decision:" + e.Prefix
				addNode(GraphNode{Kind: destKind, ID: destID, Label: e.Prefix})
			case DestinationCapabilityGap:
				destKind = "capability_gap"
				destID = "capability-gap"
				addNode(GraphNode{Kind: destKind, ID: destID, Label: "capability-gap"})
			case DestinationSkillProposal:
				destKind = "skill_proposal"
				destID = "skill-proposal"
				addNode(GraphNode{Kind: destKind, ID: destID, Label: "skill-proposal"})
			case DestinationBacklog:
				destKind = "backlog"
				destID = "backlog"
				addNode(GraphNode{Kind: destKind, ID: destID, Label: "backlog"})
			default:
				destKind = "knowledge_sink"
				destID = "prefix:" + e.Prefix
				addNode(GraphNode{Kind: destKind, ID: destID, Label: e.Prefix})
			}
			resp.Edges = append(resp.Edges, GraphEdge{From: memberID, To: destID, Prefix: e.Prefix, Kind: "output"})
		}

		for _, ctx := range m.Topics.DecisionsOwned {
			id := "decision:" + ctx
			addNode(GraphNode{Kind: "decision", ID: id, Label: ctx})
			resp.Edges = append(resp.Edges, GraphEdge{From: memberID, To: id, Prefix: ctx, Kind: "decision_owned"})
		}
		for _, ctx := range m.Topics.DecisionsConsumed {
			id := "decision:" + ctx
			addNode(GraphNode{Kind: "decision", ID: id, Label: ctx})
			resp.Edges = append(resp.Edges, GraphEdge{From: id, To: memberID, Prefix: ctx, Kind: "decision_consumed"})
		}

		if m.Topics.RaisesCapabilityGaps {
			id := "capability-gap"
			addNode(GraphNode{Kind: "capability_gap", ID: id, Label: "capability-gap"})
			resp.Edges = append(resp.Edges, GraphEdge{From: memberID, To: id, Prefix: "capability-gap", Kind: "capability_gap"})
		}
	}

	return resp
}

func stringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// DrainStatusEntry is one intake-prefix queue snapshot.
type DrainStatusEntry struct {
	Member        MemberRef `json:"member"`
	Prefix        string    `json:"prefix"`
	UnroutedCount int       `json:"unrouted_count"`
	OldestAtRFC   string    `json:"oldest_at,omitempty"`
	OldestAgeSecs int64     `json:"oldest_age_seconds,omitempty"`
}

// DrainStatusResponse is the full per-team (or per-prefix) snapshot.
type DrainStatusResponse struct {
	Entries []DrainStatusEntry `json:"entries"`
	Note    string             `json:"note,omitempty"`
}

// GetDrainStatus handles GET /topics/drain-status — returns per-intake-prefix
// queue depth + age. Returns an empty result with a note when no knowledge
// query is wired in (e.g. test harnesses without a team store).
func (h *Handlers) GetDrainStatus(w http.ResponseWriter, r *http.Request) {
	team := r.URL.Query().Get("team")
	if h.knowledgeQuery == nil {
		writeJSON(w, http.StatusOK, DrainStatusResponse{
			Note: "drain-status backend not wired (KnowledgeQuery is nil)",
		})
		return
	}

	all, err := LoadAll(h.storeDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := h.agingOpts.Now
	if now.IsZero() {
		now = time.Now()
	}

	resp := DrainStatusResponse{}
	for _, m := range all {
		if team != "" && m.Ref.Team != team {
			continue
		}
		for _, in := range m.Topics.Intake {
			entries, qerr := h.knowledgeQuery.ListUnrouted(m.Ref.Team, in.Prefix)
			if qerr != nil {
				resp.Entries = append(resp.Entries, DrainStatusEntry{
					Member: m.Ref,
					Prefix: in.Prefix,
				})
				continue
			}
			entry := DrainStatusEntry{
				Member:        m.Ref,
				Prefix:        in.Prefix,
				UnroutedCount: len(entries),
			}
			if len(entries) > 0 {
				oldest := oldestEntry(entries)
				if !oldest.At.IsZero() {
					entry.OldestAtRFC = oldest.At.UTC().Format(time.RFC3339)
					entry.OldestAgeSecs = int64(now.Sub(oldest.At).Seconds())
				}
			}
			resp.Entries = append(resp.Entries, entry)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
