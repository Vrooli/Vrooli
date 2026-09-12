package tiered

import (
	"context"
	"errors"

	"audio-tools/internal/logx"
)

// CredentialRoute builds the common BYOK -> Vrooli -> Local eligibility
// policy. Capability packages provide their request-specific credential
// readers without making the generic coordinator depend on request shapes.
func CredentialRoute[Req any](hasBYOK, hasVrooli func(Req) bool) func(Slot, Req) bool {
	return func(slot Slot, req Req) bool {
		switch slot {
		case SlotBYOK:
			return hasBYOK(req)
		case SlotVrooli:
			return hasVrooli(req)
		default:
			return true
		}
	}
}

// CredentialTerminal builds the shared terminal-error policy: invalid BYOK
// credentials and exhausted Vrooli credits must be reported, never masked by
// falling through to another tier.
func CredentialTerminal(unknownBYOK, missingBYOK, insufficientCredits error) func(Slot, error) bool {
	return func(slot Slot, err error) bool {
		switch slot {
		case SlotBYOK:
			return errors.Is(err, unknownBYOK) || errors.Is(err, missingBYOK)
		case SlotVrooli:
			return errors.Is(err, insufficientCredits)
		default:
			return false
		}
	}
}

// FallbackLogger returns an optional structured fallback hook shared by all
// capability chains. A nil logger leaves coordinator execution allocation-free.
func FallbackLogger(capability string, logger logx.Logger) func(context.Context, FallbackEvent) {
	if logger == nil {
		return nil
	}
	return func(_ context.Context, event FallbackEvent) {
		logger.Printf("event=tier_fallback capability=%s from_tier=%s to_tier=%s reason=%q", capability, event.From.String(), event.To.String(), event.Reason)
	}
}
