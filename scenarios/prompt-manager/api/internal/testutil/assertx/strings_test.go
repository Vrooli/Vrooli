package assertx

import "testing"

func TestContainsPassesWhenSubstringIsPresent(t *testing.T) {
	Contains(t, "reading world-scale config: invalid character", "world-scale config", "error contract")
}
