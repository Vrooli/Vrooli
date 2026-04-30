package domains

import "testing"

func TestCommandGroupsRegistersExpectedDomainGroups(t *testing.T) {
	groups := CommandGroups(nil)
	if len(groups) != 12 {
		t.Fatalf("expected 12 command groups, got %d", len(groups))
	}
	if groups[0].Title != "Skills" {
		t.Fatalf("skills should be first command group, got %q", groups[0].Title)
	}
}
