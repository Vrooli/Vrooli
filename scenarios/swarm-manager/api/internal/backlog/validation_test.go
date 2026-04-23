package backlog

import (
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func stringPtr(value string) *string {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func TestValidateUpdateBacklogItemRequest(t *testing.T) {
	tests := []struct {
		name    string
		kind    BacklogKind
		req     *apipb.UpdateBacklogItemRequest
		fields  backlogUpdateFieldSet
		wantErr string
	}{
		{
			name:    "empty patch",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{},
			fields:  backlogUpdateFieldSet{},
			wantErr: "at least one field is required",
		},
		{
			name:    "missing title",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{Title: stringPtr("")},
			fields:  backlogUpdateFieldSet{updateFieldTitle: {}},
			wantErr: "title is required",
		},
		{
			name:    "invalid status",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{Status: stringPtr("bad")},
			fields:  backlogUpdateFieldSet{updateFieldStatus: {}},
			wantErr: "status must be a valid backlog status",
		},
		{
			name:    "execution only status rejected",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{Status: stringPtr("queued")},
			fields:  backlogUpdateFieldSet{updateFieldStatus: {}},
			wantErr: "status 'queued' and 'in_progress' can only be set by the execution system",
		},
		{
			name:    "invalid priority",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{Priority: int32Ptr(11)},
			fields:  backlogUpdateFieldSet{updateFieldPriority: {}},
			wantErr: "priority must be between 1 and 10",
		},
		{
			name:    "duplicate tags rejected",
			kind:    KindIdea,
			req:     &apipb.UpdateBacklogItemRequest{Tags: []string{"dup", "dup"}},
			fields:  backlogUpdateFieldSet{updateFieldTags: {}},
			wantErr: `tags[1]: duplicate value "dup"`,
		},
		{
			name:    "valid partial update",
			kind:    KindResearch,
			req:     &apipb.UpdateBacklogItemRequest{Status: stringPtr("ready")},
			fields:  backlogUpdateFieldSet{updateFieldStatus: {}},
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateUpdateBacklogItemRequest(tc.req, tc.fields, tc.kind, StatusBacklog)
			if got != tc.wantErr {
				t.Fatalf("expected %q, got %q", tc.wantErr, got)
			}
		})
	}
}

func TestParseUpdateBacklogFields(t *testing.T) {
	fields, err := parseUpdateBacklogFields([]byte(`{"status":"ready","acceptanceAllow":[]}`))
	if err != nil {
		t.Fatalf("parseUpdateBacklogFields returned error: %v", err)
	}

	if !fields.Has(updateFieldStatus) {
		t.Fatal("expected status to be marked present")
	}
	if !fields.Has(updateFieldAcceptanceAllow) {
		t.Fatal("expected acceptance_allow to be marked present from camelCase alias")
	}
}

func TestParseUpdateBacklogFieldsRejectsNull(t *testing.T) {
	_, err := parseUpdateBacklogFields([]byte(`{"description":null}`))
	if err == nil {
		t.Fatal("expected parseUpdateBacklogFields to reject null values")
	}
	if !strings.Contains(err.Error(), "description cannot be null") {
		t.Fatalf("unexpected error: %v", err)
	}
}
