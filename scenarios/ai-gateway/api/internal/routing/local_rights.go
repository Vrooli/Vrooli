package routing

import (
	"context"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

// RightsContextSource reads trusted, operator-controlled context on demand.
type RightsContextSource func(context.Context) (policy.RightsContext, error)

// EvaluateLocalModelRights is the gateway-side seam for a local candidate once
// its resource owner supplies the concrete artifact identity. The trusted
// context must come from gateway configuration, never the request body.
func EvaluateLocalModelRights(model policy.Model, use policy.RightsUse, context policy.RightsContext, now time.Time) policy.RightsDecision {
	return policy.EvaluateRights(model.Rights, use, context, now)
}

// RuntimeArtifactIdentityMatches requires the provider's observed digest to
// match an explicit owner-supplied mapping to the same legal artifact. The
// source artifact SHA and runtime digest are distinct fields and are never
// compared directly.
func RuntimeArtifactIdentityMatches(binding *policy.RuntimeArtifactIdentity, rights *policy.ArtifactRights, provider, model, observedDigest string) bool {
	return policy.RuntimeArtifactIdentityMatches(binding, rights, provider, model, observedDigest)
}

func sameRuntimeArtifactIdentity(left, right *policy.RuntimeArtifactIdentity) bool {
	return left != nil && right != nil && left.Provider == right.Provider && left.Model == right.Model &&
		left.ArtifactID == right.ArtifactID && left.ArtifactRevision == right.ArtifactRevision &&
		left.DigestKind == right.DigestKind && left.Digest == right.Digest && left.Source == right.Source &&
		left.ObservedAt == right.ObservedAt
}

// RequireLocalRight checks the receipt against current trusted inputs. A valid
// digest only proves that a receipt was not edited; it does not make stale
// artifact, use, or organization facts current.
func RequireLocalRight(decision policy.RightsDecision, currentModel policy.Model, currentUse policy.RightsUse, currentContext policy.RightsContext) error {
	return policy.RequireCurrentRightsDecision(decision, currentModel.Rights, currentUse, currentContext)
}
