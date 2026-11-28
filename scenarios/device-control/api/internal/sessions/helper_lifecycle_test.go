package sessions

import "testing"

func TestHelperLifecycleTransitionMatrix(t *testing.T) {
	states := []HelperLifecycleStatus{"", HelperLifecycleRunning, HelperLifecycleStopping, HelperLifecycleControllerCleanupCompleted, HelperLifecycleUncertain, "unknown"}
	allowed := map[HelperLifecycleStatus]map[HelperLifecycleStatus]bool{
		"":                      {HelperLifecycleRunning: true},
		HelperLifecycleRunning:  {HelperLifecycleStopping: true},
		HelperLifecycleStopping: {HelperLifecycleStopping: true, HelperLifecycleControllerCleanupCompleted: true, HelperLifecycleUncertain: true},
		HelperLifecycleControllerCleanupCompleted: {},
		HelperLifecycleUncertain:                  {},
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				_, ok := from.Transition(to)
				if ok != allowed[from][to] {
					t.Fatalf("transition %q -> %q allowed = %t, want %t", from, to, ok, allowed[from][to])
				}
			})
		}
	}
}

func TestHelperLifecycleRejectsUnknownStatus(t *testing.T) {
	if HelperLifecycleStatus("done").Valid() {
		t.Fatal("unknown lifecycle status was valid")
	}
}
