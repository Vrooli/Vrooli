package rootcli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/metrics"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

type metricsTestHarness struct {
	home      string
	stdout    *bytes.Buffer
	stderr    *bytes.Buffer
	helpCalls int
}

func newMetricsHarness(t *testing.T, handlers map[topcli.CommandID]Handler[*metricsTestHarness]) *Runner[*metricsTestHarness] {
	t.Helper()
	h := &metricsTestHarness{
		home:   t.TempDir(),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	reg := NewRegistry(handlers, map[scenariocli.CommandID]Handler[*metricsTestHarness]{})
	return NewRunner(RunnerConfig[*metricsTestHarness]{
		Registry: reg,
		NewLogger: func(GlobalOptions, io.Writer) (*slog.Logger, func()) {
			return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
		},
		NewContext: func(_ GlobalOptions, _ io.Writer, _ io.Writer, _ *slog.Logger) *metricsTestHarness {
			return h
		},
		ShowMainHelp: func(ctx *metricsTestHarness) {
			ctx.helpCalls++
		},
		ShowVersion: func(*metricsTestHarness) error { return nil },
		ResolveRoot: func() (string, error) { return h.home, nil },
		SetRoot:     func(*metricsTestHarness, string) {},
		// No ShouldRebuild / RebuildAndReexec — stale check skipped.
		MetricsRecorder: metrics.New(h.home, func(err error) { t.Logf("metrics IO err: %v", err) }),
		CLIVersion:      "test-cli-1.2.3",
		PlatformVersion: "test-platform-4.5.6",
	})
}

func timingsPath(home string) string {
	return filepath.Join(home, ".vrooli", "metrics", "timings.jsonl")
}

func readEvents(t *testing.T, path string) []metrics.Event {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read timings: %v", err)
	}
	var out []metrics.Event
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		var e metrics.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("unmarshal %q: %v", scanner.Text(), err)
		}
		out = append(out, e)
	}
	return out
}

func TestDispatchRecordsHelp(t *testing.T) {
	runner := newMetricsHarness(t, nil)
	// ParseArgs returns Command="help" when args is empty.
	code := runner.Run(nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	home := runner.config.NewContext(GlobalOptions{}, nil, nil, nil).home
	events := readEvents(t, timingsPath(home))
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Command != "help" {
		t.Errorf("Command = %q, want help", events[0].Command)
	}
	if events[0].ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", events[0].ExitCode)
	}
	if events[0].CLIVersion != "test-cli-1.2.3" {
		t.Errorf("CLIVersion = %q", events[0].CLIVersion)
	}
	if events[0].DurationMs < 0 {
		t.Errorf("DurationMs = %d, want >= 0", events[0].DurationMs)
	}
}

func TestDispatchRecordsHandlerError(t *testing.T) {
	boom := errors.New("boom")
	runner := newMetricsHarness(t, map[topcli.CommandID]Handler[*metricsTestHarness]{
		topcli.CommandDoctor: func(*metricsTestHarness, []string) error { return boom },
	})
	code := runner.Run([]string{"doctor"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	home := runner.config.NewContext(GlobalOptions{}, nil, nil, nil).home
	events := readEvents(t, timingsPath(home))
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.Command != "doctor" {
		t.Errorf("Command = %q", ev.Command)
	}
	if ev.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero")
	}
	if ev.ErrorClass == "" {
		t.Errorf("ErrorClass empty, want non-empty")
	}
}

func TestDispatchNoMetricsFlagSkipsRecording(t *testing.T) {
	runner := newMetricsHarness(t, nil)
	if code := runner.Run([]string{"--no-metrics"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	home := runner.config.NewContext(GlobalOptions{}, nil, nil, nil).home
	if _, err := os.Stat(timingsPath(home)); !os.IsNotExist(err) {
		t.Fatalf("expected no timings file, got err=%v", err)
	}
}

func TestDispatchRedactsFlagValues(t *testing.T) {
	runner := newMetricsHarness(t, map[topcli.CommandID]Handler[*metricsTestHarness]{
		topcli.CommandDoctor: func(*metricsTestHarness, []string) error { return nil },
	})
	if code := runner.Run([]string{"doctor", "--token=SECRET", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	home := runner.config.NewContext(GlobalOptions{}, nil, nil, nil).home
	data, err := os.ReadFile(timingsPath(home))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "SECRET") {
		t.Fatalf("SECRET leaked into log: %s", data)
	}
	if !strings.Contains(string(data), "--token") {
		t.Errorf("expected stripped flag name --token in log: %s", data)
	}
	if !strings.Contains(string(data), "alpha") {
		t.Errorf("expected positional alpha in log: %s", data)
	}
}
