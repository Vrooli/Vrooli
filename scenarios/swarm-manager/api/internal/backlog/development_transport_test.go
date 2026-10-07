package backlog_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	sharedidentity "github.com/vrooli/api-core/identity"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/transitions"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/internal/transitioncatalog"
)

func TestDevelopmentMountedSixOperationsRequireActualVerifiedHumanChannel(t *testing.T) {
	owner, store, contract := mountedDevelopmentFixture(t)
	// Disposable preconfigured provider material remains inside this test. No
	// environment file, live credential, grant or caller identity is installed.
	const fixtureToken = "disposable-development-provider-fixture"
	provider := authn.NewPersonalLocalProviderWithSessionToken(fixtureToken, "swarm-manager:read", "swarm-manager:write")
	probe := httptest.NewRequest(http.MethodPost, "http://localhost/", nil)
	probe.RemoteAddr = "127.0.0.1:12345"
	probe.Header.Set("Authorization", "Bearer "+fixtureToken)
	principal, err := provider.VerifyRequest(context.Background(), probe)
	if err != nil {
		t.Fatal(err)
	}
	contract.Subject.Owner = principal.Subject
	contract.Owner.Subject = principal.Subject
	contract.Owner.Realm = principal.Realm
	installed, err := backlog.NewDevelopmentOwner(owner, map[string]backlog.DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	backlog.RegisterDevelopmentRoutes(router, installed)
	transitioncatalog.RegisterRoutesWithDevelopmentPreview(router, transitions.Registry{}, nil, nil, (&backlog.DevelopmentService{Owner: installed}).PreviewDevelopment)
	server := httptest.NewServer(authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(router))
	defer server.Close()
	development := apiconnect.NewDevelopmentServiceClient(server.Client(), server.URL)
	transitions := apiconnect.NewTransitionServiceClient(server.Client(), server.URL)
	humanRead := connect.NewRequest(&api.GetDevelopmentRequest{EffortId: contract.Subject.Effort})
	humanRead.Header().Set("Authorization", "Bearer "+fixtureToken)
	view, err := development.GetDevelopment(context.Background(), humanRead)
	if err != nil {
		t.Fatal("verified actual provider read", err)
	}
	ref := view.Msg.Development.Reference
	before, _ := store.Load(contract.Subject.Effort)
	for _, credential := range []string{"", "Bearer invalid-fixture-material"} {
		headers := func(h http.Header) {
			if credential != "" {
				h.Set("Authorization", credential)
			}
			h.Set("X-Vrooli-Scopes", "swarm-manager:read swarm-manager:write")
			h.Set("X-Vrooli-Actor", "operator")
			h.Set("X-Vrooli-Subject", principal.Subject)
		}
		get := connect.NewRequest(&api.GetDevelopmentRequest{EffortId: contract.Subject.Effort})
		headers(get.Header())
		if _, e := development.GetDevelopment(context.Background(), get); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("get caller claim accepted: %v", e)
		}
		artifact := connect.NewRequest(&api.GetDevelopmentArtifactRequest{Reference: ref, ArtifactId: "fixture-artifact"})
		headers(artifact.Header())
		if _, e := development.GetDevelopmentArtifact(context.Background(), artifact); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("artifact caller claim accepted: %v", e)
		}
		approve := connect.NewRequest(&api.ApproveDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "anonymous-approve"})
		headers(approve.Header())
		if _, e := development.ApproveDevelopment(context.Background(), approve); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("approve caller claim accepted: %v", e)
		}
		revoke := connect.NewRequest(&api.RevokeDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "anonymous-revoke"})
		headers(revoke.Header())
		if _, e := development.RevokeDevelopment(context.Background(), revoke); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("revoke caller claim accepted: %v", e)
		}
		accept := connect.NewRequest(&api.AcceptDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), ExpectedDispositionVersion: mountedGeneration(0), RequestId: "anonymous-accept", TestedProductDigest: mountedDigest("fixture product")})
		headers(accept.Header())
		if _, e := development.AcceptDevelopment(context.Background(), accept); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("accept caller claim accepted: %v", e)
		}
		preview := connect.NewRequest(&api.PreviewDevelopmentRequest{Reference: ref, Proposal: &api.DevelopmentProposal{Title: "Read-only fixture preview"}})
		headers(preview.Header())
		if _, e := transitions.PreviewDevelopment(context.Background(), preview); connect.CodeOf(e) != connect.CodeUnauthenticated {
			t.Fatalf("preview caller claim accepted: %v", e)
		}
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("six denied adapter operations changed canonical owner state")
	}
	artifact := connect.NewRequest(&api.GetDevelopmentArtifactRequest{Reference: ref, ArtifactId: "fixture-artifact"})
	artifact.Header().Set("Authorization", "Bearer "+fixtureToken)
	if response, e := development.GetDevelopmentArtifact(context.Background(), artifact); e != nil || string(response.Msg.Content) != "retained fixture evidence\n" {
		t.Fatalf("verified human artifact: %v %v", response, e)
	}
	preview := connect.NewRequest(&api.PreviewDevelopmentRequest{Reference: ref, Proposal: &api.DevelopmentProposal{Title: "Preview retains no approval", WorkShape: string(backlog.FiniteBoundedAuthority), TargetSubjectRef: ref.CommissionSubjectDigest, Limits: view.Msg.Development.Limits, ScopeAllow: view.Msg.Development.ScopeAllow, ScopeDeny: view.Msg.Development.ScopeDeny, RequiredCriterionIds: []string{"auth.caller.identity"}, SourceRelativePaths: []string{"scenarios/swarm-manager/api/main.go"}}})
	preview.Header().Set("Authorization", "Bearer "+fixtureToken)
	if response, e := transitions.PreviewDevelopment(context.Background(), preview); e != nil || len(response.Msg.ObservedSources) != 1 {
		t.Fatalf("verified human preview: %v %v", response, e)
	}
	approve := connect.NewRequest(&api.ApproveDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "verified-approve"})
	approve.Header().Set("Authorization", "Bearer "+fixtureToken)
	if response, e := development.ApproveDevelopment(context.Background(), approve); e != nil || response.Msg.Actor.Subject != principal.Subject || response.Msg.Generation != 1 {
		t.Fatalf("verified human approve: %v %v", response, e)
	}
	before, _ = store.Load(contract.Subject.Effort)
	accept := connect.NewRequest(&api.AcceptDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(1), ExpectedDispositionVersion: mountedGeneration(0), RequestId: "verified-unavailable-accept", TestedProductDigest: mountedDigest("fixture product")})
	accept.Header().Set("Authorization", "Bearer "+fixtureToken)
	if _, e := development.AcceptDevelopment(context.Background(), accept); connect.CodeOf(e) != connect.CodeFailedPrecondition {
		t.Fatalf("unavailable product evidence became acceptance: %v", e)
	}
	after, _ = store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unavailable acceptance changed commission/product disposition")
	}
	revoke := connect.NewRequest(&api.RevokeDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(1), RequestId: "verified-revoke"})
	revoke.Header().Set("Authorization", "Bearer "+fixtureToken)
	if response, e := development.RevokeDevelopment(context.Background(), revoke); e != nil || response.Msg.Generation != 2 {
		t.Fatalf("verified human revoke: %v %v", response, e)
	}
	after, _ = store.Load(contract.Subject.Effort)
	if !after.FiniteCommission.Revoked || after.Development.DispositionVersion != 0 {
		t.Fatal("revocation launched, accepted or renewed work")
	}
}

func mountedGeneration(v uint64) *uint64 { return &v }
func mountedDigest(v any) string         { return "sha256:" + effortauthority.Digest(v) }
func mountedDevelopmentFixture(t *testing.T) (*backlog.FiniteCommissionOwner, *backlog.FileEffortControlStore, backlog.DevelopmentContract) {
	t.Helper()
	store := backlog.NewFileEffortControlStore(t.TempDir())
	c := identity.EffortControl{EffortID: "mounted-fixture", Revision: 1, DelegatedActions: []string{identity.DelegatedActionDispatch}, Scope: identity.EffortScope{Allow: []string{"scenarios/swarm-manager/**"}}, AggregateLimits: identity.AggregateLimits{MaxWorkers: 1, MaxConcurrency: 1, MaxActiveDescendants: 1}, RepairLimits: identity.RepairLimits{PerFingerprint: 1, PerComponent: 1, PerEffort: 1, ComponentActiveMinutes: 1}, PolicyBinding: identity.PolicyBinding{Source: "fixture/effort.json", Digest: mountedDigest("policy")}, CandidatePolicy: identity.CandidatePolicyBinding{EconomicalRunner: "opencode", EconomicalModel: "fixture/model", EconomicalEffort: "runner-native", FallbackRunner: "codex", FallbackModel: "fixture/fallback"}.Bind()}
	service := backlog.NewEffortControlService(store, nil)
	admitted, err := service.Admit(c)
	if err != nil {
		t.Fatal(err)
	}
	c = admitted
	handler := backlog.NewHandler(t.TempDir(), t.TempDir())
	handler.SetEffortControlService(service)
	finite, err := backlog.NewFiniteCommissionOwner(handler, map[string]backlog.FiniteCommissionTarget{c.EffortID: {WorkShape: backlog.FiniteBoundedAuthority}})
	if err != nil {
		t.Fatal(err)
	}
	source, retained := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "scenarios/swarm-manager/api"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "scenarios/swarm-manager/api/main.go"), []byte("package fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	artifact := []byte("retained fixture evidence\n")
	if err := os.WriteFile(filepath.Join(retained, "evidence.md"), artifact, 0600); err != nil {
		t.Fatal(err)
	}
	// Artifact digest is over bytes, not an JSON string.
	contract := backlog.DevelopmentContract{Subject: effortauthority.CommissionSubject{Owner: "fixture-owner", Effort: c.EffortID, Revision: "1", ContentDigest: strings.TrimPrefix(c.AuthorityDigest(), "sha256:"), Team: "fixture-team", Members: []string{"fixture-leader"}, Repository: t.TempDir(), TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Profiles: map[string]string{"native": effortauthority.Digest("profile")}}, Owner: backlog.DevelopmentPrincipalBinding{Kind: sharedidentity.ActorHuman, Subject: "fixture-owner", Source: sharedidentity.SourcePersonalLocal, Realm: "personal_local"}, RequiredCriteria: []string{"auth.caller.identity"}, Effects: []string{"run.create"}, SourceRoot: source, RetainedRoot: retained, Artifacts: []backlog.DevelopmentArtifactBinding{{ID: "fixture-artifact", RelativePath: "evidence.md", SHA256: mountedBytesDigest(artifact), SnapshotDigest: mountedDigest("snapshot"), SizeBytes: int64(len(artifact)), CapturedAt: time.Now().UTC()}}}
	return finite, store, contract
}
func mountedBytesDigest(raw []byte) string {
	s := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(s[:])
}

// This transport keeps all requests in-process; no listener, live token,
// credential store or issuer operation is used. The verifier is a disposable
// Agent Manager boundary fixture, not proof of current live caller migration.
type mountedDevelopmentMemoryClient struct{ handler http.Handler }

func (c mountedDevelopmentMemoryClient) Do(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)
	return w.Result(), nil
}
func mountedDevelopmentCallerFixture(t *testing.T, cfg authn.Config, scenarioCookie ...string) (*backlog.FileEffortControlStore, backlog.DevelopmentContract, *api.DevelopmentReference, apiconnect.DevelopmentServiceClient, apiconnect.TransitionServiceClient) {
	t.Helper()
	finite, store, contract := mountedDevelopmentFixture(t)
	provider := authn.NewPersonalLocalProviderWithSessionToken("disposable-mounted-human", "swarm-manager:read", "swarm-manager:write")
	probe := httptest.NewRequest(http.MethodPost, "http://fixture.invalid/", nil)
	probe.RemoteAddr = "127.0.0.1:12345"
	probe.Header.Set("Authorization", "Bearer disposable-mounted-human")
	principal, err := provider.VerifyRequest(context.Background(), probe)
	if err != nil {
		t.Fatal(err)
	}
	contract.Subject.Owner = principal.Subject
	contract.Owner.Subject = principal.Subject
	contract.Owner.Realm = principal.Realm
	contract.Readers = []backlog.DevelopmentPrincipalBinding{{Kind: sharedidentity.ActorAgent, Subject: "fixture-parent", Source: sharedidentity.SourceAgentProvenance}}
	owner, err := backlog.NewDevelopmentOwner(finite, map[string]backlog.DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) == 0 {
		cfg.Providers = []authn.Provider{provider}
	}
	router := mux.NewRouter()
	router.Use(authn.Middleware(cfg))
	backlog.RegisterDevelopmentRoutes(router, owner)
	router.Use(provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token == "unavailable-fixture" {
			return nil, errors.New("disposable verifier unavailable")
		}
		if token != "verified-fixture-parent" && token != "verified-fixture-other" && token != "verified-fixture-unscoped" {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		run := "fixture-parent"
		if token == "verified-fixture-other" {
			run = "fixture-other"
		}
		scopes := []string{"swarm-manager:read", "swarm-manager:write"}
		if token == "verified-fixture-unscoped" {
			scopes = nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: run, Scopes: scopes}}, nil
	})))
	cookieName := ""
	if len(scenarioCookie) != 0 {
		cookieName = scenarioCookie[0]
	}
	router.Use(backlog.DevelopmentCallerMiddleware(cookieName))
	transitioncatalog.RegisterRoutesWithDevelopmentPreview(router, transitions.Registry{}, nil, nil, (&backlog.DevelopmentService{Owner: owner}).PreviewDevelopment)
	transport := mountedDevelopmentMemoryClient{handler: router}
	development := apiconnect.NewDevelopmentServiceClient(transport, "http://fixture.invalid")
	transition := apiconnect.NewTransitionServiceClient(transport, "http://fixture.invalid")
	// Obtain the fixture reference through the actual owner with the principal
	// verified above. Offered channel tests must not need a successful warm-up
	// request through the deliberately wrong or narrowed test provider.
	view, err := owner.GetDevelopment(sharedidentity.WithPrincipal(context.Background(), principal), contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	return store, contract, view.Reference, development, transition
}
func TestDevelopmentActualMiddlewareOrderPreservesVerifiedAgentRead(t *testing.T) {
	store, contract, _, development, _ := mountedDevelopmentCallerFixture(t, authn.Config{})
	before, _ := store.Load(contract.Subject.Effort)
	request := connect.NewRequest(&api.GetDevelopmentRequest{EffortId: contract.Subject.Effort})
	request.Header().Set(cliutil.HeaderAgentIdentityToken, "verified-fixture-parent")
	if _, err := development.GetDevelopment(context.Background(), request); err != nil {
		t.Fatalf("verified installed reader rejected by actual middleware order: %v", err)
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read changed canonical standing")
	}
}
func TestDevelopmentActualMiddlewareOrderRejectsMixedHumanRunBeforeApprove(t *testing.T) {
	store, contract, ref, development, _ := mountedDevelopmentCallerFixture(t, authn.Config{})
	before, _ := store.Load(contract.Subject.Effort)
	request := connect.NewRequest(&api.ApproveDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "mixed-channel-approve"})
	request.Header().Set("Authorization", "Bearer disposable-mounted-human")
	request.Header().Set(cliutil.HeaderAgentIdentityToken, "verified-fixture-parent")
	if _, err := development.ApproveDevelopment(context.Background(), request); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("mixed human/run channels accepted: %v", err)
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("mixed proof changed canonical standing")
	}
}

func mountedDevelopmentInvoke(ctx context.Context, op string, client apiconnect.DevelopmentServiceClient, transition apiconnect.TransitionServiceClient, contract backlog.DevelopmentContract, ref *api.DevelopmentReference, headers func(http.Header)) error {
	switch op {
	case "get":
		q := connect.NewRequest(&api.GetDevelopmentRequest{EffortId: contract.Subject.Effort})
		headers(q.Header())
		out, err := client.GetDevelopment(ctx, q)
		if err == nil && (out.Msg.Development == nil || out.Msg.Development.Reference.EffortId != contract.Subject.Effort) {
			return errors.New("wrong get projection")
		}
		return err
	case "artifact":
		q := connect.NewRequest(&api.GetDevelopmentArtifactRequest{Reference: ref, ArtifactId: "fixture-artifact"})
		headers(q.Header())
		out, err := client.GetDevelopmentArtifact(ctx, q)
		if err == nil && string(out.Msg.Content) != "retained fixture evidence\n" {
			return errors.New("wrong artifact bytes")
		}
		return err
	case "approve":
		q := connect.NewRequest(&api.ApproveDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "matrix-approve"})
		headers(q.Header())
		_, err := client.ApproveDevelopment(ctx, q)
		return err
	case "revoke":
		q := connect.NewRequest(&api.RevokeDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "matrix-revoke"})
		headers(q.Header())
		_, err := client.RevokeDevelopment(ctx, q)
		return err
	case "accept":
		q := connect.NewRequest(&api.AcceptDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), ExpectedDispositionVersion: mountedGeneration(0), RequestId: "matrix-accept", TestedProductDigest: mountedDigest("product")})
		headers(q.Header())
		_, err := client.AcceptDevelopment(ctx, q)
		return err
	case "preview":
		// Preview is a read; the valid source scope is taken from the commissioned
		// fixture, not inferred from caller identity or broadened for this test.
		q := connect.NewRequest(&api.PreviewDevelopmentRequest{Reference: ref, Proposal: &api.DevelopmentProposal{Title: "read-only bounded preview", WorkShape: string(backlog.FiniteBoundedAuthority), TargetSubjectRef: ref.CommissionSubjectDigest, Limits: &api.DevelopmentLimits{MaxWorkers: 1, MaxConcurrency: 1, MaxActiveDescendants: 1}, ScopeAllow: []string{"scenarios/swarm-manager/**"}, RequiredCriterionIds: []string{"auth.caller.identity"}, SourceRelativePaths: []string{"scenarios/swarm-manager/api/main.go"}}})
		headers(q.Header())
		_, err := transition.PreviewDevelopment(ctx, q)
		return err
	}
	panic("unknown operation")
}
func TestDevelopmentMountedSixOperationsCallerChannelMatrix(t *testing.T) {
	ops := []string{"get", "artifact", "approve", "revoke", "accept", "preview"}
	tests := []struct {
		name, bearer, run   string
		cfg                 authn.Config
		readCode, writeCode connect.Code
	}{
		{name: "absent", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "bad-human", bearer: "invalid-human", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "unscoped-verified-run", run: "verified-fixture-unscoped", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "bad-run", run: "invalid-run", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "unavailable-run", run: "unavailable-fixture", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "mixed-verified", bearer: "disposable-mounted-human", run: "verified-fixture-parent", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "human-bad-run", bearer: "disposable-mounted-human", run: "invalid-run", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "human-unavailable-run", bearer: "disposable-mounted-human", run: "unavailable-fixture", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "bad-human-verified-run", bearer: "invalid-human", run: "verified-fixture-parent", readCode: connect.CodeUnauthenticated, writeCode: connect.CodeUnauthenticated},
		{name: "wrong-installed-parent", run: "verified-fixture-other", readCode: connect.CodeFailedPrecondition, writeCode: connect.CodeUnauthenticated},
		{name: "verified-installed-parent", run: "verified-fixture-parent", readCode: connect.CodeUnknown, writeCode: connect.CodeUnauthenticated},
		{name: "read-only-human", bearer: "disposable-mounted-human", cfg: authn.Config{Providers: []authn.Provider{authn.NewPersonalLocalProviderWithSessionToken("disposable-mounted-human", "swarm-manager:read")}}, readCode: connect.CodeUnknown, writeCode: connect.CodePermissionDenied},
		{name: "wrong-realm-human", bearer: "disposable-mounted-human", cfg: authn.Config{Providers: []authn.Provider{authn.PersonalLocalProvider{SessionToken: "disposable-mounted-human", Realm: "wrong-realm", Scopes: []string{"swarm-manager:read", "swarm-manager:write"}}}}, readCode: connect.CodeFailedPrecondition, writeCode: connect.CodeFailedPrecondition},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, contract, ref, client, transition := mountedDevelopmentCallerFixture(t, tc.cfg)
			before, _ := store.Load(contract.Subject.Effort)
			for _, op := range ops {
				t.Run(op, func(t *testing.T) {
					expected := tc.readCode
					if op == "approve" || op == "revoke" || op == "accept" {
						expected = tc.writeCode
					}
					err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) {
						if tc.bearer != "" {
							h.Set("Authorization", "Bearer "+tc.bearer)
						}
						if tc.run != "" {
							h.Set(cliutil.HeaderAgentIdentityToken, tc.run)
						}
						h.Set("X-Vrooli-Actor", "operator")
						h.Set("X-Vrooli-Subject", contract.Owner.Subject)
					})
					if expected == connect.CodeUnknown {
						if err != nil {
							t.Fatalf("supported read refused: %v", err)
						}
					} else if connect.CodeOf(err) != expected {
						t.Fatalf("code=%s want%s err=%v", connect.CodeOf(err), expected, err)
					}
					after, _ := store.Load(contract.Subject.Effort)
					if !reflect.DeepEqual(before, after) {
						t.Fatal("read/refusal changed canonical owner state")
					}
				})
			}
		})
	}
}

func TestDevelopmentCallerReconciliationLeavesSiblingIdentityUnchanged(t *testing.T) {
	const token = "disposable-sibling-human"
	cfg := authn.Config{Providers: []authn.Provider{authn.NewPersonalLocalProviderWithSessionToken(token, "swarm-manager:read")}}
	verifier := provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "sibling-run", Scopes: []string{"swarm-manager:read"}}}, nil
	})
	observed := false
	handler := authn.Middleware(cfg)(provenance.Middleware(verifier)(backlog.DevelopmentCallerMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := sharedidentity.PrincipalFromContext(r.Context())
		observed = ok && principal.Kind == sharedidentity.ActorHuman && principal.Source == sharedidentity.SourcePersonalLocal && provenance.FromContext(r.Context()).IsVerifiedAgent()
		w.WriteHeader(http.StatusNoContent)
	}))))
	request := httptest.NewRequest(http.MethodGet, "http://fixture.invalid/sibling-identity-fixture", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "fixture")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !observed || response.Code != http.StatusNoContent {
		t.Fatal("adapter-specific reconciliation changed sibling middleware identity")
	}
}

// This source-qualification fixture uses the actual ScenarioAuthenticator JWT
// provider with disposable in-memory signing material and a fixed JWKS Doer.
// It does not enroll a caller, issue live credentials, or contact an issuer.
type mountedScenarioJWTFixture struct {
	key             *rsa.PrivateKey
	now             time.Time
	provider        *authn.JWTVerifier
	jwksReads       int
	jwksUnavailable bool
}

func newMountedScenarioJWTFixture(t *testing.T) *mountedScenarioJWTFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &mountedScenarioJWTFixture{key: key, now: time.Now().UTC().Truncate(time.Second)}
	jwks, err := json.Marshal(map[string]any{"keys": []any{map[string]any{"kid": "disposable-mounted-key", "kty": "RSA", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}}})
	if err != nil {
		t.Fatal(err)
	}
	f.provider = authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Issuer: "scenario-authenticator", Audience: "scenario-authenticator:swarm-manager", JWKSURL: "http://jwks-fixture.invalid/.well-known/jwks.json", CookieName: "vrooli_access_token", Now: func() time.Time { return f.now }, Doer: mountedDevelopmentMemoryClient{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.String() != "http://jwks-fixture.invalid/.well-known/jwks.json" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get(cliutil.HeaderAgentIdentityToken) != "" {
			t.Error("JWT fixture attempted a different route or forwarded offered credentials")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		f.jwksReads++
		if f.jwksUnavailable {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	})}})
	return f
}
func (f *mountedScenarioJWTFixture) claims(subject string, kind sharedidentity.ActorKind, scopes []string) map[string]any {
	return map[string]any{"sub": subject, "actor_kind": string(kind), "realm": "disposable-scenario-realm", "iss": "scenario-authenticator", "aud": "scenario-authenticator:swarm-manager", "iat": f.now.Add(-time.Minute).Unix(), "exp": f.now.Add(time.Hour).Unix(), "scope": scopes}
}
func (f *mountedScenarioJWTFixture) sign(t *testing.T, claims map[string]any) string {
	return f.signWithKid(t, claims, "disposable-mounted-key")
}
func (f *mountedScenarioJWTFixture) signWithKid(t *testing.T, claims map[string]any, kid string) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": kid, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
func mountedScenarioDevelopmentFixture(t *testing.T, f *mountedScenarioJWTFixture, change func(*backlog.DevelopmentContract)) (*backlog.FileEffortControlStore, backlog.DevelopmentContract, *api.DevelopmentReference, apiconnect.DevelopmentServiceClient, apiconnect.TransitionServiceClient) {
	t.Helper()
	finite, store, contract := mountedDevelopmentFixture(t)
	contract.Subject.Owner = "jwt-fixture-owner"
	contract.Owner = backlog.DevelopmentPrincipalBinding{Kind: sharedidentity.ActorHuman, Subject: contract.Subject.Owner, Source: sharedidentity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Realm: "disposable-scenario-realm"}
	contract.Readers = []backlog.DevelopmentPrincipalBinding{
		{Kind: sharedidentity.ActorAgent, Subject: "jwt-fixture-agent-reader", Source: sharedidentity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Realm: "disposable-scenario-realm"},
		{Kind: sharedidentity.ActorHuman, Subject: "jwt-fixture-human-reader", Source: sharedidentity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Realm: "disposable-scenario-realm"},
	}
	if change != nil {
		change(&contract)
	}
	owner, err := backlog.NewDevelopmentOwner(finite, map[string]backlog.DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	router.Use(authn.Middleware(authn.Config{Providers: []authn.Provider{f.provider}}))
	backlog.RegisterDevelopmentRoutes(router, owner)
	router.Use(provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) { return &cliutil.VerifyResult{Valid: false}, nil })))
	router.Use(backlog.DevelopmentCallerMiddleware(""))
	transitioncatalog.RegisterRoutesWithDevelopmentPreview(router, transitions.Registry{}, nil, nil, (&backlog.DevelopmentService{Owner: owner}).PreviewDevelopment)
	transport := mountedDevelopmentMemoryClient{handler: router}
	// Resolve the immutable reference with an actual verified installed reader;
	// wrong-owner-binding cases must not require a fabricated principal or a
	// successful offered-owner call to construct their rejection request.
	reader, err := f.provider.Verify(context.Background(), f.sign(t, f.claims("jwt-fixture-agent-reader", sharedidentity.ActorAgent, []string{"swarm-manager:read"})))
	if err != nil {
		t.Fatal(err)
	}
	view, err := owner.GetDevelopment(sharedidentity.WithPrincipal(context.Background(), reader), contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	return store, contract, view.Reference, apiconnect.NewDevelopmentServiceClient(transport, "http://fixture.invalid"), apiconnect.NewTransitionServiceClient(transport, "http://fixture.invalid")
}
func mountedScenarioHumanOperations(t *testing.T, cookie bool) {
	f := newMountedScenarioJWTFixture(t)
	store, contract, ref, client, transition := mountedScenarioDevelopmentFixture(t, f, nil)
	token := f.sign(t, f.claims(contract.Owner.Subject, sharedidentity.ActorHuman, []string{"swarm-manager:read", "swarm-manager:write"}))
	headers := func(h http.Header) {
		if cookie {
			h.Set("Cookie", "vrooli_access_token="+token)
		} else {
			h.Set("Authorization", "Bearer "+token)
		}
	}
	before, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"get", "artifact", "preview"} {
		t.Run(op, func(t *testing.T) {
			if err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, headers); err != nil {
				t.Fatalf("actual JWT human read refused: %v", err)
			}
		})
	}
	after, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("actual JWT reads changed canonical owner state")
	}
	approve := connect.NewRequest(&api.ApproveDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(0), RequestId: "scenario-jwt-approve"})
	headers(approve.Header())
	approved, err := client.ApproveDevelopment(context.Background(), approve)
	if err != nil || approved.Msg.Generation != 1 || approved.Msg.Actor.Subject != contract.Owner.Subject || approved.Msg.Actor.Source != string(sharedidentity.SourceScenarioAuthenticator) || approved.Msg.Actor.Provider != contract.Owner.Issuer || approved.Msg.Actor.Realm != contract.Owner.Realm {
		t.Fatalf("actual JWT approval projection: %v %v", approved, err)
	}
	before, err = store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	accept := connect.NewRequest(&api.AcceptDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(1), ExpectedDispositionVersion: mountedGeneration(0), RequestId: "scenario-jwt-held-accept", TestedProductDigest: mountedDigest("product")})
	headers(accept.Header())
	if _, err := client.AcceptDevelopment(context.Background(), accept); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unavailable concrete acceptance became available: %v", err)
	}
	after, err = store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("held acceptance changed owner state")
	}
	revoke := connect.NewRequest(&api.RevokeDevelopmentRequest{Reference: ref, ExpectedGeneration: mountedGeneration(1), RequestId: "scenario-jwt-revoke"})
	headers(revoke.Header())
	revoked, err := client.RevokeDevelopment(context.Background(), revoke)
	if err != nil || revoked.Msg.Generation != 2 || revoked.Msg.Actor.Subject != contract.Owner.Subject {
		t.Fatalf("actual JWT revoke: %v %v", revoked, err)
	}
	after, err = store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if !after.FiniteCommission.Revoked || after.Development.DispositionVersion != 0 {
		t.Fatal("revoke accepted, renewed or launched work")
	}
	if f.jwksReads != 1 {
		t.Fatalf("provider did not use bounded cached fixture JWKS: %d", f.jwksReads)
	}
}
func TestDevelopmentMountedScenarioAuthenticatorInstalledReaders(t *testing.T) {
	for _, kind := range []sharedidentity.ActorKind{sharedidentity.ActorAgent, sharedidentity.ActorHuman} {
		t.Run(string(kind), func(t *testing.T) {
			f := newMountedScenarioJWTFixture(t)
			store, contract, ref, client, transition := mountedScenarioDevelopmentFixture(t, f, nil)
			subject := "jwt-fixture-agent-reader"
			if kind == sharedidentity.ActorHuman {
				subject = "jwt-fixture-human-reader"
			}
			token := f.sign(t, f.claims(subject, kind, []string{"swarm-manager:read"}))
			before, err := store.Load(contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range []string{"get", "artifact", "preview", "approve", "revoke", "accept"} {
				t.Run(op, func(t *testing.T) {
					err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) { h.Set("Authorization", "Bearer "+token) })
					write := op == "approve" || op == "revoke" || op == "accept"
					if !write && err != nil {
						t.Fatalf("actual installed JWT reader rejected: %v", err)
					}
					if write {
						want := connect.CodeUnauthenticated
						if kind == sharedidentity.ActorHuman {
							want = connect.CodePermissionDenied
						}
						if connect.CodeOf(err) != want {
							t.Fatalf("reader write code=%s want %s: %v", connect.CodeOf(err), want, err)
						}
					}
					after, e := store.Load(contract.Subject.Effort)
					if e != nil {
						t.Fatal(e)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("reader read/refusal changed canonical owner state")
					}
				})
			}
		})
	}
}
func TestDevelopmentMountedScenarioAuthenticatorRejections(t *testing.T) {
	f := newMountedScenarioJWTFixture(t)
	for _, tc := range []struct {
		name           string
		changeClaims   func(map[string]any)
		changeContract func(*backlog.DevelopmentContract)
		rawToken       string
		code           connect.Code
	}{
		{name: "absent", rawToken: "ABSENT", code: connect.CodeUnauthenticated},
		{name: "malformed", rawToken: "not-a-jwt", code: connect.CodeUnauthenticated},
		{name: "tampered-signature", rawToken: "TAMPER", code: connect.CodeUnauthenticated},
		{name: "jwks-unavailable", rawToken: "UNAVAILABLE", code: connect.CodeUnauthenticated},
		{name: "wrong-issuer", changeClaims: func(c map[string]any) { c["iss"] = "other-issuer" }, code: connect.CodeUnauthenticated},
		{name: "wrong-audience", changeClaims: func(c map[string]any) { c["aud"] = "scenario-authenticator:other-scenario" }, code: connect.CodeUnauthenticated},
		{name: "expired", changeClaims: func(c map[string]any) { c["exp"] = f.now.Add(-time.Minute).Unix() }, code: connect.CodeUnauthenticated},
		{name: "wrong-installed-source", changeContract: func(c *backlog.DevelopmentContract) { c.Owner.Source = sharedidentity.SourceCloudflareAccess }, code: connect.CodeFailedPrecondition},
		{name: "wrong-installed-issuer", changeContract: func(c *backlog.DevelopmentContract) { c.Owner.Issuer = "different-installed-issuer" }, code: connect.CodeFailedPrecondition},
		{name: "wrong-installed-realm", changeContract: func(c *backlog.DevelopmentContract) { c.Owner.Realm = "different-installed-realm" }, code: connect.CodeFailedPrecondition},
		{name: "wrong-installed-subject", changeClaims: func(c map[string]any) { c["sub"] = "uninstalled-human" }, code: connect.CodeFailedPrecondition},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, contract, ref, client, transition := mountedScenarioDevelopmentFixture(t, f, tc.changeContract)
			claims := f.claims("jwt-fixture-owner", sharedidentity.ActorHuman, []string{"swarm-manager:read", "swarm-manager:write"})
			if tc.changeClaims != nil {
				tc.changeClaims(claims)
			}
			token := f.sign(t, claims)
			if tc.rawToken == "TAMPER" {
				parts := strings.Split(token, ".")
				sig, err := base64.RawURLEncoding.DecodeString(parts[2])
				if err != nil {
					t.Fatal(err)
				}
				sig[0] ^= 1
				parts[2] = base64.RawURLEncoding.EncodeToString(sig)
				token = strings.Join(parts, ".")
			} else if tc.rawToken == "UNAVAILABLE" {
				f.jwksUnavailable = true
				readsBeforeUnavailable := f.jwksReads
				t.Cleanup(func() {
					f.jwksUnavailable = false
					if f.jwksReads <= readsBeforeUnavailable {
						t.Fatal("unavailable JWKS fixture was not consulted")
					}
				})
				token = f.signWithKid(t, claims, "unavailable-disposable-key")
			} else if tc.rawToken != "" {
				token = tc.rawToken
			}
			before, err := store.Load(contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range []string{"get", "artifact", "preview", "approve", "revoke", "accept"} {
				t.Run(op, func(t *testing.T) {
					err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) {
						if token != "ABSENT" {
							h.Set("Authorization", "Bearer "+token)
						}
						h.Set("X-Vrooli-Actor", "operator")
						h.Set("X-Vrooli-Subject", contract.Owner.Subject)
						h.Set("X-Vrooli-Scopes", "swarm-manager:read swarm-manager:write")
					})
					if connect.CodeOf(err) != tc.code {
						t.Fatalf("code=%s want %s: %v", connect.CodeOf(err), tc.code, err)
					}
					after, e := store.Load(contract.Subject.Effort)
					if e != nil {
						t.Fatal(e)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("rejected JWT/source changed canonical owner state")
					}
				})
			}
		})
	}
}

func TestDevelopmentMountedScenarioAuthenticatorHuman(t *testing.T) {
	for _, cookie := range []bool{false, true} {
		name := "bearer"
		if cookie {
			name = "configured-cookie"
		}
		t.Run(name, func(t *testing.T) { mountedScenarioHumanOperations(t, cookie) })
	}
}

func TestDevelopmentDisabledProviderProofCannotFallThroughToVerifiedRun(t *testing.T) {
	for _, tc := range []struct{ name, configuredCookie, header, value string }{
		{name: "default-scenario-cookie", header: "Cookie", value: "vrooli_access_token=offered-fixture-jwt"},
		{name: "empty-default-scenario-cookie", header: "Cookie", value: "vrooli_access_token="},
		{name: "custom-scenario-cookie", configuredCookie: "owner_custom_cookie", header: "Cookie", value: "owner_custom_cookie=offered-fixture-jwt"},
		{name: "cloudflare-assertion", header: "Cf-Access-Jwt-Assertion", value: "offered-fixture-jwt"},
		{name: "cloudflare-empty-assertion", header: "Cf-Access-Jwt-Assertion", value: ""},
		{name: "cloudflare-client-id", header: "CF-Access-Client-Id", value: "disposable-service-id"},
		{name: "cloudflare-client-secret", header: "CF-Access-Client-Secret", value: "disposable-service-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, contract, ref, client, transition := mountedDevelopmentCallerFixture(t, authn.Config{}, tc.configuredCookie)
			before, err := store.Load(contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range []string{"get", "artifact", "preview", "approve", "revoke", "accept"} {
				t.Run(op, func(t *testing.T) {
					err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) {
						h.Set(cliutil.HeaderAgentIdentityToken, "verified-fixture-parent")
						h[http.CanonicalHeaderKey(tc.header)] = []string{tc.value}
					})
					if connect.CodeOf(err) != connect.CodeUnauthenticated {
						t.Fatalf("disabled provider proof fell through to run: %v", err)
					}
					after, e := store.Load(contract.Subject.Effort)
					if e != nil {
						t.Fatal(e)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("ambiguous provider/run channels changed canonical owner")
					}
				})
			}
		})
	}
}
func TestDevelopmentUnrelatedCookiePreservesVerifiedInstalledRunReader(t *testing.T) {
	store, contract, ref, client, transition := mountedDevelopmentCallerFixture(t, authn.Config{})
	before, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"get", "artifact", "preview", "approve", "revoke", "accept"} {
		t.Run(op, func(t *testing.T) {
			err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) {
				h.Set(cliutil.HeaderAgentIdentityToken, "verified-fixture-parent")
				h.Set("Cookie", "unrelated_preferences=light")
			})
			write := op == "approve" || op == "revoke" || op == "accept"
			if (!write && err != nil) || (write && connect.CodeOf(err) != connect.CodeUnauthenticated) {
				t.Fatalf("unrelated cookie changed existing run channel: %v", err)
			}
			after, e := store.Load(contract.Subject.Effort)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("reader/cookie operation changed canonical owner")
			}
		})
	}
}

// Multiple offered values remain visible to the mounted middleware. An empty
// first value must not hide a later proof or authorize a different channel.
func TestDevelopmentDisabledProviderProofMultiplicity(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		values       []string
		human        bool
	}{
		{"empty-run-with-human", cliutil.HeaderAgentIdentityToken, []string{""}, true},
		{"empty-first-duplicate-run-with-human", cliutil.HeaderAgentIdentityToken, []string{"", "verified-fixture-parent"}, true},
		{"duplicate-valid-run", cliutil.HeaderAgentIdentityToken, []string{"verified-fixture-parent", "verified-fixture-parent"}, false},
		{"empty-first-duplicate-authorization", "Authorization", []string{"", "Bearer disposable-mounted-human"}, false},
		{"empty-first-duplicate-cf-assertion", "Cf-Access-Jwt-Assertion", []string{"", "offered-fixture-jwt"}, false},
		{"empty-first-duplicate-cf-client-id", "CF-Access-Client-Id", []string{"", "disposable-service-id"}, false},
		{"empty-first-duplicate-cf-client-secret", "CF-Access-Client-Secret", []string{"", "disposable-service-secret"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, contract, ref, client, transition := mountedDevelopmentCallerFixture(t, authn.Config{})
			before, err := store.Load(contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range []string{"get", "artifact", "preview", "approve", "revoke", "accept"} {
				t.Run(op, func(t *testing.T) {
					err := mountedDevelopmentInvoke(context.Background(), op, client, transition, contract, ref, func(h http.Header) {
						if tc.human {
							h.Set("Authorization", "Bearer disposable-mounted-human")
						} else {
							h.Set(cliutil.HeaderAgentIdentityToken, "verified-fixture-parent")
						}
						h.Del(tc.header)
						for _, value := range tc.values {
							h.Add(tc.header, value)
						}
					})
					if connect.CodeOf(err) != connect.CodeUnauthenticated {
						t.Fatalf("ambiguous/empty proof admitted %s: %v", op, err)
					}
					after, err := store.Load(contract.Subject.Effort)
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("ambiguous/empty proof changed canonical owner state")
					}
				})
			}
		})
	}
}
