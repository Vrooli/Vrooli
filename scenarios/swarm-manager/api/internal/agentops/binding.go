package agentops

import (
	"encoding/json"
	"errors"
	"fmt"
)

// BindingLayer is a precedence layer for operation bindings (EXECUTION-MODES.md
// D2). Higher rank wins.
type BindingLayer string

const (
	LayerAuthorizedInvocation BindingLayer = "authorized-invocation"
	LayerBacklogItemOverride  BindingLayer = "backlog-item-override"
	LayerInitiativeOverride   BindingLayer = "initiative-override"
	LayerSystemDefault        BindingLayer = "system-default"
)

// layerRank orders layers highest-precedence first.
var layerRank = map[BindingLayer]int{
	LayerAuthorizedInvocation: 4,
	LayerBacklogItemOverride:  3,
	LayerInitiativeOverride:   2,
	LayerSystemDefault:        1,
}

// Typed resolution failures. The resolver fails CLOSED: every one of these is a
// definitive error, never a silent fallback to a lower layer or a default.
var (
	// ErrNoBinding: no binding resolves for the operation in scope. Absence is
	// an error, not an implicit default.
	ErrNoBinding = errors.New("no operation binding resolves in scope")
	// ErrInvalidOverride: the highest-precedence candidate is malformed or
	// explicitly disabled. Resolution stops — it does not fall through.
	ErrInvalidOverride = errors.New("highest-precedence binding is invalid")
	// ErrIncompatibleMode: the winning binding names a mode incompatible with
	// the operation/target.
	ErrIncompatibleMode = errors.New("bound mode is incompatible with the operation target")
	// ErrDeletedRevision: the pinned mode revision no longer exists.
	ErrDeletedRevision = errors.New("bound mode revision is missing")
)

// OperationBinding is the typed shape of operation-binding.schema.json.
type OperationBinding struct {
	Kind             string        `json:"kind"`
	Operation        OperationID   `json:"operation"`
	OperationVersion string        `json:"operation_version,omitempty"`
	Layer            BindingLayer  `json:"layer"`
	Owner            *BindingOwner `json:"owner,omitempty"`
	Mode             string        `json:"mode"`
	ModeRevision     string        `json:"mode_revision"`
	Disabled         bool          `json:"disabled,omitempty"`
}

type BindingOwner struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// ResolutionScope names the in-scope owner ids per override layer for one
// resolution. A binding is a candidate only when its layer's scope matches:
// a backlog-item-override applies only when its owner id equals ItemRef, etc.
// system-default bindings are always in scope.
type ResolutionScope struct {
	InvocationID   string
	ItemRef        string
	InitiativeName string
	Target         TargetKind
}

// ModeChecker answers the runtime questions the binding data cannot: whether a
// pinned mode revision still exists and whether the bound mode can implement
// the operation for the resolved target. It is injected so the resolver stays
// pure and testable.
type ModeChecker interface {
	// RevisionExists reports whether mode@revision is still registered.
	RevisionExists(mode, revision string) bool
	// ModeCompatible reports whether mode can implement operation for target.
	ModeCompatible(mode string, operation OperationID, target TargetKind) bool
}

// ResolvedBinding is the snapshot the resolver returns: the winning layer and
// the exact mode revision it pinned.
type ResolvedBinding struct {
	Operation    OperationID
	Layer        BindingLayer
	Mode         string
	ModeRevision string
	Owner        *BindingOwner
}

// ValidateBinding validates a binding document against the schema and the
// semantic rules: the operation is registered, the layer is known, and an
// override layer's owner kind matches its layer.
func ValidateBinding(raw []byte) error {
	if err := ValidateDocument(SchemaOperationBinding, raw); err != nil {
		return err
	}
	var b OperationBinding
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("decode operation binding: %w", err)
	}
	if !IsValidOperationID(b.Operation) {
		return fmt.Errorf("binding names unregistered operation %q", b.Operation)
	}
	if _, ok := layerRank[b.Layer]; !ok {
		return fmt.Errorf("binding has unknown layer %q", b.Layer)
	}
	switch b.Layer {
	case LayerSystemDefault:
		if b.Owner != nil {
			return fmt.Errorf("system-default binding must not name an owner")
		}
	default:
		if b.Owner == nil {
			return fmt.Errorf("%s binding must name an owner", b.Layer)
		}
		if err := checkOwnerKindForLayer(b.Layer, b.Owner.Kind); err != nil {
			return err
		}
	}
	return nil
}

func checkOwnerKindForLayer(layer BindingLayer, ownerKind string) error {
	want := map[BindingLayer]string{
		LayerAuthorizedInvocation: "invocation",
		LayerBacklogItemOverride:  "backlog-item",
		LayerInitiativeOverride:   "initiative",
	}[layer]
	if ownerKind != want {
		return fmt.Errorf("%s binding owner kind must be %q (got %q)", layer, want, ownerKind)
	}
	return nil
}

// inScope reports whether a binding applies given the resolution scope.
func (b OperationBinding) inScope(scope ResolutionScope) bool {
	switch b.Layer {
	case LayerSystemDefault:
		return true
	case LayerInitiativeOverride:
		return b.Owner != nil && b.Owner.ID == scope.InitiativeName && scope.InitiativeName != ""
	case LayerBacklogItemOverride:
		return b.Owner != nil && b.Owner.ID == scope.ItemRef && scope.ItemRef != ""
	case LayerAuthorizedInvocation:
		return b.Owner != nil && b.Owner.ID == scope.InvocationID && scope.InvocationID != ""
	default:
		return false
	}
}

// ResolveBinding deterministically resolves the winning binding for an
// operation. It considers only in-scope candidates for the operation, picks the
// highest-precedence layer, and fails closed on every ambiguous or invalid
// state:
//
//   - absence: no in-scope candidate → ErrNoBinding (never a silent default).
//   - invalid override: the highest-precedence candidate is disabled or
//     malformed → ErrInvalidOverride, with NO fallback to a lower layer.
//   - deleted revision: the winner's pinned revision is missing → ErrDeletedRevision.
//   - incompatible mode: the winner's mode cannot implement the operation for
//     the target → ErrIncompatibleMode.
func ResolveBinding(op OperationID, candidates []OperationBinding, scope ResolutionScope, checker ModeChecker) (ResolvedBinding, error) {
	var winner *OperationBinding
	winnerRank := 0
	for i := range candidates {
		b := candidates[i]
		if b.Operation != op || !b.inScope(scope) {
			continue
		}
		rank := layerRank[b.Layer]
		if rank == 0 {
			return ResolvedBinding{}, fmt.Errorf("%w: candidate has unknown layer %q", ErrInvalidOverride, b.Layer)
		}
		if rank > winnerRank {
			winnerRank = rank
			winner = &candidates[i]
		} else if rank == winnerRank && winner != nil {
			// Two bindings at the same precedence layer for the same operation
			// and scope is an ambiguous authoring error — fail closed rather
			// than pick arbitrarily.
			return ResolvedBinding{}, fmt.Errorf("%w: two %s bindings for operation %q in the same scope", ErrInvalidOverride, b.Layer, op)
		}
	}
	if winner == nil {
		return ResolvedBinding{}, fmt.Errorf("%w: operation %q", ErrNoBinding, op)
	}
	// The highest-precedence binding decides. A disabled winner is an explicit
	// veto that fails closed; it never falls through to a lower layer.
	if winner.Disabled {
		return ResolvedBinding{}, fmt.Errorf("%w: %s binding for operation %q is disabled", ErrInvalidOverride, winner.Layer, op)
	}
	if winner.Mode == "" || winner.ModeRevision == "" {
		return ResolvedBinding{}, fmt.Errorf("%w: %s binding for operation %q has no mode/revision", ErrInvalidOverride, winner.Layer, op)
	}
	if checker != nil {
		if !checker.RevisionExists(winner.Mode, winner.ModeRevision) {
			return ResolvedBinding{}, fmt.Errorf("%w: %s@%s (operator must re-bind)", ErrDeletedRevision, winner.Mode, winner.ModeRevision)
		}
		if !checker.ModeCompatible(winner.Mode, op, scope.Target) {
			return ResolvedBinding{}, fmt.Errorf("%w: mode %q cannot run operation %q on target %q", ErrIncompatibleMode, winner.Mode, op, scope.Target)
		}
	}
	return ResolvedBinding{
		Operation: op, Layer: winner.Layer, Mode: winner.Mode,
		ModeRevision: winner.ModeRevision, Owner: winner.Owner,
	}, nil
}
