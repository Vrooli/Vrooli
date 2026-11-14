package dispatch

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/owneridentity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type auth01OwnerVerifier func(context.Context, string) (owneridentity.Identity, error)

func (f auth01OwnerVerifier) Validate(ctx context.Context, token string) (owneridentity.Identity, error) {
	return f(ctx, token)
}

type fixtureSwitcherOwner struct{}

func (fixtureSwitcherOwner) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if token != "fixture-owner" && token != "owner-token" {
		return owneridentity.Identity{}, errors.New("bad fixture")
	}
	return owneridentity.Identity{Subject: "fixture-owner", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func TestAuth01SwitchboardRefusesInvalidOwnerBeforeTaskAndRunEffects(t *testing.T) {
	for _, variant := range []string{"absent", "wrong-channel", "unavailable", "invalid", "expired", "empty-subject", "no-write", "offered-run"} {
		t.Run(variant, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			defer srv.Close()
			owner := owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
			verify := auth01OwnerVerifier(func(_ context.Context, token string) (owneridentity.Identity, error) {
				if variant == "invalid" || token != "fixture-owner" {
					return owneridentity.Identity{}, errors.New("fixture-owner")
				}
				return owner, nil
			})
			auth, runProof := "Bearer fixture-owner", ""
			switch variant {
			case "absent":
				auth = ""
			case "wrong-channel":
				auth = "fixture-run"
			case "expired":
				owner.ExpiresAt = time.Now().Add(-time.Hour)
			case "empty-subject":
				owner.Subject = ""
			case "no-write":
				owner.Scopes = []string{"agent-manager:read"}
			case "offered-run":
				runProof = "fixture-run"
			}
			runner := AgentManagerRunner{BaseURL: srv.URL, Identity: verify}
			if variant == "unavailable" {
				runner.Identity = nil
			}
			_, err := runner.Run(WithCreateRunCaller(context.Background(), auth, runProof), "fixture-profile", nil, "hello")
			if err == nil || calls != 0 || strings.Contains(err.Error(), "Bearer fixture-owner") {
				t.Fatalf("invalid proof caused effects or leaked: %v calls%d", err, calls)
			}
		})
	}
}
func TestAuth01SwitchboardDoesNotRedirectOrLeakExistingOwnerProof(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "/other")
		w.WriteHeader(http.StatusTemporaryRedirect)
		_, _ = w.Write([]byte("fixture-owner"))
	}))
	defer srv.Close()
	_, err := (AgentManagerRunner{BaseURL: srv.URL, Identity: fixtureSwitcherOwner{}}).Run(WithAuthorization(context.Background(), "Bearer fixture-owner"), "fixture-profile", nil, "hello")
	if err == nil || calls != 1 || strings.Contains(err.Error(), "fixture-owner") {
		t.Fatalf("redirect or proof leak: %v calls%d", err, calls)
	}
}

func TestAuth01AmbiguousIngressRejectsBeforeTaskRun(t *testing.T) {
	runner := AgentManagerRunner{}
	for _, headers := range []http.Header{{"Authorization": []string{"Bearer fixture", ""}}, {"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{""}}} {
		_, _, err := runner.start(WithCreateRunHeaders(context.Background(), headers), "fixture", nil, "fixture")
		if err == nil {
			t.Fatal("ambiguous/blank offered channel ignored")
		}
	}
}
