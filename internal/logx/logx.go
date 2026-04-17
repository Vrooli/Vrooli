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
	"regexp"
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
	// AttrBuildTime identifies an embedded binary build timestamp.
	AttrBuildTime = "build_time"
	// AttrChecks identifies the number of health checks or validation checks.
	AttrChecks = "checks"
	// AttrCode identifies a machine-readable warning or error code.
	AttrCode = "code"
	// AttrComponent identifies the top-level binary or service producing logs.
	AttrComponent = "component"
	// AttrCommit identifies an embedded git commit identifier.
	AttrCommit = "commit"
	// AttrDependency identifies a dependent scenario or resource.
	AttrDependency = "dependency"
	// AttrEnvVar identifies an environment variable involved in log decisions.
	AttrEnvVar = "env_var"
	// AttrError stores structured error details as a nested slog group.
	AttrError = "error"
	// AttrFingerprint identifies an embedded source fingerprint.
	AttrFingerprint = "fingerprint"
	// AttrOperation identifies the internal operation currently executing.
	AttrOperation = "operation"
	// AttrPhase identifies a scenario lifecycle phase.
	AttrPhase = "phase"
	// AttrPort identifies a network port number.
	AttrPort = "port"
	// AttrPorts identifies a set of runtime ports.
	AttrPorts = "ports"
	// AttrProcesses identifies a process count or process payload.
	AttrProcesses = "processes"
	// AttrProjectMode identifies whether a phase is running in project mode.
	AttrProjectMode = "project_mode"
	// AttrScenario identifies a scenario slug or name.
	AttrScenario = "scenario"
	// AttrSource identifies how a configuration value was resolved.
	AttrSource = "source"
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
	// Quiet raises the logging level to warn unless a more verbose level is
	// already configured (and Verbose has not overridden it). Quiet is
	// ignored when Verbose is set.
	Quiet bool
	// SetDefault installs the created logger as slog.Default.
	SetDefault bool
	// RedirectStdlib routes package log output through the structured logger.
	RedirectStdlib bool
	// StdlibLevel controls the severity assigned to redirected stdlib log
	// records. Nil defaults to info so legacy log.Printf callers do not inherit
	// the handler's minimum threshold as their record severity.
	StdlibLevel slog.Leveler
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
	// Source identifies where the invalid or advisory value came from.
	Source ConfigSource
	// Value stores the raw user-provided value that triggered the warning.
	Value string
}

// ConfigSource identifies how a logging setting was resolved.
type ConfigSource string

const (
	ConfigSourceDefault         ConfigSource = "default"
	ConfigSourceEnv             ConfigSource = "env"
	ConfigSourceOptions         ConfigSource = "options"
	ConfigSourceJSONCompat      ConfigSource = "json_compat"
	ConfigSourceVerboseOverride ConfigSource = "verbose_override"
)

// Diagnostics captures decisions made while configuring a logger.
type Diagnostics struct {
	// Level is the effective minimum level installed on the handler.
	Level slog.Level
	// LevelSource identifies how the effective level was resolved.
	LevelSource ConfigSource
	// Format is the effective log encoding installed on the handler.
	Format Format
	// FormatSource identifies how the effective format was resolved.
	FormatSource ConfigSource
	// Warnings contains non-fatal configuration issues encountered while
	// building the logger.
	Warnings []Warning
}

// ErrorDetails describes the machine-readable error metadata shared across the
// project-level logging contract.
type ErrorDetails struct {
	Message   string
	Type      string
	Category  ErrorCategory
	Operation string
	Path      string
	URL       string
	ExitCode  int
	Stderr    string
	Timeout   bool
	Retryable bool
}

const maxStructuredErrorTextLen = 512

var (
	bearerCredentialPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/\-=]+`)
	sensitiveValuePattern   = regexp.MustCompile(`(?im)\b(authorization|api[_-]?key|token|secret|password|passwd)\b([[:space:]"'=:_-]+)([^[:space:]",;]+)`)
)

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

	level, levelSource, warnings := resolveLevel(opts.Verbose, opts.Quiet)
	format, formatSource, formatWarnings := resolveFormat(opts)
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
		Level:        level,
		LevelSource:  levelSource,
		Format:       format,
		FormatSource: formatSource,
		Warnings:     warnings,
	}
}

// Install creates a logger, optionally installs it as the slog default, and
// optionally redirects the stdlib logger. The returned restore function undoes
// any process-global logger mutations. This helper is intended for process
// bootstrap only.
func Install(opts Options) (*slog.Logger, Diagnostics, func()) {
	logger, diagnostics := New(opts)

	restoreDefault := func() {}
	restoreStdlib := func() {}
	stdlib := standardLibraryState{}
	if opts.SetDefault || opts.RedirectStdlib {
		stdlib = captureStandardLibraryState()
	}
	if opts.SetDefault {
		previousDefault := slog.Default()
		slog.SetDefault(logger)
		restoreDefault = func() {
			slog.SetDefault(previousDefault)
		}
		if !opts.RedirectStdlib {
			restoreStdlib = func() {
				stdlib.restore()
			}
		}
	}

	if opts.RedirectStdlib {
		restoreStdlib = redirectStandardLibraryWithState(logger, resolveStdlibLevel(opts), stdlib)
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
		args := []any{
			AttrCode, warning.Code,
			AttrValue, warning.Value,
		}
		if warning.EnvVar != "" {
			args = append(args, AttrEnvVar, warning.EnvVar)
		}
		if warning.Source != "" {
			args = append(args, AttrSource, warning.Source)
		}
		logger.Warn(
			warning.Message,
			args...,
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

	details := DescribeError(err)
	attrs := []any{
		slog.String("message", details.Message),
		slog.String("type", details.Type),
		slog.String("category", string(details.Category)),
	}
	if details.Operation != "" {
		attrs = append(attrs, slog.String("op", details.Operation))
	}
	if details.Path != "" {
		attrs = append(attrs, slog.String("path", details.Path))
	}
	if details.URL != "" {
		attrs = append(attrs, slog.String("url", details.URL))
	}
	if details.ExitCode != 0 {
		attrs = append(attrs, slog.Int("exit_code", details.ExitCode))
	}
	if details.Stderr != "" {
		attrs = append(attrs, slog.String("stderr", details.Stderr))
	}

	if details.Timeout {
		attrs = append(attrs, slog.Bool("timeout", true))
		if details.Retryable {
			attrs = append(attrs, slog.Bool("retryable", true))
		}
	} else if details.Retryable {
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

func resolveLevel(verbose, quiet bool) (slog.Level, ConfigSource, []Warning) {
	rawValue := strings.TrimSpace(os.Getenv(LogLevelEnvVar))
	level, err := LevelFromEnv()
	warnings := []Warning{}
	source := ConfigSourceDefault
	if err != nil {
		warnings = append(warnings, Warning{
			Code:    WarningCodeInvalidLogLevel,
			Message: fmt.Sprintf("Invalid %s value; using info level", LogLevelEnvVar),
			EnvVar:  LogLevelEnvVar,
			Source:  ConfigSourceEnv,
			Value:   rawValue,
		})
		level = slog.LevelInfo
	} else if rawValue != "" {
		source = ConfigSourceEnv
	}
	if verbose && level > slog.LevelDebug {
		level = slog.LevelDebug
		source = ConfigSourceVerboseOverride
	} else if !verbose && quiet && level > slog.LevelDebug && level < slog.LevelWarn {
		level = slog.LevelWarn
		source = ConfigSourceVerboseOverride
	}
	return level, source, warnings
}

func resolveFormat(opts Options) (Format, ConfigSource, []Warning) {
	switch {
	case opts.Format != "":
		format, err := ParseFormat(string(opts.Format))
		if err != nil {
			return FormatText, ConfigSourceDefault, []Warning{{
				Code:    WarningCodeInvalidLogFormat,
				Message: "Invalid explicit log format; using text format",
				Source:  ConfigSourceOptions,
				Value:   strings.TrimSpace(string(opts.Format)),
			}}
		}
		return format, ConfigSourceOptions, nil
	case opts.JSON:
		return FormatJSON, ConfigSourceJSONCompat, nil
	}

	rawValue := strings.TrimSpace(os.Getenv(LogFormatEnvVar))
	format, err := FormatFromEnv()
	if err != nil {
		return FormatText, ConfigSourceDefault, []Warning{{
			Code:    WarningCodeInvalidLogFormat,
			Message: fmt.Sprintf("Invalid %s value; using text format", LogFormatEnvVar),
			EnvVar:  LogFormatEnvVar,
			Source:  ConfigSourceEnv,
			Value:   rawValue,
		}}
	}
	if rawValue == "" {
		return format, ConfigSourceDefault, nil
	}
	return format, ConfigSourceEnv, nil
}

// RedirectStandardLibrary routes log.Printf-style calls through slog so
// packages using the standard library logger share the configured handler. The
// level controls the severity assigned to redirected stdlib records; nil
// defaults to info. The returned restore function reinstates the prior stdlib
// logger configuration.
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

// ClassifyError identifies the high-level operational class of an error.
func ClassifyError(err error) ErrorCategory {
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

// DescribeError extracts the structured error metadata used by the shared
// logging contract so callers and tests can reason about categorization
// without parsing a log record.
func DescribeError(err error) ErrorDetails {
	if err == nil {
		return ErrorDetails{}
	}

	details := ErrorDetails{
		Message:  sanitizeStructuredText(err.Error()),
		Type:     fmt.Sprintf("%T", err),
		Category: ClassifyError(err),
	}

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		if strings.TrimSpace(pathErr.Op) != "" {
			details.Operation = pathErr.Op
		}
		if strings.TrimSpace(pathErr.Path) != "" {
			details.Path = pathErr.Path
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if strings.TrimSpace(urlErr.Op) != "" {
			details.Operation = urlErr.Op
		}
		if strings.TrimSpace(urlErr.URL) != "" {
			details.URL = sanitizeStructuredURL(urlErr.URL)
		}
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		details.ExitCode = exitErr.ExitCode()
		if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
			details.Stderr = sanitizeStructuredText(stderr)
		}
	}

	details.Timeout, details.Retryable = timeoutAndRetryable(err)
	return details
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

func resolveStdlibLevel(opts Options) slog.Leveler {
	if opts.StdlibLevel != nil {
		return opts.StdlibLevel
	}
	return slog.LevelInfo
}

func sanitizeStructuredURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return sanitizeStructuredText(value)
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.User("redacted")
		} else {
			parsed.User = nil
		}
	}
	if query := parsed.Query(); len(query) > 0 {
		for key := range query {
			query.Set(key, "[REDACTED]")
		}
		parsed.RawQuery = query.Encode()
	}
	parsed.Fragment = ""
	return truncateStructuredText(parsed.String())
}

func sanitizeStructuredText(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = bearerCredentialPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = sensitiveValuePattern.ReplaceAllString(value, `$1$2[REDACTED]`)
	return truncateStructuredText(value)
}

func truncateStructuredText(value string) string {
	if len(value) <= maxStructuredErrorTextLen {
		return value
	}
	return fmt.Sprintf("%s... [truncated %d chars]", value[:maxStructuredErrorTextLen], len(value)-maxStructuredErrorTextLen)
}
