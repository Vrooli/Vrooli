package content

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCreatesDeclaredDatabase(t *testing.T) {
	h, stdout, _ := newHandlers(t, &fakeRunner{}, map[string]string{
		"POSTGRES_HOST": "127.0.0.1", "POSTGRES_PORT": "5433",
		"POSTGRES_USER": "vrooli", "POSTGRES_PASSWORD": "secret",
	})
	var request DatabaseEnsureRequest
	h.EnsureDatabase = func(_ context.Context, got DatabaseEnsureRequest) (bool, error) {
		request = got
		return true, nil
	}
	payload := base64.StdEncoding.EncodeToString([]byte(`{"database":"vrooli_declared_app"}`))

	if err := h.Ensure([]string{"--config-base64", payload}); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if request.Database != "vrooli_declared_app" || request.Owner != "vrooli" {
		t.Fatalf("ensure request = %#v", request)
	}
	if request.Host != "127.0.0.1" || request.Port != "5433" || request.Password != "secret" {
		t.Fatalf("connection request = %#v", request)
	}
	if !strings.Contains(stdout.String(), `"vrooli_declared_app" created`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEnsureIgnoresNonDatabaseConfig(t *testing.T) {
	runner := &fakeRunner{}
	h, stdout, _ := newHandlers(t, runner, nil)
	payload := base64.StdEncoding.EncodeToString([]byte(`{"schema":"scenario_name"}`))

	if err := h.Ensure([]string{"--config-base64", payload}); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
	if !strings.Contains(stdout.String(), "nothing to do") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEnsureRejectsInvalidDatabaseIdentifier(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	payload := base64.StdEncoding.EncodeToString([]byte(`{"database":"bad-name"}`))
	if err := h.Ensure([]string{"--config-base64", payload}); err == nil || !strings.Contains(err.Error(), "invalid identifier") {
		t.Fatalf("Ensure error = %v, want invalid identifier", err)
	}
}

func TestEnsureRequiresConfig(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	if err := h.Ensure(nil); err == nil || !strings.Contains(err.Error(), "--config-base64 is required") {
		t.Fatalf("Ensure error = %v, want required config", err)
	}
}

// --- fake runner -------------------------------------------------------------

type call struct {
	container string
	args      []string
	stdin     string
	env       []string
}

type fakeRunner struct {
	calls []call
	// responses indexed by call number
	stdouts [][]byte
	stderrs [][]byte
	errs    []error
}

func (f *fakeRunner) Run(_ context.Context, container string, args []string, stdin io.Reader, env []string) ([]byte, []byte, error) {
	data := []byte{}
	if stdin != nil {
		b, _ := io.ReadAll(stdin)
		data = b
	}
	idx := len(f.calls)
	f.calls = append(f.calls, call{container: container, args: args, stdin: string(data), env: env})
	var stdout, stderr []byte
	var err error
	if idx < len(f.stdouts) {
		stdout = f.stdouts[idx]
	}
	if idx < len(f.stderrs) {
		stderr = f.stderrs[idx]
	}
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	return stdout, stderr, err
}

func newHandlers(t *testing.T, runner Runner, env map[string]string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &Handlers{
		Runner: runner,
		GetEnv: func(k string) string {
			if env == nil {
				return ""
			}
			return env[k]
		},
		Stdout:    stdout,
		Stderr:    stderr,
		LookupDir: listSQLFiles,
	}, stdout, stderr
}

// --- Execute -----------------------------------------------------------------

func TestExecute_SQL(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{
		"POSTGRES_USER":     "vrooli",
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_DB":       "vrooli",
	})
	if err := h.Execute([]string{"--sql", "SELECT 1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 runner call, got %d", len(runner.calls))
	}
	c := runner.calls[0]
	if c.container != "vrooli-postgres-main" {
		t.Errorf("container = %q, want vrooli-postgres-main", c.container)
	}
	mustContainAll(t, c.args, []string{"psql", "-U", "vrooli", "-d", "vrooli", "-c", "SELECT 1", "-v", "ON_ERROR_STOP=1"})
	if !containsString(c.env, "PGPASSWORD=secret") {
		t.Errorf("env missing PGPASSWORD=secret: %v", c.env)
	}
}

func TestExecute_FileStreamsStdin(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "schema.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE t();"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.Execute([]string{"--file", path}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if runner.calls[0].stdin != "CREATE TABLE t();" {
		t.Errorf("stdin = %q, want CREATE TABLE t();", runner.calls[0].stdin)
	}
	if containsString(runner.calls[0].args, "-c") {
		t.Errorf("file mode should not pass -c; got %v", runner.calls[0].args)
	}
}

func TestExecute_Instance(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.Execute([]string{"--instance", "analytics", "--sql", "SELECT 1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if runner.calls[0].container != "vrooli-postgres-analytics" {
		t.Errorf("container = %q, want vrooli-postgres-analytics", runner.calls[0].container)
	}
}

func TestExecute_RequiresFileOrSQL(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.Execute([]string{})
	if err == nil || !strings.Contains(err.Error(), "--file or --sql") {
		t.Fatalf("expected --file/--sql error, got %v", err)
	}
}

func TestExecute_MutuallyExclusive(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.Execute([]string{"--file", "x", "--sql", "y"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
}

func TestExecute_MissingFile(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.Execute([]string{"--file", "/does/not/exist.sql"})
	if err == nil || !strings.Contains(err.Error(), "open SQL file") {
		t.Fatalf("expected open error, got %v", err)
	}
}

// --- CreateDatabase ----------------------------------------------------------

func TestCreateDatabase_Success(t *testing.T) {
	runner := &fakeRunner{}
	h, stdout, _ := newHandlers(t, runner, nil)
	if err := h.CreateDatabase([]string{"myapp"}); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	got := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(got, "CREATE DATABASE myapp OWNER vrooli;") {
		t.Errorf("expected CREATE DATABASE in args; got %q", got)
	}
	if !strings.Contains(stdout.String(), `"myapp" created`) {
		t.Errorf("stdout missing success message: %q", stdout.String())
	}
}

func TestCreateDatabase_AlreadyExistsIsOK(t *testing.T) {
	runner := &fakeRunner{
		stderrs: [][]byte{[]byte(`ERROR:  database "myapp" already exists`)},
		errs:    []error{errors.New("exit status 1")},
	}
	h, stdout, _ := newHandlers(t, runner, nil)
	if err := h.CreateDatabase([]string{"myapp"}); err != nil {
		t.Fatalf("CreateDatabase should ignore already-exists: %v", err)
	}
	if !strings.Contains(stdout.String(), "already exists") {
		t.Errorf("stdout missing already-exists note: %q", stdout.String())
	}
}

func TestCreateDatabase_OtherErrorsPropagate(t *testing.T) {
	runner := &fakeRunner{
		stderrs: [][]byte{[]byte(`could not connect to server`)},
		errs:    []error{errors.New("exit status 2")},
	}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.CreateDatabase([]string{"myapp"}); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestCreateDatabase_RejectsBadIdentifier(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.CreateDatabase([]string{"DROP TABLE users;"})
	if err == nil || !strings.Contains(err.Error(), "invalid identifier") {
		t.Fatalf("expected invalid identifier error, got %v", err)
	}
}

func TestCreateDatabase_RequiresDBName(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.CreateDatabase([]string{})
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestCreateDatabase_CustomOwner(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.CreateDatabase([]string{"--owner", "app_user", "myapp"}); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	got := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(got, "OWNER app_user") {
		t.Errorf("expected custom owner; got %q", got)
	}
}

// --- Add ---------------------------------------------------------------------

func TestAdd_FileRunsExecute(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "seed.sql")
	if err := os.WriteFile(path, []byte("INSERT"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.Add([]string{path}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if runner.calls[0].stdin != "INSERT" {
		t.Errorf("stdin = %q, want INSERT", runner.calls[0].stdin)
	}
}

func TestAdd_DirectoryRunsFilesInOrder(t *testing.T) {
	tmp := t.TempDir()
	for name, content := range map[string]string{
		"01_schema.sql":  "A",
		"02_seed.sql":    "B",
		"03_indexes.sql": "C",
		"README.txt":     "ignored",
	} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.Add([]string{tmp}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(runner.calls))
	}
	if runner.calls[0].stdin != "A" || runner.calls[1].stdin != "B" || runner.calls[2].stdin != "C" {
		t.Errorf("files not executed in alpha order; got stdins %q %q %q",
			runner.calls[0].stdin, runner.calls[1].stdin, runner.calls[2].stdin)
	}
}

func TestAdd_InitCreatesDatabaseFirst(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "schema.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE x();"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{
		"POSTGRES_DB": "admin_db",
	})
	if err := h.Add([]string{"--database", "myapp", "--init", path}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 calls (create + execute), got %d", len(runner.calls))
	}
	// First call: create-database issued against admin_db
	got0 := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(got0, "CREATE DATABASE myapp") {
		t.Errorf("first call missing CREATE DATABASE: %q", got0)
	}
	if !containsString(runner.calls[0].args, "admin_db") {
		t.Errorf("create should run against admin_db; args: %v", runner.calls[0].args)
	}
	// Second call: execute file against myapp
	if !containsString(runner.calls[1].args, "myapp") {
		t.Errorf("execute should target myapp; args: %v", runner.calls[1].args)
	}
}

// --- Get ---------------------------------------------------------------------

func TestGet_PrintsConnectionInfo(t *testing.T) {
	h, stdout, _ := newHandlers(t, &fakeRunner{}, map[string]string{
		"POSTGRES_USER":     "vrooli",
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_DB":       "myapp",
	})
	if err := h.Get([]string{}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Database:          myapp") {
		t.Errorf("expected Database: myapp; got %q", out)
	}
	if !strings.Contains(out, "postgres://vrooli:secret@localhost:5433/myapp") {
		t.Errorf("expected connection string; got %q", out)
	}
}

func TestGet_AsEnv(t *testing.T) {
	h, stdout, _ := newHandlers(t, &fakeRunner{}, map[string]string{
		"POSTGRES_USER":     "vrooli",
		"POSTGRES_PASSWORD": "s'ecret",
		"POSTGRES_DB":       "myapp",
	})
	if err := h.Get([]string{"--as-env"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "export POSTGRES_USER='vrooli'") {
		t.Errorf("missing USER export; got %q", out)
	}
	if !strings.Contains(out, `export POSTGRES_PASSWORD='s'\''ecret'`) {
		t.Errorf("missing quoted PASSWORD export; got %q", out)
	}
	if !strings.Contains(out, "export POSTGRES_URL=") {
		t.Errorf("missing URL export; got %q", out)
	}
}

func TestGet_PositionalDatabase(t *testing.T) {
	h, stdout, _ := newHandlers(t, &fakeRunner{}, map[string]string{
		"POSTGRES_USER":     "vrooli",
		"POSTGRES_PASSWORD": "s",
	})
	if err := h.Get([]string{"calendar_system"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(stdout.String(), "Database:          calendar_system") {
		t.Errorf("positional database not applied; got %q", stdout.String())
	}
}

// --- List --------------------------------------------------------------------

func TestList_IssuesSelect(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, nil)
	if err := h.List([]string{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	got := strings.Join(runner.calls[0].args, " ")
	if !strings.Contains(got, "pg_database") {
		t.Errorf("expected pg_database query; got %q", got)
	}
}

// --- Remove ------------------------------------------------------------------

func TestRemove_IssuesDrop(t *testing.T) {
	h, stdout, _ := newHandlers(t, &fakeRunner{}, nil)
	var request DatabaseEnsureRequest
	h.DropDatabase = func(_ context.Context, got DatabaseEnsureRequest) error {
		request = got
		return nil
	}
	if err := h.Remove([]string{"myapp"}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if request.Database != "myapp" || request.Owner != "vrooli" {
		t.Errorf("drop request = %#v", request)
	}
	if !strings.Contains(stdout.String(), `"myapp" dropped`) {
		t.Errorf("missing dropped message; got %q", stdout.String())
	}
}

func TestRemove_RejectsBadIdentifier(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeRunner{}, nil)
	err := h.Remove([]string{"foo; DROP TABLE users;"})
	if err == nil || !strings.Contains(err.Error(), "invalid identifier") {
		t.Fatalf("expected invalid identifier error, got %v", err)
	}
}

// --- helpers -----------------------------------------------------------------

func containsString(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}

func mustContainAll(t *testing.T, have []string, want []string) {
	t.Helper()
	for _, w := range want {
		if !containsString(have, w) {
			t.Errorf("args missing %q; got %v", w, have)
		}
	}
}

// Ensure Commands() registers every subcommand.
func TestCommands_RegistersAllSubcommands(t *testing.T) {
	group := Commands(nil)
	wantNames := []string{"execute", "create-database", "add", "get", "list", "remove"}
	got := map[string]bool{}
	for _, c := range group.Subcommands {
		got[c.Name] = true
		if c.Run == nil {
			t.Errorf("subcommand %q has nil Run", c.Name)
		}
	}
	for _, n := range wantNames {
		if !got[n] {
			t.Errorf("missing subcommand %q", n)
		}
	}
}
