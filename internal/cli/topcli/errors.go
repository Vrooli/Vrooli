package topcli

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

const usageCategory = "Usage"

type helpOnlyError struct {
	text string
}

func (e helpOnlyError) Error() string    { return e.text }
func (e helpOnlyError) HelpText() string { return e.text }

func commandHelpOnly(text string) error {
	return helpOnlyError{text: text}
}

func usageHint(helpTarget string) string {
	if strings.TrimSpace(helpTarget) == "" {
		return "Use --help for available commands"
	}
	return fmt.Sprintf("Run 'vrooli %s --help' for usage information", helpTarget)
}

func usageErrorf(helpTarget, format string, args ...any) error {
	return &vroolierr.Error{
		Err:      fmt.Errorf(format, args...),
		Category: usageCategory,
		Hint:     usageHint(helpTarget),
	}
}

func unknownOptionError(command, option string) error {
	return usageErrorf(command, "unknown option for %s: %s", command, option)
}
