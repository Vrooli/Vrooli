// Package safety is image-tools' Responsible-Use deployment gate (IMG-P1-015).
//
// The scenario's position (docs/internal/DECISIONS.md, 2026-06-18): editing real
// people is gated at DEPLOYMENT, not at the capability. Local/personal use is
// unrestricted; a public/monetized deployment opts into a stricter policy. The
// hard non-goals (no recognition / face-swap / deepfake / try-on) hold on EVERY
// tier regardless of policy — they live in the op catalog (those ops simply do
// not exist), not here.
//
// This package owns:
//   - tier resolution (the deployment chooses local vs public at deploy time, via
//     IMAGE_TOOLS_DEPLOYMENT_TIER — NOT a runtime end-user toggle, which would
//     defeat the gate);
//   - the per-operation consent-weight table (which ops can alter a real person);
//   - the policy derived from the tier;
//   - the Evaluate decision the AI submit edge enforces (consent required? force
//     the NSFW scan? rate-limited?);
//   - the consent log (an audit row per affirmed high-weight op).
//
// It is pure policy + a thin store; it touches no pixels and no models.
package safety

import (
	"os"
	"strings"
)

// Tier is the resolved deployment tier.
type Tier string

const (
	// TierLocal is personal/local use — unrestricted (the default).
	TierLocal Tier = "local"
	// TierPublic is a public/monetized deployment — the gate is active.
	TierPublic Tier = "public"
)

// Weight is how identity-sensitive an operation is.
type Weight string

const (
	// WeightNone — no identity concern.
	WeightNone Weight = "none"
	// WeightLow — texture/realism only, no identity alteration (naturalize).
	WeightLow Weight = "low"
	// WeightHigh — can alter a real person's identity/body/clothing/pose.
	WeightHigh Weight = "high"
)

// Valid reports whether w is a known consent-weight enum member. Exported so the
// adapter catalog (internal/adapters), which carries a per-adapter consent weight
// that elevates the effective op weight, can validate its seed against this SSOT.
func (w Weight) Valid() bool {
	switch w {
	case WeightNone, WeightLow, WeightHigh:
		return true
	default:
		return false
	}
}

// MaxWeight returns the most identity-sensitive weight among ws (WeightNone for
// an empty set). This is the effective-weight elevation rule (decision C4):
// the consent weight of a conditioned op is max(op weight, adapter weights...).
func MaxWeight(ws ...Weight) Weight {
	rank := func(w Weight) int {
		switch w {
		case WeightHigh:
			return 2
		case WeightLow:
			return 1
		default:
			return 0
		}
	}
	out := WeightNone
	for _, w := range ws {
		if rank(w) > rank(out) {
			out = w
		}
	}
	return out
}

// TierEnv is the deployment-time environment variable that selects the tier.
const TierEnv = "IMAGE_TOOLS_DEPLOYMENT_TIER"

// publicRateLimitPerMin is the AI-submit rate cap on the public tier (abuse
// throttle). 0 would mean unlimited; the local tier is always unlimited.
const publicRateLimitPerMin = 60

// opWeights is the canonical per-operation consent-weight table. Ops absent from
// the map default to WeightNone (analysis, diff, text_to_image, upscale,
// background_removal, denoise — none can alter a real person). High-weight ops
// are the identity/body/clothing/pose-altering edits.
var opWeights = map[string]Weight{
	"naturalize":         WeightLow,
	"edit_instruct":      WeightHigh,
	"inpaint":            WeightHigh,
	"outpaint":           WeightHigh,
	"object_removal":     WeightHigh,
	"background_replace": WeightHigh,
	"image_to_image":     WeightHigh,
}

// OpWeight returns the consent weight of an operation (WeightNone when unknown).
func OpWeight(op string) Weight {
	if w, ok := opWeights[op]; ok {
		return w
	}
	return WeightNone
}

// OpWeights returns a copy of the full op-weight table (for discovery).
func OpWeights() map[string]Weight {
	out := make(map[string]Weight, len(opWeights))
	for k, v := range opWeights {
		out[k] = v
	}
	return out
}

// ResolveTier reads the deployment tier from the environment, defaulting to
// local (unrestricted). Anything other than an explicit "public"/"prod"/
// "monetized" is treated as local — fail OPEN for local convenience but the
// public gate is only ever active when explicitly requested.
func ResolveTier() Tier {
	return ParseTier(os.Getenv(TierEnv))
}

// ParseTier maps a raw tier string to a Tier (exported for tests + the handler).
func ParseTier(raw string) Tier {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "public", "prod", "production", "monetized", "saas":
		return TierPublic
	default:
		return TierLocal
	}
}

// Policy is the resolved Responsible-Use policy for a tier.
type Policy struct {
	Tier              Tier
	RequireConsent    bool
	ForceNSFWScan     bool
	RequireProvenance bool
	RateLimitPerMin   int
}

// PolicyFor derives the policy for a tier. Local is fully permissive; public
// turns on consent, forced NSFW scanning, provenance marking, and rate limiting.
func PolicyFor(t Tier) Policy {
	if t == TierPublic {
		return Policy{
			Tier:              TierPublic,
			RequireConsent:    true,
			ForceNSFWScan:     true,
			RequireProvenance: true,
			RateLimitPerMin:   publicRateLimitPerMin,
		}
	}
	return Policy{Tier: TierLocal}
}

// Summary is a one-line human description of the active policy.
func (p Policy) Summary() string {
	if p.Tier == TierPublic {
		return "Public deployment: consent required for identity-altering edits, NSFW auto-scan on, provenance marked, abuse rate-limited. Hard non-goals (no recognition/face-swap/deepfake) always enforced."
	}
	return "Local/personal deployment: unrestricted. Hard non-goals (no recognition/face-swap/deepfake) always enforced."
}

// ImportServeFacts are the catalog facts the import serve gate keys on (a
// model's CapabilityLabels). Passed as primitives so safety stays model-free.
type ImportServeFacts struct {
	// Provenance is the entry's provenance label ("" for seed entries,
	// "user-imported" for guided-import entries — mirrors
	// models.ProvenanceUserImported).
	Provenance string
	// CommercialUse is the entry's commercial-use posture ("yes" once the operator
	// attested commercial rights at import time — mirrors models.CommercialUseYes).
	CommercialUse string
}

// AllowImportServe reports whether a model may be SERVED on the given tier given
// its provenance + commercial posture (decision D4). A user-imported entry whose
// commercial rights are not attested is refused on the public tier with an
// actionable reason; local use is always allowed, and seed entries (empty
// provenance) are unaffected. This is the deployment-tier half of the import gate
// — the import flow already defaults entries to local-only until attested.
func AllowImportServe(t Tier, f ImportServeFacts) (bool, string) {
	if t != TierPublic {
		return true, ""
	}
	if f.Provenance != "user-imported" {
		return true, ""
	}
	if strings.EqualFold(strings.TrimSpace(f.CommercialUse), "yes") {
		return true, ""
	}
	return false, "user-imported model with unverified commercial rights cannot be served on a public/BYOK deployment; re-import with --attest-commercial-rights to confirm you hold the rights"
}

// Decision is the outcome of evaluating a submit against the policy.
type Decision struct {
	// Allowed is false when the gate blocks the submit.
	Allowed bool
	// Reason explains a block (actionable; empty when Allowed).
	Reason string
	// RecoveryHint is a short next-step the caller can act on.
	RecoveryHint string
	// ForceNSFWScan tells the edge to enable the output auto-scan regardless of
	// what the caller requested.
	ForceNSFWScan bool
	// RecordConsent is true when an affirmed high-weight op should be logged.
	RecordConsent bool
	// Weight is the resolved consent weight of the op (for logging/telemetry).
	Weight Weight
}

// Evaluate decides whether an AI submit for op may proceed under the policy,
// given whether the caller affirmed consent. On the local tier everything is
// allowed (no consent, no forced scan). On the public tier a high-weight op
// requires consentAffirmed; the NSFW scan is always forced on.
func (p Policy) Evaluate(op string, consentAffirmed bool) Decision {
	return p.EvaluateWeight(OpWeight(op), op, consentAffirmed)
}

// EvaluateWeight is Evaluate with an explicitly supplied consent weight instead
// of the op's own — the seam the submit edge uses to enforce the conditioning
// weight ELEVATION (decision C4): a request whose adapter stack raises the
// effective weight to high (an IP-Adapter / identity LoRA on a none-weight op)
// requires consent on the public tier even though the bare op would not.
func (p Policy) EvaluateWeight(w Weight, op string, consentAffirmed bool) Decision {
	d := Decision{Allowed: true, Weight: w, ForceNSFWScan: p.ForceNSFWScan}
	if p.Tier != TierPublic {
		return d
	}
	if w == WeightHigh && p.RequireConsent && !consentAffirmed {
		d.Allowed = false
		d.Reason = "this deployment requires a consent affirmation for identity-altering edits (" + op + ")"
		d.RecoveryHint = "resubmit with consent_affirmed=true after confirming you have the right to edit the people in this image; editing real people without consent is prohibited by the Acceptable-Use Policy"
		return d
	}
	if w == WeightHigh && consentAffirmed {
		d.RecordConsent = true
	}
	return d
}
