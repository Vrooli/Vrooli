package vroolicli

import (
	"context"
	"reflect"
	"testing"
)

func TestRecoveryShowDecodesSnakeCaseFields(t *testing.T) {
	// The multi-word fields (anchor_baseline_name, shadow_instance_key,
	// ambient_var) are exactly the ones a camelCase parser silently dropped.
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"scenario":"demo","slug":"wip","mode":"shadow","variant":"shadow",
		"anchor_baseline_name":"baseline-anchor","ambient_var":"demo",
		"shadow_instance_key":"demo@shadow","ttl":"3h0m0s","expired":false
	}`)}}}
	c := New(WithRunner(runner))

	got, err := c.RecoveryShow(context.Background(), "demo", "wip")
	if err != nil {
		t.Fatalf("RecoveryShow: %v", err)
	}
	if got.GetAnchorBaselineName() != "baseline-anchor" {
		t.Errorf("anchor_baseline_name = %q, want baseline-anchor (camelCase-drift regression)", got.GetAnchorBaselineName())
	}
	if got.GetAmbientVar() != "demo" || got.GetShadowInstanceKey() != "demo@shadow" {
		t.Errorf("multi-word fields mismapped: %+v", got)
	}
	if got.GetMode() != "shadow" || got.GetVariant() != "shadow" {
		t.Errorf("mode/variant mismapped: %+v", got)
	}

	wantArgs := []string{"recovery", "show", "--scenario", "demo", "--slug", "wip", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("argv = %v, want %v", runner.calls[0].args, wantArgs)
	}
}

func TestRecoveryShowRequiresScenarioAndSlug(t *testing.T) {
	c := New(WithRunner(&stubRunner{}))
	if _, err := c.RecoveryShow(context.Background(), "", "wip"); err == nil {
		t.Error("expected error when scenario is empty")
	}
	if _, err := c.RecoveryShow(context.Background(), "demo", ""); err == nil {
		t.Error("expected error when slug is empty")
	}
}

func TestRecoveryListDecodesEngagements(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"engagements":[
			{"scenario":"a","slug":"wip","mode":"shadow","anchor_baseline_name":"x"},
			{"scenario":"b","slug":"wip","mode":"live"}
		]
	}`)}}}
	c := New(WithRunner(runner))

	got, err := c.RecoveryList(context.Background())
	if err != nil {
		t.Fatalf("RecoveryList: %v", err)
	}
	if len(got.GetEngagements()) != 2 || got.GetEngagements()[0].GetAnchorBaselineName() != "x" {
		t.Fatalf("engagements mismapped: %+v", got.GetEngagements())
	}
}

func TestRecoveryNamespaceDecodesStorageFields(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{
		"scenario":"demo","variant":"shadow","postgres_db":"vrooli_demo_shadow","data_dir":"/data/demo@shadow"
	}`)}}}
	c := New(WithRunner(runner))

	got, err := c.RecoveryNamespace(context.Background(), "demo", "shadow")
	if err != nil {
		t.Fatalf("RecoveryNamespace: %v", err)
	}
	if got.GetPostgresDb() != "vrooli_demo_shadow" || got.GetDataDir() != "/data/demo@shadow" {
		t.Errorf("storage fields mismapped: %+v", got)
	}
	wantArgs := []string{"recovery", "namespace", "--scenario", "demo", "--variant", "shadow", "--json"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("argv = %v, want %v", runner.calls[0].args, wantArgs)
	}
}
