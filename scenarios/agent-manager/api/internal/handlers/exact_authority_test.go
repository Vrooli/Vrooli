package handlers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	adb "agent-manager/internal/adapters/database"
	eventstore "agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modules"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/authn"
	coredb "github.com/vrooli/api-core/database"
	core "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

func TestExactProofNativeAttachMountedIngressReopenAndConcurrentRetry(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "exact.db") + "?_pragma=busy_timeout(100)"
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	repos := adb.NewRepositories(adb.NewDB(db, logrus.New()), logrus.New())
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	dispatcher.Close()
	o := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithSpawnDispatcher(dispatcher), orchestration.WithIdentitySecret([]byte("disposable-exact-test-secret")))
	attachment, err := o.AttachRun(ctx, orchestration.AttachRunRequest{HarnessKind: "codex", HarnessSession: "synthetic-exact-session"})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := o.VerifyIdentityToken(ctx, attachment.Token)
	if err != nil || !verified.Valid || verified.Claims.Subject != "" || len(verified.Claims.Scopes) != 0 {
		t.Fatal("actual zero-scope native attachment", err)
	}
	principal := core.Principal{Verified: true, Kind: core.ActorAgent, Subject: verified.Claims.RunID.String(), ExpiresAt: time.Unix(verified.Claims.ExpiresAt, 0)}
	ctx = core.WithPrincipal(ctx, principal)
	repo := supervision.NewRepository(db)
	if err = repo.EnsureExactAuthoritySchema(ctx); err != nil {
		t.Fatal(err)
	}
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	p := supervision.ExactPolicy{ID: "fixture-grant", Owner: "fixture-owner", Instance: "fixture-native-1", Issuer: "fixture-keyring", KeyVersion: 1, Epoch: 1, Deadline: time.Now().Add(time.Hour).Unix(), EffortRef: "effort:fixture", AuthorizedBy: "fixture-owner", AuthorityRef: "fixture-approval", Profile: supervision.ExactProfile, PublicKey: public}
	seed := &pb.EffortEnrollment{EffortRef: p.EffortRef, DisplayName: "before", Revision: 1, AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}
	if err = repo.SaveEffort(ctx, seed, &pb.EffortBoardRow{}, 0, "seed", "seed"); err != nil {
		t.Fatal(err)
	}
	// Qualify installed ingress through the current owner setup contract in
	// this disposable database, including inactive-stage refusal.
	p.Operations = []string{"effort.reconcile"}
	owner := core.WithPrincipal(context.Background(), core.Principal{Kind: core.ActorHuman, Verified: true, Subject: p.Owner, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)})
	installed := supervision.NewInstalledExactVerifier(repo)
	staged, err := installed.Setup(owner, supervision.ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil {
		t.Fatal("fixture stage", err)
	}
	stageRequest := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(seed).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "inactive-stage", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	if _, err = installed.Challenge(ctx, p.ID, stageRequest); err == nil {
		t.Fatal("staged policy admitted challenge")
	}
	activation, err := json.Marshal(staged.ActivationChallenge)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = installed.Setup(owner, supervision.ExactSetupRequest{Action: "activate", Grant: p.ID, ExpectedEpoch: p.Epoch, PolicyDigest: staged.PolicyDigest, KeyProof: base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, activation))}); err != nil {
		t.Fatal("fixture activation", err)
	}
	makeHandler := func(db *sqlx.DB) *AgentManagerConnectHandler {
		repo := supervision.NewRepository(db)
		h := NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{})})
		v := supervision.NewInstalledExactVerifier(repo)
		h.SetExactVerifier(v)
		return h
	}
	h := makeHandler(db)
	router := mux.NewRouter()
	h.RegisterExactRoutes(router)
	router.Use(provenance.Middleware(exactNativeTestVerifier{o}))
	router.Use(authn.Middleware(authn.Config{}))
	router.Use(AgentOperationMiddleware)
	req := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(seed).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "mounted-first", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	req.Enrollment.DisplayName = "after α<>&"
	action, _ := protojson.Marshal(req)
	digestBytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(req)
	sum := sha256.Sum256(digestBytes)
	challenge := httptest.NewRequest("POST", ExactChallengePath+"?grant="+p.ID, bytes.NewReader(action)).WithContext(ctx)
	challenge.Header.Set("X-Agent-Identity-Token", attachment.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, challenge)
	if response.Code != 200 {
		t.Fatal("mounted challenge", response.Code, response.Body.String())
	}
	var c supervision.ExactChallenge
	if json.Unmarshal(response.Body.Bytes(), &c) != nil || c.Executor != principal.Subject || c.Payload != hex.EncodeToString(sum[:]) {
		t.Fatal("challenge binding")
	}
	sign := func() *connect.Request[pb.ReconcileEffortMetadataRequest] {
		e := supervision.ExactEnvelope{Version: 1, Purpose: supervision.ExactPurpose, Instance: p.Instance, Audience: "agent-manager", Issuer: p.Issuer, KeyVersion: 1, Action: action, Grant: p.ID, Epoch: 1, Initiator: p.Owner, Executor: principal.Subject, Operation: "effort.reconcile", Target: c.Target, Payload: c.Payload, Request: req.IdempotencyKey, Expires: time.Now().Add(30 * time.Second).Unix()}
		raw, _ := json.Marshal(e)
		r := connect.NewRequest(req)
		r.Header().Set(supervision.ExactProofHeader, base64.RawURLEncoding.EncodeToString(raw))
		r.Header().Set(supervision.ExactSignatureHeader, base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw)))
		return r
	}
	if _, err = h.ReconcileEffortMetadata(ctx, connect.NewRequest(req)); err == nil {
		t.Fatal("bare zero-scope identity accepted")
	}
	disabled := NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{})})
	if _, err = disabled.ReconcileEffortMetadata(ctx, sign()); err == nil {
		t.Fatal("normal startup accepted proof")
	}
	for _, bad := range []func(*connect.Request[pb.ReconcileEffortMetadataRequest]){func(r *connect.Request[pb.ReconcileEffortMetadataRequest]) {
		r.Header().Set(supervision.ExactProofHeader, "")
	}, func(r *connect.Request[pb.ReconcileEffortMetadataRequest]) {
		r.Header().Add(supervision.ExactProofHeader, "duplicate")
	}} {
		r := sign()
		bad(r)
		if _, err = h.ReconcileEffortMetadata(ctx, r); err == nil {
			t.Fatal("ambiguous proof accepted")
		}
	}
	wrong := httptest.NewRequest("POST", ExactChallengePath, bytes.NewReader(action)).WithContext(ctx)
	wrong.Header.Set(supervision.ExactProofHeader, "offered")
	wrong.Header.Set("X-Agent-Identity-Token", attachment.Token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, wrong)
	if response.Code != 403 {
		t.Fatal("wrong-channel proof accepted")
	}
	// Two service instances/pools compete for one effect and existing receipt.
	db2, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	h2 := makeHandler(db2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, handler := range []*AgentManagerConnectHandler{h, h2} {
		routed := mux.NewRouter()
		path, service := apiconnect.NewAgentManagerServiceHandler(handler)
		routed.PathPrefix(path).Handler(service)
		routed.Use(provenance.Middleware(exactNativeTestVerifier{o}))
		routed.Use(authn.Middleware(authn.Config{}))
		routed.Use(AgentOperationMiddleware)
		wg.Add(1)
		go func(router http.Handler) {
			defer wg.Done()
			rpc := httptest.NewRequest("POST", apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure, bytes.NewReader(action))
			rpc.Header = sign().Header().Clone()
			rpc.Header.Set("X-Agent-Identity-Token", attachment.Token)
			rpc.Header.Set("Content-Type", "application/json")
			rpc.Header.Set("Connect-Protocol-Version", "1")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, rpc)
			if response.Code == 200 {
				errs <- nil
			} else {
				errs <- fmt.Errorf("mounted reconcile status %d: %s", response.Code, response.Body.String())
			}
		}(routed)
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes < 1 {
		t.Fatal("no effect committed")
	}
	db2.Close()
	db.Close()
	db, err = sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h = makeHandler(db)
	replay, err := h.ReconcileEffortMetadata(ctx, sign())
	if err != nil || replay.Msg.Revision != 2 {
		t.Fatal("reopened fresh-proof retry", err)
	}
	var n int
	if err = db.Get(&n, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key='reconcile-metadata:mounted-first'`); err != nil || n != 1 {
		t.Fatal("duplicate receipt", n, err)
	}
	// An uncommitted busy revoke is not acknowledged. The original row survives.
	db2, err = sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_authority SET revoked=revoked WHERE id=?`, p.ID); err != nil {
		t.Fatal(err)
	}
	if err = supervision.NewRepository(db2).RevokeExactAuthority(ctx, p.ID); !errors.Is(err, supervision.ErrExactRetryable) {
		t.Fatal("busy revoke not explicitly uncommitted/retryable", err)
	}
	if _, err = makeHandler(db2).ReconcileEffortMetadata(ctx, sign()); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatal("busy effect not explicitly retryable", err)
	}
	tx.Rollback()
	conn.Close()
	repo = supervision.NewRepository(db)
	if err = repo.RevokeExactAuthority(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = h.ReconcileEffortMetadata(ctx, sign()); err == nil {
		t.Fatal("committed revoke accepted proof retry")
	}
}

type exactNativeTestVerifier struct{ o *orchestration.Orchestrator }

func (v exactNativeTestVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	result, err := v.o.VerifyIdentityToken(context.Background(), token)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var verified cliutil.VerifyResult
	if err = json.Unmarshal(raw, &verified); err != nil {
		return nil, err
	}
	return &verified, nil
}

// The child binary is the actual CLI package test binary, built with existing
// repository dependencies. Native owns disposable storage and mounted ingress;
// synthetic leaf/key material crosses only the private stdin pipe.
func TestExactCLIJoinedMountedNativeMutationAndFutureReceipt(t *testing.T) {
	binary := os.Getenv("KEYRING_JOINED_CLI_TEST_BINARY")
	if binary == "" || os.Getenv("KEYRING_SITE_SIGN_HELPER") == "" {
		t.Skip("explicit disposable CLI binary and actual Site signer helper required")
	}
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "joined.db")+"?_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err = coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	repos := adb.NewRepositories(adb.NewDB(db, logrus.New()), logrus.New())
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	dispatcher.Close()
	events := eventstore.NewSQLiteStore(adb.NewDB(db, logrus.New()), logrus.New())
	o := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(events), orchestration.WithSpawnDispatcher(dispatcher), orchestration.WithIdentitySecret([]byte("disposable-joined-native-secret")))
	repo := supervision.NewRepository(db)
	if err = repo.EnsureExactAuthoritySchema(ctx); err != nil {
		t.Fatal(err)
	}
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	p := supervision.ExactPolicy{ID: "joined-mutation-grant", Owner: "fixture-owner", Instance: "fixture-native-1", Issuer: "fixture-keyring", KeyVersion: 1, Epoch: 1, Deadline: time.Now().Add(time.Hour).Unix(), EffortRef: "effort:joined", AuthorizedBy: "fixture-owner", AuthorityRef: "fixture-approval", Profile: supervision.ExactProfile, PublicKey: public, Operations: []string{"effort.reconcile", "receipt.read"}, ReadEligibility: supervision.ExactStandingReadEligibility}
	seed := &pb.EffortEnrollment{EffortRef: p.EffortRef, DisplayName: "before", Revision: 1, AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}
	if err = repo.SaveEffort(ctx, seed, &pb.EffortBoardRow{}, 0, "seed", "seed"); err != nil {
		t.Fatal(err)
	}
	h := NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{})})
	v := supervision.NewInstalledExactVerifier(repo)
	h.SetExactVerifier(v)
	router := mux.NewRouter()
	h.RegisterExactRoutes(router)
	bootstrapHandler := New(orchestration.NewHandlerServices(o))
	router.HandleFunc("/api/v1/runs/attach", bootstrapHandler.AttachRun).Methods("POST")
	router.HandleFunc("/api/v1/runs/{id}/detach", bootstrapHandler.DetachRun).Methods("POST")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }).Methods("GET")
	path, service := apiconnect.NewAgentManagerServiceHandler(h)
	router.PathPrefix(path).Handler(service)
	router.Use(provenance.Middleware(exactNativeTestVerifier{o}))
	router.Use(authn.Middleware(authn.Config{Providers: []authn.Provider{authn.TokenProvider{ProviderSource: core.SourceScenarioAuthenticator, Extract: func(r *http.Request) string { return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") }, Verifier: exactJoinedOwnerVerifier{p.Owner}}}}))
	router.Use(AgentOperationMiddleware)
	requests := 0
	bootstrapTokens := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			requests++
		}
		if r.URL.Path != ExactSetupPath && r.URL.Path != "/health" && r.Header.Get("Authorization") != "" {
			t.Error("operator authority reached native action")
		}
		if r.URL.Path == "/api/v1/runs/attach" {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, r)
			var result struct {
				Run struct {
					ID          string `json:"id"`
					HarnessKind string `json:"harness_kind"`
					RunnerPID   int    `json:"runner_pid"`
				} `json:"run"`
				Token string `json:"identity_token"`
			}
			if recorder.Code == 201 && json.Unmarshal(recorder.Body.Bytes(), &result) == nil {
				stored, err := repos.Runs.Get(context.Background(), uuid.MustParse(result.Run.ID))
				if err != nil || stored.HarnessKind != "command-session" || stored.RunnerPID <= 0 || stored.HarnessSessionID == "" {
					t.Error("actual command wrapper identity missing")
				}
				bootstrapTokens[result.Run.ID] = result.Token
			}
			for k, values := range recorder.Header() {
				w.Header()[k] = values
			}
			w.WriteHeader(recorder.Code)
			_, _ = w.Write(recorder.Body.Bytes())
			return
		}
		router.ServeHTTP(w, r)
	}))
	defer server.Close()
	var releases []io.WriteCloser
	var processes []*exec.Cmd
	defer func() {
		for _, r := range releases {
			_ = r.Close()
		}
		for _, cmd := range processes {
			if err := cmd.Wait(); err != nil {
				t.Error("actual launcher process cleanup", err)
			}
		}
		for id, token := range bootstrapTokens {
			verified, err := o.VerifyIdentityToken(ctx, token)
			stored, getErr := repos.Runs.Get(ctx, uuid.MustParse(id))
			count, countErr := events.Count(ctx, uuid.MustParse(id))
			if err != nil || verified.Valid || getErr != nil || stored.Status != domain.RunStatusComplete || stored.IdentityTokenRevokedAt == nil || countErr != nil || count != 2 {
				t.Error("actual command cleanup/revocation/event receipt", err, getErr, countErr, count)
			}
		}
	}()
	runCLI := func(mode, token string, policy supervision.ExactPolicy, key ed25519.PrivateKey, request []byte) []byte {
		config, _ := json.Marshal(struct {
			Base, Token, Grant, Owner, Issuer string
			Key                               []byte
			Mode                              string
			Request                           json.RawMessage
			Deadline                          int64
		}{server.URL, token, policy.ID, policy.Owner, policy.Issuer, key, mode, request, policy.Deadline})
		cmd := exec.Command(binary, "-test.run=^TestExactCLIJoinedProcess$", "-test.count=1")
		cmd.Env = []string{"KEYRING_DISPOSABLE_JOINED_CHILD=1", "KEYRING_SITE_SIGN_HELPER=" + os.Getenv("KEYRING_SITE_SIGN_HELPER"), "KEYRING_JOINED_MAIN_BINARY=" + os.Getenv("KEYRING_JOINED_MAIN_BINARY"), "KEYRING_JOINED_COMMAND_BINARY=" + os.Getenv("KEYRING_JOINED_COMMAND_BINARY"), "PATH=/usr/bin:/bin"}
		if strings.HasPrefix(mode, "setup-") {
			cmd.Stdin = bytes.NewReader(config)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("ordinary main %s failed: %s", mode, output)
			}
			return output
		}
		input, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stderr = io.Discard
		if cmd.Start() != nil {
			t.Fatal("launcher helper start")
		}
		if _, err = input.Write(append(config, '\n')); err != nil {
			t.Fatal(err)
		}
		reader := bufio.NewReader(stdout)
		var output bytes.Buffer
		for {
			line, err := reader.ReadBytes('\n')
			output.Write(line)
			if err != nil {
				t.Fatal("missing actual main receipt", output.String())
			}
			if bytes.HasPrefix(line, []byte("EXACT_MAIN_RECEIPT:")) {
				break
			}
		}
		releases = append(releases, input)
		processes = append(processes, cmd)
		// Drain only test completion after releasing the harness; never print credentials.
		go func() { _, _ = io.Copy(io.Discard, reader) }()
		return output.Bytes()
	}
	setupCLI := func(action string, request supervision.ExactSetupRequest) supervision.ExactSetupReceipt {
		raw, _ := json.Marshal(request)
		output := runCLI("setup-"+action, "disposable-verified-owner-fixture", p, key, raw)
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "EXACT_SETUP_RECEIPT:") {
				var receipt supervision.ExactSetupReceipt
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "EXACT_SETUP_RECEIPT:")), &receipt) != nil {
					t.Fatal("CLI setup receipt")
				}
				return receipt
			}
		}
		t.Fatal("missing CLI setup receipt")
		return supervision.ExactSetupReceipt{}
	}
	staged := setupCLI("stage", supervision.ExactSetupRequest{Action: "stage", Policy: &p})
	if staged.State != "staged" || staged.ActivationChallenge == nil {
		t.Fatal("inactive stage missing")
	}
	recovered := setupCLI("status", supervision.ExactSetupRequest{Action: "status", Grant: p.ID})
	if recovered.PolicyDigest != staged.PolicyDigest || recovered.ApprovedAt != staged.ApprovedAt {
		t.Fatal("CLI status changed approval")
	}
	activationProof := exactSiteActivation(t, staged.ActivationChallenge, p, key)
	active := setupCLI("activate", supervision.ExactSetupRequest{Action: "activate", Grant: p.ID, ExpectedEpoch: p.Epoch, PolicyDigest: staged.PolicyDigest, KeyProof: activationProof})
	if active.State != "active" {
		t.Fatal("CLI protected activation")
	}
	req := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(seed).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "joined-original", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	req.Enrollment.DisplayName = "joined after"
	action, _ := protojson.Marshal(req)
	digestRaw, _ := proto.MarshalOptions{Deterministic: true}.Marshal(req)
	sum := sha256.Sum256(digestRaw)
	originalOutput := runCLI("mutate", "", p, key, action)
	bootstrapID := func(output []byte) string {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "EXACT_BOOTSTRAP_RUN:") {
				return strings.TrimPrefix(line, "EXACT_BOOTSTRAP_RUN:")
			}
		}
		t.Fatal("actual CLI bootstrap missing")
		return ""
	}
	originalID := bootstrapID(originalOutput)
	originalToken := bootstrapTokens[originalID]
	original, err := o.VerifyIdentityToken(ctx, originalToken)
	if err != nil || !original.Valid || original.Claims.RunID.String() != originalID {
		t.Fatal("actual bootstrap original identity")
	}

	var count int
	_ = db.Get(&count, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key='reconcile-metadata:joined-original'`)
	if count != 1 {
		t.Fatal("actual CLI missing single effect receipt")
	}
	read := supervision.ExactReceiptRequest{OperationKey: "reconcile-metadata:" + req.IdempotencyKey, EffortRef: p.EffortRef, OriginalExecutor: original.Claims.RunID.String(), OriginalPayload: hex.EncodeToString(sum[:])}
	readJSON, _ := json.Marshal(read)
	var before string
	_ = db.Get(&before, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, p.EffortRef)
	readerID := bootstrapID(runCLI("read", "", p, key, readJSON))
	readerToken := bootstrapTokens[readerID]
	reader, err := o.VerifyIdentityToken(ctx, readerToken)
	if err != nil || !reader.Valid || reader.Claims.RunID.String() != readerID || readerID == originalID {
		t.Fatal("actual bootstrap fresh reader")
	}

	var after string
	_ = db.Get(&after, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, p.EffortRef)
	_ = db.Get(&count, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key='reconcile-metadata:joined-original'`)
	if requests != 11 || before != after || count != 1 {
		t.Fatal("read changed domain/receipt or missing actual eleven-wire launcher/setup/action/replay flow", requests, count)
	}
	// Genuine CreateRun parent/child uses the same installed standing grant.
	// The existing native owner independently authorizes runs; Keyring adds no
	// native execution scope. The synthetic runner executes no model/process.
	childToken, childID, finishNativeChild := exactJoinedNativeChild(t, repos, db)
	defer finishNativeChild()
	challengeRequest := httptest.NewRequest("POST", ExactReadChallengePath+"?grant="+p.ID, bytes.NewReader(readJSON))
	challengeRequest.Header.Set("X-Agent-Identity-Token", childToken)
	challengeResponse := httptest.NewRecorder()
	router.ServeHTTP(challengeResponse, challengeRequest)
	if challengeResponse.Code != 200 {
		t.Fatal("genuine child challenge", challengeResponse.Code, challengeResponse.Body.String())
	}
	var childChallenge supervision.ExactChallenge
	if json.Unmarshal(challengeResponse.Body.Bytes(), &childChallenge) != nil || childChallenge.Executor != childID {
		t.Fatal("child proof attribution")
	}
	childProof := exactSiteProof(t, childChallenge, p, key)
	childRequest := httptest.NewRequest("POST", ExactReceiptPath, bytes.NewReader(readJSON))
	childRequest.Header.Set("X-Agent-Identity-Token", childToken)
	childRequest.Header.Set(supervision.ExactProofHeader, childProof.Bytes)
	childRequest.Header.Set(supervision.ExactSignatureHeader, childProof.Signature)
	childResponse := httptest.NewRecorder()
	router.ServeHTTP(childResponse, childRequest)
	if childResponse.Code != 200 || !strings.Contains(childResponse.Body.String(), originalID) {
		t.Fatal("same original grant child read", childResponse.Code, childResponse.Body.String())
	}
	var approvals int
	if db.Get(&approvals, `SELECT COUNT(*) FROM keyring_exact_policy_setup`) != nil || approvals != 1 {
		t.Fatal("per-reader approval", approvals)
	}
	// Cleanup has no anonymous or cross-run admission and cannot stop native workers.
	for _, tc := range []struct{ name, target, token string }{
		{"absent", originalID, ""}, {"invalid", originalID, "invalid-fixture-leaf"}, {"foreign", originalID, readerToken}, {"not-attached", childID, childToken},
	} {
		t.Run("detach_"+tc.name, func(t *testing.T) {
			id := uuid.MustParse(tc.target)
			old, err := repos.Runs.Get(ctx, id)
			oldEvents, eventErr := events.Count(ctx, id)
			if eventErr != nil {
				t.Fatal(eventErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("POST", "/api/v1/runs/"+tc.target+"/detach", strings.NewReader(`{}`))
			if tc.token != "" {
				req.Header.Set("X-Agent-Identity-Token", tc.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != 401 && response.Code != 403 {
				t.Fatal("cleanup refusal", response.Code, response.Body.String())
			}
			current, err := repos.Runs.Get(ctx, id)
			newEvents, eventErr := events.Count(ctx, id)
			if eventErr != nil || newEvents != oldEvents || err != nil || current.Status != old.Status || current.IdentityTokenRevokedAt != nil {
				t.Fatal("refused cleanup effects", err)
			}
		})
	}

	// Actual mounted rejection paths preserve both domain and immutable receipt.
	signRead := func(change func(*supervision.ExactEnvelope)) (string, string) {
		hash := sha256.Sum256(readJSON)
		e := supervision.ExactEnvelope{Version: 1, Purpose: supervision.ExactReadPurpose, Instance: p.Instance, Audience: "agent-manager", Issuer: p.Issuer, KeyVersion: 1, Action: readJSON, Grant: p.ID, Epoch: 1, Initiator: p.Owner, Executor: reader.Claims.RunID.String(), Operation: "receipt.read", Target: "agent-manager:effort/joined", Payload: hex.EncodeToString(hash[:]), Request: read.OperationKey, Expires: time.Now().Add(30 * time.Second).Unix()}
		if change != nil {
			change(&e)
		}
		raw, _ := json.Marshal(e)
		return base64.RawURLEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))
	}
	for _, name := range []string{"absent proof", "duplicate proof", "expired proof", "mutation purpose", "wrong reader", "wrong target", "read proof on mutation", "proof on challenge"} {
		t.Run(name, func(t *testing.T) {
			signed, sig := signRead(nil)
			body := readJSON
			path := ExactReceiptPath
			token := readerToken
			switch name {
			case "expired proof":
				signed, sig = signRead(func(e *supervision.ExactEnvelope) { e.Expires = time.Now().Add(-time.Second).Unix() })
			case "mutation purpose":
				signed, sig = signRead(func(e *supervision.ExactEnvelope) { e.Purpose = supervision.ExactPurpose })
			case "wrong reader":
				token = originalToken
			case "wrong target":
				signed, sig = signRead(func(e *supervision.ExactEnvelope) { e.Target = "agent-manager:effort/other" })
			case "read proof on mutation":
				path = apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure
				body = action
			case "proof on challenge":
				path = ExactReadChallengePath
			}
			request := httptest.NewRequest("POST", path, bytes.NewReader(body))
			request.Header.Set("X-Agent-Identity-Token", token)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Connect-Protocol-Version", "1")
			if name != "absent proof" {
				request.Header.Set(supervision.ExactProofHeader, signed)
				request.Header.Set(supervision.ExactSignatureHeader, sig)
			}
			if name == "duplicate proof" {
				request.Header.Add(supervision.ExactProofHeader, signed)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != 403 {
				t.Fatal("mounted confusion accepted", response.Code)
			}
			var current string
			_ = db.Get(&current, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, p.EffortRef)
			_ = db.Get(&count, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key='reconcile-metadata:joined-original'`)
			if current != before || count != 1 {
				t.Fatal("rejection changed effect/receipt")
			}
		})
	}
}

// Disposable provider verifies only the explicitly scoped owner fixture. It
// never treats arbitrary bearer text or a native agent as owner proof.
type exactJoinedOwnerVerifier struct{ owner string }

func (v exactJoinedOwnerVerifier) Verify(_ context.Context, token string) (core.Principal, error) {
	if token != "disposable-verified-owner-fixture" {
		return core.Principal{}, core.NewFailure(core.FailureInvalid, core.SourceScenarioAuthenticator)
	}
	return core.Principal{Kind: core.ActorHuman, Verified: true, Source: core.SourceScenarioAuthenticator, Subject: v.owner, Scopes: []string{"agent-manager:write", "agent-manager:read"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// Actual owner-gated Site callable action and configured key adapter; public
// activation bytes must agree with the native staged policy digest.
func exactSiteActivation(t *testing.T, challenge *supervision.ExactActivationChallenge, policy supervision.ExactPolicy, key ed25519.PrivateKey) string {
	t.Helper()
	helper := os.Getenv("KEYRING_SITE_SIGN_HELPER")
	config, _ := json.Marshal(map[string]any{"challenge": challenge, "policy": policy, "owner": policy.Owner, "issuer": policy.Issuer, "key": key, "deadline": policy.Deadline})
	cmd := exec.Command("/home/matthalloran8/.nvm/versions/node/v22.23.2/bin/node", "--experimental-transform-types", "--experimental-loader", filepath.Join(filepath.Dir(helper), "loader.mjs"), helper)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdin = bytes.NewReader(config)
	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if cmd.Run() != nil {
		t.Fatal("actual Site activation refused", stderr.String(), output.String())
	}
	var proof struct{ Protocol, Bytes, KeyProof string }
	if json.Unmarshal(output.Bytes(), &proof) != nil || proof.Protocol != "keyring.native-activation.v1" || proof.KeyProof == "" {
		t.Fatal("actual Site activation contract")
	}
	return proof.KeyProof
}

type exactJoinedRoleResolver struct{}

func (exactJoinedRoleResolver) Resolve(_ context.Context, r domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: r, Role: role, Model: "fixture-model"}, nil
}

type exactJoinedRunner struct{ *runner.MockRunner }

func (*exactJoinedRunner) TagEnvKey() string  { return "SYNTHETIC_JOINED_TAG" }
func (*exactJoinedRunner) BinaryPath() string { return "/synthetic-in-process/claude" }
func (*exactJoinedRunner) ControlArgs(*domain.RunConfig) ([]string, error) {
	return []string{"--synthetic-in-process"}, nil
}

func (*exactJoinedRunner) RuntimeVersion(context.Context) (string, error) {
	return "synthetic-in-process/1", nil
}

func exactJoinedNativeChild(t *testing.T, repos *adb.Repositories, db *sqlx.DB) (string, string, func()) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "role-policy.json")
	if os.WriteFile(path, []byte(`{"schemaVersion":1,"metadata":{"catalogId":"joined-synthetic","updatedAt":"2026-10-11"},"defaultRole":"code.default","roles":{"code.default":{"description":"synthetic","intent":"synthetic","candidates":[{"runner":"claude-code","resourceRole":"code.default"}]}}}`), 0o600) != nil {
		t.Fatal("role fixture")
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	starts := make(chan runner.ExecuteRequest, 2)
	release := make(chan struct{})
	var once sync.Once
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		starts <- req
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "synthetic joined only"}}, nil
	}
	registry := runner.NewRegistry()
	if registry.Register(&exactJoinedRunner{mock}) != nil {
		t.Fatal("runner fixture")
	}
	cfg := orchestration.DefaultConfig()
	cfg.DefaultProjectRoot = root
	cfg.RequireSandboxByDefault = false
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 2, QueueCapacity: 4})
	native := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventstore.NewSQLiteStore(adb.NewDB(db, logrus.New()), logrus.New())), orchestration.WithConfig(cfg), orchestration.WithRunners(registry), orchestration.WithRolePolicyState(state, exactJoinedRoleResolver{}), orchestration.WithRunStateRoot(root), orchestration.WithSpawnDispatcher(dispatcher), orchestration.WithIdentitySecret([]byte("disposable-joined-native-secret")))
	owner := core.Principal{Kind: core.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: time.Now().Add(time.Hour)}
	ownerCtx := core.WithPrincipal(ctx, owner)
	profile, err := native.CreateProfile(ctx, &domain.AgentProfile{Name: "joined synthetic", ProfileKey: "joined-parent-child", RoleRef: "code.default", DeclaredScopes: owner.Scopes, MaxTurns: 5, Timeout: time.Minute, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	task, err := native.CreateTask(ctx, &domain.Task{Title: "joined synthetic parent/child", Description: "isolated in-process", ScopePath: root, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	wait := func() runner.ExecuteRequest {
		select {
		case req := <-starts:
			return req
		case <-time.After(10 * time.Second):
			for _, name := range []string{"joined-native-parent", "joined-native-child"} {
				saved, _ := repos.Runs.GetByIdempotencyKey(ctx, name)
				if saved != nil {
					t.Logf("native phase %s status=%s failure=%s", name, saved.Status, saved.ErrorMsg)
				}
			}
			var diagnostics []string
			_ = db.Select(&diagnostics, `SELECT data FROM run_events ORDER BY sequence`)
			for _, message := range diagnostics {
				t.Log("native phase event", message)
			}
			t.Fatal("native synthetic phase did not execute")
			return runner.ExecuteRequest{}
		}
	}
	parent, err := native.CreateRun(ownerCtx, orchestration.CreateRunRequest{TaskID: task.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", OwnerToken: "disposable-native-owner-fixture", IdempotencyKey: "joined-native-parent"})
	if err != nil {
		t.Fatal("native parent admission", err)
	}
	parentReq := wait()
	parentToken := parentReq.Environment[cliutil.EnvIdentityToken]
	verified, err := native.VerifyIdentityToken(ctx, parentToken)
	if err != nil || !verified.Valid || verified.Claims.RunID != parent.ID {
		t.Fatal("native phase parent custody", err)
	}
	child, err := native.CreateRun(ctx, orchestration.CreateRunRequest{TaskID: task.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID, RunIdentityToken: parentToken, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: "joined-native-child"})
	if err != nil {
		t.Fatal("native genuine child", err)
	}
	childReq := wait()
	childToken := childReq.Environment[cliutil.EnvIdentityToken]
	verified, err = native.VerifyIdentityToken(ctx, childToken)
	if err != nil || !verified.Valid || verified.Claims.RunID != child.ID || childToken == parentToken || len(verified.Claims.Scopes) != 1 || verified.Claims.Scopes[0] != "agent-manager:read" {
		t.Fatal("distinct narrowed native child custody", err)
	}
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	finish := func() {
		once.Do(func() { close(release) })
		defer dispatcher.Close()
		deadline := time.Now().Add(10 * time.Second)
		for {
			p, _ := repos.Runs.Get(ctx, parent.ID)
			c, _ := repos.Runs.Get(ctx, child.ID)
			if p != nil && c != nil && p.Status.IsTerminal() && c.Status.IsTerminal() {
				break
			}
			if time.Now().After(deadline) {
				t.Error("native synthetic cleanup timeout")
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		for _, token := range []string{parentToken, childToken} {
			if v, e := native.VerifyIdentityToken(ctx, token); e != nil || v.Valid {
				t.Error("terminal leaf retained", e)
			}
		}
	}
	return childToken, child.ID.String(), finish
}

type exactSiteSignedProof struct{ Protocol, Bytes, Signature string }

func exactSiteProof(t *testing.T, challenge supervision.ExactChallenge, policy supervision.ExactPolicy, key ed25519.PrivateKey) exactSiteSignedProof {
	t.Helper()
	helper := os.Getenv("KEYRING_SITE_SIGN_HELPER")
	raw, _ := json.Marshal(map[string]any{"challenge": challenge, "owner": policy.Owner, "issuer": policy.Issuer, "key": key, "deadline": policy.Deadline})
	cmd := exec.Command("/home/matthalloran8/.nvm/versions/node/v22.23.2/bin/node", "--experimental-transform-types", "--experimental-loader", filepath.Join(filepath.Dir(helper), "loader.mjs"), helper)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdin = bytes.NewReader(raw)
	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if cmd.Run() != nil {
		t.Fatal("actual Site child proof", stderr.String())
	}
	var proof exactSiteSignedProof
	if json.Unmarshal(output.Bytes(), &proof) != nil || proof.Protocol != "keyring.receipt-read.v1" || proof.Bytes == "" || proof.Signature == "" {
		t.Fatal("child proof contract")
	}
	return proof
}
