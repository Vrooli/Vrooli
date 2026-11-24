// Package permissions composes owner-defined permission providers for the
// onboarding read/authorize/write surface. It routes by owner-declared IDs;
// policy and effects stay with the domain provider.
package permissions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const ContractVersion = "onboarding-permission/v1"

type Operation string

const (
	OperationRead  Operation = "read"
	OperationWrite Operation = "write"
)

type TargetBinding struct {
	MachineID            string `json:"machine_id,omitempty"`
	NodeID               string `json:"node_id,omitempty"`
	EnrollmentGeneration uint64 `json:"enrollment_generation,omitempty"`
}

type Descriptor struct {
	Version       string        `json:"version"`
	ID            string        `json:"id"`
	Owner         string        `json:"owner"`
	PrincipalKind string        `json:"principal_kind"`
	PrincipalID   string        `json:"principal_id,omitempty"`
	Scope         string        `json:"scope"`
	SubjectKind   string        `json:"subject_kind"`
	SubjectID     string        `json:"subject_id"`
	TargetBound   bool          `json:"target_bound"`
	Target        TargetBinding `json:"target,omitempty"`
	Source        string        `json:"source"`
	SchemaVersion string        `json:"schema_version"`
	CanRead       bool          `json:"can_read"`
	CanWrite      bool          `json:"can_write"`
	Sensitivity   string        `json:"sensitivity"`
}

func (d Descriptor) Validate() error {
	for key, value := range map[string]string{
		"version": d.Version, "id": d.ID, "owner": d.Owner,
		"principal_kind": d.PrincipalKind, "principal_id": d.PrincipalID, "scope": d.Scope,
		"subject_kind": d.SubjectKind, "subject_id": d.SubjectID,
		"source": d.Source, "schema_version": d.SchemaVersion, "sensitivity": d.Sensitivity,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("permission descriptor %q requires %s", d.ID, key)
		}
	}
	if d.Version != ContractVersion {
		return fmt.Errorf("permission %q has unsupported contract version %q", d.ID, d.Version)
	}
	if !d.CanRead && !d.CanWrite {
		return fmt.Errorf("permission %q exposes neither read nor write", d.ID)
	}
	if d.Sensitivity != "public" && d.Sensitivity != "operator" && d.Sensitivity != "secret" {
		return fmt.Errorf("permission %q has unsupported sensitivity %q", d.ID, d.Sensitivity)
	}
	if (d.Target.MachineID == "") != (d.Target.NodeID == "") {
		return fmt.Errorf("permission %q target binding must contain both machine and node", d.ID)
	}
	if d.TargetBound && d.Target.MachineID == "" {
		return fmt.Errorf("permission %q declares target binding without a target selector", d.ID)
	}
	if d.Target.MachineID != "" && d.Target.EnrollmentGeneration == 0 {
		return fmt.Errorf("permission %q target binding requires an enrollment generation", d.ID)
	}
	if d.Target.MachineID != "" && !d.TargetBound {
		return fmt.Errorf("permission %q declares a target without enabling target binding", d.ID)
	}
	return nil
}

type ReadRequest struct {
	ActorID string        `json:"actor_id"`
	Owner   string        `json:"owner"`
	Target  TargetBinding `json:"target"`
}

type ReadResult struct {
	PermissionID string        `json:"permission_id"`
	Owner        string        `json:"owner"`
	Target       TargetBinding `json:"target"`
	Revision     string        `json:"revision"`
	Value        []byte        `json:"value,omitempty"`
	ValueDigest  string        `json:"value_digest,omitempty"`
	ObservedAt   time.Time     `json:"observed_at"`
}

type WriteRequest struct {
	ActorID          string        `json:"actor_id"`
	Owner            string        `json:"owner"`
	Target           TargetBinding `json:"target"`
	ExpectedRevision string        `json:"expected_revision"`
	IdempotencyKey   string        `json:"idempotency_key"`
	Value            []byte        `json:"-"`
}

// Decision is the owning provider's authorization result. Permit is minted
// only by Registry after it binds that decision to the complete request.
type Decision struct {
	Allowed     bool
	DecisionRef string
}

// Permit is opaque outside this package and binds the owner decision to the
// precise operation, target generation, revision, payload and idempotency key.
type Permit struct {
	permissionID     string
	owner            string
	actorID          string
	target           TargetBinding
	expectedRevision string
	idempotencyKey   string
	valueDigest      string
	operation        Operation
	decisionRef      string
}

func (p Permit) DecisionRef() string { return p.decisionRef }

func (p Permit) ValidForRead(id string, req ReadRequest) bool {
	return p.operation == OperationRead && p.permissionID == id && p.owner == req.Owner && p.actorID == req.ActorID && p.target == req.Target
}

func (p Permit) ValidForWrite(id string, req WriteRequest) bool {
	return p.operation == OperationWrite && p.permissionID == id && p.owner == req.Owner && p.actorID == req.ActorID && p.target == req.Target && p.expectedRevision == req.ExpectedRevision && p.idempotencyKey == req.IdempotencyKey && p.valueDigest == digestValue(req.Value)
}

// Receipt contains audit metadata only. Permission values and secrets must
// never be copied into it.
type Receipt struct {
	ID               string        `json:"id"`
	PermissionID     string        `json:"permission_id"`
	Owner            string        `json:"owner"`
	ActorID          string        `json:"actor_id"`
	Target           TargetBinding `json:"target"`
	ExpectedRevision string        `json:"expected_revision"`
	ResultRevision   string        `json:"result_revision"`
	IdempotencyKey   string        `json:"idempotency_key"`
	RequestDigest    string        `json:"request_digest"`
	DecisionRef      string        `json:"decision_ref,omitempty"`
	Outcome          string        `json:"outcome"`
	OwnerState       string        `json:"owner_state,omitempty"`
	At               time.Time     `json:"at"`
}

type Provider interface {
	ProviderID() string
	Owner() string
	List(context.Context) ([]Descriptor, error)
	AuthorizeRead(context.Context, Descriptor, ReadRequest) (Decision, error)
	Read(context.Context, Descriptor, Permit, ReadRequest) (ReadResult, error)
	Authorize(context.Context, Descriptor, WriteRequest) (Decision, error)
	Write(context.Context, Descriptor, Permit, WriteRequest) (Receipt, error)
}

// PermissionIDMatcher is an optional cheap index. Providers can avoid live
// owner I/O when a lookup clearly belongs to another owner namespace.
type PermissionIDMatcher interface {
	MatchesPermissionID(string) bool
}

type Registry struct{ providers map[string]Provider }

func NewRegistry(providers ...Provider) (*Registry, error) {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		if provider == nil {
			return nil, errors.New("permission provider is nil")
		}
		providerID := strings.TrimSpace(provider.ProviderID())
		if providerID == "" || strings.TrimSpace(provider.Owner()) == "" {
			return nil, errors.New("permission provider id and owner are required")
		}
		if _, exists := r.providers[providerID]; exists {
			return nil, fmt.Errorf("permission provider %q is already registered", providerID)
		}
		r.providers[providerID] = provider
	}
	return r, nil
}

func (r *Registry) Inventory(ctx context.Context) ([]Descriptor, error) {
	entries, err := r.entriesForOwner(ctx, "")
	if err != nil {
		return nil, err
	}
	descriptors := make([]Descriptor, 0, len(entries))
	for _, entry := range entries {
		descriptors = append(descriptors, entry.descriptor)
	}
	return descriptors, nil
}

// InventoryForOwner limits remote discovery to one named owner. This preserves
// fail-closed owner errors without making an unavailable owner block unrelated
// provider inventories. The aggregate Inventory remains complete-or-error.
func (r *Registry) InventoryForOwner(ctx context.Context, owner string) ([]Descriptor, error) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, errors.New("permission inventory owner is required")
	}
	if r == nil {
		return nil, errors.New("permission registry is unavailable")
	}
	for _, provider := range r.providers {
		if provider.Owner() == owner {
			entries, err := r.entriesForOwner(ctx, owner)
			if err != nil {
				return nil, err
			}
			descriptors := make([]Descriptor, 0, len(entries))
			for _, entry := range entries {
				descriptors = append(descriptors, entry.descriptor)
			}
			return descriptors, nil
		}
	}
	return nil, fmt.Errorf("permission owner %q is not registered", owner)
}

// InventoryForProvider enumerates one registered source provider without
// probing sibling providers that may share an owner. It remains dynamically
// routed by provider ID and applies the same descriptor/owner validation as
// the aggregate inventory.
func (r *Registry) InventoryForProvider(ctx context.Context, providerID string) ([]Descriptor, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, errors.New("permission inventory provider is required")
	}
	if r == nil {
		return nil, errors.New("permission registry is unavailable")
	}
	provider, ok := r.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("permission provider %q is not registered", providerID)
	}
	entries, err := provider.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions from provider %q: %w", providerID, err)
	}
	descriptors := make([]Descriptor, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, descriptor := range entries {
		if err := descriptor.Validate(); err != nil {
			return nil, err
		}
		if descriptor.Owner != provider.Owner() {
			return nil, fmt.Errorf("permission %q owner %q does not match provider owner %q", descriptor.ID, descriptor.Owner, provider.Owner())
		}
		if _, exists := seen[descriptor.ID]; exists {
			return nil, fmt.Errorf("permission inventory contains duplicate %q", descriptor.ID)
		}
		seen[descriptor.ID] = struct{}{}
		descriptors = append(descriptors, descriptor)
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID < descriptors[j].ID })
	return descriptors, nil
}

type registeredPermission struct {
	provider   Provider
	descriptor Descriptor
}

func (r *Registry) entries(ctx context.Context) ([]registeredPermission, error) {
	return r.entriesForOwner(ctx, "")
}

func (r *Registry) entriesForOwner(ctx context.Context, ownerFilter string) ([]registeredPermission, error) {
	if r == nil {
		return nil, errors.New("permission registry is unavailable")
	}
	providerIDs := make([]string, 0, len(r.providers))
	for id := range r.providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	entries := make([]registeredPermission, 0, len(r.providers))
	seen := make(map[string]struct{})
	for _, providerID := range providerIDs {
		provider := r.providers[providerID]
		if ownerFilter != "" && provider.Owner() != ownerFilter {
			continue
		}
		descriptors, err := provider.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list permissions from owner %q: %w", provider.Owner(), err)
		}
		for _, d := range descriptors {
			if err := d.Validate(); err != nil {
				return nil, err
			}
			if d.Owner != provider.Owner() {
				return nil, fmt.Errorf("permission %q owner %q does not match provider owner %q", d.ID, d.Owner, provider.Owner())
			}
			if _, exists := seen[d.ID]; exists {
				return nil, fmt.Errorf("permission inventory contains duplicate %q", d.ID)
			}
			seen[d.ID] = struct{}{}
			entries = append(entries, registeredPermission{provider: provider, descriptor: d})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].descriptor.ID < entries[j].descriptor.ID })
	return entries, nil
}

func (r *Registry) Read(ctx context.Context, id string, req ReadRequest) (ReadResult, error) {
	p, d, err := r.provider(ctx, id)
	if err != nil {
		return ReadResult{}, err
	}
	if !d.CanRead {
		return ReadResult{}, fmt.Errorf("permission %q is not readable", id)
	}
	if err := validateRequest(d, req.ActorID, req.Owner, req.Target); err != nil {
		return ReadResult{}, err
	}
	decision, err := p.AuthorizeRead(ctx, d, req)
	if err != nil {
		return ReadResult{}, err
	}
	if !decision.Allowed {
		return ReadResult{}, fmt.Errorf("permission %q owner refused read authorization", id)
	}
	permit := Permit{permissionID: id, owner: d.Owner, actorID: req.ActorID, target: req.Target, operation: OperationRead}
	result, err := p.Read(ctx, d, permit, req)
	if err != nil {
		return ReadResult{}, err
	}
	if result.PermissionID != id || result.Owner != d.Owner || result.Target != req.Target || strings.TrimSpace(result.Revision) == "" {
		return ReadResult{}, fmt.Errorf("permission %q provider returned an unbound read result", id)
	}
	if d.Sensitivity == "secret" && len(result.Value) > 0 {
		return ReadResult{}, fmt.Errorf("permission %q returned a secret value", id)
	}
	return result, nil
}

func (r *Registry) Write(ctx context.Context, id string, req WriteRequest) (Receipt, error) {
	req.Value = bytes.Clone(req.Value)
	requestDigest := digestValue(req.Value)
	p, d, err := r.provider(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	if !d.CanWrite {
		return Receipt{}, fmt.Errorf("permission %q is not writable", id)
	}
	if err := validateRequest(d, req.ActorID, req.Owner, req.Target); err != nil {
		return Receipt{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return Receipt{}, errors.New("expected revision and idempotency key are required")
	}
	decision, err := p.Authorize(ctx, d, req)
	if err != nil {
		return Receipt{}, err
	}
	if !decision.Allowed {
		return Receipt{}, fmt.Errorf("permission %q owner refused write authorization", id)
	}
	permit := Permit{permissionID: id, owner: d.Owner, actorID: req.ActorID, target: req.Target, expectedRevision: req.ExpectedRevision, idempotencyKey: req.IdempotencyKey, valueDigest: requestDigest, operation: OperationWrite, decisionRef: decision.DecisionRef}
	receipt, err := p.Write(ctx, d, permit, req)
	if err != nil {
		return receipt, err
	}
	if receipt.ID == "" || receipt.PermissionID != id || receipt.Owner != d.Owner || receipt.ActorID != req.ActorID || receipt.Target != req.Target || receipt.ExpectedRevision != req.ExpectedRevision || receipt.ResultRevision == "" || receipt.IdempotencyKey != req.IdempotencyKey || receipt.RequestDigest != requestDigest || receipt.DecisionRef != decision.DecisionRef || receipt.Outcome == "" || receipt.At.IsZero() {
		return Receipt{}, fmt.Errorf("permission %q provider returned an incomplete or unbound receipt", id)
	}
	return receipt, nil
}

func (r *Registry) provider(ctx context.Context, id string) (Provider, Descriptor, error) {
	if r == nil {
		return nil, Descriptor{}, errors.New("permission registry is unavailable")
	}
	requestedID := strings.TrimSpace(id)
	providerIDs := make([]string, 0, len(r.providers))
	for providerID := range r.providers {
		providerIDs = append(providerIDs, providerID)
	}
	sort.Strings(providerIDs)
	var found *registeredPermission
	var listErrors []error
	for _, providerID := range providerIDs {
		provider := r.providers[providerID]
		if matcher, ok := provider.(PermissionIDMatcher); ok && !matcher.MatchesPermissionID(requestedID) {
			continue
		}
		descriptors, err := provider.List(ctx)
		if err != nil {
			listErrors = append(listErrors, fmt.Errorf("list permissions from owner %q: %w", provider.Owner(), err))
			continue
		}
		for _, d := range descriptors {
			if d.ID != requestedID {
				continue
			}
			if err := d.Validate(); err != nil {
				return nil, Descriptor{}, err
			}
			if d.Owner != provider.Owner() {
				return nil, Descriptor{}, fmt.Errorf("permission %q owner %q does not match provider owner %q", d.ID, d.Owner, provider.Owner())
			}
			if found != nil {
				return nil, Descriptor{}, fmt.Errorf("permission inventory contains duplicate %q", requestedID)
			}
			found = &registeredPermission{provider: provider, descriptor: d}
		}
	}
	if found != nil {
		return found.provider, found.descriptor, nil
	}
	if len(listErrors) > 0 {
		return nil, Descriptor{}, errors.Join(listErrors...)
	}
	return nil, Descriptor{}, fmt.Errorf("permission %q is unknown", id)
}

func validateRequest(d Descriptor, actor, owner string, target TargetBinding) error {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(owner) != d.Owner || actor != d.PrincipalID {
		return fmt.Errorf("permission %q owner or actor does not match", d.ID)
	}
	if d.TargetBound {
		if target.MachineID == "" || target.NodeID == "" || target.EnrollmentGeneration == 0 {
			return fmt.Errorf("permission %q requires a current machine/node/generation target binding", d.ID)
		}
		if d.Target.MachineID != "" && target != d.Target {
			return fmt.Errorf("permission %q target binding is stale or out of scope", d.ID)
		}
	} else if target != (TargetBinding{}) {
		return fmt.Errorf("permission %q is not bound to the requested target", d.ID)
	}
	return nil
}

func digestValue(value []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(value))
}
