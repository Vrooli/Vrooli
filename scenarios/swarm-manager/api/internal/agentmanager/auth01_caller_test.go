package agentmanager

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
)

type auth01FixtureVerifier struct{}

func (auth01FixtureVerifier) Validate(_ context.Context, _ string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func auth01FixtureCaller(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	out, err := owneridentity.AuthorizeCreateRunCaller(ctx, http.Header{"Authorization": []string{"Bearer fixture-owner"}}, auth01FixtureVerifier{}, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAuth01GoalRunWithoutCallerRejectsBeforeTaskEffect(t *testing.T) {
	s := &AgentService{enabled: true}
	_, err := s.CreateGoalRun(context.Background(), GoalRunRequest{Prompt: "fixture", Until: "fixture"})
	if err != owneridentity.ErrCreateRunCaller {
		t.Fatal("unqualified queued caller reached task/profile effects", err)
	}
}
