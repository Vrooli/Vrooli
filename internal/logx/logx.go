package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
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
