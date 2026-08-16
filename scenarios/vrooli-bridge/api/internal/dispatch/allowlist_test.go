package dispatch_test

import (
	"errors"
	"testing"

	"vrooli-bridge/internal/dispatch"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P0-004] The allowlist gate is the highest-stakes decision in the
// scenario. A job is accepted only when its verb is a recognised manifest verb
// AND covered by the node's granted scopes AND carries no shell metacharacter;
// every other case is a typed rejection naming a distinct reason.
func TestAllow_Matrix(t *testing.T) {
	manifest := dispatch.DefaultManifest

	cases := []struct {
		name    string
		job     dispatch.Job
		scopes  []string
		wantErr any // nil, or a typed sentinel to errors.As against
	}{
		{
			name:    "allowlisted verb in scope is accepted",
			job:     dispatch.Job{Verb: "scenario test", Scenario: "web-search", Args: []string{"--json"}},
			scopes:  []string{"scenario test*"},
			wantErr: nil,
		},
		{
			name:    "verb absent from the manifest is rejected as not-in-manifest",
			job:     dispatch.Job{Verb: "scenario deploy", Scenario: "web-search"},
			scopes:  []string{"scenario deploy*"}, // even with a matching scope
			wantErr: dispatch.ErrVerbNotInManifest{},
		},
		{
			name:    "secrets verb is never dispatchable (not in manifest)",
			job:     dispatch.Job{Verb: "secrets list"},
			scopes:  []string{"*"},
			wantErr: dispatch.ErrVerbNotInManifest{},
		},
		{
			name:    "manifest verb outside the node's scopes is rejected as out-of-scope",
			job:     dispatch.Job{Verb: "scenario test", Scenario: "web-search"},
			scopes:  []string{"scenario status*"},
			wantErr: dispatch.ErrVerbOutOfScope{},
		},
		{
			name:    "cataloged destructive verb is out-of-scope with personal grants",
			job:     dispatch.Job{Verb: "scenario stop-all"},
			scopes:  []string{"vrooli-bridge:read", "vrooli-bridge:write"},
			wantErr: dispatch.ErrVerbOutOfScope{},
		},
		{
			name:    "a node with no scopes can run nothing",
			job:     dispatch.Job{Verb: "scenario test"},
			scopes:  nil,
			wantErr: dispatch.ErrVerbOutOfScope{},
		},
		{
			name:    "empty verb is an invalid job",
			job:     dispatch.Job{Verb: "   "},
			scopes:  []string{"*"},
			wantErr: dispatch.ErrInvalidJob{},
		},
		{
			name:    "shell metacharacter in args is rejected as unsafe (no-shell defence)",
			job:     dispatch.Job{Verb: "scenario test", Args: []string{"web-search; rm -rf /"}},
			scopes:  []string{"scenario test*"},
			wantErr: dispatch.ErrUnsafeToken{},
		},
		{
			name:    "shell metacharacter in scenario is rejected as unsafe",
			job:     dispatch.Job{Verb: "scenario test", Scenario: "$(whoami)"},
			scopes:  []string{"scenario test*"},
			wantErr: dispatch.ErrUnsafeToken{},
		},
		{
			name:    "backtick in verb is rejected as unsafe",
			job:     dispatch.Job{Verb: "scenario `id`"},
			scopes:  []string{"*"},
			wantErr: dispatch.ErrUnsafeToken{},
		},
		{
			name:    "universal scope covers an allowlisted verb",
			job:     dispatch.Job{Verb: "scenario logs", Scenario: "web-search"},
			scopes:  []string{"*"},
			wantErr: nil,
		},
		{
			name:    "exact scope (no wildcard) matches exactly",
			job:     dispatch.Job{Verb: "scenario status", Scenario: "web-search"},
			scopes:  []string{"scenario status"},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := dispatch.Allow(tc.job, tc.scopes, manifest)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			// errors.As against the expected typed sentinel.
			switch tc.wantErr.(type) {
			case dispatch.ErrVerbNotInManifest:
				var e dispatch.ErrVerbNotInManifest
				require.True(t, errors.As(err, &e), "got %T: %v", err, err)
			case dispatch.ErrVerbOutOfScope:
				var e dispatch.ErrVerbOutOfScope
				require.True(t, errors.As(err, &e), "got %T: %v", err, err)
			case dispatch.ErrInvalidJob:
				var e dispatch.ErrInvalidJob
				require.True(t, errors.As(err, &e), "got %T: %v", err, err)
			case dispatch.ErrUnsafeToken:
				var e dispatch.ErrUnsafeToken
				require.True(t, errors.As(err, &e), "got %T: %v", err, err)
			default:
				t.Fatalf("unhandled expected error type %T", tc.wantErr)
			}
		})
	}
}

// [REQ:BRG-P0-004] The scope glob grammar matches exactly, by trailing-* prefix,
// or universally — and never matches across the namespace boundary.
func TestAllow_ScopeGlobBoundary(t *testing.T) {
	manifest := []string{"scenario test"}
	// "scenario tes*" must NOT be tricked; but it IS a prefix of "scenario test"
	// so it legitimately matches. Prove a non-prefix scope does not match.
	require.Error(t, dispatch.Allow(
		dispatch.Job{Verb: "scenario test"},
		[]string{"scenario build*"}, manifest,
	))
	// A trailing-* prefix that does match.
	require.NoError(t, dispatch.Allow(
		dispatch.Job{Verb: "scenario test"},
		[]string{"scenario t*"}, manifest,
	))
}
