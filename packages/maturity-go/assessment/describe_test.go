package assessment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// writeDescriptor materializes a minimal but valid provider descriptor so the
// loader exercises the real parse path rather than a hand-built struct.
func writeDescriptor(t *testing.T, scenario, phase string, validation map[string]any) string {
	t.Helper()
	root := t.TempDir()
	scenarioDir := filepath.Join(root, scenario)
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	descriptor := map[string]any{
		"scenario": scenario,
		"phase":    phase,
		"maturity": map[string]any{
			"version": "2.1.0",
			"levels": []map[string]any{
				{"id": "L0", "name": "None", "description": "d", "entry_criteria": []string{}, "exit_criteria": []string{"x"}},
				{"id": "L1", "name": "Some", "description": "d", "entry_criteria": []string{"x"}, "exit_criteria": []string{}},
			},
			"findings": map[string]any{},
			"fallback": map[string]any{
				"local_level_impact": "L1",
				"global_impact":      "evolvability_gap",
				"dimension":          "structure",
				"severity_default":   "SEVERITY_WARNING",
			},
		},
	}
	if validation != nil {
		descriptor["validation"] = validation
	}
	raw, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	path := filepath.Join(scenarioDir, ".vrooli", "test-genie.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
	return scenarioDir
}

func TestLoadProviderDescriptionReadsIdentityAndCapabilities(t *testing.T) {
	dir := writeDescriptor(t, "security-health", "security", map[string]any{
		"contract": "scenario-validation/v1",
	})

	got, err := LoadProviderDescription(dir)
	if err != nil {
		t.Fatalf("LoadProviderDescription: %v", err)
	}
	if got.Provider != "security-health" {
		t.Errorf("Provider = %q, want security-health", got.Provider)
	}
	if got.Phase != "security" {
		t.Errorf("Phase = %q, want security", got.Phase)
	}
	if got.SpecVersion != "2.1.0" {
		t.Errorf("SpecVersion = %q, want 2.1.0", got.SpecVersion)
	}
	if got.Contract != "scenario-validation/v1" {
		t.Errorf("Contract = %q", got.Contract)
	}
	// A descriptor with no execution flag has no cheap inspection mode: this is
	// exactly the security-health shape that made a ValidateScenario readiness
	// probe cost a full analysis.
	if got.SupportsExecution {
		t.Error("SupportsExecution = true, want false for an inspection-only provider")
	}
	if got.DeliveryMode != "inline" {
		t.Errorf("DeliveryMode = %q, want inline default", got.DeliveryMode)
	}
}

func TestLoadProviderDescriptionDefaultsContract(t *testing.T) {
	dir := writeDescriptor(t, "proto-health", "proto", nil)

	got, err := LoadProviderDescription(dir)
	if err != nil {
		t.Fatalf("LoadProviderDescription: %v", err)
	}
	if got.Contract != DefaultContract {
		t.Errorf("Contract = %q, want %q", got.Contract, DefaultContract)
	}
}

func TestLoadProviderDescriptionExecutionFlags(t *testing.T) {
	cases := []struct {
		name       string
		validation map[string]any
		want       bool
	}{
		{"explicit execution", map[string]any{"execution": true}, true},
		{"retired includeExecution", map[string]any{"includeExecution": true}, true},
		{"durable-run always executes", map[string]any{"deliveryMode": "durable-run", "execution": true}, true},
		{"inspection only", map[string]any{"deliveryMode": "inline"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeDescriptor(t, "unit-health", "unit", tc.validation)
			got, err := LoadProviderDescription(dir)
			if err != nil {
				t.Fatalf("LoadProviderDescription: %v", err)
			}
			if got.SupportsExecution != tc.want {
				t.Errorf("SupportsExecution = %v, want %v", got.SupportsExecution, tc.want)
			}
		})
	}
}

func TestDescriberZeroValueReturnsUnimplemented(t *testing.T) {
	var d Describer
	if d.Configured() {
		t.Error("zero Describer reports Configured")
	}

	_, err := d.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err == nil {
		t.Fatal("zero Describer returned no error; consumers rely on Unimplemented to fall back")
	}
	if got := connect.CodeOf(err); got != connect.CodeUnimplemented {
		t.Errorf("code = %v, want %v", got, connect.CodeUnimplemented)
	}
}

func TestDescriberAnswersFromLoadedFacts(t *testing.T) {
	dir := writeDescriptor(t, "architecture-cartographer", "architecture", map[string]any{
		"contract": "scenario-validation/v1",
	})
	d, err := LoadDescriber(dir)
	if err != nil {
		t.Fatalf("LoadDescriber: %v", err)
	}
	if !d.Configured() {
		t.Fatal("loaded Describer reports unconfigured")
	}

	resp, err := d.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatalf("DescribeProvider: %v", err)
	}
	msg := resp.Msg
	if msg.GetProvider() != "architecture-cartographer" {
		t.Errorf("Provider = %q", msg.GetProvider())
	}
	if msg.GetPhase() != "architecture" {
		t.Errorf("Phase = %q", msg.GetPhase())
	}
	if msg.GetSpecVersion() != "2.1.0" {
		t.Errorf("SpecVersion = %q", msg.GetSpecVersion())
	}
	if msg.GetCapabilities() == nil {
		t.Fatal("capabilities missing")
	}
	if msg.GetCapabilities().GetDeliveryMode() != "inline" {
		t.Errorf("DeliveryMode = %q", msg.GetCapabilities().GetDeliveryMode())
	}
}

// The response must be immutable across calls: it is resolved once and shared,
// so a consumer mutating what it receives must not corrupt later responses.
func TestDescribeProviderReturnsIndependentCopies(t *testing.T) {
	dir := writeDescriptor(t, "storage-manager", "storage", nil)
	d, err := LoadDescriber(dir)
	if err != nil {
		t.Fatalf("LoadDescriber: %v", err)
	}

	first, err := d.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	first.Msg.Provider = "mutated"

	second, err := d.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second.Msg.GetProvider() != "storage-manager" {
		t.Errorf("Provider = %q after caller mutated an earlier response", second.Msg.GetProvider())
	}
}

func TestWithFixesSetsCapabilityWithoutMutatingSource(t *testing.T) {
	base := NewDescriber(ProviderDescription{
		Provider: "quality-health", Phase: "quality", Contract: DefaultContract, DeliveryMode: "inline",
	})
	withFixes := base.WithFixes(true)

	baseResp, err := base.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatalf("base: %v", err)
	}
	if baseResp.Msg.GetCapabilities().GetSupportsFixes() {
		t.Error("WithFixes mutated the source Describer")
	}

	fixResp, err := withFixes.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatalf("withFixes: %v", err)
	}
	if !fixResp.Msg.GetCapabilities().GetSupportsFixes() {
		t.Error("SupportsFixes = false after WithFixes(true)")
	}
}

func TestWithFixesOnZeroDescriberStaysUnimplemented(t *testing.T) {
	var d Describer
	if d.WithFixes(true).Configured() {
		t.Error("WithFixes configured a zero Describer; it must stay Unimplemented")
	}
}

func TestCurrentBuildNeverInventsValues(t *testing.T) {
	build := CurrentBuild()
	// Under `go test` the binary always exists, so mtime should resolve. The
	// contract that matters is that nothing is fabricated: a zero value means
	// unknown, and must never be a plausible-looking recent timestamp.
	if !build.BinaryModifiedAt.IsZero() && build.BinaryModifiedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("BinaryModifiedAt is in the future: %v", build.BinaryModifiedAt)
	}
	if !build.BuiltAt.IsZero() && build.BuiltAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("BuiltAt is in the future: %v", build.BuiltAt)
	}
}

func TestToProtoNilReceiver(t *testing.T) {
	var d *ProviderDescription
	if d.ToProto() != nil {
		t.Error("nil ProviderDescription must render nil")
	}
}
