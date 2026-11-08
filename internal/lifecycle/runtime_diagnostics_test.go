package lifecycle

import (
	"testing"

	"github.com/vrooli/api-core/runtimediagnostics"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestRuntimeDiagnosticsUsesOnlyControlPlaneDirectoryAndLaunchIdentity(t *testing.T) {
	session := runtimeRegistrySession{enabled: true, diagnosticsDir: "/operator/state/runtime-diagnostics", instance: scenarioruntime.Instance{InstanceID: "inst-current", Generation: 7}}
	env := map[string]string{runtimediagnostics.DirectoryEnv: "/untrusted"}
	session.injectEnv(env)
	if env[runtimediagnostics.DirectoryEnv] != session.diagnosticsDir || env[runtimeRegistryInstanceEnv] != "inst-current" || env[runtimeRegistryGenerationEnv] != "7" {
		t.Fatalf("injected environment=%v", env)
	}
	disabledRuntimeRegistrySession().injectEnv(nil)
}
