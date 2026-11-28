package programs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/bindings"
	"program-runtime/internal/sessions"
)

func findRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate test source")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "../../../../.."))
}

type artifactSessionFixture struct{ session *sessions.Session }

func (f artifactSessionFixture) Get(_ context.Context, id string) (*sessions.Session, error) {
	if f.session.ID != id {
		return nil, os.ErrNotExist
	}
	return f.session, nil
}

type artifactProgramFixture struct{ sessionID, name string }

func (f artifactProgramFixture) ExecutionIdentity(_ context.Context, _ string) (string, string, string, error) {
	return f.name, strings.Repeat("a", 64), f.sessionID, nil
}

func (f artifactProgramFixture) Get(_ context.Context, id string) (*programsv1.Program, error) {
	return &programsv1.Program{Id: id, Status: programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING}, nil
}

type artifactSourceFixture struct {
	opened                    int
	target, runID, artifactID string
	data                      string
}

func (f *artifactSourceFixture) OpenTestGenieArtifactStream(_ context.Context, _, _, target, runID, artifactID string, _ []string, _ bindings.InvocationMetadata, _ *http.Client) (bindings.TestGenieArtifact, io.ReadCloser, error) {
	f.opened++
	f.target, f.runID, f.artifactID = target, runID, artifactID
	data := f.data
	if data == "" {
		data = "synthetic"
	}
	return bindings.TestGenieArtifact{Target: target, RunID: runID, ArtifactID: artifactID, Kind: "findings", SizeBytes: int64(len(data)), AccessCapability: "ARTIFACT_ACCESS_CAPABILITY_STREAM"}, io.NopCloser(strings.NewReader(data)), nil
}

func TestSyntheticProgramInvocationStreamsThroughTrustedArtifactEffect(t *testing.T) {
	const raw = "synthetic-private-artifact-bytes"
	workspace := t.TempDir()
	source := &artifactSourceFixture{data: raw}
	bridge := &ArtifactStreamBridge{
		Sessions: artifactSessionFixture{session: &sessions.Session{ID: "sess-1", State: "running", SandboxWorkspace: workspace, Grants: map[string]struct{}{"workspace:write": {}}}},
		Programs: artifactProgramFixture{sessionID: "sess-1", name: runEvidenceProgramName},
		Bindings: source,
	}
	var requestJSON string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read bridge request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		requestJSON = string(body)
		if strings.Contains(requestJSON, raw) {
			t.Error("artifact bytes entered persistent bridge request JSON")
		}
		r.Body = io.NopCloser(strings.NewReader(requestJSON))
		bridge.ServeHTTP(w, r)
	}))
	defer server.Close()
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repoRoot, "scenarios/program-runtime/api/handlers/programs/testdata/run_evidence_artifact_integration.py")
	run := func(path string) []byte {
		t.Helper()
		cmd := exec.Command("python3", script)
		cmd.Env = append(os.Environ(), "VROOLI_REPO_ROOT="+repoRoot, "PR4_BRIDGE_URL="+server.URL+"/bindings/execute", "PR4_OUTPUT_PATH="+path)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("synthetic consumer execution: %v\n%s", err, out)
		}
		return out
	}

	output := string(run("evidence/report.bin"))
	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("consumer JSON: %v (%s)", err, output)
	}
	if envelope["status"] != "ok" || strings.Contains(output, raw) {
		t.Fatalf("status=%v or artifact bytes leaked: %s", envelope["status"], output)
	}
	signals := envelope["signals"].(map[string]any)
	receipt := signals["receipt"].(map[string]any)
	wantHash := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
	if receipt["scenario"] != "demo" || receipt["run_id"] != "run-1" || receipt["artifact_id"] != "artifact_0123456789abcdef0123456789abcdef" || receipt["output_path"] != "evidence/report.bin" || receipt["bytes"] != float64(len(raw)) || receipt["sha256"] != wantHash || receipt["source_digest_verified"] != false {
		t.Fatalf("metadata receipt mismatch: %v", receipt)
	}
	if source.opened != 1 || source.target != "demo" || source.runID != "run-1" || source.artifactID != "artifact_0123456789abcdef0123456789abcdef" {
		t.Fatalf("stream identity: %+v", source)
	}
	if strings.Contains(requestJSON, raw) {
		t.Fatal("bridge request contained artifact bytes")
	}
	written, err := os.ReadFile(filepath.Join(workspace, "evidence/report.bin"))
	if err != nil || string(written) != raw {
		t.Fatalf("synthetic file content=%q err=%v", written, err)
	}

	// A second write is refused without explicit overwrite and preserves the first file.
	noClobberOutput := string(run("evidence/report.bin"))
	if !strings.Contains(noClobberOutput, `"status":"partial"`) {
		t.Fatalf("no-clobber did not return an uncertain refusal: %s", noClobberOutput)
	}
	written, err = os.ReadFile(filepath.Join(workspace, "evidence/report.bin"))
	if err != nil || string(written) != raw {
		t.Fatalf("no-clobber changed destination: %q err=%v", written, err)
	}

	// The typed consumer rejects traversal before invoking the stream effect.
	traversalOutput := string(run("../escape.bin"))
	if !strings.Contains(traversalOutput, `"status":"failed"`) || source.opened != 2 {
		t.Fatalf("traversal was not refused before streaming: opened=%d output=%s", source.opened, traversalOutput)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(workspace), "escape.bin")); !os.IsNotExist(err) {
		t.Fatalf("traversal escaped workspace: %v", err)
	}
	bridgeRequest := `{"session_id":"sess-1","program_id":"prog-1","scenario":"demo","run_id":"run-1","artifact_id":"artifact_0123456789abcdef0123456789abcdef","path":"../bridge-escape.bin","overwrite":false}`
	response := httptest.NewRecorder()
	bridge.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(bridgeRequest)))
	if response.Code != http.StatusBadRequest || source.opened != 3 {
		t.Fatalf("bridge did not reject escaped path: status=%d opens=%d body=%s", response.Code, source.opened, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(workspace), "bridge-escape.bin")); !os.IsNotExist(err) {
		t.Fatalf("bridge path escaped admitted workspace: %v", err)
	}
}

func TestArtifactStreamBridgeWritesMetadataReceiptAndEnforcesPinnedOwner(t *testing.T) {
	workspace := t.TempDir()
	source := &artifactSourceFixture{}
	bridge := &ArtifactStreamBridge{
		Sessions: artifactSessionFixture{session: &sessions.Session{ID: "sess-1", State: "running", SandboxWorkspace: workspace, Grants: map[string]struct{}{"workspace:write": {}}}},
		Programs: artifactProgramFixture{sessionID: "sess-1", name: runEvidenceProgramName},
		Bindings: source,
	}
	request := `{"session_id":"sess-1","program_id":"prog-1","scenario":"demo","run_id":"run-1","artifact_id":"artifact_0123456789abcdef0123456789abcdef","path":"receipts/findings.json","overwrite":false}`
	response := httptest.NewRecorder()
	bridge.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(request)))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var receipt map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["scenario"] != "demo" || receipt["run_id"] != "run-1" || receipt["artifact_id"] != "artifact_0123456789abcdef0123456789abcdef" || receipt["output_path"] != "receipts/findings.json" || receipt["bytes"] != float64(len("synthetic")) || receipt["sha256"] == "" {
		t.Fatalf("receipt=%v", receipt)
	}
	if strings.Contains(response.Body.String(), "synthetic") {
		t.Fatal("artifact bytes leaked into the JSON receipt")
	}
	got, err := os.ReadFile(filepath.Join(workspace, "receipts/findings.json"))
	if err != nil || string(got) != "synthetic" {
		t.Fatalf("written artifact=%q err=%v", got, err)
	}
	if source.opened != 1 {
		t.Fatalf("source opens=%d", source.opened)
	}

	wrongOwner := *bridge
	wrongOwner.Programs = artifactProgramFixture{sessionID: "sess-1", name: "other.program"}
	response = httptest.NewRecorder()
	wrongOwner.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(request)))
	if response.Code != http.StatusForbidden || source.opened != 1 {
		t.Fatalf("wrong owner status=%d source opens=%d", response.Code, source.opened)
	}
}

func TestArtifactStreamBridgeRejectsInactiveSessionAndMalformedInput(t *testing.T) {
	workspace := t.TempDir()
	source := &artifactSourceFixture{}
	bridge := &ArtifactStreamBridge{
		Sessions: artifactSessionFixture{session: &sessions.Session{ID: "sess-1", State: "stopped", SandboxWorkspace: workspace}},
		Programs: artifactProgramFixture{sessionID: "sess-1", name: runEvidenceProgramName},
		Bindings: source,
	}
	request := `{"session_id":"sess-1","program_id":"prog-1","scenario":"demo","run_id":"run-1","artifact_id":"artifact_0123456789abcdef0123456789abcdef","path":"out.bin"}`
	response := httptest.NewRecorder()
	bridge.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(request)))
	if response.Code != http.StatusForbidden || source.opened != 0 {
		t.Fatalf("inactive session status=%d source opens=%d", response.Code, source.opened)
	}
	response = httptest.NewRecorder()
	bridge.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"session_id":"sess-1","unexpected":true}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d body=%s", response.Code, response.Body.String())
	}
}
