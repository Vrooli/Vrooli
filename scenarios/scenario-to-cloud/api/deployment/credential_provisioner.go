package deployment

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"
)

// credentialProvisioner is the executor's credentials.provision and
// grants.revoke owner: it plans the manifest's descriptor bindings, then
// materialises them through the credential lifecycle bound to the deployment
// target (receipted ingest verbs over the bound transport). Values reach the
// lifecycle keyed by binding id and never appear in any receipt.
type credentialProvisioner struct {
	bind  CredentialBinder
	store credentials.Store
}

var _ vps.CredentialProvisioner = (*credentialProvisioner)(nil)

func (p *credentialProvisioner) plan(ctx context.Context, req vps.CredentialProvisionRequest) (*credentials.Service, credentials.Target, []domain.CredentialBinding, error) {
	service, err := p.bind(ctx, req.DeploymentID, req.Target, req.Manifest)
	if err != nil {
		return nil, credentials.Target{}, nil, err
	}
	if service.Store == nil {
		service.Store = p.store
	}
	target := credentials.Target{DeploymentID: req.DeploymentID, Ref: req.Target, Fence: req.Identity.Fence}
	var plans []domain.BundleSecretPlan
	if req.Manifest.Secrets != nil {
		plans = req.Manifest.Secrets.BundleSecrets
	}
	bindings, err := credentials.PlanBindings(credentials.PlanInputs{DeploymentID: req.DeploymentID, Plans: plans})
	if err != nil {
		return nil, credentials.Target{}, nil, err
	}
	return service, target, bindings, nil
}

// bindingValues re-keys secret-id keyed values by binding id.
func bindingValues(deploymentID string, plans []domain.BundleSecretPlan, bySecretID map[string]string) map[string]string {
	out := map[string]string{}
	for _, plan := range plans {
		if plan.Descriptor == nil {
			continue
		}
		value := bySecretID[plan.ID]
		if value == "" {
			value = bySecretID[strings.TrimSpace(plan.Target.Name)]
		}
		if value == "" {
			continue
		}
		id := credentials.BindingID(deploymentID, domain.CredentialDescriptor{LogicalID: plan.Descriptor.LogicalID, Field: plan.Descriptor.Field})
		out[id] = value
	}
	return out
}

func (p *credentialProvisioner) Provision(ctx context.Context, req vps.CredentialProvisionRequest) (vps.CredentialProvisionResult, error) {
	service, target, bindings, err := p.plan(ctx, req)
	if err != nil {
		return vps.CredentialProvisionResult{}, err
	}
	var plans []domain.BundleSecretPlan
	if req.Manifest.Secrets != nil {
		plans = req.Manifest.Secrets.BundleSecrets
	}
	result, err := service.Materialize(ctx, credentials.MaterializeRequest{
		Target:          target,
		Bindings:        bindings,
		OperationID:     req.Identity.OperationID,
		GeneratedValues: bindingValues(req.DeploymentID, plans, req.GeneratedValues),
		OperatorValues:  bindingValues(req.DeploymentID, plans, req.OperatorValues),
	})
	if err != nil {
		return vps.CredentialProvisionResult{}, err
	}
	return vps.CredentialProvisionResult{Preserved: result.Preserved, Materialized: result.Materialized, Skipped: result.Skipped}, nil
}

func (p *credentialProvisioner) Revoke(ctx context.Context, req vps.CredentialProvisionRequest) (vps.CredentialProvisionResult, error) {
	service, target, _, err := p.plan(ctx, req)
	if err != nil {
		return vps.CredentialProvisionResult{}, err
	}
	existing, err := service.Store.ListBindings(ctx, req.DeploymentID)
	if err != nil {
		return vps.CredentialProvisionResult{}, err
	}
	out := vps.CredentialProvisionResult{}
	for _, binding := range existing {
		if binding.Version.Number == 0 || binding.State == domain.CredentialBindingRevoked {
			continue
		}
		if _, err := service.Revoke(ctx, credentials.RevokeBindingRequest{Target: target, DeploymentID: req.DeploymentID, BindingID: binding.ID, RequestKey: req.Identity.OperationID + ":retire:" + binding.ID}); err != nil {
			return out, fmt.Errorf("revoke %s: %w", binding.ID, err)
		}
		out.Revoked = append(out.Revoked, binding.ID)
	}
	return out, nil
}
