package measures

import (
	"strings"
	"testing"
)

func i64(v int64) *int64 { return &v }

func TestAssembleJoinsThreeSources(t *testing.T) {
	// proto-derived schema: a time_window message field, a static enum, and a
	// plain string the manifest upgrades to a dynamic enum.
	protoParams := []ParamSchema{
		{Name: "window", Type: ParamTypeTimeWindow, MessageType: TimeWindowMessageName},
		{Name: "status", Type: "enum", EnumValues: []string{"done", "archived"}},
		{Name: "initiative", Type: "string"},
		{Name: "limit", Type: "int32", Optional: true, Min: i64(1), Max: i64(100)},
	}
	m := ManifestMeasure{
		Intent:    "How many backlog items completed.",
		Questions: []string{"how many backlog items did we complete this week"},
		Params: map[string]ManifestParam{
			"window":     {Default: "this_week"},
			"initiative": {ValuesSource: "initiative_names"},
		},
		Result: Result{Kind: ResultScalar, ValueField: "count", Unit: "items", SummaryTemplate: "{count} items"},
	}
	g := Governance{Effect: EffectRead, RunEligible: true}
	b := Binding{Service: "StatsService", Method: "BacklogCompletionCount"}

	decl, err := Assemble("backlog.completed", "backlog", b, m, g, protoParams)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if decl.Params["window"].Type != ParamTypeTimeWindow {
		t.Errorf("window type = %q", decl.Params["window"].Type)
	}
	if decl.Params["window"].Default != "this_week" {
		t.Errorf("window default not overlaid: %q", decl.Params["window"].Default)
	}
	// proto static enum preserved
	if got := decl.Params["status"]; got.Type != ParamTypeEnum || len(got.EnumValues) != 2 {
		t.Errorf("status = %+v", got)
	}
	// manifest upgraded the plain string to a dynamic enum
	if got := decl.Params["initiative"]; got.Type != ParamTypeEnum || got.ValuesSource != "initiative_names" {
		t.Errorf("initiative = %+v, want dynamic enum", got)
	}
	// bounds preserved
	if p := decl.Params["limit"]; p.Min == nil || *p.Min != 1 || p.Max == nil || *p.Max != 100 {
		t.Errorf("limit bounds = %+v", p)
	}
	if !decl.AutoExecutable() {
		t.Error("read + run_eligible should be auto-executable")
	}
}

func TestAssembleRejectsManifestParamWithoutProtoField(t *testing.T) {
	protoParams := []ParamSchema{{Name: "window", Type: ParamTypeTimeWindow}}
	m := ManifestMeasure{
		Questions: []string{"q"},
		Params:    map[string]ManifestParam{"ghost": {Default: "x"}},
		Result:    Result{Kind: ResultScalar, ValueField: "count"},
	}
	_, err := Assemble("x.y", "x", Binding{Service: "S", Method: "M"}, m, Governance{Effect: EffectRead}, protoParams)
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("expected drift error naming ghost, got %v", err)
	}
}

func TestAssembleRequiresValueField(t *testing.T) {
	protoParams := []ParamSchema{{Name: "window", Type: ParamTypeTimeWindow}}
	m := ManifestMeasure{
		Questions: []string{"q"},
		Result:    Result{Kind: ResultScalar}, // missing value_field
	}
	_, err := Assemble("x.y", "x", Binding{Service: "S", Method: "M"}, m, Governance{Effect: EffectRead}, protoParams)
	if err == nil || !strings.Contains(err.Error(), "value_field") {
		t.Fatalf("expected value_field error, got %v", err)
	}
}

func TestAssembleInfersRequiredFromNonOptional(t *testing.T) {
	// A non-optional, non-defaulted scalar param is inferred required so a
	// missing value abstains rather than silently answering wrong.
	protoParams := []ParamSchema{{Name: "id", Type: "string", Format: "uuid"}}
	m := ManifestMeasure{Questions: []string{"q"}, Result: Result{Kind: ResultScalar, ValueField: "v"}}
	decl, err := Assemble("x.y", "x", Binding{Service: "S", Method: "M"}, m, Governance{Effect: EffectRead}, protoParams)
	if err != nil {
		t.Fatal(err)
	}
	if !decl.Params["id"].Required {
		t.Error("non-optional non-defaulted scalar should be inferred required")
	}
}

func TestEffectGate(t *testing.T) {
	if EffectRead.AutoExecutable() != true || EffectWrite.AutoExecutable() || EffectDestructive.AutoExecutable() {
		t.Error("only read effect is auto-executable")
	}
}
