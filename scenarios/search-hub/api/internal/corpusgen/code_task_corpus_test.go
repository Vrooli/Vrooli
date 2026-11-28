package corpusgen

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// These fixtures exercise metadata admission mechanics only. They are not
// evaluation cases and make no claim about coding-model quality.
type acceptingCodeTaskOwners struct{}

func (acceptingCodeTaskOwners) VerifyCodeTaskSource(context.Context, CodeTaskSourceIdentityBinding) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskActorIdentity(context.Context, CodeTaskSourceIdentityBinding) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRoute(context.Context, GatewayRouteProof) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRights(context.Context, string, string) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskPrivacy(context.Context, string, CodeTaskCorpusConsentBinding) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskRepository(context.Context, CodeTaskRepositoryBase) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskObjectiveSuite(context.Context, CodeTaskRepositoryBase, CodeTaskObjectiveSuite) error {
	return nil
}
func (acceptingCodeTaskOwners) VerifyCodeTaskIndependentLabel(context.Context, CodeTaskIndependentLabelRef, CodeTaskIndependentLabelBinding) error {
	return nil
}

type recordingCodeTaskOwners struct {
	acceptingCodeTaskOwners
	sourceBinding   CodeTaskSourceIdentityBinding
	sourceCalls     int
	privacyBinding  CodeTaskCorpusConsentBinding
	labelBinding    CodeTaskIndependentLabelBinding
	privacyCalls    int
	labelCalls      int
	identityBinding CodeTaskSourceIdentityBinding
	identityCalls   int
}

func (o *recordingCodeTaskOwners) VerifyCodeTaskSource(_ context.Context, binding CodeTaskSourceIdentityBinding) error {
	o.sourceBinding = binding
	o.sourceCalls++
	return nil
}

type testCodeTaskEvidenceOwner interface {
	CodeTaskSourceEvidenceOwner
	CodeTaskSourceIdentityEvidenceOwner
	CodeTaskRouteEvidenceOwner
	CodeTaskRightsEvidenceOwner
	CodeTaskPrivacyEvidenceOwner
	CodeTaskRepositoryEvidenceOwner
	CodeTaskObjectiveEvidenceOwner
	CodeTaskIndependentLabelEvidenceOwner
}

func (o *recordingCodeTaskOwners) VerifyCodeTaskActorIdentity(_ context.Context, binding CodeTaskSourceIdentityBinding) error {
	o.identityBinding = binding
	o.identityCalls++
	return nil
}

func (o *recordingCodeTaskOwners) VerifyCodeTaskPrivacy(_ context.Context, _ string, binding CodeTaskCorpusConsentBinding) error {
	o.privacyCalls++
	o.privacyBinding = binding
	return nil
}

func (o *recordingCodeTaskOwners) VerifyCodeTaskIndependentLabel(_ context.Context, _ CodeTaskIndependentLabelRef, binding CodeTaskIndependentLabelBinding) error {
	o.labelCalls++
	o.labelBinding = binding
	return nil
}

func TestBuildCodeTaskCorpusRetainsRejectionsAndDoesNotSealBelowCoverage(t *testing.T) {
	candidate := codeTaskCandidateFixture(1, "single-file-change")
	candidate.Source.MessageSHA256 = "short"
	missingSuiteReceipt := codeTaskCandidateFixture(2, "single-file-change")
	missingSuiteReceipt.ObjectiveSuite.ValidationReceiptSHA256 = ""
	snapshot := BuildCodeTaskCorpus(context.Background(), []CodeTaskCandidate{candidate, missingSuiteReceipt}, "split-v1", testCodeTaskEvidenceAdapter(t, acceptingCodeTaskOwners{}))
	if snapshot.Sealed || len(snapshot.Cases) != 0 || len(snapshot.Rejections) != 2 || snapshot.Rejections[0].ReasonCode != "source_identity_incomplete" || snapshot.Rejections[1].ReasonCode != "objective_test_suite_identity_incomplete" {
		t.Fatalf("expected rejected source and unsealed corpus, got %+v", snapshot)
	}
	if len(snapshot.CoverageDeficits) == 0 {
		t.Fatal("unsealed corpus must report deficits")
	}
}

func TestBuildCodeTaskCorpusPassesCompleteIdentityToPrivacyAndLabelOwners(t *testing.T) {
	candidate := codeTaskCandidateFixture(42, "single-file-change")
	owners := &recordingCodeTaskOwners{}
	adapter, err := NewCodeTaskEvidenceOwnerAdapter(testCodeTaskOwnerClients(owners))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := BuildCodeTaskCorpus(context.Background(), []CodeTaskCandidate{candidate}, "split-v1", adapter)
	if len(snapshot.Cases) != 1 || len(snapshot.Rejections) != 0 {
		t.Fatalf("complete source fixture should pass mechanics owners: %+v", snapshot.Rejections)
	}
	wantPrivacy := CodeTaskCorpusConsentBinding{
		Purpose: CodeTaskCorpusConsentPurpose, Source: candidate.Source,
		TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository,
		ObjectiveSuite: candidate.ObjectiveSuite, ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
	}
	wantLabel := CodeTaskIndependentLabelBinding{
		Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256,
		Repository: candidate.Repository, ObjectiveSuite: candidate.ObjectiveSuite,
		ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
	}
	if !reflect.DeepEqual(owners.privacyBinding, wantPrivacy) {
		t.Fatalf("privacy owner got an incomplete or misbound request: got=%+v want=%+v", owners.privacyBinding, wantPrivacy)
	}
	if !reflect.DeepEqual(owners.labelBinding, wantLabel) {
		t.Fatalf("independent-label owner got an incomplete or misbound request: got=%+v want=%+v", owners.labelBinding, wantLabel)
	}
	wantIdentityBinding := CodeTaskSourceIdentityBinding{Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository}
	if owners.sourceCalls != 1 || !reflect.DeepEqual(owners.sourceBinding, wantIdentityBinding) || owners.identityCalls != 1 || !reflect.DeepEqual(owners.identityBinding, wantIdentityBinding) || owners.privacyCalls != 1 || owners.labelCalls != 1 {
		t.Fatalf("source/identity/privacy/label owners were not called with exact bindings: source=%d identity=%d consent=%d label=%d sourceBinding=%+v identityBinding=%+v", owners.sourceCalls, owners.identityCalls, owners.privacyCalls, owners.labelCalls, owners.sourceBinding, owners.identityBinding)
	}
}

func TestCodeTaskEvidenceOwnerAdapterRefusesUnboundConsentAndLabelRequests(t *testing.T) {
	candidate := codeTaskCandidateFixture(50, "single-file-change")
	owners := &recordingCodeTaskOwners{}
	adapter, err := NewCodeTaskEvidenceOwnerAdapter(testCodeTaskOwnerClients(owners))
	if err != nil {
		t.Fatal(err)
	}
	unboundSource := candidate.Source
	unboundSource.ActorIdentityReceiptID = ""
	if err := adapter.VerifyCodeTaskActorIdentity(context.Background(), CodeTaskSourceIdentityBinding{Source: unboundSource, TaskSpecSHA256: "bad", Repository: codeTaskCandidateFixture(0, "single-file-change").Repository}); err == nil {
		t.Fatal("adapter accepted a source without an owner identity receipt")
	}
	if owners.identityCalls != 0 {
		t.Fatal("incomplete source identity reached the identity owner")
	}
	privacy := CodeTaskCorpusConsentBinding{
		Purpose: CodeTaskCorpusConsentPurpose, Source: candidate.Source,
		TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository,
		ObjectiveSuite: candidate.ObjectiveSuite, ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
	}
	label := CodeTaskIndependentLabelBinding{
		Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256,
		Repository: candidate.Repository, ObjectiveSuite: candidate.ObjectiveSuite,
		ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
	}
	privacyMutations := map[string]func(*CodeTaskCorpusConsentBinding){
		"source_run":        func(b *CodeTaskCorpusConsentBinding) { b.Source.RunID = "" },
		"source_event":      func(b *CodeTaskCorpusConsentBinding) { b.Source.EventID = "" },
		"source_message":    func(b *CodeTaskCorpusConsentBinding) { b.Source.MessageSHA256 = "" },
		"task":              func(b *CodeTaskCorpusConsentBinding) { b.TaskSpecSHA256 = "" },
		"repository":        func(b *CodeTaskCorpusConsentBinding) { b.Repository.RepositoryID = "" },
		"repository_rev":    func(b *CodeTaskCorpusConsentBinding) { b.Repository.Revision = strings.Repeat("c", 40) },
		"repository_tree":   func(b *CodeTaskCorpusConsentBinding) { b.Repository.TreeSHA256 = "" },
		"suite_receipt":     func(b *CodeTaskCorpusConsentBinding) { b.ObjectiveSuite.ValidationReceiptID = "" },
		"expected_behavior": func(b *CodeTaskCorpusConsentBinding) { b.ExpectedBehaviorSHA256 = "" },
		"purpose":           func(b *CodeTaskCorpusConsentBinding) { b.Purpose = "remote-inference" },
	}
	for name, mutate := range privacyMutations {
		t.Run("consent/"+name, func(t *testing.T) {
			changed := privacy
			mutate(&changed)
			if err := adapter.VerifyCodeTaskPrivacy(context.Background(), "privacy-decision", changed); err == nil {
				t.Fatal("owner adapter accepted an incomplete or misbound privacy request")
			}
			if owners.privacyCalls != 0 {
				t.Fatal("invalid privacy binding reached the external owner")
			}
		})
	}
	labelMutations := map[string]func(*CodeTaskIndependentLabelBinding){
		"source_run":        func(b *CodeTaskIndependentLabelBinding) { b.Source.RunID = "" },
		"source_event":      func(b *CodeTaskIndependentLabelBinding) { b.Source.EventID = "" },
		"task":              func(b *CodeTaskIndependentLabelBinding) { b.TaskSpecSHA256 = "" },
		"repository_rev":    func(b *CodeTaskIndependentLabelBinding) { b.Repository.Revision = strings.Repeat("c", 40) },
		"repository_tree":   func(b *CodeTaskIndependentLabelBinding) { b.Repository.TreeSHA256 = "" },
		"suite_receipt":     func(b *CodeTaskIndependentLabelBinding) { b.ObjectiveSuite.ValidationReceiptSHA256 = "" },
		"expected_behavior": func(b *CodeTaskIndependentLabelBinding) { b.ExpectedBehaviorSHA256 = "" },
	}
	for name, mutate := range labelMutations {
		t.Run("label/"+name, func(t *testing.T) {
			changed := label
			mutate(&changed)
			if err := adapter.VerifyCodeTaskIndependentLabel(context.Background(), candidate.IndependentLabel, changed); err == nil {
				t.Fatal("owner adapter accepted an incomplete or misbound label request")
			}
			if owners.labelCalls != 0 {
				t.Fatal("invalid label binding reached the external owner")
			}
		})
	}
}

type sourceIdentityReceiptFake struct {
	want    AgentManagerCodeTaskSourceIdentityBinding
	receipt AgentManagerCodeTaskSourceIdentityReceipt
	err     error
}

func (f *sourceIdentityReceiptFake) GetCodeTaskSourceIdentityReceipt(_ context.Context, binding AgentManagerCodeTaskSourceIdentityBinding) (AgentManagerCodeTaskSourceIdentityReceipt, error) {
	f.want = binding
	return f.receipt, f.err
}

func TestAgentManagerIdentityReceiptBindsTaskAndRepositoryRevision(t *testing.T) {
	candidate := codeTaskCandidateFixture(920, "repo-grounded-citation")
	binding := CodeTaskSourceIdentityBinding{Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository}
	want := AgentManagerCodeTaskSourceIdentityBinding{Purpose: "local-coding-benchmark-fixture/v1", RunID: candidate.Source.RunID, EventID: candidate.Source.EventID, Sequence: candidate.Source.Sequence, ConversationID: candidate.Source.ConversationID, MessageSHA256: candidate.Source.MessageSHA256, TaskSpecSHA256: candidate.TaskSpecSHA256, RepositoryID: candidate.Repository.RepositoryID, RepositoryRevision: candidate.Repository.Revision, RepositoryTreeSHA256: candidate.Repository.TreeSHA256}
	reader := &sourceIdentityReceiptFake{receipt: AgentManagerCodeTaskSourceIdentityReceipt{Binding: want, ActorSubjectSHA256: candidate.Source.ActorSubjectSHA256, ActorSubjectDigestVersion: candidate.Source.ActorSubjectDigestVersion, ActorIdentityReceiptID: candidate.Source.ActorIdentityReceiptID, ActorIdentityReceiptSHA256: candidate.Source.ActorIdentityReceiptSHA256}}
	verifier := AgentManagerCodeTaskSourceIdentityVerifier{Receipts: reader}
	if err := verifier.VerifyCodeTaskActorIdentity(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
	if reader.want != want {
		t.Fatalf("owner request binding=%+v want=%+v", reader.want, want)
	}
	wrong := reader.receipt
	wrong.Binding.RepositoryRevision = strings.Repeat("f", 40)
	reader.receipt = wrong
	if err := verifier.VerifyCodeTaskActorIdentity(context.Background(), binding); err == nil {
		t.Fatal("accepted receipt from a different repository revision")
	}
	wrong = reader.receipt
	wrong.Binding = want
	wrong.Binding.TaskSpecSHA256 = strings.Repeat("e", 64)
	reader.receipt = wrong
	if err := verifier.VerifyCodeTaskActorIdentity(context.Background(), binding); err == nil {
		t.Fatal("accepted receipt from a different task digest")
	}
}

func TestBuildCodeTaskCorpusSealsOnlyCompleteMetadataDenominator(t *testing.T) {
	var candidates []CodeTaskCandidate
	index := 0
	for stratum, minimum := range codeTaskRequiredStrata {
		for n := 0; n < minimum; n++ {
			index++
			candidates = append(candidates, codeTaskCandidateFixture(index, stratum))
		}
	}
	if !strings.HasPrefix(codeTaskCaseID(candidates[0]), "lm22-") {
		t.Fatalf("Search Hub case identity %q is outside the Workspace Sandbox LM22 owner namespace", codeTaskCaseID(candidates[0]))
	}
	snapshot := BuildCodeTaskCorpus(context.Background(), candidates, "split-v1", testCodeTaskEvidenceAdapter(t, acceptingCodeTaskOwners{}))
	if !snapshot.Sealed || len(snapshot.Cases) < CodeTaskCorpusMinimumCases || len(snapshot.CaseIDs) != len(snapshot.Cases) {
		t.Fatalf("complete owner-verified metadata denominator did not seal: cases=%d deficits=%v", len(snapshot.Cases), snapshot.CoverageDeficits)
	}
	if snapshot.DenominatorSHA256 == "" || snapshot.CorpusSHA256 == "" {
		t.Fatal("sealed denominator is missing corpus or case-id digest")
	}
	projected, err := snapshot.EvaluationCases()
	if err != nil || len(projected) != len(snapshot.Cases) {
		t.Fatalf("sealed corpus did not project into the Gateway handoff: cases=%d err=%v", len(projected), err)
	}
	raw, err := json.Marshal(projected[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"source_run_id", "source_actor_id", "source_message_role", "gateway_request_id", "selected_resource_role", "capacity_claim_id", "repository_revision", "objective_suite_sha256", "objective_suite_receipt_id", "objective_suite_receipt_sha256", "independent_label_receipt_sha256", "independent_reviewer_id", "privacy_decision_id"} {
		if fields[field] == nil {
			t.Fatalf("Gateway case projection omitted %q: %s", field, raw)
		}
	}
	snapshot.CaseIDs[0] = "tampered"
	if _, err := snapshot.EvaluationCases(); err == nil {
		t.Fatal("expected sealed denominator mutation to refuse evaluation projection")
	}
}

func TestCodeTaskCorpusProjectionRefusesMutationOfEveryBoundCaseField(t *testing.T) {
	base := completeCodeTaskCorpusFixture(t)
	if _, err := base.EvaluationCases(); err != nil {
		t.Fatalf("valid frozen fixture should project: %v", err)
	}

	paths := codeTaskCorpusLeafPaths(reflect.ValueOf(base.Cases), nil)
	if len(paths) == 0 {
		t.Fatal("test fixture has no corpus fields to mutate")
	}
	for _, path := range paths {
		path := append([]int(nil), path...)
		t.Run(codeTaskCorpusFieldPath(path), func(t *testing.T) {
			tampered := cloneCodeTaskCorpusSnapshot(t, base)
			field := reflect.ValueOf(&tampered.Cases).Elem()
			for _, index := range path {
				switch field.Kind() {
				case reflect.Slice, reflect.Array:
					field = field.Index(index)
				case reflect.Struct:
					field = field.Field(index)
				default:
					t.Fatalf("invalid mutation path at %s (%s)", field.Type(), field.Kind())
				}
			}
			mutateCodeTaskCorpusLeaf(t, field)
			if _, err := tampered.EvaluationCases(); err == nil {
				t.Fatalf("projection accepted a mutation at %s", codeTaskCorpusFieldPath(path))
			}
		})
	}

	t.Run("split_version", func(t *testing.T) {
		tampered := cloneCodeTaskCorpusSnapshot(t, base)
		tampered.SplitVersion += "-tampered"
		if _, err := tampered.EvaluationCases(); err == nil {
			t.Fatal("projection accepted a changed split identity")
		}
	})
}

func completeCodeTaskCorpusFixture(t *testing.T) CodeTaskCorpusSnapshot {
	var candidates []CodeTaskCandidate
	index := 0
	for stratum, minimum := range codeTaskRequiredStrata {
		for n := 0; n < minimum; n++ {
			index++
			candidates = append(candidates, codeTaskCandidateFixture(index, stratum))
		}
	}
	return BuildCodeTaskCorpus(context.Background(), candidates, "split-v1", testCodeTaskEvidenceAdapter(t, acceptingCodeTaskOwners{}))
}

func TestNewCodeTaskEvidenceOwnerAdapterRequiresAllOwnerClients(t *testing.T) {
	base := testCodeTaskOwnerClients(acceptingCodeTaskOwners{})
	for _, name := range []string{"source", "source_identity", "route", "rights", "privacy", "repository", "objective", "independent_label"} {
		t.Run(name, func(t *testing.T) {
			clients := base
			switch name {
			case "source":
				clients.Source = nil
			case "source_identity":
				clients.SourceIdentity = nil
			case "route":
				clients.Route = nil
			case "rights":
				clients.Rights = nil
			case "privacy":
				clients.Privacy = nil
			case "repository":
				clients.Repository = nil
			case "objective":
				clients.Objective = nil
			case "independent_label":
				clients.IndependentLabel = nil
			}
			if _, err := NewCodeTaskEvidenceOwnerAdapter(clients); err == nil {
				t.Fatalf("adapter accepted missing %s owner", name)
			}
		})
	}
	var typedNil *recordingCodeTaskOwners
	clients := base
	clients.Privacy = typedNil
	if _, err := NewCodeTaskEvidenceOwnerAdapter(clients); err == nil {
		t.Fatal("adapter accepted an interface containing a nil privacy owner")
	}
}

func testCodeTaskOwnerClients(owner testCodeTaskEvidenceOwner) CodeTaskEvidenceOwnerClients {
	return CodeTaskEvidenceOwnerClients{
		Source: owner, SourceIdentity: owner, Route: owner, Rights: owner, Privacy: owner,
		Repository: owner, Objective: owner, IndependentLabel: owner,
	}
}

func testCodeTaskEvidenceAdapter(t *testing.T, owner testCodeTaskEvidenceOwner) *CodeTaskEvidenceOwnerAdapter {
	t.Helper()
	adapter, err := NewCodeTaskEvidenceOwnerAdapter(testCodeTaskOwnerClients(owner))
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func cloneCodeTaskCorpusSnapshot(t *testing.T, input CodeTaskCorpusSnapshot) CodeTaskCorpusSnapshot {
	t.Helper()
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var cloned CodeTaskCorpusSnapshot
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func codeTaskCorpusLeafPaths(value reflect.Value, prefix []int) [][]int {
	var paths [][]int
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			paths = append(paths, codeTaskCorpusLeafPaths(value.Field(i), append(append([]int(nil), prefix...), i))...)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			paths = append(paths, codeTaskCorpusLeafPaths(value.Index(i), append(append([]int(nil), prefix...), i))...)
		}
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		paths = append(paths, append([]int(nil), prefix...))
	}
	return paths
}

func codeTaskCorpusFieldPath(path []int) string {
	parts := make([]string, len(path))
	for i, index := range path {
		parts[i] = strconv.Itoa(index)
	}
	return "cases." + strings.Join(parts, ".")
}

func mutateCodeTaskCorpusLeaf(t *testing.T, value reflect.Value) {
	t.Helper()
	if !value.CanSet() {
		t.Fatalf("corpus field %s cannot be changed in test", value.Type())
	}
	switch value.Kind() {
	case reflect.String:
		value.SetString(value.String() + "-tampered")
	case reflect.Bool:
		value.SetBool(!value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(value.Int() + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(value.Uint() + 1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(value.Float() + 1)
	default:
		t.Fatalf("unsupported corpus leaf type %s", value.Type())
	}
}

func codeTaskCandidateFixture(index int, stratum string) CodeTaskCandidate {
	uid := fmt.Sprintf("%08x-1111-4111-8111-%012x", index, index)
	sha := strings.Repeat("a", 64)
	return CodeTaskCandidate{
		Source:         CodeTaskSourceRef{RunID: uid, EventID: fmt.Sprintf("%08x-2222-4222-8222-%012x", index, index), Sequence: uint64(index), ConversationID: fmt.Sprintf("%08x-3333-4333-8333-%012x", index, index), RunStatus: "complete", MessageRole: "user", MessageSource: "operator", ActorID: "operator", ActorSubjectSHA256: strings.Repeat("1", 64), ActorSubjectDigestVersion: "agent-manager-source-author-hmac/v1", ActorIdentityReceiptID: "source-identity-receipt", ActorIdentityReceiptSHA256: sha, MessageSHA256: sha},
		Route:          GatewayRouteProof{SourceRunID: uid, SourceEventID: fmt.Sprintf("%08x-2222-4222-8222-%012x", index, index), SourceSequence: uint64(index), GatewayEventID: uid, GatewayRequestID: uid, GatewayRoleEvidenceURI: "gateway-ledger://events/" + uid, TaskRole: "coding", Consumer: "opencode", Status: "succeeded", SelectedProvider: "local", SelectedModel: "fixture", SelectedResourceRole: CodeTaskCorpusRole, Locality: "local", CapacityClaimID: uid, CapacityRequestedBytes: 1, CapacityRequiredBytes: 1, CapacityGrantedBytes: 1, CapacityVerdict: "fit", CapacityEstimateEvidenceURI: "capacity-footprint://fixture"},
		Repository:     CodeTaskRepositoryBase{RepositoryID: "fixture", Revision: strings.Repeat("b", 40), TreeSHA256: sha},
		ObjectiveSuite: CodeTaskObjectiveSuite{SuiteID: "fixture-tests", Revision: strings.Repeat("b", 40), SHA256: sha, ValidationReceiptID: "fixture-test-receipt", ValidationReceiptSHA256: sha},
		TaskSpecSHA256: sha, ExpectedBehaviorSHA256: sha, RightsDecisionID: "rights-fixture", PrivacyDecisionID: "privacy-fixture",
		IndependentLabel: CodeTaskIndependentLabelRef{ReviewerID: "reviewer", ReviewerSubjectSHA256: strings.Repeat("2", 64), EvidenceURI: "review://fixture", ReceiptSHA256: sha, ExpectedBehaviorSHA256: sha}, Split: "heldout", Strata: []string{stratum},
	}
}
