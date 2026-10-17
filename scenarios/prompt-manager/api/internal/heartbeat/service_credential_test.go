package heartbeat

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeTokenSource struct {
	token string
	err   error
}

func (f fakeTokenSource) Token(context.Context) (string, error) { return f.token, f.err }

// An unattended heartbeat with no human/run proof forwards Prompt Manager's
// service bearer, so Agent Manager sees a verified service identity.
func TestPrepareCreateRunCallerForwardsServiceBearer(t *testing.T) {
	c := &AgentManagerClient{serviceToken: fakeTokenSource{token: "svc-token"}}
	ctx, err := c.prepareCreateRunCaller(context.Background())
	if err != nil {
		t.Fatalf("prepare refused: %v", err)
	}
	proof := createRunCaller(ctx)
	if proof.authorization != "Bearer svc-token" {
		t.Fatalf("service bearer not forwarded as authorization: %q", proof.authorization)
	}
	if proof.runIdentity != "" {
		t.Fatalf("service path must not set a run identity: %q", proof.runIdentity)
	}
}

// A human owner proof already in context is never replaced by the service
// identity — attribution stays with the human.
func TestPrepareCreateRunCallerKeepsExistingHumanProof(t *testing.T) {
	c := &AgentManagerClient{serviceToken: fakeTokenSource{token: "svc-token"}}
	seeded := withCreateRunCaller(context.Background(), "Bearer human-token", "")
	ctx, err := c.prepareCreateRunCaller(seeded)
	if err != nil {
		t.Fatalf("prepare refused: %v", err)
	}
	if proof := createRunCaller(ctx); proof.authorization != "Bearer human-token" {
		t.Fatalf("human proof was overridden by the service identity: %q", proof.authorization)
	}
}

// When no service token can be minted (credential not yet issued), the service
// path does not inject a bearer; the caller proceeds on the prior unqualified
// path rather than presenting a bogus credential.
func TestPrepareCreateRunCallerFallsThroughWhenNoServiceToken(t *testing.T) {
	c := &AgentManagerClient{serviceToken: fakeTokenSource{err: errors.New("credential not issued")}}
	ctx, err := c.prepareCreateRunCaller(context.Background())
	if err != nil {
		t.Fatalf("prepare should not fail when the service token is unavailable: %v", err)
	}
	if proof := createRunCaller(ctx); proof.authorization != "" {
		t.Fatalf("a bogus bearer was injected without a usable token: %q", proof.authorization)
	}
}

func TestServiceTokenSourceCachesUntilRefreshMargin(t *testing.T) {
	now := time.Now()
	calls := 0
	src := newServiceTokenSource(
		func() (string, error) { return "cred", nil },
		func(context.Context, string) (string, time.Time, error) {
			calls++
			return "tok", now.Add(time.Hour), nil
		},
		func() time.Time { return now },
	)
	for i := 0; i < 3; i++ {
		if tok, err := src.Token(context.Background()); err != nil || tok != "tok" {
			t.Fatalf("token: %q %v", tok, err)
		}
	}
	if calls != 1 {
		t.Fatalf("expected a single exchange while cached, got %d", calls)
	}
}
