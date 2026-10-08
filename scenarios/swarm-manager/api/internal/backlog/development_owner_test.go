package backlog

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/effortauthority"
	sharedidentity "github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/proto"
	"swarm-manager/internal/identity"
)

// These owner/adapter fixtures never issue a token, install a live authority,
// invoke a launcher or substitute fixture evidence for owner product receipts.
func developmentFixture(t *testing.T) (*DevelopmentOwner, context.Context, *FileEffortControlStore, DevelopmentContract) {
	t.Helper()
	h, _ := setupTestHandler(t)
	store := NewFileEffortControlStore(t.TempDir())
	c := testEffortControlRequest("development-fixture", 1)
	c.PolicyBinding = identity.PolicyBinding{Source: "fixture/effort.json", Digest: developmentDigest("fixture policy")}
	c.CandidatePolicy = testEffortCandidate("fixture/model")
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	service := NewEffortControlService(store, nil)
	admitted, err := service.Admit(c)
	if err != nil {
		t.Fatal(err)
	}
	c = admitted
	h.SetEffortControlService(service)
	finite, err := NewFiniteCommissionOwner(h, map[string]FiniteCommissionTarget{c.EffortID: {WorkShape: FiniteBoundedAuthority}})
	if err != nil {
		t.Fatal(err)
	}
	subject := effortauthority.CommissionSubject{Owner: "fixture-owner", Effort: c.EffortID, Revision: "1", ContentDigest: strings.TrimPrefix(c.AuthorityDigest(), "sha256:"), Team: "fixture-team", Members: []string{"fixture-leader"}, Repository: t.TempDir(), TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Profiles: map[string]string{"native": effortauthority.Digest("profile")}}
	root, retained := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios/swarm-manager/api"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios/swarm-manager/api/main.go"), []byte("package fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	artifact := []byte("retained fixture evidence\n")
	if err := os.WriteFile(filepath.Join(retained, "evidence.md"), artifact, 0600); err != nil {
		t.Fatal(err)
	}
	contract := DevelopmentContract{Subject: subject, Owner: DevelopmentPrincipalBinding{Kind: sharedidentity.ActorHuman, Subject: subject.Owner, Source: sharedidentity.SourcePersonalLocal, Realm: "fixture-local"}, RequiredCriteria: []string{"auth.caller.identity"}, Effects: []string{"run.create"}, SourceRoot: root, RetainedRoot: retained, Artifacts: []DevelopmentArtifactBinding{{ID: "fixture-artifact", RelativePath: "evidence.md", SHA256: developmentBytesDigest(artifact), SnapshotDigest: developmentDigest("snapshot"), SizeBytes: int64(len(artifact)), MediaType: "text/markdown", CapturedAt: time.Now().UTC()}}}
	owner, err := NewDevelopmentOwner(finite, map[string]DevelopmentContract{c.EffortID: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	principal := sharedidentity.Principal{Kind: sharedidentity.ActorHuman, Subject: subject.Owner, Source: sharedidentity.SourcePersonalLocal, Realm: contract.Owner.Realm, Verified: true, Scopes: []string{"swarm-manager:read", "swarm-manager:write"}}
	return owner, sharedidentity.WithPrincipal(context.Background(), principal), store, contract
}
func developmentGeneration(value uint64) *uint64 { return &value }

func TestDevelopmentOwnerApproveReplayRevokePreservesAuthoredState(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	before, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	view, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if view.Approved || view.Accounting.GetTokens() != 0 || view.Accounting.Tokens != nil || len(view.LaunchBlockers) < 3 {
		t.Fatalf("unknown standing presented as qualified: %v", view)
	}
	req := &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve-1"}
	first, err := owner.ApproveDevelopment(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := owner.ApproveDevelopment(ctx, req)
	if err != nil || !proto.Equal(first, replay) {
		t.Fatalf("replay changed receipt: %v %v", replay, err)
	}
	current, _ := store.Load(contract.Subject.Effort)
	if current.FiniteCommission == nil || current.FiniteCommission.DevelopmentContractDigest != view.Reference.ContractDigest || current.FiniteCommission.Generation != 1 || current.Development == nil || len(current.Development.Decisions) != 1 {
		t.Fatalf("approval not atomically retained: %+v", current)
	}
	authored := current
	authored.FiniteCommission = nil
	authored.Development = nil
	if !reflect.DeepEqual(before, authored) {
		t.Fatal("approval changed authored authority/completion")
	}
	rev, err := owner.RevokeDevelopment(ctx, &api.RevokeDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(1), RequestId: "revoke-1"})
	if err != nil || rev.Generation != 2 {
		t.Fatalf("revoke: %v %v", rev, err)
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !after.FiniteCommission.Revoked || after.Development.DispositionVersion != 0 || len(after.Development.Decisions) != 2 || !reflect.DeepEqual(before.Completion, after.Completion) {
		t.Fatal("revocation renewed/accepted work or lost obligations")
	}
	if err := store.Save(current); err == nil {
		t.Fatal("ordinary stale save restored revoked standing")
	}
	replay, err = owner.ApproveDevelopment(ctx, req)
	if err != nil || !proto.Equal(first, replay) {
		t.Fatal("historical same-reference replay restored approval or changed receipt")
	}
	final, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(after, final) {
		t.Fatal("historical replay mutated standing")
	}
}

func TestDevelopmentOwnerRejectedIdentityHasNoDurableEffects(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	view, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	valid, ok := sharedidentity.PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("fixture principal missing")
	}
	cases := map[string]sharedidentity.Principal{"absent": {}, "unverified": valid, "wrong-subject": valid, "agent-channel": valid, "wrong-realm": valid, "wrong-source": valid, "expired": valid, "read-only": valid}
	x := cases["unverified"]
	x.Verified = false
	cases["unverified"] = x
	x = cases["wrong-subject"]
	x.Subject = "other"
	cases["wrong-subject"] = x
	x = cases["agent-channel"]
	x.Kind = sharedidentity.ActorAgent
	cases["agent-channel"] = x
	x = cases["wrong-realm"]
	x.Realm = "other"
	cases["wrong-realm"] = x
	x = cases["wrong-source"]
	x.Source = sharedidentity.SourceUnknown
	cases["wrong-source"] = x
	x = cases["expired"]
	x.ExpiresAt = time.Now().Add(-time.Second)
	cases["expired"] = x
	x = cases["read-only"]
	x.Scopes = []string{"swarm-manager:read"}
	cases["read-only"] = x
	before, _ := store.Load(contract.Subject.Effort)
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			rejected := sharedidentity.WithPrincipal(context.Background(), p)
			if _, err := owner.ApproveDevelopment(rejected, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "reject-" + name}); err == nil {
				t.Fatal("identity accepted")
			}
			after, _ := store.Load(contract.Subject.Effort)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("refusal changed durable owner state")
			}
		})
	}
}

func TestDevelopmentOwnerContractMigrationDoesNotGrandfatherApproval(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	view, _ := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if _, err := owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve"}); err != nil {
		t.Fatal(err)
	}
	changed := contract
	changed.RequiredCriteria = []string{"different.denominator"}
	replacement, err := NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: changed}, nil)
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := replacement.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Approved || migrated.Reference.ContractDigest == view.Reference.ContractDigest {
		t.Fatal("contract replacement reused old approval")
	}
	before, _ := store.Load(contract.Subject.Effort)
	if _, err := replacement.AcceptDevelopment(ctx, &api.AcceptDevelopmentRequest{Reference: migrated.Reference, ExpectedGeneration: developmentGeneration(1), ExpectedDispositionVersion: developmentGeneration(0), RequestId: "accept", TestedProductDigest: developmentDigest("product")}); err == nil {
		t.Fatal("changed denominator accepted")
	}
	if _, err := replacement.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: migrated.Reference, ExpectedGeneration: developmentGeneration(1), RequestId: "renew"}); err == nil {
		t.Fatal("same reviewed revision silently renewed")
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("migration refusals changed state")
	}
}

func TestDevelopmentOwnerUnavailableEvidenceNeverAcceptsProduct(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	view, _ := owner.GetDevelopment(ctx, contract.Subject.Effort)
	for _, approved := range []bool{false, true} {
		if approved {
			if _, err := owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve"}); err != nil {
				t.Fatal(err)
			}
		}
		before, _ := store.Load(contract.Subject.Effort)
		gen := uint64(0)
		if approved {
			gen = 1
		}
		if _, err := owner.AcceptDevelopment(ctx, &api.AcceptDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(gen), ExpectedDispositionVersion: developmentGeneration(0), RequestId: "accept", TestedProductDigest: developmentDigest("product")}); err == nil {
			t.Fatal("unavailable evidence accepted")
		}
		after, _ := store.Load(contract.Subject.Effort)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("failed acceptance changed standing")
		}
	}
}

func TestDevelopmentOwnerArtifactsAndPreviewAreBoundedReadOnly(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	view, _ := owner.GetDevelopment(ctx, contract.Subject.Effort)
	before, _ := store.Load(contract.Subject.Effort)
	artifact, err := owner.GetDevelopmentArtifact(ctx, view.Reference, "fixture-artifact")
	if err != nil || string(artifact.Content) != "retained fixture evidence\n" {
		t.Fatalf("artifact: %v %v", artifact, err)
	}
	if err := os.WriteFile(filepath.Join(contract.RetainedRoot, "evidence.md"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.GetDevelopmentArtifact(ctx, view.Reference, "fixture-artifact"); err == nil {
		t.Fatal("corrupt retained bytes returned")
	}
	proposal := &api.DevelopmentProposal{WorkShape: string(FiniteBoundedAuthority), Title: "Fixture preview", RequiredCriterionIds: []string{"auth.caller.identity"}, ScopeAllow: view.ScopeAllow, ScopeDeny: view.ScopeDeny, Limits: view.Limits, SourceRelativePaths: []string{"scenarios/swarm-manager/api/main.go"}, ProposedEffects: []string{"run.create"}, TargetSubjectRef: view.Reference.CommissionSubjectDigest}
	preview, err := owner.PreviewDevelopment(ctx, &api.PreviewDevelopmentRequest{Reference: view.Reference, Proposal: proposal})
	if err != nil || len(preview.ObservedSources) != 1 || len(preview.LaunchBlockers) == 0 {
		t.Fatalf("preview: %v %v", preview, err)
	}
	for _, p := range []string{"../outside.go", ".env", "runtime.db"} {
		q := proto.Clone(proposal).(*api.DevelopmentProposal)
		q.SourceRelativePaths = []string{p}
		if _, err := owner.PreviewDevelopment(ctx, &api.PreviewDevelopmentRequest{Reference: view.Reference, Proposal: q}); err == nil {
			t.Fatalf("unreviewable path %s exposed", p)
		}
	}
	changed := proto.Clone(proposal).(*api.DevelopmentProposal)
	changed.Limits.MaxWorkers++
	conflict, err := owner.PreviewDevelopment(ctx, &api.PreviewDevelopmentRequest{Reference: view.Reference, Proposal: changed})
	if err != nil || len(conflict.Conflicts) == 0 || conflict.ProposalDigest == preview.ProposalDigest {
		t.Fatal("changed proposed ceiling not visible")
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only artifact/preview changed owner state")
	}
	view.RequiredCriterionIds[0] = "mutated"
	next, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil || next.RequiredCriterionIds[0] != "auth.caller.identity" {
		t.Fatal("output mutated installed contract")
	}
}

func TestDevelopmentActualAdapterRefusesMissingIdentityAndOwner(t *testing.T) {
	service := &DevelopmentService{}
	if _, err := service.GetDevelopment(context.Background(), connect.NewRequest(&api.GetDevelopmentRequest{EffortId: "fixture"})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("missing identity: %v", err)
	}
	_, ctx, _, _ := developmentFixture(t)
	if _, err := service.GetDevelopment(ctx, connect.NewRequest(&api.GetDevelopmentRequest{EffortId: "fixture"})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("uninstalled owner: %v", err)
	}
}

func TestDevelopmentOwnerPreservesActualVerifiedAgentReadForms(t *testing.T) {
	owner, human, store, contract := developmentFixture(t)
	for _, binding := range []DevelopmentPrincipalBinding{
		{Kind: sharedidentity.ActorAgent, Subject: "fixture-run", Source: sharedidentity.SourceAgentProvenance},
		{Kind: sharedidentity.ActorAgent, Subject: "fixture-jwt-agent", Source: sharedidentity.SourceScenarioAuthenticator, Issuer: "fixture-issuer", Realm: "fixture-realm"},
	} {
		t.Run(string(binding.Source), func(t *testing.T) {
			configured := contract
			configured.Readers = []DevelopmentPrincipalBinding{binding}
			readerOwner, err := NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: configured}, nil)
			if err != nil {
				t.Fatal(err)
			}
			ctx := sharedidentity.WithPrincipal(context.Background(), sharedidentity.Principal{Kind: binding.Kind, Subject: binding.Subject, Source: binding.Source, Issuer: binding.Issuer, Realm: binding.Realm, Verified: true, Scopes: []string{"swarm-manager:read", "swarm-manager:write"}})
			view, err := readerOwner.GetDevelopment(ctx, contract.Subject.Effort)
			if err != nil {
				t.Fatal("supported verified agent read refused", err)
			}
			before, _ := store.Load(contract.Subject.Effort)
			if _, err := readerOwner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "agent-approve"}); err == nil {
				t.Fatal("reader acquired human write authority")
			}
			after, _ := store.Load(contract.Subject.Effort)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("agent write refusal changed standing")
			}
			if _, err := readerOwner.GetDevelopment(human, contract.Subject.Effort); err != nil {
				t.Fatal("reader binding displaced exact human owner")
			}
		})
	}
}

func TestDevelopmentHistoricalArtifactAndReplaySurviveAmendAndRestart(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	old, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	req := &api.ApproveDevelopmentRequest{Reference: old.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "original-approve"}
	original, err := owner.ApproveDevelopment(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	current.Revision++
	current.Development = nil // Requests never need to echo protected runtime history.
	amended, err := owner.finite.handler.effortControl.Amend(current)
	if err != nil {
		t.Fatal(err)
	}
	if amended.Development == nil || len(amended.Development.History) != 1 {
		t.Fatal("ordinary amendment lost retained history")
	}
	contract.Subject.Revision = "2"
	contract.Subject.ContentDigest = strings.TrimPrefix(amended.AuthorityDigest(), "sha256:")
	oldRoot := contract.RetainedRoot
	contract.RetainedRoot = t.TempDir()
	if err := os.RemoveAll(oldRoot); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := restarted.GetDevelopmentArtifact(ctx, old.Reference, "fixture-artifact")
	if err != nil || string(artifact.Content) != "retained fixture evidence\n" {
		t.Fatalf("original immutable artifact unavailable after restart/amend: %v %v", artifact, err)
	}
	before, err := store.Load(contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := restarted.ApproveDevelopment(ctx, req)
	if err != nil || !proto.Equal(original, replay) {
		t.Fatalf("historical exact replay changed: %v %v", replay, err)
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("historical replay restored admission or changed current owner state")
	}
	latest, err := restarted.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil || latest.Approved || latest.OutcomeAccepted || proto.Equal(latest.Reference, old.Reference) {
		t.Fatalf("old approval projected as current: %v %v", latest, err)
	}
	different := proto.Clone(req).(*api.ApproveDevelopmentRequest)
	different.ExpectedGeneration = developmentGeneration(1)
	if _, err := restarted.ApproveDevelopment(ctx, different); err == nil {
		t.Fatal("historical key accepted changed payload")
	}
	after, _ = store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected historical payload changed state")
	}
}

func TestDevelopmentHistoricalVisibilityUsesCurrentReaders(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	reader := DevelopmentPrincipalBinding{Kind: sharedidentity.ActorAgent, Subject: "retained-fixture-reader", Source: sharedidentity.SourceAgentProvenance}
	contract.Readers = []DevelopmentPrincipalBinding{reader}
	original, err := NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, _ := original.GetDevelopment(ctx, contract.Subject.Effort)
	if _, err := original.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve-with-reader"}); err != nil {
		t.Fatal(err)
	}
	readerContext := sharedidentity.WithPrincipal(context.Background(), sharedidentity.Principal{Kind: reader.Kind, Subject: reader.Subject, Source: reader.Source, Verified: true, Scopes: []string{"swarm-manager:read"}})
	if _, err := original.GetDevelopmentArtifact(readerContext, view.Reference, "fixture-artifact"); err != nil {
		t.Fatal("original verified reader refused", err)
	}
	contract.Readers = nil
	migrated, err := NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: contract}, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := store.Load(contract.Subject.Effort)
	if _, err := migrated.GetDevelopmentArtifact(readerContext, view.Reference, "fixture-artifact"); err == nil {
		t.Fatal("historical ACL restored removed reader access")
	}
	after, _ := store.Load(contract.Subject.Effort)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("historical read refusal changed state")
	}
}

func TestDevelopmentCorruptHistoryCannotFallBackToMutableInputs(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	view, _ := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if _, err := owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve"}); err != nil {
		t.Fatal(err)
	}
	// Simulate corruption through the trusted fixture store seam; public Save
	// must continue refusing body changes to history.
	err := store.withFiniteLock(contract.Subject.Effort, func() error {
		c, e := store.Load(contract.Subject.Effort)
		if e != nil {
			return e
		}
		h := c.Development.History[developmentDigest(view.Reference)]
		h.ArtifactBytes["fixture-artifact"] = []byte("corrupt")
		c.Development.History[developmentDigest(view.Reference)] = h
		return store.saveUnlocked(c)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.GetDevelopmentArtifact(ctx, view.Reference, "fixture-artifact"); err == nil {
		t.Fatal("corrupt immutable history fell back to mutable original input")
	}
}

// This controlled owner exercises persistence only. It does not qualify a
// current-product fence, and is never installed in the concrete server.
type developmentPersistenceEvidenceFixture struct {
	snapshot   DevelopmentEvidenceSnapshot
	guardError error
}

func (f *developmentPersistenceEvidenceFixture) ReadDevelopmentEvidence(context.Context, *api.DevelopmentReference, []string) (DevelopmentEvidenceSnapshot, error) {
	return f.snapshot, nil
}
func (f *developmentPersistenceEvidenceFixture) CommitDevelopmentEvidence(_ context.Context, _ *api.DevelopmentReference, _ DevelopmentEvidenceSnapshot, save func(func() error) error) error {
	return save(func() error { return f.guardError })
}
func TestDevelopmentEvidenceHistoryAppendConflictAndBound(t *testing.T) {
	owner, ctx, store, contract := developmentFixture(t)
	f := &developmentPersistenceEvidenceFixture{}
	owner.evidence = f
	view, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve-history"}); err != nil {
		t.Fatal(err)
	}
	set := func(id string, b []byte) {
		product := developmentDigest("product-" + id)
		f.snapshot = DevelopmentEvidenceSnapshot{ProductDigest: product, EvidenceSetDigest: developmentDigest("set-" + id), Version: "fixture", Receipts: []DevelopmentRetainedEvidence{{ID: id, Producer: "test-genie", Digest: developmentBytesDigest(b), Binding: []byte("exact binding"), Bytes: b}}, Criteria: []*api.DevelopmentCriterionEvidence{{CriterionId: contract.RequiredCriteria[0], ReceiptIds: []string{id}, ReceiptSetDigest: developmentDigest("set-" + id), TestedProductDigest: product, Satisfied: true}}}
	}
	accept := func(key string, v uint64) error {
		_, e := owner.AcceptDevelopment(ctx, &api.AcceptDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(1), ExpectedDispositionVersion: developmentGeneration(v), RequestId: key, TestedProductDigest: f.snapshot.ProductDigest})
		return e
	}
	set("first", []byte("first immutable producer receipt"))
	if err = accept("accept-first", 0); err != nil {
		t.Fatal(err)
	}
	first, _ := store.Load(contract.Subject.Effort)
	preserved := first.Development.EvidenceReceipts["first"]
	set("second", []byte("second immutable producer receipt"))
	if err = accept("accept-second", 1); err != nil {
		t.Fatal(err)
	}
	second, _ := store.Load(contract.Subject.Effort)
	if len(second.Development.EvidenceReceipts) != 2 || !reflect.DeepEqual(preserved, second.Development.EvidenceReceipts["first"]) {
		t.Fatal("later decision lost earlier exact evidence")
	}
	owner, err = NewDevelopmentOwner(owner.finite, map[string]DevelopmentContract{contract.Subject.Effort: contract}, f)
	if err != nil {
		t.Fatal(err)
	}
	set("first", []byte("first immutable producer receipt"))
	if err = accept("accept-first", 0); err != nil {
		t.Fatal("original decision did not replay after owner reconstruction", err)
	}
	if err = accept("accept-identical-reuse", 2); err != nil {
		t.Fatal("exact immutable evidence reuse refused", err)
	}
	reused, _ := store.Load(contract.Subject.Effort)
	if len(reused.Development.EvidenceReceipts) != 2 || !reflect.DeepEqual(preserved, reused.Development.EvidenceReceipts["first"]) {
		t.Fatal("reconstruction/reuse changed retained bytes")
	}
	for _, test := range []struct {
		id   string
		data []byte
	}{{"first", []byte("conflicting reused producer ID")}, {"overflow", make([]byte, developmentMaxHistoryBytes)}} {
		before, _ := store.Load(contract.Subject.Effort)
		set(test.id, test.data)
		if accept("refuse-"+test.id, 3) == nil {
			t.Fatal("conflict or bounded overflow accepted")
		}
		after, _ := store.Load(contract.Subject.Effort)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("rejection changed canonical owner state")
		}
	}
}
