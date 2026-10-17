package operationstanding

import (
	"bytes"
	"testing"
)

type testStanding struct {
	lifecycle string
	directive string
	eta       int32
	etaKnown  bool
	reattach  string
}

func (s testStanding) GetLifecycle() string                { return s.lifecycle }
func (s testStanding) GetActivePhase() string              { return "" }
func (s testStanding) GetEtaKnown() bool                   { return s.etaKnown }
func (s testStanding) GetEstimatedRemainingSeconds() int32 { return s.eta }
func (s testStanding) GetDirective() string                { return s.directive }
func (s testStanding) GetReattachCommand() string          { return s.reattach }

func TestWriteTextRendersOwnerStandingWithoutInference(t *testing.T) {
	var out bytes.Buffer
	err := WriteText(&out, testStanding{lifecycle: "preparing", directive: "wait", eta: 45, etaKnown: true, reattach: "test-genie runs wait --json demo run-1"})
	if err != nil {
		t.Fatalf("WriteText: %v", err)
	}
	got := out.String()
	for _, want := range []string{"lifecycle: preparing", "estimated remaining: ~45s", "action: wait", "reattach once: test-genie"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}
