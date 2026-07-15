package opsrunner

import (
	"context"
	"fmt"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
)

// OverrideStore reads the layered binding overrides that live in domain storage
// (never in the shipped catalog): a backlog item pins its own mode, an
// initiative pins the mode for its members. Implementations must return only
// documents that pass agentops.ValidateBinding; a malformed override is a
// fail-closed error, never silently skipped, because dropping a higher-precedence
// override would let a lower layer win against the operator's intent.
type OverrideStore interface {
	// OverridesFor returns the backlog-item and initiative override bindings
	// in scope for a target. A backlog-item target contributes its own item
	// override and (when it belongs to an initiative) that initiative's override;
	// an initiative target contributes only its initiative override.
	OverridesFor(ctx context.Context, target TargetRef, operation agentops.OperationID) ([]agentops.OperationBinding, error)
}

// BindingResolver wires agentops.ResolveBinding to real sources: catalog system
// defaults + domain overrides + an optional authorized-invocation binding. It
// snapshots the winning layer and pins the exact mode revision, failing closed
// on every ambiguous or invalid state (no silent fallback from a malformed
// higher-precedence override).
type BindingResolver struct {
	catalog   *opscatalog.Catalog
	overrides OverrideStore
	checker   agentops.ModeChecker
	// InitiativeOfItem, when set, names the initiative a backlog-item target
	// belongs to (empty when none) so the initiative-override layer is in scope
	// for member items. It must be the SAME mapping the override store uses
	// (FSOverrideStore.InitiativeOfItem), or a fetched inherited override would
	// be rejected as out of scope. Nil means item targets resolve without
	// initiative inheritance.
	InitiativeOfItem func(itemRef string) (string, error)
}

// NewBindingResolver constructs a resolver. checker answers revision-existence
// and mode-compatibility; pass the ModePreparer-backed checker so a binding to a
// deleted revision or an incompatible mode is a typed error.
func NewBindingResolver(catalog *opscatalog.Catalog, overrides OverrideStore, checker agentops.ModeChecker) *BindingResolver {
	return &BindingResolver{catalog: catalog, overrides: overrides, checker: checker}
}

// Resolution is a resolved binding plus the catalog provenance that decided it.
type Resolution struct {
	Binding        agentops.ResolvedBinding
	Scope          agentops.ResolutionScope
	PolicyID       string
	PolicyRevision string
}

// Resolve gathers all in-scope candidate bindings and returns the winning
// binding with deterministic precedence. It fails closed via the agentops typed
// errors (ErrNoBinding/ErrInvalidOverride/ErrDeletedRevision/ErrIncompatibleMode).
func (r *BindingResolver) Resolve(ctx context.Context, req InvokeRequest) (Resolution, error) {
	scope := agentops.ResolutionScope{Target: req.Target.Kind}
	switch req.Target.Kind {
	case agentops.TargetBacklogItem:
		scope.ItemRef = req.Target.ID
		// A member item also resolves inside its initiative's scope, so the
		// initiative-override layer applies to it (inheritance).
		if r.InitiativeOfItem != nil {
			initName, err := r.InitiativeOfItem(req.Target.ID)
			if err != nil {
				return Resolution{}, fmt.Errorf("resolve item initiative: %w", err)
			}
			scope.InitiativeName = initName
		}
	case agentops.TargetInitiative:
		scope.InitiativeName = req.Target.ID
	}

	var candidates []agentops.OperationBinding
	// system-default layer, from the shipped catalog.
	if sys, ok := r.catalog.SystemBindingFor(req.Operation, req.OperationVersion); ok {
		candidates = append(candidates, sys.Binding)
	}
	// override layers, from domain storage.
	if r.overrides != nil {
		ovs, err := r.overrides.OverridesFor(ctx, req.Target, req.Operation)
		if err != nil {
			return Resolution{}, fmt.Errorf("load binding overrides: %w", err)
		}
		candidates = append(candidates, ovs...)
	}
	// authorized-invocation layer, pinned by the caller for this one run.
	if req.AuthorizedInvocationBinding != nil {
		b := *req.AuthorizedInvocationBinding
		scope.InvocationID = ""
		if b.Owner != nil {
			scope.InvocationID = b.Owner.ID
		}
		candidates = append(candidates, b)
	}

	resolved, err := agentops.ResolveBinding(req.Operation, candidates, scope, r.checker)
	if err != nil {
		return Resolution{}, err
	}

	res := Resolution{Binding: resolved, Scope: scope}
	// Pin the transition policy governing this target's domain, when authored.
	domainKind := domainKindFor(req.Target.Kind)
	if lp, ok := r.catalog.PolicyForDomain(domainKind); ok {
		res.PolicyID = lp.Policy.ID
		res.PolicyRevision = lp.Revision
	}
	return res, nil
}

// domainKindFor maps a target kind to its workflow domain kind. plan-execution
// has no owning domain workflow (it is a transient unit of work), so it maps to
// itself and simply has no policy.
func domainKindFor(kind agentops.TargetKind) string {
	switch kind {
	case agentops.TargetBacklogItem:
		return "backlog-item"
	case agentops.TargetInitiative:
		return "initiative"
	default:
		return string(kind)
	}
}

// provenanceBinding projects a resolved binding onto its immutable provenance
// form, attributing a system-default binding to the "system" owner.
func provenanceBinding(b agentops.ResolvedBinding) agentops.ProvenanceBinding {
	pb := agentops.ProvenanceBinding{Layer: b.Layer}
	if b.Layer == agentops.LayerSystemDefault || b.Owner == nil {
		pb.OwnerKind = "system"
		pb.OwnerID = "system"
		return pb
	}
	pb.OwnerKind = b.Owner.Kind
	pb.OwnerID = b.Owner.ID
	return pb
}
