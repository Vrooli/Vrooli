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
	manifest, _, err := dispatch.BuildManifest()
	require.NoError(t, err)

	cases := []struct {
		name    string
		job     dispatch.Job
		scopes  []string
		wantErr any // nil, or a typed sentinel to errors.As against
	}{
		{
			name:    "allowlisted verb in scope is accepted",
			job:     dispatch.Job{Verb: "scenario test", Scenario: "web-search", Args: []string{"--json"}},
			scopes:  []string{"vrooli-bridge:write", "vrooli:write"},
			wantErr: nil,
		},
		{
			name:    "verb absent from the manifest is rejected as not-in-manifest",
			job:     dispatch.Job{Verb: "scenario deploy", Scenario: "web-search"},
			scopes:  []string{"*"}, // even with universal authority
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
			scopes:  []string{"vrooli-bridge:write", "web-console:write"},
			wantErr: dispatch.ErrVerbOutOfScope{},
		},
		{
			name:    "cataloged destructive verb is out-of-scope with personal grants",
			job:     dispatch.Job{Verb: "device-control device forget"},
			scopes:  []string{"vrooli-bridge:read", "vrooli-bridge:write", "device-control:read", "device-control:write"},
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
			scopes:  []string{"vrooli-bridge:write", "vrooli:write"},
			wantErr: dispatch.ErrUnsafeToken{},
		},
		{
			name:    "shell metacharacter in scenario is rejected as unsafe",
			job:     dispatch.Job{Verb: "scenario test", Scenario: "$(whoami)"},
			scopes:  []string{"vrooli-bridge:write", "vrooli:write"},
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
			scopes:  []string{"vrooli-bridge:read", "vrooli:read"},
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

// [REQ:BRG-P0-004] Namespace grants use the shared catalog grammar and retain
// a separate Bridge transport-level effect capability.
func TestAllow_NamespaceGrantGrammar(t *testing.T) {
	manifest, _, err := dispatch.BuildManifest()
	require.NoError(t, err)
	tests := []struct {
		name   string
		verb   string
		scopes []string
		want   bool
	}{
		{name: "exact", verb: "scenario status", scopes: []string{"vrooli-bridge:read", "vrooli:read"}, want: true},
		{name: "namespace wildcard", verb: "scenario status", scopes: []string{"vrooli-bridge:read", "vrooli:*"}, want: true},
		{name: "effect wildcard", verb: "scenario status", scopes: []string{"vrooli-bridge:read", "*:read"}, want: true},
		{name: "universal", verb: "scenario status", scopes: []string{"*"}, want: true},
		{name: "unrelated namespace", verb: "scenario status", scopes: []string{"vrooli-bridge:read", "web-console:read"}},
		{name: "higher effect is not implied", verb: "device-control device forget", scopes: []string{"vrooli-bridge:write", "device-control:write"}},
		{name: "leading whitespace is malformed", verb: "scenario status", scopes: []string{"vrooli-bridge:read", " vrooli:read"}},
		{name: "transport capability remains required", verb: "scenario status", scopes: []string{"vrooli:read"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dispatch.Allow(dispatch.Job{Verb: test.verb}, test.scopes, manifest)
			if test.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAllow_NamespaceGrantCoversNewCatalogCommandWithoutGrantEdit(t *testing.T) {
	const separator = "\x00"
	grants := []string{"vrooli-bridge:read", "example:read"}
	before := []string{"example status" + separator + "example:read" + separator + "vrooli-bridge:read"}
	after := append(before, "example inspect"+separator+"example:read"+separator+"vrooli-bridge:read")

	require.NoError(t, dispatch.Allow(dispatch.Job{Verb: "example status"}, grants, before))
	require.NoError(t, dispatch.Allow(dispatch.Job{Verb: "example inspect"}, grants, after), "the manifest changed; the node grant did not")
}
