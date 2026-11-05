package protoconv

import (
	"testing"

	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestReplayPackageContractDecodesStrictly(t *testing.T) {
	fixture := []byte(`{
		"id":"2d8da0e4-070f-4ed9-a7d2-a5aa1c5d20f5",
		"schemaVersion":"bas-replay/v1",
		"executionId":"d72eb0a2-1eec-4cff-9afc-6f3cf53fb2b1",
		"evidence":{
			"id":"18458d69-0151-4d04-91e4-1e308a5f7a2a",
			"executionId":"d72eb0a2-1eec-4cff-9afc-6f3cf53fb2b1",
			"schemaVersion":"bas-evidence/v1",
			"artifacts":[{
				"id":"f11a7f2d-2101-4193-a65b-9f2b79b8f8dc",
				"kind":"ARTIFACT_KIND_SCREENSHOT",
				"mediaType":"image/png",
				"sizeBytes":"42",
				"sha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				"classification":"CONTENT_CLASSIFICATION_INTERNAL",
				"retentionClass":"RETENTION_CLASS_STANDARD",
				"accessPolicy":"ACCESS_POLICY_PROJECT_MEMBERS",
				"executionId":"d72eb0a2-1eec-4cff-9afc-6f3cf53fb2b1",
				"producer":"playwright-driver"
			}]
		}
	}`)

	var replay basevidence.ReplayPackage
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(fixture, &replay); err != nil {
		t.Fatalf("strictly decode replay package fixture: %v", err)
	}
	if got := replay.Evidence.Artifacts[0].Kind; got != basevidence.ArtifactKind_ARTIFACT_KIND_SCREENSHOT {
		t.Fatalf("artifact kind = %v, want screenshot", got)
	}
}

func TestReplayPackageContractRejectsUnknownFields(t *testing.T) {
	var replay basevidence.ReplayPackage
	err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(`{"schemaVersion":"bas-replay/v1","unknown":true}`), &replay)
	if err == nil {
		t.Fatal("expected unknown replay package field to be rejected")
	}
}
