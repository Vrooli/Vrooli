package lifecycle

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestRunnerStartStopRestart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	var logs bytes.Buffer
	originalDefault := slog.Default()
	logger, _ := logx.New(logx.Options{Component: "vrooli", Writer: &logs, JSON: true})
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(originalDefault) })
	runner, err := NewRunner(root, home, io.Discard, io.Discard, logger)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	start, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if start.Health != "healthy" {
		t.Fatalf("health = %q, want healthy", start.Health)
	}
	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	live := process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after start, got %#v", live)
	}
	firstPID := live[0].PID

	setupNeeded, _, err := runner.SetupNeeded(start.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded after start: %v", err)
	}
	if setupNeeded {
		t.Fatalf("expected setup to be current after start")
	}

	restarted, err := runner.Restart("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if restarted.Health != "healthy" {
		t.Fatalf("restart health = %q, want healthy", restarted.Health)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after restart: %v", err)
	}
	live = process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after restart, got %#v", live)
	}
	if live[0].PID == firstPID {
		t.Fatalf("expected new PID after restart, still %d", firstPID)
	}

	if err := runner.Stop("alpha", StopOptions{}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after stop: %v", err)
	}
	if len(process.LiveRecords(records)) != 0 {
		t.Fatalf("expected no live records after stop: %#v", records)
	}
	if !strings.Contains(logs.String(), `"msg":"Scenario start completed"`) {
		t.Fatalf("expected structured start log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), `"msg":"Scenario stop completed"`) {
		t.Fatalf("expected structured stop log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), `"scenario":"alpha"`) {
		t.Fatalf("expected scenario attribute in logs, got %q", logs.String())
	}
}

func TestSetupNeededDetectsUpdatedSources(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	sourcePath := filepath.Join(root, "scenarios", "alpha", "api", "handler.go")
	testkitgo.WriteFile(t, sourcePath, "package main\n")
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(sourcePath, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", sourcePath, err)
	}

	setupNeeded, reasons, err := runner.SetupNeeded(result.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded: %v", err)
	}
	if !setupNeeded {
		t.Fatalf("expected setup to be needed after touching source")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected setup reasons to be populated")
	}

	_ = runner.Stop("alpha", StopOptions{})
}

func TestRunnerStartStartsRequiredDependencies(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()

	writeLifecycleFixture(t, root, "beta")
	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	t.Cleanup(func() {
		_ = runner.Stop("alpha", StopOptions{})
		_ = runner.Stop("beta", StopOptions{})
	})

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(result.FailedDependencies) != 0 {
		t.Fatalf("unexpected failed dependencies: %#v", result.FailedDependencies)
	}

	for _, name := range []string{"alpha", "beta"} {
		records, err := process.ReadScenarioRecords(home, name)
		if err != nil {
			t.Fatalf("ReadScenarioRecords(%s): %v", name, err)
		}
		if len(process.LiveRecords(records)) != 1 {
			t.Fatalf("expected %s to be running, records=%#v", name, records)
		}
	}

	if _, err := os.Stat(process.ScenarioDegradedPath(home, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("unexpected degraded state after successful dependency start: %v", err)
	}
}

func TestListeningPIDsDetectsLiveListener(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof is not installed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	found := false
	for attempt := 0; attempt < 10; attempt++ {
		pids, err := listeningPIDs(port)
		if err != nil {
			t.Fatalf("listeningPIDs: %v", err)
		}
		for _, pid := range pids {
			if pid == os.Getpid() {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Fatalf("expected current pid %d to own listener on port %d", os.Getpid(), port)
	}
}
