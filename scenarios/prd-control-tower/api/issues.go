package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Triage-class issue reporting now flows into swarm-manager's backlog over the
// typed BacklogService Connect contract. prd-control-tower files a `fix` item
// tagged with its origin and the target scenario, then surfaces the item's live
// status + queue position back to the user (the feedback contract). The old
// hand-rolled app-issue-tracker HTTP client is gone; the shared generated proto
// types are what keep this consumer from drifting on the wire.

const (
	backlogOriginTag = "origin:prd-control-tower"
	swarmManagerID   = "swarm-manager"
)

var backlogHTTPClient = &http.Client{Timeout: 15 * time.Second}

// resolveSwarmManagerURL resolves the swarm-manager API base URL. Package-level
// so tests can point it at a fake Connect server.
var resolveSwarmManagerURL = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURL(ctx, swarmManagerID, "API_PORT")
}

// resolveBacklogClient builds an inline BacklogService Connect client pointed at
// the locally-resolved swarm-manager API. Constructed per request — matches the
// repo norm (no shared wrapper package); the shared contract is the proto.
func resolveBacklogClient(ctx context.Context) (apiconnect.BacklogServiceClient, error) {
	baseURL, err := resolveSwarmManagerURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve swarm-manager: %w", err)
	}
	return apiconnect.NewBacklogServiceClient(backlogHTTPClient, baseURL), nil
}

// BacklogFeedback is the per-item feedback contract returned to the UI: the
// created/queried backlog item's id, deep link into swarm-manager, live status,
// queue position (items-ahead; null when not pending), priority, and whether the
// create was deduped onto an existing item. No time-based ETA — queue position
// is the honest signal in a deep variable-runtime queue.
type BacklogFeedback struct {
	ItemID        string `json:"item_id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	DeepLink      string `json:"deep_link"`
	Status        string `json:"status"`
	QueuePosition *int32 `json:"queue_position,omitempty"`
	Priority      int    `json:"priority"`
	Deduped       bool   `json:"deduped"`
}

// ScenarioIssueReportRequest defines the payload accepted by POST /issues/report.
// The UI continues to send entity_type/entity_name + selections; we map them
// onto a backlog `fix` item.
type ScenarioIssueReportRequest struct {
	EntityType  string                 `json:"entity_type"`
	EntityName  string                 `json:"entity_name"`
	Source      string                 `json:"source"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    string                 `json:"priority,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Selections  []IssueReportSelection `json:"selections"`
}

// IssueReportSelection captures a single checkbox entry from the UI.
type IssueReportSelection struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Category  string `json:"category"`
	Severity  string `json:"severity"`
	Reference string `json:"reference"`
	Notes     string `json:"notes"`
}

// handleGetScenarioIssuesStatus reads a single backlog item's live status +
// queue position. Query params: kind (default "fix") + name.
func handleGetScenarioIssuesStatus(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind == "" {
		kind = "fix"
	}

	client, err := resolveBacklogClient(r.Context())
	if err != nil {
		respondError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	resp, err := client.GetItem(r.Context(), connect.NewRequest(&apipb.GetBacklogItemRequest{Kind: kind, Name: name}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			respondError(w, "backlog item not found", http.StatusNotFound)
			return
		}
		slog.Warn("backlog get failed", "name", name, "error", err)
		respondError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	respondJSON(w, http.StatusOK, feedbackFromResponse(resp.Msg))
}

// handleSubmitIssueReport files a backlog `fix` item and returns the feedback
// contract (id + deep link + status + queue position + deduped).
func handleSubmitIssueReport(w http.ResponseWriter, r *http.Request) {
	var req ScenarioIssueReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityName = strings.TrimSpace(req.EntityName)
	req.Source = strings.TrimSpace(req.Source)
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.EntityType == "" || req.EntityName == "" {
		respondBadRequest(w, "entity_type and entity_name are required")
		return
	}
	if !isValidEntityType(req.EntityType) {
		respondInvalidEntityType(w)
		return
	}
	if req.Title == "" {
		respondBadRequest(w, "title is required")
		return
	}
	if req.Description == "" {
		respondBadRequest(w, "description is required")
		return
	}
	if len(req.Selections) == 0 {
		respondBadRequest(w, "at least one issue selection is required")
		return
	}

	client, err := resolveBacklogClient(r.Context())
	if err != nil {
		respondError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	createReq := buildBacklogCreateRequest(&req)
	resp, err := client.CreateItem(r.Context(), connect.NewRequest(createReq))
	if err != nil {
		slog.Error("backlog create failed", "entity", req.EntityName, "error", err)
		respondError(w, fmt.Sprintf("failed to submit issue report: %v", err), http.StatusBadGateway)
		return
	}

	respondJSON(w, http.StatusOK, feedbackFromResponse(resp.Msg))
}

// buildBacklogCreateRequest maps the UI report onto a backlog `fix` item.
func buildBacklogCreateRequest(req *ScenarioIssueReportRequest) *apipb.CreateBacklogItemRequest {
	description := buildIssueDescription(req)

	tags := []string{backlogOriginTag, slaClassTag(req.Source)}
	for _, t := range req.Tags {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			tags = append(tags, strings.ToLower(trimmed))
		}
	}
	tags = dedupeStrings(tags)

	priority := determineBacklogPriority(req)
	return &apipb.CreateBacklogItemRequest{
		Name:            req.Title,
		Title:           req.Title,
		Kind:            "fix",
		Description:     &description,
		Priority:        &priority,
		Tags:            tags,
		AcceptanceAllow: []string{fmt.Sprintf("scenarios/%s/**", req.EntityName)},
	}
}

// slaClassTag derives the SLA-class tag from the report source. A user-driven
// report is user-initiated; everything else is treated as auto-detected.
func slaClassTag(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "", "auto", "auto-detected", "scanner", "quality-scanner":
		return "sla:auto-detected"
	default:
		return "sla:user-initiated"
	}
}

// determineBacklogPriority maps the report's severity/priority to the backlog
// 1-10 scale (1 = highest).
func determineBacklogPriority(req *ScenarioIssueReportRequest) int32 {
	switch strings.ToLower(strings.TrimSpace(req.Priority)) {
	case "critical", "blocker", "p0":
		return 1
	case "high", "major", "p1":
		return 3
	case "low", "p3":
		return 8
	case "medium", "p2":
		return 5
	}
	// Derive from the most severe selection.
	best := int32(5)
	for _, sel := range req.Selections {
		switch strings.ToLower(sel.Severity) {
		case "critical", "blocker", "p0":
			return 1
		case "high", "major", "p1":
			if best > 3 {
				best = 3
			}
		case "low", "p3":
			if best == 5 {
				best = 8
			}
		}
	}
	return best
}

// buildIssueDescription renders the report body + selected issues as markdown.
func buildIssueDescription(req *ScenarioIssueReportRequest) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(req.Description))
	if len(req.Selections) > 0 {
		b.WriteString("\n\n## Selected issues\n")
		for _, sel := range req.Selections {
			var parts []string
			if c := strings.TrimSpace(sel.Category); c != "" {
				parts = append(parts, c)
			}
			if s := strings.TrimSpace(sel.Severity); s != "" {
				parts = append(parts, strings.ToUpper(s))
			}
			if t := strings.TrimSpace(sel.Title); t != "" {
				parts = append(parts, t)
			}
			line := strings.Join(parts, " · ")
			if detail := strings.TrimSpace(sel.Detail); detail != "" {
				line = fmt.Sprintf("%s — %s", line, detail)
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(line))
			if ref := strings.TrimSpace(sel.Reference); ref != "" {
				b.WriteString(" (ref: " + ref + ")")
			}
			if notes := strings.TrimSpace(sel.Notes); notes != "" {
				b.WriteString("\n  Notes: " + notes)
			}
			b.WriteString("\n")
		}
	}
	if summary := strings.TrimSpace(req.Summary); summary != "" {
		b.WriteString("\n> " + summary + "\n")
	}
	b.WriteString("\n_Reported via PRD Control Tower._\n")
	return b.String()
}

// feedbackFromResponse maps a BacklogItemResponse into the UI feedback contract.
func feedbackFromResponse(resp *apipb.BacklogItemResponse) BacklogFeedback {
	item := resp.GetItem()
	fb := BacklogFeedback{
		Kind:     item.GetKind(),
		Name:     item.GetName(),
		ItemID:   item.GetKind() + "/" + item.GetName(),
		Status:   item.GetStatus(),
		Priority: int(item.GetPriority()),
		Deduped:  resp.GetDeduped(),
		DeepLink: backlogDeepLink(item.GetKind(), item.GetName()),
	}
	if item.QueuePosition != nil {
		pos := item.GetQueuePosition()
		fb.QueuePosition = &pos
	}
	return fb
}

// backlogDeepLink builds the swarm-manager UI deep link for a backlog item.
func backlogDeepLink(kind, name string) string {
	return fmt.Sprintf("/apps/%s/proxy/backlog/%s/%s", swarmManagerID, kind, name)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
