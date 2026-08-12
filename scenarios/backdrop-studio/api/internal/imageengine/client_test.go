package imageengine

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientApplyRunsTheOrderedImageToolsChain(t *testing.T) {
	var operations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operations = append(operations, r.URL.Path)
		require.Equal(t, "bytes", r.URL.Query().Get("output"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()
		input, err := io.ReadAll(file)
		require.NoError(t, err)
		if len(operations) == 1 {
			require.Equal(t, []byte{1, 2}, input)
		} else {
			require.Equal(t, []byte{1, 2, 1}, input)
		}
		params := r.FormValue("params")
		if strings.HasSuffix(r.URL.Path, "/duotone") {
			require.Contains(t, params, "#123456")
		}
		_, _ = w.Write(append(input, byte(len(operations))))
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	out, err := client.Apply(context.Background(), []byte{1, 2}, []string{"duotone", "grain"}, nil, map[string]string{"$brand.primary": "#123456"})
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 1, 2}, out)
	require.Equal(t, []string{"/api/v1/ops/duotone", "/api/v1/ops/grain"}, operations)
}

func TestClientApplyRefusesMissingInputsAndUnresolvedTreatmentNames(t *testing.T) {
	client := &Client{Resolve: func(context.Context) (string, error) { return "http://example.test", nil }}
	_, err := client.Apply(context.Background(), nil, []string{"grain"}, nil, nil)
	require.ErrorContains(t, err, "input image is empty")
	_, err = client.Apply(context.Background(), []byte{1}, []string{"$brand.primary"}, nil, nil)
	require.ErrorContains(t, err, "invalid treatment")
}

func TestClientGenerateUsesImageToolsInferenceAndBlocksOnceForResult(t *testing.T) {
	var sawConditioning bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/image_to_image":
			require.NoError(t, r.ParseMultipartForm(1<<20))
			_, _, err := r.FormFile("file")
			require.NoError(t, err)
			sawConditioning = true
			w.Header().Set("Content-Type", "application/json")
			// image-tools' REST submit edge serialises with protojson PROTO
			// names, so this is `job_id`. The fake previously wrote `jobId`,
			// which is why the suite stayed green while every real
			// model-backed render failed with "returned no job id".
			_, _ = w.Write([]byte(`{"job_id":"job-1"}`))
		case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"job":{"state":"JOB_STATE_SUCCEEDED","resultRef":"out/result.png"}}`))
		case "/api/v1/blobs/out/result.png":
			_, _ = w.Write([]byte{9, 8, 7})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	out, err := client.Generate(context.Background(), GenerationRequest{Width: 512, Height: 320, Prompt: "quiet field", Conditioning: []byte{1, 2}})
	require.NoError(t, err)
	require.Equal(t, []byte{9, 8, 7}, out)
	require.True(t, sawConditioning)
}

// TestUnresolvedBrandSlotFailsClosed is the regression gate for the defect that
// made ten of sixteen seeded styles unrenderable.
//
// `mergedParams` used to fall through and write the literal slot string onto
// the wire when the palette lookup missed, so image-tools answered
// `422 invalid color "$brand.primary"`. The failure was invisible to the suite
// because every test bound a brand — the one thing a CLI caller never did.
func TestUnresolvedBrandSlotFailsClosed(t *testing.T) {
	var reached bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer server.Close()
	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}

	_, err := client.Apply(
		context.Background(),
		[]byte{1, 2},
		[]string{"duotone"},
		map[string]string{"duotone": `{"dark":"$brand.primary","light":"#ffffff"}`},
		map[string]string{"$brand.background": "#ffffff"}, // primary is absent
	)

	var unresolved *UnresolvedSlotError
	require.ErrorAs(t, err, &unresolved, "an unbindable slot must be a typed error, never a literal on the wire")
	require.Equal(t, "$brand.primary", unresolved.Slot)
	require.Equal(t, "duotone", unresolved.Operation)
	require.Equal(t, "dark", unresolved.Field)
	require.False(t, reached, "the request must never leave the process once a slot is unbindable")
}

// TestResolvedBrandSlotReachesTheWire is the positive half: proving the gate
// fails is only useful alongside proving it passes what it should.
func TestResolvedBrandSlotReachesTheWire(t *testing.T) {
	raw, err := ResolveParams("duotone", `{"dark":"$brand.primary","light":"$brand.background"}`, map[string]string{
		"$brand.primary":    "#1B3FBF",
		"$brand.background": "#F5EFDC",
	})
	require.NoError(t, err)
	require.Contains(t, raw, "#1B3FBF")
	require.Contains(t, raw, "#F5EFDC")
	require.NotContains(t, raw, "$brand.")
}
