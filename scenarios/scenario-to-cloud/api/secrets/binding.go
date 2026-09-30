package secrets

import (
	"context"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
)

// BindingLifecycle is the post-deploy secrets management surface expressed on
// the credential binding model: keys are exact descriptors on the
// deployment, creation materialises a version, replacement is a rotation,
// deletion is a revocation. Values travel in and never out.
type BindingLifecycle interface {
	List(ctx context.Context) ([]domain.VPSSecretEntry, error)
	Get(ctx context.Context, key string) (*domain.VPSSecretEntry, error)
	Create(ctx context.Context, key, value string) (*domain.CredentialBinding, error)
	Replace(ctx context.Context, key, value string) (*domain.CredentialRotation, error)
	Delete(ctx context.Context, key string) (*domain.CredentialRotation, error)
	Restart(ctx context.Context) error
}

// bindingLifecycle binds one deployment's lifecycle service.
type bindingLifecycle struct {
	service  *credentials.Service
	target   credentials.Target
	manifest domain.CloudManifest
}

// NewBindingLifecycle adapts a bound credentials.Service to the management
// surface.
func NewBindingLifecycle(service *credentials.Service, target credentials.Target, manifest domain.CloudManifest) BindingLifecycle {
	return &bindingLifecycle{service: service, target: target, manifest: manifest}
}

// descriptorForKey resolves a management key to the exact descriptor. A key
// that names a declared secret (by id or target name) uses that secret's
// declared descriptor; anything else is an operator-added credential under
// the scenario's own identity with the normalised field name.
func (l *bindingLifecycle) descriptorForKey(key string) (domain.CredentialDescriptor, *domain.BundleSecretPlan) {
	trimmed := strings.TrimSpace(key)
	var plans []domain.BundleSecretPlan
	if l.manifest.Secrets != nil {
		plans = l.manifest.Secrets.BundleSecrets
	}
	for i := range plans {
		plan := &plans[i]
		if plan.Descriptor == nil || strings.TrimSpace(plan.Descriptor.Field) == "" {
			continue
		}
		if plan.ID == trimmed || plan.Target.Name == trimmed || plan.Descriptor.Field == trimmed {
			return domain.CredentialDescriptor{LogicalID: plan.Descriptor.LogicalID, Field: plan.Descriptor.Field}, plan
		}
	}
	return domain.CredentialDescriptor{LogicalID: "vrooli/" + strings.TrimSpace(l.manifest.Scenario.ID), Field: CredentialField(trimmed)}, nil
}

func entryFor(view credentials.BindingView) domain.VPSSecretEntry {
	b := view.Binding
	key := b.Target.Name
	if strings.TrimSpace(key) == "" {
		key = b.Descriptor.Field
	}
	return domain.VPSSecretEntry{
		Key: key, Masked: true, Source: "credential-binding",
		LastUpdated: b.UpdatedAt.UTC().Format(time.RFC3339),
		BindingID:   b.ID, Descriptor: b.Descriptor.Address(), Class: string(b.Class), Version: b.Version.Number, State: string(b.State),
	}
}

func (l *bindingLifecycle) List(ctx context.Context) ([]domain.VPSSecretEntry, error) {
	views, err := l.service.ListBindings(ctx, l.target.DeploymentID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.VPSSecretEntry, 0, len(views))
	for _, view := range views {
		out = append(out, entryFor(view))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (l *bindingLifecycle) find(ctx context.Context, key string) (*domain.CredentialBinding, error) {
	descriptor, _ := l.descriptorForKey(key)
	binding, err := l.service.Store.GetBinding(ctx, l.target.DeploymentID, credentials.BindingID(l.target.DeploymentID, descriptor))
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, apierrors.New(credentials.CodeBindingNotFound, "no credential binding for this key on the deployment").WithDetail("key", key).WithDetail("descriptor", descriptor.Address())
	}
	return binding, nil
}

func (l *bindingLifecycle) Get(ctx context.Context, key string) (*domain.VPSSecretEntry, error) {
	binding, err := l.find(ctx, key)
	if err != nil {
		return nil, err
	}
	acks, err := l.service.Store.ListAcks(ctx, binding.ID)
	if err != nil {
		return nil, err
	}
	entry := entryFor(credentials.BindingView{Binding: *binding, Acks: acks})
	return &entry, nil
}

// Create binds a new key and materialises its first version from the
// operator value. An existing materialised binding is a conflict; a revoked
// or planned one is materialised again.
func (l *bindingLifecycle) Create(ctx context.Context, key, value string) (*domain.CredentialBinding, error) {
	descriptor, plan := l.descriptorForKey(key)
	id := credentials.BindingID(l.target.DeploymentID, descriptor)
	existing, err := l.service.Store.GetBinding(ctx, l.target.DeploymentID, id)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.State == domain.CredentialBindingMaterialized {
		return nil, apierrors.New(credentials.CodeRotationConflict, "the key already has an active credential version; replace it instead").WithDetail("binding_id", id).WithDetail("version", existing.Version.Number)
	}
	now := time.Now().UTC()
	binding := domain.CredentialBinding{
		ID: id, DeploymentID: l.target.DeploymentID, Descriptor: descriptor,
		Class: domain.CredentialClassExternalAPICredential, SourceClass: domain.SecretClassUserPrompt,
		Target:       domain.BundleSecretTarget{Type: "env", Name: strings.TrimSpace(key)},
		ConsumerRefs: []string{"scenario:" + strings.TrimSpace(l.manifest.Scenario.ID)},
		State:        domain.CredentialBindingPlanned, CreatedAt: now, UpdatedAt: now,
	}
	if plan != nil {
		binding.Target = plan.Target
		binding.SourceClass = plan.Class
		binding.Class = credentials.Classify(*plan, descriptor, binding.ConsumerRefs, nil)
	}
	if existing != nil {
		binding = *existing
		binding.State = domain.CredentialBindingPlanned
	}
	result, err := l.service.Materialize(ctx, credentials.MaterializeRequest{Target: l.target, Bindings: []domain.CredentialBinding{binding}, OperatorValues: map[string]string{id: value}})
	if err != nil {
		return nil, err
	}
	if len(result.Materialized) != 1 {
		return nil, apierrors.New(credentials.CodeDistributionFailed, "the credential was not materialised").WithDetail("result", result)
	}
	return l.service.Store.GetBinding(ctx, l.target.DeploymentID, id)
}

// Replace rotates the key to the operator value.
func (l *bindingLifecycle) Replace(ctx context.Context, key, value string) (*domain.CredentialRotation, error) {
	binding, err := l.find(ctx, key)
	if err != nil {
		return nil, err
	}
	return l.service.Rotate(ctx, credentials.RotateRequest{Target: l.target, DeploymentID: l.target.DeploymentID, BindingID: binding.ID, Value: value})
}

// Delete revokes the key's active version.
func (l *bindingLifecycle) Delete(ctx context.Context, key string) (*domain.CredentialRotation, error) {
	binding, err := l.find(ctx, key)
	if err != nil {
		return nil, err
	}
	return l.service.Revoke(ctx, credentials.RevokeBindingRequest{Target: l.target, DeploymentID: l.target.DeploymentID, BindingID: binding.ID})
}

// Restart restarts the deployed scenario through the lifecycle restarter.
func (l *bindingLifecycle) Restart(ctx context.Context) error {
	if l.service.Restarter == nil {
		return apierrors.New(apierrors.CodeUnsupportedCapability, "this transport has no consumer restart path")
	}
	return l.service.Restarter.Restart(ctx, l.target, []string{"scenario:" + strings.TrimSpace(l.manifest.Scenario.ID)})
}
