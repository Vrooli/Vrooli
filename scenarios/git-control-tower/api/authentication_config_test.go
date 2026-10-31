package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/config"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/reviewjobs"
	"git-control-tower/internal/stagedmessages"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/cloudflareaccess"
	testdb "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/identity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

func TestPersonalLocalAuthenticationKeepsExplicitHumanProvidersAndOwnerPolicy(t *testing.T) {
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	t.Setenv("VROOLI_AUTH_PROVIDERS", "")
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"operator","canonical_subject":"owner","scopes":["agent-manager:write"]}]`)
	cfg, err := buildAuthenticationConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	sources := map[identity.AuthSource]bool{}
	for _, p := range cfg.Providers {
		sources[p.Source()] = true
	}
	if !sources[identity.SourcePersonalLocal] || !sources[identity.SourceScenarioAuthenticator] || !sources[identity.SourceCloudflareAccess] || len(cfg.OwnerBindings) != 1 {
		t.Fatalf("explicit identity providers or owner policy dropped: sources=%v bindings=%d", sources, len(cfg.OwnerBindings))
	}
}

func TestStagedMessagePersonalLocalProxyForwardsSignedCloudflareOwner(t *testing.T) {
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	t.Setenv("VROOLI_AUTH_PROVIDERS", "")
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", `[{"source":"cloudflare_access","subject":"operator","canonical_subject":"owner","scopes":["agent-manager:write"]}]`)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "fixture", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
	}))
	defer jwks.Close()
	cfg, err := buildAuthenticationConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range cfg.Providers {
		if managed, ok := p.(*authn.ManagedCloudflareProvider); ok {
			found = true
			managed.Resolve = func(_ context.Context, resource string) (cloudflareaccess.Config, error) {
				if resource != "git-control-tower" {
					t.Fatalf("wrong relying resource: %s", resource)
				}
				return cloudflareaccess.Config{TeamDomain: "https://team.example.test", Audience: "fixture-audience", JWKSURL: jwks.URL}, nil
			}
		}
	}
	if !found {
		t.Fatal("managed Cloudflare verifier missing in personal-local proxy configuration")
	}
	encode := func(v any) string {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	sign := func(audience string, expiry time.Time) string {
		unsigned := encode(map[string]any{"alg": "RS256", "kid": "fixture"}) + "." + encode(map[string]any{"iss": "https://team.example.test", "aud": audience, "sub": "operator", "email": "operator@example.test", "type": "app", "iat": time.Now().Add(-time.Minute).Unix(), "exp": expiry.Unix()})
		digest := sha256.Sum256([]byte(unsigned))
		signature, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if e != nil {
			t.Fatal(e)
		}
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	}
	for _, tc := range []struct {
		name, audience string
		expiry         time.Time
		valid          bool
	}{
		{"valid", "fixture-audience", time.Now().Add(time.Hour), true},
		{"wrong audience", "unrelated-audience", time.Now().Add(time.Hour), false},
		{"expired", "fixture-audience", time.Now().Add(-time.Hour), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/start", nil)
			req.RemoteAddr = "127.0.0.1:41234" // Immediate UI proxy, not original browser location.
			token := sign(tc.audience, tc.expiry)
			req.Header.Set("Cf-Access-Jwt-Assertion", token)
			principal, e := cfg.Authenticate(t.Context(), req)
			if !tc.valid {
				if e == nil {
					t.Fatal("invalid offered proof fell back to local OS authority")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if principal.Source != identity.SourceCloudflareAccess || principal.Subject != "owner" {
				t.Fatalf("proxy replaced verified browser owner: %+v", principal)
			}
			req.RemoteAddr = "192.0.2.10:41234"
			if direct, err := cfg.Authenticate(t.Context(), req); err != nil || direct.Source != identity.SourceCloudflareAccess {
				t.Fatalf("local provider vetoed independently verified remote browser: %v", err)
			}
			ctx := policygate.WithVerifiedOwnerProof(identity.WithPrincipal(t.Context(), principal), req.Header)
			w := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING)}
			if _, e := stagedAgentFixture(w, nil).Start(ctx, stagedmessages.DraftInput{Snapshot: stagedAgentSnapshot()}, "explicit-browser-start"); e != nil {
				t.Fatal(e)
			}
			if w.startHeaders.Get("Cf-Access-Jwt-Assertion") != token || w.startHeaders.Get("X-Vrooli-Workflow-Initiator") != "human" || w.startHeaders.Get("X-Agent-Identity-Token") != "" {
				t.Fatal("verified browser proof lost or replaced at execution owner boundary")
			}
			// The destination independently verifies the same configured audience and
			// applies its own owner policy. No GCT-local OS identity crosses the hop.
			verifier, e := cloudflareaccess.NewVerifier(cloudflareaccess.Config{TeamDomain: "https://team.example.test", Audience: "fixture-audience", JWKSURL: jwks.URL})
			if e != nil {
				t.Fatal(e)
			}
			ownerCfg := authn.Config{Providers: []authn.Provider{verifier}, OwnerBindings: cfg.OwnerBindings}
			ownerReq := httptest.NewRequest(http.MethodPost, "/workflow", nil)
			ownerReq.Header = w.startHeaders
			owner, e := ownerCfg.Authenticate(t.Context(), ownerReq)
			if e != nil {
				t.Fatal(e)
			}
			ownerCtx := identity.WithPrincipal(t.Context(), owner)
			if _, e = authn.RequireIndependentHuman(ownerCtx); e != nil {
				t.Fatal(e)
			}
			if _, e = authn.RequireCapability(ownerCtx, "agent-manager:write"); e != nil {
				t.Fatal(e)
			}
			checkStagedMessageHTTPAuthenticationChain(t, cfg, ownerCfg, token, false)
			t.Run("destination lacks launch capability", func(t *testing.T) {
				ownerCfg.OwnerBindings = nil
				checkStagedMessageHTTPAuthenticationChain(t, cfg, ownerCfg, token, true)
			})
		})
	}
}

type authenticatedDraftOwner struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	starts int
	header http.Header
}

func (o *authenticatedDraftOwner) StartWorkflowExecution(ctx context.Context, req *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error) {
	if _, err := authn.RequireIndependentHuman(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if _, err := authn.RequireCapability(ctx, "agent-manager:write"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	o.starts++
	o.header = req.Header().Clone()
	return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING)}), nil
}

type authenticationSnapshot struct{}

func (authenticationSnapshot) Capture(_ context.Context, repo, _ string) (stagedmessages.Captured, error) {
	snapshot := stagedAgentSnapshot()
	snapshot.RepositoryId = repo
	return stagedmessages.Captured{Snapshot: snapshot}, nil
}

func (authenticationSnapshot) Identity(context.Context, string, string) (string, error) {
	return "snapshot-1", nil
}

func checkStagedMessageHTTPAuthenticationChain(t *testing.T, inbound, destination authn.Config, token string, denied bool) {
	t.Helper()
	owner := &authenticatedDraftOwner{}
	amPath, amHandler := apiconnect.NewAgentManagerServiceHandler(owner)
	amMux := http.NewServeMux()
	amMux.Handle(amPath, amHandler)
	am := httptest.NewServer(authn.Middleware(destination)(amMux))
	defer am.Close()
	c := NewAgentManagerClient(time.Second)
	c.cachedBaseURL = am.URL
	c.httpClient = am.Client()
	repos := newTestRepoStore(t)
	record, err := repos.Upsert(t.Context(), RepoRecord{Path: t.TempDir(), Name: "authentication-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := reviewjobs.New(testdb.OpenSQLiteMemory(t))
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{agentManagerClient: c, repos: NewRepoService(repos, NewFakeGitRunner())}
	p := newStagedMessageProvider(s).(*stagedMessageAgent)
	p.search = func(string) stagedMessageSearchAPI {
		return &stagedSearchFake{response: &ampb.SearchConversationsResponse{}}
	}
	s.stagedMessages = stagedmessages.New(store, authenticationSnapshot{}, nil, p)
	path, handler := advisory_v1connect.NewAdvisoryServiceHandler(advisoryConnectServer{server: s}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	gct := httptest.NewServer(authn.Middleware(inbound)(mux))
	defer gct.Close()
	client := advisory_v1connect.NewAdvisoryServiceClient(gct.Client(), gct.URL)
	repoID := repositoryIDFor(&record)
	collectReq := connect.NewRequest(&v1.CollectStagedMessageRequest{RepositoryId: repoID})
	collectReq.Header().Set("Cf-Access-Jwt-Assertion", token)
	collection, err := client.CollectStagedMessage(t.Context(), collectReq)
	if err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&v1.StartStagedMessageRequest{RepositoryId: repoID, CollectionId: collection.Msg.Id, SnapshotDigest: collection.Msg.Snapshot.Digest, RequestKey: "http-browser-start"})
	request.Header().Set("Cf-Access-Jwt-Assertion", token)
	job, err := client.StartStagedMessage(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if denied {
		if job.Msg.Status != "failed" || job.Msg.ExecutionId != "" || owner.starts != 0 {
			t.Fatalf("destination owner policy bypassed: status=%s starts=%d", job.Msg.Status, owner.starts)
		}
		return
	}
	if job.Msg.Status != "queued" || job.Msg.ExecutionId == "" || owner.starts != 1 || owner.header.Get("Cf-Access-Jwt-Assertion") != token || owner.header.Get("X-Agent-Identity-Token") != "" {
		t.Fatalf("HTTP owner admission lost browser proof: status=%s execution=%s starts=%d", job.Msg.Status, job.Msg.ExecutionId, owner.starts)
	}
}
