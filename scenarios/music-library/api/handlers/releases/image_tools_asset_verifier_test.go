package releases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	domain "music-library/internal/releases"
)

func TestImageToolsAssetVerifierRejectsUnboundAssetsBeforeOwnerLookup(t *testing.T) {
	lookups := 0
	verifier := &imageToolsAssetVerifier{baseURL: func(context.Context) (string, error) {
		lookups++
		return "http://127.0.0.1:1234", nil
	}}
	asset := domain.Asset{
		OwnerScenario: "image-tools", Ref: "outputs/job/cover.png", SHA256: "not-a-hash",
		OperationID: "job-1", ModelID: "model-1", LicenseLane: "MIT",
		RightsStatus: "personal_use", ProvenanceRef: "another-owner:job-1",
	}
	verified, err := verifier.VerifyAsset(context.Background(), asset)
	require.NoError(t, err)
	require.False(t, verified)
	require.Zero(t, lookups, "an unbound provenance reference must be rejected without contacting the owner")

	asset.ProvenanceRef = "image-tools:job-1"
	asset.Ref = "outputs/../private/cover.png"
	verified, err = verifier.VerifyAsset(context.Background(), asset)
	require.NoError(t, err)
	require.False(t, verified)
	require.Zero(t, lookups, "path traversal must be rejected before any owner request")
}

func TestImageToolsAssetVerifierFailsClosedWhenOwnerDiscoveryIsUnavailable(t *testing.T) {
	verifier := &imageToolsAssetVerifier{baseURL: func(context.Context) (string, error) {
		return "", errors.New("owner unavailable")
	}}
	verified, err := verifier.VerifyAsset(context.Background(), domain.Asset{
		OwnerScenario: "image-tools", Ref: "outputs/job/cover.png", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OperationID: "job-1", ModelID: "model-1", LicenseLane: "MIT",
		RightsStatus: "personal_use", ProvenanceRef: "image-tools:job-1",
	})
	require.NoError(t, err)
	require.False(t, verified, "an unqueryable owner cannot turn a proposed reference into evidence")
}
