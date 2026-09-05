package setup

import (
	"testing"

	"github.com/vrooli/vrooli/internal/operatorinput"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
)

func TestPresetsExpandToTargetedSelections(t *testing.T) {
	for _, name := range []string{"development", "production", "minimal", "managed-connection", "presence", "deployment-target", "production-runtime", "development-runner"} {
		selection, err := ExpandPreset(name, "node-1")
		if err != nil {
			t.Fatalf("ExpandPreset(%q): %v", name, err)
		}
		if selection.SchemaVersion != "v1" || selection.Target != "node-1" || len(selection.Scenarios) == 0 {
			t.Fatalf("ExpandPreset(%q) = %#v; want version, target, and scenarios", name, selection)
		}
	}
}

func TestExpandPresetRejectsUnknownName(t *testing.T) {
	if _, err := ExpandPreset("staging", "local"); err == nil {
		t.Fatal("unknown preset was accepted")
	}
}

func TestOperatorInputKindsStayAlignedWithWireEnum(t *testing.T) {
	for _, kind := range []operatorinput.Kind{
		operatorinput.KindSecret, operatorinput.KindChoice, operatorinput.KindConfirm,
		operatorinput.KindPath, operatorinput.KindEnum, operatorinput.KindBoolean,
		operatorinput.KindDuration, operatorinput.KindConfirmation,
	} {
		if value, ok := OperatorInputKind(kind); !ok || value == setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_UNSPECIFIED {
			t.Fatalf("operator input kind %q has no wire enum", kind)
		}
	}
}
