package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/api-core/identity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

func paidCodingAuthorityFixture(t *testing.T) *Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "refresh_token": "refresh", "expires_at": "2030-01-01T00:00:00Z"})
		case "/api/v1/ai/keys/device":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Error("device issuance lost consumer identity")
			}
			json.NewEncoder(w).Encode(codingKey{ID: "device", Key: "device-secret", ExpiresAt: time.Now().Add(time.Hour).Unix()})
		case "/api/v1/ai/models":
			if r.Header.Get("Authorization") != "Bearer device-secret" {
				t.Error("models omitted issued device credential")
			}
			json.NewEncoder(w).Encode(map[string]any{"data": []codingModel{{ID: "coding", Name: "Vrooli coding"}}})
		case "/api/v1/ai/keys/device/unused":
			http.NotFound(w, r)
		case "/api/v1/ai/keys/session":
			json.NewEncoder(w).Encode(codingKey{ID: "session", Key: "session-secret", ExpiresAt: time.Now().Add(time.Hour).Unix()})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return &Server{subscriptionResolver: &credentialclient.ConsumerSessionResolver{Credentials: &businessAccountCredentialFixture{}, LPBSBaseURL: server.URL, HTTPClient: server.Client()}}
}

func TestPaidCodingSessionCredentialIsSessionBoundAndRevocable(t *testing.T) {
	var revoked atomic.Int32
	authority := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/v1/ai/keys/device" || r.Header.Get("Authorization") != "Bearer device-secret" {
			t.Error("revoke did not use private parent authority")
		}
		revoked.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer authority.Close()
	registry := newPaidCodingSessionRegistry()
	registry.Register("session", paidCodingBinding{authority: codingAuthority{BaseURL: authority.URL}, device: codingKey{ID: "device", Key: "device-secret"}, key: codingKey{ExpiresAt: time.Now().Add(time.Minute).Unix()}, environment: map[string]string{"VROOLI_CODING_SESSION_KEY": "session-secret"}})
	if registry.Environment("session")["VROOLI_CODING_SESSION_KEY"] != "session-secret" {
		t.Fatal("issued key missing")
	}
	registry.RemoveSession("session")
	registry.RemoveSession("session")
	if registry.Environment("session") != nil || revoked.Load() != 1 {
		t.Fatal("revocation was not idempotent")
	}
}

func TestPaidCodingSessionInjectsOpenCodeProviderConfigOnlyIntoPTYEnvironment(t *testing.T) {
	server := paidCodingAuthorityFixture(t)
	register, revoke, err := server.preparePaidCodingSession(t.Context(), "account", "opencode", "coding", 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer revoke()
	register("session")
	defer activePaidCodingSessions.RemoveSession("session")
	env := activePaidCodingSessions.Environment("session")
	if strings.Contains(env[cliutil.ModelSourceEnvironment], "session-secret") || env["VROOLI_CODING_SESSION_KEY"] != "session-secret" {
		t.Fatal("credential escaped its dedicated child environment slot")
	}
	for _, value := range env {
		if strings.Contains(value, "device-secret") {
			t.Fatal("device authority exposed to harness")
		}
	}
}

func TestCodingBackupFailedRevocationPersistsAndRetries(t *testing.T) {
	var attempts atomic.Int32
	door := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.Header.Get("Authorization") != "Bearer fixture-parent" {
			t.Error("wrong revocation request")
		}
		if attempts.Add(1) == 1 {
			http.Error(w, "unavailable", 503)
			return
		}
		w.WriteHeader(204)
	}))
	defer door.Close()
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "isolated.enc"), "fixture passphrase")
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "fixture-owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
	owner := codingBackupOwner(ctx)
	consent := cliutil.CodingBackupConsent{Harness: "codex", Account: "fixture-account", Model: "fixture-model", MaxCredits: 1000, ExpiresAt: time.Now().Add(time.Hour), Automatic: true}
	record := cliutil.CodingBackupRecord{Consent: consent, Owner: owner, BaseURL: door.URL, ParentID: "fixture-id", ParentKey: "fixture-parent"}
	encoded, _ := json.Marshal(record)
	if _, err := credentials.Provision(ctx, credentialclient.ProvisionRequest{Identity: cliutil.CodingBackupIdentity, Field: cliutil.CodingBackupField(owner, "codex"), Value: string(encoded)}); err != nil {
		t.Fatal(err)
	}
	server := &Server{credentialClient: credentials}
	request := func(s *Server) int {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/coding-backup?harness=codex", nil).WithContext(ctx)
		req.RemoteAddr = "127.0.0.1:4000"
		w := httptest.NewRecorder()
		s.codingBackupHandler(w, req)
		return w.Code
	}
	if got := request(server); got != 503 {
		t.Fatalf("failed revoke reported %d", got)
	}
	if server.backupGrant(ctx, "codex") != nil {
		t.Fatal("pending revoke still permits backup")
	}
	restarted := &Server{credentialClient: credentials}
	pending := restarted.savedBackupGrant(ctx, "codex")
	if pending == nil || !pending.RevocationPending {
		t.Fatal("revocation retry lost after restart")
	}
	raw, err := credentials.Resolve(ctx, cliutil.CodingBackupIdentity, cliutil.CodingBackupField(owner, "codex"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		t.Fatal(err)
	}
	if _, err := cliutil.IssueCodingBackupSession(ctx, record, owner, door.Client()); err == nil {
		t.Fatal("shared unattended issuer accepted pending revoke")
	}
	if got := request(restarted); got != 204 {
		t.Fatalf("successful retry reported %d", got)
	}
	if attempts.Load() != 2 {
		t.Fatalf("revoke attempts=%d", attempts.Load())
	}
	if restarted.savedBackupGrant(ctx, "codex") != nil {
		t.Fatal("successful revoke retained consent")
	}
}

// SM8 consent states are observations of the existing credential authority.
// Revoked and never granted both observe absent; neither needs a new store.
func TestCodingBackupConsentLifecycleMatrix(t *testing.T) {
	for _, state := range []string{"absent", "active", "expired", "pending"} {
		for _, event := range []string{"observe", "issue", "revoke", "revoke_unavailable", "recover"} {
			t.Run(state+"/"+event, func(t *testing.T) {
				var issued, revoked int
				door := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer matrix-parent" {
						t.Error("parent custody lost")
					}
					if r.Method == http.MethodDelete {
						revoked++
						if event == "revoke_unavailable" {
							http.Error(w, "fixture unavailable", 503)
							return
						}
						w.WriteHeader(204)
						return
					}
					issued++
					json.NewEncoder(w).Encode(codingKey{ID: "matrix-child", Key: "matrix-child-secret", ExpiresAt: time.Now().Add(time.Minute).Unix()})
				}))
				defer door.Close()
				authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "matrix.enc"), "isolated test passphrase")
				if err != nil {
					t.Fatal(err)
				}
				client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
				if err != nil {
					t.Fatal(err)
				}
				ctx := identity.WithPrincipal(t.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "matrix-owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
				consent := cliutil.CodingBackupConsent{Harness: "codex", Account: "account", Model: "model", MaxCredits: 1000, ExpiresAt: time.Now().Add(time.Hour), Automatic: true}
				record := cliutil.CodingBackupRecord{Owner: codingBackupOwner(ctx), Consent: consent, BaseURL: door.URL, ParentID: "matrix-parent-id", ParentKey: "matrix-parent", RevocationPending: state == "pending"}
				if state == "expired" {
					record.Consent.ExpiresAt = time.Now().Add(-time.Second)
				}
				if state != "absent" {
					encoded, _ := json.Marshal(record)
					if _, err := client.Provision(ctx, credentialclient.ProvisionRequest{Identity: cliutil.CodingBackupIdentity, Field: cliutil.CodingBackupField(record.Owner, "codex"), Value: string(encoded)}); err != nil {
						t.Fatal(err)
					}
				}
				s := &Server{credentialClient: client}
				if (s.backupGrant(ctx, "codex") != nil) != (state == "active") {
					t.Fatal("initial admission differs from consent state")
				}
				switch event {
				case "issue":
					// A captured launch is not authority: issuance must resolve the
					// current stored grant, including revocation and expiry.
					captured := &codingBackupGrant{Owner: record.Owner, Consent: consent, Device: codingKey{ID: record.ParentID}}
					launch := context.WithValue(ctx, codingBackupContextKey{}, codingBackupLaunch{Grant: captured})
					register, cleanup, err := s.preparePaidCodingSession(launch, "account", "codex", "model", 1000)
					if (err == nil) != (state == "active") {
						t.Fatalf("issuance err=%v state=%s", err, state)
					}
					if err == nil {
						register("sm8-matrix")
						env := activePaidCodingSessions.Environment("sm8-matrix")
						if env["VROOLI_CODING_SESSION_KEY"] != "matrix-child-secret" || strings.Contains(fmt.Sprint(env), "matrix-parent") {
							t.Fatal("child custody violated")
						}
						activePaidCodingSessions.RemoveSession("sm8-matrix")
						_ = cleanup // RemoveSession owns cleanup after registration.
					}
					if (issued == 1) != (state == "active") {
						t.Fatalf("issuance count=%d", issued)
					}
				case "revoke", "revoke_unavailable":
					err := s.revokeBackupGrant(ctx, s.savedBackupGrant(ctx, "codex"))
					wantAttempt := state == "active" || state == "pending"
					if (revoked == 1) != wantAttempt || (err != nil) != (wantAttempt && event == "revoke_unavailable") {
						t.Fatalf("revoke count=%d err=%v", revoked, err)
					}
					if s.backupGrant(ctx, "codex") != nil {
						t.Fatal("revoke retained admission")
					}
					fresh := &Server{credentialClient: client}
					if fresh.backupGrant(ctx, "codex") != nil {
						t.Fatal("recovery reopened revoked admission")
					}
					if wantAttempt && event == "revoke_unavailable" {
						g := fresh.savedBackupGrant(ctx, "codex")
						if g == nil || !g.RevocationPending {
							t.Fatal("recovery lost pending intent")
						}
					}
				case "recover":
					s = &Server{credentialClient: client}
					if (s.backupGrant(ctx, "codex") != nil) != (state == "active") {
						t.Fatal("recovered admission changed")
					}
				case "observe":
					g := s.savedBackupGrant(ctx, "codex")
					if (g != nil) != (state == "active" || state == "pending") {
						t.Fatal("saved observation changed")
					}
				}
				t.Logf("owner transition state=%s event=%s issued=%d revoked=%d", state, event, issued, revoked)
			})
		}
	}
}

// All session phases cross all restart events. The create seam records the
// requested replacement; TestCodingBackupProductionJourney proves the native
// process, credential registration and actual door effects behind this seam.
func TestCodingBackupSessionLifecycleMatrix(t *testing.T) {
	for _, phase := range []string{"chosen", "subscription_active", "credits_active"} {
		for _, event := range []string{"switch_back", "switch_fresh", "switch_stale", "switch_without_consent", "switch_pending", "switch_expired", "switch_create_failure", "wrong_owner"} {
			t.Run(phase+"/"+event, func(t *testing.T) {
				ctx := identity.WithPrincipal(t.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "matrix-owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
				owner := codingBackupOwner(ctx)
				store := sessionstore.NewInMemory()
				s := &Server{sessionStore: store, agentUsageURL: "http://127.0.0.1:1"}
				observed := time.Now()
				if event == "switch_stale" {
					observed = observed.Add(-6 * time.Minute)
				}
				path := filepath.Join(t.TempDir(), "synthetic-limit.jsonl")
				if err := os.WriteFile(path, []byte(fmt.Sprintf("{\"timestamp\":%q,\"type\":\"error\",\"error\":{\"code\":\"usage_limit_reached\"}}\n", observed.UTC().Format(time.RFC3339Nano))), 0600); err != nil {
					t.Fatal(err)
				}
				row := sessionstore.Metadata{ID: "sm8-existing", AgentType: sessionstore.AgentCodex, Owner: owner, LastRolloutPath: path}
				if phase == "credits_active" {
					row.BusinessAccountID = "account"
				}
				if event == "wrong_owner" {
					row.Owner = "someone-else"
				}
				if phase != "chosen" {
					if err := store.Save(ctx, row); err != nil {
						t.Fatal(err)
					}
				}
				consent := cliutil.CodingBackupConsent{Harness: "codex", Account: "account", Model: "model", MaxCredits: 1000, ExpiresAt: time.Now().Add(time.Hour), Automatic: true}
				if event == "switch_expired" {
					consent.ExpiresAt = time.Now().Add(-time.Second)
				}
				if event != "switch_without_consent" {
					s.codingBackups = map[string]*codingBackupGrant{owner + "/codex": {Owner: owner, Consent: consent, Device: codingKey{ID: "parent"}, RevocationPending: event == "switch_pending"}}
				}
				creates := 0
				s.desktopCreateSession = func(ctx context.Context, in sessionsH.CreateInput) (sessionsH.Session, error) {
					creates++
					launch := ctx.Value(codingBackupContextKey{}).(codingBackupLaunch)
					if event == "switch_back" {
						if !launch.Subscription || launch.Grant != nil || in.BusinessAccountID != "" || in.AgentLaunch.Provider != "" || in.AgentLaunch.Model != "" {
							t.Error("switch-back retained credit authority")
						}
					} else if launch.Grant == nil || in.BusinessAccountID != "account" || in.AgentLaunch.Provider != "vrooli-credits" || in.AgentLaunch.Model != "model" || in.AgentLaunch.MaxCredits != 1000 {
						t.Error("bounded paid selection lost")
					}
					if event == "switch_create_failure" {
						return sessionsH.Session{}, errors.New("isolated create failure")
					}
					return sessionsH.Session{ID: "replacement"}, nil
				}
				source := "credits"
				if event == "switch_back" {
					source = "subscription"
				}
				req := httptest.NewRequest(http.MethodPost, "/api/v1/coding-backup/restart", strings.NewReader(fmt.Sprintf(`{"session_id":"sm8-existing","source":%q}`, source))).WithContext(ctx)
				req.RemoteAddr = "127.0.0.1:4000"
				w := httptest.NewRecorder()
				s.codingBackupRestart(w, req)
				want, wantCreates := 409, 0
				if phase == "chosen" || event == "wrong_owner" {
					want = 404
				} else if event == "switch_fresh" || event == "switch_back" {
					want, wantCreates = 200, 1
				} else if event == "switch_create_failure" {
					wantCreates = 1
				}
				if w.Code != want || creates != wantCreates {
					t.Fatalf("status=%d creates=%d want=%d/%d: %s", w.Code, creates, want, wantCreates, w.Body.String())
				}
				if phase != "chosen" {
					current, err := store.Get(ctx, row.ID)
					if err != nil || current.BusinessAccountID != row.BusinessAccountID || current.Owner != row.Owner {
						t.Fatal("restart mutated original session")
					}
				}
				t.Logf("phase=%s event=%s status=%d replacements=%d", phase, event, w.Code, creates)
			})
		}
	}
}

func TestCodingBackupChosenSourceLaunchMatrix(t *testing.T) {
	for _, choice := range []string{"default", "provider", "model", "account", "remote", "literal", "not_executing", "not_agent", "switch_back"} {
		for _, event := range []string{"fresh_automatic", "fresh_manual", "stale_automatic", "pending_automatic"} {
			t.Run(choice+"/"+event, func(t *testing.T) {
				ctx := identity.WithPrincipal(t.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "launch-owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
				owner := codingBackupOwner(ctx)
				store := sessionstore.NewInMemory()
				s := &Server{sessionStore: store, agentUsageURL: "http://127.0.0.1:1"}
				observed := time.Now()
				if event == "stale_automatic" {
					observed = observed.Add(-6 * time.Minute)
				}
				path := filepath.Join(t.TempDir(), "synthetic.jsonl")
				if err := os.WriteFile(path, []byte(fmt.Sprintf("{\"timestamp\":%q,\"type\":\"error\",\"error\":{\"code\":\"usage_limit_reached\"}}\n", observed.UTC().Format(time.RFC3339Nano))), 0600); err != nil {
					t.Fatal(err)
				}
				if err := store.Save(ctx, sessionstore.Metadata{ID: "prior", Owner: owner, AgentType: sessionstore.AgentCodex, LastRolloutPath: path}); err != nil {
					t.Fatal(err)
				}
				g := &codingBackupGrant{Owner: owner, RevocationPending: event == "pending_automatic", Consent: cliutil.CodingBackupConsent{Harness: "codex", Account: "backup-account", Model: "backup-model", MaxCredits: 1000, ExpiresAt: time.Now().Add(time.Hour), Automatic: event != "fresh_manual"}}
				s.codingBackups = map[string]*codingBackupGrant{owner + "/codex": g}
				in := sessionsH.CreateInput{AgentLaunch: &sessionsH.AgentLaunch{AgentID: "codex"}, LaunchCommand: "codex", ExecuteLaunchCommand: true, TargetID: "local"}
				switch choice {
				case "provider":
					in.AgentLaunch.Provider = "subscription"
				case "model":
					in.AgentLaunch.Model = "selected-model"
				case "account":
					in.BusinessAccountID = "selected-account"
				case "remote":
					in.TargetID = "remote"
				case "literal":
					in.LaunchCommand = "codex --model selected-model"
				case "not_executing":
					in.ExecuteLaunchCommand = false
				case "not_agent":
					in.AgentLaunch = nil
				case "switch_back":
					ctx = context.WithValue(ctx, codingBackupContextKey{}, codingBackupLaunch{Subscription: true})
				}
				before, _ := json.Marshal(in)
				outCtx, out, err := s.resolveCodingBackup(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				want := choice == "default" && event == "fresh_automatic"
				if want {
					if out.BusinessAccountID != "backup-account" || out.AgentLaunch.Model != "backup-model" || out.AgentLaunch.Provider != "vrooli-credits" || out.AgentLaunch.MaxCredits != 1000 {
						t.Fatal("automatic selection lost bounded consent")
					}
					launch, _ := outCtx.Value(codingBackupContextKey{}).(codingBackupLaunch)
					if launch.Grant != g {
						t.Fatal("selection lost current consent")
					}
				} else {
					after, _ := json.Marshal(out)
					if string(before) != string(after) {
						t.Fatal("selected source changed without eligible automatic backup")
					}
				}
				unchanged, _ := json.Marshal(in)
				if string(before) != string(unchanged) {
					t.Fatal("selection mutated caller's original launch")
				}
				t.Logf("choice=%s event=%s automatic_switch=%t", choice, event, want)
			})
		}
	}
}

type pendingWriteFailure struct {
	credentialclient.Client
	fail bool
}

func (c *pendingWriteFailure) Provision(ctx context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	if c.fail {
		return credentialclient.ProvisionResponse{}, errors.New("fixture credential store unavailable")
	}
	return c.Client.Provision(ctx, request)
}

func TestCodingBackupRevocationBlocksLaunchWhenPendingWriteFails(t *testing.T) {
	for _, doorFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("doorFails=%t", doorFails), func(t *testing.T) {
			var attempts atomic.Int32
			var unavailable atomic.Bool
			unavailable.Store(doorFails)
			door := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				if r.Method != http.MethodDelete || r.Header.Get("Authorization") != "Bearer fixture-parent" {
					t.Error("wrong revocation request")
				}
				if unavailable.Load() {
					http.Error(w, "fixture door unavailable", http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer door.Close()
			authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "isolated.enc"), "fixture passphrase")
			if err != nil {
				t.Fatal(err)
			}
			client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
			if err != nil {
				t.Fatal(err)
			}
			credentials := &pendingWriteFailure{Client: client}
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "fixture-owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
			owner := codingBackupOwner(ctx)
			record := cliutil.CodingBackupRecord{Owner: owner, BaseURL: door.URL, ParentID: "fixture-id", ParentKey: "fixture-parent", Consent: cliutil.CodingBackupConsent{Harness: "codex", Account: "fixture-account", Model: "fixture-model", MaxCredits: 1000, ExpiresAt: time.Now().Add(time.Hour), Automatic: true}}
			encoded, _ := json.Marshal(record)
			if _, err := credentials.Provision(ctx, credentialclient.ProvisionRequest{Identity: cliutil.CodingBackupIdentity, Field: cliutil.CodingBackupField(owner, "codex"), Value: string(encoded)}); err != nil {
				t.Fatal(err)
			}
			server := &Server{credentialClient: credentials}
			grant := server.backupGrant(ctx, "codex")
			if grant == nil {
				t.Fatal("fixture consent missing")
			}
			credentials.fail = true
			err = server.revokeBackupGrant(ctx, grant)
			if server.backupGrant(ctx, "codex") != nil {
				t.Error("failed pending write retained usable consent")
			}
			if attempts.Load() != 1 {
				t.Fatalf("parent revocation attempts=%d, want 1 despite storage failure", attempts.Load())
			}
			// A launch may already have captured the active grant before revoke
			// started. It must recheck consent before minting its child key.
			staleLaunch := context.WithValue(ctx, codingBackupContextKey{}, codingBackupLaunch{Grant: grant})
			if _, _, err := server.preparePaidCodingSession(staleLaunch, record.Consent.Account, "codex", record.Consent.Model, record.Consent.MaxCredits); err == nil {
				t.Error("stale launch retained revoked consent")
			}
			if attempts.Load() != 1 {
				t.Fatal("stale launch attempted child issuance after revocation began")
			}
			if doorFails {
				if err == nil {
					t.Fatal("failed revocation reported success")
				}
				if pending := server.savedBackupGrant(ctx, "codex"); pending == nil || !pending.RevocationPending {
					t.Fatal("pending retry lost")
				}
				credentials.fail = false
				unavailable.Store(false)
				if err := server.revokeBackupGrant(ctx, server.savedBackupGrant(ctx, "codex")); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatalf("confirmed parent revoke and credential removal failed: %v", err)
			}
			if restarted := (&Server{credentialClient: credentials}).savedBackupGrant(ctx, "codex"); restarted != nil {
				t.Fatal("completed revocation reloaded old consent")
			}
		})
	}
}
