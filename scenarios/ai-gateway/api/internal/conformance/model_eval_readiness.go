package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const (
	modelEvalMinimumRetrievalSuites = 35
	modelEvalMinimumCasesPerRole    = 20
	modelEvalCorpusValidationMaxAge = 30 * 24 * time.Hour
)

// ModelEvalReadinessSource reads Search Hub's immutable, independently
// reviewed suite snapshots. It has no execution method: readiness cannot run
// inference or mutate suite data.
type ModelEvalReadinessSource interface {
	Readiness(context.Context, ModelEvalRoleDenominators) (ModelEvalReadiness, error)
}

type ModelEvalReadiness struct {
	Ready                      bool                        `json:"ready"`
	RegisteredSuites           int                         `json:"registered_suites"`
	RequiredRetrievalSuites    int                         `json:"required_retrieval_suites"`
	SuiteInventoryDeficit      int                         `json:"suite_inventory_deficit"`
	FixedDenominatorSetSHA256  string                      `json:"fixed_denominator_set_sha256,omitempty"`
	MissingDenominatorIDs      []string                    `json:"missing_denominator_suite_ids"`
	InvalidDenominatorIDs      []string                    `json:"invalid_denominator_suite_ids"`
	MissingCorpusValidationIDs []string                    `json:"missing_corpus_validation_suite_ids"`
	InvalidCorpusValidationIDs []string                    `json:"invalid_corpus_validation_suite_ids"`
	CorpusValidation           []ModelEvalCorpusValidation `json:"corpus_validation"`
	StaleExpectedDocumentCases int32                       `json:"stale_expected_document_cases"`
	RoleCoverage               []RoleCoverage              `json:"role_coverage"`
	TaskRoleCoverage           []RoleCoverage              `json:"task_role_coverage"`
	ConsumerCoverage           []RoleCoverage              `json:"consumer_coverage"`
	ObservedRuntimeModels      []ModelEvalRuntimeModel     `json:"observed_runtime_models"`
}

// ModelEvalCorpusValidation is the privacy-safe rollup of Search Hub's latest
// provider-owned corpus probe. Stale expected documents remain in the fixed
// denominator and count as misses; validation never removes or rewrites labels.
type ModelEvalCorpusValidation struct {
	SuiteID        string `json:"suite_id"`
	ValidatedAt    string `json:"validated_at"`
	Positives      int32  `json:"positives"`
	Live           int32  `json:"live"`
	Hard           int32  `json:"hard"`
	Stale          int32  `json:"stale"`
	Inconclusive   int32  `json:"inconclusive"`
	ProviderErrors int32  `json:"provider_errors"`
}

type ModelEvalRuntimeModel struct {
	Scenario         string `json:"scenario"`
	Role             string `json:"role"`
	Provider         string `json:"provider"`
	Locality         string `json:"locality"`
	Model            string `json:"model"`
	Status           string `json:"status"`
	SampleCount      int64  `json:"sample_count"`
	LatestObservedAt string `json:"latest_observed_at"`
}

type RoleCoverage struct {
	Role      string `json:"role"`
	Cases     int    `json:"accepted_cases"`
	Required  int    `json:"required_cases"`
	Deficient bool   `json:"deficient"`
}

type searchHubEvalClient interface {
	ListSuites(context.Context, *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error)
	GetFixedDenominator(context.Context, *connect.Request[evalv1.GetFixedDenominatorRequest]) (*connect.Response[evalv1.GetFixedDenominatorResponse], error)
	GetLatestCorpusValidation(context.Context, *connect.Request[evalv1.GetLatestCorpusValidationRequest]) (*connect.Response[evalv1.GetLatestCorpusValidationResponse], error)
}

type ModelEvalRoleDenominators struct {
	PolicyRoles []string
	TaskRoles   []string
	Consumers   []string
}

type searchHubModelEvalSource struct {
	client searchHubEvalClient
}

type discoveredModelEvalSource struct {
	resolve func(context.Context, string) (string, error)
	http    connect.HTTPClient
}

func newSearchHubModelEvalSource(client searchHubEvalClient) ModelEvalReadinessSource {
	return searchHubModelEvalSource{client: client}
}

func (s discoveredModelEvalSource) Readiness(ctx context.Context, roles ModelEvalRoleDenominators) (ModelEvalReadiness, error) {
	if s.resolve == nil {
		return ModelEvalReadiness{}, fmt.Errorf("Search Hub discovery resolver is not configured")
	}
	baseURL, err := s.resolve(ctx, "search-hub")
	if err != nil {
		return ModelEvalReadiness{}, fmt.Errorf("resolve Search Hub: %w", err)
	}
	client := s.http
	if client == nil {
		client = httpClientWithTimeout
	}
	return newSearchHubModelEvalSource(evalconnect.NewEvalServiceClient(client, baseURL)).Readiness(ctx, roles)
}

func (s searchHubModelEvalSource) Readiness(ctx context.Context, roles ModelEvalRoleDenominators) (ModelEvalReadiness, error) {
	if s.client == nil {
		return ModelEvalReadiness{}, fmt.Errorf("Search Hub eval client is not configured")
	}
	listed, err := s.client.ListSuites(ctx, connect.NewRequest(&evalv1.ListSuitesRequest{}))
	if err != nil {
		return ModelEvalReadiness{}, fmt.Errorf("list Search Hub suites: %w", err)
	}
	if listed == nil || listed.Msg == nil {
		return ModelEvalReadiness{}, fmt.Errorf("Search Hub returned no suite inventory")
	}
	suites := listed.Msg.GetSuites()
	readiness := ModelEvalReadiness{
		RegisteredSuites:           len(suites),
		RequiredRetrievalSuites:    modelEvalMinimumRetrievalSuites,
		MissingDenominatorIDs:      []string{},
		InvalidDenominatorIDs:      []string{},
		MissingCorpusValidationIDs: []string{},
		InvalidCorpusValidationIDs: []string{},
		CorpusValidation:           make([]ModelEvalCorpusValidation, 0, len(suites)),
		RoleCoverage:               make([]RoleCoverage, 0, len(roles.PolicyRoles)),
		TaskRoleCoverage:           make([]RoleCoverage, 0, len(roles.TaskRoles)),
		ConsumerCoverage:           make([]RoleCoverage, 0, len(roles.Consumers)),
	}
	if len(suites) < modelEvalMinimumRetrievalSuites {
		readiness.SuiteInventoryDeficit = modelEvalMinimumRetrievalSuites - len(suites)
	}
	sortedSuites := append([]*evalv1.EvalSuite(nil), suites...)
	sort.Slice(sortedSuites, func(i, j int) bool {
		if sortedSuites[i] == nil {
			return sortedSuites[j] != nil
		}
		if sortedSuites[j] == nil {
			return false
		}
		return sortedSuites[i].GetSuiteId() < sortedSuites[j].GetSuiteId()
	})
	seenSuites := map[string]struct{}{}
	for _, suite := range sortedSuites {
		if suite == nil {
			continue
		}
		id := strings.TrimSpace(suite.GetSuiteId())
		if id == "" {
			continue
		}
		if _, duplicate := seenSuites[id]; duplicate {
			readiness.InvalidDenominatorIDs = append(readiness.InvalidDenominatorIDs, id+":duplicate-suite-id")
		}
		seenSuites[id] = struct{}{}
	}
	var denominatorIdentities []string

	roleSet := make(map[string]struct{}, len(roles.PolicyRoles))
	roleCases := make(map[string]map[string]struct{}, len(roles.PolicyRoles))
	for _, role := range roles.PolicyRoles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		roleSet[role] = struct{}{}
		roleCases[role] = map[string]struct{}{}
	}
	taskRoleSet := make(map[string]struct{}, len(roles.TaskRoles))
	taskRoleCases := make(map[string]map[string]struct{}, len(roles.TaskRoles))
	for _, role := range roles.TaskRoles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		taskRoleSet[role] = struct{}{}
		taskRoleCases[role] = map[string]struct{}{}
	}
	consumerSet := make(map[string]struct{}, len(roles.Consumers))
	consumerCases := make(map[string]map[string]struct{}, len(roles.Consumers))
	for _, consumer := range roles.Consumers {
		consumer = strings.TrimSpace(consumer)
		if consumer == "" {
			continue
		}
		consumerSet[consumer] = struct{}{}
		consumerCases[consumer] = map[string]struct{}{}
	}

	type snapshotResult struct {
		snapshot        *evalv1.FixedDenominatorSnapshot
		found           bool
		err             bool
		validation      *evalv1.GetLatestCorpusValidationResponse
		validationFound bool
		validationErr   bool
	}
	results := make([]snapshotResult, len(sortedSuites))
	var wait sync.WaitGroup
	semaphore := make(chan struct{}, 4)
	for i, suite := range sortedSuites {
		if suite == nil || strings.TrimSpace(suite.GetSuiteId()) == "" {
			readiness.InvalidDenominatorIDs = append(readiness.InvalidDenominatorIDs, "suite-inventory-has-empty-id")
			continue
		}
		wait.Add(1)
		go func(i int, suite *evalv1.EvalSuite) {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				results[i].err = true
				return
			}
			defer func() { <-semaphore }()
			response, err := s.client.GetFixedDenominator(ctx, connect.NewRequest(&evalv1.GetFixedDenominatorRequest{SuiteId: suite.GetSuiteId()}))
			if err != nil {
				results[i].err = true
			} else if response != nil && response.Msg != nil && response.Msg.GetFound() && response.Msg.GetSnapshot() != nil {
				results[i].snapshot = response.Msg.GetSnapshot()
				results[i].found = true
			}
			validation, validationErr := s.client.GetLatestCorpusValidation(ctx, connect.NewRequest(&evalv1.GetLatestCorpusValidationRequest{SuiteId: suite.GetSuiteId()}))
			if validationErr != nil {
				results[i].validationErr = true
				return
			}
			if validation != nil && validation.Msg != nil && validation.Msg.GetFound() {
				results[i].validation = validation.Msg
				results[i].validationFound = true
			}
		}(i, suite)
	}
	wait.Wait()

	for i, suite := range sortedSuites {
		if suite == nil || strings.TrimSpace(suite.GetSuiteId()) == "" {
			continue
		}
		if results[i].err {
			readiness.InvalidDenominatorIDs = append(readiness.InvalidDenominatorIDs, suite.GetSuiteId()+":read-error")
			continue
		}
		if !results[i].found {
			readiness.MissingDenominatorIDs = append(readiness.MissingDenominatorIDs, suite.GetSuiteId())
			continue
		}
		snapshot := results[i].snapshot
		if reason := validateModelEvalSnapshot(suite, snapshot); reason != "" {
			readiness.InvalidDenominatorIDs = append(readiness.InvalidDenominatorIDs, suite.GetSuiteId()+":"+reason)
			continue
		}
		if results[i].validationErr {
			readiness.InvalidCorpusValidationIDs = append(readiness.InvalidCorpusValidationIDs, suite.GetSuiteId()+":read-error")
		} else if !results[i].validationFound {
			readiness.MissingCorpusValidationIDs = append(readiness.MissingCorpusValidationIDs, suite.GetSuiteId())
		} else if reason := validateModelEvalCorpusValidation(suite, snapshot, results[i].validation, time.Now().UTC()); reason != "" {
			readiness.InvalidCorpusValidationIDs = append(readiness.InvalidCorpusValidationIDs, suite.GetSuiteId()+":"+reason)
		} else {
			rollup := results[i].validation.GetRollup()
			readiness.CorpusValidation = append(readiness.CorpusValidation, ModelEvalCorpusValidation{
				SuiteID: suite.GetSuiteId(), ValidatedAt: results[i].validation.GetCreatedAt(),
				Positives: rollup.GetPositives(), Live: rollup.GetLive(), Hard: rollup.GetHard(),
				Stale: rollup.GetStale(), Inconclusive: rollup.GetInconclusive(), ProviderErrors: rollup.GetProviderErrors(),
			})
			readiness.StaleExpectedDocumentCases += rollup.GetStale()
		}
		denominatorIdentities = append(denominatorIdentities, suite.GetSuiteId()+":"+snapshot.GetCorpusSha256()+":"+snapshot.GetDenominatorSha256())
		for _, c := range snapshot.GetCases() {
			if !acceptedCallerCase(c) {
				continue
			}
			role := roleForCase(c)
			if _, known := roleSet[role]; !known {
				continue
			}
			roleCases[role][c.GetCaseId()] = struct{}{}
			taskRole := taskRoleForCase(c)
			if _, known := taskRoleSet[taskRole]; known {
				taskRoleCases[taskRole][c.GetCaseId()] = struct{}{}
			}
			consumer := consumerForCase(c)
			if _, known := consumerSet[consumer]; known {
				consumerCases[consumer][c.GetCaseId()] = struct{}{}
			}
		}
	}

	for _, role := range sortedRoleKeys(roleSet) {
		count := len(roleCases[role])
		readiness.RoleCoverage = append(readiness.RoleCoverage, RoleCoverage{
			Role: role, Cases: count, Required: modelEvalMinimumCasesPerRole,
			Deficient: count < modelEvalMinimumCasesPerRole,
		})
	}
	for _, role := range sortedRoleKeys(taskRoleSet) {
		count := len(taskRoleCases[role])
		readiness.TaskRoleCoverage = append(readiness.TaskRoleCoverage, RoleCoverage{
			Role: role, Cases: count, Required: modelEvalMinimumCasesPerRole,
			Deficient: count < modelEvalMinimumCasesPerRole,
		})
	}
	for _, consumer := range sortedRoleKeys(consumerSet) {
		count := len(consumerCases[consumer])
		readiness.ConsumerCoverage = append(readiness.ConsumerCoverage, RoleCoverage{
			Role: consumer, Cases: count, Required: modelEvalMinimumCasesPerRole,
			Deficient: count < modelEvalMinimumCasesPerRole,
		})
	}
	sort.Strings(readiness.MissingDenominatorIDs)
	sort.Strings(readiness.InvalidDenominatorIDs)
	sort.Strings(readiness.MissingCorpusValidationIDs)
	sort.Strings(readiness.InvalidCorpusValidationIDs)
	readiness.Ready = readiness.SuiteInventoryDeficit == 0 && len(readiness.MissingDenominatorIDs) == 0 && len(readiness.InvalidDenominatorIDs) == 0 && len(readiness.MissingCorpusValidationIDs) == 0 && len(readiness.InvalidCorpusValidationIDs) == 0
	for _, role := range readiness.RoleCoverage {
		readiness.Ready = readiness.Ready && !role.Deficient
	}
	for _, role := range readiness.TaskRoleCoverage {
		readiness.Ready = readiness.Ready && !role.Deficient
	}
	for _, consumer := range readiness.ConsumerCoverage {
		readiness.Ready = readiness.Ready && !consumer.Deficient
	}
	if readiness.SuiteInventoryDeficit == 0 && len(readiness.MissingDenominatorIDs) == 0 && len(readiness.InvalidDenominatorIDs) == 0 && len(readiness.MissingCorpusValidationIDs) == 0 && len(readiness.InvalidCorpusValidationIDs) == 0 && len(denominatorIdentities) == len(sortedSuites) {
		readiness.FixedDenominatorSetSHA256 = searcheval.CaseIDsDigest(denominatorIdentities)
	}
	return readiness, nil
}

func validateModelEvalCorpusValidation(suite *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot, validation *evalv1.GetLatestCorpusValidationResponse, now time.Time) string {
	if suite == nil || snapshot == nil || validation == nil || !validation.GetFound() {
		return "receipt-missing"
	}
	if validation.GetSuiteId() != suite.GetSuiteId() || validation.GetProviderId() != suite.GetProviderId() {
		return "suite-identity-mismatch"
	}
	if !isSHA256(validation.GetSuiteSha256()) || validation.GetSuiteSha256() != searcheval.SuiteCorpusDigest(suite) {
		return "suite-digest-mismatch"
	}
	if !sameSet(validation.GetDenominatorCaseIds(), snapshot.GetCaseIds()) {
		return "denominator-case-set-mismatch"
	}
	if !isSHA256(validation.GetDenominatorSha256()) || !isSHA256(validation.GetReceiptSha256()) {
		return "receipt-digest-invalid"
	}
	rollup := validation.GetRollup()
	if rollup == nil || int(rollup.GetPositives()) != len(snapshot.GetCaseIds()) ||
		rollup.GetLive()+rollup.GetHard()+rollup.GetStale()+rollup.GetInconclusive()+rollup.GetProviderErrors() != rollup.GetPositives() {
		return "rollup-denominator-mismatch"
	}
	createdAt, err := time.Parse(time.RFC3339Nano, validation.GetCreatedAt())
	if err != nil {
		return "validation-timestamp-invalid"
	}
	age := now.Sub(createdAt)
	if age < 0 || age > modelEvalCorpusValidationMaxAge {
		return "validation-receipt-stale"
	}
	return ""
}

func validateModelEvalSnapshot(suite *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot) string {
	if snapshot.GetSuiteId() != suite.GetSuiteId() || snapshot.GetProviderId() != suite.GetProviderId() {
		return "suite-identity-mismatch"
	}
	if snapshot.GetReviewerId() == "" || snapshot.GetReviewerId() == suite.GetProviderId() || strings.TrimSpace(snapshot.GetReviewEvidenceUri()) == "" {
		return "independent-review-identity-missing"
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshot.GetReviewedAt()); err != nil {
		return "review-timestamp-invalid"
	}
	if !isSHA256(snapshot.GetCorpusSha256()) || !isSHA256(snapshot.GetDenominatorSha256()) {
		return "digest-invalid"
	}
	if searcheval.SuiteCorpusDigest(suite) != snapshot.GetCorpusSha256() {
		return "suite-changed-since-review"
	}
	caseIDs := make([]string, 0, len(snapshot.GetCases()))
	caseSet := map[string]struct{}{}
	searchCases := make([]aisearch.TestCase, 0, len(snapshot.GetCases()))
	for _, c := range snapshot.GetCases() {
		if c == nil || strings.TrimSpace(c.GetCaseId()) == "" {
			return "case-identity-missing"
		}
		if _, duplicate := caseSet[c.GetCaseId()]; duplicate {
			return "duplicate-case-id"
		}
		caseSet[c.GetCaseId()] = struct{}{}
		caseIDs = append(caseIDs, c.GetCaseId())
		searchCases = append(searchCases, toModelEvalSearchCase(c))
	}
	if len(caseIDs) == 0 || !sameSet(caseIDs, snapshot.GetCaseIds()) {
		return "case-snapshot-mismatch"
	}
	sealed, err := aisearch.NewFixedDenominator(snapshot.GetSuiteId(), searchCases, snapshot.GetReviewerId(), snapshot.GetReviewEvidenceUri())
	if err != nil || sealed.SHA256 != snapshot.GetDenominatorSha256() {
		return "denominator-digest-mismatch"
	}
	if err := sealed.Validate(); err != nil {
		return "denominator-invalid"
	}
	return ""
}

func toModelEvalSearchCase(c *evalv1.EvalCase) aisearch.TestCase {
	return aisearch.TestCase{
		ID: c.GetCaseId(), Query: c.GetQuery(), Scope: c.GetScope(), Status: c.GetStatus(),
		Tags: append([]string(nil), c.GetTags()...), ExpectIDs: append([]string(nil), c.GetExpectIds()...),
		ExpectWithinTopK: int(c.GetExpectWithinTopK()), ExpectMinScore: c.GetExpectMinScore(),
		ExpectMaxScore: c.GetExpectMaxScore(), ExpectNoStrongHit: c.GetExpectNoStrongHit(), Note: c.GetNote(),
	}
}

func acceptedCallerCase(c *evalv1.EvalCase) bool {
	if c == nil || c.GetStatus() != "reviewed" || len(c.GetExpectIds()) == 0 || hasEvalTag(c, "generated") || !hasEvalTag(c, "caller-audit") || !hasEvalTag(c, "agent-manager") || !hasEvalTag(c, "independently-reviewed") {
		return false
	}
	role := roleForCase(c)
	consumer := consumerForCase(c)
	if role == "" || taskRoleForCase(c) == "" || consumer == "" || c.GetCaseId() != evalCallerCaseID(c.GetNote(), role) {
		return false
	}
	fields := make(map[string]string)
	for _, raw := range strings.Fields(c.GetNote()) {
		key, value, ok := strings.Cut(raw, "=")
		if ok {
			fields[key] = value
		}
	}
	source := strings.TrimPrefix(fields["source"], "agent-manager://runs/")
	parts := strings.Split(source, "/events/")
	sequence, err := strconv.ParseInt(fields["seq"], 10, 64)
	if err != nil || sequence < 0 || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	expectedRequestID, err := searcheval.AgentManagerReplayRequestID(parts[0], parts[1], sequence, taskRoleForCase(c), consumer)
	if err != nil || fields["gateway_request"] != expectedRequestID {
		return false
	}
	eventID := fields["gateway_event"]
	return fields["gateway_request"] != "" && fields["selected_resource_role"] == role && fields["model"] != "" &&
		fields["task_role"] == taskRoleForCase(c) &&
		fields["consumer"] == consumer &&
		fields["role_evidence"] == "gateway-ledger://events/"+eventID &&
		strings.HasPrefix(fields["capacity_estimate"], "capacity-footprint://") &&
		fields["label_review"] != "" && fields["reviewer"] != ""
}

func taskRoleForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "task-role:") {
			return strings.TrimPrefix(tag, "task-role:")
		}
	}
	return ""
}

func consumerForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "consumer:") {
			return strings.TrimPrefix(tag, "consumer:")
		}
	}
	return ""
}

func evalCallerCaseID(note, role string) string {
	fields := make(map[string]string)
	for _, raw := range strings.Fields(note) {
		key, value, ok := strings.Cut(raw, "=")
		if ok {
			fields[key] = value
		}
	}
	source := strings.TrimPrefix(fields["source"], "agent-manager://runs/")
	parts := strings.Split(source, "/events/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || fields["gateway_event"] == "" {
		return ""
	}
	material := parts[0] + "/" + parts[1] + "/" + role + "/" + fields["gateway_event"]
	sum := sha256.Sum256([]byte(material))
	return "caller-" + hex.EncodeToString(sum[:12])
}

func roleForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "resource-role:") {
			return strings.TrimPrefix(tag, "resource-role:")
		}
	}
	return ""
}

func hasEvalTag(c *evalv1.EvalCase, tag string) bool {
	for _, candidate := range c.GetTags() {
		if candidate == tag {
			return true
		}
	}
	return false
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int, len(left))
	for _, value := range left {
		seen[value]++
	}
	for _, value := range right {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func sortedRoleKeys(values map[string]struct{}) []string {
	roles := make([]string, 0, len(values))
	for role := range values {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

var _ searchHubEvalClient = evalconnect.EvalServiceClient(nil)

// NewDiscoveredSearchHubModelEvalSource creates a lazy client using Vrooli's
// service discovery. It never starts Search Hub or calls inference.
func NewDiscoveredSearchHubModelEvalSource(resolve func(context.Context, string) (string, error), httpClient connect.HTTPClient) ModelEvalReadinessSource {
	return discoveredModelEvalSource{resolve: resolve, http: httpClient}
}

var httpClientWithTimeout = &http.Client{Timeout: 15 * time.Second}

// ReadModelEvalReadiness inspects accepted suite snapshots without execution.
// It is separate from RunSuite and therefore safe without inference admission.
func ReadModelEvalReadiness(ctx context.Context, source ModelEvalReadinessSource, roles ModelEvalRoleDenominators) (ModelEvalReadiness, error) {
	if source == nil {
		return ModelEvalReadiness{}, fmt.Errorf("model-eval accepted-suite source is not configured")
	}
	return source.Readiness(ctx, roles)
}

func LoadGatewayTaskRoles(catalogPath string) ([]string, error) {
	if strings.TrimSpace(catalogPath) == "" {
		return nil, fmt.Errorf("AI Gateway task-role catalog path is not configured")
	}
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("read AI Gateway task-role catalog: %w", err)
	}
	var catalog struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("decode AI Gateway task-role catalog: %w", err)
	}
	roles := make([]string, 0, len(catalog.Roles))
	for role := range catalog.Roles {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("AI Gateway task-role catalog declares no roles")
	}
	sort.Strings(roles)
	return roles, nil
}

func LoadModelEvalRoles(policyPath string) ([]string, error) {
	if strings.TrimSpace(policyPath) == "" {
		return nil, fmt.Errorf("authoritative Ollama model policy path is not configured")
	}
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("read authoritative Ollama model policy: %w", err)
	}
	var policy struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return nil, fmt.Errorf("decode authoritative Ollama model policy: %w", err)
	}
	roles := make([]string, 0, len(policy.Roles))
	for role := range policy.Roles {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("authoritative Ollama model policy declares no roles")
	}
	sort.Strings(roles)
	return roles, nil
}
