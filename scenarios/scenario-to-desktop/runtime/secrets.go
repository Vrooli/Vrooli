package bundleruntime

import (
	"fmt"
	"path/filepath"
	"strings"

	"scenario-to-desktop-runtime/manifest"
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
func (s *Supervisor) missingRequiredSecretsFrom(secrets map[string]string) []string {
	return s.secretStore.MissingRequiredFrom(secrets)
}

// persistSecrets saves secrets to the secrets file.
// Delegates to the injected SecretStore.
func (s *Supervisor) persistSecrets(secrets map[string]string) error {
	return s.secretStore.Persist(secrets)
}

// findSecret looks up a secret definition by ID.
// Delegates to the injected SecretStore.
func (s *Supervisor) findSecret(id string) *manifest.Secret {
	return s.secretStore.FindSecret(id)
}

// applySecrets injects secrets into the environment for a service.
// Secrets can be injected as environment variables or written to files.
func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	secrets := s.secretsCopy()
	for _, secretID := range svc.Secrets {
		secret := s.findSecret(secretID)
		if secret == nil {
			return fmt.Errorf("service %s references unknown secret %s", svc.ID, secretID)
		}

		value := strings.TrimSpace(secrets[secretID])
		required := true
		if secret.Required != nil {
			required = *secret.Required
		}

		if value == "" {
			if required {
				return fmt.Errorf("secret %s missing for service %s", secretID, svc.ID)
			}
			continue
		}

		switch secret.Target.Type {
		case "env":
			name := secret.Target.Name
			if name == "" {
				name = strings.ToUpper(secret.ID)
			}
			env[name] = value

		case "file":
			if err := s.writeSecretToFile(secret, value, env); err != nil {
				return err
			}

		default:
			return fmt.Errorf("secret %s has unsupported target type %s", secretID, secret.Target.Type)
		}
	}
	return nil
}

// writeSecretToFile writes a secret value to a file and adds the path to env.
func (s *Supervisor) writeSecretToFile(secret *manifest.Secret, value string, env map[string]string) error {
	if secret.Target.Name == "" {
		return fmt.Errorf("secret %s missing file path target", secret.ID)
	}

	path := manifest.ResolvePath(s.appData, secret.Target.Name)
	if err := s.fs.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secret %s path setup: %w", secret.ID, err)
	}
	if err := s.fs.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("secret %s write: %w", secret.ID, err)
	}

	// Add file path to environment for service to discover.
	envName := fmt.Sprintf("SECRET_FILE_%s", strings.ToUpper(secret.ID))
	env[envName] = path
	return nil
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
