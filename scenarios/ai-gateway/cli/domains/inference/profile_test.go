package inference

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type inferenceCaptureClient struct {
	inferenceconnect.InferenceServiceClient
	run      *inferencev1.RunRequest
	runBatch *inferencev1.RunBatchRequest
}

func (c *inferenceCaptureClient) Run(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	c.run = req.Msg
	return connect.NewResponse(&inferencev1.RunResponse{Validated: true}), nil
}

func (c *inferenceCaptureClient) RunBatch(_ context.Context, req *connect.Request[inferencev1.RunBatchRequest]) (*connect.Response[inferencev1.RunBatchResponse], error) {
	c.runBatch = req.Msg
	return connect.NewResponse(&inferencev1.RunBatchResponse{}), nil
}

func inferenceRunContext(t *testing.T, flags map[string]string) cliapp.RunContext {
	t.Helper()
	flagSchema := make([]cliapp.Flag, 0, len(flags))
	for name := range flags {
		flagSchema = append(flagSchema, cliapp.Flag{Name: name})
	}
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: flagSchema}, Flags: flags, JSON: true, Stdout: &bytes.Buffer{},
	})
}

func TestTypedInferenceHandlersPreserveExplicitAndUnknownPrivacy(t *testing.T) {
	temp := t.TempDir()
	schema := filepath.Join(temp, "schema.json")
	require.NoError(t, os.WriteFile(schema, []byte(`{"type":"object"}`), 0o600))
	items := filepath.Join(temp, "items.json")
	require.NoError(t, os.WriteFile(items, []byte(`[{"source":"fixture"}]`), 0o600))

	for _, tc := range []struct {
		name    string
		privacy string
		want    sharedv1.PrivacyClass
	}{
		{name: "omitted remains unknown", want: sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED},
		{name: "explicit public is preserved", privacy: "public", want: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := map[string]string{"source": "fixture", "schema": schema, "instruction": "classify", "role": "classify.fast", "profile": "remote-only", "privacy-class": tc.privacy, "temperature": "", "max-output-tokens": ""}
			client := &inferenceCaptureClient{}
			err := (&handlers{client: client}).run(inferenceRunContext(t, base))
			require.NoError(t, err)
			require.NotNil(t, client.run)
			require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, client.run.GetProfile())
			require.Equal(t, tc.want, client.run.GetPrivacyClass())

			batchFlags := map[string]string{"items": items, "schema": schema, "instruction": "classify", "role": "classify.fast", "profile": "remote-only", "privacy-class": tc.privacy}
			client = &inferenceCaptureClient{}
			err = (&handlers{client: client}).runBatch(inferenceRunContext(t, batchFlags))
			require.NoError(t, err)
			require.NotNil(t, client.runBatch)
			require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, client.runBatch.GetProfile())
			require.Equal(t, tc.want, client.runBatch.GetPrivacyClass())
		})
	}
}
