package credentials

import (
	"context"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// SSHKeyDescriptor is the credential address of the operator-held SSH key a
// deployment reaches its target with over the ssh transport. The binding
// references the key by file location; the bytes never enter the ledger.
var SSHKeyDescriptor = domain.CredentialDescriptor{LogicalID: "vrooli/scenario-to-cloud", Field: "ssh-key"}

// NewSSHKeyBinding builds the binding that names the operator-held key file
// for a deployment. Class is machine enrollment (the credential that lets
// the cloud reach the machine); the source class is infrastructure (the
// value is operator-held, never generated or fetched); the file target is
// the locator of the key on the operator side.
func NewSSHKeyBinding(deploymentID, keyPath string, now time.Time) *domain.CredentialBinding {
	keyPath = strings.TrimSpace(keyPath)
	return &domain.CredentialBinding{
		ID:           BindingID(deploymentID, SSHKeyDescriptor),
		DeploymentID: deploymentID,
		Descriptor:   SSHKeyDescriptor,
		Class:        domain.CredentialClassMachineEnrollment,
		SourceClass:  domain.SecretClassInfrastructure,
		Target:       domain.BundleSecretTarget{Type: "file", Name: keyPath},
		ConsumerRefs: []string{"scenario-to-cloud"},
		State:        domain.CredentialBindingMaterialized,
		Version:      domain.CredentialVersion{Number: 1, ContentRef: "operator-held", CreatedAt: now},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ResolveSSHKeyPath returns the key file the deployment's SSH-key binding
// names, or "" when the deployment holds no such binding (the operator's
// ambient identity authenticates then). A revoked binding yields "".
func ResolveSSHKeyPath(ctx context.Context, store Store, deploymentID string) (string, error) {
	if store == nil || strings.TrimSpace(deploymentID) == "" {
		return "", nil
	}
	binding, err := store.GetBinding(ctx, deploymentID, BindingID(deploymentID, SSHKeyDescriptor))
	if err != nil {
		return "", err
	}
	if binding == nil || binding.State == domain.CredentialBindingRevoked || binding.Target.Type != "file" {
		return "", nil
	}
	return strings.TrimSpace(binding.Target.Name), nil
}
