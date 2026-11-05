package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// UseClass is server-derived from the validated feature contract and request
// shape. It must never be copied from an untrusted request body.
type UseClass string

const (
	UseOwnAccount     UseClass = "own_account"
	UseBoundedFeature UseClass = "bounded_feature"
	UseInApp          UseClass = "in_app"
	UseHarnessAccess  UseClass = "harness_access"
)

var useClassOrder = []UseClass{UseHarnessAccess, UseInApp, UseBoundedFeature, UseOwnAccount}

// Permission is the supplier's permission for a given plan and use class.
type Permission string

const (
	PermissionAllowed              Permission = "allowed"
	PermissionNeedsWrittenApproval Permission = "needs_written_approval"
	PermissionForbidden            Permission = "forbidden"
	PermissionSilent               Permission = "silent"
	PermissionUnverified           Permission = "unverified"
)

type ReviewStatus string

const (
	ReviewProposed ReviewStatus = "proposed"
	ReviewPending  ReviewStatus = "pending_review"
	ReviewApproved ReviewStatus = "approved"
	ReviewReopened ReviewStatus = "reopened"
	ReviewExpired  ReviewStatus = "expired"
	ReviewDenied   ReviewStatus = "denied"
)

type AccessTier string

const (
	AccessTierGrant    AccessTier = "1a"
	AccessTierNoBar    AccessTier = "1b"
	AccessTierSignup   AccessTier = "2"
	AccessTierApply    AccessTier = "3"
	AccessTierContract AccessTier = "4"
	AccessTierBarred   AccessTier = "barred"
	AccessTierUnknown  AccessTier = "unknown"
)

type Evidence struct {
	URL          string `json:"url"`
	DocumentDate string `json:"document_date,omitempty"`
	ReadAt       string `json:"read_at,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

type Review struct {
	Status      ReviewStatus `json:"status"`
	DecisionRef string       `json:"decision_ref,omitempty"`
}

type PlanRule struct {
	Plan       string                  `json:"plan"`
	AccessTier AccessTier              `json:"access_tier"`
	UseClasses map[UseClass]Permission `json:"use_classes"`
	Conditions []string                `json:"conditions,omitempty"`
	Clause     string                  `json:"clause,omitempty"`
	Quote      string                  `json:"quote,omitempty"`
	Reason     string                  `json:"reason,omitempty"`
}

type DownstreamRules struct {
	DataCollection string   `json:"data_collection,omitempty"`
	PinnedHosts    []string `json:"pinned_hosts,omitempty"`
	PinFeePercent  *float64 `json:"pin_fee_pct,omitempty"`
}

// ProviderTerms is the supplier's evidence-backed contract record. It is data,
// never executable policy code; ReviewApproved is valid only with a decision
// reference supplied by the Notification Hub authority.
type ProviderTerms struct {
	PlanRules     []PlanRule      `json:"plan_rules"`
	Evidence      Evidence        `json:"evidence"`
	Review        Review          `json:"review"`
	EffectiveFrom string          `json:"effective_from"`
	NextReviewAt  string          `json:"next_review_at,omitempty"`
	ExpiresAt     string          `json:"expires_at,omitempty"`
	OnExpiry      string          `json:"on_expiry"`
	Downstream    DownstreamRules `json:"downstream,omitempty"`
}

type TermsOverride struct {
	UseClasses map[UseClass]Permission `json:"use_classes"`
	Reason     string                  `json:"reason"`
	Evidence   Evidence                `json:"evidence"`
}

type TrustDecision struct {
	CustomerCodeAllowed *bool    `json:"customer_code_allowed"`
	Facts               []string `json:"facts"`
	DecisionRef         string   `json:"decision_ref,omitempty"`
}

type Agreement struct {
	Supplier         string        `json:"supplier"`
	Trust            TrustDecision `json:"trust"`
	UseClasses       []UseClass    `json:"use_classes,omitempty"`
	Models           []string      `json:"models,omitempty"`
	EffectiveAt      time.Time     `json:"effective_at,omitempty"`
	ExpiresAt        time.Time     `json:"expires_at,omitempty"`
	DecisionRef      string        `json:"decision_ref,omitempty"`
	decisionVerified bool
}

// DecisionVerifier must check a reference against the authoritative
// Notification Hub decision store. Empty or missing verifiers deny approvals.
type DecisionVerifier func(decisionRef string) bool

// NewVerifiedRuleRecord creates an eligibility record only after the approval
// reference has been validated by the Notification Hub authority. Proposed or
// pending records are loadable but never eligibility-bearing.
func NewVerifiedRuleRecord(supplier string, terms ProviderTerms, models map[string]ModelRule, verify DecisionVerifier) (RuleRecord, error) {
	record := RuleRecord{Supplier: strings.TrimSpace(supplier), Terms: terms, Models: models}
	if record.Supplier == "" {
		return RuleRecord{}, errors.New("supplier is required")
	}
	if terms.Review.Status != ReviewApproved {
		return record, nil
	}
	if terms.Review.DecisionRef == "" || verify == nil || !verify(terms.Review.DecisionRef) {
		return RuleRecord{}, errors.New("provider rule approval is not verified by Notification Hub")
	}
	record.decisionVerified = true
	return record, nil
}

// NewVerifiedAgreement binds a narrowly-scoped grant/trust decision to a
// Notification Hub reference before it can widen a rule or trust customer code.
func NewVerifiedAgreement(agreement Agreement, verify DecisionVerifier) (Agreement, error) {
	if strings.TrimSpace(agreement.Supplier) == "" || agreement.DecisionRef == "" || verify == nil || !verify(agreement.DecisionRef) {
		return Agreement{}, errors.New("agreement decision is not verified by Notification Hub")
	}
	if len(agreement.Models) == 0 {
		return Agreement{}, errors.New("agreement must name at least one model")
	}
	agreement.Models = append([]string(nil), agreement.Models...)
	for i, model := range agreement.Models {
		model = strings.TrimSpace(model)
		if model == "" {
			return Agreement{}, errors.New("agreement models must be non-empty and trimmed")
		}
		agreement.Models[i] = model
	}
	if agreement.Trust.DecisionRef != "" && agreement.Trust.DecisionRef != agreement.DecisionRef {
		return Agreement{}, errors.New("trust decision reference must match the verified Notification Hub decision")
	}
	agreement.Supplier = strings.TrimSpace(agreement.Supplier)
	agreement.decisionVerified = true
	return agreement, nil
}

// HarnessBillingSource captures a runner-owned billing-source constraint. It
// remains distinct from supplier permission and customer-code trust.
type HarnessBillingSource struct {
	ID                    string     `json:"id"`
	LauncherConfigAllowed *bool      `json:"launcher_config_allowed"`
	CustomerConfigAllowed Permission `json:"customer_config_allowed"`
	Evidence              Evidence   `json:"evidence"`
	Review                Review     `json:"review"`
	decisionVerified      bool
}

// HarnessBillingSourceDigest binds one runner-owned billing condition to its
// human decision. Supplier route data cannot create runner permission.
func HarnessBillingSourceDigest(harness string, source HarnessBillingSource) (string, error) {
	if strings.TrimSpace(harness) == "" || strings.TrimSpace(source.ID) == "" {
		return "", errors.New("harness and billing source required")
	}
	source.Review = Review{}
	data, err := json.Marshal(struct {
		Scope   string               `json:"scope"`
		Harness string               `json:"harness"`
		Source  HarnessBillingSource `json:"source"`
	}{ProviderDecisionScopeRule, strings.TrimSpace(harness), source})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func NewVerifiedHarnessBillingSource(source HarnessBillingSource, verify DecisionVerifier) (HarnessBillingSource, error) {
	if source.Review.Status != ReviewApproved {
		return source, nil
	}
	if source.Review.DecisionRef == "" || verify == nil || !verify(source.Review.DecisionRef) {
		return HarnessBillingSource{}, errors.New("harness billing approval is not verified")
	}
	source.decisionVerified = true
	return source, nil
}

type Route struct {
	ID            string                `json:"id"`
	Supplier      string                `json:"supplier"`
	Plan          string                `json:"plan"`
	Model         string                `json:"model"`
	Endpoint      string                `json:"endpoint"`
	AccountID     string                `json:"account_id,omitempty"`
	Profile       string                `json:"profile,omitempty"`
	QuoteRef      string                `json:"quote_ref,omitempty"`
	QuotedCredits int64                 `json:"quoted_credits"`
	StreamQuote   StreamQuote           `json:"stream_quote,omitempty"`
	Capabilities  []string              `json:"capabilities,omitempty"`
	Harness       string                `json:"harness,omitempty"`
	HarnessSource *HarnessBillingSource `json:"-"`
}

// StreamQuote is a supplier-owned upper-bound quote for one text generation.
// The price applies to the entire input/output envelope, including tools and
// reasoning tokens. A route without an enforceable quote cannot stream.
type StreamQuote struct {
	MaxInputBytes   int    `json:"max_input_bytes"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	MaxCostMicros   int64  `json:"max_cost_micros"`
	TokenLimitField string `json:"token_limit_field"`
}

type FeatureContract struct {
	ID                      string
	MaxInputBytes           int
	MaxOutputBytes          int
	MaxUserTurns            int
	PinnedInstructionDigest string
}

type FeatureInvocation struct {
	InstructionHash      string
	Endpoint             string
	RequiredCapabilities []string
	UserTurns            int
	ToolCount            int
	InputBytes           int
	OutputBytes          int
	SchemaValid          bool
	HostedService        bool
}

// FeatureCheck is constructible only through CheckFeatureInvocation. The
// unexported checked bit prevents an untrusted decoded request from assigning
// a feature id or use class directly.
type FeatureCheck struct {
	featureID            string
	endpoint             string
	requiredCapabilities []string
	pinned               bool
	hostedService        bool
	userTurns            int
	toolCount            int
	inputBytes           int
	maxInputBytes        int
	outputBytes          int
	maxOutputBytes       int
	schemaValid          bool
	checked              bool
}

// Request contains only facts computed by server-side contract validation.
// Call NewEligibleRequest after resolving the feature from the endpoint path;
// never decode Request directly from HTTP JSON.
type Request struct {
	featureID            string
	endpoint             string
	requiredCapabilities []string
	useClass             UseClass
	model                string
	profile              string
	budget               int64
	privacy              string
	harness              string
}

func CheckFeatureInvocation(contract FeatureContract, invocation FeatureInvocation) (FeatureCheck, error) {
	if err := validateFeatureContract(contract); err != nil {
		return FeatureCheck{}, err
	}
	if err := validateFeatureInvocation(contract, invocation); err != nil {
		return FeatureCheck{}, err
	}
	pinned := strings.TrimSpace(contract.PinnedInstructionDigest) != ""
	return FeatureCheck{
		featureID: contract.ID, endpoint: strings.TrimSpace(invocation.Endpoint),
		requiredCapabilities: append([]string(nil), invocation.RequiredCapabilities...),
		pinned:               pinned, hostedService: invocation.HostedService,
		userTurns: invocation.UserTurns, toolCount: invocation.ToolCount,
		inputBytes: invocation.InputBytes, maxInputBytes: contract.MaxInputBytes,
		outputBytes: invocation.OutputBytes, maxOutputBytes: contract.MaxOutputBytes,
		schemaValid: invocation.SchemaValid, checked: true,
	}, nil
}

func validateFeatureContract(contract FeatureContract) error {
	if strings.TrimSpace(contract.ID) == "" || contract.ID != strings.TrimSpace(contract.ID) {
		return errors.New("declared feature id is required and must be trimmed")
	}
	if contract.MaxInputBytes <= 0 || contract.MaxOutputBytes <= 0 || contract.MaxUserTurns != 1 {
		return errors.New("declared feature limits are invalid")
	}
	if digest := strings.TrimSpace(contract.PinnedInstructionDigest); digest != "" && !validInstructionDigest(digest) {
		return errors.New("pinned instruction digest must be a lowercase SHA-256 hex digest")
	}
	return nil
}

func validateFeatureInvocation(contract FeatureContract, invocation FeatureInvocation) error {
	if err := validateResolvedRole(invocation); err != nil {
		return err
	}
	if err := validateInvocationShape(invocation); err != nil {
		return err
	}
	return validateInvocationLimits(contract, invocation)
}

func validateInvocationShape(invocation FeatureInvocation) error {
	if invocation.UserTurns != 1 || invocation.ToolCount != 0 || !invocation.SchemaValid {
		return errors.New("feature contract check failed")
	}
	return nil
}

func validateInvocationLimits(contract FeatureContract, invocation FeatureInvocation) error {
	if invocation.InputBytes < 0 || invocation.InputBytes > contract.MaxInputBytes || invocation.OutputBytes < 0 || invocation.OutputBytes > contract.MaxOutputBytes {
		return errors.New("feature input/output exceeds contract cap")
	}
	pinned := strings.TrimSpace(contract.PinnedInstructionDigest) != ""
	if pinned && (contract.PinnedInstructionDigest != invocation.InstructionHash) {
		return errors.New("feature instruction digest mismatch")
	}
	return nil
}

func validateResolvedRole(invocation FeatureInvocation) error {
	if strings.TrimSpace(invocation.Endpoint) == "" || strings.TrimSpace(invocation.Endpoint) != invocation.Endpoint {
		return errors.New("resolved endpoint is required and must be trimmed")
	}
	seen := make(map[string]struct{}, len(invocation.RequiredCapabilities))
	for _, capability := range invocation.RequiredCapabilities {
		if strings.TrimSpace(capability) == "" || capability != strings.TrimSpace(capability) {
			return errors.New("resolved capabilities must be trimmed and non-empty")
		}
		if _, exists := seen[capability]; exists {
			return fmt.Errorf("resolved capabilities contain duplicate %q", capability)
		}
		seen[capability] = struct{}{}
	}
	return nil
}

func validInstructionDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func NewEligibleRequest(check FeatureCheck, model, profile, privacy, harness string, budget int64) (Request, error) {
	useClass := UseInApp
	if check.pinned || check.hostedService {
		useClass = UseBoundedFeature
	} else if strings.TrimSpace(harness) != "" {
		useClass = UseHarnessAccess
	}
	return NewEligibleRequestForUse(check, useClass, model, profile, privacy, harness, budget)
}

// NewEligibleRequestForUse accepts a caller class selected by trusted server
// routing code after contract validation. It must never receive a request-body
// use class. Context checks prevent hosted or pinned features from being
// reclassified as user or harness traffic.
func NewEligibleRequestForUse(check FeatureCheck, useClass UseClass, model, profile, privacy, harness string, budget int64) (Request, error) {
	if err := validateFeatureCheck(check); err != nil {
		return Request{}, err
	}
	if !validUseClass(useClass) {
		return Request{}, fmt.Errorf("invalid caller use class %q", useClass)
	}
	if err := validateCallerUse(check, useClass, harness); err != nil {
		return Request{}, err
	}
	return Request{
		featureID: strings.TrimSpace(check.featureID), endpoint: check.endpoint,
		requiredCapabilities: append([]string(nil), check.requiredCapabilities...),
		useClass:             useClass, model: model, profile: profile, privacy: privacy, harness: harness, budget: budget,
	}, nil
}

func validateFeatureCheck(check FeatureCheck) error {
	if !check.checked || strings.TrimSpace(check.featureID) == "" {
		return errors.New("feature id is required")
	}
	if check.userTurns != 1 || check.toolCount != 0 || !check.schemaValid {
		return errors.New("feature contract check failed")
	}
	if check.maxInputBytes > 0 && check.inputBytes > check.maxInputBytes {
		return errors.New("feature input exceeds contract cap")
	}
	if check.maxOutputBytes > 0 && check.outputBytes > check.maxOutputBytes {
		return errors.New("feature output exceeds contract cap")
	}
	return nil
}

func validateCallerUse(check FeatureCheck, useClass UseClass, harness string) error {
	if (check.pinned || check.hostedService) && useClass != UseBoundedFeature {
		return errors.New("pinned or hosted features must use bounded_feature")
	}
	if useClass == UseHarnessAccess && strings.TrimSpace(harness) == "" {
		return errors.New("harness_access requires a server-resolved harness")
	}
	if useClass != UseHarnessAccess && strings.TrimSpace(harness) != "" {
		return errors.New("harness is only valid for harness_access")
	}
	return nil
}

type Account struct {
	Plan              string
	ConnectedAccounts []string
	BalanceStanding   bool
}

type RuleRecord struct {
	Supplier         string
	Terms            ProviderTerms
	Models           map[string]ModelRule
	decisionVerified bool
}

type ModelRule struct {
	LicenceCommercialUse *bool
	TermsOverride        *TermsOverride
}

type EligibilityResult struct {
	Routes   []Route
	Refusals []Refusal
}

type Refusal struct {
	RouteID string `json:"route_id"`
	Reason  string `json:"reason"`
}

// Eligible is the one server-only policy function used by both picker and
// execution. It does not approve records, mutate trust, or derive facts from a
// client-supplied use class.
func Eligible(request Request, account Account, records []RuleRecord, agreements []Agreement, routes []Route, now time.Time) EligibilityResult {
	out := EligibilityResult{Routes: make([]Route, 0), Refusals: make([]Refusal, 0)}
	if request.featureID == "" || request.useClass == "" {
		return denyAll(routes, "invalid_contract_request")
	}
	if !account.BalanceStanding && request.useClass != UseOwnAccount {
		return denyAll(routes, "account_not_in_standing")
	}
	bySupplier := make(map[string]RuleRecord, len(records))
	for _, record := range records {
		bySupplier[strings.ToLower(strings.TrimSpace(record.Supplier))] = record
	}
	for _, route := range routes {
		refusal := eligibilityRefusal(request, account, route, bySupplier, agreements, now)
		if refusal != "" {
			out.Refusals = append(out.Refusals, Refusal{RouteID: route.ID, Reason: refusal})
			continue
		}
		out.Routes = append(out.Routes, route)
	}
	sort.SliceStable(out.Routes, func(i, j int) bool {
		if out.Routes[i].QuotedCredits != out.Routes[j].QuotedCredits {
			return out.Routes[i].QuotedCredits < out.Routes[j].QuotedCredits
		}
		if out.Routes[i].Profile != out.Routes[j].Profile {
			return out.Routes[i].Profile < out.Routes[j].Profile
		}
		return out.Routes[i].ID < out.Routes[j].ID
	})
	return out
}

// PickRoutes and AuthorizeExecution intentionally delegate to Eligible. This
// makes stale picker state unable to grant execution an otherwise denied route.
func PickRoutes(req Request, acct Account, records []RuleRecord, agreements []Agreement, routes []Route, now time.Time) EligibilityResult {
	return Eligible(req, acct, records, agreements, routes, now)
}

func AuthorizeExecution(req Request, acct Account, records []RuleRecord, agreements []Agreement, routes []Route, now time.Time, selectedRouteID string) (Route, error) {
	decision := Eligible(req, acct, records, agreements, routes, now)
	for _, route := range decision.Routes {
		if route.ID == selectedRouteID {
			return route, nil
		}
	}
	return Route{}, fmt.Errorf("selected route is not eligible: %s", refusalFor(decision.Refusals, selectedRouteID))
}

func eligibilityRefusal(req Request, acct Account, route Route, records map[string]RuleRecord, agreements []Agreement, now time.Time) string {
	if reason := validateRouteIdentity(req, acct, route); reason != "" {
		return reason
	}
	record, ok := records[strings.ToLower(strings.TrimSpace(route.Supplier))]
	if !ok {
		return "no_record"
	}
	if reason := validateRecordWindow(record, now); reason != "" {
		return reason
	}
	planRule := findPlanRule(record.Terms.PlanRules, route.Plan)
	if planRule == nil {
		return "no_plan_rule"
	}
	if reason := validatePlanRoute(req, record, *planRule, route, agreements, now); reason != "" {
		return reason
	}
	if req.privacy == "private" && record.Terms.Downstream.DataCollection != "deny" {
		return "privacy_mismatch"
	}
	if reason := validateRouteQuote(req, route); reason != "" {
		return reason
	}
	return ""
}

func validateRouteIdentity(req Request, acct Account, route Route) string {
	if strings.TrimSpace(route.Model) == "" || strings.TrimSpace(route.Supplier) == "" || strings.TrimSpace(route.Plan) == "" {
		return "invalid_route"
	}
	if req.useClass == UseOwnAccount && (strings.TrimSpace(route.AccountID) == "" || !containsString(acct.ConnectedAccounts, route.AccountID)) {
		return "own_account_not_connected"
	}
	if req.model != "" && req.model != route.Model {
		return "model_mismatch"
	}
	if req.useClass == UseHarnessAccess && (route.HarnessSource == nil || route.Harness != req.harness) {
		return "harness_billing_source_denied"
	}
	return validateRouteRequirements(req, route)
}

func validateRouteRequirements(req Request, route Route) string {
	if route.Endpoint != req.endpoint {
		return "endpoint_mismatch"
	}
	for _, capability := range req.requiredCapabilities {
		if !containsString(route.Capabilities, capability) {
			return "capability_mismatch"
		}
	}
	return ""
}

func validateRecordWindow(record RuleRecord, now time.Time) string {
	terms := record.Terms
	if terms.Review.Status != ReviewApproved || strings.TrimSpace(terms.Review.DecisionRef) == "" || !record.decisionVerified {
		return "unapproved"
	}
	effective, err := time.Parse("2006-01-02", terms.EffectiveFrom)
	if err != nil || now.Before(effective) {
		return "not_effective"
	}
	if reason := validateRecordDeadline(terms.ExpiresAt, now, "expired"); reason != "" {
		return reason
	}
	return validateRecordDeadline(terms.NextReviewAt, now, "review_due")
}

func validateRecordDeadline(date string, now time.Time, reason string) string {
	if date == "" {
		return ""
	}
	deadline, err := time.Parse("2006-01-02", date)
	if err != nil || !now.Before(deadline.Add(24*time.Hour)) {
		return reason
	}
	return ""
}

func findPlanRule(rules []PlanRule, plan string) *PlanRule {
	for i := range rules {
		if rules[i].Plan == plan {
			return &rules[i]
		}
	}
	return nil
}

func validatePlanRoute(req Request, record RuleRecord, planRule PlanRule, route Route, agreements []Agreement, now time.Time) string {
	if planRule.AccessTier == AccessTierBarred || planRule.AccessTier == AccessTierUnknown {
		return "access_tier_" + string(planRule.AccessTier)
	}
	modelRule := record.Models[route.Model]
	if reason := validateUsePermission(req, planRule, modelRule, route, agreements, now); reason != "" {
		return reason
	}
	if reason := validateModelLicence(req, modelRule); reason != "" {
		return reason
	}
	if req.useClass == UseHarnessAccess {
		if reason := validateHarnessRoute(route, agreements, now); reason != "" {
			return reason
		}
	}
	if route.Profile != "" && req.profile != "" && route.Profile != req.profile {
		return "profile_mismatch"
	}
	return ""
}

func validateUsePermission(req Request, rule PlanRule, modelRule ModelRule, route Route, agreements []Agreement, now time.Time) string {
	permission := rule.UseClasses[req.useClass]
	if modelRule.TermsOverride != nil {
		if override, exists := modelRule.TermsOverride.UseClasses[req.useClass]; exists {
			if override == PermissionForbidden {
				return "model_terms_forbidden"
			}
			if permissionRank(override) < permissionRank(permission) {
				permission = override
			}
		}
	}
	if permission == PermissionAllowed || hasAgreement(agreements, route, req.useClass, now) {
		return ""
	}
	return "permission_" + string(permission)
}

func validateModelLicence(req Request, modelRule ModelRule) string {
	if modelRule.LicenceCommercialUse == nil || *modelRule.LicenceCommercialUse || req.useClass == UseOwnAccount {
		return ""
	}
	return "model_licence_denies"
}

func validateHarnessRoute(route Route, agreements []Agreement, now time.Time) string {
	if route.HarnessSource == nil || !route.HarnessSource.decisionVerified || route.HarnessSource.Review.Status != ReviewApproved || route.HarnessSource.LauncherConfigAllowed == nil || !*route.HarnessSource.LauncherConfigAllowed || route.HarnessSource.CustomerConfigAllowed != PermissionAllowed {
		return "harness_billing_source_denied"
	}
	if !trustAllows(agreements, route, now) {
		return "customer_code_not_trusted"
	}
	return ""
}

func validateRouteQuote(req Request, route Route) string {
	if strings.TrimSpace(route.QuoteRef) == "" {
		return "missing_quote"
	}
	if req.budget > 0 && route.QuotedCredits > req.budget {
		return "budget_exceeded"
	}
	return ""
}

func hasAgreement(agreements []Agreement, route Route, useClass UseClass, now time.Time) bool {
	for _, a := range agreements {
		if agreementGrantsRoute(a, route, useClass, now) {
			return true
		}
	}
	return false
}

func agreementGrantsRoute(agreement Agreement, route Route, useClass UseClass, now time.Time) bool {
	if !validAgreementWindow(agreement, route, now) {
		return false
	}
	if !containsUseClass(agreement.UseClasses, useClass) {
		return false
	}
	return len(agreement.Models) > 0 && containsString(agreement.Models, route.Model)
}

func validAgreementWindow(agreement Agreement, route Route, now time.Time) bool {
	if !strings.EqualFold(agreement.Supplier, route.Supplier) || agreement.DecisionRef == "" || !agreement.decisionVerified {
		return false
	}
	if !agreement.EffectiveAt.IsZero() && now.Before(agreement.EffectiveAt) {
		return false
	}
	return agreement.ExpiresAt.IsZero() || now.Before(agreement.ExpiresAt)
}

func trustAllows(agreements []Agreement, route Route, now time.Time) bool {
	for _, a := range agreements {
		if trustAgreementAllows(a, route, now) {
			return true
		}
	}
	return false
}

func trustAgreementAllows(agreement Agreement, route Route, now time.Time) bool {
	return agreementGrantsRoute(agreement, route, UseHarnessAccess, now) && agreement.Trust.CustomerCodeAllowed != nil && *agreement.Trust.CustomerCodeAllowed && agreement.Trust.DecisionRef != ""
}

func permissionRank(p Permission) int {
	switch p {
	case PermissionAllowed:
		return 5
	case PermissionNeedsWrittenApproval:
		return 4
	case PermissionSilent:
		return 3
	case PermissionUnverified:
		return 2
	case PermissionForbidden:
		return 1
	default:
		return 0
	}
}

func denyAll(routes []Route, reason string) EligibilityResult {
	out := EligibilityResult{Routes: []Route{}, Refusals: make([]Refusal, 0, len(routes))}
	for _, route := range routes {
		out.Refusals = append(out.Refusals, Refusal{RouteID: route.ID, Reason: reason})
	}
	return out
}

func refusalFor(refusals []Refusal, routeID string) string {
	for _, refusal := range refusals {
		if refusal.RouteID == routeID {
			return refusal.Reason
		}
	}
	return "no_eligible_route"
}

func containsUseClass(values []UseClass, target UseClass) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
