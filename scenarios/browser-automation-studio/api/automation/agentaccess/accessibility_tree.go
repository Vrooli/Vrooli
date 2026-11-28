package agentaccess

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

// AccessibilityTree performs one View-authorized AX observation through the
// existing owned session. User-attached sessions remain unavailable unless a
// separate bridge implementation is later explicitly qualified.
func (b *Broker) AccessibilityTree(ctx context.Context, sessionID string) (*driver.AccessibilityTreeResponse, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return nil, err
	}
	if err := b.lockEntry(e); err != nil {
		return nil, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		return nil, errors.New("accessibility tree is unsupported for selected-tab sessions")
	}
	if !e.profile.View {
		return nil, sessionpermission.ErrDenied
	}
	rows, err := b.Capabilities(ctx, []sessionpermission.Capability{
		sessionpermission.CapabilityObserve,
		sessionpermission.CapabilityAccessibilityTree,
	})
	if err != nil {
		return nil, err
	}
	qualified := false
	for _, row := range rows {
		if string(row.Backend) == e.backend {
			qualified = row.Accepted && row.Qualified
			break
		}
	}
	if !qualified {
		return nil, fmt.Errorf("accessibility tree capability is not live-qualified for backend %q", e.backend)
	}
	response, err := e.session.ObserveAccessibilityTree(ctx, "")
	if err != nil {
		return nil, err
	}
	b.addEvidenceBytes(e, evidencePayloadSize(response))
	b.record(e, TraceEvent{Kind: "observation", OccurredAt: time.Now().UTC(), URL: safeOrigin(response.Tree.URL)})
	if evidenceErr := b.recordActionEvidence(ctx, e, "observe", "completed"); evidenceErr != nil {
		response.EvidenceWarning = evidenceErr.Error()
	}
	return response, nil
}
