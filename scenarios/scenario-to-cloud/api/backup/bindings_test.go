package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// TestResolveBindingsFromClosure [REQ:STC-P0-029] proves declared persistent
// data resolves to owner-provided providers and that a database can never be
// captured by the generic object-store path.
func TestResolveBindingsFromClosure(t *testing.T) {
	closure := &domain.Closure{PersistentData: []domain.ClosurePersistentData{
		{ID: "records-db", Owner: "postgres", Binding: "database:fixture_records", BackupProvider: "data-backup-manager", MigrationOwner: "scenario", DeclaredBy: "scenario:app"},
		{ID: "uploads", Owner: "app", Binding: "dir:uploads", MigrationOwner: "scenario", DeclaredBy: "scenario:app"},
		{ID: "cache-db", Owner: "app", Binding: "sqlite:/srv/app/state/cache.sqlite", MigrationOwner: "resource", DeclaredBy: "scenario:app"},
	}}
	in := BindingInputs{Workdir: "/root/Vrooli", Hooks: map[string]domain.DataHook{"app": {Tool: "app-cli", Argv: []string{"maintenance", "enter"}}}, ReleaseHooks: map[string]domain.DataHook{"app": {Tool: "app-cli", Argv: []string{"maintenance", "exit"}}}}
	bindings, err := ResolveBindings(closure, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 3 || bindings[0].ID != "cache-db" || bindings[0].Provider != domain.BackupProviderSQLite {
		t.Fatalf("bindings = %+v", bindings)
	}
	records := bindings[1]
	if records.Provider != domain.BackupProviderPostgres || records.Locator != "fixture_records" || records.Quiesce == nil || records.Quiesce.Argv[1] != "enter" || records.Release == nil {
		t.Fatalf("records = %+v", records)
	}
	uploads := bindings[2]
	if uploads.Provider != domain.BackupProviderObjectStore || uploads.Locator != "/root/Vrooli/scenarios/app/uploads" {
		t.Fatalf("uploads = %+v", uploads)
	}
	if bindings[0].Quiesce != nil {
		t.Fatalf("resource-owned migration must not inherit the scenario hook: %+v", bindings[0])
	}
	if err := ValidateBindings(bindings); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"database:", "tarball:/x", "dir:../escape", "sqlite:relative.db", "application:hooks-missing"} {
		_, err := ResolveBindings(&domain.Closure{PersistentData: []domain.ClosurePersistentData{{ID: "x", Owner: "app", Binding: bad, DeclaredBy: "scenario:nohooks"}}}, in)
		if !apierrors.Is(err, apierrors.CodeInvalidRequest) {
			t.Fatalf("%q must be refused: %v", bad, err)
		}
	}
	if err := ValidateBindings([]domain.DataBinding{{ID: "db", Kind: domain.DataBindingKindSQL, Provider: domain.BackupProviderObjectStore, Locator: "/var/lib/postgresql"}}); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("generic copy of a database must be refused: %v", err)
	}
	if _, err := ResolveBindings(nil, in); !apierrors.Is(err, apierrors.CodeClosureUnavailable) {
		t.Fatalf("nil closure: %v", err)
	}
	round := DomainBindings(EngineBindings(bindings))
	if round[1].Quiesce == nil || round[1].Quiesce.Tool != "app-cli" {
		t.Fatalf("round trip lost hooks: %+v", round[1])
	}
}

// TestPostgresHookMatchesResourceDeclaration [REQ:STC-P0-029] keeps the
// resource owner's declared backup argv (resources/postgres/resource.json
// deployment.backup) and the engine's PostgreSQL provider in step, so the
// database is only ever captured by its owner's database-native tooling.
func TestPostgresHookMatchesResourceDeclaration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "resources", "postgres", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Deployment struct {
			PersistentData []struct {
				ID             string `json:"id"`
				BackupProvider string `json:"backup_provider"`
			} `json:"persistent_data"`
			Backup struct {
				Provider    string          `json:"provider"`
				Consistency string          `json:"consistency"`
				Dump        domain.DataHook `json:"dump"`
				Restore     domain.DataHook `json:"restore"`
			} `json:"backup"`
		} `json:"deployment"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hook := doc.Deployment.Backup
	if hook.Provider != domain.BackupProviderPostgres || hook.Consistency != recoverypoint.ModeDatabaseNative {
		t.Fatalf("postgres declares %+v", hook)
	}
	if hook.Dump.Tool != "pg_dump" || hook.Restore.Tool != "pg_restore" {
		t.Fatalf("postgres tools %+v", hook)
	}
	if got, want := hook.Dump.Argv, recoverypoint.PostgresDumpArgv; !equalStrings(got, want) {
		t.Fatalf("dump argv %v differs from the provider's %v", got, want)
	}
	if got, want := hook.Restore.Argv, recoverypoint.PostgresRestoreArgv; !equalStrings(got, want) {
		t.Fatalf("restore argv %v differs from the provider's %v", got, want)
	}
	if len(doc.Deployment.PersistentData) == 0 || doc.Deployment.PersistentData[0].BackupProvider != domain.BackupProviderPostgres {
		t.Fatalf("postgres must declare its databases as persistent data captured by its own provider: %+v", doc.Deployment.PersistentData)
	}
	for _, argv := range [][]string{hook.Dump.Argv, hook.Restore.Argv} {
		for _, arg := range argv {
			if arg == "--password" || len(arg) > 0 && arg[0] != '-' && arg[0] != '{' {
				t.Fatalf("argv %v must carry placeholders and flags only, never a credential", argv)
			}
		}
	}
}

// TestDBMRegistrarUsesArgvAndReportsUnreachable [REQ:STC-P0-029] proves the
// backup owner is reached through typed argv and that an unreachable owner
// is a typed backup_provider_unavailable, never a silent skip.
func TestDBMRegistrarUsesArgvAndReportsUnreachable(t *testing.T) {
	var calls [][]string
	reg := DBMRegistrar{Runner: func(_ context.Context, tool string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{tool}, args...))
		return []byte(`{"target":{"id":"t-42"}}`), nil
	}}
	binding := domain.DataBinding{ID: "records-db", Kind: domain.DataBindingKindSQL, Provider: domain.BackupProviderPostgres, Locator: "fixture_records"}
	ref, err := reg.Register(context.Background(), "app", binding)
	if err != nil || ref != "data-backup-manager:t-42" {
		t.Fatalf("ref=%q err=%v", ref, err)
	}
	want := []string{"data-backup-manager", "targets", "register", "--owner", "app", "--name", "records-db", "--kind", "postgres", "--locator", "fixture_records", "--json"}
	if len(calls) != 1 || !equalStrings(calls[0], want) {
		t.Fatalf("argv = %v", calls)
	}
	down := DBMRegistrar{Runner: func(context.Context, string, ...string) ([]byte, error) {
		return []byte("connection refused"), os.ErrNotExist
	}}
	_, err = down.Register(context.Background(), "app", binding)
	typed := apierrors.As(err)
	if typed == nil || typed.Code != apierrors.CodeBackupProviderUnavailable || !typed.Retryable || typed.Status() != 503 {
		t.Fatalf("unreachable owner: %+v", typed)
	}
	svc := &Service{Repo: nil, Registrar: down}
	_ = svc
}
