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
	"google.golang.org/protobuf/encoding/protojson"
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

type artifactCatalogClient struct {
	fakeClient
	response *runspb.ListRunArtifactsResponse
}

func (c *artifactCatalogClient) ListRunArtifacts(_ context.Context, req *connect.Request[runspb.ListRunArtifactsRequest]) (*connect.Response[runspb.ListRunArtifactsResponse], error) {
	if req.Msg.GetTarget() != "demo" || req.Msg.GetRunId() != "run-1" {
		return nil, fmt.Errorf("unexpected artifact catalog request: %+v", req.Msg)
	}
	return connect.NewResponse(c.response), nil
}

func artifactMetadataContext(t *testing.T) cliapp.OperationContext {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "run_id", Required: true}, {Name: "artifact_id", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "kinds"},
			{Name: "phase"},
		},
	}, []string{"--scenario", "demo", "run-1", testArtifactID}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("parse artifact metadata args: %v", err)
	}
	return ctx
}

func TestArtifactCatalogCLIUsesOpaqueIdsAndHidesStreamPath(t *testing.T) {
	streamPath := "/api/v1/scenarios/demo/runs/run-1/artifacts/" + testArtifactID
	withFakeClient(t, &artifactCatalogClient{response: &runspb.ListRunArtifactsResponse{
		SchemaVersion: 1,
		Artifacts:     []*runspb.ArtifactRef{{Id: testArtifactID, Kind: "findings.report", AccessPath: streamPath}},
	}})

	ctx := artifactMetadataContext(t)
	response, err := listRunArtifactsCall(nil)(ctx)
	if err != nil {
		t.Fatalf("listRunArtifactsCall: %v", err)
	}
	if got := response.GetArtifacts()[0].GetAccessPath(); got != "" {
		t.Fatalf("CLI artifact catalog exposed stream path %q", got)
	}
	payload, err := protojson.Marshal(response)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	if strings.Contains(string(payload), "accessPath") || strings.Contains(string(payload), streamPath) {
		t.Fatalf("JSON catalog exposed stream path: %s", payload)
	}
	report := listRunArtifactsReport(ctx, response)
	if strings.Contains(strings.Join(append(report.Summary, report.Results...), "\n"), streamPath) {
		t.Fatal("human artifact catalog exposed stream path")
	}
	if summary := strings.Join(report.Summary, "\n"); !strings.Contains(summary, "download-artifact <scenario> <run_id> <artifact_id>") || !strings.Contains(summary, "SHA-256 receipt") || !strings.Contains(summary, "runs logs <scenario> <run_id>") {
		t.Fatalf("catalog did not describe the download checks accurately: %q", summary)
	}
}

func TestArtifactMetadataCLIUsesDownloadCommandInsteadOfStreamPath(t *testing.T) {
	streamPath := "/api/v1/scenarios/demo/runs/run-1/artifacts/" + testArtifactID
	withFakeClient(t, &artifactMetadataClient{response: &runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
		Id: testArtifactID, Kind: "findings.report", AccessPath: streamPath,
	}}})

	ctx := artifactMetadataContext(t)
	response, err := getRunArtifactCall(nil)(ctx)
	if err != nil {
		t.Fatalf("getRunArtifactCall: %v", err)
	}
	if got := response.GetArtifact().GetAccessPath(); got != "" {
		t.Fatalf("CLI artifact metadata exposed stream path %q", got)
	}
	payload, err := protojson.Marshal(response)
	if err != nil {
		t.Fatalf("marshal artifact metadata: %v", err)
	}
	if strings.Contains(string(payload), "accessPath") || strings.Contains(string(payload), streamPath) {
		t.Fatalf("JSON metadata exposed stream path: %s", payload)
	}
	report := getRunArtifactReport(ctx, response)
	combined := strings.Join(append(report.Summary, report.Results...), "\n")
	if strings.Contains(combined, streamPath) || !strings.Contains(combined, "download-artifact") {
		t.Fatalf("human metadata did not hide path and direct to CLI download: %q", combined)
	}
}

func artifactDownloadContext(t *testing.T, output string, overwrite bool) cliapp.OperationContext {
	t.Helper()
	var stdout, stderr bytes.Buffer
	args := []string{"demo", "run-1", testArtifactID, "--out", output}
	if overwrite {
		args = append(args, "--overwrite")
	}
	ctx, err := cliapp.NewTestRunContextFromArgs(downloadArtifactSchema, args, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("parse download args: %v", err)
	}
	return ctx
}

var downloadArtifactSchema = cliapp.ArgSchema{
	Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "run_id", Required: true}, {Name: "artifact_id", Required: true}},
	Flags:       []cliapp.Flag{{Name: "out"}, {Name: "overwrite", Bool: true}},
}

// artifactServer serves one artifact body per id on its run-scoped route.
func artifactServer(t *testing.T, bodies map[string]string) (*httptest.Server, *cliutil.APIClient) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for id, body := range bodies {
			if r.URL.Path == "/api/v1/scenarios/demo/runs/run-1/artifacts/"+id {
				_, _ = fmt.Fprint(w, body)
				return
			}
		}
		t.Errorf("unexpected artifact request %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	return server, apiClient
}

func streamRef(id, kind, phase, label, created string, size int) *runspb.ArtifactRef {
	return &runspb.ArtifactRef{
		Id: id, Kind: kind, ProducingPhase: phase, Label: label, CreatedAt: created, SizeBytes: int64(size),
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
		AccessPath:       "/api/v1/scenarios/demo/runs/run-1/artifacts/" + id,
	}
}

// catalogClient answers catalog and metadata reads from one artifact list.
type catalogClient struct {
	fakeClient
	artifacts []*runspb.ArtifactRef
	listed    *runspb.ListRunArtifactsRequest
}

func (c *catalogClient) ListRunArtifacts(_ context.Context, req *connect.Request[runspb.ListRunArtifactsRequest]) (*connect.Response[runspb.ListRunArtifactsResponse], error) {
	c.listed = req.Msg
	return connect.NewResponse(&runspb.ListRunArtifactsResponse{SchemaVersion: 1, Artifacts: c.artifacts}), nil
}

func (c *catalogClient) GetRunArtifact(_ context.Context, req *connect.Request[runspb.GetRunArtifactRequest]) (*connect.Response[runspb.GetRunArtifactResponse], error) {
	for _, artifact := range c.artifacts {
		if artifact.GetId() == req.Msg.GetArtifactId() {
			return connect.NewResponse(&runspb.GetRunArtifactResponse{Artifact: artifact}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no artifact %s", req.Msg.GetArtifactId()))
}

func runHandler(t *testing.T, handler cliapp.PrimitiveHandler, schema cliapp.ArgSchema, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, args, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	err = handler.Run(ctx)
	return stdout.String(), err
}

func TestDownloadArtifactPrintsContentToStdoutByDefault(t *testing.T) {
	_, apiClient := artifactServer(t, map[string]string{testArtifactID: "line one\nline two\n"})
	withFakeClient(t, &catalogClient{artifacts: []*runspb.ArtifactRef{streamRef(testArtifactID, "findings.report", "", "findings.json", "t1", len("line one\nline two\n"))}})

	for _, args := range [][]string{
		{"demo", "run-1", testArtifactID},
		{"demo", "run-1", testArtifactID, "--out", "-"},
	} {
		out, err := runHandler(t, downloadArtifactHandler(apiClient), downloadArtifactSchema, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if out != "line one\nline two\n" {
			t.Fatalf("%v: stdout = %q, want the artifact bytes alone", args, out)
		}
	}
}

func TestRunLogsPrintsEachPhaseLogUnderAHeader(t *testing.T) {
	structure, unit := "artifact_aaaa", "artifact_bbbb"
	_, apiClient := artifactServer(t, map[string]string{structure: "structure ok\n", unit: "FAIL TestX\n"})
	fake := &catalogClient{artifacts: []*runspb.ArtifactRef{
		streamRef(unit, commandLogKind, "unit", "unit.log", "2026-11-06T10:00:02Z", len("FAIL TestX\n")),
		streamRef(structure, commandLogKind, "structure", "structure.log", "2026-11-06T10:00:01Z", len("structure ok\n")),
	}}
	withFakeClient(t, fake)
	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "run_id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "phase"}},
	}

	out, err := runHandler(t, runLogsHandler(apiClient), schema, "demo", "run-1")
	if err != nil {
		t.Fatalf("runs logs: %v", err)
	}
	want := "==> phase structure: structure.log (artifact_aaaa, 13 bytes) <==\nstructure ok\n\n==> phase unit: unit.log (artifact_bbbb, 11 bytes) <==\nFAIL TestX\n"
	if out != want {
		t.Fatalf("runs logs output:\n%s\nwant:\n%s", out, want)
	}
	if got := fake.listed.GetKinds(); len(got) != 1 || got[0] != commandLogKind {
		t.Fatalf("catalog request kinds = %v, want only %s", got, commandLogKind)
	}

	fake.artifacts = fake.artifacts[:1]
	out, err = runHandler(t, runLogsHandler(apiClient), schema, "demo", "run-1", "--phase", "unit")
	if err != nil || out != "FAIL TestX\n" {
		t.Fatalf("single phase log = %q err=%v, want the bare log", out, err)
	}
	if fake.listed.GetProducingPhase() != "unit" {
		t.Fatalf("phase filter not sent: %+v", fake.listed)
	}

	fake.artifacts = nil
	_, err = runHandler(t, runLogsHandler(apiClient), schema, "demo", "run-1", "--phase", "lint")
	if err == nil || !strings.Contains(err.Error(), "test-genie runs artifacts demo run-1") {
		t.Fatalf("missing log error should name the catalog command, got %v", err)
	}
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
