package transitionrunner

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestSnapshotFromSubjectDerivesTokensFromCurrentProjection(t *testing.T) {
	input, err := structpb.NewValue(map[string]any{"name": "item"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := SnapshotFromSubject(input, map[string]any{"version": "stored-v1", "state": "open"}, map[string]any{"phase": 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := SnapshotFromSubject(input, map[string]any{"version": "stored-v2", "state": "open"}, map[string]any{"phase": 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.EntityVersion == second.EntityVersion {
		t.Fatal("entity token echoed a stored version instead of the current subject projection")
	}
	if first.EntityVersion == "stored-v1" || second.EntityVersion == "stored-v2" {
		t.Fatal("entity token echoed a stored version instead of deriving from current subject state")
	}
	if first.FrontierDigest != second.FrontierDigest {
		t.Fatal("frontier token changed when only the subject projection changed")
	}
}
