package handlers

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/rolepolicy"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/effortauthority"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type joinedNativeScope struct {
	profiles interface {
		GetByKey(context.Context, string) (*domain.AgentProfile, error)
	}
	store effortauthority.Store
}

func (s joinedNativeScope) CheckEffortScope(ctx context.Context, p effortauthority.Policy) error {
	rec, e := s.store.Get(ctx, p.ID)
	if e != nil || effortauthority.Digest(rec.Policy) != effortauthority.Digest(p) {
		return effortauthority.ErrRefused
	}
	for key, digest := range p.Profiles {
		profile, e := s.profiles.GetByKey(ctx, key)
		if e != nil || orchestration.EffortProfileDigest(profile) != digest {
			return effortauthority.ErrRefused
		}
	}
	return nil
}

// Invoked only by the joined disposable PM test, never by scenario lifecycle.
func TestFiniteJoinedNativeHTTPFixture(t *testing.T) {
	root := os.Getenv("VROOLI_DISPOSABLE_FINITE_JOINED_ROOT")
	if root == "" {
		t.Skip("joined fixture helper requires its disposable parent")
	}
	ctx := context.Background()
	deadline := time.Now().Add(45 * time.Second)
	now := func() time.Time {
		b, e := os.ReadFile(filepath.Join(root, "clock.json"))
		if e != nil {
			t.Fatal(e)
		}
		var n time.Time
		if json.Unmarshal(b, &n) != nil {
			t.Fatal("bad clock")
		}
		return n
	}
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	db, e := sql.Open("sqlite", filepath.Join(root, "ledger.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ledger := &effortauthority.SQLStore{DB: db}
	if e = ledger.Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	dispatcher.Close() // Real admission; no native model/process dispatch.
	policyPath := filepath.Join(root, "model-policy.json")
	if e = os.WriteFile(policyPath, []byte(`{"schemaVersion":1,"metadata":{"catalogId":"disposable-joined","updatedAt":"2026-10-04"},"defaultRole":"code.default","roles":{"code.default":{"description":"fixture","intent":"fixture","candidates":[{"runner":"codex","resourceRole":"code.default"},{"runner":"claude-code","resourceRole":"code.default"},{"runner":"opencode","resourceRole":"code.default"}]}}}`), 0600); e != nil {
		t.Fatal(e)
	}
	role, e := rolepolicy.NewState(policyPath, rolepolicy.Requirement{Required: true})
	if e != nil {
		t.Fatal(e)
	}
	registry := runner.NewRegistry()
	if e = registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode)); e != nil {
		t.Fatal(e)
	}
	// Configure before any run request. Metadata preparation itself is fixture setup.
	ownerClient := &effortauthority.OwnerReadClient{}
	authority := &effortauthority.Authority{Store: ledger, Owners: ownerClient, Scope: joinedNativeScope{repos.Profiles, ledger}, Now: now}
	var nativeEngine effortauthority.Engine = authority
	remote := &effortauthority.BrokerClient{}
	_, useBrokerErr := os.Stat(filepath.Join(root, "broker-enabled"))
	if useBrokerErr == nil {
		nativeEngine = remote
	}
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Hour, MaxConcurrentRuns: 4, DefaultProjectRoot: root}), orchestration.WithEvents(events), orchestration.WithRunners(registry), orchestration.WithRolePolicyState(role, handlerRoleResolver{}), orchestration.WithRunStateRoot(filepath.Join(root, "run-state")), orchestration.WithClock(now), orchestration.WithIdentitySecret([]byte("disposable-joined-native-secret")), orchestration.WithSpawnDispatcher(dispatcher), orchestration.WithEffortAuthority(nativeEngine))
	profile, e := orch.CreateProfile(ctx, &domain.AgentProfile{Name: "Disposable joined", ProfileKey: "qualified-profile", RoleRef: "code.default", MaxTurns: 10, Timeout: time.Hour, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, DeclaredScopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}})
	if e != nil {
		t.Fatal(e)
	}
	profile, e = repos.Profiles.GetByKey(ctx, profile.ProfileKey)
	if e != nil {
		t.Fatal(e)
	}
	router := mux.NewRouter()
	New(orchestration.NewHandlerServices(orch)).RegisterRoutes(router)
	// Read-only fixture metadata provides the full native profile digest.
	router.HandleFunc("/fixture/profile-digest", func(w http.ResponseWriter, r *http.Request) {
		p, e := repos.Profiles.GetByKey(ctx, "qualified-profile")
		if e != nil {
			http.Error(w, "profile read failed", 500)
			return
		}
		json.NewEncoder(w).Encode(orchestration.EffortProfileDigest(p))
	})
	router.HandleFunc("/fixture/settle/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, e := uuid.Parse(mux.Vars(r)["id"])
		if e != nil {
			http.Error(w, "bad id", 400)
			return
		}
		run, e := repos.Runs.Get(ctx, id)
		if e != nil || run.ResolvedConfig.Admission.Effort == nil {
			http.Error(w, "missing fixture run", 404)
			return
		}
		run.Status = domain.RunStatusComplete
		end := now()
		run.EndedAt = &end
		if e = repos.Runs.Update(ctx, run); e != nil {
			http.Error(w, "fixture update failed", 500)
			return
		}
		if e = orch.SyncEffortTerminal(ctx, *run.ResolvedConfig.Admission.Effort, run.IdempotencyKey, run.ID.String()); e != nil {
			http.Error(w, "fixture settle failed", 500)
			return
		}
		w.WriteHeader(204)
	}).Methods("POST")
	server := httptest.NewServer(router)
	defer server.Close()
	meta, _ := json.Marshal(map[string]string{"url": server.URL, "profileDigest": orchestration.EffortProfileDigest(profile)})
	if e = os.WriteFile(filepath.Join(root, "native.json"), meta, 0600); e != nil {
		t.Fatal(e)
	}
	for time.Now().Before(deadline) {
		b, e := os.ReadFile(filepath.Join(root, "owner.json"))
		if e == nil {
			var owner struct{ URL, Pin string }
			if json.Unmarshal(b, &owner) != nil {
				t.Fatal("owner metadata")
			}
			cert, e := tls.LoadX509KeyPair(filepath.Join(root, "peer.crt"), filepath.Join(root, "peer.key"))
			if e != nil {
				t.Fatal(e)
			}
			pem, e := os.ReadFile(filepath.Join(root, "peer.crt"))
			if e != nil {
				t.Fatal(e)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				t.Fatal("fixture trust")
			}
			ownerClient.URL = owner.URL
			ownerClient.ServerPin = owner.Pin
			ownerClient.HTTP = &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}
			os.WriteFile(filepath.Join(root, "native-ready"), []byte("ready"), 0600)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if useBrokerErr == nil {
		configured := false
		for time.Now().Before(deadline) {
			raw, e := os.ReadFile(filepath.Join(root, "broker.json"))
			if e == nil {
				var meta struct{ URL, Pin string }
				if json.Unmarshal(raw, &meta) != nil {
					t.Fatal("broker metadata")
				}
				cert, e := tls.LoadX509KeyPair(filepath.Join(root, "native-peer.crt"), filepath.Join(root, "native-peer.key"))
				if e != nil {
					t.Fatal(e)
				}
				ca, e := os.ReadFile(filepath.Join(root, "peer.crt"))
				if e != nil {
					t.Fatal(e)
				}
				pool := x509.NewCertPool()
				if !pool.AppendCertsFromPEM(ca) {
					t.Fatal("broker trust")
				}
				remote.URL = meta.URL
				remote.ServerPin = meta.Pin
				remote.HTTP = &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}
				os.WriteFile(filepath.Join(root, "broker-native-ready"), []byte("ready"), 0600)
				configured = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !configured {
			t.Fatal("broker native configuration timeout")
		}
	}
	for time.Now().Before(deadline) {
		if _, e = os.Stat(filepath.Join(root, "done")); e == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("joined disposable parent did not complete within bounded fixture deadline")
}
