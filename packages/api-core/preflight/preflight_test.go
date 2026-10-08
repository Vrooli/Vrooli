package preflight

import "testing"

func TestRunAllDisabled(t *testing.T) {
	if Run(Config{DisableStaleness: true, DisableLifecycleGuard: true}) {
		t.Fatal("disabled preflight must not report a re-exec")
	}
}
