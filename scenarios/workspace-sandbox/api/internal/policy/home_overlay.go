// Package policy. home_overlay.go: the single source of truth for
// "is the per-sandbox $HOME overlay currently usable?"
//
// Before this file existed, the same comparison
// (`profile.RequiresHomeOverlay && sb.HomeOverlayState != Present`)
// lived inline in the workspace-sandbox handler exec gate, the
// agent-manager preflight, and (less directly) the driver capability
// check. Three places to drift, three places to forget.
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

// HomeOverlay error codes. Kept as constants so the contract test and
// the caller agree on the strings.
const (
	CodeHomeOverlayRequired          = "HOME_OVERLAY_REQUIRED"
	CodeHomeOverlayUnsupportedDriver = "HOME_OVERLAY_UNSUPPORTED_DRIVER"
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
//  1. If the requested isolation profile does NOT require a home
//     overlay, the operation is unconditionally allowed.
//  2. If the profile requires a home overlay and the active driver
//     does not advertise that capability, the operation is refused
//     with HOME_OVERLAY_UNSUPPORTED_DRIVER. (Switching drivers or
//     picking a different profile is the operator fix.)
//  3. If the profile requires a home overlay and the sandbox's
//     HomeOverlayState is anything other than Present, the operation
//     is refused with HOME_OVERLAY_REQUIRED.
//
// The function is pure — no I/O, no logging — so it can be exercised
// from a contract test matrix without scaffolding.
func DecideHomeOverlay(caps driver.DriverCapabilities, profile config.IsolationProfile, sb types.Sandbox) HomeOverlayDecision {
	if !profile.RequiresHomeOverlay {
		return HomeOverlayDecision{Allowed: true}
	}
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
}
