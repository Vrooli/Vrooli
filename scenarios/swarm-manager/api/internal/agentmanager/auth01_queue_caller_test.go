package agentmanager

import (
	"context"
	"github.com/vrooli/api-core/owneridentity"
	"net/http"
	"testing"
	"time"
)

type auth01QueueCallerVerifier struct{}

func (auth01QueueCallerVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "human", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func TestAuth01ActualGoalQueueCallerQualification(t *testing.T) {
	service := &AgentService{createCallerVerifier: auth01QueueCallerVerifier{}}
	for _, headers := range []http.Header{{}, {"Authorization": []string{"Bearer one", "Bearer two"}}, {"Authorization": []string{"Bearer one"}, "X-Agent-Identity-Token": []string{""}}} {
		if _, err := service.PrepareCreateRunCaller(owneridentity.WithCreateRunHeaders(context.Background(), headers)); err == nil {
			t.Fatal("invalid offered proof accepted")
		}
	}
	ctx, err := service.PrepareCreateRunCaller(owneridentity.WithCreateRunHeaders(context.Background(), http.Header{"Authorization": []string{"Bearer original"}}))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := owneridentity.CreateRunAuthorization(ctx, time.Now())
	if err != nil || proof != "Bearer original" {
		t.Fatal("original proof lost", err)
	}
}
