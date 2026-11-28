package rehearsals

import "testing"

func TestProbeNormalizePublic2385(t *testing.T) {
	_, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/2385.json"))
	t.Logf("normalize: %v", err)
}
