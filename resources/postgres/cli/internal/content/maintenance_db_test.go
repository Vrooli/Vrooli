package content

import (
	"strings"
	"testing"
)

// psqlDatabaseArg returns the database the runner was told to connect to
// (`-d <db>`), which is the whole subject of these tests.
func psqlDatabaseArg(t *testing.T, args []string) string {
	t.Helper()
	for i, arg := range args {
		if arg == "-d" && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("no -d argument in %q", strings.Join(args, " "))
	return ""
}

// Creating a database must not require that database to already exist. The CLI
// connected to POSTGRES_DB (default "vrooli") to issue the CREATE, so on a host
// without a `vrooli` database `create-database` could never succeed — including
// `create-database vrooli`, which had to connect to vrooli to create vrooli.
func TestCreateDatabaseConnectsToAMaintenanceDatabase(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{
		"POSTGRES_USER": "vrooli",
		"POSTGRES_DB":   "vrooli",
	})
	if err := h.CreateDatabase([]string{"vrooli"}); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if got := psqlDatabaseArg(t, runner.calls[0].args); got != "postgres" {
		t.Fatalf("connected to %q; a database being created cannot be its own connection target", got)
	}
}

// Reading the catalog is work about databases, not inside one.
func TestListConnectsToAMaintenanceDatabase(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{"POSTGRES_DB": "vrooli"})
	if err := h.List([]string{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := psqlDatabaseArg(t, runner.calls[0].args); got != "postgres" {
		t.Fatalf("list connected to %q, want the maintenance database", got)
	}
}

// An operator whose cluster names its maintenance database differently can say so.
func TestMaintenanceDatabaseIsOverridable(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{"POSTGRES_MAINTENANCE_DB": "template1"})
	if err := h.List([]string{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := psqlDatabaseArg(t, runner.calls[0].args); got != "template1" {
		t.Fatalf("maintenance database = %q, want template1", got)
	}
}

// The target database for work *inside* a database is unchanged: only the
// admin/catalog paths moved to the maintenance connection.
func TestDataOperationsStillUseTheConfiguredDatabase(t *testing.T) {
	runner := &fakeRunner{}
	h, _, _ := newHandlers(t, runner, map[string]string{"POSTGRES_DB": "vrooli_secrets_manager"})
	if err := h.Execute([]string{"--sql", "SELECT 1"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := psqlDatabaseArg(t, runner.calls[0].args); got != "vrooli_secrets_manager" {
		t.Fatalf("execute connected to %q, want the configured application database", got)
	}
}
