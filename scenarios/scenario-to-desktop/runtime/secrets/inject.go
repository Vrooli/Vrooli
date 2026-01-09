package secrets

import (
	"fmt"
	"path/filepath"
	"strings"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
)

// Injector injects secrets into service environments.
type Injector struct {
	Store   Store
	FS      infra.FileSystem
	AppData string
}

// NewInjector creates a new Injector.
func NewInjector(store Store, fs infra.FileSystem, appData string) *Injector {
	return &Injector{
		Store:   store,
		FS:      fs,
		AppData: appData,
	}
}

// Apply injects secrets into the environment for a service.
// Secrets can be injected as environment variables or written to files.
func (inj *Injector) Apply(env map[string]string, svc manifest.Service) error {
	secrets := inj.Store.Get()
	for _, secretID := range svc.Secrets {
		secret := inj.Store.FindSecret(secretID)
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
			if err := inj.writeToFile(secret, value, env); err != nil {
				return err
			}

		default:
			return fmt.Errorf("secret %s has unsupported target type %s", secretID, secret.Target.Type)
		}
	}
	return nil
}

// writeToFile writes a secret value to a file and adds the path to env.
func (inj *Injector) writeToFile(secret *manifest.Secret, value string, env map[string]string) error {
	if secret.Target.Name == "" {
		return fmt.Errorf("secret %s missing file path target", secret.ID)
	}

	path := manifest.ResolvePath(inj.AppData, secret.Target.Name)
	if err := inj.FS.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secret %s path setup: %w", secret.ID, err)
	}
	if err := inj.FS.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("secret %s write: %w", secret.ID, err)
	}

	// Add file path to environment for service to discover.
	envName := fmt.Sprintf("SECRET_FILE_%s", strings.ToUpper(secret.ID))
	env[envName] = path
	return nil
}
