package setup

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const SelectionSchemaVersion = "v1"

// ValidateSelectionContract rejects an explicitly supplied selection version
// that this control plane cannot interpret. An empty version remains accepted
// for compatibility with older callers that predate the version field.
func ValidateSelectionContract(selection *setupv1.Selection) error {
	if selection == nil {
		return errors.New("setup selection is required")
	}
	version := strings.TrimSpace(selection.GetSchemaVersion())
	if version != "" && version != SelectionSchemaVersion {
		return fmt.Errorf("unsupported setup selection schema version %q; supported version is %q", version, SelectionSchemaVersion)
	}
	return nil
}

// DecodeSelectionB64 decodes the argv-safe setup/v1 handoff used by remote
// setup invocations. The selection is validated before setup can resolve or
// apply host requirements, so an unsupported contract fails closed.
func DecodeSelectionB64(encoded string) (*setupv1.Selection, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode setup selection: %w", err)
	}
	selection := &setupv1.Selection{}
	if err := protojson.Unmarshal(raw, selection); err != nil {
		return nil, fmt.Errorf("decode setup selection JSON: %w", err)
	}
	if err := ValidateSelectionContract(selection); err != nil {
		return nil, err
	}
	return selection, nil
}

// OperatorInputKind converts the control-plane vocabulary to the versioned
// wire enum. Keeping this mapping beside preset expansion gives schema tests a
// single place to detect vocabulary drift.
func OperatorInputKind(kind operatorcapability.Kind) (setupv1.OperatorInputKind, bool) {
	values := map[operatorcapability.Kind]setupv1.OperatorInputKind{
		operatorcapability.KindSecret:       setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_SECRET,
		operatorcapability.KindChoice:       setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CHOICE,
		operatorcapability.KindConfirm:      setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRM,
		operatorcapability.KindPath:         setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_PATH,
		operatorcapability.KindEnum:         setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_ENUM,
		operatorcapability.KindBoolean:      setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_BOOLEAN,
		operatorcapability.KindDuration:     setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_DURATION,
		operatorcapability.KindConfirmation: setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRMATION,
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
