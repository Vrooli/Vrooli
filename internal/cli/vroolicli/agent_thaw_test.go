package vroolicli

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

type fakeRunner struct{ out string }

func (f fakeRunner) LookPath(name string) (string, error) { return name, nil }
func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) {
	if f.out == "" {
		return nil, errors.New("no unit")
	}
	return []byte(f.out + "\n"), nil
}

// [REQ:STORM-002] `vrooli agent thaw` resolves a unit name through the user
// manager, accepts a cgroup ref as written, and refuses anything that is not
// an agent session scope before platform-go is even asked.
func TestAgentThawResolvesOnlyAgentScopes(t *testing.T) {
	ref, err := agentScopeRef(context.Background(), "vrooli-agent-abc", fakeRunner{out: "/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-abc.scope"})
	if err != nil || ref.Kind != platformgo.ScopeKindCgroup || !strings.HasSuffix(ref.Path, "vrooli-agent-abc.scope") || ref.Name != "vrooli-agent-abc" {
		t.Fatalf("ref = %+v, err = %v", ref, err)
	}
	if _, err := agentScopeRef(context.Background(), "vrooli-runtime-supervisor", fakeRunner{out: "/x"}); err == nil || !strings.Contains(err.Error(), "not an agent session scope") {
		t.Fatalf("supervisor accepted: %v", err)
	}
	if _, err := agentScopeRef(context.Background(), "vrooli-agent-gone", fakeRunner{out: ""}); err == nil {
		t.Fatal("a scope with no control group must be an error")
	}
	ref, err = agentScopeRef(context.Background(), "cgroup:/user.slice/vrooli-agents.slice/vrooli-agent-z.scope", fakeRunner{})
	if err != nil || ref.Path != "/user.slice/vrooli-agents.slice/vrooli-agent-z.scope" {
		t.Fatalf("cgroup ref = %+v, err = %v", ref, err)
	}
	if err := platformgo.ThawScope(platformgo.ScopeRef{Kind: platformgo.ScopeKindCgroup, Path: "/user.slice/app.slice/vrooli-runtime-supervisor.service"}); err == nil {
		t.Fatal("platform-go must refuse to thaw a supervisor")
	}
}
