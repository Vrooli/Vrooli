package runner

import (
	"context"
	"testing"
)

// Caller-supplied parent identity is data, never a protected serial receipt.
// The positive native owner/fake-driver matrix lives in nativeisolation;
// these adapter cases qualify refusal and absence of the host fallback only.
func TestFiniteSerialCallerCannotSubstituteEditableParentForOwnedHandoff(t *testing.T) {
	for _, name := range []string{"parent-only", "root-with-parent", "source-recovery", "missing-key"} {
		t.Run(name, func(t *testing.T) {
			m, cfg, id, req := finiteLaunchFixture()
			intent := cfg.Admission.EffortIntent
			intent.Effect = "run.child"
			intent.ParentRunID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
			intent.IdempotencyKey = "exact-looking-unproven-key"
			switch name {
			case "root-with-parent":
				intent.Effect = "run.create"
			case "source-recovery":
				intent.SourceRunID = intent.ParentRunID
			case "missing-key":
				intent.IdempotencyKey = ""
			}
			factory, err := NewFiniteNativeFactory(m)
			if err != nil {
				t.Fatal(err)
			}
			selector := NewLauncherSelector(nil, nil)
			selector.SetFiniteNativeFactory(factory)
			launcher := selector.PickFor(context.Background(), id, cfg, nil, nil)
			if _, ok := launcher.(*HostLauncher); ok {
				t.Fatal("serial refusal fell back to host")
			}
			if _, err = launcher.Launch(context.Background(), req); err == nil {
				t.Fatal("editable parent launched without protected owner")
			}
		})
	}
}
