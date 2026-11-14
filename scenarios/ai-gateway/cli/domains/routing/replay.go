package routing

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const (
	maxReplayInputBytes = 10 << 20
	maxReplayCases      = 1000 // accommodates overlapping role/task-role/consumer floors while keeping replay bounded.
	maxReplayParallel   = 16
)

var replayUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ReplayCase is one real caller request selected for a bounded mixed-load
// replay. The source references are mandatory so a fixture cannot silently
// turn generated prompts into acceptance evidence.
type ReplayCase struct {
	CaseID                      string `json:"case_id"`
	SourceRunID                 string `json:"source_run_id"`
	SourceEventID               string `json:"source_event_id"`
	SourceSequence              int64  `json:"source_sequence"`
	ConversationID              string `json:"conversation_id,omitempty"`
	Role                        string `json:"role"`
	TaskRole                    string `json:"task_role"`
	Profile                     string `json:"profile"`
	PrivacyClass                string `json:"privacy_class"`
	PrivacyProvenance           string `json:"privacy_provenance"`
	Operation                   string `json:"operation"`
	Scenario                    string `json:"scenario"`
	TimeoutMS                   int32  `json:"timeout_ms,omitempty"`
	RequiredVRAMBytes           int64  `json:"required_vram_bytes,omitempty"`
	CapacityEstimateEvidenceURI string `json:"capacity_estimate_evidence_uri,omitempty"`
	InputText                   string `json:"input_text"`
}

// ReplayReceipt intentionally excludes prompts and model output. It retains
// source provenance, route evidence and timing for the comparison record.
type ReplayReceipt struct {
	CaseID                      string                       `json:"case_id"`
	Mode                        string                       `json:"mode"`
	SourceRunID                 string                       `json:"source_run_id"`
	SourceEventID               string                       `json:"source_event_id"`
	SourceSequence              int64                        `json:"source_sequence"`
	ConversationID              string                       `json:"conversation_id,omitempty"`
	PrivacyProvenance           string                       `json:"privacy_provenance"`
	Scenario                    string                       `json:"scenario"`
	Role                        string                       `json:"role"`
	TaskRole                    string                       `json:"task_role"`
	SelectedResourceRole        string                       `json:"selected_resource_role,omitempty"`
	Status                      string                       `json:"status"`
	SelectedProvider            string                       `json:"selected_provider,omitempty"`
	GatewayEventID              string                       `json:"gateway_event_id,omitempty"`
	GatewayRequestID            string                       `json:"gateway_request_id,omitempty"`
	GatewayRoleEvidenceURI      string                       `json:"gateway_role_evidence_uri,omitempty"`
	CapacityClaimID             string                       `json:"capacity_claim_id,omitempty"`
	CapacityRequestedBytes      int64                        `json:"capacity_requested_bytes,omitempty"`
	CapacityRequiredBytes       int64                        `json:"capacity_required_bytes,omitempty"`
	CapacityGrantedBytes        int64                        `json:"capacity_granted_bytes,omitempty"`
	CapacityVerdict             string                       `json:"capacity_verdict,omitempty"`
	CapacityEstimateEvidenceURI string                       `json:"capacity_estimate_evidence_uri,omitempty"`
	RoutePlanID                 string                       `json:"route_plan_id,omitempty"`
	SelectedModel               string                       `json:"selected_model,omitempty"`
	Locality                    string                       `json:"locality,omitempty"`
	Profile                     string                       `json:"profile"`
	CandidateEligibility        []ReplayCandidateEligibility `json:"candidate_eligibility,omitempty"`
	LatencyMS                   int64                        `json:"latency_ms"`
	LatencyKind                 string                       `json:"latency_kind"`
	Valid                       *bool                        `json:"valid,omitempty"`
	Error                       string                       `json:"error,omitempty"`
}

// ReplayCandidateEligibility is a redacted snapshot of the generic router's
// candidate decision. It keeps role/locality and stable rejection codes while
// omitting input, prompts, and provider-specific metadata.
type ReplayCandidateEligibility struct {
	Provider        string `json:"provider"`
	Role            string `json:"role"`
	Locality        string `json:"locality"`
	Selected        bool   `json:"selected"`
	Fallback        bool   `json:"fallback_eligible"`
	RejectionCode   string `json:"rejection_code,omitempty"`
	CapacityVerdict string `json:"capacity_verdict,omitempty"`
}

// ReadReplayCases reads a bounded JSONL file. It rejects missing provenance,
// duplicate IDs, empty caller inputs and oversized files before any inference
// can be dispatched.
func ReadReplayCases(r io.Reader) ([]ReplayCase, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReplayInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read replay input: %w", err)
	}
	if len(data) > maxReplayInputBytes {
		return nil, fmt.Errorf("replay input exceeds %d bytes", maxReplayInputBytes)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	seen := map[string]struct{}{}
	cases := make([]ReplayCase, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(cases) >= maxReplayCases {
			return nil, fmt.Errorf("replay input exceeds %d cases", maxReplayCases)
		}
		var c ReplayCase
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			return nil, fmt.Errorf("replay case line %d: %w", len(cases)+1, err)
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			if err == nil {
				return nil, fmt.Errorf("replay case line %d contains more than one JSON value", len(cases)+1)
			}
			return nil, fmt.Errorf("replay case line %d: %w", len(cases)+1, err)
		}
		c.CaseID = strings.TrimSpace(c.CaseID)
		c.SourceRunID = strings.TrimSpace(c.SourceRunID)
		c.SourceEventID = strings.TrimSpace(c.SourceEventID)
		c.Role = strings.TrimSpace(c.Role)
		c.TaskRole = strings.TrimSpace(c.TaskRole)
		if c.TaskRole == "" {
			c.TaskRole = c.Role
		}
		if c.CaseID == "" || c.SourceRunID == "" || c.SourceEventID == "" || c.SourceSequence < 0 || c.Role == "" || strings.TrimSpace(c.Scenario) == "" || strings.TrimSpace(c.InputText) == "" {
			return nil, fmt.Errorf("replay case line %d requires case_id, source_run_id, source_event_id, nonnegative source_sequence, role, scenario, input_text and privacy_provenance", len(cases)+1)
		}
		if strings.TrimSpace(c.PrivacyProvenance) == "" {
			return nil, fmt.Errorf("replay case %q requires explicit privacy_provenance", c.CaseID)
		}
		if !replayUUIDPattern.MatchString(c.SourceRunID) {
			return nil, fmt.Errorf("replay case %q source_run_id must be a UUID", c.CaseID)
		}
		if !replayUUIDPattern.MatchString(c.SourceEventID) {
			return nil, fmt.Errorf("replay case %q source_event_id must be a UUID", c.CaseID)
		}
		if len(c.InputText) > 64<<10 {
			return nil, fmt.Errorf("replay case %q input_text exceeds 64 KiB", c.CaseID)
		}
		if c.Profile == "" {
			c.Profile = "local-only"
		}
		profile := replayProfile(c.Profile)
		if profile != sharedv1.Profile_PROFILE_LOCAL_ONLY && profile != sharedv1.Profile_PROFILE_REMOTE_ONLY {
			return nil, fmt.Errorf("replay case %q profile must be local-only or remote-only", c.CaseID)
		}
		if replayPrivacy(c.PrivacyClass) == sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
			return nil, fmt.Errorf("replay case %q has invalid privacy_class %q", c.CaseID, c.PrivacyClass)
		}
		if c.TimeoutMS < 0 || c.TimeoutMS > 300000 {
			return nil, fmt.Errorf("replay case %q timeout_ms must be between 0 and 300000", c.CaseID)
		}
		if _, exists := seen[c.CaseID]; exists {
			return nil, fmt.Errorf("duplicate replay case_id %q", c.CaseID)
		}
		seen[c.CaseID] = struct{}{}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read replay input: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("replay input contains no cases")
	}
	return cases, nil
}

type (
	replayExecutor       func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error)
	replaySourceVerifier func(context.Context, ReplayCase) error
)

// ReplayCallerCases dispatches a fixed caller set with bounded parallelism.
// Results stay in input order even when calls complete out of order.
func ReplayCallerCases(ctx context.Context, cases []ReplayCase, parallel int, verify replaySourceVerifier, execute replayExecutor) ([]ReplayReceipt, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("no replay cases")
	}
	if len(cases) > maxReplayCases {
		return nil, fmt.Errorf("replay has %d cases; maximum is %d", len(cases), maxReplayCases)
	}
	if parallel < 1 || parallel > maxReplayParallel {
		return nil, fmt.Errorf("parallelism must be between 1 and %d", maxReplayParallel)
	}
	if execute == nil {
		return nil, fmt.Errorf("replay executor is required")
	}
	if verify == nil {
		return nil, fmt.Errorf("Agent Manager source verifier is required")
	}
	for _, c := range cases {
		if replayCaseProfile(c) != sharedv1.Profile_PROFILE_LOCAL_ONLY {
			return nil, fmt.Errorf("execute replay case %q must use local-only; remote inference is disabled", c.CaseID)
		}
		if strings.TrimSpace(c.PrivacyProvenance) == "" {
			return nil, fmt.Errorf("execute replay case %q requires explicit privacy_provenance", c.CaseID)
		}
		if c.RequiredVRAMBytes <= 0 || c.RequiredVRAMBytes > 16<<30 {
			return nil, fmt.Errorf("replay case %q requires an explicit VRAM reservation between 1 byte and 16 GiB", c.CaseID)
		}
		if !strings.HasPrefix(c.CapacityEstimateEvidenceURI, "capacity-footprint://") {
			return nil, fmt.Errorf("replay case %q requires a capacity-footprint evidence URI", c.CaseID)
		}
		if err := verify(ctx, c); err != nil {
			return nil, fmt.Errorf("verify source for replay case %q: %w", c.CaseID, err)
		}
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
				c := job.item
				c.TaskRole = replayTaskRole(c)
				requestID := replaySourceRequestID(c)
				receipt := ReplayReceipt{CaseID: c.CaseID, Mode: "execute", SourceRunID: c.SourceRunID, SourceEventID: c.SourceEventID, SourceSequence: c.SourceSequence, ConversationID: c.ConversationID, PrivacyProvenance: c.PrivacyProvenance, Scenario: c.Scenario, Role: c.Role, TaskRole: c.TaskRole, Profile: profileLabel(replayCaseProfile(c)), LatencyKind: "gateway_execution"}
				receipt.CapacityRequestedBytes = c.RequiredVRAMBytes
				req := &routingv1.ExecuteRouteRequest{Request: &sharedv1.GatewayRequest{
					Kind:              sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
					Role:              c.Role,
					TaskRole:          c.TaskRole,
					Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
					PrivacyClass:      replayPrivacy(c.PrivacyClass),
					PrivacyProvenance: c.PrivacyProvenance,
					Operation:         c.Operation,
					Scenario:          c.Scenario,
					RequestId:         requestID,
					TimeoutMs:         c.TimeoutMS,
					RequiredVramBytes: &c.RequiredVRAMBytes,
				}, InputText: c.InputText}
				start := time.Now()
				response, err := execute(ctx, req)
				receipt.LatencyMS = time.Since(start).Milliseconds()
				if err != nil {
					receipt.Status = "error"
					receipt.Error = "execute_route_failed"
					results[job.index] = receipt
					continue
				}
				if response == nil || response.GetEvidence() == nil {
					receipt.Status = "error"
					receipt.Error = "gateway returned no route evidence"
					results[job.index] = receipt
					continue
				}
				ev := response.GetEvidence()
				if ev.GetScenario() != c.Scenario {
					receipt.Status = "error"
					receipt.Error = "gateway route receipt consumer scenario does not match replay case"
					results[job.index] = receipt
					continue
				}
				if ev.GetTaskRole() != c.TaskRole {
					receipt.Status = "error"
					receipt.Error = "gateway route receipt task role does not match replay request"
					results[job.index] = receipt
					continue
				}
				receipt.GatewayEventID = ev.GetEventId()
				receipt.GatewayRequestID = ev.GetRequestId()
				receipt.CapacityClaimID = ev.GetCapacityClaimId()
				receipt.CapacityRequiredBytes = ev.GetCapacityRequiredBytes()
				receipt.CapacityGrantedBytes = ev.GetCapacityGrantedBytes()
				receipt.CapacityVerdict = ev.GetCapacityVerdict()
				receipt.CapacityEstimateEvidenceURI = c.CapacityEstimateEvidenceURI
				if receipt.GatewayEventID == "" || receipt.GatewayRequestID == "" {
					receipt.Status = "error"
					receipt.Error = "gateway route receipt omitted event or request identity"
					results[job.index] = receipt
					continue
				}
				if receipt.GatewayRequestID != requestID {
					receipt.Status = "error"
					receipt.Error = "gateway route receipt request identity does not match Agent Manager source"
					results[job.index] = receipt
					continue
				}
				if receipt.CapacityVerdict != "fit" || receipt.CapacityClaimID == "" || receipt.CapacityRequiredBytes != c.RequiredVRAMBytes || receipt.CapacityGrantedBytes < c.RequiredVRAMBytes {
					receipt.Status = "error"
					receipt.Error = "gateway route receipt omitted verified capacity admission"
					results[job.index] = receipt
					continue
				}
				receipt.GatewayRoleEvidenceURI = "gateway-ledger://events/" + receipt.GatewayEventID
				receipt.Status = ev.GetStatus()
				receipt.SelectedProvider = ev.GetSelectedProvider()
				receipt.SelectedModel = ev.GetSelectedModel()
				receipt.TaskRole = ev.GetTaskRole()
				receipt.SelectedResourceRole = ev.GetSelectedResourceRole()
				if receipt.Status == "succeeded" && receipt.SelectedResourceRole == "" {
					receipt.Status = "error"
					receipt.Error = "successful gateway route receipt omitted selected resource role"
					results[job.index] = receipt
					continue
				}
				receipt.Locality = ev.GetSelectedLocality()
				if !response.GetValid() && receipt.Status == "" {
					receipt.Status = "invalid"
				}
				results[job.index] = receipt
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
	for _, c := range cases {
		if strings.TrimSpace(c.PrivacyProvenance) == "" {
			return nil, fmt.Errorf("preview replay case %q requires explicit privacy_provenance", c.CaseID)
		}
		if err := verify(ctx, c); err != nil {
			return nil, fmt.Errorf("verify source for replay case %q: %w", c.CaseID, err)
		}
	}
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
				c := work.item
				c.TaskRole = replayTaskRole(c)
				receipt := ReplayReceipt{CaseID: c.CaseID, Mode: "preview", SourceRunID: c.SourceRunID, SourceEventID: c.SourceEventID, SourceSequence: c.SourceSequence, ConversationID: c.ConversationID, PrivacyProvenance: c.PrivacyProvenance, Scenario: c.Scenario, Role: c.Role, TaskRole: c.TaskRole, Profile: profileLabel(replayCaseProfile(c)), LatencyKind: "route_preview"}
				req := &routingv1.PreviewRouteRequest{Request: &sharedv1.GatewayRequest{
					Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, Role: c.Role, TaskRole: c.TaskRole,
					Profile: replayCaseProfile(c), PrivacyClass: replayPrivacy(c.PrivacyClass),
					PrivacyProvenance: c.PrivacyProvenance,
					Operation:         c.Operation, Scenario: c.Scenario, TimeoutMs: c.TimeoutMS,
				}}
				start := time.Now()
				response, err := preview(ctx, req)
				receipt.LatencyMS = time.Since(start).Milliseconds()
				if err != nil || response == nil {
					receipt.Status = "error"
					receipt.Error = "preview_route_failed"
				} else {
					valid := response.GetValid()
					receipt.Valid = &valid
					receipt.Status = "invalid"
					if valid {
						receipt.Status = "valid"
					}
					receipt.SelectedProvider = response.GetSelectedProvider()
					receipt.RoutePlanID = response.GetRoutePlanId()
					for _, candidate := range response.GetCandidates() {
						receipt.CandidateEligibility = append(receipt.CandidateEligibility, ReplayCandidateEligibility{
							Provider: candidate.GetProvider(), Role: candidate.GetRole(), Locality: candidate.GetLocality(),
							Selected: candidate.GetSelected(), Fallback: candidate.GetFallbackEligible(),
							RejectionCode: candidate.GetRejectionReason(), CapacityVerdict: candidate.GetCapacityVerdict(),
						})
					}
				}
				results[work.index] = receipt
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
