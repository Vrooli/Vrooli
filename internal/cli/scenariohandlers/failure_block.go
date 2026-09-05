package scenariohandlers

import (
	"errors"
	"fmt"
	"io"
	"strings"

	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

// emitLifecycleFailure writes a human-facing block pointing users at the
// scenario lifecycle log and the follow-up command for tailing it. The
// block is intentionally suppressed in JSON mode — the structured error
// envelope already carries the error shape for programmatic consumers.
func emitLifecycleFailure[C any](
	deps HandlerDeps[C],
	ctx C,
	verb string,
	names []string,
	err error,
) {
	if err == nil {
		return
	}
	if deps.Globals != nil && deps.Globals(ctx).JSON {
		return
	}
	var stderr io.Writer
	if deps.Stderr != nil {
		stderr = deps.Stderr(ctx)
	}
	if stderr == nil {
		return
	}

	home := ""
	if deps.HomeDir != nil {
		if h, hErr := deps.HomeDir(ctx); hErr == nil {
			home = h
		}
	}

	names = cleanFailureNames(names)
	if len(names) == 0 {
		names = []string{""}
	}
	for _, name := range names {
		writeLifecycleFailureBlock(stderr, verb, name, home, err)
	}
}

func writeLifecycleFailureBlock(w io.Writer, verb, name, home string, err error) {
	_, _ = fmt.Fprintln(w)
	if name != "" {
		_, _ = fmt.Fprintf(w, "✗ Failed to %s '%s'\n", verb, name)
	} else {
		_, _ = fmt.Fprintf(w, "✗ Failed to %s\n", verb)
	}
	var phaseErr *lifecycle.PhaseStepError
	if errors.As(err, &phaseErr) {
		if strings.TrimSpace(phaseErr.Phase) != "" {
			_, _ = fmt.Fprintf(w, "  Phase: %s\n", phaseErr.Phase)
		}
		if strings.TrimSpace(phaseErr.Step) != "" {
			_, _ = fmt.Fprintf(w, "  Step: %s\n", phaseErr.Step)
		}
		if exit := phaseErr.ExitCode(); exit > 0 {
			_, _ = fmt.Fprintf(w, "  Exit code: %d\n", exit)
		}
	}
	_, _ = fmt.Fprintf(w, "  Error: %s\n", shortErrorMessage(err))
	if home != "" && name != "" {
		if logPath, err := process.ScenarioLifecycleLogPath(home, name); err == nil {
			_, _ = fmt.Fprintf(w, "  Full log: %s\n", logPath)
		}
	}
	if name != "" {
		_, _ = fmt.Fprintf(w, "  Tail:     vrooli scenario logs %s --tail 100\n", name)
	}
}

// shortErrorMessage reduces the error to its first line, trimmed and
// length-capped. Long stack-like errors (errors.Join chains, piped
// stderr from failing tools) are better surfaced via the log file
// link than rendered inline.
func shortErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = msg[:idx]
	}
	msg = strings.TrimSpace(msg)
	const maxLen = 240
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "…"
	}
	return msg
}

func cleanFailureNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// silentLifecycleError wraps an error so rootcli's PrintErrorWithContext
// skips the default "error: …" printout — we've already surfaced the
// failure via the block above. The exit code contract is preserved:
// callers unwrap via errors.As to find any embedded ExitCode.
type silentLifecycleError struct{ inner error }

func (e silentLifecycleError) Error() string            { return e.inner.Error() }
func (e silentLifecycleError) Unwrap() error            { return e.inner }
func (silentLifecycleError) Silent() bool               { return true }
func (e silentLifecycleError) ExitCode() int            { return vroolierr.ExitCode(e.inner, 1) }
func (e silentLifecycleError) ErrorCategory() string    { return vroolierr.Category(e.inner) }
func (silentLifecycleError) ErrorHint() string          { return "" }
func (silentLifecycleError) ErrorSuggestions() []string { return nil }

// withLifecycleFailureBlock wraps a run function so that any error
// emits a failure block and is then returned as a silent error so the
// root CLI does not re-print it. The signature matches the scenario binder
// contract used by Start/Restart/Stop handlers.
func withLifecycleFailureBlock[C any, Req any](
	deps HandlerDeps[C],
	verb string,
	names func(Req) []string,
	run func(C, cliout.Format, Req) ([]LifecycleItemOutput, error),
) func(C, cliout.Format, Req) ([]LifecycleItemOutput, error) {
	return func(ctx C, format cliout.Format, req Req) ([]LifecycleItemOutput, error) {
		items, err := run(ctx, format, req)
		if err != nil {
			emitLifecycleFailure(deps, ctx, verb, names(req), err)
			err = silentLifecycleError{inner: err}
		}
		return items, err
	}
}
