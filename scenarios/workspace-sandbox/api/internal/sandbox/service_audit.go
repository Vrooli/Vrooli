package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/types"
)

// service_audit.go: audit-event logging + agent-manager notification.
//
// The audit-event API on Service is two convenience wrappers
// (logAuditEvent for default Source, logAuditEventWithSource for an
// explicit one) that funnel into logAuditEventWith. The latter builds
// the immutable sandbox-state snapshot used for forensic analysis.
//
// notifyAgentManager + resolveAgentManagerURL are HTTP outcalls — kept
// alongside audit because they're triggered from the same approval
// transitions and share the metadata helper.

const (
	metadataAgentManagerRunID = "agent_manager_run_id"
	metadataAgentManagerURL   = "agent_manager_url"
)

func metadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return ""
	}
	switch val := raw.(type) {
	case string:
		return strings.TrimSpace(val)
	case fmt.Stringer:
		return strings.TrimSpace(val.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func (s *Service) resolveAgentManagerURL(ctx context.Context, sandbox *types.Sandbox) (string, error) {
	if s.config.AgentManagerURL != "" {
		return strings.TrimRight(s.config.AgentManagerURL, "/"), nil
	}
	if sandbox != nil {
		if url := metadataString(sandbox.Metadata, metadataAgentManagerURL); url != "" {
			return strings.TrimRight(url, "/"), nil
		}
	}
	resolved, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(resolved, "/"), nil
}

func (s *Service) notifyAgentManager(ctx context.Context, sandbox *types.Sandbox, status, actor string, result *types.ApprovalResult) {
	if !s.config.AgentManagerSyncEnabled {
		return
	}
	runID := metadataString(sandbox.Metadata, metadataAgentManagerRunID)
	if runID == "" {
		return
	}

	baseURL, err := s.resolveAgentManagerURL(ctx, sandbox)
	if err != nil {
		log.Printf("agent-manager sync: failed to resolve agent-manager URL for sandbox %s: %v", sandbox.ID, err)
		return
	}

	if strings.TrimSpace(actor) == "" {
		actor = "workspace-sandbox"
	}

	payload := map[string]interface{}{
		"runId":     runID,
		"sandboxId": sandbox.ID.String(),
		"status":    status,
		"actor":     actor,
	}
	if result != nil {
		payload["applied"] = result.Applied
		payload["remaining"] = result.Remaining
		payload["isPartial"] = result.IsPartial
		payload["commitHash"] = result.CommitHash
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("agent-manager sync: failed to encode payload for sandbox %s: %v", sandbox.ID, err)
		return
	}

	timeout := s.config.AgentManagerSyncTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, baseURL+"/api/v1/runs/"+runID+"/sandbox-sync", bytes.NewReader(body))
	if err != nil {
		log.Printf("agent-manager sync: failed to create request for sandbox %s: %v", sandbox.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("agent-manager sync: request failed for sandbox %s: %v", sandbox.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("agent-manager sync: non-2xx response for sandbox %s: %s", sandbox.ID, strings.TrimSpace(string(respBody)))
	}
}

// logAuditEventWithSource records an audit event with an explicit
// ApprovalSource. Used by the manual-review TTL GC path to stamp
// Source=SourceWorkspaceSandboxGC on system-initiated denials.
func (s *Service) logAuditEventWithSource(ctx context.Context, sandbox *types.Sandbox, eventType, actor, actorType string, source types.ApprovalSource, details map[string]interface{}) {
	s.logAuditEventWith(ctx, sandbox, eventType, actor, actorType, source, details)
}

// logAuditEvent records an audit event with the default
// SourceUnspecified. Most lifecycle/approval call sites use this.
func (s *Service) logAuditEvent(ctx context.Context, sandbox *types.Sandbox, eventType, actor, actorType string, details map[string]interface{}) {
	s.logAuditEventWith(ctx, sandbox, eventType, actor, actorType, types.SourceUnspecified, details)
}

// logAuditEventWith builds the immutable sandbox-state snapshot and
// emits an audit event through the audit.Emitter seam. Failures are
// logged, never surfaced — auditing must not block the operation it
// audits.
//
// [OT-P1-004] Audit Trail Metadata
func (s *Service) logAuditEventWith(ctx context.Context, sandbox *types.Sandbox, eventType, actor, actorType string, source types.ApprovalSource, details map[string]interface{}) {
	sandboxState := map[string]interface{}{
		"id":          sandbox.ID.String(),
		"scopePath":   sandbox.ScopePath,
		"projectRoot": sandbox.ProjectRoot,
		"status":      string(sandbox.Status),
		"owner":       sandbox.Owner,
		"ownerType":   string(sandbox.OwnerType),
		"sizeBytes":   sandbox.SizeBytes,
		"fileCount":   sandbox.FileCount,
		"driver":      sandbox.DriverID,
		"createdAt":   sandbox.CreatedAt.Format(time.RFC3339),
	}

	if sandbox.StoppedAt != nil {
		sandboxState["stoppedAt"] = sandbox.StoppedAt.Format(time.RFC3339)
	}
	if sandbox.ApprovedAt != nil {
		sandboxState["approvedAt"] = sandbox.ApprovedAt.Format(time.RFC3339)
	}
	if sandbox.DeletedAt != nil {
		sandboxState["deletedAt"] = sandbox.DeletedAt.Format(time.RFC3339)
	}

	if sandbox.ErrorMsg != "" {
		sandboxState["errorMessage"] = sandbox.ErrorMsg
	}

	// Service-level events default ActorType to "user" when an actor
	// is named — audit.Emitter would have defaulted to "system",
	// which is wrong for user-driven approvals/rejections. The
	// system/user distinction is what the audit query later groups
	// on, so getting it right here matters.
	if actorType == "" && actor != "" {
		actorType = "user"
	}

	if err := s.audit.Emit(ctx, audit.Event{
		EventType:    eventType,
		SandboxID:    &sandbox.ID,
		Actor:        actor,
		ActorType:    actorType,
		Source:       source,
		Details:      details,
		SandboxState: sandboxState,
	}); err != nil {
		fmt.Printf("warning: failed to log audit event: %v\n", err)
	}
}
