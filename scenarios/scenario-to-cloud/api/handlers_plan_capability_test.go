package main

import (
	"context"
	"errors"
	"testing"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// stubCredentialClient answers presence questions only.
type stubCredentialClient struct {
	credentialclient.Client
	configured map[string]bool
	err        error
}

func (c stubCredentialClient) Status(_ context.Context, identity, field string) (credentialclient.CredentialStatus, error) {
	if c.err != nil {
		return credentialclient.CredentialStatus{}, c.err
	}
	return credentialclient.CredentialStatus{
		Identity:   identity,
		Field:      field,
		Configured: c.configured[identity+":"+field],
	}, nil
}

func withCapabilityCredentials(t *testing.T, client credentialclient.Client, err error) {
	t.Helper()
	original := capabilityCredentialClient
	capabilityCredentialClient = func() (credentialclient.Client, error) { return client, err }
	t.Cleanup(func() { capabilityCredentialClient = original })
}

func capabilityManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{
			{
				ID: "sendgrid-api-key", Class: "user_prompt", Capability: "customer sign-in email",
				Target:     domain.BundleSecretTarget{Type: "env", Name: "SENDGRID_API_KEY"},
				Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "sendgrid-api-key"},
			},
			{
				ID: "smtp-password", Class: "user_prompt", Capability: "customer sign-in email (SMTP fallback)",
				Target:     domain.BundleSecretTarget{Type: "env", Name: "SMTP_PASSWORD"},
				Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "smtp-password"},
			},
		}},
	}
}

// The live gap: an optional credential nothing ever provisioned counted as
// "satisfied", so the capability warning could never fire.
func TestUnconfiguredCapabilityCredentialsNamesOnlyMissingValues(t *testing.T) {
	withCapabilityCredentials(t, stubCredentialClient{configured: map[string]bool{
		"vrooli/app:smtp-password": true,
	}}, nil)

	got := unconfiguredCapabilityCredentials(capabilityManifest())

	if len(got) != 1 || got[0] != "vrooli/app:sendgrid-api-key" {
		t.Fatalf("unconfigured = %v, want only the credential with no value", got)
	}
}

func TestUnconfiguredCapabilityCredentialsSilentWhenEveryValueExists(t *testing.T) {
	withCapabilityCredentials(t, stubCredentialClient{configured: map[string]bool{
		"vrooli/app:sendgrid-api-key": true,
		"vrooli/app:smtp-password":    true,
	}}, nil)

	if got := unconfiguredCapabilityCredentials(capabilityManifest()); len(got) != 0 {
		t.Errorf("unconfigured = %v, want none", got)
	}
}

// A warning must rest on evidence. An unreachable authority is not evidence
// that a credential is missing, so it must not manufacture warnings.
func TestUnconfiguredCapabilityCredentialsSilentWhenAuthorityIsUnavailable(t *testing.T) {
	withCapabilityCredentials(t, nil, errors.New("authority unavailable"))

	if got := unconfiguredCapabilityCredentials(capabilityManifest()); len(got) != 0 {
		t.Errorf("unconfigured = %v, want none when the authority cannot be asked", got)
	}

	withCapabilityCredentials(t, stubCredentialClient{err: errors.New("status failed")}, nil)

	if got := unconfiguredCapabilityCredentials(capabilityManifest()); len(got) != 0 {
		t.Errorf("unconfigured = %v, want none when a status read fails", got)
	}
}

// The observer turns those addresses into unsatisfied inputs, which is what
// the compiler reads when it decides whether to warn.
func TestRecordObserverDoesNotClaimUnconfiguredCapabilityCredentialsAreSatisfied(t *testing.T) {
	withCapabilityCredentials(t, stubCredentialClient{}, nil)
	manifest := capabilityManifest()
	closureDoc := &domain.Closure{Components: []domain.ClosureComponent{
		{
			ID: "vrooli/app:sendgrid-api-key", Kind: domain.ClosureKindCredentialDescriptor,
			Credential: &domain.ClosureCredential{LogicalID: "vrooli/app", Field: "sendgrid-api-key"},
		},
	}}

	obs, err := recordObserver{}.Observe(context.Background(), &domain.Deployment{}, manifest, closureDoc)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}

	for _, satisfied := range obs.SatisfiedInputs {
		if satisfied == "vrooli/app:sendgrid-api-key" {
			t.Fatal("an unconfigured capability credential must not be reported as satisfied")
		}
	}
}

func TestRecordObserverTreatsMaterializedPromptCredentialAsSatisfied(t *testing.T) {
	manifest := domain.CloudManifest{Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{
		{
			ID: "admin-default-password", Class: "user_prompt", Required: true,
			Target:     domain.BundleSecretTarget{Type: "env", Name: "ADMIN_DEFAULT_PASSWORD"},
			Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "admin-default-password"},
		},
	}}}
	closureDoc := &domain.Closure{Components: []domain.ClosureComponent{{
		ID: "vrooli/app:admin-default-password", Kind: domain.ClosureKindCredentialDescriptor,
		Credential: &domain.ClosureCredential{LogicalID: "vrooli/app", Field: "admin-default-password"},
	}}}
	observer := recordObserver{listBindings: func(context.Context, string) ([]credentials.BindingView, error) {
		return []credentials.BindingView{{Binding: domain.CredentialBinding{
			Descriptor: domain.CredentialDescriptor{LogicalID: "vrooli/app", Field: "admin-default-password"},
			State:      domain.CredentialBindingMaterialized,
			Version:    domain.CredentialVersion{Number: 1},
		}}}, nil
	}}

	obs, err := observer.Observe(context.Background(), &domain.Deployment{ID: "dep-1"}, manifest, closureDoc)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	for _, satisfied := range obs.SatisfiedInputs {
		if satisfied == "vrooli/app:admin-default-password" {
			return
		}
	}
	t.Fatalf("satisfied inputs = %v, want materialized prompt credential", obs.SatisfiedInputs)
}
