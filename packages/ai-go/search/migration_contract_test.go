package aisearch

import "testing"

func TestValidateReindexMigrationRequest(t *testing.T) {
	validShadow := func() *ReindexMigrationRequest {
		return &ReindexMigrationRequest{Action: "shadow", ShadowCollection: "candidate-a", EmbeddingModel: "model@sha256:abc", EmbeddingRole: "embedding.default", EmbeddingDimensions: 768, EmbeddingPolicySchemaVersion: "v1"}
	}
	tests := []struct {
		name    string
		request *ReindexMigrationRequest
		allowed []string
		wantErr bool
	}{
		{name: "legacy reindex remains supported", request: &ReindexMigrationRequest{}},
		{name: "unknown action rejected", request: &ReindexMigrationRequest{Action: "retarget"}, wantErr: true},
		{name: "unsupported capability rejected", request: validShadow(), wantErr: true},
		{name: "shadow requires complete target", request: &ReindexMigrationRequest{Action: "shadow", ShadowCollection: "candidate-a"}, allowed: []string{"shadow"}, wantErr: true},
		{name: "shadow complete", request: validShadow(), allowed: []string{"shadow"}},
		{name: "malformed shadow collection rejected", request: func() *ReindexMigrationRequest { r := validShadow(); r.ShadowCollection = "../live"; return r }(), allowed: []string{"shadow"}, wantErr: true},
		{name: "promote alias normalizes to cutover", request: &ReindexMigrationRequest{Action: "promote", ShadowCollection: "candidate-a"}, allowed: []string{"cutover"}},
		{name: "capabilities is read-only", request: &ReindexMigrationRequest{Action: "capabilities"}, allowed: []string{"capabilities"}},
		{name: "capabilities rejects migration targets", request: &ReindexMigrationRequest{Action: "capabilities", ShadowCollection: "candidate-a"}, allowed: []string{"capabilities"}, wantErr: true},
		{name: "cutover requires exact shadow", request: &ReindexMigrationRequest{Action: "cutover"}, allowed: []string{"cutover"}, wantErr: true},
		{name: "cutover refuses mixed recipe", request: &ReindexMigrationRequest{Action: "cutover", ShadowCollection: "candidate-a", EmbeddingModel: "model"}, allowed: []string{"cutover"}, wantErr: true},
		{name: "rollback requires retained collection", request: &ReindexMigrationRequest{Action: "rollback"}, allowed: []string{"rollback"}, wantErr: true},
		{name: "targets without action rejected", request: &ReindexMigrationRequest{EmbeddingModel: "model"}, wantErr: true},
		{name: "nil request rejected", request: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReindexMigrationRequest(tt.request, tt.allowed...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateReindexMigrationRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
