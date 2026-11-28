package routing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const maxReplayParallel = 16

type (
	replayExecutor       func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error)
	replaySourceVerifier func(context.Context, ReplayCase) error
)

// ReplayCallerCases dispatches a fixed caller set with bounded parallelism.
// Results stay in input order even when calls complete out of order.
func ReplayCallerCases(ctx context.Context, cases []ReplayCase, parallel int, verify replaySourceVerifier, execute replayExecutor) ([]ReplayReceipt, error) {
	if err := validateExecuteReplayInputs(ctx, cases, parallel, verify, execute); err != nil {
		return nil, err
	}
	results := make([]ReplayReceipt, len(cases))
	type replayJob struct {
		index int
		item  ReplayCase
	}
	jobs := make(chan replayJob)
	var wg sync.WaitGroup
	workers := parallel
	if workers > len(cases) {
		workers = len(cases)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results[job.index] = executeReplayCase(ctx, job.item, execute)
			}
		}()
	}
	dispatchErr := error(nil)
	for i, c := range cases {
		select {
		case jobs <- replayJob{index: i, item: c}:
		case <-ctx.Done():
			dispatchErr = ctx.Err()
			goto dispatched
		}
	}
dispatched:
	close(jobs)
	wg.Wait()
	if dispatchErr != nil {
		return nil, dispatchErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func validateExecuteReplayInputs(ctx context.Context, cases []ReplayCase, parallel int, verify replaySourceVerifier, execute replayExecutor) error {
	if len(cases) == 0 {
		return fmt.Errorf("no replay cases")
	}
	if len(cases) > maxReplayCases {
		return fmt.Errorf("replay has %d cases; maximum is %d", len(cases), maxReplayCases)
	}
	if parallel < 1 || parallel > maxReplayParallel {
		return fmt.Errorf("parallelism must be between 1 and %d", maxReplayParallel)
	}
	if execute == nil {
		return fmt.Errorf("replay executor is required")
	}
	if verify == nil {
		return fmt.Errorf("Agent Manager source verifier is required")
	}
	for _, c := range cases {
		if err := validateExecuteReplayCase(ctx, c, verify); err != nil {
			return err
		}
	}
	return nil
}

func validateExecuteReplayCase(ctx context.Context, c ReplayCase, verify replaySourceVerifier) error {
	if replayCaseProfile(c) != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		return fmt.Errorf("execute replay case %q must use local-only; remote inference is disabled", c.CaseID)
	}
	if strings.TrimSpace(c.PrivacyProvenance) == "" {
		return fmt.Errorf("execute replay case %q requires explicit privacy_provenance", c.CaseID)
	}
	if c.RequiredVRAMBytes <= 0 || c.RequiredVRAMBytes > 16<<30 {
		return fmt.Errorf("replay case %q requires an explicit VRAM reservation between 1 byte and 16 GiB", c.CaseID)
	}
	if !strings.HasPrefix(c.CapacityEstimateEvidenceURI, "capacity-footprint://") {
		return fmt.Errorf("replay case %q requires a capacity-footprint evidence URI", c.CaseID)
	}
	if err := verify(ctx, c); err != nil {
		return fmt.Errorf("verify source for replay case %q: %w", c.CaseID, err)
	}
	return nil
}

func executeReplayCase(ctx context.Context, c ReplayCase, execute replayExecutor) ReplayReceipt {
	c.TaskRole = replayTaskRole(c)
	requestID := replaySourceRequestID(c)
	receipt := ReplayReceipt{CaseID: c.CaseID, Mode: "execute", SourceRunID: c.SourceRunID, SourceEventID: c.SourceEventID, SourceSequence: c.SourceSequence, ConversationID: c.ConversationID, PrivacyProvenance: c.PrivacyProvenance, Scenario: c.Scenario, Role: c.Role, TaskRole: c.TaskRole, Profile: profileLabel(replayCaseProfile(c)), LatencyKind: "gateway_execution", CapacityRequestedBytes: c.RequiredVRAMBytes}
	req := &routingv1.ExecuteRouteRequest{Request: &sharedv1.GatewayRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, Role: c.Role, TaskRole: c.TaskRole,
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: replayPrivacy(c.PrivacyClass),
		PrivacyProvenance: c.PrivacyProvenance, Operation: c.Operation, Scenario: c.Scenario,
		RequestId: requestID, TimeoutMs: c.TimeoutMS, RequiredVramBytes: &c.RequiredVRAMBytes,
	}, InputText: c.InputText}
	start := time.Now()
	response, err := execute(ctx, req)
	receipt.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		return replayFailure(receipt, "execute_route_failed")
	}
	if response == nil || response.GetEvidence() == nil {
		return replayFailure(receipt, "gateway returned no route evidence")
	}
	if err := applyReplayEvidence(&receipt, c, requestID, response); err != nil {
		return replayFailure(receipt, err.Error())
	}
	return receipt
}

func replayFailure(receipt ReplayReceipt, message string) ReplayReceipt {
	receipt.Status, receipt.Error = "error", message
	return receipt
}

func applyReplayEvidence(receipt *ReplayReceipt, c ReplayCase, requestID string, response *routingv1.ExecuteRouteResponse) error {
	ev := response.GetEvidence()
	if err := validateReplayEvidenceIdentity(ev, c); err != nil {
		return err
	}
	receipt.GatewayEventID, receipt.GatewayRequestID = ev.GetEventId(), ev.GetRequestId()
	receipt.CapacityClaimID = ev.GetCapacityClaimId()
	receipt.CapacityRequiredBytes, receipt.CapacityGrantedBytes = ev.GetCapacityRequiredBytes(), ev.GetCapacityGrantedBytes()
	receipt.CapacityVerdict, receipt.CapacityEstimateEvidenceURI = ev.GetCapacityVerdict(), c.CapacityEstimateEvidenceURI
	if err := validateReplayReceiptAdmission(*receipt, c, requestID); err != nil {
		return err
	}
	receipt.GatewayRoleEvidenceURI = "gateway-ledger://events/" + receipt.GatewayEventID
	receipt.Status, receipt.SelectedProvider, receipt.SelectedModel = ev.GetStatus(), ev.GetSelectedProvider(), ev.GetSelectedModel()
	receipt.TaskRole, receipt.SelectedResourceRole, receipt.Locality = ev.GetTaskRole(), ev.GetSelectedResourceRole(), ev.GetSelectedLocality()
	if err := validateSelectedResourceRole(*receipt); err != nil {
		return err
	}
	if !response.GetValid() && receipt.Status == "" {
		receipt.Status = "invalid"
	}
	return nil
}

func validateReplayEvidenceIdentity(ev interface {
	GetScenario() string
	GetTaskRole() string
}, c ReplayCase,
) error {
	if ev.GetScenario() != c.Scenario {
		return fmt.Errorf("gateway route receipt consumer scenario does not match replay case")
	}
	if ev.GetTaskRole() != replayTaskRole(c) {
		return fmt.Errorf("gateway route receipt task role does not match replay request")
	}
	return nil
}

func validateReplayReceiptAdmission(receipt ReplayReceipt, c ReplayCase, requestID string) error {
	if receipt.GatewayEventID == "" || receipt.GatewayRequestID == "" {
		return fmt.Errorf("gateway route receipt omitted event or request identity")
	}
	if receipt.GatewayRequestID != requestID {
		return fmt.Errorf("gateway route receipt request identity does not match Agent Manager source")
	}
	if receipt.CapacityVerdict != "fit" || receipt.CapacityClaimID == "" || receipt.CapacityRequiredBytes != c.RequiredVRAMBytes || receipt.CapacityGrantedBytes < c.RequiredVRAMBytes {
		return fmt.Errorf("gateway route receipt omitted verified capacity admission")
	}
	return nil
}

func validateSelectedResourceRole(receipt ReplayReceipt) error {
	if receipt.Status == "succeeded" && receipt.SelectedResourceRole == "" {
		return fmt.Errorf("successful gateway route receipt omitted selected resource role")
	}
	return nil
}

// replaySourceRequestID carries the exact Agent Manager source identity into
// the durable Gateway ledger. Its final digest binds task role and consumer
// without storing case or prompt content in Gateway evidence.
func replaySourceRequestID(c ReplayCase) string {
	requestID, _ := searcheval.AgentManagerReplayRequestID(c.SourceRunID, c.SourceEventID, c.SourceSequence, replayTaskRole(c), c.Scenario)
	return requestID
}

func replayTaskRole(c ReplayCase) string {
	if role := strings.TrimSpace(c.TaskRole); role != "" {
		return role
	}
	return strings.TrimSpace(c.Role)
}

type replayPreviewExecutor func(context.Context, *routingv1.PreviewRouteRequest) (*routingv1.PreviewRouteResponse, error)

// PreviewCallerCases verifies genuine Agent Manager sources and replays their
// role/profile/privacy routing decisions without calling a model provider.
func PreviewCallerCases(ctx context.Context, cases []ReplayCase, parallel int, verify replaySourceVerifier, preview replayPreviewExecutor) ([]ReplayReceipt, error) {
	if len(cases) == 0 || len(cases) > maxReplayCases {
		return nil, fmt.Errorf("preview replay must contain 1-%d cases", maxReplayCases)
	}
	if parallel < 1 || parallel > maxReplayParallel {
		return nil, fmt.Errorf("parallelism must be between 1 and %d", maxReplayParallel)
	}
	if preview == nil || verify == nil {
		return nil, fmt.Errorf("preview executor and Agent Manager source verifier are required")
	}
	if err := verifyPreviewSources(ctx, cases, verify); err != nil {
		return nil, err
	}
	return dispatchPreviewCases(ctx, cases, parallel, preview)
}

func dispatchPreviewCases(ctx context.Context, cases []ReplayCase, parallel int, preview replayPreviewExecutor) ([]ReplayReceipt, error) {
	results := make([]ReplayReceipt, len(cases))
	type job struct {
		index int
		item  ReplayCase
	}
	jobs := make(chan job)
	workers := min(parallel, len(cases))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range jobs {
				results[work.index] = previewReplayCase(ctx, work.item, preview)
			}
		}()
	}
	var dispatchErr error
	for i, c := range cases {
		select {
		case jobs <- job{index: i, item: c}:
		case <-ctx.Done():
			dispatchErr = ctx.Err()
			goto previewDispatched
		}
	}
previewDispatched:
	close(jobs)
	wg.Wait()
	if dispatchErr != nil {
		return nil, dispatchErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func verifyPreviewSources(ctx context.Context, cases []ReplayCase, verify replaySourceVerifier) error {
	for _, c := range cases {
		if strings.TrimSpace(c.PrivacyProvenance) == "" {
			return fmt.Errorf("preview replay case %q requires explicit privacy_provenance", c.CaseID)
		}
		if err := verify(ctx, c); err != nil {
			return fmt.Errorf("verify source for replay case %q: %w", c.CaseID, err)
		}
	}
	return nil
}

func previewReplayCase(ctx context.Context, c ReplayCase, preview replayPreviewExecutor) ReplayReceipt {
	c.TaskRole = replayTaskRole(c)
	receipt := ReplayReceipt{CaseID: c.CaseID, Mode: "preview", SourceRunID: c.SourceRunID, SourceEventID: c.SourceEventID, SourceSequence: c.SourceSequence, ConversationID: c.ConversationID, PrivacyProvenance: c.PrivacyProvenance, Scenario: c.Scenario, Role: c.Role, TaskRole: c.TaskRole, Profile: profileLabel(replayCaseProfile(c)), LatencyKind: "route_preview"}
	request := previewRequest(c)
	start := time.Now()
	response, err := preview(ctx, request)
	receipt.LatencyMS = time.Since(start).Milliseconds()
	if err != nil || response == nil {
		return failedPreviewReceipt(receipt)
	}
	valid := response.GetValid()
	receipt.Valid = &valid
	receipt.Status = "invalid"
	if response.GetValid() {
		receipt.Status = "valid"
	}
	receipt.SelectedProvider = response.GetSelectedProvider()
	receipt.RoutePlanID = response.GetRoutePlanId()
	receipt.CandidateEligibility = previewCandidateEligibility(response.GetCandidates())
	return receipt
}

func previewRequest(c ReplayCase) *routingv1.PreviewRouteRequest {
	return &routingv1.PreviewRouteRequest{Request: &sharedv1.GatewayRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, Role: c.Role, TaskRole: c.TaskRole,
		Profile: replayCaseProfile(c), PrivacyClass: replayPrivacy(c.PrivacyClass),
		PrivacyProvenance: c.PrivacyProvenance, Operation: c.Operation, Scenario: c.Scenario, TimeoutMs: c.TimeoutMS,
	}}
}

func failedPreviewReceipt(receipt ReplayReceipt) ReplayReceipt {
	receipt.Status, receipt.Error = "error", "preview_route_failed"
	return receipt
}

func previewCandidateEligibility(candidates []*routingv1.RouteCandidate) []ReplayCandidateEligibility {
	eligibility := make([]ReplayCandidateEligibility, 0, len(candidates))
	for _, candidate := range candidates {
		eligibility = append(eligibility, ReplayCandidateEligibility{Provider: candidate.GetProvider(), Role: candidate.GetRole(), Locality: candidate.GetLocality(), Selected: candidate.GetSelected(), Fallback: candidate.GetFallbackEligible(), RejectionCode: candidate.GetRejectionReason(), CapacityVerdict: candidate.GetCapacityVerdict()})
	}
	return eligibility
}

func replayProfile(v string) sharedv1.Profile {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(v), "_", "-")) {
	case "", "local-first":
		return sharedv1.Profile_PROFILE_LOCAL_FIRST
	case "local-only":
		return sharedv1.Profile_PROFILE_LOCAL_ONLY
	case "remote-only":
		return sharedv1.Profile_PROFILE_REMOTE_ONLY
	case "quality-first":
		return sharedv1.Profile_PROFILE_QUALITY_FIRST
	case "cheap-first":
		return sharedv1.Profile_PROFILE_CHEAP_FIRST
	case "privacy-sensitive":
		return sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE
	default:
		return sharedv1.Profile_PROFILE_UNSPECIFIED
	}
}

func replayCaseProfile(c ReplayCase) sharedv1.Profile {
	if strings.TrimSpace(c.Profile) == "" {
		return sharedv1.Profile_PROFILE_LOCAL_ONLY
	}
	return replayProfile(c.Profile)
}

func profileLabel(profile sharedv1.Profile) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(profile.String(), "PROFILE_"), "_", "-"))
}

func overrideReplayProfile(cases []ReplayCase, profile, mode string) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil
	}
	resolved := replayProfile(profile)
	if resolved != sharedv1.Profile_PROFILE_LOCAL_ONLY && resolved != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		return fmt.Errorf("replay profile override must be local-only or remote-only")
	}
	if mode == "execute" && resolved == sharedv1.Profile_PROFILE_REMOTE_ONLY {
		return fmt.Errorf("remote-only replay is preview-only; remote inference is disabled")
	}
	value := "local-only"
	if resolved == sharedv1.Profile_PROFILE_REMOTE_ONLY {
		value = "remote-only"
	}
	for i := range cases {
		cases[i].Profile = value
	}
	return nil
}

func replayPrivacy(v string) sharedv1.PrivacyClass {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "internal":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL
	case "public":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC
	case "confidential":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	case "secret":
		return sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
	default:
		return sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED
	}
}

func executeReplayCall(ctx context.Context, client interface {
	ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
}, req *routingv1.ExecuteRouteRequest,
) (*routingv1.ExecuteRouteResponse, error) {
	response, err := client.ExecuteRoute(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("gateway returned an empty response")
	}
	return response.Msg, nil
}

func previewReplayCall(ctx context.Context, client interface {
	PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error)
}, req *routingv1.PreviewRouteRequest,
) (*routingv1.PreviewRouteResponse, error) {
	response, err := client.PreviewRoute(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	return response.Msg, nil
}
