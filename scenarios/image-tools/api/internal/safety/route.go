package safety

import "strings"

// Locality says where a resolved route executes.
type Locality string

const (
	// LocalityLocal runs on this machine (the user's own CPU or GPU).
	LocalityLocal Locality = "local"
	// LocalityRemote runs somewhere else: a cloud provider reached with a key
	// (BYOK) or any route this process cannot show stays on this machine.
	LocalityRemote Locality = "remote"
)

// tierLocality is the route-locality rule as data: which route tiers keep the
// work on this machine. A tier missing from the table leaves the machine, so a
// new route kind must declare itself local to skip the affirmation instead of
// having to remember to opt in to it.
var tierLocality = map[string]Locality{
	"local-gpu": LocalityLocal,
	"local-cpu": LocalityLocal,
	// A model the AI Gateway serves on this machine: the image stays here.
	"gateway-local": LocalityLocal,
	"byok-cloud":    LocalityRemote,
}

// LocalityOfTier classifies a route tier (backends.Tier.String()).
func LocalityOfTier(tier string) Locality {
	if l, ok := tierLocality[strings.TrimSpace(tier)]; ok {
		return l
	}
	return LocalityRemote
}

// Consent bases: which rule asks for the affirmation.
const (
	// ConsentBasisTier is the deployment-tier rule (Policy.RequireConsent).
	ConsentBasisTier = "tier"
	// ConsentBasisRoute is the route-locality rule (Policy.ConsentNonLocal).
	ConsentBasisRoute = "route"
)

// ConsentRequirement says whether a request must carry consent_affirmed and why.
// It is the one answer shared by the submit edge (which enforces it), the
// engine's automatic routing (which must not move an un-affirmed request off
// this machine) and the plan/explain surface (which tells the UI when to show
// the checkbox), so the three cannot disagree.
type ConsentRequirement struct {
	Required bool
	// Basis is ConsentBasisTier or ConsentBasisRoute when Required.
	Basis string
	// Reason is a plain-language sentence naming the rule that applies.
	Reason string
	// Weight is the consent weight the decision was made on.
	Weight Weight
	// Locality is where the route runs.
	Locality Locality
	// FaceGated is true while the requirement depends on whether the input shows
	// people (a low-weight op on a route that leaves this machine). WithFaces
	// settles it; until then Required is the answer when nobody has looked.
	FaceGated bool
	// Detail says why the request did or did not need an affirmation, in the form
	// the job's meta records as consent_basis: faces_detected=N, detector_unavailable,
	// not_required_local or not_required_control_off. It is empty for a weight that
	// never asks, and for a requirement still waiting on its people check.
	Detail string
}

// ConsentFor reports whether a request of op at consent weight w, resolved to
// route tier tier, needs a consent affirmation under this policy.
//
//   - Deployment-tier rule: on a deployment with the consent control on, every
//     high-weight op needs it.
//   - Route-locality rule: on any tier, a high-weight op whose route leaves this
//     machine needs it, unless the deployment turned that control off. Local runs
//     stay unrestricted: a person editing their own images with local open
//     weights is not asked anything.
func (p Policy) ConsentFor(w Weight, op, tier string) ConsentRequirement {
	out := p.consentRule(w, op, tier)
	if !out.Required && !out.FaceGated && out.Detail == "" && w != WeightNone {
		out.Detail = ConsentDetailControlOff
		if out.Locality == LocalityLocal {
			out.Detail = ConsentDetailLocal
		}
	}
	return out
}

func (p Policy) consentRule(w Weight, op, tier string) ConsentRequirement {
	loc := LocalityOfTier(tier)
	out := ConsentRequirement{Weight: w, Locality: loc}
	if w == WeightNone {
		return out
	}
	if w == WeightLow {
		// A low-weight op changes nothing about who is shown (a caption, boxes,
		// tags, a colourisation), so the deployment-tier rule never asks. The image
		// still leaves this machine on a remote route, which is a different act
		// from running a local model, so the route-locality rule does, but only for
		// an image that shows people: the rule protects the people in it, so the
		// decision waits for the people check (FaceGated; see WithFaces).
		if loc == LocalityRemote && p.ConsentNonLocal {
			out.Required, out.Basis, out.FaceGated = true, ConsentBasisRoute, true
			out.Reason = "this request would send the image to a provider outside this machine (" + routeLabel(tier) + "), so a consent affirmation is needed for " + op + " when the image shows people: confirm you have the right to send the people in this image"
		}
		return out
	}
	if p.Tier == TierPublic && p.RequireConsent {
		out.Required, out.Basis = true, ConsentBasisTier
		out.Reason = "this deployment requires a consent affirmation for identity-altering edits (" + op + ")"
		return out
	}
	if loc == LocalityRemote && p.ConsentNonLocal {
		out.Required, out.Basis = true, ConsentBasisRoute
		out.Reason = "this request would run on a provider outside this machine (" + routeLabel(tier) + "), and identity-altering edits (" + op + ") need a consent affirmation there: confirm you have the right to edit the people in this image"
	}
	return out
}

func routeLabel(tier string) string {
	if tier = strings.TrimSpace(tier); tier != "" {
		return tier
	}
	return "unresolved route"
}

// EvaluateRoute decides a submit once its route is known. It applies the
// consent requirement for the resolved tier, and reports RecordConsent when an
// affirmation is both required and given, so the audit log holds exactly the
// affirmations that unlocked something.
func (p Policy) EvaluateRoute(w Weight, op, tier string, consentAffirmed bool) Decision {
	// Nobody has looked at the image here, so a face-gated requirement stands.
	return p.decideRoute(p.ConsentFor(w, op, tier).WithFaces(0, ErrNoFaceCounter), w, op, consentAffirmed)
}

// decideRoute turns a settled consent requirement into the submit decision.
func (p Policy) decideRoute(need ConsentRequirement, w Weight, op string, consentAffirmed bool) Decision {
	d := Decision{Allowed: true, Weight: w, ForceNSFWScan: p.ForceNSFWScan, ConsentBasis: need.Detail}
	if d.ConsentBasis == "" {
		d.ConsentBasis = need.Basis
	}
	if !need.Required {
		return d
	}
	if !consentAffirmed {
		d.Allowed = false
		d.Reason = need.Reason
		right := "edit"
		if w == WeightLow {
			right = "send"
		}
		d.RecoveryHint = "resubmit with consent_affirmed=true (CLI: --consent; UI: tick the consent box) to record that you have the right to " + right + " the people in this image"
		if need.Basis == ConsentBasisRoute {
			d.RecoveryHint += ", or keep the work on this machine (a local model, fallback_policy=local_only) where no affirmation is asked. Set " + ConsentNonLocalEnv + "=0 to turn this control off if you own the deployment"
		} else {
			d.RecoveryHint += ". Set " + ConsentEnv + "=0 to disable this control if you own the deployment"
		}
		return d
	}
	d.RecordConsent = true
	return d
}
