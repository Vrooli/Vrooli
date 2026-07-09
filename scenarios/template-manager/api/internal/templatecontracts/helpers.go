package templatecontracts

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

func TemplateCommandHelpText() string {
	return commandtree.RenderHelpText(commandtree.Help{
		Title:        "Template Manager Template Commands",
		Usage:        "template-manager template <subcommand> [options]",
		DefaultGroup: "Scenario Templates",
	}, templateCommandSpecs())
}

func DesignCommandHelpText() string {
	return commandtree.RenderHelpText(commandtree.Help{
		Title:        "Template Manager Design Commands",
		Usage:        "template-manager design <subcommand> [options]",
		DefaultGroup: "Scenario Design",
	}, designCommandSpecs())
}

func TemplateGenerateHelpText() string {
	return commandtree.HelpText("", "template-manager generate", "Scaffold a scenario from a template.", commandtree.Help{
		Usage: "template-manager generate <template> --id <slug> --display-name <name> --description <text> [options]",
	}, templateGenerateArgSchema())
}

func FormatScenarioTemplateRequiredFlags(requiredVars map[string]TemplateVar) string {
	parts := make([]string, 0, len(requiredVars))
	keys := make([]string, 0, len(requiredVars))
	for key := range requiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		flag := strings.TrimSpace(requiredVars[key].Flag)
		if flag == "" {
			flag = "--var " + key + "=..."
		}
		parts = append(parts, flag)
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

func marshalScenarioStatus(w io.Writer, msg any) error {
	if msg == nil {
		return fmt.Errorf("cannot marshal nil message")
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(msg)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
