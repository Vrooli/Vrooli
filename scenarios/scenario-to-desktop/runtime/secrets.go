package bundleruntime

import (
	"fmt"
	"strings"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
)

// secretsCopy returns a thread-safe copy of the current secrets.
// Delegates to the injected SecretStore.
func (s *Supervisor) secretsCopy() map[string]string {
	return s.secretStore.Get()
}

// missingRequiredSecrets returns a list of required secrets that are missing.
// Delegates to the injected SecretStore.
func (s *Supervisor) missingRequiredSecrets() []string {
	return s.secretStore.MissingRequired()
}

// missingRequiredSecretsFrom checks a secrets map for missing required values.
// Delegates to the injected SecretStore.
func (s *Supervisor) missingRequiredSecretsFrom(secretsMap map[string]string) []string {
	return s.secretStore.MissingRequiredFrom(secretsMap)
}

// persistSecrets saves secrets to the secrets file.
// Delegates to the injected SecretStore.
func (s *Supervisor) persistSecrets(secretsMap map[string]string) error {
	return s.secretStore.Persist(secretsMap)
}

// findSecret looks up a secret definition by ID.
// Delegates to the injected SecretStore.
func (s *Supervisor) findSecret(id string) *manifest.Secret {
	return s.secretStore.FindSecret(id)
}

// applySecrets injects secrets into the environment for a service.
// Delegates to secrets.Injector.
func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	injector := secrets.NewInjector(s.secretStore, s.fs, s.appData)
	return injector.Apply(env, svc)
}

// UpdateSecrets merges new secrets and persists them.
// Triggers service startup if all required secrets are now available.
func (s *Supervisor) UpdateSecrets(newSecrets map[string]string) error {
	merged := s.secretStore.Merge(newSecrets)

	missing := s.missingRequiredSecretsFrom(merged)
	if len(missing) > 0 {
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}

	if err := s.persistSecrets(merged); err != nil {
		return err
	}

	s.secretStore.Set(merged)

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})

	// Trigger service startup if not already started.
	if !s.servicesStarted {
		s.startServicesAsync()
	}
	return nil
}
