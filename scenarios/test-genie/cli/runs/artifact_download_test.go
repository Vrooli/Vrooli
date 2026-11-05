package runs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const testArtifactID = "artifact_0123456789abcdef0123456789abcdef"

type artifactMetadataClient struct {
	fakeClient
	response *runspb.GetRunArtifactResponse
}

func (c *artifactMetadataClient) GetRunArtifact(_ context.Context, req *connect.Request[runspb.GetRunArtifactRequest]) (*connect.Response[runspb.GetRunArtifactResponse], error) {
	if req.Msg.GetTarget() != "demo" || req.Msg.GetRunId() != "run-1" || req.Msg.GetArtifactId() != testArtifactID {
		return nil, fmt.Errorf("unexpected artifact metadata request: %+v", req.Msg)
	}
	return connect.NewResponse(c.response), nil
}

func artifactDownloadContext(t *testing.T, output string, overwrite bool) cliapp.OperationContext {
	t.Helper()
	var stdout, stderr bytes.Buffer
	args := []string{"--scenario", "demo", "--out", output}
	if overwrite {
		args = append(args, "--overwrite")
	}
	args = append(args, "run-1", testArtifactID)
	ctx, err := cliapp.NewTestRunContextFromArgs(cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "run_id", Required: true}, {Name: "artifact_id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "scenario", Required: true}, {Name: "out", Required: true}, {Name: "overwrite", Bool: true}},
	}, args, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("parse download args: %v", err)
	}
	return ctx
}

func TestDownloadArtifactStreamsToAtomicOutputAndReportsDigest(t *testing.T) {
	payload := strings.Repeat("evidence-line\n", 20_000)
	path := "/api/v1/scenarios/demo/runs/run-1/artifacts/" + testArtifactID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("download request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, payload)
	}))
	defer server.Close()

	withFakeClient(t, &artifactMetadataClient{response: &runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
		Id: testArtifactID, Kind: "phase_report", AccessPath: path,
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
		SizeBytes:        int64(len(payload)),
	}}})
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	output := filepath.Join(t.TempDir(), "evidence.txt")

	result, err := downloadArtifactCall(apiClient)(artifactDownloadContext(t, output, false))
	if err != nil {
		t.Fatalf("downloadArtifactCall: %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read published artifact: %v", err)
	}
	if string(got) != payload || result.Bytes != int64(len(payload)) || result.OutputPath != output {
		t.Fatalf("download result=%+v bytes=%d want=%d", result, len(got), len(payload))
	}
	if result.SHA256 == "" || result.ArtifactID != testArtifactID || result.Kind != "phase_report" {
		t.Fatalf("download metadata missing: %+v", result)
	}
	wantDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
	if result.SHA256 != wantDigest {
		t.Fatalf("SHA-256 = %s, want %s", result.SHA256, wantDigest)
	}
}

func TestDownloadArtifactPreservesExistingOutputWithoutOverwrite(t *testing.T) {
	path := "/api/v1/scenarios/demo/runs/run-1/artifacts/" + testArtifactID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, "new") }))
	defer server.Close()
	withFakeClient(t, &artifactMetadataClient{response: &runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
		Id: testArtifactID, Kind: "phase_report", AccessPath: path,
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
		SizeBytes:        int64(len("new")),
	}}})
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	output := filepath.Join(t.TempDir(), "existing.txt")
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := downloadArtifactCall(apiClient)(artifactDownloadContext(t, output, false)); err == nil {
		t.Fatal("download unexpectedly replaced an existing destination")
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing destination changed: %q err=%v", got, err)
	}
}

func TestDownloadArtifactOverwriteReplacesDestination(t *testing.T) {
	path := "/api/v1/scenarios/demo/runs/run-1/artifacts/" + testArtifactID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, "new") }))
	defer server.Close()
	withFakeClient(t, &artifactMetadataClient{response: &runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
		Id: testArtifactID, Kind: "phase_report", AccessPath: path,
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
		SizeBytes:        int64(len("new")),
	}}})
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	output := filepath.Join(t.TempDir(), "existing.txt")
	if err := os.WriteFile(output, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := downloadArtifactCall(apiClient)(artifactDownloadContext(t, output, true)); err != nil {
		t.Fatalf("overwrite download: %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != "new" {
		t.Fatalf("destination = %q err=%v, want new artifact", got, err)
	}
}

func TestDownloadArtifactRejectsUnexpectedAccessPath(t *testing.T) {
	withFakeClient(t, &artifactMetadataClient{response: &runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
		Id: testArtifactID, Kind: "phase_report", AccessPath: "https://example.invalid/secret",
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
	}}})
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: "http://127.0.0.1:1"}
	}, nil)
	if _, err := downloadArtifactCall(apiClient)(artifactDownloadContext(t, filepath.Join(t.TempDir(), "out"), false)); err == nil {
		t.Fatal("download accepted a server access path outside the expected run-scoped route")
	}
}
