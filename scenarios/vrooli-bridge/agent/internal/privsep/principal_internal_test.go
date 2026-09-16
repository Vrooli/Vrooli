package privsep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
)

func TestOwnerEnvReplacesIdentityAndSearchPath(t *testing.T) {
	env := ownerEnv("/Users/owner", "owner", []string{"HOME=/var/root", "USER=root", "PATH=/usr/bin", "LANG=en_US.UTF-8"})
	joined := strings.Join(env, "\n")
	require.Contains(t, joined, "HOME=/Users/owner")
	require.Contains(t, joined, "USER=owner")
	require.Contains(t, joined, "LOGNAME=owner")
	require.Contains(t, joined, "LANG=en_US.UTF-8")
	require.NotContains(t, joined, "/var/root")
	require.Contains(t, joined, "PATH=/Users/owner/.vrooli/bin:/Users/owner/.local/bin:")
}

type scriptedStep struct {
	outputs map[string]string
	ran     []string
}

func (s *scriptedStep) Run(_ context.Context, argv []string, _ string, onLog func(string)) (int, error) {
	line := strings.Join(argv, " ")
	s.ran = append(s.ran, line)
	for prefix, out := range s.outputs {
		if strings.HasPrefix(line, prefix) {
			onLog(out)
		}
	}
	return 0, nil
}

// After a provision, only the running scenarios whose build is stale against
// the new checkout are restarted.
func TestRestartStaleScenariosRestartsOnlyStaleRunningScenarios(t *testing.T) {
	step := &scriptedStep{outputs: map[string]string{
		"vrooli scenario list --json": `{"success":true,"scenarios":[` +
			`{"name":"web-console","status":"running"},{"name":"code-facts","status":"running"},{"name":"idle-app","status":"available"}]}`,
		"vrooli scenario freshness web-console --json": `{"scenario":"web-console","stale":true}`,
		"vrooli scenario freshness code-facts --json":  `note: checking` + "\n" + `{"scenario":"code-facts","stale":false}`,
	}}
	h := NewHelper("vrooli", "/work", nil, WithStepRunner(step), WithStaleScenarioRestart())
	var events []string
	h.restartStaleScenarios(context.Background(), func(ev *provisionv1.ProvisionEvent) error {
		events = append(events, ev.GetStatus())
		return nil
	})
	require.Contains(t, step.ran, "vrooli scenario restart web-console")
	require.NotContains(t, step.ran, "vrooli scenario restart code-facts")
	require.NotContains(t, strings.Join(step.ran, "\n"), "idle-app")
	require.Contains(t, events, "restarting stale scenario web-console")
}

// A provision replaces the checkout out of band, so the marker recording the
// last working-tree ship is dropped: the next ship must send everything again.
func TestProvisionForgetsTheShipDigestMarker(t *testing.T) {
	home := t.TempDir()
	workDir := filepath.Join(home, "vrooli")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	marker := filepath.Join(home, ".vrooli.bridge-ship-digest")
	require.NoError(t, os.WriteFile(marker, []byte("digest"), 0o644))

	h := NewHelper("vrooli", workDir, nil, WithStepRunner(&scriptedStep{}))
	h.forgetShipDigest()

	_, err := os.Stat(marker)
	require.True(t, os.IsNotExist(err), "the ship marker must be gone after a provision")
}
