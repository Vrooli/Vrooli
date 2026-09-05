package setup

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorinput"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
)

// OperatorInputKind converts the control-plane vocabulary to the versioned
// wire enum. Keeping this mapping beside preset expansion gives schema tests a
// single place to detect vocabulary drift.
func OperatorInputKind(kind operatorinput.Kind) (setupv1.OperatorInputKind, bool) {
	values := map[operatorinput.Kind]setupv1.OperatorInputKind{
		operatorinput.KindSecret:       setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_SECRET,
		operatorinput.KindChoice:       setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CHOICE,
		operatorinput.KindConfirm:      setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRM,
		operatorinput.KindPath:         setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_PATH,
		operatorinput.KindEnum:         setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_ENUM,
		operatorinput.KindBoolean:      setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_BOOLEAN,
		operatorinput.KindDuration:     setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_DURATION,
		operatorinput.KindConfirmation: setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRMATION,
	}
	value, ok := values[kind]
	return value, ok
}

// Preset is a named operator convenience. Presets expand in memory into the
// versioned Selection contract; the preset name itself is never persisted as
// node configuration.
type Preset struct {
	Name        string
	Description string
	Environment string
	Selection   *setupv1.Selection
}

var presets = map[string]Preset{
	"development":        {Name: "development", Environment: "development", Description: "A local development node", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"web-console"}, UpdateControl: "own", SessionMode: "interactive"}},
	"production":         {Name: "production", Environment: "production", Description: "A managed production node", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge"}, UpdateControl: "guard", SessionMode: "service"}},
	"minimal":            {Name: "minimal", Environment: "minimal", Description: "A minimal connected node", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge"}, UpdateControl: "observe", SessionMode: "service"}},
	"managed-connection": {Name: "managed-connection", Environment: "minimal", Description: "A node reachable by the control plane", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge"}, UpdateControl: "observe", SessionMode: "service"}},
	"presence":           {Name: "presence", Environment: "minimal", Description: "A node that reports presence", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge"}, UpdateControl: "observe", SessionMode: "service"}},
	"deployment-target":  {Name: "deployment-target", Environment: "production", Description: "A node used for deployments", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge", "deployment-manager"}, UpdateControl: "guard", SessionMode: "service"}},
	"production-runtime": {Name: "production-runtime", Environment: "production", Description: "A node running production workloads", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge", "system-monitor"}, UpdateControl: "guard", SessionMode: "service"}},
	"development-runner": {Name: "development-runner", Environment: "development", Description: "A node used for development runs", Selection: &setupv1.Selection{SchemaVersion: "v1", Scenarios: []string{"vrooli-bridge", "test-genie"}, UpdateControl: "own", SessionMode: "interactive"}},
	"custom":             {Name: "custom", Environment: "development", Description: "A node configured from explicit selections", Selection: &setupv1.Selection{SchemaVersion: "v1", UpdateControl: "own", SessionMode: "interactive"}},
}

// EnvironmentForPreset is the sole expansion point for the legacy setup
// environment switch. Callers carry only a named preset across APIs and
// persistence; bootstrap receives the expanded value at the final host seam.
func EnvironmentForPreset(name string) (string, bool) {
	preset, ok := presets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", false
	}
	return preset.Environment, true
}

// Presets returns a copy of the registered preset names in stable order.
func Presets() []Preset {
	result := make([]Preset, 0, len(presets))
	for _, preset := range presets {
		result = append(result, clonePreset(preset))
	}
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Name < result[j-1].Name; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}

// ExpandPreset returns an inspectable selection with the requested target.
func ExpandPreset(name, target string) (*setupv1.Selection, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	preset, ok := presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown setup preset %q", name)
	}
	selection := clonePreset(preset).Selection
	selection.Target = strings.TrimSpace(target)
	return selection, nil
}

func clonePreset(preset Preset) Preset {
	selection := *preset.Selection
	selection.Scenarios = append([]string(nil), preset.Selection.Scenarios...)
	return Preset{Name: preset.Name, Description: preset.Description, Environment: preset.Environment, Selection: &selection}
}
