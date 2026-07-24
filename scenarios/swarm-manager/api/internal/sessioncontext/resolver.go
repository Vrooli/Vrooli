// Package sessioncontext resolves composer-selected Swarm Manager records into
// bounded agent-session message context snapshots.
package sessioncontext

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/operations"
	"swarm-manager/internal/runtimepaths"
)

type Resolver struct {
	scenarioRoot string
	scenariosDir string
	sessionStore agentsessions.Store
	briefings    operationsBriefingBuilder
	snapshots    operationsSnapshotProvider
}

type operationsBriefingBuilder interface {
	Build(ctx context.Context, filters operations.Filters) (*operations.OperationsBriefing, error)
}

// operationsSnapshotProvider supplies the cached, ranked goal snapshot
// that augments the swarm_operations startup brief. Optional: when nil, the
// brief omits the ranked-goals section and falls back to the activity
// briefing alone.
type operationsSnapshotProvider interface {
	GetSnapshot(ctx context.Context) (*operations.OperationsSnapshot, error)
}

// SetSnapshotBuilder wires the operations snapshot provider after
// construction. Kept separate from NewResolver so the snapshot builder (which
// depends on the overview service) can be injected without widening the
// constructor's variadic briefings parameter.
func (r *Resolver) SetSnapshotBuilder(p operationsSnapshotProvider) {
	r.snapshots = p
}

func NewResolver(scenarioRoot, scenariosDir string, sessionStore agentsessions.Store, briefings ...operationsBriefingBuilder) *Resolver {
	var briefingBuilder operationsBriefingBuilder
	if len(briefings) > 0 {
		briefingBuilder = briefings[0]
	}
	return &Resolver{
		scenarioRoot: scenarioRoot,
		scenariosDir: scenariosDir,
		sessionStore: sessionStore,
		briefings:    briefingBuilder,
	}
}

func (r *Resolver) ResolveSessionMessageContext(ctx context.Context, refs []agentsessions.ContextRef, limits agentsessions.ContextLimits) ([]agentsessions.ContextItem, error) {
	items := make([]agentsessions.ContextItem, 0, len(refs))
	for _, ref := range refs {
		item, err := r.resolve(ctx, ref, limits)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Resolver) resolve(ctx context.Context, ref agentsessions.ContextRef, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	switch ref.Type {
	case agentsessions.ContextBacklogItem:
		return r.resolveBacklogItem(ref.Ref, limits)
	case agentsessions.ContextCapture:
		return r.resolveJSONFile(ref, filepath.Join(r.scenarioRoot, "captures", ref.Ref, "capture.json"), "capture", "capture/"+ref.Ref, limits)
	case agentsessions.ContextScenario:
		return r.resolveScenario(ref.Ref, limits)
	case agentsessions.ContextSession:
		return r.resolveSession(ref.Ref, limits)
	case agentsessions.ContextOperationsBriefing:
		return r.resolveOperationsBriefing(ctx, ref.Ref, limits)
	case agentsessions.ContextStartupBrief:
		kind, err := kindForStartupBriefRef(ref.Ref)
		if err != nil {
			return agentsessions.ContextItem{}, err
		}
		return r.ResolveSessionStartupBrief(ctx, kind, limits)
	case agentsessions.ContextExecution:
		return r.resolveJSONFile(ref, filepath.Join(r.scenarioRoot, "executions", ref.Ref, "execution.json"), "execution", "execution-record/"+ref.Ref, limits)
	case agentsessions.ContextAgentActivity:
		return r.resolveJSONListEntry(ref, filepath.Join(r.scenarioRoot, "state", "agent-activities.json"), "activity_id", "agent activity", "agent-activity/"+ref.Ref, limits)
	case agentsessions.ContextOperatingMode:
		return agentsessions.ContextItem{
			Type:    ref.Type,
			Ref:     ref.Ref,
			Title:   titleFromRef(ref.Ref),
			Summary: "Operating mode selected by the operator: " + ref.Ref,
			NodeID:  "operatingMode/" + ref.Ref,
		}, nil
	case agentsessions.ContextPlanDependencyCycles:
		return resolvePlanDependencyCycles(ref.Ref, limits)
	case agentsessions.ContextPlanEta:
		return resolvePlanEta(ref.Ref, limits)
	case agentsessions.ContextGoal:
		return r.resolveGoal(ref, limits)
	default:
		return agentsessions.ContextItem{}, fmt.Errorf("%w: unsupported context type", agentsessions.ErrValidation)
	}
}

func (r *Resolver) resolveGoal(ref agentsessions.ContextRef, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	// Goal records are durable runtime data. Keep the local root first so
	// resolver tests and explicit injected roots remain isolated, then use the
	// production storage resolver when the scenario source tree is supplied.
	path := filepath.Join(r.scenarioRoot, "goals", ref.Ref, "goal.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if dataRoot, dataErr := runtimepaths.DataPath(""); dataErr == nil {
			path = filepath.Join(dataRoot, "goals", ref.Ref, "goal.json")
		}
	}
	return r.resolveJSONFile(ref, path, "goal", "goal/"+ref.Ref, limits)
}

func (r *Resolver) resolveOperationsBriefing(ctx context.Context, ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	if strings.TrimSpace(ref) != agentsessions.OperationsBriefingLatestRef {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: operations briefing ref must be %s", agentsessions.ErrValidation, agentsessions.OperationsBriefingLatestRef)
	}
	if r.briefings == nil {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: operations briefing context is unavailable", agentsessions.ErrValidation)
	}
	briefing, err := r.briefings.Build(ctx, operations.Filters{})
	if err != nil {
		return agentsessions.ContextItem{}, err
	}
	metadata := map[string]any{
		"generated_at":          briefing.GeneratedAt.Format(time.RFC3339),
		"window_seconds":        briefing.WindowSeconds,
		"active_activity_count": briefing.Summary.ActiveActivityCount,
		"needs_attention_count": len(briefing.NeedsAttention),
	}
	data, _ := json.Marshal(metadata)
	return agentsessions.ContextItem{
		Type:         agentsessions.ContextOperationsBriefing,
		Ref:          agentsessions.OperationsBriefingLatestRef,
		Title:        "Current operations briefing",
		Summary:      truncate(formatOperationsBriefingSummary(briefing), limits.MaxSummaryRunes),
		NodeID:       "/operations",
		MetadataJSON: string(data),
	}, nil
}

func formatOperationsBriefingSummary(briefing *operations.OperationsBriefing) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Generated: %s. Window: %ds. Active: %d. Recently finished: %d. Queue: %d/%d. Active goals: %d. Blocked items: %d. Active sessions: %d.\n",
		briefing.GeneratedAt.Format(time.RFC3339),
		briefing.WindowSeconds,
		briefing.Summary.ActiveActivityCount,
		briefing.Summary.RecentlyFinishedCount,
		briefing.Summary.QueueDepth,
		briefing.Summary.MaxQueueDepth,
		briefing.Summary.ActiveGoals,
		briefing.Summary.BlockedItems,
		briefing.Summary.ActiveSessions,
	)
	if len(briefing.Summary.SaturatedLanes) > 0 {
		fmt.Fprintf(&b, "Saturated lanes: %s.\n", strings.Join(briefing.Summary.SaturatedLanes, ", "))
	}
	if len(briefing.NeedsAttention) > 0 {
		b.WriteString("Needs attention:\n")
		for _, item := range briefing.NeedsAttention {
			fmt.Fprintf(&b, "- %s [%s]: %s", item.Title, item.Severity, item.Reason)
			if item.Command != "" {
				fmt.Fprintf(&b, " (%s)", item.Command)
			}
			b.WriteString("\n")
		}
	}
	if len(briefing.ActiveWork) > 0 {
		b.WriteString("Active work:\n")
		for _, item := range briefing.ActiveWork {
			fmt.Fprintf(&b, "- %s [%s/%s] %s", firstNonEmpty(item.OwnerTitle, item.OwnerName, item.ActivityID), item.Lane, item.Status, firstNonEmpty(item.RunID, item.ActivityID))
			if item.Mode != "" || item.Phase != "" {
				fmt.Fprintf(&b, " mode=%s phase=%s", item.Mode, item.Phase)
			}
			b.WriteString("\n")
		}
	}
	if len(briefing.RecommendedNextActions) > 0 {
		b.WriteString("Recommended next actions:\n")
		for _, action := range briefing.RecommendedNextActions {
			fmt.Fprintf(&b, "- %s: %s", action.Label, action.Reason)
			if action.Command != "" {
				fmt.Fprintf(&b, " (%s)", action.Command)
			}
			b.WriteString("\n")
		}
	}
	if len(briefing.DrillDownCommands) > 0 {
		b.WriteString("Drill-down commands:\n")
		for _, command := range briefing.DrillDownCommands {
			fmt.Fprintf(&b, "- %s: %s\n", command.Label, command.Command)
		}
	}
	if len(briefing.Warnings) > 0 {
		fmt.Fprintf(&b, "Warnings: %s.\n", strings.Join(briefing.Warnings, "; "))
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (r *Resolver) resolveBacklogItem(ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: backlog item ref must be kind/name", agentsessions.ErrValidation)
	}
	// Backlog FileStore stores kind directories directly under the runtime data
	// root (for example, <data>/fix/item/spec.json); unlike source assets there
	// is no intermediate "backlog" directory.
	path := filepath.Join(r.scenarioRoot, parts[0], parts[1], "spec.json")
	return r.resolveJSONFile(agentsessions.ContextRef{Type: agentsessions.ContextBacklogItem, Ref: ref}, path, "backlog item", "backlog-item/"+ref, limits)
}

// planEtaBand mirrors the client PlanEtaBandData fields that are meaningful for
// a text summary. The composer serializes this into the plan_eta context ref.
type planEtaBand struct {
	P50Label       string `json:"p50Label"`
	P80Label       string `json:"p80Label"`
	BasisLabel     string `json:"basisLabel"`
	Confidence     string `json:"confidence"`
	RemainingItems int    `json:"remainingItems"`
	LaneCapacity   int    `json:"laneCapacity"`
}

func resolvePlanDependencyCycles(ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	var cycles []string
	if err := json.Unmarshal([]byte(ref), &cycles); err != nil || len(cycles) == 0 {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: plan dependency cycles ref must be a non-empty JSON array of chains", agentsessions.ErrValidation)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The plan has %d dependency %s that block a clean execution order:\n", len(cycles), plural(len(cycles), "cycle", "cycles"))
	for _, cycle := range cycles {
		fmt.Fprintf(&b, "- %s\n", strings.TrimSpace(cycle))
	}
	return agentsessions.ContextItem{
		Type:    agentsessions.ContextPlanDependencyCycles,
		Ref:     ref,
		Title:   fmt.Sprintf("Dependency cycles (%d)", len(cycles)),
		Summary: truncate(b.String(), limits.MaxSummaryRunes),
		NodeID:  "/plan",
	}, nil
}

func resolvePlanEta(ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	var eta planEtaBand
	if err := json.Unmarshal([]byte(ref), &eta); err != nil || strings.TrimSpace(eta.P50Label) == "" {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: plan ETA ref must be a JSON object with a p50 label", agentsessions.ErrValidation)
	}
	summary := fmt.Sprintf(
		"Estimated completion band: p50 %s, p80 %s. Remaining: %d %s across %d execute %s. Confidence: %s (%s).",
		eta.P50Label, eta.P80Label,
		eta.RemainingItems, plural(eta.RemainingItems, "item", "items"),
		eta.LaneCapacity, plural(eta.LaneCapacity, "lane", "lanes"),
		firstNonEmpty(eta.Confidence, "unknown"), firstNonEmpty(eta.BasisLabel, "priors only"),
	)
	return agentsessions.ContextItem{
		Type:    agentsessions.ContextPlanEta,
		Ref:     ref,
		Title:   fmt.Sprintf("Plan ETA %s–%s", eta.P50Label, eta.P80Label),
		Summary: truncate(summary, limits.MaxSummaryRunes),
		NodeID:  "/plan",
	}, nil
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func (r *Resolver) resolveScenario(ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	for _, name := range []string{"scenario.json", "service.json"} {
		item, err := r.resolveJSONFile(agentsessions.ContextRef{Type: agentsessions.ContextScenario, Ref: ref}, filepath.Join(r.scenariosDir, ref, name), "scenario", "scenario/"+ref, limits)
		if err == nil {
			return item, nil
		}
	}
	return agentsessions.ContextItem{}, fmt.Errorf("%w: scenario context %q not found", agentsessions.ErrValidation, ref)
}

func (r *Resolver) resolveSession(ref string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	if r.sessionStore == nil {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: session context is unavailable", agentsessions.ErrValidation)
	}
	session, err := r.sessionStore.LoadSession(ref)
	if err != nil {
		return agentsessions.ContextItem{}, err
	}
	summary := fmt.Sprintf("Kind: %s. Status: %s. Messages: %d. Proposals: %d. Artifacts: %d.", session.Kind, session.Status, len(session.Messages), len(session.Proposals), len(session.Artifacts))
	if len(session.Messages) > 0 {
		last := session.Messages[len(session.Messages)-1]
		if strings.TrimSpace(last.Content) != "" {
			summary += " Last message: " + truncate(last.Content, limits.MaxSummaryRunes/2)
		}
	}
	return agentsessions.ContextItem{
		Type:    agentsessions.ContextSession,
		Ref:     ref,
		Title:   session.Title,
		Summary: truncate(summary, limits.MaxSummaryRunes),
		NodeID:  "/sessions/" + ref,
	}, nil
}

func (r *Resolver) resolveJSONFile(ref agentsessions.ContextRef, path, label, nodeID string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	var payload map[string]any
	if err := readJSON(path, &payload); err != nil {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: %s context %q not found", agentsessions.ErrValidation, label, ref.Ref)
	}
	return itemFromMap(ref, payload, nodeID, limits), nil
}

func (r *Resolver) resolveJSONListEntry(ref agentsessions.ContextRef, path, idField, label, nodeID string, limits agentsessions.ContextLimits) (agentsessions.ContextItem, error) {
	var payload []map[string]any
	if err := readJSON(path, &payload); err != nil {
		return agentsessions.ContextItem{}, fmt.Errorf("%w: %s context %q not found", agentsessions.ErrValidation, label, ref.Ref)
	}
	for _, entry := range payload {
		if stringValue(entry, idField) == ref.Ref || stringValue(entry, "id") == ref.Ref {
			return itemFromMap(ref, entry, nodeID, limits), nil
		}
	}
	return agentsessions.ContextItem{}, fmt.Errorf("%w: %s context %q not found", agentsessions.ErrValidation, label, ref.Ref)
}

func itemFromMap(ref agentsessions.ContextRef, payload map[string]any, nodeID string, limits agentsessions.ContextLimits) agentsessions.ContextItem {
	title := firstString(payload, "title", "name", "id", "text")
	if title == "" {
		title = titleFromRef(ref.Ref)
	}
	summary := firstString(payload, "description", "summary", "text", "note", "status")
	if summary == "" {
		summary = compactJSON(payload)
	}
	metadata := map[string]any{}
	for _, key := range []string{"status", "kind", "priority", "updated", "created"} {
		if value, ok := payload[key]; ok {
			metadata[key] = value
		}
	}
	// Goal mutation proposals use the durable Updated value as their optimistic
	// concurrency base_version. Expose that value under its proposal-contract
	// name as well as the generic metadata field so an agent never has to infer
	// which snapshot field is required by the envelope.
	if ref.Type == agentsessions.ContextGoal {
		if updated, ok := payload["updated"]; ok {
			metadata["base_version"] = updated
		}
	}
	metadataJSON := ""
	if len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err == nil {
			metadataJSON = string(data)
		}
	}
	return agentsessions.ContextItem{
		Type:         ref.Type,
		Ref:          ref.Ref,
		Title:        truncate(title, 160),
		Summary:      truncate(summary, limits.MaxSummaryRunes),
		NodeID:       nodeID,
		MetadataJSON: metadataJSON,
	}
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringValue(payload, key)); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}

func compactJSON(payload map[string]any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

func titleFromRef(ref string) string {
	title := strings.ReplaceAll(ref, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "/", " / ")
	return strings.TrimSpace(title)
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
