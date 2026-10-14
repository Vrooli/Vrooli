// finite_native_fixture_export.go exposes a TEST-ONLY admission-only finite-native
// installer so cross-package fixtures (the disposable joined PM subprocess fixture in
// package handlers) can compose the same hardened admission the production startup path
// installs atomically with effort authority.
//
// This is test infrastructure, NOT a production behavior change. It never qualifies a
// host, starts a service, or launches an agent: the admission checker is a pure manifest
// composition that honors CheckBinding (policy/profile/repo/deadline) exactly like the
// established in-package fixture (installFixtureNativeFactory / fixtureNativeManifest /
// fixtureNativeAdmission / fixtureUnitTerminal in effort_admission_test.go). Production
// installation still requires a concrete factory with a captured continuing witness via
// InstallFiniteNativeIsolation.
package orchestration

import (
	"context"
	"fmt"
	"testing"

	"agent-manager/internal/domain"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
)

// fixtureFiniteNativeAdmission is synthetic admission-only evidence, never an installed
// launcher or host proof. It mirrors the unexported fixtureNativeAdmission.
type fixtureFiniteNativeAdmission struct{ runtime *isolation.Runtime }

func (f fixtureFiniteNativeAdmission) Enabled() bool { return f.runtime.Enabled() }
func (f fixtureFiniteNativeAdmission) CheckBinding(id string, b isolation.Binding) error {
	return f.runtime.CheckBinding(id, b)
}

// fixtureFiniteNativeTerminal is synthetic owner evidence; no real unit is launched.
type fixtureFiniteNativeTerminal struct{}

func (fixtureFiniteNativeTerminal) Terminal(context.Context, string) error { return nil }

// fixtureFiniteNativeManifest mirrors fixtureNativeManifest: it binds the policy id to the
// qualified profile digest, repository and deadline so CheckBinding genuinely qualifies the
// binding rather than trivially accepting everything.
func fixtureFiniteNativeManifest(p effortauthority.Policy, profile *domain.AgentProfile) isolation.Manifest {
	h := effortauthority.Digest("fixture executable")
	return isolation.Manifest{
		Enabled:         true,
		Backend:         "linux-systemd-service-v1",
		SystemdRun:      isolation.Executable{Path: "/usr/bin/systemd-run", SHA256: h},
		Systemctl:       isolation.Executable{Path: "/usr/bin/systemctl", SHA256: h},
		RootFS:          "/var/lib/finite-fixture/image",
		WorkspaceSource: "/var/lib/finite-fixture/worktree",
		RootFSDigest:    h,
		NativeUID:       1234,
		NativeGID:       1234,
		Isolation:       &effortauthority.NativeUIDIsolation{NativeUID: 1234, ProtectedPaths: []string{"/var/lib/finite-authority/key"}},
		Bindings: map[string]isolation.Binding{p.ID: {
			PolicyDigest:  effortauthority.Digest(p),
			ProfileDigest: p.Profiles[profile.ProfileKey],
			Repository:    p.Repository,
			Deadline:      p.Deadline,
		}},
		EnvKeys:        []string{"PATH", "HOME", "VROOLI_AGENT_IDENTITY_TOKEN"},
		NativeBinaries: map[string]string{"/usr/bin/codex": h},
	}
}

// InstallFixtureFiniteNativeForTest installs the admission-only finite-native composition on
// o, mirroring the production invariant (authority ⇒ factory ⇒ terminal) that
// finite_installation.Apply establishes atomically. Pure manifest composition qualifies
// admission only; the dispatcher is untouched. TEST USE ONLY.
func InstallFixtureFiniteNativeForTest(o *Orchestrator, p effortauthority.Policy, profile *domain.AgentProfile) error {
	// Fail closed outside tests. This installs an admission-only factory that
	// qualifies bindings without a real native launcher/terminal; it must never
	// run in a production process, where the hardened startup path
	// (finite_installation.Apply + InstallFiniteNativeIsolation) is the only
	// legitimate installer.
	if !testing.Testing() {
		return fmt.Errorf("InstallFixtureFiniteNativeForTest is test-only and refused outside go test")
	}
	runtime, e := isolation.NewRuntime(fixtureFiniteNativeManifest(p, profile))
	if e != nil {
		return e
	}
	o.finiteNativeFactory = fixtureFiniteNativeAdmission{runtime}
	o.finiteNativeTerminal = fixtureFiniteNativeTerminal{}
	return nil
}
