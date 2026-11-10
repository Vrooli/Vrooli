package lifecycleinference_test

import (
	"os"
	"path/filepath"
	"runtime"
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
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, got.Class, got.Reason)
	require.True(t, got.ScanComplete)
	require.Contains(t, got.Scope, "production Go")
	require.Greater(t, got.FilesScanned, 0)
	require.GreaterOrEqual(t, len(got.Evidence), 2, "retain a value-flow origin and the resolved database/sql sink")
	require.Contains(t, got.Evidence[len(got.Evidence)-1].Text, "database/sql.ExecContext")
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
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), "../../../../.."))
	require.NoError(t, err)
	classifier := lifecycleinference.NewPersistenceClassifier(root)
	attempt := classifier.Classify(lifecycleCandidate("device-studio", "api/internal/jobs", "Attempt", "State"))
	computed := classifier.Classify(lifecycleCandidate("device-studio", "api/internal/devices", "Compatibility", "Status"))
	require.Equal(t, lifecycleinference.ClassPersistedLifecycle, attempt.Class, attempt.Reason, attempt.Evidence)
	require.True(t, attempt.ScanComplete)
	require.Equal(t, lifecycleinference.ClassUnknown, computed.Class, computed.Reason,
		"computed Compatibility.Status is a labeled false positive control and must not be promoted by field-name or result-shape similarity")
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
