package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestCreateCommandLoggerRespectsVerboseAndLogLevel(t *testing.T) {
	t.Setenv("VROOLI_LOG_LEVEL", "info")

	var normal bytes.Buffer
	logger := createCommandLogger(false, &normal)
	logger.Debug("debug should be hidden")
	if normal.Len() != 0 {
		t.Fatalf("debug message should be hidden at info level: %q", normal.String())
	}

	var verbose bytes.Buffer
	logger = createCommandLogger(true, &verbose)
	logger.Debug("debug should be visible with verbose")
	if !strings.Contains(verbose.String(), "debug should be visible with verbose") {
		t.Fatalf("verbose logger did not emit debug output: %q", verbose.String())
	}
}

func TestRunEmitsDebugLogsWhenVerbose(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	var captured bytes.Buffer
	newLoggerFn = func(verbose bool, _ io.Writer) *slog.Logger {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		return slog.New(slog.NewTextHandler(&captured, &slog.HandlerOptions{
			Level: level,
		}))
	}
	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }
	t.Setenv("VROOLI_LOG_LEVEL", "info")

	var stdout bytes.Buffer
	code := run([]string{"--verbose", "version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(captured.String(), "Parsed command") {
		t.Fatalf("missing debug log for parsed command: %q", captured.String())
	}
	if !strings.Contains(captured.String(), "Resolved root") {
		t.Fatalf("missing debug log for root resolution: %q", captured.String())
	}
}

func TestPrintErrorWithContextCategorizesRuntimeError(t *testing.T) {
	var output bytes.Buffer
	err := newErrorWithCategory(
		errors.New("pipeline failed"),
		errorCategoryRuntime,
		"Check the command inputs and try again.",
		[]string{"setup", "status"},
	)
	printErrorWithContext(&output, err)

	rendered := output.String()
	if !strings.Contains(rendered, "Runtime error: pipeline failed") {
		t.Fatalf("rendered runtime error = %q", rendered)
	}
	if !strings.Contains(rendered, "Check the command inputs and try again.") {
		t.Fatalf("rendered runtime hint = %q", rendered)
	}
	if !strings.Contains(rendered, "Did you mean one of these?") {
		t.Fatalf("missing suggestion block = %q", rendered)
	}
	if !strings.Contains(rendered, "Run 'vrooli --help' for usage information") {
		t.Fatalf("missing fallback usage line = %q", rendered)
	}
	if strings.Count(rendered, "Did you mean one of these?") != 1 {
		t.Fatalf("expected single suggestion section: %q", rendered)
	}
}

func TestPrintErrorWithContextPreservesPlainErrors(t *testing.T) {
	var output bytes.Buffer
	printErrorWithContext(&output, errors.New("plain failure"))

	got := strings.TrimSpace(output.String())
	if got != "plain failure" {
		t.Fatalf("unexpected plain error output: %q", got)
	}
}

func TestSuggestTopLevelCommandsIsDeterministic(t *testing.T) {
	first := suggestTopLevelCommands("statz")
	second := suggestTopLevelCommands("statz")

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("suggestions not deterministic: %v != %v", first, second)
	}
}

func TestNewUnknownCommandErrorIncludesSuggestionCategory(t *testing.T) {
	var output bytes.Buffer
	printErrorWithContext(&output, newUnknownCommandError("statz"))

	rendered := output.String()
	if !strings.Contains(rendered, "Unknown command: statz") {
		t.Fatalf("unknown command render = %q", rendered)
	}
	if !strings.Contains(rendered, "Did you mean one of these?") {
		t.Fatalf("missing suggestion list = %q", rendered)
	}
	if !strings.Contains(rendered, "status") {
		t.Fatalf("expected status suggestion, got %q", rendered)
	}
}

func TestNewErrorWithCategoryPreservesExitCode(t *testing.T) {
	err := newErrorWithCategory(exitCodeError{code: 23, message: "wrapped"}, errorCategoryRuntime, "", nil)
	if got, want := exitCode(err), 23; got != want {
		t.Fatalf("exitCode = %d, want %d", got, want)
	}
}
