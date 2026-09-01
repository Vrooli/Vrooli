package domains

import "testing"

func TestCommandGroupsRegisterFlatReadVerbs(t *testing.T) {
	got := CommandGroups(nil)
	if len(got) != 1 || len(got[0].Commands) != 6 { t.Fatalf("unexpected flat command registration: %#v", got) }
}

func TestSubcommandGroupsRemainEmpty(t *testing.T) {
	if got := SubcommandGroups(nil); got != nil { t.Fatalf("unexpected hierarchical groups: %#v", got) }
}
