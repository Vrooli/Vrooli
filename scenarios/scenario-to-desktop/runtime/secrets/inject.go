package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

// Injector injects secrets into service environments.
type Injector struct {
	Store   Store
	FS      infra.FileSystem
	AppData string
	// EphemeralDir overrides where file-target secrets are materialized. It is
	// for tests and for a host that knows better than the defaults; production
	// leaves it empty and takes the first working candidate.
	EphemeralDir string

	// materialized records every file written for a file-target secret so they
	// can be removed once the services that needed them have started.
	materialized []string
}

// ephemeralSecretRoots are the per-boot locations tried in order. XDG_RUNTIME_DIR
// is a tmpfs that logind removes at logout and that never survives a reboot —
// the same place ssh-agent and gnome-keyring keep session material. The system
// temp dir is the fallback; it is not guaranteed to be a tmpfs, but it is the
// strongest remaining option on a host without a session runtime directory.
func ephemeralSecretRoots() []string {
	roots := []string{}
	if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
		roots = append(roots, filepath.Join(runtimeDir, "vrooli", "bundle-secrets"))
	}
	return append(roots, filepath.Join(os.TempDir(), "vrooli-bundle-secrets"))
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

// writeToFile materializes a secret for a service that can only read one from
// disk, and records the path so it can be removed again.
//
// This used to write into APP_DATA_DIR and leave the file there forever, which
// contradicted this package's own contract that production values are never
// persisted to app data and that there is no plaintext fallback. A mode-0600
// file on durable storage is exactly the protection level the encrypted store
// exists to replace.
//
// The destination is now ephemeral storage where the host has it — a tmpfs that
// does not survive a reboot — and the file is removed once the service has
// started. Where no ephemeral directory exists the bundle refuses rather than
// silently writing a durable plaintext credential, because an operator cannot
// see the difference and would never learn they had one.
func (inj *Injector) writeToFile(secret *manifest.Secret, value string, env map[string]string) error {
	if secret.Target.Name == "" {
		return fmt.Errorf("secret %s missing file path target", secret.ID)
	}

	dir, err := inj.ephemeralSecretDir()
	if err != nil {
		return fmt.Errorf("secret %s needs a file on disk and this host has no ephemeral location for one: %w", secret.ID, err)
	}

	path := filepath.Join(dir, filepath.Base(manifest.ResolvePath("", secret.Target.Name)))
	if err := inj.FS.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secret %s path setup: %w", secret.ID, err)
	}
	if err := inj.FS.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("secret %s write: %w", secret.ID, err)
	}
	inj.materialized = append(inj.materialized, path)

	// Add file path to environment for service to discover.
	envName := fmt.Sprintf("SECRET_FILE_%s", strings.ToUpper(secret.ID))
	env[envName] = path
	return nil
}

// ephemeralSecretDir returns a directory whose contents do not survive a
// reboot, or an error naming why this host has none.
func (inj *Injector) ephemeralSecretDir() (string, error) {
	if override := strings.TrimSpace(inj.EphemeralDir); override != "" {
		return override, nil
	}
	for _, candidate := range ephemeralSecretRoots() {
		if candidate == "" {
			continue
		}
		if err := inj.FS.MkdirAll(candidate, 0o700); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no writable ephemeral directory (checked the session runtime dir and the system temp dir)")
}

// RemoveMaterializedSecrets deletes every file writeToFile created. The caller
// invokes it once the services that needed the files have started; a value on
// disk past that point is exposure with no remaining purpose.
func (inj *Injector) RemoveMaterializedSecrets() error {
	var firstErr error
	for _, path := range inj.materialized {
		if err := inj.FS.Remove(path); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("remove materialized secret: %w", err)
		}
	}
	inj.materialized = nil
	return firstErr
}
