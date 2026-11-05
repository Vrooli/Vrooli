package sessions

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DesktopFlowClaim is durable admission, not proof of successful execution.
// No flow text or credential is retained. Claims are scoped to the active lease.
type DesktopFlowClaim struct {
	RunID       string    `json:"run_id"`
	Digest      string    `json:"digest"`
	ClaimedAt   time.Time `json:"claimed_at"`
	Steps       uint32    `json:"steps"`
	Disposition string    `json:"disposition"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	Confirmed   uint32    `json:"confirmed"`
}

// ClaimFlow commits before execution begins. Only fresh=true permits execution.
// A duplicate after a crash remains claimed: callers must not restart it.
func (c *DesktopController) ClaimFlow(ctx context.Context, lease DesktopLease, runID, digest string, steps uint32) (claim DesktopFlowClaim, fresh bool, err error) {
	decoded, e := hex.DecodeString(digest)
	if strings.ContainsRune(runID, ':') || steps == 0 || steps > 32 || runID == "" || len(runID) > 120 || e != nil || len(decoded) != 32 || !lease.Control || !c.valid(lease) || c.authority.Authorize(ctx, lease, "act") != nil {
		return claim, false, ErrDesktopAdmission
	}
	ctx, cancel := context.WithDeadline(ctx, lease.ExpiresAt)
	defer cancel()
	err = c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "act") != nil {
			return ErrDesktopAdmission
		}
		if prior, ok := s.FlowClaims[runID]; ok {
			if prior.Digest != digest || prior.Steps != steps {
				return fmt.Errorf("conflicting flow identity")
			}
			claim = prior
			return nil
		}
		if len(s.FlowClaims) >= 128 {
			return fmt.Errorf("session flow limit reached")
		}
		if s.FlowClaims == nil {
			s.FlowClaims = map[string]DesktopFlowClaim{}
		}
		claim = DesktopFlowClaim{RunID: runID, Digest: digest, ClaimedAt: c.now().UTC(), Steps: steps, Disposition: "claimed"}
		s.FlowClaims[runID] = claim
		fresh = true
		return nil
	})
	if err != nil {
		return DesktopFlowClaim{}, false, err
	}
	return claim, fresh, nil
}

// FinishFlow records terminal metadata only. Passing requires every expected
// command receipt to be applied. An incomplete run cannot later become passed.
func (c *DesktopController) FinishFlow(ctx context.Context, lease DesktopLease, runID, digest, disposition string) (DesktopFlowClaim, error) {
	if (disposition != "passed" && disposition != "incomplete") || !lease.Control || !c.valid(lease) || c.authority.Authorize(ctx, lease, "act") != nil {
		return DesktopFlowClaim{}, ErrDesktopAdmission
	}
	ctx, cancel := context.WithDeadline(ctx, lease.ExpiresAt)
	defer cancel()
	var result DesktopFlowClaim
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "act") != nil {
			return ErrDesktopAdmission
		}
		claim, ok := s.FlowClaims[runID]
		if !ok || claim.Digest != digest {
			return ErrDesktopAdmission
		}
		if claim.Disposition != "claimed" {
			if claim.Disposition != disposition {
				return fmt.Errorf("flow terminal disposition conflicts")
			}
			result = claim
			return nil
		}
		var confirmed uint32
		for index := uint32(0); index < claim.Steps; index++ {
			receipt, ok := s.Receipts[runID+":"+strconv.FormatUint(uint64(index), 10)]
			if !ok || receipt.Outcome != "applied" {
				break
			}
			confirmed++
		}
		if disposition == "passed" && confirmed != claim.Steps {
			return fmt.Errorf("flow has unconfirmed commands")
		}
		claim.Disposition = disposition
		claim.Confirmed = confirmed
		claim.FinishedAt = c.now().UTC()
		s.FlowClaims[runID] = claim
		result = claim
		return nil
	})
	if err != nil {
		return DesktopFlowClaim{}, err
	}
	return result, nil
}

func admitFlowCommand(s *DesktopState, id string) error {
	run, indexText, ok := strings.Cut(id, ":")
	if !ok {
		return nil
	}
	claim, ok := s.FlowClaims[run]
	if !ok {
		return nil
	}
	index, err := strconv.ParseUint(indexText, 10, 32)
	if err != nil || strconv.FormatUint(index, 10) != indexText || index >= uint64(claim.Steps) || claim.Disposition != "claimed" {
		return ErrDesktopAdmission
	}
	return nil
}
