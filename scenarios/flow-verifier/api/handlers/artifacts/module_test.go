package artifacts_test

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	handlers "flow-verifier/handlers/artifacts"
	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/clock"
	"flow-verifier/internal/runs"
	"flow-verifier/internal/scenarios"
	"flow-verifier/internal/server"
	"flow-verifier/internal/testkit"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/db"
	"flow-verifier/internal/testutil/httpx"

	"github.com/stretchr/testify/require"
)

// stubScenariosSvc resolves a flow id to a fixed root, mirroring how the
// real scenarios.Service answers the artifacts handler's lookups. Errors
// are returned verbatim so the 404 / 500 paths are exercisable.
type stubScenariosSvc struct {
	rows   []scenarios.Summary
	detail scenarios.Detail
	detErr error
}

func (s *stubScenariosSvc) List() ([]scenarios.Summary, error) { return s.rows, nil }
func (s *stubScenariosSvc) Detail(id string) (scenarios.Detail, error) {
	if s.detErr != nil {
		return scenarios.Detail{}, s.detErr
	}
	if id != s.detail.ID {
		return scenarios.Detail{}, scenarios.ErrScenarioNotFound
	}
	return s.detail, nil
}

// stubGenerator captures generate calls without invoking quint.
type stubGenerator struct {
	calls []string
	files map[string]string // relative path → contents to write
	err   error
}

func (g *stubGenerator) Generate(_ context.Context, root, flowID string) error {
	g.calls = append(g.calls, flowID)
	if g.err != nil {
		return g.err
	}
	for rel, body := range g.files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(abs), 0o755)
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func newLive(t *testing.T, root, scenarioID, flowID string, gen *stubGenerator) *httpx.LiveServer {
	t.Helper()
	d := db.NewSQLite(t)
	// Apply minimal schema for runs table even though the stub generator
	// doesn't touch it — pipelineGenerator would, and tests should match
	// production wiring as closely as possible.
	_, err := d.Exec(runs.Schema())
	require.NoError(t, err)
	runsSvc := runs.NewService(runs.NewSQLiteRepository(d, clock.System{}))
	svc := artifacts.NewService(gen)
	sc := &stubScenariosSvc{
		rows: []scenarios.Summary{{ID: scenarioID, Path: root, FlowCount: 1}},
		detail: scenarios.Detail{
			Summary: scenarios.Summary{ID: scenarioID, Path: root, FlowCount: 1},
		},
	}
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		handlers.ModuleWithDeps(svc, runsSvc, sc),
	)
	return httpx.NewLiveServer(t, srv)
}

func writeBuiltinFlow(t *testing.T, root string) string {
	t.Helper()
	raw := testkit.ValidRawContract()
	testkit.WriteFlowJSON(t, root, "api/example/flow/flow.json", raw)
	return "example.workflow.api"
}

func TestModule_Shape(t *testing.T) {
	mod := handlers.ModuleWithDeps(nil, nil, &stubScenariosSvc{})
	require.Equal(t, "artifacts", mod.Name)
	require.NotNil(t, mod.Mount)
	require.NotEmpty(t, mod.Endpoints)
}

// TestStatus_MissingArtifacts asserts that GET on a flow with no
// generated/ tree returns status="missing" and lists every expected path.
func TestStatus_MissingArtifacts(t *testing.T) {
	root := t.TempDir()
	flowID := writeBuiltinFlow(t, root)
	live := newLive(t, root, "scn", flowID, &stubGenerator{})

	resp, body := live.Do(t, http.MethodGet, "/api/v1/flows/"+flowID+"/artifacts?root="+root, nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	rep := assertx.MustDecodeJSON[artifacts.Report](t, body)
	require.Equal(t, flowID, rep.FlowID)
	require.Equal(t, artifacts.StatusMissing, rep.Status)
	require.NotEmpty(t, rep.Missing, "every artifact should be missing for a fresh flow")
	require.GreaterOrEqual(t, len(rep.Files), 4)
}

// TestGenerate_InvokesGenerator confirms the route delegates to the
// configured Generator and returns the post-generate status.
func TestGenerate_InvokesGenerator(t *testing.T) {
	root := t.TempDir()
	flowID := writeBuiltinFlow(t, root)
	gen := &stubGenerator{files: map[string]string{
		"api/example/flow/generated/model.qnt":     "(* model *)",
		"api/example/flow/generated/artifact.json": "{}",
		"api/example/flow/generated/runtime.go":    "package generated",
		"api/example/flow/generated/replay.go":     "package generated",
	}}
	live := newLive(t, root, "scn", flowID, gen)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/flows/"+flowID+"/artifacts:generate?root="+root, nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	rep := assertx.MustDecodeJSON[artifacts.Report](t, body)
	require.Equal(t, []string{flowID}, gen.calls)
	require.Equal(t, artifacts.StatusFresh, rep.Status)
	require.Empty(t, rep.Missing)
}

// TestClear_RemovesGeneratedFiles asserts DELETE removes every file the
// generated/ tree contains and refuses to escape root.
func TestClear_RemovesGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	flowID := writeBuiltinFlow(t, root)
	gen := &stubGenerator{files: map[string]string{
		"api/example/flow/generated/runtime.go": "package generated",
	}}
	require.NoError(t, gen.Generate(context.Background(), root, flowID))

	live := newLive(t, root, "scn", flowID, &stubGenerator{})
	resp, body := live.Do(t, http.MethodDelete, "/api/v1/flows/"+flowID+"/artifacts?root="+root, nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	res := assertx.MustDecodeJSON[artifacts.ClearResult](t, body)
	require.Equal(t, flowID, res.FlowID)
	require.NotEmpty(t, res.Removed)

	// And the file is gone.
	_, err := os.Stat(filepath.Join(root, "api/example/flow/generated/runtime.go"))
	require.True(t, errors.Is(err, os.ErrNotExist))
}

// TestStatus_UnknownFlow returns 404.
func TestStatus_UnknownFlow(t *testing.T) {
	root := t.TempDir()
	writeBuiltinFlow(t, root)
	live := newLive(t, root, "scn", "example.workflow.api", &stubGenerator{})
	resp, _ := live.Do(t, http.MethodGet, "/api/v1/flows/no-such-flow/artifacts?root="+root, nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)
}
