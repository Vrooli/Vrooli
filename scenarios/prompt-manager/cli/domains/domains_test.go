package domains

import "testing"

func TestCommandGroupsRegistersExpectedDomainGroups(t *testing.T) {
	groups := CommandGroups(nil)
	if len(groups) != 13 {
		t.Fatalf("expected 13 command groups, got %d", len(groups))
	}
	if groups[0].Title != "Skills" {
		t.Fatalf("skills should be first command group, got %q", groups[0].Title)
	}
	if groups[1].Title != "Actions" {
		t.Fatalf("actions should be second command group, got %q", groups[1].Title)
	}
}
