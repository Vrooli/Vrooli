// Read receipts: "has the calling agent run read this skill?"
//
// Why this exists: scenario CLIs remind an agent to read the skill that governs
// a command (for example Notification Hub's decision-writing skill) when the
// agent has not read it. The answer must come from verified identity, never
// from a caller-supplied run ID, so a caller can only ask about itself.
package skills

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Read receipt statuses. They are part of the GetSkillReadReceipt contract.
const (
	ReceiptRead        = "read"
	ReceiptNotRead     = "not_read"
	ReceiptUnverified  = "unverified"
	ReceiptUnavailable = "unavailable"
)

// ErrReceiptSkillRequired rejects a receipt request without a skill ID.
var ErrReceiptSkillRequired = errors.New("skill_id is required")

// ReadReceipt is the caller's own read of one skill.
type ReadReceipt struct {
	SkillID     string
	Status      string
	RunID       string
	FirstReadAt time.Time
	LastReadAt  time.Time
	ReadCount   int
	Reason      string
}

// SkillReadReceipt verifies the presented identity token and reports whether
// that run read skillID through Prompt Manager. It never fails because identity
// or storage is unavailable; it reports that state so callers stay fail-open.
func (h *Handlers) SkillReadReceipt(ctx context.Context, identityToken, skillID string) (ReadReceipt, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return ReadReceipt{}, ErrReceiptSkillRequired
	}
	receipt := ReadReceipt{SkillID: skillID}
	if strings.TrimSpace(identityToken) == "" {
		receipt.Status, receipt.Reason = ReceiptUnverified, "no agent identity token was presented"
		return receipt, nil
	}
	if h.identityVerifier == nil {
		receipt.Status, receipt.Reason = ReceiptUnverified, "identity verification is not configured"
		return receipt, nil
	}
	identity, err := h.identityVerifier.VerifyWorkflowIdentity(ctx, identityToken)
	if err != nil || identity == nil || strings.TrimSpace(identity.RunID) == "" {
		receipt.Status, receipt.Reason = ReceiptUnverified, "the agent identity token could not be verified"
		return receipt, nil
	}
	receipt.RunID = identity.RunID
	if h.readRecorder == nil || h.readRecorder.reads == nil {
		receipt.Status, receipt.Reason = ReceiptUnavailable, "read receipts are not recorded on this server"
		return receipt, nil
	}
	stored, found, err := h.readRecorder.reads.WithContext(ctx).Receipt(identity.RunID, skillID)
	if err != nil {
		receipt.Status, receipt.Reason = ReceiptUnavailable, "the read receipt store could not be read"
		return receipt, nil
	}
	if !found {
		receipt.Status = ReceiptNotRead
		return receipt, nil
	}
	receipt.Status = ReceiptRead
	receipt.FirstReadAt, receipt.LastReadAt, receipt.ReadCount = stored.FirstReadAt, stored.LastReadAt, stored.ReadCount
	return receipt, nil
}
