package assertx

import "testing"

func TestContainsStringPassesWhenValueExists(t *testing.T) {
	ContainsString(t, []string{"main", "feature/test"}, "feature/test")
}
