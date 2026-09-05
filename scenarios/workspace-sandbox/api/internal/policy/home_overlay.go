// Package policy. home_overlay.go: the single source of truth for
// "is the per-sandbox $HOME overlay currently usable?"
//
// Owns the single comparison
// (`profile.HomeOverlayRequirement` × `sb.HomeOverlayState` × driver
// capability) so the workspace-sandbox handler exec gate, the
// agent-manager preflight, and the driver capability check all consult
// one decision rather than re-deriving the rule.
//
// DOC: home-overlay seam — unified policy decision.
package policy

import (
	"fmt"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// HomeOverlayDecision is the structured outcome of a home-overlay
// availability check.
type HomeOverlayDecision struct {
	// Allowed is true when the requested operation may proceed without
	// a home-overlay-related refusal.
	Allowed bool

	// Reason is the human-readable explanation when Allowed is false.
	// Empty when Allowed is true. Suitable for logs and audit trails.
	Reason string

	// Code is the stable machine-readable error code paired with the
	// refusal. Matches the existing types.HomeOverlayRequiredError code
	// surface so downstream consumers (UI, agent-manager run timeline)
	// can key off it.
	Code string
}

// HomeOverlay decision codes. Kept as constants so the contract test
// and the caller agree on the strings. The first two are refusal
// codes; HOME_OVERLAY_FALLBACK is a success-with-warning surfaced by
// callers as a structured audit event so soft fallbacks remain
// observable.
const (
	CodeHomeOverlayRequired          = "HOME_OVERLAY_REQUIRED"
	CodeHomeOverlayUnsupportedDriver = "HOME_OVERLAY_UNSUPPORTED_DRIVER"
	CodeHomeOverlayFallback          = "HOME_OVERLAY_FALLBACK"
)

// IsHomeOverlayPresent reports whether the sandbox's home overlay is
// in the satisfied state. Sole authority on the meaning of
// HomeOverlayState. Both call sites (workspace-sandbox handler exec
// gating and agent-manager command translation) consult this rather
// than comparing against the constant inline.
func IsHomeOverlayPresent(state types.HomeOverlayState) bool {
	return state == types.HomeOverlayPresent
}

// DecideHomeOverlay applies the workspace-sandbox profile contract:
//
//  1. HomeOverlayNotNeeded: unconditionally allowed, no code.
//  2. HomeOverlayOptional: allowed in every state. When the overlay
//     is anything other than Present the decision carries
//     HOME_OVERLAY_FALLBACK so callers can record the soft
//     degradation. Driver capability is irrelevant — optional means
//     the profile copes either way.
//  3. HomeOverlayRequired + driver does NOT support the overlay:
//     refused with HOME_OVERLAY_UNSUPPORTED_DRIVER. Switching drivers
//     or picking a different profile is the operator fix.
//  4. HomeOverlayRequired + state != Present: refused with
//     HOME_OVERLAY_REQUIRED.
//  5. HomeOverlayRequired + state == Present: allowed.
//
// An empty/unknown HomeOverlayRequirement is treated as
// HomeOverlayNotNeeded so partial-decode JSON callers don't trip
// refusals; the validating profile loader rejects unknown values
// at registry-load time.
//
// The function is pure — no I/O, no logging — so it can be exercised
// from a contract test matrix without scaffolding.
func DecideHomeOverlay(caps driver.DriverCapabilities, profile config.IsolationProfile, sb types.Sandbox) HomeOverlayDecision {
	switch profile.HomeOverlayRequirement {
	case types.HomeOverlayRequired:
		if !caps.HomeOverlay {
			return HomeOverlayDecision{
				Allowed: false,
				Reason: fmt.Sprintf(
					"isolation profile %q requires a home overlay but the active driver does not support it",
					profile.ID,
				),
				Code: CodeHomeOverlayUnsupportedDriver,
			}
		}
		if !IsHomeOverlayPresent(sb.HomeOverlayState) {
			return HomeOverlayDecision{
				Allowed: false,
				Reason: fmt.Sprintf(
					"isolation profile %q requires a home overlay; sandbox home_overlay_state=%q",
					profile.ID, sb.HomeOverlayState,
				),
				Code: CodeHomeOverlayRequired,
			}
		}
		return HomeOverlayDecision{Allowed: true}
	case types.HomeOverlayOptional:
		if IsHomeOverlayPresent(sb.HomeOverlayState) {
			return HomeOverlayDecision{Allowed: true}
		}
		return HomeOverlayDecision{
			Allowed: true,
			Reason: fmt.Sprintf(
				"isolation profile %q allows fallback; sandbox home_overlay_state=%q",
				profile.ID, sb.HomeOverlayState,
			),
			Code: CodeHomeOverlayFallback,
		}
	default:
		// HomeOverlayNotNeeded (or empty/unknown — treated as not_needed).
		return HomeOverlayDecision{Allowed: true}
	}
}
