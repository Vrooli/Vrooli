package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/sshadapter"
)

// recordingRunner captures the connection config every command runs with.
type recordingRunner struct {
	mu       sync.Mutex
	configs  []sshadapter.ConnectionConfig
	commands []string
	answer   func(command string) sshadapter.Result
}

func (r *recordingRunner) Run(_ context.Context, cfg sshadapter.ConnectionConfig, command string, _ sshadapter.RunOptions) (sshadapter.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs = append(r.configs, cfg)
	r.commands = append(r.commands, command)
	if r.answer != nil {
		return r.answer(command), nil
	}
	return sshadapter.Result{ExitCode: 0}, nil
}

// [REQ:STC-P0-032] A production-shaped row (manifest still carrying
// target.vps.key_path, explicit_key ssh_identity) keeps working after the
// key leaves the wire contract: startup conversion binds the key file as
// vrooli/scenario-to-cloud:ssh-key, the server's resolver hands the SSH
// adapter that path, and a read-only route reaches the target with it. A
// host without a binding is reached with the ambient identity.
func TestSSHKeyBindingResolvesConnectionForLegacyProductionRow(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ssh-key-binding-resolver?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	srv := newTestServer()
	srv.repo = repo
	srv.deploymentRepo = repo
	srv.historyRecorder = repo
	srv.setupRoutes()
	const depID = "7e493e04-5297-47eb-9df9-000000000001"
	const keyPath = "/home/operator/.ssh/canary-key-9f3a"
	legacyManifest := `{
		"version": "1",
		"target": {"type": "vps", "vps": {"host": "203.0.113.42", "port": 22, "user": "root", "workdir": "/root/Vrooli", "key_path": "` + keyPath + `"}},
		"scenario": {"id": "landing-page-business-suite", "ref": "main"},
		"dependencies": {"scenarios": ["landing-page-business-suite"]},
		"bundle": {"include_packages": true, "include_autoheal": true},
		"ports": {"api": 15000, "ui": 3000},
		"edge": {"domain": "vrooli.example", "caddy": {"enabled": true}}
	}`
	now := time.Now().UTC()
	if err := repo.CreateDeployment(ctx, &domain.Deployment{
		ID: depID, Name: "landing-page-business-suite @ vrooli.example", ScenarioID: "landing-page-business-suite", Environment: "production",
		Status: domain.StatusDeployed, Manifest: json.RawMessage(legacyManifest),
		Target:    identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.42", Port: 22, User: "root", Workdir: "/root/Vrooli"}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := repo.UpdateDeploymentSSHIdentity(ctx, depID, json.RawMessage(`{"key_path":"`+keyPath+`","auth_mode":"explicit_key","verification_state":"authorized"}`)); err != nil {
		t.Fatalf("seed ssh identity: %v", err)
	}
	// The API starts: schema init runs the conversion over the existing row.
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("startup schema/conversion: %v", err)
	}
	if bound, err := credentials.ResolveSSHKeyPath(ctx, repo, depID); err != nil || bound != keyPath {
		t.Fatalf("binding resolves %q err=%v, want %q", bound, err, keyPath)
	}

	// The resolver the router's SSH adapter uses yields the bound key.
	target := identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.42", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
	cfg, err := srv.sshConfigForTarget(ctx, target)
	if err != nil {
		t.Fatalf("resolve connection: %v", err)
	}
	if cfg.KeyPath != keyPath || cfg.Host != "203.0.113.42" || cfg.Port != 22 || cfg.User != "root" {
		t.Fatalf("connection config = %+v", cfg)
	}

	// A fake runner proves the adapter runs commands with that config and
	// that a read-only route answers through it.
	runner := &recordingRunner{answer: func(command string) sshadapter.Result {
		if strings.Contains(command, "'uname'") {
			return sshadapter.Result{Stdout: "Linux\nx86_64", ExitCode: 0}
		}
		return sshadapter.Result{Stdout: "", ExitCode: 0}
	}}
	srv.sshRunner = runner
	// The server's own seam (no router wired, as in every handler test)
	// resolves the connection exactly like the production router's adapter.
	res, err := srv.targetReach().Exec(ctx, target, reach.Command{Program: "uname", Args: []string{"-s"}, Timeout: time.Second})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("exec through the bound key: res=%+v err=%v", res, err)
	}
	if len(runner.configs) == 0 || runner.configs[0].KeyPath != keyPath {
		t.Fatalf("adapter did not run with the bound key: %+v", runner.configs)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/"+depID+"/logs?tail=5", nil)
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logs route = %d body=%s", rec.Code, rec.Body.String())
	}
	for _, cfg := range runner.configs {
		if cfg.KeyPath != keyPath {
			t.Fatalf("a command ran without the bound key: %+v", cfg)
		}
	}
	for _, command := range runner.commands {
		if strings.Contains(command, "canary-key") {
			t.Fatalf("the key path must never appear in a remote command: %s", command)
		}
	}

	// A host with no binding is reached with the ambient identity.
	ambient, err := srv.sshConfigForTarget(ctx, identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.99", Port: 22, User: "root"}})
	if err != nil || ambient.KeyPath != "" {
		t.Fatalf("unbound host config = %+v err=%v, want no key path", ambient, err)
	}
}
