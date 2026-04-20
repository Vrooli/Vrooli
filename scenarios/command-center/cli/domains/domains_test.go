package domains

import "testing"

func TestCommandGroupsReturnsNilForScaffold(t *testing.T) {
	// The scaffold registers no flat command groups; sibling execute items add
	// domain packages later. Guarding the contract keeps the registry honest.
	if got := CommandGroups(nil); got != nil {
		t.Fatalf("CommandGroups(nil) = %v, want nil", got)
	}
}

func TestSubcommandGroupsReturnsNilForScaffold(t *testing.T) {
	if got := SubcommandGroups(nil); got != nil {
		t.Fatalf("SubcommandGroups(nil) = %v, want nil", got)
	}
}
