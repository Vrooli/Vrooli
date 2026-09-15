package catalogexperience

import (
	"testing"

	"react-component-library/internal/experience"
)

func TestMatchesCurrentStateRejectsRenamedSpecimens(t *testing.T) {
	snapshot := experience.Snapshot{
		States: []experience.State{{ID: "default", ExampleName: "ready"}},
	}
	if matchesCurrentState(snapshot, experience.Evidence{StateID: "default", ExampleName: "default"}) {
		t.Fatal("renamed specimen should not be accepted as current state evidence")
	}
	if !matchesCurrentState(snapshot, experience.Evidence{StateID: "default", ExampleName: "ready"}) {
		t.Fatal("current specimen should be accepted")
	}
}
