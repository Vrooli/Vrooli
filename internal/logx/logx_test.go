package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-12

func TestParseLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  slog.Level
		err   bool
	}{
		{name: "default", input: "", want: slog.LevelInfo},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "debug", input: "DEBUG", want: slog.LevelDebug},
		{name: "warn", input: " warn ", want: slog.LevelWarn},
		{name: "error", input: "err", want: slog.LevelError},
		{name: "invalid", input: "trace", want: slog.LevelInfo, err: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseLevel(tc.input)
			if (err != nil) != tc.err {
				t.Fatalf("ParseLevel(%q) error = %v, want error=%v", tc.input, err, tc.err)
			}
			if got != tc.want {
				t.Fatalf("ParseLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestLevelFromEnv(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "debug")
	got, err := LevelFromEnv()
	if err != nil {
		t.Fatalf("LevelFromEnv returned error: %v", err)
	}
	if got != slog.LevelDebug {
		t.Fatalf("LevelFromEnv = %v, want debug", got)
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  Format
		err   bool
	}{
		{name: "default", input: "", want: FormatText},
		{name: "text", input: "text", want: FormatText},
		{name: "json", input: " JSON ", want: FormatJSON},
		{name: "invalid", input: "yaml", want: FormatText, err: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseFormat(tc.input)
			if (err != nil) != tc.err {
				t.Fatalf("ParseFormat(%q) error = %v, want error=%v", tc.input, err, tc.err)
			}
			if got != tc.want {
				t.Fatalf("ParseFormat(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatFromEnv(t *testing.T) {
	t.Setenv(LogFormatEnvVar, "json")
	got, err := FormatFromEnv()
	if err != nil {
		t.Fatalf("FormatFromEnv returned error: %v", err)
	}
	if got != FormatJSON {
		t.Fatalf("FormatFromEnv = %v, want json", got)
	}
}

func TestNewIncludesComponentAndSupportsJSON(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli-api",
		Writer:    &buffer,
		JSON:      true,
	})
	if len(diagnostics.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", diagnostics.Warnings)
	}

	logger.Info("hello", AttrPort, 8092)
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record[AttrComponent]; got != "vrooli-api" {
		t.Fatalf("component = %#v, want %q", got, "vrooli-api")
	}
	if got := numericValue(t, record[AttrPort]); got != 8092 {
		t.Fatalf("port = %v, want 8092", got)
	}
}

func TestNewIncludesSubsystem(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli-api",
		Subsystem: "health",
		Writer:    &buffer,
		Format:    FormatJSON,
	})
	if diagnostics.Format != FormatJSON {
		t.Fatalf("format = %v, want json", diagnostics.Format)
	}

	logger.Info("hello")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record[AttrSubsystem]; got != "health" {
		t.Fatalf("subsystem = %#v, want %q", got, "health")
	}
}

func TestNewVerboseOverridesConfiguredInfoLevel(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "info")

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli",
		Writer:    &buffer,
		JSON:      true,
		Verbose:   true,
	})
	if got := diagnostics.Level; got != slog.LevelDebug {
		t.Fatalf("level = %v, want debug", got)
	}

	logger.Debug("visible")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["msg"]; got != "visible" {
		t.Fatalf("msg = %#v, want %q", got, "visible")
	}
}

func TestNewUsesFormatFromEnv(t *testing.T) {
	t.Setenv(LogFormatEnvVar, "json")

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli",
		Writer:    &buffer,
	})
	if diagnostics.Format != FormatJSON {
		t.Fatalf("format = %v, want json", diagnostics.Format)
	}

	logger.Info("env driven")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["msg"]; got != "env driven" {
		t.Fatalf("msg = %#v, want %q", got, "env driven")
	}
}

func TestNewOptionFormatOverridesJSONCompatibilityFlag(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Writer: &buffer,
		Format: FormatText,
		JSON:   true,
	})
	if diagnostics.Format != FormatText {
		t.Fatalf("format = %v, want text", diagnostics.Format)
	}

	logger.Info("text wins")
	if strings.Contains(buffer.String(), `"msg":"text wins"`) {
		t.Fatalf("expected text output when Format is explicit, got %q", buffer.String())
	}
}

func TestNewInvalidFormatReturnsWarning(t *testing.T) {
	t.Setenv(LogFormatEnvVar, "yaml")

	var buffer bytes.Buffer
	_, diagnostics := New(Options{Writer: &buffer})
	if len(diagnostics.Warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", diagnostics.Warnings)
	}
	if diagnostics.Warnings[0].Code != WarningCodeInvalidLogFormat {
		t.Fatalf("warning code = %q, want %q", diagnostics.Warnings[0].Code, WarningCodeInvalidLogFormat)
	}
	if diagnostics.Format != FormatText {
		t.Fatalf("format = %v, want text", diagnostics.Format)
	}
}

func TestNewDefaultsWriterToStderr(t *testing.T) {
	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = reader.Close()
		_ = writer.Close()
	})

	logger, _ := New(Options{Component: "vrooli"})
	logger.Info("stderr default")
	_ = writer.Close()

	data, err := ioReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(data), "stderr default") {
		t.Fatalf("expected stderr output, got %q", string(data))
	}
}

func TestNewPropagatesReplaceAttr(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, _ := New(Options{
		Writer: &buffer,
		JSON:   true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				return slog.String("message_text", a.Value.String())
			}
			return a
		},
	})

	logger.Info("rewritten")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["message_text"]; got != "rewritten" {
		t.Fatalf("message_text = %#v, want %q", got, "rewritten")
	}
	if _, exists := record[slog.MessageKey]; exists {
		t.Fatalf("expected original message key to be replaced: %#v", record)
	}
}

func TestInstallReturnsDiagnosticsWithoutSelfLogging(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "trace")

	var buffer bytes.Buffer
	logger, diagnostics, restore := Install(Options{
		Component:      "vrooli",
		Writer:         &buffer,
		JSON:           true,
		SetDefault:     true,
		RedirectStdlib: true,
	})
	defer restore()

	if logger == nil {
		t.Fatal("expected logger")
	}
	if len(diagnostics.Warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", diagnostics.Warnings)
	}
	warning := diagnostics.Warnings[0]
	if warning.Code != WarningCodeInvalidLogLevel {
		t.Fatalf("warning code = %q, want %q", warning.Code, WarningCodeInvalidLogLevel)
	}
	if warning.EnvVar != LogLevelEnvVar {
		t.Fatalf("warning env var = %q, want %q", warning.EnvVar, LogLevelEnvVar)
	}
	if warning.Value != "trace" {
		t.Fatalf("warning value = %q, want %q", warning.Value, "trace")
	}
	if buffer.Len() != 0 {
		t.Fatalf("expected Install not to emit diagnostics directly, got %q", buffer.String())
	}
}

func TestInstallAndReportEmitsWarnings(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "trace")
	t.Setenv(LogFormatEnvVar, "yaml")

	var buffer bytes.Buffer
	_, diagnostics, restore := InstallAndReport(Options{
		Writer:         &buffer,
		Format:         "",
		SetDefault:     true,
		RedirectStdlib: true,
	})
	defer restore()

	if len(diagnostics.Warnings) != 2 {
		t.Fatalf("warnings = %v, want 2 warnings", diagnostics.Warnings)
	}
	output := buffer.String()
	if !strings.Contains(output, WarningCodeInvalidLogLevel) {
		t.Fatalf("expected invalid level warning in output, got %q", output)
	}
	if !strings.Contains(output, WarningCodeInvalidLogFormat) {
		t.Fatalf("expected invalid format warning in output, got %q", output)
	}
}

func TestInstallRestoreReinstatesPriorLoggerState(t *testing.T) {
	originalDefault := slog.Default()
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	var sink bytes.Buffer
	log.SetOutput(&sink)
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("before:")
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	var buffer bytes.Buffer
	logger, diagnostics, restore := Install(Options{
		Writer:         &buffer,
		SetDefault:     true,
		RedirectStdlib: true,
	})
	if slog.Default() != logger {
		t.Fatal("expected Install to set the default slog logger")
	}
	if log.Flags() != 0 {
		t.Fatalf("expected redirected flags to be cleared, got %d", log.Flags())
	}

	restore()

	if slog.Default() != originalDefault {
		t.Fatal("expected original slog default to be restored")
	}
	if log.Flags() != log.Lshortfile {
		t.Fatalf("flags = %d, want %d", log.Flags(), log.Lshortfile)
	}
	if log.Prefix() != "before:" {
		t.Fatalf("prefix = %q, want %q", log.Prefix(), "before:")
	}
	if diagnostics.Level != slog.LevelInfo {
		t.Fatalf("level = %v, want info", diagnostics.Level)
	}

	log.Print("restored output")
	if !strings.Contains(sink.String(), "restored output") {
		t.Fatalf("expected restored stdlib output to reach prior sink, got %q", sink.String())
	}
}

func TestRedirectStandardLibrary(t *testing.T) {
	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "test",
		Writer:    &buffer,
		JSON:      true,
	})

	restore := RedirectStandardLibrary(logger, diagnostics.Level)
	defer restore()

	log.Printf("redirected log")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["msg"]; got != "redirected log" {
		t.Fatalf("msg = %#v, want %q", got, "redirected log")
	}
}

func TestRedirectStandardLibraryNilLoggerNoOp(t *testing.T) {
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	restore := RedirectStandardLibrary(nil, slog.LevelInfo)
	restore()

	if log.Writer() != originalWriter {
		t.Fatal("expected writer to remain unchanged")
	}
	if log.Flags() != originalFlags {
		t.Fatalf("flags = %d, want %d", log.Flags(), originalFlags)
	}
	if log.Prefix() != originalPrefix {
		t.Fatalf("prefix = %q, want %q", log.Prefix(), originalPrefix)
	}
}

func TestRedirectStandardLibraryNestedRestoreOrder(t *testing.T) {
	var first bytes.Buffer
	loggerA, diagnosticsA := New(Options{Writer: &first, JSON: true})
	restoreA := RedirectStandardLibrary(loggerA, diagnosticsA.Level)

	var second bytes.Buffer
	loggerB, diagnosticsB := New(Options{Writer: &second, JSON: true})
	restoreB := RedirectStandardLibrary(loggerB, diagnosticsB.Level)

	log.Printf("second")
	restoreB()
	log.Printf("first")
	restoreA()

	secondRecord := decodeSingleJSONRecord(t, second.Bytes())
	if got := secondRecord["msg"]; got != "second" {
		t.Fatalf("second logger msg = %#v, want %q", got, "second")
	}
	firstRecord := decodeSingleJSONRecord(t, first.Bytes())
	if got := firstRecord["msg"]; got != "first" {
		t.Fatalf("first logger msg = %#v, want %q", got, "first")
	}
}

func TestWithSubsystemAddsSubsystemAttribute(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, _ := New(Options{Component: "vrooli", Writer: &buffer, JSON: true})
	WithSubsystem(logger, "lifecycle").Info("hello")

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record[AttrSubsystem]; got != "lifecycle" {
		t.Fatalf("subsystem = %#v, want %q", got, "lifecycle")
	}
}

func TestWithSubsystemFallsBackToDefaultLogger(t *testing.T) {
	originalDefault := slog.Default()
	var buffer bytes.Buffer
	fallback := slog.New(slog.NewJSONHandler(&buffer, nil))
	slog.SetDefault(fallback)
	t.Cleanup(func() { slog.SetDefault(originalDefault) })

	WithSubsystem(nil, "").Info("hello")
	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["msg"]; got != "hello" {
		t.Fatalf("msg = %#v, want %q", got, "hello")
	}
}

func TestErrorArgsNilOmitsErrorField(t *testing.T) {
	t.Parallel()

	if got := ErrorArgs(nil); got != nil {
		t.Fatalf("ErrorArgs(nil) = %#v, want nil", got)
	}
}

func TestErrorAttrStructuredFields(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, _ := New(Options{Writer: &buffer, JSON: true})
	logger.Error("failed", ErrorArgs(os.ErrNotExist)...)

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField, ok := record[AttrError].(map[string]any)
	if !ok {
		t.Fatalf("error field = %#v, want object", record[AttrError])
	}
	if got := errField["message"]; got != os.ErrNotExist.Error() {
		t.Fatalf("error.message = %#v, want %q", got, os.ErrNotExist.Error())
	}
	if got := errField["category"]; got != "not_found" {
		t.Fatalf("error.category = %#v, want %q", got, "not_found")
	}
	if got := errField["type"]; got == "" {
		t.Fatalf("expected error.type to be populated: %#v", errField)
	}
}

func TestErrorAttrCapturesPathContext(t *testing.T) {
	t.Parallel()

	pathErr := &fs.PathError{Op: "open", Path: "/tmp/missing", Err: fs.ErrNotExist}
	attr := ErrorAttr(pathErr)

	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	logger.Error("failed", attr)

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField := record[AttrError].(map[string]any)
	if got := errField["op"]; got != "open" {
		t.Fatalf("error.op = %#v, want %q", got, "open")
	}
	if got := errField["path"]; got != "/tmp/missing" {
		t.Fatalf("error.path = %#v, want %q", got, "/tmp/missing")
	}
}

func TestErrorAttrCapturesTimeoutAndRetryable(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger, _ := New(Options{Writer: &buffer, JSON: true})
	logger.Error("failed", ErrorArgs(timeoutTemporaryError{err: context.DeadlineExceeded})...)

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField := record[AttrError].(map[string]any)
	if got := errField["category"]; got != "timeout" {
		t.Fatalf("error.category = %#v, want %q", got, "timeout")
	}
	if got := errField["timeout"]; got != true {
		t.Fatalf("error.timeout = %#v, want true", got)
	}
	if got := errField["retryable"]; got != true {
		t.Fatalf("error.retryable = %#v, want true", got)
	}
}

func TestErrorAttrClassifiesCanceled(t *testing.T) {
	t.Parallel()

	attr := ErrorAttr(context.Canceled)
	if got := attr.Key; got != AttrError {
		t.Fatalf("key = %q, want %q", got, AttrError)
	}

	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	logger.Error("failed", attr)

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField := record[AttrError].(map[string]any)
	if got := errField["category"]; got != "canceled" {
		t.Fatalf("error.category = %#v, want %q", got, "canceled")
	}
}

func TestErrorAttrCapturesExecExitMetadata(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("bash", "-lc", "printf 'boom' >&2; exit 7")
	_, err := cmd.Output()
	if err == nil {
		t.Fatal("expected command failure")
	}

	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	logger.Error("failed", ErrorAttr(err))

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField := record[AttrError].(map[string]any)
	if got := errField["category"]; got != string(ErrorCategoryExternal) {
		t.Fatalf("error.category = %#v, want %q", got, ErrorCategoryExternal)
	}
	if got := numericValue(t, errField["exit_code"]); got != 7 {
		t.Fatalf("error.exit_code = %v, want 7", got)
	}
	if got := errField["stderr"]; got != "boom" {
		t.Fatalf("error.stderr = %#v, want %q", got, "boom")
	}
}

func TestErrorAttrCapturesURLContext(t *testing.T) {
	t.Parallel()

	target := &url.Error{Op: "Get", URL: "http://localhost:1", Err: context.DeadlineExceeded}
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	logger.Error("failed", ErrorAttr(target))

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	errField := record[AttrError].(map[string]any)
	if got := errField["category"]; got != string(ErrorCategoryTimeout) {
		t.Fatalf("error.category = %#v, want %q", got, ErrorCategoryTimeout)
	}
	if got := errField["url"]; got != "http://localhost:1" {
		t.Fatalf("error.url = %#v, want %q", got, "http://localhost:1")
	}
}

func TestErrorHelperIncludesSharedErrorContract(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	Error(logger, "failed", os.ErrPermission, "step", "bootstrap")

	record := decodeSingleJSONRecord(t, buffer.Bytes())
	if got := record["step"]; got != "bootstrap" {
		t.Fatalf("step = %#v, want %q", got, "bootstrap")
	}
	errField := record[AttrError].(map[string]any)
	if got := errField["category"]; got != string(ErrorCategoryPermission) {
		t.Fatalf("error.category = %#v, want %q", got, ErrorCategoryPermission)
	}
}

func decodeSingleJSONRecord(t *testing.T, data []byte) map[string]any {
	t.Helper()

	line := strings.TrimSpace(string(data))
	if line == "" {
		t.Fatal("expected log output")
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("decode log record: %v\nrecord: %s", err, line)
	}
	return record
}

func numericValue(t *testing.T, value any) int {
	t.Helper()

	number, ok := value.(float64)
	if !ok {
		t.Fatalf("value = %#v, want numeric", value)
	}
	return int(number)
}

func ioReadAll(r *os.File) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := buffer.ReadFrom(r)
	return buffer.Bytes(), err
}

type timeoutTemporaryError struct {
	err error
}

func (e timeoutTemporaryError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e timeoutTemporaryError) Unwrap() error { return e.err }
func (e timeoutTemporaryError) Timeout() bool { return true }
func (e timeoutTemporaryError) Temporary() bool {
	return true
}

var _ error = timeoutTemporaryError{err: errors.New("boom")}
