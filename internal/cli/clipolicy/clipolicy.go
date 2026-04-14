package clipolicy

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	ErrorCategoryUsage       = "Usage"
	ErrorCategoryEnvironment = "Environment"
	ErrorCategoryRuntime     = "Runtime"

	MainHelpHint    = "Run 'vrooli --help' for usage information"
	GeneralHelpHint = "Use --help for available commands"

	UnknownCommandLabel         = "Unknown command"
	UnknownScenarioCommandLabel = "Unknown scenario command"
)

type HelpOnlyError struct {
	text string
}

func (e HelpOnlyError) Error() string    { return e.text }
func (e HelpOnlyError) HelpText() string { return e.text }

func CommandHelpOnly(text string) error {
	return HelpOnlyError{text: text}
}

type commandError interface {
	error
	ErrorCategory() string
	ErrorHint() string
	ErrorSuggestions() []string
}

func NewErrorWithCategory(err error, category, hint string, suggestions []string) error {
	return &vroolierr.Error{
		Err:         err,
		Category:    category,
		Hint:        hint,
		Suggestions: append([]string(nil), suggestions...),
	}
}

func UsageHint(helpTarget string) string {
	if strings.TrimSpace(helpTarget) == "" {
		return GeneralHelpHint
	}
	return fmt.Sprintf("Run 'vrooli %s --help' for usage information", helpTarget)
}

func NewUsageError(message, helpTarget string) error {
	return NewErrorWithCategory(fmt.Errorf("%s", message), ErrorCategoryUsage, UsageHint(helpTarget), nil)
}

func UsageErrorf(helpTarget, format string, args ...any) error {
	return NewUsageError(fmt.Sprintf(format, args...), helpTarget)
}

func UnknownOptionError(command, option string) error {
	return UsageErrorf(command, "unknown option for %s: %s", command, option)
}

func NewUnknownCommandError(command string, suggestions []string) error {
	return &vroolierr.Error{
		Err:         fmt.Errorf("unknown command: %s", command),
		Category:    ErrorCategoryUsage,
		Hint:        MainHelpHint,
		Suggestions: append([]string(nil), suggestions...),
		Code:        "unknown_command",
	}
}

func NewUnknownScenarioCommandError(command string, suggestions []string) error {
	return &vroolierr.Error{
		Err:         fmt.Errorf("unknown scenario command: %s", command),
		Category:    ErrorCategoryUsage,
		Hint:        UsageHint("scenario"),
		Suggestions: append([]string(nil), suggestions...),
		Code:        "unknown_scenario_command",
	}
}

func PrintErrorWithContext(w io.Writer, err error) {
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
	if strings.HasPrefix(strings.ToLower(message), "unknown command: ") && category == ErrorCategoryUsage {
		command := strings.TrimSpace(strings.TrimPrefix(message, "unknown command: "))
		printUnknownCommand(w, UnknownCommandLabel, command, annotated.ErrorSuggestions(), MainHelpHint)
		return
	}
	if strings.HasPrefix(strings.ToLower(message), "unknown scenario command: ") && category == ErrorCategoryUsage {
		command := strings.TrimSpace(strings.TrimPrefix(message, "unknown scenario command: "))
		printUnknownCommand(w, UnknownScenarioCommandLabel, command, annotated.ErrorSuggestions(), UsageHint("scenario"))
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
	_, _ = fmt.Fprintln(w, MainHelpHint)
}

func printUnknownCommand(w io.Writer, label, command string, suggestions []string, usageHint string) {
	_, _ = fmt.Fprintf(w, "%s: %s\n", label, command)
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Did you mean one of these?")
		for _, suggestion := range suggestions {
			_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, usageHint)
}
