package credentialauthority

import "testing"

func TestRecoveryEpochRejectsOldCapabilitiesAfterAdvance(t *testing.T) {
	epoch := NewRecoveryEpoch(0)
	old := epoch.Current()
	if old != 1 || !epoch.Accept(old) {
		t.Fatalf("fresh epoch = %d, accepts old=%v", old, epoch.Accept(old))
	}
	current := epoch.Advance()
	if current != 2 {
		t.Fatalf("advanced epoch = %d, want 2", current)
	}
	if epoch.Accept(old) || !epoch.Accept(current) || epoch.Accept(0) {
		t.Fatalf("epoch acceptance after advance is incorrect")
	}
}
