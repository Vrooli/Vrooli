package setup

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestValidateSelectionContractRejectsUnknownVersion(t *testing.T) {
	selection := &setupv1.Selection{SchemaVersion: "v99"}
	err := ValidateSelectionContract(selection)
	if err == nil || !strings.Contains(err.Error(), "unsupported setup selection schema version") {
		t.Fatalf("ValidateSelectionContract error = %v", err)
	}
}

func TestValidateSelectionContractAcceptsLegacyAndCurrentVersions(t *testing.T) {
	for _, version := range []string{"", SelectionSchemaVersion} {
		if err := ValidateSelectionContract(&setupv1.Selection{SchemaVersion: version}); err != nil {
			t.Fatalf("version %q rejected: %v", version, err)
		}
	}
}

func TestDecodeSelectionB64RoundTripsTypedSelection(t *testing.T) {
	want := &setupv1.Selection{SchemaVersion: "v1", Target: "node-1", Scenarios: []string{"web", "worker"}, OptionalResources: []string{"redis"}, Apply: true}
	raw, err := protojson.Marshal(want)
	if err != nil {
		t.Fatalf("marshal selection: %v", err)
	}
	got, err := DecodeSelectionB64(base64.RawURLEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatalf("DecodeSelectionB64: %v", err)
	}
	if got.GetSchemaVersion() != want.GetSchemaVersion() || got.GetTarget() != want.GetTarget() || !strings.EqualFold(strings.Join(got.GetScenarios(), ","), "web,worker") || got.GetOptionalResources()[0] != "redis" || !got.GetApply() {
		t.Fatalf("decoded selection = %+v", got)
	}
}

func TestDecodeSelectionB64RejectsUnsupportedVersion(t *testing.T) {
	raw, err := protojson.Marshal(&setupv1.Selection{SchemaVersion: "v99"})
	if err != nil {
		t.Fatalf("marshal selection: %v", err)
	}
	_, err = DecodeSelectionB64(base64.RawURLEncoding.EncodeToString(raw))
	if err == nil || !strings.Contains(err.Error(), "unsupported setup selection schema version") {
		t.Fatalf("DecodeSelectionB64 error = %v", err)
	}
}

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
	for _, kind := range []operatorcapability.Kind{
		operatorcapability.KindSecret, operatorcapability.KindChoice, operatorcapability.KindConfirm,
		operatorcapability.KindPath, operatorcapability.KindEnum, operatorcapability.KindBoolean,
		operatorcapability.KindDuration, operatorcapability.KindConfirmation,
	} {
		if value, ok := OperatorInputKind(kind); !ok || value == setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_UNSPECIFIED {
			t.Fatalf("operator input kind %q has no wire enum", kind)
		}
	}
}
