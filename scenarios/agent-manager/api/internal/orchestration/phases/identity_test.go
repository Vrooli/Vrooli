package phases

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
)

func TestGenerateIdentityToken_SignsWorkflowMetadata(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New()}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"),
		Meta: map[string]string{"workflowExecutionId": "execution-1", "workflowNodeId": "node-a", "workflowAttemptId": "attempt-1"},
	})
	if token == "" {
		t.Fatal("expected token")
	}
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Meta["workflowExecutionId"] != "execution-1" || claims.Meta["workflowNodeId"] != "node-a" || claims.Meta["workflowAttemptId"] != "attempt-1" {
		t.Fatalf("claims metadata = %#v", claims.Meta)
	}
}
