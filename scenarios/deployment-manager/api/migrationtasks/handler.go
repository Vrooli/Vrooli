// Package migrationtasks files dependency-swap migration tasks into
// swarm-manager's backlog over the typed BacklogService Connect contract.
//
// When a dependency swap is approved, the actual source-code migration must be
// performed by a developer (swaps only touch deployment profiles, never source).
// deployment-manager closes that loop by filing a backlog `fix` item tagged with
// its origin and the target scenario, then surfacing the item's live status +
// queue position back to the user. The shared generated proto types are what keep
// this consumer from drifting on the wire.
package migrationtasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	"deployment-manager/shared"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

const (
	backlogOriginTag = "origin:deployment-manager"
	swarmManagerID   = "swarm-manager"
)

// Handler files and queries migration-task backlog items.
type Handler struct {
	log func(string, map[string]interface{})
	// resolveURL resolves the swarm-manager API base URL. A field (not a
	// package-level var) so tests can point it at a fake Connect server.
	resolveURL func(ctx context.Context) (string, error)
	httpClient *http.Client
}

// NewHandler creates a migration-task handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{
		log: log,
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURL(ctx, swarmManagerID, "API_PORT")
		},
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// client builds an inline BacklogService Connect client pointed at the
// locally-resolved swarm-manager API. Constructed per request — matches the repo
// norm (no shared wrapper package); the shared contract is the proto.
func (h *Handler) client(ctx context.Context) (apiconnect.BacklogServiceClient, error) {
	baseURL, err := h.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve swarm-manager: %w", err)
	}
	return apiconnect.NewBacklogServiceClient(h.httpClient, baseURL), nil
}

// Feedback is the per-item feedback contract returned to the UI: the
// created/queried backlog item's id, deep link into swarm-manager, live status,
// queue position (items-ahead; null when not pending), priority, and whether the
// create was deduped onto an existing item. No time-based ETA — queue position is
// the honest signal in a deep variable-runtime queue.
type Feedback struct {
	ItemID        string `json:"item_id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	DeepLink      string `json:"deep_link"`
	Status        string `json:"status"`
	QueuePosition *int32 `json:"queue_position,omitempty"`
	Priority      int    `json:"priority"`
	Deduped       bool   `json:"deduped"`
}

// ReportRequest is the payload accepted by POST /api/v1/migration-tasks. It maps
// an approved dependency swap onto a backlog `fix` item.
type ReportRequest struct {
	// Scenario is the scenario whose source code must be migrated. The backlog
	// item's acceptance is scoped to scenarios/<scenario>/**.
	Scenario string `json:"scenario"`
	// FromDependency and ToDependency describe the approved swap.
	FromDependency string `json:"from_dependency"`
	ToDependency   string `json:"to_dependency"`
	// ProfileID links the task back to the deployment profile that triggered it.
	ProfileID string `json:"profile_id,omitempty"`
	// Title overrides the auto-generated title when provided.
	Title string `json:"title,omitempty"`
	// Notes captures any migration rationale/context to include in the body.
	Notes string `json:"notes,omitempty"`
}

// Report files a migration-task backlog item and returns the feedback contract.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.JSONError(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	req.Scenario = strings.TrimSpace(req.Scenario)
	req.FromDependency = strings.TrimSpace(req.FromDependency)
	req.ToDependency = strings.TrimSpace(req.ToDependency)
	req.ProfileID = strings.TrimSpace(req.ProfileID)
	req.Title = strings.TrimSpace(req.Title)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.Scenario == "" {
		shared.JSONError(w, "scenario is required", http.StatusBadRequest)
		return
	}
	if req.FromDependency == "" || req.ToDependency == "" {
		shared.JSONError(w, "from_dependency and to_dependency are required", http.StatusBadRequest)
		return
	}

	client, err := h.client(r.Context())
	if err != nil {
		shared.JSONError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	createReq := buildCreateRequest(&req)
	resp, err := client.CreateItem(r.Context(), connect.NewRequest(createReq))
	if err != nil {
		if h.log != nil {
			h.log("migration-task create failed", map[string]interface{}{"scenario": req.Scenario, "error": err.Error()})
		}
		shared.JSONError(w, fmt.Sprintf("failed to file migration task: %v", err), http.StatusBadGateway)
		return
	}

	shared.JSONOK(w, feedbackFromResponse(resp.Msg))
}

// Status reads a single migration-task backlog item's live status + queue
// position. Query params: name (required) + kind (default "fix").
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		shared.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind == "" {
		kind = "fix"
	}

	client, err := h.client(r.Context())
	if err != nil {
		shared.JSONError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	resp, err := client.GetItem(r.Context(), connect.NewRequest(&apipb.GetBacklogItemRequest{Kind: kind, Name: name}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			shared.JSONError(w, "migration task not found", http.StatusNotFound)
			return
		}
		if h.log != nil {
			h.log("migration-task status failed", map[string]interface{}{"name": name, "error": err.Error()})
		}
		shared.JSONError(w, fmt.Sprintf("swarm-manager unavailable: %v", err), http.StatusBadGateway)
		return
	}

	shared.JSONOK(w, feedbackFromResponse(resp.Msg))
}

// buildCreateRequest maps an approved swap onto a backlog `fix` item. These are
// auto migration-task reports, so the SLA class is always auto-detected.
func buildCreateRequest(req *ReportRequest) *apipb.CreateBacklogItemRequest {
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("Migrate %s: %s → %s", req.Scenario, req.FromDependency, req.ToDependency)
	}

	description := buildDescription(req)
	priority := int32(5) // medium — migrations are deliberate, not emergencies
	tags := []string{
		backlogOriginTag,
		"sla:auto-detected",
		"kind:migration",
		"scenario:" + strings.ToLower(req.Scenario),
	}

	return &apipb.CreateBacklogItemRequest{
		Name:            title,
		Title:           title,
		Kind:            "fix",
		Description:     &description,
		Priority:        &priority,
		Tags:            tags,
		AcceptanceAllow: []string{fmt.Sprintf("scenarios/%s/**", req.Scenario)},
	}
}

// buildDescription renders the migration context as markdown.
func buildDescription(req *ReportRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Dependency swap approved for **%s** — source-code migration required.\n\n", req.Scenario)
	fmt.Fprintf(&b, "- **From:** %s\n", req.FromDependency)
	fmt.Fprintf(&b, "- **To:** %s\n", req.ToDependency)
	if req.ProfileID != "" {
		fmt.Fprintf(&b, "- **Deployment profile:** %s\n", req.ProfileID)
	}
	if req.Notes != "" {
		b.WriteString("\n## Notes\n")
		b.WriteString(req.Notes)
		b.WriteString("\n")
	}
	b.WriteString("\n_Filed automatically by deployment-manager when the swap was approved. Swaps only touch deployment profiles; the source migration must be done by a developer._\n")
	return b.String()
}

// feedbackFromResponse maps a BacklogItemResponse into the UI feedback contract.
func feedbackFromResponse(resp *apipb.BacklogItemResponse) Feedback {
	item := resp.GetItem()
	fb := Feedback{
		Kind:     item.GetKind(),
		Name:     item.GetName(),
		ItemID:   item.GetKind() + "/" + item.GetName(),
		Status:   item.GetStatus(),
		Priority: int(item.GetPriority()),
		Deduped:  resp.GetDeduped(),
		DeepLink: deepLink(item.GetKind(), item.GetName()),
	}
	if item.QueuePosition != nil {
		pos := item.GetQueuePosition()
		fb.QueuePosition = &pos
	}
	return fb
}

// deepLink builds the swarm-manager UI deep link for a backlog item.
func deepLink(kind, name string) string {
	return fmt.Sprintf("/apps/%s/proxy/backlog/%s/%s", swarmManagerID, kind, name)
}
