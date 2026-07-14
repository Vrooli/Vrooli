package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// knownHostsFileName is the bridge-owned known_hosts inside the state dir.
const knownHostsFileName = "known_hosts"

// Service owns the bridge's first-touch SSH capability: a state directory
// holding the onboarding keypair + known_hosts, plus the seams for local
// command execution (ssh-keygen), remote command execution (system ssh), and
// password-authenticated key copy (x/crypto). The seams are injectable so tests
// can drive the flow against an in-process sshd or fakes.
type Service struct {
	stateDir string
	cmd      CommandRunner
	runner   Runner
	copier   KeyCopier
}

// Option configures a Service.
type Option func(*Service)

// WithCommandRunner overrides the local command runner (ssh-keygen).
func WithCommandRunner(c CommandRunner) Option {
	return func(s *Service) {
		if c != nil {
			s.cmd = c
		}
	}
}

// WithRunner overrides the remote command runner (system ssh test path).
func WithRunner(r Runner) Option {
	return func(s *Service) {
		if r != nil {
			s.runner = r
		}
	}
}

// WithKeyCopier overrides the password-authenticated key copier.
func WithKeyCopier(k KeyCopier) Option {
	return func(s *Service) {
		if k != nil {
			s.copier = k
		}
	}
}

// NewService constructs a Service rooted at stateDir with the production seams.
func NewService(stateDir string, opts ...Option) *Service {
	s := &Service{
		stateDir: stateDir,
		cmd:      ExecCommandRunner{},
		runner:   ExecRunner{},
		copier:   ExecKeyCopier{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// StateDir returns the directory holding the onboarding key material.
func (s *Service) StateDir() string { return s.stateDir }

// knownHostsPath returns the bridge-owned known_hosts path.
func (s *Service) knownHostsPath() string {
	return filepath.Join(s.stateDir, knownHostsFileName)
}

// ensureDir0700 creates dir if missing and enforces owner-only (0700) perms
// even when it already existed with a looser mode — the key material and
// known_hosts must never be group/world-readable.
func ensureDir0700(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	return nil
}

// validateKeyPath ensures keyPath resolves inside the state dir (no traversal).
func (s *Service) validateKeyPath(keyPath string) error {
	if strings.Contains(keyPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	abs, err := filepath.Abs(keyPath)
	if err != nil {
		return fmt.Errorf("invalid key path: %w", err)
	}
	base, err := filepath.Abs(s.stateDir)
	if err != nil {
		return fmt.Errorf("invalid state dir: %w", err)
	}
	if abs != base && !strings.HasPrefix(abs, base+string(os.PathSeparator)) {
		return fmt.Errorf("key path must be within the bridge SSH state dir")
	}
	return nil
}

// ResolveStateDir returns the bridge-owned directory that holds the onboarding
// SSH keypair + known_hosts. It mirrors main.cpKeyDir so the material lands in
// the same variant-aware (shadow-safe) storage namespace as the SQLite DB and
// the control-plane identity key. BRIDGE_SSH_STATE_DIR overrides for tests/ops.
func ResolveStateDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("BRIDGE_SSH_STATE_DIR")); dir != "" {
		return dir, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-bridge")
	if err != nil {
		return "", fmt.Errorf("resolve vrooli-bridge storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		filepath.Join("onboard-ssh", ".keep"),
	)
	if err != nil {
		return "", fmt.Errorf("resolve onboard-ssh state dir: %w", err)
	}
	return filepath.Dir(path), nil
}
