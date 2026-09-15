package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/backup"
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

func backupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	srv, repo := newIdentityTestServer(t, "backup-"+t.Name())
	seedDeployment(t, repo, "dep-1", "demo-app", "production", "203.0.113.10", "demo.example")
	root := t.TempDir()
	srv.backupSvc = &backup.Service{
		Repo: repo, Root: filepath.Join(root, "recovery-points"),
		Keys:      recoverypoint.StaticKeys{"fixture/recovery:key": []byte("fixture-recovery-key-material")},
		Sealer:    recoverypoint.Sealer{Iterations: 1000},
		Providers: recoverypoint.Registry{domain.BackupProviderObjectStore: recoverypoint.ObjectStore{}},
		Budgets:   backup.Budgets{FreshHostRestoreSeconds: 900, RecoveryPointSeconds: 300},
	}
	srv.registerBackupRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	return srv, root
}

func doJSON(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	return rec
}

// TestRecoveryPointRoutes [REQ:STC-P0-029] [REQ:STC-P0-030] [REQ:STC-P0-031]
// proves the REST surface captures, lists, verifies and restores a recovery
// point, refuses a restore into an occupied target with
// restore_target_not_clean, refuses an incompatible rollback with a typed
// plan and never prunes a protected point.
func TestRecoveryPointRoutes(t *testing.T) {
	srv, root := backupTestServer(t)
	source := filepath.Join(root, "host-a", "uploads")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "invoice-001.txt"), []byte("invoice one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	capture := backup.CaptureRequest{
		OperationID: "op-1", Step: "data.backup",
		Bindings:      []domain.DataBinding{{ID: "uploads", Owner: "demo-app", Kind: domain.DataBindingKindFiles, Provider: domain.BackupProviderObjectStore, Locator: source}},
		ReleaseDigest: "sha256:" + strings.Repeat("a", 64), SchemaVersion: "v2", RecoveryKeyRef: "fixture/recovery:key",
		MigrationPosture: domain.MigrationPostureGreenfieldWithData, ProtectedBy: []string{"operation:op-1"},
	}
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points", capture)
	if rec.Code != http.StatusCreated {
		t.Fatalf("capture: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		RecoveryPoint domain.RecoveryPoint `json:"recovery_point"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	rp := created.RecoveryPoint
	if rp.ID != "op-1-data.backup" || !rp.Encrypted || rp.RecoveryKeyRef != "fixture/recovery:key" || rp.Checksums["uploads"].Count != 1 || !rp.Protected {
		t.Fatalf("recovery point = %+v", rp)
	}
	if strings.Contains(rec.Body.String(), "fixture-recovery-key-material") {
		t.Fatalf("response must never carry key material")
	}
	if rec := doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points", backup.CaptureRequest{OperationID: "op-2", Step: "data.backup", Bindings: capture.Bindings, MigrationPosture: domain.MigrationPostureGreenfield}); rec.Code != http.StatusBadRequest || apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeInvalidRequest {
		t.Fatalf("capture without a key reference must be refused: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/recovery-points", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), rp.ID) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, srv, http.MethodGet, "/api/v1/deployments/missing/recovery-points", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown deployment: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/recovery-points/"+rp.ID+"/verify?open=1", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"artifacts_opened":true`) {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/recovery-points/nope/verify", nil); apierrors.FromHTTP(rec.Code, rec.Body.Bytes()).Code != apierrors.CodeRecoveryPointNotFound || rec.Code != http.StatusNotFound {
		t.Fatalf("unknown point: %d %s", rec.Code, rec.Body.String())
	}

	target := filepath.Join(root, "host-b", "uploads")
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points/"+rp.ID+"/restore", backup.RestoreRequest{TargetRef: "host-b", Into: map[string]string{"uploads": target}})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"outcome":"succeeded"`) || !strings.Contains(rec.Body.String(), `"within_budgets":true`) {
		t.Fatalf("restore: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points/"+rp.ID+"/restore", backup.RestoreRequest{TargetRef: "host-b", Into: map[string]string{"uploads": target}})
	typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
	if rec.Code != http.StatusConflict || typed.Code != apierrors.CodeRestoreTargetNotClean || typed.Details["restore_receipt"] == nil {
		t.Fatalf("occupied target: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points/rollback-admission", RollbackAdmissionRequest{CurrentSchema: "v3", TargetSchema: "v2", SchemaStrategy: backup.SchemaStrategyExplicitRestore, CodeRollback: "any_predecessor"})
	typed = apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
	if rec.Code != http.StatusConflict || typed.Code != apierrors.CodeRollbackIncompatible || typed.NextAction == nil || typed.NextAction.Reference != rp.ID {
		t.Fatalf("incompatible rollback: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points/rollback-admission", RollbackAdmissionRequest{CurrentSchema: "v2", TargetSchema: "v2", SchemaStrategy: backup.SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"compatible":true`) {
		t.Fatalf("compatible rollback: %d %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-1/recovery-points/prune", PruneRequest{KeepLast: 0, MaxAgeSeconds: 1, References: backup.References{NonTerminalOperations: []string{"op-1"}}})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"delete":[]`) || !strings.Contains(rec.Body.String(), `"operation:op-1"`) {
		t.Fatalf("prune must keep the referenced point: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-1/recovery-points", nil)
	if !strings.Contains(rec.Body.String(), rp.ID) {
		t.Fatalf("protected point pruned: %s", rec.Body.String())
	}
}
