package imageengine

import (
	"context"
	"encoding/json"
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
	out, err := client.Apply(context.Background(), ApplyRequest{Input: []byte{1, 2}, Treatments: []string{"duotone", "grain"}, Palette: map[string]string{"$brand.primary": "#123456"}})
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 1, 2}, out)
	require.Equal(t, []string{"/api/v1/ops/duotone", "/api/v1/ops/grain"}, operations)
}

func TestClientApplyRefusesMissingInputsAndUnresolvedTreatmentNames(t *testing.T) {
	client := &Client{Resolve: func(context.Context) (string, error) { return "http://example.test", nil }}
	_, err := client.Apply(context.Background(), ApplyRequest{Treatments: []string{"grain"}})
	require.ErrorContains(t, err, "input image is empty")
	_, err = client.Apply(context.Background(), ApplyRequest{Input: []byte{1}, Treatments: []string{"$brand.primary"}})
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
			_, _ = w.Write([]byte(`{"job_id":"job-1","model_id":"sd-1.5/local-gpu","tier":"local-gpu"}`))
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
	require.Equal(t, []byte{9, 8, 7}, out.PNG)
	require.True(t, sawConditioning)
	// The model the router chose comes back with the bytes. Without it a
	// model-backed release has no model to disclose and is refused.
	require.Equal(t, "sd-1.5/local-gpu", out.ModelID)
	require.Equal(t, "local-gpu", out.Tier)
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

	_, err := client.Apply(context.Background(), ApplyRequest{
		Input:      []byte{1, 2},
		Treatments: []string{"duotone"},
		Params:     map[string]string{"duotone": `{"dark":"$brand.primary","light":"#ffffff"}`},
		Palette:    map[string]string{"$brand.background": "#ffffff"}, // primary is absent
	})

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
	}, nil)
	require.NoError(t, err)
	require.Contains(t, raw, "#1B3FBF")
	require.Contains(t, raw, "#F5EFDC")
	require.NotContains(t, raw, "$brand.")
}

// TestReservedSpaceReachesEveryOperationOnTheWire pins the half of the knockout
// that lives in this package: the rectangle has to arrive, on every operation in
// the chain, as a sibling of the operation's own parameters.
//
// Asserted against the bytes actually sent rather than against the struct that
// produced them, because the failure this guards is precisely a field that is
// set, validated and then not transmitted. That has happened once already in
// this client — a routing policy was carried on the request type, checked, and
// never written to the wire, and every unit test passed because the fake on the
// other side answered the same way with or without it.
func TestReservedSpaceReachesEveryOperationOnTheWire(t *testing.T) {
	var sent []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(r.FormValue("params")), &decoded))
		sent = append(sent, decoded)
		_, _ = w.Write([]byte{1, 2})
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	_, err := client.Apply(context.Background(), ApplyRequest{
		Input:      []byte{1, 2},
		Treatments: []string{"duotone", "dither_ordered", "grain"},
		Reserve:    &Knockout{X: 0.06, Y: 0.1, Width: 0.42, Height: 0.34, Feather: 0.09},
	})
	require.NoError(t, err)
	require.Len(t, sent, 3, "every operation in the chain is a separate request")

	for i, params := range sent {
		reserve, ok := params["knockout"].(map[string]any)
		require.Truef(t, ok, "operation %d sent no knockout: %v", i, params)
		require.InDelta(t, 0.06, reserve["x"], 1e-9)
		require.InDelta(t, 0.42, reserve["width"], 1e-9)
		require.InDelta(t, 0.09, reserve["feather"], 1e-9)
	}
}

// TestNoReserveSendsNoKnockout keeps the common case honest. A style that
// reserves nothing must send nothing: an always-present zero rectangle would
// read downstream as a declaration rather than an absence, and the two have to
// stay distinguishable for image-tools to skip the work entirely.
func TestNoReserveSendsNoKnockout(t *testing.T) {
	var params map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		require.NoError(t, json.Unmarshal([]byte(r.FormValue("params")), &params))
		_, _ = w.Write([]byte{1})
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	_, err := client.Apply(context.Background(), ApplyRequest{Input: []byte{1}, Treatments: []string{"grain"}})
	require.NoError(t, err)
	require.NotContains(t, params, "knockout")
}
