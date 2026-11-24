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
