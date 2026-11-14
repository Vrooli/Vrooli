package gatewayreq

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func requestTestContext(flags map[string]string) cliapp.RunContext {
	values := map[string]string{
		"kind":                   "text",
		"role":                   "chat.small",
		"profile":                "local-only",
		"privacy-class":          "internal",
		"operation":              "test",
		"scenario":               "test",
		"timeout-ms":             "90000",
		"max-output-tokens":      "180",
		"max-cost-usd":           "0",
		"request-id":             "test-request",
		"required-vram-bytes":    "",
		"expected-worker-digest": "",
	}
	for key, value := range flags {
		values[key] = value
	}
	flagSchema := make([]cliapp.Flag, 0, len(values))
	for name := range values {
		flagSchema = append(flagSchema, cliapp.Flag{Name: name})
	}
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema},
		Flags:  values,
	})
}

func TestInferenceContractKeepsMissingPrivacyUnknownAndBlocksRemote(t *testing.T) {
	ctx := requestTestContext(map[string]string{"profile": "remote-only", "privacy-class": ""})
	profile, privacy, err := InferenceContract(ctx)
	require.NoError(t, err)
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, profile)
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED, privacy)
	// The request contract intentionally leaves the enforcement to the shared
	// routing policy; this caller test proves it does not invent a class.
}

func TestInferenceContractAcceptsOnlyRecognizedExplicitProfileAndPrivacy(t *testing.T) {
	ctx := requestTestContext(map[string]string{"profile": "remote-only", "privacy-class": "public"})
	profile, privacy, err := InferenceContract(ctx)
	require.NoError(t, err)
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, profile)
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC, privacy)
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC, privacy)

	for _, test := range []struct{ profile, privacy string }{
		{"future-profile", "public"},
		{"remote-only", "future-class"},
	} {
		ctx := requestTestContext(map[string]string{"profile": test.profile, "privacy-class": test.privacy})
		_, _, err := InferenceContract(ctx)
		require.Error(t, err)
	}
}

func TestFromContextCarriesNoReclaimFootprintAndExpectedWorker(t *testing.T) {
	digest := "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd"
	ctx := requestTestContext(map[string]string{
		"required-vram-bytes":    "6442450944",
		"expected-worker-digest": digest,
	})

	request, err := FromContext(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 6442450944, request.GetRequiredVramBytes())
	require.Equal(t, "6442450944", request.GetMetadata()["required_vram_bytes"])
	require.Equal(t, "false", request.GetMetadata()["capacity_allow_reclaim"])
	require.Equal(t, digest, request.GetMetadata()["expected_worker_digest"])
}

func TestFromContextRejectsDigestWithoutFootprint(t *testing.T) {
	ctx := requestTestContext(map[string]string{
		"expected-worker-digest": "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd",
	})

	_, err := FromContext(ctx)
	require.ErrorContains(t, err, "requires a positive --required-vram-bytes")
}

func TestFromContextRejectsInvalidFootprintOrDigest(t *testing.T) {
	for name, flags := range map[string]map[string]string{
		"zero footprint": {"required-vram-bytes": "0"},
		"malformed digest": {
			"required-vram-bytes":    "6442450944",
			"expected-worker-digest": "not-a-digest",
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := requestTestContext(flags)
			_, err := FromContext(ctx)
			require.Error(t, err)
		})
	}
}
