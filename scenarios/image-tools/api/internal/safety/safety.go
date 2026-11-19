// Package safety is image-tools' Responsible-Use deployment gate (IMG-P1-015).
//
// The scenario's position (docs/internal/DECISIONS.md, 2026-07-04): this is a
// LOCAL-FIRST tool. No legitimate capability is blocked at the code layer. Editing
// real people — including identity/face/garment/appearance edits — is available on
// every tier. Harm protection is a set of CONFIGURABLE controls (attribution /
// provenance marking, consent affirmation, content scanning, abuse throttling)
// that a deployment can tighten or disable.
//
// The controls therefore gate *how a deployment exposes* the capability, not
// whether the capability exists. A hosted/public deployment is expected to run
// the strict preset because that is where legal exposure lives; a personal
// install may run fully permissive. The operator owns that choice.
//
// One thing is not a configurable control: sexual content involving minors. That
// is illegal content rather than a contested-use question, so the documented
// posture is a hard refusal on any deployment that serves other people. See
// docs/internal/SECURITY.md ("Non-negotiable") for its current build status —
// it is a classifier-policy requirement, not a per-op allowlist.
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
	"strconv"
	"strings"

	"image-tools/internal/capability"
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

// Per-control overrides. Each one wins over the tier preset in BOTH directions,
// so an operator can tighten a local install (e.g. mark provenance) or loosen a
// public one they run themselves. These control harm protection only; they never
// add or remove a capability.
const (
	// ConsentEnv toggles the consent affirmation requirement.
	ConsentEnv = "IMAGE_TOOLS_REQUIRE_CONSENT"
	// NSFWScanEnv toggles forced content scanning of generated output.
	NSFWScanEnv = "IMAGE_TOOLS_NSFW_SCAN"
	// ProvenanceEnv toggles provenance/attribution marking of outputs.
	ProvenanceEnv = "IMAGE_TOOLS_REQUIRE_PROVENANCE"
	// RateLimitEnv sets the AI-submit rate cap (requests/minute, 0 = unlimited).
	RateLimitEnv = "IMAGE_TOOLS_RATE_LIMIT_PER_MIN"
)

// publicRateLimitPerMin is the default AI-submit rate cap on the public tier
// (abuse throttle). 0 would mean unlimited; the local preset is always unlimited.
const publicRateLimitPerMin = 60

// Consent weights are declared in the capability manifest set
// (internal/capability, manifests.seed.json) and read here. That makes the
// manifest the single source: a conformance test requires every AI-catalog
// operation to declare a weight, so a newly added identity-altering op cannot
// silently inherit "none".
//
// Weight NEVER gates availability. It only decides whether an enabled control
// (consent affirmation, elevated provenance, forced scan) applies — and only when
// the deployment has that control on. A weight of High on a permissive local
// install means "nothing"; on a public deployment it means "consent required".
const (
	weightNone = Weight(capability.ConsentWeightNone)
	weightLow  = Weight(capability.ConsentWeightLow)
	weightHigh = Weight(capability.ConsentWeightHigh)
)

// OpWeight returns the consent weight of an operation (WeightNone when unknown).
func OpWeight(op string) Weight {
	switch capability.ConsentWeightFor(op) {
	case capability.ConsentWeightHigh:
		return weightHigh
	case capability.ConsentWeightLow:
		return weightLow
	default:
		return weightNone
	}
}

// OpWeights returns a copy of the full op-weight table (for discovery).
func OpWeights() map[string]Weight {
	declared := capability.Weights()
	out := make(map[string]Weight, len(declared))
	for id, w := range declared {
		out[id] = Weight(w)
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
	// MinorSafetyRequired is the one non-configurable control: a deployment that
	// serves other people must review for sexual content involving minors. It is
	// on for the public tier and is not overridable, unlike every other control.
	MinorSafetyRequired bool
	// MinorSafetyDetector reports whether the deployment actually configured a
	// reviewer. Required-but-absent is a refusal, never a silent pass: the policy
	// cannot claim the boundary while nothing enforces it.
	MinorSafetyDetector bool
}

// PolicyFor derives the DEFAULT control set for a tier. Local is fully
// permissive; public turns on consent, forced content scanning, provenance
// marking, and rate limiting. These are defaults, not mandates — ResolvePolicy
// applies the per-control environment overrides on top.
func PolicyFor(t Tier) Policy {
	if t == TierPublic {
		return Policy{
			Tier:                TierPublic,
			RequireConsent:      true,
			ForceNSFWScan:       true,
			RequireProvenance:   true,
			RateLimitPerMin:     publicRateLimitPerMin,
			MinorSafetyRequired: true,
		}
	}
	return Policy{Tier: TierLocal}
}

// ResolvePolicy returns the effective policy for this process: the tier preset
// with any per-control environment overrides applied. This is what the submit
// edge and the discovery surface should use.
func ResolvePolicy() Policy {
	return ApplyOverrides(PolicyFor(ResolveTier()), lookupEnv)
}

// ApplyOverrides layers environment overrides onto a base policy. Exported with
// an injected lookup so tests can drive it without mutating the process env.
func ApplyOverrides(base Policy, lookup func(string) (string, bool)) Policy {
	if v, ok := parseBoolEnv(lookup, ConsentEnv); ok {
		base.RequireConsent = v
	}
	if v, ok := parseBoolEnv(lookup, NSFWScanEnv); ok {
		base.ForceNSFWScan = v
	}
	if v, ok := parseBoolEnv(lookup, ProvenanceEnv); ok {
		base.RequireProvenance = v
	}
	if v, ok := parseIntEnv(lookup, RateLimitEnv); ok {
		if v < 0 {
			v = 0
		}
		base.RateLimitPerMin = v
	}
	return base
}

func lookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

func parseBoolEnv(lookup func(string) (string, bool), key string) (bool, bool) {
	raw, ok := lookup(key)
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func parseIntEnv(lookup func(string) (string, bool), key string) (int, bool) {
	raw, ok := lookup(key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return n, true
}

// Summary is a one-line human description of the active policy. It is DERIVED
// from the resolved controls rather than asserting a fixed posture, so it cannot
// claim a protection the deployment has turned off (or deny a capability that
// exists).
func (p Policy) Summary() string {
	if p.Tier == TierPublic {
		return "Public deployment policy: " + p.controlsSummary() + ". All image capabilities remain available."
	}
	return "Local/personal deployment policy: " + p.controlsSummary() + ". All image capabilities remain available."
}

func (p Policy) controlsSummary() string {
	on := func(b bool, name string) string {
		if b {
			return name + " on"
		}
		return name + " off"
	}
	rate := "rate limit off"
	if p.RateLimitPerMin > 0 {
		rate = "rate limit " + strconv.Itoa(p.RateLimitPerMin) + "/min"
	}
	minor := "minor-safety review off"
	if p.MinorSafetyRequired {
		minor = "minor-safety review required"
		if p.MinorSafetyDetector {
			minor += " (detector configured)"
		} else {
			minor += " (NO DETECTOR — image-output submissions are refused)"
		}
	}
	return on(p.RequireConsent, "consent affirmation") + ", " +
		on(p.ForceNSFWScan, "content scan") + ", " +
		on(p.RequireProvenance, "provenance marking") + ", " + rate + ", " + minor +
		". Override with " + ConsentEnv + " / " + NSFWScanEnv + " / " + ProvenanceEnv + " / " + RateLimitEnv
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
	// RequireMinorReview tells the submit edge that this operation must be
	// reviewed for sexual content involving minors before it runs.
	RequireMinorReview bool
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
	// The minor-safety boundary is checked before any capability runs, and it is
	// not a consent question: no affirmation makes it acceptable. A public
	// deployment with no detector cannot produce images at all, because the
	// boundary it promises would otherwise be unenforced.
	if p.RequiresMinorReview(op) {
		d.RequireMinorReview = true
		if !p.MinorSafetyDetector {
			d.Allowed = false
			d.Reason = "this deployment serves other people and requires a minor-safety reviewer for image output, but none is configured"
			d.RecoveryHint = "set " + MinorSafetyEnv + " to a program implementing the documented JSON review contract; the local/personal deployment tier (the default) does not require one"
			return d
		}
	}
	if w == WeightHigh && p.RequireConsent && !consentAffirmed {
		d.Allowed = false
		d.Reason = "this deployment requires a consent affirmation for identity-altering edits (" + op + ")"
		d.RecoveryHint = "resubmit with consent_affirmed=true to record that you have the right to edit the people in this image; this deployment's Acceptable-Use Policy requires the affirmation. Set " + ConsentEnv + "=0 to disable this control if you own the deployment"
		return d
	}
	if w == WeightHigh && consentAffirmed {
		d.RecordConsent = true
	}
	return d
}

// RequiresMinorReview reports whether producing this operation's output must be
// reviewed for sexual content involving minors. It is scoped to image output:
// an operation that returns data or a mesh cannot depict a person.
func (p Policy) RequiresMinorReview(op string) bool {
	if p.Tier != TierPublic {
		return false
	}
	m, ok := capability.Get(op)
	return ok && m.IO.Output.Kind == "image"
}
