package registry

import (
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

func TestEmbeddingMigrationSubmissionStateNeverClaimsShadowReadyOnAcceptance(t *testing.T) {
	if state, err := embeddingMigrationSubmissionState("shadow", false, &controlv1.ReindexResponse{JobId: "job-1"}); err != nil || state != "building" {
		t.Fatalf("submitted shadow state=%q err=%v, want building", state, err)
	}
	if state, err := embeddingMigrationSubmissionState("shadow", true, &controlv1.ReindexResponse{PlannedUpserts: 3}); err != nil || state != "planned" {
		t.Fatalf("shadow dry-run state=%q err=%v, want planned", state, err)
	}
	if _, err := embeddingMigrationSubmissionState("shadow", false, &controlv1.ReindexResponse{}); err == nil {
		t.Fatal("provider response without a job ID was treated as a submitted shadow build")
	}
	if state, err := embeddingMigrationSubmissionState("cutover", false, &controlv1.ReindexResponse{}); err != nil || state != "cutover" {
		t.Fatalf("cutover state=%q err=%v", state, err)
	}
	if state, err := embeddingMigrationSubmissionState("rollback", false, &controlv1.ReindexResponse{}); err != nil || state != "rolled_back" {
		t.Fatalf("rollback state=%q err=%v", state, err)
	}
}

func TestNegotiatedMigrationCapabilityRequiresMatchingVersionAndAction(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *controlv1.ReindexResponse
		action   string
		wantErr  bool
	}{
		{name: "nil response", action: "cutover", wantErr: true},
		{name: "wrong protocol version", response: &controlv1.ReindexResponse{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion + 1, SupportedMigrationActions: []string{"cutover"}}, action: "cutover", wantErr: true},
		{name: "unsupported action", response: &controlv1.ReindexResponse{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion, SupportedMigrationActions: []string{"shadow"}}, action: "cutover", wantErr: true},
		{name: "cutover advertised", response: &controlv1.ReindexResponse{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion, SupportedMigrationActions: []string{"shadow", "cutover"}}, action: "cutover"},
		{name: "promote advertisement canonicalized", response: &controlv1.ReindexResponse{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion, SupportedMigrationActions: []string{"promote"}}, action: "cutover"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNegotiatedMigrationCapability(tc.response, tc.action)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateNegotiatedMigrationCapability() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
