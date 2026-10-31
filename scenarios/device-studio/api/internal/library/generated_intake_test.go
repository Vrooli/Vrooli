package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"testing"
)

func TestPersistGeneratedKeepsUnreviewedBytesAndFullLineage(t *testing.T) {
	ctx, service, _, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "device-studio-physical-glb-v1", ProfileVersion: 1, Supported: true, TriangleCount: 4}, nil
	})
	data := intakeFixtureGLB()
	ref := sha256.Sum256([]byte("reference"))
	recipe := sha256.Sum256([]byte("recipe"))
	outcome := sha256.Sum256([]byte("outcome"))
	lineage := CreationLineage{JobID: "geometry-job-1", RecipeSHA256: hex.EncodeToString(recipe[:]), InferenceOutcomeSHA256: hex.EncodeToString(outcome[:]), ReferenceSHA256: []string{hex.EncodeToString(ref[:])}, Provider: "ollama", Model: "local-unknown", ProviderCalls: 1, InputTokens: 12, OutputTokens: 25, InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, AppliedSettings: &sharedv1.AppliedSettings{TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN, MaxOutputTokensEffective: 4096, MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST}}
	claims := intakeClaims("created-candidate")
	claims.SourceLabel = "Device Studio local AI"
	claims.LicenseClaim = "unknown"
	claims.Attribution = ""
	got, err := service.PersistGenerated(ctx, claims, lineage.JobID, lineage, data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ImportMethod != "local_ai_geometry" || got.ReviewStatus != "unreviewed" || got.RenderStatus != "not_reviewed" || got.Creation == nil || got.Creation.RecipeSHA256 != lineage.RecipeSHA256 || got.Creation.ReferenceSHA256[0] != lineage.ReferenceSHA256[0] || got.Creation.AppliedSettings.GetMaxOutputTokensEffective() != 4096 || got.Creation.InputTokensProvenance != lineage.InputTokensProvenance || got.Creation.InferenceOutcomeSHA256 != lineage.InferenceOutcomeSHA256 {
		t.Fatalf("generated custody/lineage was lost: %+v", got)
	}
	opened, reader, err := service.Open(ctx, got.ID, got.Revision)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	_, err = output.ReadFrom(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.OriginalSHA256 != got.OriginalSHA256 || !bytes.Equal(data, output.Bytes()) {
		t.Fatal("generated GLB did not reopen as its exact immutable bytes")
	}
	if _, err = service.PersistGenerated(ctx, claims, lineage.JobID, lineage, data); err != nil {
		t.Fatalf("exact job retry failed to reconcile candidate: %v", err)
	}
}

func TestPersistGeneratedRejectsUnboundLineage(t *testing.T) {
	ctx, service, _, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) { return IntakeInspection{}, nil })
	if _, err := service.PersistGenerated(ctx, intakeClaims("bad"), "job", CreationLineage{JobID: "job", ProviderCalls: 1}, intakeFixtureGLB()); err == nil {
		t.Fatal("incomplete model lineage must be refused")
	}
}
