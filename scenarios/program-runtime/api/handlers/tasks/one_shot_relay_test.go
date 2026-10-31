package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	libraryH "program-runtime/handlers/library"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	taskstore "program-runtime/internal/tasks"
)

func TestSealedKernelCheckpointPreservesVerifiedOwnerWithoutBearerDisclosure(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema(), programs.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}
	store, repo := taskstore.NewStore(db), library.NewRepository(db)
	declaration := []byte(`{"name":"fixture.checkpoint","version":"1","inputs":{},"budget":{"wall_ms":10000,"output_bytes":4096},"verbs":["learn.task"]}`)
	source := `import json
import bridge_transport
assert not hasattr(bridge_transport, "AGENT_IDENTITY")
assert not hasattr(bridge_transport, "AGENT_OPERATION")
identity = learn.task(scope="fixture-usage", operation="fixture.checkpoint", key="checkpoint/v1")
print(json.dumps({"attempt_id": identity["attempt_id"], "task_id": identity["task_id"]}))`
	contract := archivedFixture(t, repo, "fixture.checkpoint", source, declaration)
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-owner-proof", Claims: &cliutil.VerifiedClaims{RunID: "owner-run"}}, nil
	})
	manager := sessions.NewManager(sessions.Options{})
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	require.NoError(t, err)
	runner := programs.NewSubprocessRunnerWithBindings(kernel, nil, server.URL+"/internal/program-runtime/bindings/execute", "")
	defer runner.Close()
	runner.SetLibraryProvider(func() []programs.LibrarySpec {
		return []programs.LibrarySpec{{Name: contract.Name, Scenario: contract.Scenario, Contract: true, Current: true, Source: contract.Source, Declaration: contract.Declaration, Digest: contract.Digest}}
	})
	service := programs.NewService(programs.Options{Store: db, Runner: runner, AuthorizeSession: manager.AuthorizeSubmission})
	bridge := &Bridge{Store: store, Contracts: contracts.NewIndex(), Library: repo, Sessions: manager, Programs: service}
	mux.Handle("/internal/program-runtime/tasks/", provenance.Middleware(verify)(bridge))
	var completed *programsv1.Program
	var sessionID string
	submit := provenance.Middleware(verify)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, admissionCtx, createErr := manager.CreateOneShot(r.Context(), "checkpoint", "", nil, 0, 0, 0, 0)
		if createErr != nil {
			t.Error(createErr)
			return
		}
		sessionID = session.ID
		completed, err = service.Submit(admissionCtx, session.ID, source, programsv1.Provenance_PROVENANCE_AGENT, false)
		if err != nil {
			t.Error(err)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodPost, "/submit", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-owner-proof")
	response := httptest.NewRecorder()
	submit.ServeHTTP(response, request)
	require.Equal(t, http.StatusAccepted, response.Code)
	require.NotNil(t, completed)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, completed.Status, completed.FailureDetail)
	require.Equal(t, "owner-run", completed.CallerRunId)
	require.NotContains(t, completed.Stdout, "synthetic-owner-proof")
	var output map[string]string
	require.NoError(t, json.Unmarshal([]byte(completed.Stdout), &output))
	record, err := store.Get(ctx, output["attempt_id"])
	require.NoError(t, err)
	require.Equal(t, completed.Id, record.ProgramID)
	_, err = service.Submit(ctx, sessionID, "print(2)", programsv1.Provenance_PROVENANCE_AGENT, false)
	require.Error(t, err, "a later caller cannot reuse the sealed process")
}

// The public declared-program owner must create and consume the sealed session;
// exercising the manager directly alone would not establish this integration.
func TestDeclaredProgramUsesSealedOwnerRelayForLearningCheckpoint(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	db.SetMaxOpenConns(1)
	for _, schema := range []string{taskstore.Schema(), library.Schema(), programs.Schema()} {
		_, err := db.Exec(schema)
		require.NoError(t, err)
	}
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(root))
	contract, found := index.Get("program-runtime", "learn-verbs-example")
	require.True(t, found)
	require.Empty(t, contract.ValidationError)
	store, repo := taskstore.NewStore(db), library.NewRepository(db)
	manager := sessions.NewManager(sessions.Options{})
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	kernel := filepath.Join(root, "scenarios", "program-runtime", "kernel", "host", "engine.py")
	runner := programs.NewSubprocessRunnerWithBindings(kernel, nil, server.URL+"/internal/program-runtime/bindings/execute", "")
	defer runner.Close()
	runner.SetLibraryProvider(func() []programs.LibrarySpec {
		return []programs.LibrarySpec{{Name: contract.Name, Scenario: contract.Scenario, Contract: true, Current: true, Source: contract.Source, Declaration: contract.Declaration, Digest: contract.Digest}}
	})
	service := programs.NewService(programs.Options{Store: db, Runner: runner, AuthorizeSession: manager.AuthorizeSubmission})
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-declared-proof", Claims: &cliutil.VerifiedClaims{RunID: "declared-owner"}}, nil
	})
	mux.Handle("/internal/program-runtime/tasks/", provenance.Middleware(verify)(&Bridge{Store: store, Contracts: index, Library: repo, Sessions: manager, Programs: service, RepoRoot: root}))
	declared := libraryH.DeclaredRunner(nil, index, libraryH.RunDependencies{Repository: repo, Sessions: manager, Programs: service})
	var result *libraryv1.RunDeclaredProgramResponse
	var runErr error
	submit := provenance.Middleware(verify)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, err := declared.RunDeclaredProgram(r.Context(), connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{Name: contract.ID, ExpectedDigest: contract.Digest, Provenance: programsv1.Provenance_PROVENANCE_AGENT}))
		runErr = err
		if response != nil {
			result = response.Msg
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodPost, "/submit", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-declared-proof")
	submit.ServeHTTP(httptest.NewRecorder(), request)
	require.NoError(t, runErr)
	require.NotNil(t, result)
	require.True(t, result.Terminal)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, result.Program.Status, result.Program.FailureDetail)
	require.Equal(t, "declared-owner", result.Program.CallerRunId)
	var output map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Program.Stdout), &output))
	identity := output["signals"].(map[string]any)["identity"].(map[string]any)
	record, err := store.Get(ctx, identity["attempt_id"].(string))
	require.NoError(t, err)
	require.Equal(t, result.Program.Id, record.ProgramID)
	require.NotContains(t, result.Program.Stdout, "synthetic-declared-proof")
}
