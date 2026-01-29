package backlog

import (
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestValidateUpdateBacklogItemRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *apipb.UpdateBacklogItemRequest
		wantErr string
	}{
		{
			name:    "missing title",
			req:     &apipb.UpdateBacklogItemRequest{Title: "", Status: "backlog", Priority: 3},
			wantErr: "title is required",
		},
		{
			name:    "invalid status",
			req:     &apipb.UpdateBacklogItemRequest{Title: "Backlog", Status: "bad", Priority: 3},
			wantErr: "status must be a valid backlog status",
		},
		{
			name:    "invalid priority",
			req:     &apipb.UpdateBacklogItemRequest{Title: "Backlog", Status: "backlog", Priority: 11},
			wantErr: "priority must be between 1 and 10",
		},
		{
			name:    "valid",
			req:     &apipb.UpdateBacklogItemRequest{Title: "Backlog", Status: "ready", Priority: 5},
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateUpdateBacklogItemRequest(tc.req)
			if got != tc.wantErr {
				t.Fatalf("expected %q, got %q", tc.wantErr, got)
			}
		})
	}
}
