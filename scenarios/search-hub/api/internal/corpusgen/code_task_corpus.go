package corpusgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const (
	CodeTaskCorpusSchemaVersion = "local-code-task-corpus/v2"
	CodeTaskCorpusRole          = "code.local"
	CodeTaskCorpusMinimumCases  = 20
)

var codeTaskSourceUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var codeTaskRequiredStrata = map[string]int{
	"image-tools-transcript": 1,
	"repo-grounded-citation": 3,
	"single-file-change":     3,
	"multi-file-change":      3,
	"tool-recovery":          2,
	"review-positive":        2,
	"review-clean-control":   2,
	"remediation-validation": 2,
	"other-caller":           2,
}

// CodeTaskSourceRef identifies an exact, completed operator message. The
// message body is never stored in a code corpus snapshot; owners re-fetch it
// through Agent Manager when execution is separately authorized.
type CodeTaskSourceRef struct {
	RunID                      string `json:"run_id"`
	EventID                    string `json:"event_id"`
	Sequence                   uint64 `json:"sequence"`
	ConversationID             string `json:"conversation_id"`
	RunStatus                  string `json:"run_status"`
	MessageRole                string `json:"message_role"`
	MessageSource              string `json:"message_source"`
	ActorID                    string `json:"actor_id"`             // event author kind; not a stable human identity
	ActorSubjectSHA256         string `json:"actor_subject_sha256"` // opaque owner-keyed human-identity digest, never a raw subject
	ActorSubjectDigestVersion  string `json:"actor_subject_digest_version"`
	ActorIdentityReceiptID     string `json:"actor_identity_receipt_id"`
	ActorIdentityReceiptSHA256 string `json:"actor_identity_receipt_sha256"`
	MessageSHA256              string `json:"message_sha256"`
}

// CodeTaskRepositoryBase binds one task to an immutable repository tree.
type CodeTaskRepositoryBase struct {
	RepositoryID string `json:"repository_id"`
	Revision     string `json:"revision"`
	TreeSHA256   string `json:"tree_sha256"`
}

// CodeTaskObjectiveSuite pins the fixed suite command and the repository
// revision whose tests it runs. ValidationReceiptID/SHA bind the actual owner
// execution over the repository tree; this package never grades model output.
type CodeTaskObjectiveSuite struct {
	SuiteID                 string `json:"suite_id"`
	Revision                string `json:"revision"`
	SHA256                  string `json:"sha256"`
	ValidationReceiptID     string `json:"validation_receipt_id"`
	ValidationReceiptSHA256 string `json:"validation_receipt_sha256"`
}

// CodeTaskIndependentLabelRef points to a separately authored expected
// behavior label. Its body remains at the receipt owner.
type CodeTaskIndependentLabelRef struct {
	ReviewerID             string `json:"reviewer_id"`
	ReviewerSubjectSHA256  string `json:"reviewer_subject_sha256"` // same owner-keyed identity namespace as ActorSubjectSHA256
	EvidenceURI            string `json:"evidence_uri"`
	ReceiptSHA256          string `json:"receipt_sha256"`
	ExpectedBehaviorSHA256 string `json:"expected_behavior_sha256"`
}

const CodeTaskCorpusConsentPurpose = "local-coding-benchmark-fixture/v1"

// CodeTaskCorpusConsentBinding is the exact unit of authorization requested
// from the privacy owner. A decision ID by itself is not authority: a
// production adapter must call the privacy authority and prove its current
// grant binds the exact completed run/event/sequence/conversation/message
// digest, task digest, immutable repository tree/revision, objective suite
// receipt, expected behavior and local fixture-retention purpose. It must
// refuse a grant whose verified human cannot be linked to the source author.
// The binding contains hashes and owner IDs only; never add message or patch
// bodies here.
type CodeTaskCorpusConsentBinding struct {
	Purpose                string                 `json:"purpose"`
	Source                 CodeTaskSourceRef      `json:"source"`
	TaskSpecSHA256         string                 `json:"task_spec_sha256"`
	Repository             CodeTaskRepositoryBase `json:"repository"`
	ObjectiveSuite         CodeTaskObjectiveSuite `json:"objective_suite"`
	ExpectedBehaviorSHA256 string                 `json:"expected_behavior_sha256"`
}

// CodeTaskIndependentLabelBinding binds a separate label receipt to all
// evidence whose correctness it attests. A production adapter must call the
// label receipt owner and prove the reviewer is an independently identified
// human, not the source author, and refuse a label for a different source,
// task, repository tree/revision or objective suite. It contains hashes and
// owner IDs only; never add task or patch bodies here.
type CodeTaskIndependentLabelBinding struct {
	Source                 CodeTaskSourceRef      `json:"source"`
	TaskSpecSHA256         string                 `json:"task_spec_sha256"`
	Repository             CodeTaskRepositoryBase `json:"repository"`
	ObjectiveSuite         CodeTaskObjectiveSuite `json:"objective_suite"`
	ExpectedBehaviorSHA256 string                 `json:"expected_behavior_sha256"`
}

// CodeTaskCandidate contains references and hashes only. It intentionally has
// no prompt, response, patch body, or test log field.
type CodeTaskCandidate struct {
	Source                 CodeTaskSourceRef           `json:"source"`
	Route                  GatewayRouteProof           `json:"route"`
	Repository             CodeTaskRepositoryBase      `json:"repository"`
	ObjectiveSuite         CodeTaskObjectiveSuite      `json:"objective_suite"`
	TaskSpecSHA256         string                      `json:"task_spec_sha256"`
	ExpectedBehaviorSHA256 string                      `json:"expected_behavior_sha256"`
	RightsDecisionID       string                      `json:"rights_decision_id"`
	PrivacyDecisionID      string                      `json:"privacy_decision_id"`
	IndependentLabel       CodeTaskIndependentLabelRef `json:"independent_label"`
	Split                  string                      `json:"split"`
	Strata                 []string                    `json:"strata"`
}

// CodeTaskCase is the accepted, immutable metadata projection used by the
// paired executor. Raw task and repository content remain with their owners.
type CodeTaskCase struct {
	CaseID                 string                      `json:"case_id"`
	Source                 CodeTaskSourceRef           `json:"source"`
	Route                  GatewayRouteProof           `json:"route"`
	Repository             CodeTaskRepositoryBase      `json:"repository"`
	ObjectiveSuite         CodeTaskObjectiveSuite      `json:"objective_suite"`
	TaskSpecSHA256         string                      `json:"task_spec_sha256"`
	ExpectedBehaviorSHA256 string                      `json:"expected_behavior_sha256"`
	RightsDecisionID       string                      `json:"rights_decision_id"`
	PrivacyDecisionID      string                      `json:"privacy_decision_id"`
	IndependentLabel       CodeTaskIndependentLabelRef `json:"independent_label"`
	Split                  string                      `json:"split"`
	Strata                 []string                    `json:"strata"`
}

type CodeTaskRejection struct {
	CandidateID string `json:"candidate_id,omitempty"`
	ReasonCode  string `json:"reason_code"`
}

type CodeTaskCorpusSnapshot struct {
	SchemaVersion     string              `json:"schema_version"`
	Role              string              `json:"role"`
	SplitVersion      string              `json:"split_version"`
	Cases             []CodeTaskCase      `json:"cases"`
	CaseIDs           []string            `json:"case_ids"`
	DenominatorSHA256 string              `json:"denominator_sha256,omitempty"`
	CorpusSHA256      string              `json:"corpus_sha256,omitempty"`
	Sealed            bool                `json:"sealed"`
	CoverageDeficits  []string            `json:"coverage_deficits,omitempty"`
	Rejections        []CodeTaskRejection `json:"rejections"`
}

// CodeTaskEvaluationCase is the JSON handoff shape consumed by AI Gateway's
// metadata-only pair scorer. Keep field names aligned with CodeEvalCase while
// avoiding a cross-scenario Go package dependency.
type CodeTaskEvaluationCase struct {
	CaseID                        string   `json:"case_id"`
	Strata                        []string `json:"strata"`
	SourceRunID                   string   `json:"source_run_id"`
	SourceEventID                 string   `json:"source_event_id"`
	ConversationID                string   `json:"conversation_id"`
	SourceActorID                 string   `json:"source_actor_id"`
	SourceActorDigestVersion      string   `json:"source_actor_digest_version"`
	SourceRunStatus               string   `json:"source_run_status"`
	SourceMessageRole             string   `json:"source_message_role"`
	SourceMessageSource           string   `json:"source_message_source"`
	GatewayEventID                string   `json:"gateway_event_id"`
	GatewayRequestID              string   `json:"gateway_request_id"`
	GatewayRoleEvidenceURI        string   `json:"gateway_role_evidence_uri"`
	SelectedResourceRole          string   `json:"selected_resource_role"`
	TaskRole                      string   `json:"task_role"`
	Consumer                      string   `json:"consumer"`
	RouteLocality                 string   `json:"route_locality"`
	CapacityClaimID               string   `json:"capacity_claim_id"`
	CapacityRequiredBytes         int64    `json:"capacity_required_bytes"`
	CapacityGrantedBytes          int64    `json:"capacity_granted_bytes"`
	CapacityVerdict               string   `json:"capacity_verdict"`
	CapacityEstimateEvidenceURI   string   `json:"capacity_estimate_evidence_uri"`
	SourceMessageSHA256           string   `json:"source_message_sha256"`
	TaskSpecSHA256                string   `json:"task_spec_sha256"`
	ExpectedBehaviorSHA256        string   `json:"expected_behavior_sha256"`
	RepositoryID                  string   `json:"repository_id"`
	RepositoryRevision            string   `json:"repository_revision"`
	RepositoryTreeSHA256          string   `json:"repository_tree_sha256"`
	ObjectiveSuiteID              string   `json:"objective_suite_id"`
	ObjectiveSuiteRevision        string   `json:"objective_suite_revision"`
	ObjectiveSuiteSHA256          string   `json:"objective_suite_sha256"`
	ObjectiveSuiteReceiptID       string   `json:"objective_suite_receipt_id"`
	ObjectiveSuiteReceiptSHA256   string   `json:"objective_suite_receipt_sha256"`
	IndependentLabelReceiptSHA256 string   `json:"independent_label_receipt_sha256"`
	IndependentReviewerID         string   `json:"independent_reviewer_id"`
	RightsDecisionID              string   `json:"rights_decision_id"`
	PrivacyDecisionID             string   `json:"privacy_decision_id"`
}

// EvaluationCases exposes only a frozen denominator and verifies that the
// case IDs still match its sealed digest before producing the Gateway handoff.
func (snapshot CodeTaskCorpusSnapshot) EvaluationCases() ([]CodeTaskEvaluationCase, error) {
	if !snapshot.Sealed || snapshot.Role != CodeTaskCorpusRole || snapshot.SchemaVersion != CodeTaskCorpusSchemaVersion || snapshot.CorpusSHA256 == "" || snapshot.SplitVersion == "" {
		return nil, fmt.Errorf("code task corpus is not sealed for evaluation")
	}
	if snapshot.CorpusSHA256 != codeTaskCorpusDigest(snapshot) {
		return nil, fmt.Errorf("sealed code task corpus content or split identity changed")
	}
	ids := make([]string, 0, len(snapshot.Cases))
	for _, item := range snapshot.Cases {
		ids = append(ids, item.CaseID)
	}
	if len(ids) != len(snapshot.CaseIDs) || searcheval.CaseIDsDigest(ids) != snapshot.DenominatorSHA256 || searcheval.CaseIDsDigest(snapshot.CaseIDs) != snapshot.DenominatorSHA256 {
		return nil, fmt.Errorf("sealed code task denominator does not match its case IDs")
	}
	projected := make([]CodeTaskEvaluationCase, 0, len(snapshot.Cases))
	for _, item := range snapshot.Cases {
		projected = append(projected, CodeTaskEvaluationCase{
			CaseID: item.CaseID, Strata: append([]string(nil), item.Strata...),
			SourceRunID: item.Source.RunID, SourceEventID: item.Source.EventID,
			ConversationID: item.Source.ConversationID, SourceActorID: item.Source.ActorSubjectSHA256, SourceActorDigestVersion: item.Source.ActorSubjectDigestVersion,
			SourceRunStatus: item.Source.RunStatus, SourceMessageRole: item.Source.MessageRole, SourceMessageSource: item.Source.MessageSource,
			GatewayEventID: item.Route.GatewayEventID, GatewayRequestID: item.Route.GatewayRequestID,
			GatewayRoleEvidenceURI: item.Route.GatewayRoleEvidenceURI, SelectedResourceRole: item.Route.SelectedResourceRole,
			TaskRole: item.Route.TaskRole, Consumer: item.Route.Consumer, RouteLocality: item.Route.Locality,
			CapacityClaimID: item.Route.CapacityClaimID, CapacityRequiredBytes: item.Route.CapacityRequiredBytes,
			CapacityGrantedBytes: item.Route.CapacityGrantedBytes, CapacityVerdict: item.Route.CapacityVerdict,
			CapacityEstimateEvidenceURI: item.Route.CapacityEstimateEvidenceURI,
			SourceMessageSHA256:         item.Source.MessageSHA256, TaskSpecSHA256: item.TaskSpecSHA256,
			ExpectedBehaviorSHA256: item.ExpectedBehaviorSHA256, RepositoryID: item.Repository.RepositoryID,
			RepositoryRevision: item.Repository.Revision, RepositoryTreeSHA256: item.Repository.TreeSHA256,
			ObjectiveSuiteID: item.ObjectiveSuite.SuiteID, ObjectiveSuiteRevision: item.ObjectiveSuite.Revision,
			ObjectiveSuiteSHA256: item.ObjectiveSuite.SHA256, IndependentLabelReceiptSHA256: item.IndependentLabel.ReceiptSHA256,
			ObjectiveSuiteReceiptID: item.ObjectiveSuite.ValidationReceiptID, ObjectiveSuiteReceiptSHA256: item.ObjectiveSuite.ValidationReceiptSHA256,
			IndependentReviewerID: item.IndependentLabel.ReviewerSubjectSHA256,
			RightsDecisionID:      item.RightsDecisionID, PrivacyDecisionID: item.PrivacyDecisionID,
		})
	}
	return projected, nil
}

// The owner client interfaces below are the production dependency contracts
// for the Search Hub evidence adapter. Implementations perform exact, bounded
// reads from the named owner and return an error on missing or mismatched
// evidence. They never return task or patch bodies to the corpus builder.
type CodeTaskSourceEvidenceOwner interface {
	VerifyCodeTaskSource(context.Context, CodeTaskSourceIdentityBinding) error
}

type CodeTaskSourceIdentityEvidenceOwner interface {
	// VerifyCodeTaskActorIdentity must resolve the stable human identity through
	// the source owner and bind its receipt to the exact task and immutable repo.
	// The message author kind "operator" is not human identity evidence.
	VerifyCodeTaskActorIdentity(context.Context, CodeTaskSourceIdentityBinding) error
}

// CodeTaskSourceIdentityBinding is passed to the identity owner so its receipt
// binds the exact event and task to the immutable repository base.
type CodeTaskSourceIdentityBinding struct {
	Source         CodeTaskSourceRef      `json:"source"`
	TaskSpecSHA256 string                 `json:"task_spec_sha256"`
	Repository     CodeTaskRepositoryBase `json:"repository"`
}

// AgentManagerCodeTaskSourceIdentityBinding matches Agent Manager's owner API
// request. It contains no raw sender subject or message body.
type AgentManagerCodeTaskSourceIdentityBinding struct {
	Purpose              string `json:"purpose"`
	RunID                string `json:"run_id"`
	EventID              string `json:"event_id"`
	Sequence             uint64 `json:"sequence"`
	ConversationID       string `json:"conversation_id"`
	MessageSHA256        string `json:"message_sha256"`
	TaskSpecSHA256       string `json:"task_spec_sha256"`
	RepositoryID         string `json:"repository_id"`
	RepositoryRevision   string `json:"repository_revision"`
	RepositoryTreeSHA256 string `json:"repository_tree_sha256"`
}

type AgentManagerCodeTaskSourceIdentityReceipt struct {
	Binding                    AgentManagerCodeTaskSourceIdentityBinding `json:"binding"`
	ActorSubjectSHA256         string                                    `json:"actor_subject_sha256"`
	ActorSubjectDigestVersion  string                                    `json:"actor_subject_digest_version"`
	ActorIdentityReceiptID     string                                    `json:"actor_identity_receipt_id"`
	ActorIdentityReceiptSHA256 string                                    `json:"actor_identity_receipt_sha256"`
}

type AgentManagerCodeTaskSourceIdentityReceiptReader interface {
	GetCodeTaskSourceIdentityReceipt(context.Context, AgentManagerCodeTaskSourceIdentityBinding) (AgentManagerCodeTaskSourceIdentityReceipt, error)
}

// AgentManagerCodeTaskSourceIdentityVerifier checks the owner receipt against
// the exact body-free source/task/repository binding and candidate pseudonym.
type AgentManagerCodeTaskSourceIdentityVerifier struct {
	Receipts AgentManagerCodeTaskSourceIdentityReceiptReader
}

func (v AgentManagerCodeTaskSourceIdentityVerifier) VerifyCodeTaskActorIdentity(ctx context.Context, binding CodeTaskSourceIdentityBinding) error {
	if v.Receipts == nil || validateCodeTaskSourceIdentityBinding(binding) != nil {
		return fmt.Errorf("Agent Manager identity receipt reader and complete binding are required")
	}
	want := AgentManagerCodeTaskSourceIdentityBinding{
		Purpose: "local-coding-benchmark-fixture/v1", RunID: binding.Source.RunID,
		EventID: binding.Source.EventID, Sequence: binding.Source.Sequence,
		ConversationID: binding.Source.ConversationID, MessageSHA256: binding.Source.MessageSHA256,
		TaskSpecSHA256: binding.TaskSpecSHA256, RepositoryID: binding.Repository.RepositoryID,
		RepositoryRevision: binding.Repository.Revision, RepositoryTreeSHA256: binding.Repository.TreeSHA256,
	}
	receipt, err := v.Receipts.GetCodeTaskSourceIdentityReceipt(ctx, want)
	if err != nil {
		return err
	}
	if receipt.Binding != want || receipt.ActorSubjectSHA256 != binding.Source.ActorSubjectSHA256 || receipt.ActorSubjectDigestVersion != binding.Source.ActorSubjectDigestVersion ||
		receipt.ActorIdentityReceiptID != binding.Source.ActorIdentityReceiptID ||
		receipt.ActorIdentityReceiptSHA256 != binding.Source.ActorIdentityReceiptSHA256 ||
		!validCodeTaskSHA256(receipt.ActorIdentityReceiptSHA256) {
		return fmt.Errorf("Agent Manager identity receipt does not bind the exact source task and repository revision")
	}
	return nil
}

type CodeTaskRouteEvidenceOwner interface {
	VerifyCodeTaskRoute(context.Context, GatewayRouteProof) error
}

type CodeTaskRightsEvidenceOwner interface {
	VerifyCodeTaskRights(context.Context, string, string) error
}

type CodeTaskPrivacyEvidenceOwner interface {
	// VerifyCodeTaskPrivacy must validate the source-scoped fixture-use grant
	// at its owner. It must not translate a remote-provider grant, a historical
	// decision name, or an unbound boolean into local corpus consent.
	VerifyCodeTaskPrivacy(context.Context, string, CodeTaskCorpusConsentBinding) error
}

type CodeTaskRepositoryEvidenceOwner interface {
	VerifyCodeTaskRepository(context.Context, CodeTaskRepositoryBase) error
}

type CodeTaskObjectiveEvidenceOwner interface {
	VerifyCodeTaskObjectiveSuite(context.Context, CodeTaskRepositoryBase, CodeTaskObjectiveSuite) error
}

type CodeTaskIndependentLabelEvidenceOwner interface {
	// VerifyCodeTaskIndependentLabel must resolve the cited receipt at the
	// independent label owner and compare its complete binding, including the
	// stable author/reviewer identities. A matching digest supplied only by the
	// candidate is not owner evidence.
	VerifyCodeTaskIndependentLabel(context.Context, CodeTaskIndependentLabelRef, CodeTaskIndependentLabelBinding) error
}

// CodeTaskEvidenceOwnerClients collects the exact owner dependencies needed
// to build a real-caller coding corpus. The constructor refuses partial wiring;
// in particular, privacy and independent-label owners are mandatory.
type CodeTaskEvidenceOwnerClients struct {
	Source           CodeTaskSourceEvidenceOwner
	SourceIdentity   CodeTaskSourceIdentityEvidenceOwner
	Route            CodeTaskRouteEvidenceOwner
	Rights           CodeTaskRightsEvidenceOwner
	Privacy          CodeTaskPrivacyEvidenceOwner
	Repository       CodeTaskRepositoryEvidenceOwner
	Objective        CodeTaskObjectiveEvidenceOwner
	IndependentLabel CodeTaskIndependentLabelEvidenceOwner
}

// CodeTaskEvidenceOwnerAdapter is the only accepted owner bundle for the
// primary BuildCodeTaskCorpus path. Its constructor prevents accidental use
// of a partly configured live adapter.
type CodeTaskEvidenceOwnerAdapter struct {
	clients CodeTaskEvidenceOwnerClients
}

func NewCodeTaskEvidenceOwnerAdapter(clients CodeTaskEvidenceOwnerClients) (*CodeTaskEvidenceOwnerAdapter, error) {
	if nilCodeTaskOwner(clients.Source) || nilCodeTaskOwner(clients.SourceIdentity) || nilCodeTaskOwner(clients.Route) || nilCodeTaskOwner(clients.Rights) || nilCodeTaskOwner(clients.Privacy) ||
		nilCodeTaskOwner(clients.Repository) || nilCodeTaskOwner(clients.Objective) || nilCodeTaskOwner(clients.IndependentLabel) {
		return nil, fmt.Errorf("all eight code-task evidence owners are required")
	}
	return &CodeTaskEvidenceOwnerAdapter{clients: clients}, nil
}

func nilCodeTaskOwner(owner any) bool {
	if owner == nil {
		return true
	}
	value := reflect.ValueOf(owner)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskSource(ctx context.Context, binding CodeTaskSourceIdentityBinding) error {
	if err := validateCodeTaskSourceIdentityBinding(binding); err != nil {
		return err
	}
	return a.clients.Source.VerifyCodeTaskSource(ctx, binding)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskActorIdentity(ctx context.Context, binding CodeTaskSourceIdentityBinding) error {
	if err := validateCodeTaskSourceIdentityBinding(binding); err != nil {
		return err
	}
	return a.clients.SourceIdentity.VerifyCodeTaskActorIdentity(ctx, binding)
}

func validateCodeTaskSourceIdentityBinding(binding CodeTaskSourceIdentityBinding) error {
	if err := validateCodeTaskSourceIdentity(binding.Source); err != nil {
		return err
	}
	if !validCodeTaskSHA256(binding.TaskSpecSHA256) || binding.TaskSpecSHA256 != binding.Source.MessageSHA256 {
		return fmt.Errorf("identity receipt must bind the exact source task digest")
	}
	return validateCodeTaskRepositoryBinding(binding.Repository)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskRoute(ctx context.Context, route GatewayRouteProof) error {
	return a.clients.Route.VerifyCodeTaskRoute(ctx, route)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskRights(ctx context.Context, decisionID, model string) error {
	return a.clients.Rights.VerifyCodeTaskRights(ctx, decisionID, model)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskPrivacy(ctx context.Context, decisionID string, binding CodeTaskCorpusConsentBinding) error {
	if strings.TrimSpace(decisionID) == "" {
		return fmt.Errorf("code-task privacy decision ID is required")
	}
	if err := validateCodeTaskCorpusConsentBinding(binding); err != nil {
		return fmt.Errorf("invalid code-task privacy binding: %w", err)
	}
	return a.clients.Privacy.VerifyCodeTaskPrivacy(ctx, decisionID, binding)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskRepository(ctx context.Context, repository CodeTaskRepositoryBase) error {
	return a.clients.Repository.VerifyCodeTaskRepository(ctx, repository)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskObjectiveSuite(ctx context.Context, repository CodeTaskRepositoryBase, suite CodeTaskObjectiveSuite) error {
	return a.clients.Objective.VerifyCodeTaskObjectiveSuite(ctx, repository, suite)
}

func (a *CodeTaskEvidenceOwnerAdapter) VerifyCodeTaskIndependentLabel(ctx context.Context, ref CodeTaskIndependentLabelRef, binding CodeTaskIndependentLabelBinding) error {
	if err := validateCodeTaskIndependentLabelBinding(ref, binding); err != nil {
		return fmt.Errorf("invalid independent-label binding: %w", err)
	}
	return a.clients.IndependentLabel.VerifyCodeTaskIndependentLabel(ctx, ref, binding)
}

func validateCodeTaskCorpusConsentBinding(binding CodeTaskCorpusConsentBinding) error {
	if binding.Purpose != CodeTaskCorpusConsentPurpose {
		return fmt.Errorf("fixture-retention purpose must be %q", CodeTaskCorpusConsentPurpose)
	}
	if err := validateCodeTaskSourceBinding(binding.Source); err != nil {
		return err
	}
	if !validCodeTaskSHA256(binding.TaskSpecSHA256) || !validCodeTaskSHA256(binding.ExpectedBehaviorSHA256) {
		return fmt.Errorf("task and expected-behavior digests are required")
	}
	if err := validateCodeTaskRepositoryBinding(binding.Repository); err != nil {
		return err
	}
	if err := validateCodeTaskObjectiveBinding(binding.Repository, binding.ObjectiveSuite); err != nil {
		return err
	}
	return nil
}

func validateCodeTaskIndependentLabelBinding(ref CodeTaskIndependentLabelRef, binding CodeTaskIndependentLabelBinding) error {
	if err := validateCodeTaskSourceBinding(binding.Source); err != nil {
		return err
	}
	if !validCodeTaskSHA256(binding.TaskSpecSHA256) || !validCodeTaskSHA256(binding.ExpectedBehaviorSHA256) ||
		ref.ReviewerID == "" || !validCodeTaskSHA256(ref.ReviewerSubjectSHA256) || ref.ReviewerSubjectSHA256 == binding.Source.ActorSubjectSHA256 || ref.EvidenceURI == "" ||
		!validCodeTaskSHA256(ref.ReceiptSHA256) || ref.ExpectedBehaviorSHA256 != binding.ExpectedBehaviorSHA256 {
		return fmt.Errorf("reviewer, receipt and task digests must identify a distinct-source-author label")
	}
	if err := validateCodeTaskRepositoryBinding(binding.Repository); err != nil {
		return err
	}
	if err := validateCodeTaskObjectiveBinding(binding.Repository, binding.ObjectiveSuite); err != nil {
		return err
	}
	return nil
}

func validateCodeTaskSourceBinding(source CodeTaskSourceRef) error {
	if !codeTaskSourceUUID.MatchString(source.RunID) || !codeTaskSourceUUID.MatchString(source.EventID) ||
		!codeTaskSourceUUID.MatchString(source.ConversationID) || source.RunStatus != "complete" ||
		source.MessageRole != "user" || source.MessageSource != "operator" || source.ActorID == "" || !validCodeTaskSHA256(source.MessageSHA256) ||
		validateCodeTaskSourceIdentity(source) != nil {
		return fmt.Errorf("exact completed source run/event/sequence/conversation/actor/message digest is required")
	}
	return nil
}

func validateCodeTaskSourceIdentity(source CodeTaskSourceRef) error {
	if !validCodeTaskSHA256(source.ActorSubjectSHA256) || source.ActorSubjectDigestVersion != "agent-manager-source-author-hmac/v1" || strings.TrimSpace(source.ActorIdentityReceiptID) == "" || !validCodeTaskSHA256(source.ActorIdentityReceiptSHA256) {
		return fmt.Errorf("stable source-author subject and owner identity receipt are required")
	}
	return nil
}

func validateCodeTaskRepositoryBinding(repository CodeTaskRepositoryBase) error {
	if strings.TrimSpace(repository.RepositoryID) == "" || !validCodeTaskGitRevision(repository.Revision) || !validCodeTaskSHA256(repository.TreeSHA256) {
		return fmt.Errorf("repository ID, immutable revision and tree digest are required")
	}
	return nil
}

func validateCodeTaskObjectiveBinding(repository CodeTaskRepositoryBase, suite CodeTaskObjectiveSuite) error {
	if strings.TrimSpace(suite.SuiteID) == "" || !validCodeTaskGitRevision(suite.Revision) ||
		!strings.EqualFold(suite.Revision, repository.Revision) || !validCodeTaskSHA256(suite.SHA256) ||
		strings.TrimSpace(suite.ValidationReceiptID) == "" || !validCodeTaskSHA256(suite.ValidationReceiptSHA256) {
		return fmt.Errorf("objective suite and durable receipt must bind the exact repository revision")
	}
	return nil
}

// BuildCodeTaskCorpus rejects candidates whose exact caller, Gateway route,
// immutable repository, objective suite, rights/privacy context, or independent
// label cannot be verified by its owning scenario. Refusal reasons are kept in
// the snapshot. A denominator is sealed only when every required stratum and
// the 20-case floor are satisfied.
func BuildCodeTaskCorpus(ctx context.Context, candidates []CodeTaskCandidate, splitVersion string, owners *CodeTaskEvidenceOwnerAdapter) CodeTaskCorpusSnapshot {
	snapshot := CodeTaskCorpusSnapshot{SchemaVersion: CodeTaskCorpusSchemaVersion, Role: CodeTaskCorpusRole, SplitVersion: strings.TrimSpace(splitVersion)}
	seenCases, seenSources := map[string]bool{}, map[string]bool{}
	for _, candidate := range candidates {
		caseID, reason := validateCodeTaskCandidateShape(candidate)
		if reason != "" {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: reason})
			continue
		}
		if seenCases[caseID] || seenSources[candidate.Source.RunID+"/"+candidate.Source.EventID] {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "duplicate_source_or_case"})
			continue
		}
		if owners == nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "owner_verifier_unavailable"})
			continue
		}
		identityBinding := CodeTaskSourceIdentityBinding{Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository}
		if owners.VerifyCodeTaskSource(ctx, identityBinding) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "source_owner_refused"})
			continue
		}
		if owners.VerifyCodeTaskActorIdentity(ctx, identityBinding) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "source_actor_identity_owner_refused"})
			continue
		}
		if owners.VerifyCodeTaskRoute(ctx, candidate.Route) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "gateway_route_owner_refused"})
			continue
		}
		if owners.VerifyCodeTaskRights(ctx, candidate.RightsDecisionID, candidate.Route.SelectedModel) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "rights_owner_refused"})
			continue
		}
		consentBinding := CodeTaskCorpusConsentBinding{
			Purpose: CodeTaskCorpusConsentPurpose, Source: candidate.Source,
			TaskSpecSHA256: candidate.TaskSpecSHA256, Repository: candidate.Repository,
			ObjectiveSuite: candidate.ObjectiveSuite, ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
		}
		if owners.VerifyCodeTaskPrivacy(ctx, candidate.PrivacyDecisionID, consentBinding) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "privacy_owner_refused"})
			continue
		}
		if owners.VerifyCodeTaskRepository(ctx, candidate.Repository) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "repository_base_owner_refused"})
			continue
		}
		if owners.VerifyCodeTaskObjectiveSuite(ctx, candidate.Repository, candidate.ObjectiveSuite) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "objective_suite_owner_refused"})
			continue
		}
		labelBinding := CodeTaskIndependentLabelBinding{
			Source: candidate.Source, TaskSpecSHA256: candidate.TaskSpecSHA256,
			Repository: candidate.Repository, ObjectiveSuite: candidate.ObjectiveSuite,
			ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
		}
		if owners.VerifyCodeTaskIndependentLabel(ctx, candidate.IndependentLabel, labelBinding) != nil {
			snapshot.Rejections = append(snapshot.Rejections, CodeTaskRejection{CandidateID: caseID, ReasonCode: "independent_label_owner_refused"})
			continue
		}
		caseRecord := CodeTaskCase{
			CaseID: caseID, Source: candidate.Source, Route: candidate.Route,
			Repository: candidate.Repository, ObjectiveSuite: candidate.ObjectiveSuite,
			TaskSpecSHA256: candidate.TaskSpecSHA256, ExpectedBehaviorSHA256: candidate.ExpectedBehaviorSHA256,
			RightsDecisionID: candidate.RightsDecisionID, PrivacyDecisionID: candidate.PrivacyDecisionID,
			IndependentLabel: candidate.IndependentLabel, Split: candidate.Split,
			Strata: append([]string(nil), candidate.Strata...),
		}
		seenCases[caseID] = true
		seenSources[candidate.Source.RunID+"/"+candidate.Source.EventID] = true
		snapshot.Cases = append(snapshot.Cases, caseRecord)
	}
	sort.Slice(snapshot.Cases, func(i, j int) bool { return snapshot.Cases[i].CaseID < snapshot.Cases[j].CaseID })
	for _, c := range snapshot.Cases {
		snapshot.CaseIDs = append(snapshot.CaseIDs, c.CaseID)
	}
	if len(snapshot.CaseIDs) > 0 {
		snapshot.DenominatorSHA256 = searcheval.CaseIDsDigest(snapshot.CaseIDs)
		snapshot.CorpusSHA256 = codeTaskCorpusDigest(snapshot)
	}
	snapshot.CoverageDeficits = codeTaskCorpusDeficits(snapshot.Cases)
	snapshot.Sealed = snapshot.SplitVersion != "" && snapshot.CorpusSHA256 != "" && len(snapshot.CoverageDeficits) == 0
	return snapshot
}

// codeTaskCorpusDigest binds the full immutable case projection and the
// schema/role/split identity. The denominator digest alone is intentionally
// insufficient: it would not detect edits to source, route, repository,
// objective-suite, rights, privacy, or independent-label lineage fields.
func codeTaskCorpusDigest(snapshot CodeTaskCorpusSnapshot) string {
	identity := struct {
		SchemaVersion string         `json:"schema_version"`
		Role          string         `json:"role"`
		SplitVersion  string         `json:"split_version"`
		Cases         []CodeTaskCase `json:"cases"`
	}{snapshot.SchemaVersion, snapshot.Role, snapshot.SplitVersion, snapshot.Cases}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func validateCodeTaskCandidateShape(candidate CodeTaskCandidate) (caseID, reason string) {
	caseID = codeTaskCaseID(candidate)
	s := candidate.Source
	if !codeTaskSourceUUID.MatchString(s.RunID) || !codeTaskSourceUUID.MatchString(s.EventID) || !codeTaskSourceUUID.MatchString(s.ConversationID) || s.RunStatus != "complete" || s.MessageRole != "user" || s.MessageSource != "operator" || s.ActorID == "" || !validCodeTaskSHA256(s.ActorSubjectSHA256) || strings.TrimSpace(s.ActorIdentityReceiptID) == "" || !validCodeTaskSHA256(s.ActorIdentityReceiptSHA256) || !validCodeTaskSHA256(s.MessageSHA256) {
		return caseID, "source_identity_incomplete"
	}
	if candidate.Route.SourceRunID != s.RunID || candidate.Route.SourceEventID != s.EventID || candidate.Route.SourceSequence != s.Sequence ||
		candidate.Route.GatewayEventID == "" || candidate.Route.GatewayRequestID == "" || candidate.Route.GatewayRoleEvidenceURI != "gateway-ledger://events/"+candidate.Route.GatewayEventID ||
		candidate.Route.Status != "succeeded" || candidate.Route.SelectedProvider == "" || candidate.Route.SelectedModel == "" ||
		candidate.Route.SelectedResourceRole != CodeTaskCorpusRole || candidate.Route.Locality != "local" || candidate.Route.CapacityVerdict != "fit" ||
		candidate.Route.CapacityClaimID == "" || candidate.Route.CapacityRequestedBytes <= 0 || candidate.Route.CapacityRequiredBytes != candidate.Route.CapacityRequestedBytes || candidate.Route.CapacityGrantedBytes < candidate.Route.CapacityRequiredBytes || !strings.HasPrefix(candidate.Route.CapacityEstimateEvidenceURI, "capacity-footprint://") ||
		candidate.Route.TaskRole == "" || candidate.Route.Consumer == "" || strings.TrimSpace(candidate.RightsDecisionID) == "" || strings.TrimSpace(candidate.PrivacyDecisionID) == "" {
		return caseID, "gateway_rights_privacy_or_capacity_identity_incomplete"
	}
	if strings.TrimSpace(candidate.Repository.RepositoryID) == "" || !validCodeTaskGitRevision(candidate.Repository.Revision) || !validCodeTaskSHA256(candidate.Repository.TreeSHA256) {
		return caseID, "repository_base_identity_incomplete"
	}
	if strings.TrimSpace(candidate.ObjectiveSuite.SuiteID) == "" || !validCodeTaskGitRevision(candidate.ObjectiveSuite.Revision) || !strings.EqualFold(candidate.ObjectiveSuite.Revision, candidate.Repository.Revision) || !validCodeTaskSHA256(candidate.ObjectiveSuite.SHA256) || strings.TrimSpace(candidate.ObjectiveSuite.ValidationReceiptID) == "" || !validCodeTaskSHA256(candidate.ObjectiveSuite.ValidationReceiptSHA256) {
		return caseID, "objective_test_suite_identity_incomplete"
	}
	if !validCodeTaskSHA256(candidate.TaskSpecSHA256) || candidate.TaskSpecSHA256 != s.MessageSHA256 || !validCodeTaskSHA256(candidate.ExpectedBehaviorSHA256) ||
		candidate.IndependentLabel.ReviewerID == "" || !validCodeTaskSHA256(candidate.IndependentLabel.ReviewerSubjectSHA256) || candidate.IndependentLabel.ReviewerSubjectSHA256 == s.ActorSubjectSHA256 || candidate.IndependentLabel.EvidenceURI == "" ||
		!validCodeTaskSHA256(candidate.IndependentLabel.ReceiptSHA256) || candidate.IndependentLabel.ExpectedBehaviorSHA256 != candidate.ExpectedBehaviorSHA256 {
		return caseID, "independent_label_or_task_spec_incomplete"
	}
	if candidate.Split != "heldout" || len(candidate.Strata) == 0 {
		return caseID, "sealed_split_or_strata_incomplete"
	}
	seen := map[string]bool{}
	for _, stratum := range candidate.Strata {
		if _, known := codeTaskRequiredStrata[stratum]; !known || seen[stratum] {
			return caseID, "stratum_invalid_or_duplicate"
		}
		seen[stratum] = true
	}
	return caseID, ""
}

func codeTaskCaseID(candidate CodeTaskCandidate) string {
	material := struct {
		RunID, EventID string
		Sequence       uint64
		Role           string
		RepositoryID   string
		Revision       string
		TaskSpecSHA256 string
		SuiteID        string
		SuiteRevision  string
	}{candidate.Source.RunID, candidate.Source.EventID, candidate.Source.Sequence, CodeTaskCorpusRole,
		candidate.Repository.RepositoryID, strings.ToLower(candidate.Repository.Revision), candidate.TaskSpecSHA256,
		candidate.ObjectiveSuite.SuiteID, strings.ToLower(candidate.ObjectiveSuite.Revision)}
	encoded, err := json.Marshal(material)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	// Workspace Sandbox scopes this isolated objective-test operation to
	// workstream-issued lm22-* case IDs. Keep the corpus owner ID in that
	// namespace so the exact case identity can cross the Test Genie adapter.
	return "lm22-" + hex.EncodeToString(digest[:12])
}

func codeTaskCorpusDeficits(cases []CodeTaskCase) []string {
	counts := make(map[string]int, len(codeTaskRequiredStrata))
	for _, c := range cases {
		for _, stratum := range c.Strata {
			counts[stratum]++
		}
	}
	var deficits []string
	if len(cases) < CodeTaskCorpusMinimumCases {
		deficits = append(deficits, fmt.Sprintf("total=%d<%d", len(cases), CodeTaskCorpusMinimumCases))
	}
	for stratum, minimum := range codeTaskRequiredStrata {
		if counts[stratum] < minimum {
			deficits = append(deficits, fmt.Sprintf("%s=%d<%d", stratum, counts[stratum], minimum))
		}
	}
	sort.Strings(deficits)
	return deficits
}

func validCodeTaskGitRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validCodeTaskSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
