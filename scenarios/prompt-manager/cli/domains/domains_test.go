package domains

import "testing"

// The registered groups are asserted by name and order rather than by count.
// A bare count told an author only that a number had changed, never which
// group appeared or vanished, so registering a new group failed here with no
// way to see what it was.
func TestCommandGroupsRegistersExpectedDomainGroups(t *testing.T) {
	want := []string{
		"Skills",
		"Goals",
		"Actions",
		"Experiments",
		"Tags",
		"Members",
		"Agents",
		"Teams",
		"Topics",
		"Testing",
		"Metadata",
		"Search",
		"Discovery",
		"Graph",
		"Coverage Space",
	}

	groups := CommandGroups(nil)
	got := make([]string, 0, len(groups))
	for _, group := range groups {
		got = append(got, group.Title)
	}

	if len(got) != len(want) {
		t.Fatalf("command groups = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("command group %d = %q, want %q", i, got[i], want[i])
		}
	}
}
