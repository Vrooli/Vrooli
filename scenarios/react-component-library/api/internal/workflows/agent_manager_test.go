package workflows

import (
	"reflect"
	"testing"
)

func TestWorkflowKeyForKind(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindExtract, ExtractWorkflowKey},
		{KindAdopt, AdoptWorkflowKey},
	}
	for _, tt := range tests {
		got, err := workflowKeyForKind(tt.kind)
		if err != nil || got != tt.want {
			t.Fatalf("workflowKeyForKind(%q) = %q, %v; want %q, nil", tt.kind, got, err, tt.want)
		}
	}
	if _, err := workflowKeyForKind("unexpected"); err == nil {
		t.Fatal("unknown workflow kind was accepted")
	}
}

func TestWorkflowInputMatchesEachDeclaredKindSchema(t *testing.T) {
	extract := workflowInput(StartInput{Kind: KindExtract, AssetID: "asset", SourceScenario: "source", SourcePath: "Card.tsx", TargetScenario: "must-not-send", ConfirmOverwrite: true})
	if want := map[string]any{"kind": "extract", "assetId": "asset", "sourceScenario": "source", "sourcePath": "Card.tsx", "requestedVersion": ""}; !reflect.DeepEqual(extract, want) {
		t.Fatalf("extract input = %#v, want %#v", extract, want)
	}
	adopt := workflowInput(StartInput{Kind: KindAdopt, AssetID: "asset", SourceScenario: "must-not-send", TargetScenario: "target", SourcePath: "target/Card.tsx", ConfirmOverwrite: true})
	if _, sent := adopt["sourceScenario"]; sent {
		t.Fatalf("adopt input leaked extract-only sourceScenario: %#v", adopt)
	}
	if adopt["targetScenario"] != "target" || adopt["confirmOverwrite"] != true {
		t.Fatalf("adopt input = %#v", adopt)
	}
}
