package scenariocli

import "testing"

func TestDemandHistoryRequiresScenarioAndBoundedLimit(t *testing.T) {
	for _, args := range [][]string{{"history"}, {"history", "--scenario", "demo", "--limit", "0"}, {"history", "--scenario", "demo", "--limit", "1001"}, {"history", "--scenario", "demo", "--limit", "NaN"}, {"acquire", "--limit", "2"}} {
		if _, err := ParseDemandRequest(false, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	req, err := ParseDemandRequest(false, []string{"history", "--scenario", "demo", "--variant", "shadow", "--limit", "3", "--json"})
	if err != nil || req.Limit != 3 || req.Variant != "shadow" || !req.JSON {
		t.Fatalf("history=%+v err=%v", req, err)
	}
}
