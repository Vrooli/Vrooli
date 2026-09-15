package continuity

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	continuityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity"
	continuityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity/continuity_v1connect"
)

const GroupName = "continuity"

type handlers struct {
	client continuityconnect.ContinuityServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, _ := cliapp.NewConnectHTTPClient(core)
	// Keep the generated client URL relative. The CLI core resolves relative
	// Connect paths at request time, after global flags such as --api-base have
	// been parsed. Capturing APIRootBase here would make a stale saved config
	// defeat an explicit per-invocation override.
	h := &handlers{client: continuityconnect.NewContinuityServiceClient(httpClient, "")}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ContinuityService.Integrity":   h.integrity,
		"ContinuityService.Reconcile":   h.reconcile,
		"ContinuityService.ListCatalog": h.listCatalog,
		"ContinuityService.Search":      h.search,
		"inspect":                       h.inspect,
		"ContinuityService.GetReceipt":  h.receipt,
		"ContinuityService.Rollback":    h.rollback,
		"ContinuityService.Publish":     h.publish,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("continuity: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) publish(ctx cliapp.RunContext) error {
	limit, err := parseIntFlag(ctx.Flag("limit"))
	if err != nil {
		return fmt.Errorf("--limit: %w", err)
	}
	resp, err := h.client.Publish(context.Background(), connect.NewRequest(&continuityv1.PublishRequest{Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("continuity publish", err, nil)
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("attempted=%d published=%d failed=%d next=%d", resp.Msg.GetAttempted(), resp.Msg.GetPublished(), resp.Msg.GetFailed(), resp.Msg.GetNext())}, ResultsHeading: "Publication", Results: []string{"Agent Manager publication is asynchronous and retry-safe."}}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func (h *handlers) rollback(ctx cliapp.RunContext) error {
	manifestHash, operationID := ctx.Flag("manifest-hash"), ctx.Flag("operation-id")
	if manifestHash == "" || operationID == "" {
		return fmt.Errorf("--manifest-hash and --operation-id are required")
	}
	resp, err := h.client.Rollback(context.Background(), connect.NewRequest(&continuityv1.RollbackRequest{ManifestHash: manifestHash, OperationId: operationID}))
	if err != nil {
		return cliapp.WrapAPIError("continuity rollback", err, nil)
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("manifest=%s receipt=%s status=%s", resp.Msg.GetManifestHash(), resp.Msg.GetReceiptId(), resp.Msg.GetReceiptStatus())}, ResultsHeading: "Rollback", Results: []string{"catalog projection restored; durable evidence was not modified"}}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func (h *handlers) receipt(ctx cliapp.RunContext) error {
	operationID := ctx.Flag("operation-id")
	if operationID == "" {
		return fmt.Errorf("--operation-id is required")
	}
	resp, err := h.client.GetReceipt(context.Background(), connect.NewRequest(&continuityv1.GetReceiptRequest{OperationId: operationID}))
	if err != nil {
		return cliapp.WrapAPIError("continuity receipt", err, nil)
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("%s: %s", resp.Msg.GetOperationId(), resp.Msg.GetStatus())}, ResultsHeading: "Receipt", Results: []string{fmt.Sprintf("session: %s", resp.Msg.GetSessionId()), fmt.Sprintf("command: %s", resp.Msg.GetCommand()), fmt.Sprintf("transition: %s -> %s", resp.Msg.GetFromState(), resp.Msg.GetToState()), fmt.Sprintf("created_at: %s", resp.Msg.GetCreatedAt()), fmt.Sprintf("completed_at: %s", resp.Msg.GetCompletedAt()), fmt.Sprintf("error_code: %s", resp.Msg.GetErrorCode())}}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := ctx.Flag("query")
	if query == "" && ctx.Flag("session-id") == "" && ctx.Flag("agent-session-id") == "" && ctx.Flag("title") == "" && ctx.Flag("topic") == "" && ctx.Flag("agent") == "" && ctx.Flag("cwd") == "" && ctx.Flag("after") == "" && (ctx.Flag("state") == "" || ctx.Flag("state") == "any") {
		return fmt.Errorf("at least one search query or filter is required")
	}
	return h.searchQuery(ctx, query)
}

func (h *handlers) inspect(ctx cliapp.RunContext) error {
	query := optionalFlag(ctx, "session-id")
	threadID := optionalFlag(ctx, "thread-id")
	if query == "" {
		query = threadID
	}
	if query == "" {
		return fmt.Errorf("--session-id or --thread-id is required")
	}
	return h.searchQueryWithAgentSession(ctx, query, threadID)
}

func (h *handlers) searchQuery(ctx cliapp.RunContext, query string) error {
	return h.searchQueryWithAgentSession(ctx, query, optionalFlag(ctx, "agent-session-id"))
}

func (h *handlers) searchQueryWithAgentSession(ctx cliapp.RunContext, query, agentSessionID string) error {
	limit, err := parseIntFlag(optionalFlag(ctx, "limit"))
	if err != nil {
		return fmt.Errorf("--limit: %w", err)
	}
	resp, err := h.client.Search(context.Background(), connect.NewRequest(&continuityv1.SearchRequest{Query: query, CreatedAfter: optionalFlag(ctx, "after"), LifecycleState: optionalFlag(ctx, "state"), AgentType: optionalFlag(ctx, "agent"), Cwd: optionalFlag(ctx, "cwd"), SessionId: optionalFlag(ctx, "session-id"), AgentSessionId: agentSessionID, Title: optionalFlag(ctx, "title"), TopicSummary: optionalFlag(ctx, "topic"), Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("continuity search", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetMatches()))
	for _, m := range resp.Msg.GetMatches() {
		rows = append(rows, fmt.Sprintf("%s %s [%s] %s", m.GetSessionId(), m.GetLifecycleState(), m.GetCreatedAt(), m.GetExcerpt()))
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("matches=%d sessions=%d truncated=%t", resp.Msg.GetTotalMatches(), resp.Msg.GetDistinctSessions(), resp.Msg.GetTruncated())}, ResultsHeading: "Conversation matches", Results: rows}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func optionalFlag(ctx cliapp.RunContext, name string) string {
	if !ctx.FlagDeclared(name) {
		return ""
	}
	return ctx.Flag(name)
}

func (h *handlers) listCatalog(ctx cliapp.RunContext) error {
	resp, err := h.client.ListCatalog(context.Background(), connect.NewRequest(&continuityv1.ListCatalogRequest{LifecycleState: ctx.Flag("state")}))
	if err != nil {
		return cliapp.WrapAPIError("continuity catalog", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetRecords()))
	for _, record := range resp.Msg.GetRecords() {
		rows = append(rows, fmt.Sprintf("%s %s [%s] %s", record.GetSessionId(), record.GetLifecycleState(), record.GetAgentType(), record.GetCurrentTitle()))
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("Canonical records: %d", len(rows))}, ResultsHeading: "Conversation catalog", Results: rows}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func (h *handlers) integrity(ctx cliapp.RunContext) error {
	resp, err := h.client.Integrity(context.Background(), connect.NewRequest(&continuityv1.IntegrityRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("continuity integrity", err, nil)
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("Continuity generation: %s", resp.Msg.GetGeneration())}, ResultsHeading: "Integrity", Results: []string{
		fmt.Sprintf("sessions: %d", resp.Msg.GetSessions()), fmt.Sprintf("conversation_sessions: %d", resp.Msg.GetConversationSessions()), fmt.Sprintf("conversation_events: %d", resp.Msg.GetConversationEvents()), fmt.Sprintf("checkpoints: %d", resp.Msg.GetCheckpoints()), fmt.Sprintf("workspace_panes: %d", resp.Msg.GetWorkspacePanes()), fmt.Sprintf("orphans: conversations=%d checkpoints=%d panes=%d", resp.Msg.GetOrphanConversations(), resp.Msg.GetOrphanCheckpoints(), resp.Msg.GetOrphanWorkspacePanes()),
		fmt.Sprintf("event_content_hash: %s", resp.Msg.GetEventContentHash()),
	}}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func (h *handlers) reconcile(ctx cliapp.RunContext) error {
	apply := ctx.BoolFlag("apply")
	offset, err := parseIntFlag(ctx.Flag("offset"))
	if err != nil {
		return fmt.Errorf("--offset: %w", err)
	}
	batchSize, err := parseIntFlag(ctx.Flag("batch-size"))
	if err != nil {
		return fmt.Errorf("--batch-size: %w", err)
	}
	resp, err := h.client.Reconcile(context.Background(), connect.NewRequest(&continuityv1.ReconcileRequest{Apply: apply, Generation: ctx.Flag("generation"), OperationId: ctx.Flag("operation-id"), ManifestHash: ctx.Flag("manifest-hash"), Offset: offset, BatchSize: batchSize}))
	if err != nil {
		return cliapp.WrapAPIError("continuity reconcile", err, nil)
	}
	rows := make([]string, 0, len(resp.Msg.GetItems()))
	for _, item := range resp.Msg.GetItems() {
		rows = append(rows, fmt.Sprintf("%s %s (%s)", item.GetAction(), item.GetSessionId(), item.GetReason()))
	}
	r := cliapp.ListReport{Summary: []string{fmt.Sprintf("observations=%d mutations=%d applied=%t manifest=%s next_offset=%d complete=%t", resp.Msg.GetObservations(), resp.Msg.GetMutations(), resp.Msg.GetApplied(), resp.Msg.GetManifestHash(), resp.Msg.GetNextOffset(), resp.Msg.GetComplete())}, ResultsHeading: "Reconciliation", Results: rows}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), r)
	}
	return cliapp.RenderListReport(ctx.Stdout(), r)
}

func parseIntFlag(value string) (int32, error) {
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("must be a non-negative integer")
	}
	return int32(n), nil
}
