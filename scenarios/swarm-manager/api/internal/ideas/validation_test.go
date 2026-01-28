package ideas

import (
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestValidateUpdateIdeaRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *apipb.UpdateIdeaRequest
		wantErr string
	}{
		{
			name:    "missing title",
			req:     &apipb.UpdateIdeaRequest{Title: "", Status: "backlog", Priority: 3},
			wantErr: "title is required",
		},
		{
			name:    "invalid status",
			req:     &apipb.UpdateIdeaRequest{Title: "Idea", Status: "bad", Priority: 3},
			wantErr: "status must be a valid idea status",
		},
		{
			name:    "invalid priority",
			req:     &apipb.UpdateIdeaRequest{Title: "Idea", Status: "backlog", Priority: 11},
			wantErr: "priority must be between 1 and 10",
		},
		{
			name:    "valid",
			req:     &apipb.UpdateIdeaRequest{Title: "Idea", Status: "ready", Priority: 5},
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateUpdateIdeaRequest(tc.req)
			if got != tc.wantErr {
				t.Fatalf("expected %q, got %q", tc.wantErr, got)
			}
		})
	}
}
