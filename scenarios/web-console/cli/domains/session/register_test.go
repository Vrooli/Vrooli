package session

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// idSchema declares only the session-id positional, matching the handlers that
// read it via ctx.Positional("session-id").
func idSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "session-id", Required: true, Description: "Session ID"},
		},
	}
}

// policySetSchema declares the session-id positional plus the flags policySet
// reads (mode, duration, body-file).
func policySetSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "session-id", Required: true, Description: "Session ID"},
		},
		Flags: []cliapp.Flag{
			{Name: "mode", Description: "Policy mode (required)"},
			{Name: "duration", Description: "Optional duration (e.g. 1h, 30m)"},
			{Name: "body-file", Description: "Path to a JSON body (overrides --mode/--duration)"},
		},
	}
}

// Validation tests build a handlers value with a nil *cliapp.ScenarioApp because
// the checks fire before any API client is used. If a validation path stops
// working that way, this test panics loudly instead of making a network call.
func TestValidation(t *testing.T) {
	h := &handlers{core: nil}

	newCtx := func(schema cliapp.ArgSchema, positionals, flags map[string]string) cliapp.RunContext {
		return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema:      schema,
			Positionals: positionals,
			Flags:       flags,
			Core:        nil,
		})
	}

	t.Run("get_requires_id", func(t *testing.T) {
		err := h.get(newCtx(idSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session get <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("delete_requires_id", func(t *testing.T) {
		err := h.delete(newCtx(idSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session delete <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_get_requires_id", func(t *testing.T) {
		err := h.policyGet(newCtx(idSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session policy-get <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_set_requires_id", func(t *testing.T) {
		err := h.policySet(newCtx(policySetSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session policy-set <session-id> --mode <mode> [--duration <dur>]" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("policy_set_requires_mode_when_no_body", func(t *testing.T) {
		err := h.policySet(newCtx(policySetSchema(), map[string]string{"session-id": "sess-1"}, nil))
		if err == nil || err.Error() != "--mode is required (or provide --body-file)" {
			t.Fatalf("expected missing mode error, got %v", err)
		}
	})

	t.Run("recover_requires_id", func(t *testing.T) {
		err := h.recover(newCtx(idSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session recover <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("dismiss_requires_id", func(t *testing.T) {
		err := h.dismiss(newCtx(idSchema(), nil, nil))
		if err == nil || err.Error() != "usage: session dismiss <session-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})
}
