package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

func TestCreateCommandLoggerRespectsVerboseAndLogLevel(t *testing.T) {
	t.Setenv("VROOLI_LOG_LEVEL", "info")

	var normal bytes.Buffer
	logger, restore := createCommandLogger(globalOptions{}, &normal)
	defer restore()
	logger.Debug("debug should be hidden")
	if normal.Len() != 0 {
		t.Fatalf("debug message should be hidden at info level: %q", normal.String())
	}

	var verbose bytes.Buffer
	logger, restore = createCommandLogger(globalOptions{verbose: true}, &verbose)
	defer restore()
	logger.Debug("debug should be visible with verbose")
	if !strings.Contains(verbose.String(), "debug should be visible with verbose") {
		t.Fatalf("verbose logger did not emit debug output: %q", verbose.String())
	}
}

func TestRunEmitsDebugLogsWhenVerbose(t *testing.T) {
	var captured bytes.Buffer
	app := configuredApp()
	app.newLogger = func(globals globalOptions, _ io.Writer) (*slog.Logger, func()) {
		level := slog.LevelInfo
		if globals.verbose {
			level = slog.LevelDebug
		}
		return slog.New(slog.NewTextHandler(&captured, &slog.HandlerOptions{
			Level: level,
		})), func() {}
	}
	app.resolveSourceRoot = func() (string, error) { return "/repo", nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	t.Setenv("VROOLI_LOG_LEVEL", "info")

	var stdout bytes.Buffer
	code := app.Run([]string{"--verbose", "version"}, &stdout, &bytes.Buffer{})
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

func TestCreateCommandLoggerUsesJSONWhenCommandOutputIsJSON(t *testing.T) {
	var captured bytes.Buffer
	logger, restore := createCommandLogger(globalOptions{json: true}, &captured)
	defer restore()

	logger.Info("machine readable")
	if !strings.Contains(captured.String(), `"msg":"machine readable"`) {
		t.Fatalf("expected json log output, got %q", captured.String())
	}
}

func TestCreateCommandLoggerUsesEnvFormatWhenCommandOutputIsHuman(t *testing.T) {
	t.Setenv("VROOLI_LOG_FORMAT", "json")

	var captured bytes.Buffer
	logger, restore := createCommandLogger(globalOptions{}, &captured)
	defer restore()

	logger.Info("env format")
	if !strings.Contains(captured.String(), `"msg":"env format"`) {
		t.Fatalf("expected env-driven json log output, got %q", captured.String())
	}
}

func TestCreateCommandLoggerEmitsConfigurationWarningsOnce(t *testing.T) {
	t.Setenv("VROOLI_LOG_LEVEL", "trace")
	t.Setenv("VROOLI_LOG_FORMAT", "yaml")

	var captured bytes.Buffer
	_, restore := createCommandLogger(globalOptions{}, &captured)
	defer restore()

	got := captured.String()
	if strings.Count(got, "invalid_log_level") != 1 {
		t.Fatalf("expected one invalid_log_level warning, got %q", got)
	}
	if strings.Count(got, "invalid_log_format") != 1 {
		t.Fatalf("expected one invalid_log_format warning, got %q", got)
	}
}

func TestCommandContextOutputFormatRespectsForceJSON(t *testing.T) {
	ctx := &commandContext{Globals: globalOptions{}}

	format, err := ctx.outputFormat(true)
	if err != nil {
		t.Fatalf("outputFormat: %v", err)
	}
	if format != cliout.FormatJSON {
		t.Fatalf("format = %q, want %q", format, cliout.FormatJSON)
	}
}

func TestExecutionContextForFormatRedirectsJSONRunnerOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx := &commandContext{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	jsonCtx := ctx.executionContextForFormat(cliout.FormatJSON)
	if jsonCtx.Stdout != ctx.Stderr {
		t.Fatalf("json stdout was not redirected to stderr")
	}

	humanCtx := ctx.executionContextForFormat(cliout.FormatHuman)
	if humanCtx.Stdout != ctx.Stdout {
		t.Fatalf("human stdout changed unexpectedly")
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
