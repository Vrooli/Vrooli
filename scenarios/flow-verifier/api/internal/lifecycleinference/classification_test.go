package lifecycleinference_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/lifecycleinference"
	"github.com/stretchr/testify/require"
)

func TestClassifyPersistenceRequiresTypedDurableValueFlow(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/device-studio/api/internal/jobs/attempt.go", `package jobs
type AttemptState string
const ( Pending AttemptState = "pending"; Running AttemptState = "running" )
type Attempt struct { State AttemptState }
type Job struct { Attempts []Attempt }
func (a *Attempt) Start() { a.State = Running }
`)
	write("scenarios/device-studio/api/internal/jobs/sqlite.go", `package jobs
import (
  "context"
  "database/sql"
  "encoding/json"
)
func encode(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func persistAttempts(ctx context.Context, conn *sql.Conn, job Job) error {
  for _, attempt := range job.Attempts {
    record, err := encode(attempt)
    if err != nil { return err }
    _, err = conn.ExecContext(ctx, "INSERT INTO attempts(record) VALUES (?)", record)
    if err != nil { return err }
  }
  return nil
}

`)
	candidate := lifecycleCandidate("device-studio", "api/internal/jobs", "Attempt", "State")
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason, got.Evidence)
	require.True(t, got.ScanComplete)
	require.Contains(t, got.Scope, "production Go")
	require.Greater(t, got.FilesScanned, 0)
	require.GreaterOrEqual(t, len(got.Evidence), 2, "retain a value-flow origin and the resolved database/sql sink")
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
}

func TestClassifyPersistenceDoesNotTreatSQLControlExecAsDurableWrite(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/state.go", `package domain
type State string
const Pending State = "pending"
type Attempt struct { State State }
`)
	write("scenarios/alpha/api/domain/lock.go", `package domain
import (
  "context"
  "database/sql"
)
func acquireLock(ctx context.Context, db *sql.DB, attempt Attempt) error {
  _, err := db.ExecContext(ctx, "/* lock control */ SELECT pg_advisory_lock(1)", attempt.State)
  return err
}
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, "a constant SELECT used for lock/control flow does not persist the candidate state: "+got.Reason)
}

func TestClassifyPersistenceRecognizesExplicitResponseAndReadModel(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/deployment-manager/api/deployments/response.go", `package deployments
import "connectrpc.com/connect"
type DeployDesktopResponse struct { Status string }
func respond() *connect.Response[DeployDesktopResponse] {
  result := DeployDesktopResponse{Status: "ready"}
  return connect.NewResponse(&result)
}
`)
	response := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("deployment-manager", "api/deployments", "DeployDesktopResponse", "Status"))
	require.Equal(t, lifecycleinference.ClassTransientResult, response.Class, response.Reason)
	require.Contains(t, response.Evidence[0].Text, "Connect response")

	write("scenarios/deployment-manager/api/deployments/projection.go", `package deployments
import (
  "context"
  "database/sql"
  "connectrpc.com/connect"
)
type RunProjection struct { Status string }
func projection(ctx context.Context, db *sql.DB) *connect.Response[RunProjection] {
  rows, _ := db.QueryContext(ctx, "SELECT status FROM runs")
  var view RunProjection
  rows.Next()
  rows.Scan(&view.Status)
  return connect.NewResponse(&view)
}
`)
	projection := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("deployment-manager", "api/deployments", "RunProjection", "Status"))
	require.Equal(t, lifecycleinference.ClassDerivedReadModel, projection.Class, projection.Reason)
	require.GreaterOrEqual(t, len(projection.Evidence), 2)
}

func TestClassifyPersistenceAllowsEnumOnlyOnlyForUnusedUnannotatedVocabulary(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/state.go", `package domain
type State string
const Pending State = "pending"
type Labels struct { State State }
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Labels", "State"))
	require.Equal(t, lifecycleinference.ClassEnumOnly, got.Class, got.Reason)
	require.True(t, got.ScanComplete)
	require.Contains(t, got.Reason, "other-language paths are outside")

	write("scenarios/alpha/api/domain/state.go", "package domain\ntype State string\nconst Pending State = \"pending\"\ntype Labels struct { State State `gorm:\"column:state\"` }\n")
	annotated := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Labels", "State"))
	require.Equal(t, lifecycleinference.ClassUnknown, annotated.Class, "storage annotations invalidate an absence-based enum-only claim")
}

func TestClassifyPersistenceRejectsTestSinksNameCollisionsAndUnsupportedHelpers(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/state.go", `package domain
type State string
const Pending State = "pending"
type Attempt struct { State State }
func NewAttempt() Attempt { return Attempt{State: Pending} }
type fakeRepo struct{}
func (fakeRepo) Save(*Attempt) error { return nil }
func saveThroughHelper(repo fakeRepo, attempt *Attempt) error { return repo.Save(attempt) }
`)
	write("scenarios/alpha/api/domain/state_test.go", `package domain
type fakeStore struct{}
func (fakeStore) Save(*Attempt) error { return nil }
func TestSaveIsNotProductionPersistence(t *testing.T) { _ = fakeStore{}.Save(&Attempt{}) }
`)
	write("scenarios/alpha/api/domain/other.go", `package domain
import (
  "context"
  "database/sql"
  foreign "third.party/example/domain"
)
type Client struct{}
func (Client) Update(any) error { return nil }
func collision(ctx context.Context, conn *sql.Conn, client Client, other foreign.Attempt) {
  client.Update(other.State)
  conn.ExecContext(ctx, "INSERT INTO unrelated(status) VALUES (?)", other.State)
}
`)
	candidate := lifecycleCandidate("alpha", "api/domain", "Attempt", "State")
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, got.Reason)
	require.NotContains(t, got.Reason, "persisted_lifecycle")

	write("scenarios/alpha/api/domain/orm.go", "package domain\ntype ORMRow struct { State string `db:\"state\"` }\n")
	annotated := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassUnknown, annotated.Class)
}

func TestClassifyPersistenceTracksInferredAliasesAndPointerValues(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/attempt.go", `package domain
import (
  "context"
  "database/sql"
  "encoding/json"
)
type State string
const Pending State = "pending"
type Attempt struct { State State }
func encode(value any) ([]byte, error) { return json.Marshal(value) }
func persist(ctx context.Context, conn *sql.Tx, input *Attempt) error {
  copied := *input
  alias := &copied
  encoded, err := encode(alias)
  if err != nil { return err }
  _, err = conn.ExecContext(ctx, "INSERT INTO attempts(record) VALUES (?)", encoded)
  return err
}
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	require.GreaterOrEqual(t, len(got.Evidence), 2)
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
}

func TestClassifyPersistenceRequiresTheLifecycleFieldAtTheSink(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/attempt.go", `package domain
import (
  "context"
  "database/sql"
)
type Attempt struct { State string; Payload string }
func persistPayload(ctx context.Context, db *sql.DB, attempt Attempt) error {
  _, err := db.ExecContext(ctx, "INSERT INTO attempts(payload) VALUES (?)", attempt.Payload)
  return err
}
func loadPayload(ctx context.Context, db *sql.DB, attempt *Attempt) error {
  row := db.QueryRowContext(ctx, "SELECT payload FROM attempts")
  return row.Scan(&attempt.Payload)
}
`)
	candidate := lifecycleCandidate("alpha", "api/domain", "Attempt", "State")
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, "sibling-field writes and scans do not establish State persistence")

	write("scenarios/alpha/api/domain/attempt.go", `package domain
import (
  "context"
  "database/sql"
)
type Attempt struct { State string; Payload string }
func persistState(ctx context.Context, db *sql.DB, attempt Attempt) error {
  _, err := db.ExecContext(ctx, "INSERT INTO attempts(state) VALUES (?)", attempt.State)
  return err
}
`)
	persisted := lifecycleinference.NewPersistenceClassifier(root).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, persisted.Class, persisted.Reason)
}

func TestClassifyPersistenceFollowsExactCrossPackageImportAlias(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/go.mod", "module alpha/api\n")
	write("scenarios/alpha/api/domain/attempt.go", `package domain
type State string
type Attempt struct { State State }
`)
	write("scenarios/alpha/api/storage/store.go", `package storage
import (
  "context"
  "database/sql"
  domain "alpha/api/domain"
)
func persist(ctx context.Context, db *sql.DB, attempt *domain.Attempt) error {
  _, err := db.ExecContext(ctx, "INSERT INTO attempts(state) VALUES (?)", attempt.State)
  return err
}
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
}

func TestClassifyPersistenceDeviceStudioControlPairAgainstRepositorySource(t *testing.T) {
	root := repositoryRootForFixture(t)
	classifier := lifecycleinference.NewPersistenceClassifier(root)
	attempt := classifier.Classify(lifecycleCandidate("device-studio", "api/internal/jobs", "Attempt", "State"))
	computed := classifier.Classify(lifecycleCandidate("device-studio", "api/internal/devices", "Compatibility", "Status"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, attempt.Class, attempt.Reason, attempt.Evidence)
	require.True(t, attempt.ScanComplete)
	require.Equal(t, lifecycleinference.ClassUnknown, computed.Class, computed.Reason,
		"computed Compatibility.Status is a labeled false positive control and must not be promoted by field-name or result-shape similarity")
}

func TestClassifyPersistenceAgentManagerWorkflowExecutionThroughEmbeddedTx(t *testing.T) {
	root := repositoryRootForFixture(t)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("agent-manager", "api/internal/domain", "WorkflowExecution", "Status"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason, got.Evidence)
	require.True(t, got.ScanComplete)
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
}

func TestClassifyPersistenceRecognizesEmbeddedSQLTransactionWrapper(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/go.mod", "module alpha/api\n\ngo 1.24\n")
	write("scenarios/alpha/api/domain/workflow.go", `package domain
type Status string
const ( Pending Status = "pending"; Running Status = "running" )
type WorkflowExecution struct { ID string; Status Status }
`)
	write("scenarios/alpha/api/internal/database/connection.go", `package database
import "database/sql"
type Tx struct { *sql.Tx }
`)
	write("scenarios/alpha/api/internal/database/repository.go", `package database
import (
  "context"
  "alpha/api/domain"
)
func persistExecution(ctx context.Context, tx *Tx, execution *domain.WorkflowExecution) error {
  _, err := tx.ExecContext(ctx, "UPDATE workflow_executions SET status=? WHERE id=?", execution.Status, execution.ID)
  return err
}
`)

	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "WorkflowExecution", "Status"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	require.True(t, got.ScanComplete)
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
}

func TestClassifyPersistenceDoesNotTrustOpaqueExecContextWrapper(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/go.mod", "module alpha/api\n\ngo 1.24\n")
	write("scenarios/alpha/api/domain/workflow.go", `package domain
type Status string
const ( Pending Status = "pending"; Running Status = "running" )
type WorkflowExecution struct { ID string; Status Status }
`)
	write("scenarios/alpha/api/internal/database/connection.go", `package database
import "context"
type Tx struct{}
func (*Tx) ExecContext(context.Context, string, ...any) (int, error) { return 0, nil }
`)
	write("scenarios/alpha/api/internal/database/repository.go", `package database
import (
  "context"
  "alpha/api/domain"
)
func persistExecution(ctx context.Context, tx *Tx, execution *domain.WorkflowExecution) error {
  _, err := tx.ExecContext(ctx, "UPDATE workflow_executions SET status=? WHERE id=?", execution.Status, execution.ID)
  return err
}
`)

	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "WorkflowExecution", "Status"))
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, "a same-named custom method does not prove a SQL sink")
}

func repositoryRootForFixture(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	require.NoError(t, err)
	for {
		marker := filepath.Join(root, "scenarios", "device-studio", "api", "internal", "jobs", "types.go")
		if _, err := os.Stat(marker); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	t.Fatalf("repository root not found from test working directory")
	return ""
}

func TestClassifyPersistenceReturnsUnknownOnIncompleteOrUnresolvedSource(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/domain/attempt.go", `package domain
import "database/sql"
type Attempt struct { State string }
func persist(db *sql.DB, a Attempt) { db.Exec("INSERT INTO attempts(state) VALUES (?)", a.State) }
`)
	write("scenarios/alpha/api/broken.go", "package broken\nfunc unfinished( {\n")
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class)
	require.False(t, got.ScanComplete)
	require.Contains(t, got.Reason, "incomplete")
	require.NotEmpty(t, got.Evidence)

	outOfScope := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "cli/domains", "Attempt", "State"))
	require.Equal(t, lifecycleinference.ClassUnknown, outOfScope.Class)
	require.Contains(t, outOfScope.Reason, "outside scanned API")
}

func lifecycleCandidate(scenario, packagePath, typeName, fieldName string) lifecycleinference.Candidate {
	return lifecycleinference.Candidate{
		ScenarioID: scenario, Package: packagePath, TypeName: typeName, FieldName: fieldName,
		Declaration: lifecycleinference.Evidence{Path: "scenarios/" + scenario + "/" + packagePath + "/state.go", Line: 3, Text: fieldName + " string"},
		States:      []lifecycleinference.StateEvidence{{Value: "pending"}, {Value: "running"}},
	}
}

func sourceWriter(t *testing.T, root string) func(string, string) {
	t.Helper()
	return func(path, source string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(source), 0o644))
	}
}

func TestClassifyPersistenceRecognizesJSONFileStorageHelpers(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/go.mod", "module alpha\n\ngo 1.24\n")
	write("scenarios/alpha/api/domain/types.go", `package domain
import "alpha/storage"
type Session struct { Status string }
func save(session Session) error { return storage.WriteJSONAtomic("session.json", session) }
`)
	write("scenarios/alpha/api/storage/json.go", `package storage
import (
  "encoding/json"
  "os"
)
func WriteJSONAtomic(path string, value any) error {
  bytes, err := json.MarshalIndent(value, "", "  ")
  if err != nil { return err }
  file, err := os.CreateTemp("", "record-*.json")
  if err != nil { return err }
  if _, err := file.Write(bytes); err != nil { return err }
  if err := file.Close(); err != nil { return err }
  return os.Rename(file.Name(), path)
}
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Session", "Status"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	require.Contains(t, got.Reason, "durable sink")
	require.NotEmpty(t, got.Evidence)
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "alpha/storage.WriteJSONAtomic")
}

func TestClassifyPersistenceKeepsTransientJSONResponseAndUnboundJSONUnknown(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/go.mod", "module alpha\n\ngo 1.24\n")
	write("scenarios/alpha/api/domain/record.go", `package domain
import (
  "encoding/json"
  "connectrpc.com/connect"
)
type Record struct { Status string }
func encode(value Record) ([]byte, error) { return json.Marshal(value) }
func respond(value Record) *connect.Response[Record] { return connect.NewResponse(&value) }
func encodeOnly(value Record) ([]byte, error) { return encode(value) }
`)
	classifier := lifecycleinference.NewPersistenceClassifier(root)
	response := classifier.Classify(lifecycleCandidate("alpha", "api/domain", "Record", "Status"))
	require.Equal(t, lifecycleinference.ClassTransientResult, response.Class, response.Reason)
	require.NotContains(t, response.Reason, "durable sink")

	write("scenarios/alpha/api/domain/record.go", `package domain
import "encoding/json"
type Record struct { Status string }
func encode(value Record) ([]byte, error) { return json.Marshal(value) }
func encodeOnly(value Record) ([]byte, error) { return encode(value) }
`)
	unknown := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/domain", "Record", "Status"))
	require.Equal(t, lifecycleinference.ClassUnknown, unknown.Class, unknown.Reason)
	require.Contains(t, unknown.Reason, "does not prove")
}

func TestClassifyPersistenceRecognizesSwarmManagerFileStoreAndReviewRound(t *testing.T) {
	root := repositoryRootForFixture(t)
	classifier := lifecycleinference.NewPersistenceClassifier(root)
	cases := []struct {
		candidate lifecycleinference.Candidate
		ownerPath string
		helper    string
	}{
		{lifecycleCandidate("swarm-manager", "api/internal/agentsessions", "Session", "Status"), "scenarios/swarm-manager/api/internal/agentsessions/store.go", "swarm-manager/internal/storage.WriteJSONAtomic"},
		{lifecycleCandidate("swarm-manager", "api/internal/agentsessions", "Proposal", "Status"), "scenarios/swarm-manager/api/internal/agentsessions/store.go", "swarm-manager/internal/storage.WriteJSONAtomic"},
		{lifecycleCandidate("swarm-manager", "api/internal/review", "Round", "Status"), "scenarios/swarm-manager/api/internal/review/store.go", "swarm-manager/internal/attemptstore.SaveRound"},
	}
	for _, tc := range cases {
		got := classifier.Classify(tc.candidate)
		require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, tc.candidate.TypeName+"."+tc.candidate.FieldName+": "+got.Reason)
		require.True(t, got.ScanComplete)
		require.Contains(t, got.Reason, "durable sink")
		fileSink := false
		for _, evidence := range got.Evidence {
			if strings.Contains(evidence.Path, tc.ownerPath) && strings.Contains(evidence.Text, tc.helper) {
				fileSink = true
			}
		}
		require.True(t, fileSink, "expected candidate-specific file persistence evidence for %s.%s: %+v", tc.candidate.TypeName, tc.candidate.FieldName, got.Evidence)
	}
}

func TestClassifyPersistenceFollowsSameTypeReceiverFileWriter(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/alpha/api/pipeline/status.go", `package pipeline
import (
  "encoding/json"
  "os"
)
type State string
const ( Pending State = "pending"; Running State = "running" )
type Status struct { State State }
type FileStore struct{}
func (s *FileStore) writeToDisk(status *Status) error {
  data, err := json.MarshalIndent(status, "", "  ")
  if err != nil { return err }
  return os.WriteFile("status.json", data, 0644)
}

func (s *FileStore) Update(status *Status) { _ = s.writeToDisk(status) }
`)
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(lifecycleCandidate("alpha", "api/pipeline", "Status", "State"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	var evidenceText []string
	for _, evidence := range got.Evidence {
		evidenceText = append(evidenceText, evidence.Text)
	}
	require.Contains(t, strings.Join(evidenceText, " "), "JSON file persistence helper")
}
func TestClassifyPersistenceDoesNotPromoteUnrelatedNestedRecordWrite(t *testing.T) {
	root := t.TempDir()
	write := sourceWriter(t, root)
	write("scenarios/deployment-manager/api/deployments/orchestrator.go", `package deployments
import (
  "encoding/json"
  "os"
)
type DeployDesktopResponse struct { Status string }
type deployState struct { response DeployDesktopResponse; manifest string }
type Orchestrator struct{}
func (o *Orchestrator) writeManifest(ds *deployState) error {
  data, err := json.Marshal(ds.manifest)
  if err != nil { return err }
  return os.WriteFile("manifest.json", data, 0644)
}

func (o *Orchestrator) deploy(ds *deployState) error { return o.writeManifest(ds) }
`)
	candidate := lifecycleCandidate("deployment-manager", "api/deployments", "DeployDesktopResponse", "Status")
	got := lifecycleinference.NewPersistenceClassifier(root).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, got.Reason)
}

func TestClassifyNavigationSessionStatusIgnoresUnrelatedExportFileWrites(t *testing.T) {
	candidate := lifecycleCandidate("browser-automation-studio", "api/services/vision", "NavigationSession", "Status")
	got := lifecycleinference.NewPersistenceClassifier(repositoryRootForFixture(t)).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, got.Reason)
}

func TestClassifyDeploymentResponseStatusIgnoresUnrelatedManifestFileWrites(t *testing.T) {
	candidate := lifecycleCandidate("deployment-manager", "api/deployments", "DeployDesktopResponse", "Status")
	got := lifecycleinference.NewPersistenceClassifier(repositoryRootForFixture(t)).Classify(candidate)
	require.Equal(t, lifecycleinference.ClassUnknown, got.Class, got.Reason)
}
