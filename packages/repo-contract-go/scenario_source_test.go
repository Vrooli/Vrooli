package repocontract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestResolveScenarioSourceRoot(t *testing.T) {
	first := fixtureRoot(t)
	second := fixtureRoot(t)
	for _, root := range []string{first, second} {
		repocontracttest.WriteFile(t, filepath.Join(root, "scenarios", "agent-manager", "api", "fixture"), "source")
	}
	retained := t.TempDir()
	for _, tc := range []struct {
		name     string
		cwd      string
		override string
		env      map[string]string
		want     string
		wantErr  bool
	}{
		{name: "ordinary cwd", cwd: filepath.Join(first, "scenarios", "agent-manager", "api"), want: first},
		{name: "retained lifecycle paths", cwd: retained, env: map[string]string{"SCENARIO_NAME": "agent-manager", "VROOLI_SCENARIO": "agent-manager", "SCENARIO_PATH": filepath.Join(first, "scenarios", "agent-manager"), "VROOLI_SCENARIO_DIR": filepath.Join(first, "scenarios", "agent-manager")}, want: first},
		{name: "second lifecycle alias", cwd: retained, env: map[string]string{"VROOLI_SCENARIO": "agent-manager", "VROOLI_SCENARIO_DIR": filepath.Join(first, "scenarios", "agent-manager")}, want: first},
		{name: "source override precedence", cwd: retained, env: map[string]string{"VROOLI_SOURCE_ROOT": filepath.Join(first, "scenarios"), "VROOLI_ROOT": second}, want: first},
		{name: "repo override", cwd: retained, env: map[string]string{"VROOLI_ROOT": first}, want: first},
		{name: "explicit override precedence", cwd: retained, override: second, env: map[string]string{"VROOLI_SOURCE_ROOT": first, "SCENARIO_PATH": "/missing"}, want: second},
		{name: "invalid explicit root", cwd: filepath.Join(first, "scenarios"), override: filepath.Join(second, "missing"), wantErr: true},
		{name: "explicit descendant is not a root", cwd: retained, override: filepath.Join(first, "scenarios"), wantErr: true},
		{name: "invalid source override cannot fall through", cwd: first, env: map[string]string{"VROOLI_SOURCE_ROOT": filepath.Join(first, "missing"), "VROOLI_ROOT": second}, wantErr: true},
		{name: "invalid repo override cannot fall through", cwd: first, env: map[string]string{"VROOLI_ROOT": filepath.Join(first, "missing")}, wantErr: true},
		{name: "wrong identity", cwd: first, env: map[string]string{"SCENARIO_NAME": "other", "SCENARIO_PATH": filepath.Join(first, "scenarios", "agent-manager")}, wantErr: true},
		{name: "conflicting identity aliases", cwd: first, env: map[string]string{"SCENARIO_NAME": "agent-manager", "VROOLI_SCENARIO": "other", "SCENARIO_PATH": filepath.Join(first, "scenarios", "agent-manager")}, wantErr: true},
		{name: "missing identity", cwd: first, env: map[string]string{"SCENARIO_PATH": filepath.Join(first, "scenarios", "agent-manager")}, wantErr: true},
		{name: "wrong scenario path", cwd: first, env: map[string]string{"SCENARIO_NAME": "agent-manager", "SCENARIO_PATH": filepath.Join(first, "scenarios")}, wantErr: true},
		{name: "conflicting path aliases", cwd: first, env: map[string]string{"SCENARIO_NAME": "agent-manager", "SCENARIO_PATH": filepath.Join(first, "scenarios", "agent-manager"), "VROOLI_SCENARIO_DIR": filepath.Join(second, "scenarios", "agent-manager")}, wantErr: true},
		{name: "missing owner source cannot fall through", cwd: first, env: map[string]string{"SCENARIO_NAME": "agent-manager", "SCENARIO_PATH": filepath.Join(first, "missing")}, wantErr: true},
		{name: "relative owner path rejected", cwd: first, env: map[string]string{"SCENARIO_NAME": "agent-manager", "SCENARIO_PATH": "scenarios/agent-manager"}, wantErr: true},
		{name: "retained cwd without owner is unavailable", cwd: retained, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT", "SCENARIO_NAME", "VROOLI_SCENARIO", "SCENARIO_PATH", "VROOLI_SCENARIO_DIR"} {
				t.Setenv(key, tc.env[key])
			}
			oldWD, oldExec := getwdPath, executablePath
			getwdPath = func() (string, error) { return tc.cwd, nil }
			executablePath = func() (string, error) { return filepath.Join(retained, "api"), nil }
			t.Cleanup(func() { getwdPath, executablePath = oldWD, oldExec })
			got, err := ResolveScenarioSourceRoot("agent-manager", tc.override)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("root=%q error=%v; want=%q error=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestResolveScenarioSourceRootUsesContractLayout(t *testing.T) {
	root := fixtureRoot(t, repocontracttest.WithScenarioDir("apps"))
	source := filepath.Join(root, "apps", "agent-manager")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_SOURCE_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("SCENARIO_NAME", "agent-manager")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("SCENARIO_PATH", source)
	t.Setenv("VROOLI_SCENARIO_DIR", "")
	got, err := ResolveScenarioSourceRoot("agent-manager", "")
	if err != nil || got != root {
		t.Fatalf("root=%q error=%v", got, err)
	}
}
