// Package modelpromotion joins model-catalog declarations to owner-produced
// qualification receipts. A seed reference or status string is never proof.
// Catalog selection consumes results through DefaultForVerified and
// Result.VerifiedFor. Corpus quality, interactive latency, and model-memory
// receipt resolution are integrated; defaults remain unchanged until all five
// facets have passing owner receipts.
package modelpromotion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"audio-tools/internal/modelcatalog"
)

const (
	FacetCorpusQuality = "corpus_quality"
	FacetLatency       = "latency"
	FacetMemory        = "memory"
	FacetPlatform      = "platform"
	FacetLicense       = "license"
)

var requiredFacets = []string{FacetCorpusQuality, FacetLatency, FacetMemory, FacetPlatform, FacetLicense}

// Receipt is the identity-bearing, owner-verified result returned by a facet
// owner. Capability is part of the join even when the owner's underlying
// persistence schema stores only a unique model ID.
type Receipt struct {
	Reference  string
	Owner      string
	Facet      string
	ModelID    string
	Capability modelcatalog.Capability
	Passed     bool
}

type Resolver interface {
	ResolvePromotionReceipt(context.Context, string, string, modelcatalog.Model) (Receipt, error)
}

type Result struct {
	ModelID        string
	Capability     modelcatalog.Capability
	Qualified      bool
	VerifiedFacets []string
	Blockers       []string
	verified       bool // Set only by Verify; prevents caller-assembled results becoming proof.
	verifiedModel  string
	verifiedCap    modelcatalog.Capability
	verifiedFacets []string
	verifiedDigest [sha256.Size]byte
}

// VerifiedFor reports whether this Verify-produced result qualifies exactly
// the supplied catalog record. Result fields alone are intentionally not proof.
func (r Result) VerifiedFor(model modelcatalog.Model) bool {
	if !r.verified || !r.Qualified || r.verifiedModel != model.ID || r.verifiedCap != model.Capability || r.ModelID != model.ID || r.Capability != model.Capability || len(r.Blockers) != 0 || len(r.verifiedFacets) != len(requiredFacets) {
		return false
	}
	digest, err := declarationDigest(model)
	if err != nil || digest != r.verifiedDigest {
		return false
	}
	for i, facet := range requiredFacets {
		if r.verifiedFacets[i] != facet {
			return false
		}
	}
	return true
}

// declarationDigest binds receipts to every declared selector and adapter
// fact, including evidence refs, providers, engines, roles, and readiness.
func declarationDigest(model modelcatalog.Model) ([sha256.Size]byte, error) {
	data, err := json.Marshal(model)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(data), nil
}

// OwnerByFacet lists only currently integrated receipt owners. An absent
// entry is a hard blocker; callers cannot satisfy a missing facet with a
// fabricated reference or an unregistered resolver.
func ownerForFacet(facet string) (string, bool) {
	if facet == FacetCorpusQuality {
		return "audio-tools.experiment.qualification_evidence", true
	}
	if facet == FacetLatency {
		return "audio-tools.experiment.qualification_evidence", true
	}
	if facet == FacetMemory {
		return "audio-tools.experiment.qualification_evidence", true
	}
	return "", false
}

func Verify(ctx context.Context, model modelcatalog.Model, resolvers map[string]Resolver) Result {
	return verify(ctx, model, resolvers, nil)
}

// verifyWithOwners lets package tests exercise the generic join. Production
// owners are code-wired below; callers cannot declare an owner in seed data.
func verifyWithOwners(ctx context.Context, model modelcatalog.Model, resolvers map[string]Resolver, owners map[string]string) Result {
	return verify(ctx, model, resolvers, owners)
}

func verify(ctx context.Context, model modelcatalog.Model, resolvers map[string]Resolver, owners map[string]string) Result {
	result := Result{ModelID: model.ID, Capability: model.Capability}
	for _, facet := range requiredFacets {
		declaration := evidenceFor(model.Promotion, facet)
		if declaration.Status != "qualified" {
			state := declaration.Status
			if state == "" {
				state = "not declared"
			}
			result.Blockers = append(result.Blockers, fmt.Sprintf("%s evidence is %s", facet, state))
			continue
		}
		if strings.TrimSpace(declaration.Ref) == "" {
			result.Blockers = append(result.Blockers, fmt.Sprintf("%s evidence has no owner receipt reference", facet))
			continue
		}
		owner, supported := ownerForFacet(facet)
		if owners != nil {
			owner, supported = owners[facet]
			supported = supported && strings.TrimSpace(owner) != ""
		}
		resolver := resolvers[facet]
		if !supported || resolver == nil {
			result.Blockers = append(result.Blockers, fmt.Sprintf("%s has no integrated receipt owner; declared reference %q is unresolved", facet, declaration.Ref))
			continue
		}
		receipt, err := resolver.ResolvePromotionReceipt(ctx, facet, declaration.Ref, model)
		if err != nil {
			result.Blockers = append(result.Blockers, fmt.Sprintf("%s receipt %q could not be resolved: %v", facet, declaration.Ref, err))
			continue
		}
		if receipt.Reference != declaration.Ref || receipt.Owner != owner || receipt.Facet != facet || receipt.ModelID != model.ID || receipt.Capability != model.Capability || !receipt.Passed {
			result.Blockers = append(result.Blockers, fmt.Sprintf("%s receipt %q failed owner, facet, model, capability, or outcome verification", facet, declaration.Ref))
			continue
		}
		result.VerifiedFacets = append(result.VerifiedFacets, facet)
	}
	result.Qualified = len(result.Blockers) == 0 && len(result.VerifiedFacets) == len(requiredFacets)
	result.verified = result.Qualified
	if result.verified {
		digest, err := declarationDigest(model)
		if err != nil {
			result.verified = false
			result.Qualified = false
			result.Blockers = append(result.Blockers, fmt.Sprintf("model declaration could not be bound: %v", err))
			return result
		}
		result.verifiedModel = model.ID
		result.verifiedCap = model.Capability
		result.verifiedFacets = append([]string(nil), result.VerifiedFacets...)
		result.verifiedDigest = digest
	}
	return result
}

func evidenceFor(e modelcatalog.PromotionEvidence, facet string) modelcatalog.EvidenceFacet {
	switch facet {
	case FacetCorpusQuality:
		return e.CorpusQuality
	case FacetLatency:
		return e.Latency
	case FacetMemory:
		return e.Memory
	case FacetPlatform:
		return e.Platform
	case FacetLicense:
		return e.License
	default:
		return modelcatalog.EvidenceFacet{}
	}
}
