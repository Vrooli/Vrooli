package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

func suggestTopLevelCommands(command string) []string {
	return suggestCommandNames(command, topLevelCommandNames())
}

func suggestScenarioCommands(command string) []string {
	return suggestCommandNames(command, scenarioCommandNames())
}

func suggestCommandNames(command string, names []string) []string {
	candidates := make([]string, 0, len(names))
	for _, candidate := range names {
		if candidate == command {
			continue
		}
		if simpleDistance(command, candidate) <= 2 {
			candidates = append(candidates, candidate)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func printErrorWithContext(w io.Writer, err error) {
	if err == nil {
		return
	}
	if silent, ok := err.(interface{ Silent() bool }); ok && silent.Silent() {
		return
	}
	annotated, ok := err.(commandError)
	if !ok {
		_, _ = fmt.Fprintln(w, err)
		return
	}
	category := strings.TrimSpace(annotated.ErrorCategory())
	message := annotated.Error()
	if strings.HasPrefix(strings.ToLower(message), "unknown command: ") && category == errorCategoryUsage {
		command := strings.TrimSpace(strings.TrimPrefix(message, "unknown command: "))
		printUnknownCommand(w, command, annotated.ErrorSuggestions())
		return
	}
	if strings.HasPrefix(strings.ToLower(message), "unknown scenario command: ") && category == errorCategoryUsage {
		command := strings.TrimSpace(strings.TrimPrefix(message, "unknown scenario command: "))
		printUnknownScenarioCommand(w, command, annotated.ErrorSuggestions())
		return
	}
	if category != "" {
		_, _ = fmt.Fprintf(w, "%s error: %s\n", category, message)
	} else {
		_, _ = fmt.Fprintln(w, message)
	}
	if hint := strings.TrimSpace(annotated.ErrorHint()); hint != "" {
		_, _ = fmt.Fprintln(w, hint)
	}
	suggestions := annotated.ErrorSuggestions()
	if len(suggestions) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Did you mean one of these?")
	for _, suggestion := range suggestions {
		_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
	}
	_, _ = fmt.Fprintln(w, "Run 'vrooli --help' for usage information")
}

type commandError interface {
	error
	ErrorCategory() string
	ErrorHint() string
	ErrorSuggestions() []string
}

func newErrorWithCategory(err error, category, hint string, suggestions []string) error {
	return &vroolierr.Error{
		Err:         err,
		Category:    category,
		Hint:        hint,
		Suggestions: append([]string(nil), suggestions...),
	}
}

func usageHint(helpTarget string) string {
	if strings.TrimSpace(helpTarget) == "" {
		return "Use --help for available commands"
	}
	return fmt.Sprintf("Run 'vrooli %s --help' for usage information", helpTarget)
}

func newUsageError(message, helpTarget string) error {
	return newErrorWithCategory(errors.New(message), errorCategoryUsage, usageHint(helpTarget), nil)
}

func usageErrorf(helpTarget, format string, args ...any) error {
	return newUsageError(fmt.Sprintf(format, args...), helpTarget)
}

func unknownOptionError(command, option string) error {
	return usageErrorf(command, "unknown option for %s: %s", command, option)
}

func invalidArgumentError(command, message string) error {
	return newUsageError(message, command)
}

func newUnknownCommandError(command string) error {
	return &vroolierr.Error{
		Err:         fmt.Errorf("unknown command: %s", command),
		Category:    errorCategoryUsage,
		Hint:        "Run 'vrooli --help' for usage information",
		Suggestions: suggestTopLevelCommands(command),
		Code:        "unknown_command",
	}
}

func newUnknownScenarioCommandError(command string) error {
	return &vroolierr.Error{
		Err:         fmt.Errorf("unknown scenario command: %s", command),
		Category:    errorCategoryUsage,
		Hint:        "Run 'vrooli scenario --help' for usage information",
		Suggestions: suggestScenarioCommands(command),
		Code:        "unknown_scenario_command",
	}
}

// simpleDistance intentionally uses a cheap prefix-aware heuristic instead of a
// full edit-distance implementation. The CLI only needs rough typo recovery for
// obvious mistakes, and keeping this dependency-free keeps startup lightweight.
func simpleDistance(left, right string) int {
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}
	distance := maxLen - minLen
	for i := 0; i < minLen; i++ {
		if left[i] != right[i] {
			distance++
		}
	}
	return distance
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if codeErr, ok := err.(exitCodeError); ok {
		return codeErr.ExitCode()
	}
	return vroolierr.ExitCode(err, 1)
}

type exitCodeError struct {
	code    int
	message string
	silent  bool
}

func (e exitCodeError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func (e exitCodeError) ExitCode() int {
	return e.code
}

func (e exitCodeError) Silent() bool {
	return e.silent
}
