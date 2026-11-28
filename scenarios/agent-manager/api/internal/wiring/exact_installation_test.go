package wiring

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/handlers"
	"agent-manager/internal/modules"
	"agent-manager/internal/supervision"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

// Uses ordinary route assembly and module-owned schema with no manually
// supplied verifier. All provider identities are explicit disposable fixtures.
func TestExactOrdinaryWiringProtectedSetupAndSameGrantRead(t *testing.T) {
	t.Setenv("VROOLI_AUTH_PROVIDERS", "")
	t.Setenv("VROOLI_EVENTS_API_BASE", "http://fixture.invalid")
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "wired.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	efforts := supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{})
	router := mux.NewRouter()
	SetupRoutes(router, RouteDependencies{SupervisionService: &supervision.Service{Efforts: efforts}})
	owner := identity.Principal{Kind: identity.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "agent-manager:read"}, ExpiresAt: time.Now().Add(time.Hour)}
	agent := identity.Principal{Kind: identity.ActorAgent, Verified: true, Subject: uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}
	call := func(path string, body []byte, pr identity.Principal, proof, sig string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, bytes.NewReader(body))
		req = req.WithContext(identity.WithPrincipal(ctx, pr))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		req.Header.Set("Origin", "http://example.com")
		if proof != "" {
			req.Header.Set(supervision.ExactProofHeader, proof)
			req.Header.Set(supervision.ExactSignatureHeader, sig)
		}
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	p := supervision.ExactPolicy{ID: "wired-standing", Owner: owner.Subject, Instance: "wired-native", Issuer: "wired-keyring", KeyVersion: 1, Epoch: 1, Deadline: time.Now().Add(time.Hour).Unix(), EffortRef: "effort:wired", AuthorizedBy: owner.Subject, AuthorityRef: "fixture-approval", Profile: supervision.ExactProfile, PublicKey: public, Operations: []string{"effort.reconcile", "receipt.read"}, ReadEligibility: supervision.ExactStandingReadEligibility}
	seed := &pb.EffortEnrollment{EffortRef: p.EffortRef, Revision: 1, DisplayName: "before", AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}
	if err = repo.SaveEffort(ctx, seed, &pb.EffortBoardRow{}, 0, "seed", "seed"); err != nil {
		t.Fatal(err)
	}
	req := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(seed).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "wired-original", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	req.Enrollment.DisplayName = "after"
	body, _ := protojson.Marshal(req)
	if out := call(handlers.ExactChallengePath+"?grant="+p.ID, body, agent, "", ""); out.Code != 403 {
		t.Fatal("empty setup did not fail closed", out.Code)
	}
	raw, _ := json.Marshal(supervision.ExactSetupRequest{Action: "stage", Policy: &p})
	out := call(handlers.ExactSetupPath, raw, owner, "", "")
	if out.Code != 200 {
		t.Fatal("normal protected setup missing", out.Code, out.Body.String())
	}
	var staged supervision.ExactSetupReceipt
	if json.Unmarshal(out.Body.Bytes(), &staged) != nil || staged.State != "staged" {
		t.Fatal("stage receipt")
	}
	if out = call(handlers.ExactChallengePath+"?grant="+p.ID, body, agent, "", ""); out.Code != 403 {
		t.Fatal("staged authority active")
	}
	activation, _ := json.Marshal(staged.ActivationChallenge)
	raw, _ = json.Marshal(supervision.ExactSetupRequest{Action: "activate", Grant: p.ID, ExpectedEpoch: 1, PolicyDigest: staged.PolicyDigest, KeyProof: base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, activation))})
	out = call(handlers.ExactSetupPath, raw, owner, "", "")
	if out.Code != 200 {
		t.Fatal("normal activation", out.Code, out.Body.String())
	}
	signed := func(challengePath string, action []byte, actor identity.Principal) (string, string) {
		out := call(challengePath+"?grant="+p.ID, action, actor, "", "")
		if out.Code != 200 {
			t.Fatal("wired challenge", out.Code, out.Body.String())
		}
		var c supervision.ExactChallenge
		if json.Unmarshal(out.Body.Bytes(), &c) != nil {
			t.Fatal("challenge decode")
		}
		e := supervision.ExactEnvelope{Version: 1, Purpose: c.Purpose, Instance: c.Instance, Audience: c.Audience, Issuer: p.Issuer, KeyVersion: 1, Action: action, Grant: p.ID, Epoch: 1, Initiator: p.Owner, Executor: c.Executor, Operation: c.Operation, Target: c.Target, Payload: c.Payload, Request: c.Request, Expires: time.Now().Add(30 * time.Second).Unix()}
		raw, _ := json.Marshal(e)
		return base64.RawURLEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))
	}
	proof, sig := signed(handlers.ExactChallengePath, body, agent)
	out = call(apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure, body, agent, proof, sig)
	if out.Code != 200 {
		t.Fatal("same-store mutation seal failed", out.Code, out.Body.String())
	}
	deterministic, _ := proto.MarshalOptions{Deterministic: true}.Marshal(req)
	hash := sha256.Sum256(deterministic)
	read := supervision.ExactReceiptRequest{OperationKey: "reconcile-metadata:" + req.IdempotencyKey, EffortRef: p.EffortRef, OriginalExecutor: agent.Subject, OriginalPayload: hex.EncodeToString(hash[:])}
	readBody, _ := json.Marshal(read)
	for i := 0; i < 3; i++ {
		reader := agent
		reader.Subject = uuid.NewString()
		proof, sig = signed(handlers.ExactReadChallengePath, readBody, reader)
		out = call(handlers.ExactReceiptPath, readBody, reader, proof, sig)
		if out.Code != 200 {
			t.Fatal("same-store read seal failed", out.Code, out.Body.String())
		}
		var receipt supervision.ExactReceipt
		if json.Unmarshal(out.Body.Bytes(), &receipt) != nil || receipt.OriginalExecutor != agent.Subject || receipt.OriginalPayload != read.OriginalPayload {
			t.Fatal("original provenance rewritten")
		}
	}
	var count int
	if db.Get(&count, `SELECT COUNT(*) FROM keyring_exact_policy_setup`) != nil || count != 1 {
		t.Fatal("reader required another approval", count)
	}
	if db.Get(&count, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key='reconcile-metadata:wired-original'`) != nil || count != 1 {
		t.Fatal("read mutated receipt", count)
	}
}
