package support

import (
	"fmt"
	"strings"
)

// FormatNextSteps returns the bounded progressive-disclosure footer shared by
// run-inspection commands. Empty suggestions are ignored.
func FormatNextSteps(commands ...string) string {
	steps := make([]string, 0, len(commands))
	for _, command := range commands {
		if command = strings.TrimSpace(command); command != "" {
			steps = append(steps, command)
		}
	}
	if len(steps) > 0 {
		return fmt.Sprintf("Next: %s", strings.Join(steps, "; "))
	}
	return ""
}

// NextSteps emits the shared inspection footer.
func NextSteps(commands ...string) {
	if footer := FormatNextSteps(commands...); footer != "" {
		fmt.Println(footer)
	}
}
