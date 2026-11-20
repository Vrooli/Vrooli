package conformance

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
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
	suites, err := s.listSuites(ctx)
	if err != nil {
		return ModelEvalReadiness{}, err
	}
	work := newModelEvalReadinessWork(suites, roles)
	results := s.readSuiteResults(ctx, work.suites, &work.readiness)
	work.applySuiteResults(results)
	work.finish()
	return work.readiness, nil
}

func (s searchHubModelEvalSource) listSuites(ctx context.Context) ([]*evalv1.EvalSuite, error) {
	listed, err := s.client.ListSuites(ctx, connect.NewRequest(&evalv1.ListSuitesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list Search Hub suites: %w", err)
	}
	if listed == nil || listed.Msg == nil {
		return nil, fmt.Errorf("Search Hub returned no suite inventory")
	}
	return listed.Msg.GetSuites(), nil
}

type modelEvalSnapshotResult struct {
	snapshot        *evalv1.FixedDenominatorSnapshot
	found           bool
	err             bool
	validation      *evalv1.GetLatestCorpusValidationResponse
	validationFound bool
	validationErr   bool
}

type modelEvalReadinessWork struct {
	readiness      ModelEvalReadiness
	suites         []*evalv1.EvalSuite
	denominatorIDs []string
	roleSet        map[string]struct{}
	roleCases      map[string]map[string]struct{}
	taskRoleSet    map[string]struct{}
	taskRoleCases  map[string]map[string]struct{}
	consumerSet    map[string]struct{}
	consumerCases  map[string]map[string]struct{}
}

func newModelEvalReadinessWork(suites []*evalv1.EvalSuite, roles ModelEvalRoleDenominators) modelEvalReadinessWork {
	readiness := ModelEvalReadiness{
		RegisteredSuites: len(suites), RequiredRetrievalSuites: modelEvalMinimumRetrievalSuites,
		MissingDenominatorIDs: []string{}, InvalidDenominatorIDs: []string{},
		MissingCorpusValidationIDs: []string{}, InvalidCorpusValidationIDs: []string{},
		CorpusValidation: make([]ModelEvalCorpusValidation, 0, len(suites)),
		RoleCoverage:     make([]RoleCoverage, 0, len(roles.PolicyRoles)),
		TaskRoleCoverage: make([]RoleCoverage, 0, len(roles.TaskRoles)),
		ConsumerCoverage: make([]RoleCoverage, 0, len(roles.Consumers)),
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
	work := modelEvalReadinessWork{readiness: readiness, suites: sortedSuites}
	work.markDuplicateSuites()
	work.roleSet, work.roleCases = modelEvalCoverageSets(roles.PolicyRoles)
	work.taskRoleSet, work.taskRoleCases = modelEvalCoverageSets(roles.TaskRoles)
	work.consumerSet, work.consumerCases = modelEvalCoverageSets(roles.Consumers)
	return work
}

func (w *modelEvalReadinessWork) markDuplicateSuites() {
	seen := map[string]struct{}{}
	for _, suite := range w.suites {
		if suite == nil {
			continue
		}
		id := strings.TrimSpace(suite.GetSuiteId())
		if id == "" {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			w.readiness.InvalidDenominatorIDs = append(w.readiness.InvalidDenominatorIDs, id+":duplicate-suite-id")
		}
		seen[id] = struct{}{}
	}
}

func modelEvalCoverageSets(names []string) (map[string]struct{}, map[string]map[string]struct{}) {
	set := make(map[string]struct{}, len(names))
	cases := make(map[string]map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
		cases[name] = map[string]struct{}{}
	}
	return set, cases
}

func (s searchHubModelEvalSource) readSuiteResults(ctx context.Context, suites []*evalv1.EvalSuite, readiness *ModelEvalReadiness) []modelEvalSnapshotResult {
	results := make([]modelEvalSnapshotResult, len(suites))
	var wait sync.WaitGroup
	semaphore := make(chan struct{}, 4)
	for i, suite := range suites {
		if suite == nil || strings.TrimSpace(suite.GetSuiteId()) == "" {
			readiness.InvalidDenominatorIDs = append(readiness.InvalidDenominatorIDs, "suite-inventory-has-empty-id")
			continue
		}
		wait.Add(1)
		go s.readSuiteResult(ctx, i, suite, semaphore, &wait, results)
	}
	wait.Wait()
	return results
}

func (s searchHubModelEvalSource) readSuiteResult(ctx context.Context, i int, suite *evalv1.EvalSuite, semaphore chan struct{}, wait *sync.WaitGroup, results []modelEvalSnapshotResult) {
	defer wait.Done()
	select {
	case semaphore <- struct{}{}:
	case <-ctx.Done():
		results[i].err = true
		return
	}
	defer func() { <-semaphore }()
	s.readFixedDenominator(ctx, suite, &results[i])
	s.readCorpusValidation(ctx, suite, &results[i])
}

func (s searchHubModelEvalSource) readFixedDenominator(ctx context.Context, suite *evalv1.EvalSuite, result *modelEvalSnapshotResult) {
	response, err := s.client.GetFixedDenominator(ctx, connect.NewRequest(&evalv1.GetFixedDenominatorRequest{SuiteId: suite.GetSuiteId()}))
	if err != nil {
		result.err = true
	} else if response != nil && response.Msg != nil && response.Msg.GetFound() && response.Msg.GetSnapshot() != nil {
		result.snapshot, result.found = response.Msg.GetSnapshot(), true
	}
}

func (s searchHubModelEvalSource) readCorpusValidation(ctx context.Context, suite *evalv1.EvalSuite, result *modelEvalSnapshotResult) {
	validation, validationErr := s.client.GetLatestCorpusValidation(ctx, connect.NewRequest(&evalv1.GetLatestCorpusValidationRequest{SuiteId: suite.GetSuiteId()}))
	if validationErr != nil {
		result.validationErr = true
		return
	}
	if validation != nil && validation.Msg != nil && validation.Msg.GetFound() {
		result.validation, result.validationFound = validation.Msg, true
	}
}

func (w *modelEvalReadinessWork) applySuiteResults(results []modelEvalSnapshotResult) {
	for i, suite := range w.suites {
		if suite == nil || strings.TrimSpace(suite.GetSuiteId()) == "" {
			continue
		}
		result := results[i]
		if result.err {
			w.readiness.InvalidDenominatorIDs = append(w.readiness.InvalidDenominatorIDs, suite.GetSuiteId()+":read-error")
			continue
		}
		if !result.found {
			w.readiness.MissingDenominatorIDs = append(w.readiness.MissingDenominatorIDs, suite.GetSuiteId())
			continue
		}
		if reason := validateModelEvalSnapshot(suite, result.snapshot); reason != "" {
			w.readiness.InvalidDenominatorIDs = append(w.readiness.InvalidDenominatorIDs, suite.GetSuiteId()+":"+reason)
			continue
		}
		w.applyValidation(suite, result)
		w.denominatorIDs = append(w.denominatorIDs, suite.GetSuiteId()+":"+result.snapshot.GetCorpusSha256()+":"+result.snapshot.GetDenominatorSha256())
		w.collectCallerCases(result.snapshot.GetCases())
	}
}

func (w *modelEvalReadinessWork) applyValidation(suite *evalv1.EvalSuite, result modelEvalSnapshotResult) {
	if result.validationErr {
		w.readiness.InvalidCorpusValidationIDs = append(w.readiness.InvalidCorpusValidationIDs, suite.GetSuiteId()+":read-error")
		return
	}
	if !result.validationFound {
		w.readiness.MissingCorpusValidationIDs = append(w.readiness.MissingCorpusValidationIDs, suite.GetSuiteId())
		return
	}
	if reason := validateModelEvalCorpusValidation(suite, result.snapshot, result.validation, time.Now().UTC()); reason != "" {
		w.readiness.InvalidCorpusValidationIDs = append(w.readiness.InvalidCorpusValidationIDs, suite.GetSuiteId()+":"+reason)
		return
	}
	rollup := result.validation.GetRollup()
	w.readiness.CorpusValidation = append(w.readiness.CorpusValidation, ModelEvalCorpusValidation{
		SuiteID: suite.GetSuiteId(), ValidatedAt: result.validation.GetCreatedAt(),
		Positives: rollup.GetPositives(), Live: rollup.GetLive(), Hard: rollup.GetHard(), Stale: rollup.GetStale(),
		Inconclusive: rollup.GetInconclusive(), ProviderErrors: rollup.GetProviderErrors(),
	})
	w.readiness.StaleExpectedDocumentCases += rollup.GetStale()
}

func (w *modelEvalReadinessWork) collectCallerCases(cases []*evalv1.EvalCase) {
	for _, c := range cases {
		if !acceptedCallerCase(c) {
			continue
		}
		role := roleForCase(c)
		if _, known := w.roleSet[role]; !known {
			continue
		}
		w.roleCases[role][c.GetCaseId()] = struct{}{}
		taskRole := taskRoleForCase(c)
		if _, known := w.taskRoleSet[taskRole]; known {
			w.taskRoleCases[taskRole][c.GetCaseId()] = struct{}{}
		}
		consumer := consumerForCase(c)
		if _, known := w.consumerSet[consumer]; known {
			w.consumerCases[consumer][c.GetCaseId()] = struct{}{}
		}
	}
}

func (w *modelEvalReadinessWork) finish() {
	w.readiness.RoleCoverage = coverageRows(w.roleSet, w.roleCases)
	w.readiness.TaskRoleCoverage = coverageRows(w.taskRoleSet, w.taskRoleCases)
	w.readiness.ConsumerCoverage = coverageRows(w.consumerSet, w.consumerCases)
	sort.Strings(w.readiness.MissingDenominatorIDs)
	sort.Strings(w.readiness.InvalidDenominatorIDs)
	sort.Strings(w.readiness.MissingCorpusValidationIDs)
	sort.Strings(w.readiness.InvalidCorpusValidationIDs)
	w.readiness.Ready = modelEvalEvidenceComplete(w.readiness)
	if modelEvalDenominatorSetComplete(w.readiness) && len(w.denominatorIDs) == len(w.suites) {
		w.readiness.FixedDenominatorSetSHA256 = searcheval.CaseIDsDigest(w.denominatorIDs)
	}
}

func coverageRows(names map[string]struct{}, cases map[string]map[string]struct{}) []RoleCoverage {
	rows := make([]RoleCoverage, 0, len(names))
	for _, name := range sortedRoleKeys(names) {
		count := len(cases[name])
		rows = append(rows, RoleCoverage{Role: name, Cases: count, Required: modelEvalMinimumCasesPerRole, Deficient: count < modelEvalMinimumCasesPerRole})
	}
	return rows
}

func modelEvalEvidenceComplete(r ModelEvalReadiness) bool {
	if !modelEvalDenominatorSetComplete(r) {
		return false
	}
	for _, rows := range [][]RoleCoverage{r.RoleCoverage, r.TaskRoleCoverage, r.ConsumerCoverage} {
		for _, row := range rows {
			if row.Deficient {
				return false
			}
		}
	}
	return true
}

func modelEvalDenominatorSetComplete(r ModelEvalReadiness) bool {
	return r.SuiteInventoryDeficit == 0 && len(r.MissingDenominatorIDs) == 0 && len(r.InvalidDenominatorIDs) == 0 &&
		len(r.MissingCorpusValidationIDs) == 0 && len(r.InvalidCorpusValidationIDs) == 0
}

func validateModelEvalCorpusValidation(suite *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot, validation *evalv1.GetLatestCorpusValidationResponse, now time.Time) string {
	if suite == nil || snapshot == nil || validation == nil || !validation.GetFound() {
		return "receipt-missing"
	}
	if reason := validateCorpusValidationIdentity(suite, snapshot, validation); reason != "" {
		return reason
	}
	if reason := validateCorpusValidationRollup(snapshot, validation); reason != "" {
		return reason
	}
	return validateCorpusValidationTimestamp(validation, now)
}

func validateCorpusValidationIdentity(suite *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot, validation *evalv1.GetLatestCorpusValidationResponse) string {
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
	return ""
}

func validateCorpusValidationRollup(snapshot *evalv1.FixedDenominatorSnapshot, validation *evalv1.GetLatestCorpusValidationResponse) string {
	rollup := validation.GetRollup()
	if rollup == nil || int(rollup.GetPositives()) != len(snapshot.GetCaseIds()) ||
		rollup.GetLive()+rollup.GetHard()+rollup.GetStale()+rollup.GetInconclusive()+rollup.GetProviderErrors() != rollup.GetPositives() {
		return "rollup-denominator-mismatch"
	}
	return ""
}

func validateCorpusValidationTimestamp(validation *evalv1.GetLatestCorpusValidationResponse, now time.Time) string {
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
	if reason := validateModelEvalReview(suite, snapshot); reason != "" {
		return reason
	}
	caseIDs, searchCases, reason := modelEvalSnapshotCases(snapshot)
	if reason != "" {
		return reason
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

func validateModelEvalReview(suite *evalv1.EvalSuite, snapshot *evalv1.FixedDenominatorSnapshot) string {
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
	return ""
}

func modelEvalSnapshotCases(snapshot *evalv1.FixedDenominatorSnapshot) ([]string, []aisearch.TestCase, string) {
	caseIDs := make([]string, 0, len(snapshot.GetCases()))
	caseSet := map[string]struct{}{}
	searchCases := make([]aisearch.TestCase, 0, len(snapshot.GetCases()))
	for _, c := range snapshot.GetCases() {
		if c == nil || strings.TrimSpace(c.GetCaseId()) == "" {
			return nil, nil, "case-identity-missing"
		}
		if _, duplicate := caseSet[c.GetCaseId()]; duplicate {
			return nil, nil, "duplicate-case-id"
		}
		caseSet[c.GetCaseId()] = struct{}{}
		caseIDs = append(caseIDs, c.GetCaseId())
		searchCases = append(searchCases, toModelEvalSearchCase(c))
	}
	return caseIDs, searchCases, ""
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

var httpClientWithTimeout = &http.Client{Timeout: 15 * time.Second}
