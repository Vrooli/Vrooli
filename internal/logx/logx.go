package logx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// LogLevelEnvVar controls the minimum structured log level for project-level
// Go binaries. Accepted values are "", "info", "debug", "warn", "warning",
// "error", and "err".
const LogLevelEnvVar = "VROOLI_LOG_LEVEL"

// LogFormatEnvVar controls the structured log encoder for project-level Go
// binaries. Accepted values are "", "text", and "json".
const LogFormatEnvVar = "VROOLI_LOG_FORMAT"

// Format identifies the encoded representation used for log output.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

const (
	// AttrAction identifies a lifecycle or command action in structured logs.
	AttrAction = "action"
	// AttrCode identifies a machine-readable warning or error code.
	AttrCode = "code"
	// AttrComponent identifies the top-level binary or service producing logs.
	AttrComponent = "component"
	// AttrEnvVar identifies an environment variable involved in log decisions.
	AttrEnvVar = "env_var"
	// AttrError stores structured error details as a nested slog group.
	AttrError = "error"
	// AttrOperation identifies the internal operation currently executing.
	AttrOperation = "operation"
	// AttrPhase identifies a scenario lifecycle phase.
	AttrPhase = "phase"
	// AttrPort identifies a network port number.
	AttrPort = "port"
	// AttrScenario identifies a scenario slug or name.
	AttrScenario = "scenario"
	// AttrStatus identifies a high-level runtime or health status.
	AttrStatus = "status"
	// AttrStep identifies a lifecycle step inside a phase.
	AttrStep = "step"
	// AttrSubsystem scopes logs to an internal package or subsystem.
	AttrSubsystem = "subsystem"
	// AttrValue stores the raw user-provided value for structured diagnostics.
	AttrValue = "value"
)

// Options configures the logger shape used across project-level Go commands.
type Options struct {
	// Component is attached to every record when non-empty.
	Component string
	// Subsystem is attached to every record when non-empty.
	Subsystem string
	// Writer receives log output. Nil defaults to os.Stderr.
	Writer io.Writer
	// Format selects the log encoding. When empty, the package resolves the
	// encoder from LogFormatEnvVar and finally falls back to text.
	Format Format
	// JSON selects slog's JSON handler instead of the text handler.
	// Deprecated: prefer Format.
	JSON bool
	// Verbose forces debug logging when the configured level is less verbose.
	Verbose bool
	// SetDefault installs the created logger as slog.Default.
	SetDefault bool
	// RedirectStdlib routes package log output through the structured logger.
	RedirectStdlib bool
	// ReplaceAttr customizes record attributes before they are encoded.
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

const (
	WarningCodeInvalidLogLevel  = "invalid_log_level"
	WarningCodeInvalidLogFormat = "invalid_log_format"
)

// ErrorCategory identifies the high-level operational class of a failure.
type ErrorCategory string

const (
	ErrorCategoryCanceled     ErrorCategory = "canceled"
	ErrorCategoryTimeout      ErrorCategory = "timeout"
	ErrorCategoryNotFound     ErrorCategory = "not_found"
	ErrorCategoryPermission   ErrorCategory = "permission_denied"
	ErrorCategoryAlreadyExist ErrorCategory = "already_exists"
	ErrorCategoryInvalid      ErrorCategory = "invalid"
	ErrorCategoryExternal     ErrorCategory = "external"
	ErrorCategoryNetwork      ErrorCategory = "network"
	ErrorCategoryRuntime      ErrorCategory = "runtime"
)

// Warning describes a non-fatal logging configuration issue.
type Warning struct {
	// Code is a stable machine-readable identifier for the warning.
	Code string
	// Message is a concise human-readable summary.
	Message string
	// EnvVar identifies the environment variable involved, when applicable.
	EnvVar string
	// Value stores the raw user-provided value that triggered the warning.
	Value string
}

// Diagnostics captures decisions made while configuring a logger.
type Diagnostics struct {
	// Level is the effective minimum level installed on the handler.
	Level slog.Level
	// Format is the effective log encoding installed on the handler.
	Format Format
	// Warnings contains non-fatal configuration issues encountered while
	// building the logger.
	Warnings []Warning
}

// ParseLevel parses a user-provided log level string into a slog level.
func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "err":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", strings.TrimSpace(value))
	}
}

// LevelFromEnv returns the slog level controlled by VROOLI_LOG_LEVEL.
func LevelFromEnv() (slog.Level, error) {
	return ParseLevel(os.Getenv(LogLevelEnvVar))
}

// ParseFormat parses a user-provided log output format.
func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(FormatText):
		return FormatText, nil
	case string(FormatJSON):
		return FormatJSON, nil
	default:
		return FormatText, fmt.Errorf("unsupported log format %q", strings.TrimSpace(value))
	}
}

// FormatFromEnv returns the log format controlled by VROOLI_LOG_FORMAT.
func FormatFromEnv() (Format, error) {
	return ParseFormat(os.Getenv(LogFormatEnvVar))
}

// New returns a logger configured for project-level commands together with the
// effective level and any configuration warnings.
func New(opts Options) (*slog.Logger, Diagnostics) {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	level, warnings := resolveLevel(opts.Verbose)
	format, formatWarnings := resolveFormat(opts)
	warnings = append(warnings, formatWarnings...)
	handlerOptions := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: opts.ReplaceAttr,
	}

	var handler slog.Handler
	if format == FormatJSON {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	} else {
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	logger := slog.New(handler)
	if strings.TrimSpace(opts.Component) != "" {
		logger = logger.With(AttrComponent, opts.Component)
	}
	if strings.TrimSpace(opts.Subsystem) != "" {
		logger = logger.With(AttrSubsystem, opts.Subsystem)
	}

	return logger, Diagnostics{
		Level:    level,
		Format:   format,
		Warnings: warnings,
	}
}

// Install creates a logger, optionally installs it as the slog default, and
// optionally redirects the stdlib logger. The returned restore function undoes
// any stdlib redirection and is a no-op when RedirectStdlib is false.
func Install(opts Options) (*slog.Logger, Diagnostics, func()) {
	logger, diagnostics := New(opts)

	stdlib := captureStandardLibraryState()
	restoreDefault := func() {}
	if opts.SetDefault {
		previousDefault := slog.Default()
		slog.SetDefault(logger)
		restoreDefault = func() {
			slog.SetDefault(previousDefault)
		}
	}

	restoreStdlib := func() {}
	if opts.RedirectStdlib {
		restoreStdlib = redirectStandardLibraryWithState(logger, diagnostics.Level, stdlib)
	}

	restore := func() {
		restoreDefault()
		restoreStdlib()
	}

	return logger, diagnostics, restore
}

// InstallAndReport installs a logger and emits any non-fatal configuration
// warnings through the configured logger so callers do not need to duplicate
// bootstrap reporting policy.
func InstallAndReport(opts Options) (*slog.Logger, Diagnostics, func()) {
	logger, diagnostics, restore := Install(opts)
	EmitWarnings(logger, diagnostics.Warnings)
	return logger, diagnostics, restore
}

// EmitWarnings writes logger-configuration diagnostics through the active
// logger using the shared structured-warning contract.
func EmitWarnings(logger *slog.Logger, warnings []Warning) {
	if logger == nil {
		logger = slog.Default()
	}
	for _, warning := range warnings {
		logger.Warn(
			warning.Message,
			AttrCode, warning.Code,
			AttrEnvVar, warning.EnvVar,
			AttrValue, warning.Value,
		)
	}
}

// WithSubsystem scopes a logger to an internal subsystem without changing the
// caller's top-level component identity.
func WithSubsystem(logger *slog.Logger, subsystem string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(subsystem) == "" {
		return logger
	}
	return logger.With(AttrSubsystem, subsystem)
}

// ErrorAttr renders an error into the shared structured logging contract.
func ErrorAttr(err error) slog.Attr {
	if err == nil {
		return slog.Group(AttrError)
	}

	attrs := []any{
		slog.String("message", err.Error()),
		slog.String("type", fmt.Sprintf("%T", err)),
		slog.String("category", string(classifyError(err))),
	}
	attrs = append(attrs, errorContextAttrs(err)...)

	if timeout, retryable := timeoutAndRetryable(err); timeout {
		attrs = append(attrs, slog.Bool("timeout", true))
		if retryable {
			attrs = append(attrs, slog.Bool("retryable", true))
		}
	} else if retryable {
		attrs = append(attrs, slog.Bool("retryable", true))
	}

	return slog.Group(AttrError, attrs...)
}

// ErrorArgs renders an error into the shared structured logging contract and
// omits the field entirely when err is nil.
func ErrorArgs(err error) []any {
	if err == nil {
		return nil
	}
	return []any{ErrorAttr(err)}
}

// Error logs an error using the shared structured error contract.
func Error(logger *slog.Logger, msg string, err error, args ...any) {
	if logger == nil {
		logger = slog.Default()
	}
	args = append(args, ErrorArgs(err)...)
	logger.Error(msg, args...)
}

func resolveLevel(verbose bool) (slog.Level, []Warning) {
	level, err := LevelFromEnv()
	warnings := []Warning{}
	if err != nil {
		warnings = append(warnings, Warning{
			Code:    WarningCodeInvalidLogLevel,
			Message: fmt.Sprintf("Invalid %s value; using info level", LogLevelEnvVar),
			EnvVar:  LogLevelEnvVar,
			Value:   strings.TrimSpace(os.Getenv(LogLevelEnvVar)),
		})
		level = slog.LevelInfo
	}
	if verbose && level > slog.LevelDebug {
		level = slog.LevelDebug
	}
	return level, warnings
}

func resolveFormat(opts Options) (Format, []Warning) {
	switch {
	case opts.Format != "":
		return opts.Format, nil
	case opts.JSON:
		return FormatJSON, nil
	}

	format, err := FormatFromEnv()
	if err != nil {
		return FormatText, []Warning{{
			Code:    WarningCodeInvalidLogFormat,
			Message: fmt.Sprintf("Invalid %s value; using text format", LogFormatEnvVar),
			EnvVar:  LogFormatEnvVar,
			Value:   strings.TrimSpace(os.Getenv(LogFormatEnvVar)),
		}}
	}
	return format, nil
}

// RedirectStandardLibrary routes log.Printf-style calls through slog so older
// code can share one logger during the migration. The returned restore
// function reinstates the prior stdlib logger configuration.
func RedirectStandardLibrary(logger *slog.Logger, level slog.Leveler) func() {
	return redirectStandardLibraryWithState(logger, level, captureStandardLibraryState())
}

func redirectStandardLibraryWithState(logger *slog.Logger, level slog.Leveler, original standardLibraryState) func() {
	if logger == nil {
		return func() {}
	}

	if level == nil {
		level = slog.LevelInfo
	}
	resolvedLevel := slog.LevelInfo
	if level != nil {
		resolvedLevel = level.Level()
	}
	stdLogger := slog.NewLogLogger(logger.Handler(), resolvedLevel)
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(stdLogger.Writer())

	return func() {
		original.restore()
	}
}

type standardLibraryState struct {
	writer io.Writer
	flags  int
	prefix string
}

func captureStandardLibraryState() standardLibraryState {
	return standardLibraryState{
		writer: log.Writer(),
		flags:  log.Flags(),
		prefix: log.Prefix(),
	}
}

func (s standardLibraryState) restore() {
	log.SetOutput(s.writer)
	log.SetFlags(s.flags)
	log.SetPrefix(s.prefix)
}

func classifyError(err error) ErrorCategory {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrorCategoryCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorCategoryTimeout
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, exec.ErrNotFound):
		return ErrorCategoryNotFound
	case errors.Is(err, fs.ErrPermission):
		return ErrorCategoryPermission
	case errors.Is(err, fs.ErrExist):
		return ErrorCategoryAlreadyExist
	case errors.Is(err, os.ErrInvalid):
		return ErrorCategoryInvalid
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ErrorCategoryExternal
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorCategoryNetwork
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return ErrorCategoryNetwork
	}

	switch {
	default:
		return ErrorCategoryRuntime
	}
}

func errorContextAttrs(err error) []any {
	attrs := make([]any, 0, 6)

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		if strings.TrimSpace(pathErr.Op) != "" {
			attrs = append(attrs, slog.String("op", pathErr.Op))
		}
		if strings.TrimSpace(pathErr.Path) != "" {
			attrs = append(attrs, slog.String("path", pathErr.Path))
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if strings.TrimSpace(urlErr.Op) != "" {
			attrs = append(attrs, slog.String("op", urlErr.Op))
		}
		if strings.TrimSpace(urlErr.URL) != "" {
			attrs = append(attrs, slog.String("url", urlErr.URL))
		}
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		attrs = append(attrs, slog.Int("exit_code", exitErr.ExitCode()))
		if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
			attrs = append(attrs, slog.String("stderr", stderr))
		}
	}

	return attrs
}

func timeoutAndRetryable(err error) (timeout bool, retryable bool) {
	type timeoutError interface {
		Timeout() bool
	}
	type temporaryError interface {
		Temporary() bool
	}
	type retryableError interface {
		Retryable() bool
	}

	var te timeoutError
	if errors.As(err, &te) && te.Timeout() {
		timeout = true
		retryable = true
	}

	var temp temporaryError
	if errors.As(err, &temp) && temp.Temporary() {
		retryable = true
	}

	var re retryableError
	if errors.As(err, &re) && re.Retryable() {
		retryable = true
	}

	return timeout, retryable
}
