package bundleruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// loadSecrets reads persisted secrets from the secrets file.
// Returns an empty map if the file doesn't exist.
func (s *Supervisor) loadSecrets() (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(s.secretsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}

	// Try new format first: {"secrets": {...}}
	var wrapper struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil || wrapper.Secrets == nil {
		// Fall back to legacy flat format.
		var legacy map[string]string
		if err2 := json.Unmarshal(data, &legacy); err2 == nil {
			return legacy, nil
		}
		return nil, err
	}
	return wrapper.Secrets, nil
}

// persistSecrets saves secrets to the secrets file.
// The file is created with 0600 permissions for security.
func (s *Supervisor) persistSecrets(secrets map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.secretsPath), 0o700); err != nil {
		return err
	}

	payload := struct {
		Secrets map[string]string `json:"secrets"`
	}{
		Secrets: secrets,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.secretsPath, data, 0o600)
}

// secretsCopy returns a thread-safe copy of the current secrets.
func (s *Supervisor) secretsCopy() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.secrets))
	for k, v := range s.secrets {
		out[k] = v
	}
	return out
}

// missingRequiredSecrets returns a list of required secrets that are missing.
func (s *Supervisor) missingRequiredSecrets() []string {
	return s.missingRequiredSecretsFrom(s.secretsCopy())
}

// missingRequiredSecretsFrom checks a secrets map for missing required values.
func (s *Supervisor) missingRequiredSecretsFrom(secrets map[string]string) []string {
	var missing []string
	for _, sec := range s.opts.Manifest.Secrets {
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}
		if !required {
			continue
		}
		val := strings.TrimSpace(secrets[sec.ID])
		if val == "" {
			missing = append(missing, sec.ID)
		}
	}
	return missing
}

// findSecret looks up a secret definition by ID.
func (s *Supervisor) findSecret(id string) *manifest.Secret {
	for i := range s.opts.Manifest.Secrets {
		if s.opts.Manifest.Secrets[i].ID == id {
			return &s.opts.Manifest.Secrets[i]
		}
	}
	return nil
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
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secret %s path setup: %w", secret.ID, err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
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
	merged := s.secretsCopy()
	for k, v := range newSecrets {
		merged[k] = v
	}

	missing := s.missingRequiredSecretsFrom(merged)
	if len(missing) > 0 {
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}

	if err := s.persistSecrets(merged); err != nil {
		return err
	}

	s.mu.Lock()
	s.secrets = merged
	s.mu.Unlock()

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})

	// Trigger service startup if not already started.
	if !s.servicesStarted {
		s.startServicesAsync()
	}
	return nil
}
