package agentinstall

import (
	"errors"
	harnesscatalog "github.com/vrooli/cli-core/agentcatalog/harness"
	"testing"
)

func TestNativeSignInClassifiesOnlyDeclaredEvidence(t *testing.T) {
	probe := &harnesscatalog.SignInProbe{SignedInPrefixes: []string{"Logged in using "}, SignedOutLines: []string{"Not logged in"}}
	for _, tc := range []struct {
		output string
		err    error
		want   string
	}{
		{"Not logged in\n", errors.New("exit 1"), "signed_out"},
		{"Logged in using ChatGPT\n", nil, "signed_in"},
		{"Logged in using API key: sensitive-value\n", nil, "signed_in"},
		{"Not logged in\nLogged in using ChatGPT", nil, "unknown"},
		{"Logged in using ChatGPT", errors.New("failed"), "unknown"},
		{"network unavailable", errors.New("exit 1"), "unknown"},
		{"", nil, "unknown"},
	} {
		if got := classifySignIn(tc.output, tc.err, probe); got != tc.want {
			t.Errorf("state=%s want %s", got, tc.want)
		}
	}
}
