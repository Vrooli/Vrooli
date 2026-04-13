package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSplitRunStatusUsesNativeProjectController(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("status should not route through CLI bash shim: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "status", "--scenarios"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"scenarios_total": 1`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name": "alpha"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"maintenance"`) {
		t.Fatalf("stdout missing maintenance snapshot: %q", stdout.String())
	}
}

func TestSplitRunStatusAcceptsTrailingGlobalJSONFlag(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("status should not route through CLI bash shim: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"status", "--scenarios", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"name": "alpha"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunDoctorAcceptsTrailingGlobalJSONFlag(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("doctor should not route through CLI bash shim: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"doctor", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"checks"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"listener_inspection"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunStopAcceptsTrailingGlobalJSONFlag(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("stop should not route through CLI bash shim: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"stop", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"success": true`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunCleanupLocksUsesNativeMaintenance(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "cleanup", "locks"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock removal, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), `"success": true`) || !strings.Contains(stdout.String(), `"21234"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunLocksCommandListsNativeState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"locks", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"stale": true`) || !strings.Contains(stdout.String(), `"port": 21234`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunLocksCommandHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"locks"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Port") || !strings.Contains(output, "21234") || !strings.Contains(output, "stale") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestSplitRunDiagnosePortReturnsJSON(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"--json", "diagnose-port", "21234", "alpha"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"port": 21234`) || !strings.Contains(stdout.String(), `"scenario": "alpha"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"listener_inspection"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSplitRunDiagnosePortHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"diagnose-port", "21234", "alpha"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Port 21234") || !strings.Contains(output, "Scenario: alpha") || !strings.Contains(output, "Listener inspection:") || !strings.Contains(output, "Recommended actions:") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestSplitRunCleanupCommandRoutesTargets(t *testing.T) {
	t.Run("orphans", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)

		t.Setenv("HOME", home)
		err := runCleanupCommand(root, parsedArgs{
			args: []string{"orphans", "help"},
			globals: globalOptions{
				json:    true,
				verbose: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
	})

	t.Run("locks", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
		lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
		writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

		t.Setenv("HOME", home)
		err := runCleanupCommand(root, parsedArgs{
			args: []string{"locks"},
			globals: globalOptions{
				noColor: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Fatalf("expected stale lock removal, stat err=%v", err)
		}
	})
}

func TestSplitRunCleanupCommandHelpAndUnknownTarget(t *testing.T) {
	var stdout bytes.Buffer
	if err := runCleanupCommand("/repo", parsedArgs{}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runCleanupCommand help: %v", err)
	}
	if !strings.Contains(stdout.String(), "vrooli cleanup") {
		t.Fatalf("missing cleanup help output: %s", stdout.String())
	}

	err := runCleanupCommand("/repo", parsedArgs{args: []string{"bogus"}}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode(err) != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode(err))
	}
	if !strings.Contains(err.Error(), "unknown cleanup target: bogus") {
		t.Fatalf("err = %v", err)
	}
}
