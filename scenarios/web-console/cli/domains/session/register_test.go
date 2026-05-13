package session

import "testing"

// Validation tests pass a nil *cliapp.ScenarioApp because the checks
// fire before any API client is constructed. If a validation path
// stops working that way, this test will panic loudly instead of
// silently making a network call.
func TestValidation(t *testing.T) {
	t.Run("get_requires_id", func(t *testing.T) {
		err := runGet(nil, []string{})
		if err == nil || err.Error() != "usage: session get <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("delete_requires_id", func(t *testing.T) {
		err := runDelete(nil, []string{})
		if err == nil || err.Error() != "usage: session delete <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_get_requires_id", func(t *testing.T) {
		err := runPolicyGet(nil, []string{})
		if err == nil || err.Error() != "usage: session policy-get <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_set_requires_id", func(t *testing.T) {
		err := runPolicySet(nil, []string{})
		if err == nil || err.Error() != "usage: session policy-set <session-id> --mode <mode> [--duration <dur>]" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_set_requires_mode_when_no_body", func(t *testing.T) {
		err := runPolicySet(nil, []string{"sess-1"})
		if err == nil || err.Error() != "--mode is required (or provide --body-file)" {
			t.Fatalf("expected missing mode error, got %v", err)
		}
	})

	t.Run("recover_requires_id", func(t *testing.T) {
		err := runRecover(nil, []string{})
		if err == nil || err.Error() != "usage: session recover <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("dismiss_requires_id", func(t *testing.T) {
		err := runDismiss(nil, []string{})
		if err == nil || err.Error() != "usage: session dismiss <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})
}
