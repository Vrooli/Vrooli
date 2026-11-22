package routing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"

	"ai-gateway/internal/providers"
)

func applySamplingEvidence(ev *routingv1.RouteEvidence, temperature *float64, support providers.SamplingSupport) {
	if temperature != nil {
		value := *temperature
		ev.SamplingTemperature = &value
	}
	ev.SamplingTemperatureSupport = samplingSupportEnum(string(support))
}

// appliedSettings reports what the gateway actually did. The routing path never
// substitutes a cap of its own, so the only cap it can report is the caller's:
// zero means the gateway imposed nothing and the resource role's policy default
// applies, which it deliberately does not guess at.
func appliedSettings(ev *routingv1.RouteEvidence, req *sharedv1.GatewayRequest) *sharedv1.AppliedSettings {
	applied := &sharedv1.AppliedSettings{
		TemperatureSupport:       ev.GetSamplingTemperatureSupport(),
		MaxOutputTokensEffective: req.GetMaxOutputTokens(),
		MaxOutputTokensSource:    sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_NONE_IMPOSED,
	}
	if req.GetMaxOutputTokens() > 0 {
		applied.MaxOutputTokensSource = sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST
	}
	if ev.SamplingTemperature != nil {
		value := ev.GetSamplingTemperature()
		applied.TemperatureSent = &value
	}
	return applied
}

func (s *Service) ListEvidence(ctx context.Context, filter EvidenceFilter) ([]*routingv1.RouteEvidence, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) GetEvidence(ctx context.Context, eventID string) (*routingv1.RouteEvidence, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	ev, err := s.repo.Get(ctx, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("route evidence %q was not found", eventID)
	}
	return ev, err
}

func (s *Service) evidence(req *sharedv1.GatewayRequest, plan routePlan, status string, latency time.Duration, fallback bool, failures []string) *routingv1.RouteEvidence {
	ev := &routingv1.RouteEvidence{
		EventId:           newID("rt"),
		RequestId:         firstNonEmpty(req.GetRequestId(), newID("req")),
		Scenario:          strings.TrimSpace(req.GetScenario()),
		Operation:         strings.TrimSpace(req.GetOperation()),
		Role:              strings.TrimSpace(req.GetRole()),
		TaskRole:          strings.TrimSpace(firstNonEmpty(req.GetTaskRole(), req.GetRole())),
		Profile:           req.GetProfile(),
		PrivacyClass:      req.GetPrivacyClass(),
		PrivacyProvenance: strings.TrimSpace(req.GetPrivacyProvenance()),
		SelectedProvider:  plan.selectedProvider,
		SelectedLocality:  plan.selectedLocality,
		Status:            status,
		PolicyReasons:     append([]string{}, plan.policyReasons...),
		FailureReasons:    stableFailureReasons(failures),
		FallbackUsed:      fallback,
		// The redacted-evidence seam reflects whether a PII-detection pass ran,
		// not an unconditional claim. A DETECT_PII request scans its own input
		// text (the "prompt"), so a successful detect pass sets PromptRedacted.
		// ResponseRedacted stays false until a response-redaction policy exists
		// (owned by the agent-manager redact follow-up), rather than asserting a
		// redaction that never happened.
		PromptRedacted:   detectPassRan(req, status),
		ResponseRedacted: false,
		LatencyMs:        latency.Milliseconds(),
		TotalDurationNs:  max(int64(latency), 0),
		CreatedAt:        s.now().UTC().Format(time.RFC3339Nano),
	}
	populateAttachmentEvidence(ev, req.GetAttachments())
	return ev
}

// stableFailureReasons is the durable privacy boundary for adapter failures.
// Never persist provider errors, stderr, prompts, outputs, or arbitrary future
// caller strings in route evidence. Unknown values collapse to one stable code.
func stableFailureReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		failureReasonAdapterMissing: {},
		failureReasonRoleResolve:    {},
		failureReasonProfile:        {},
		failureReasonRights:         {},
		failureReasonSampling:       {},
		failureReasonCapacity:       {},
		failureReasonExecute:        {},
		failureReasonNoProvider:     {},
	}
	result := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if _, ok := allowed[reason]; !ok {
			reason = failureReasonUnknown
		}
		result = append(result, reason)
	}
	return result
}

// detectPassRan reports whether a PII-detection pass was applied to this
// request's input. It is the signal that makes the prompt_redacted route-
// evidence column meaningful: true only when a DETECT_PII request completed
// successfully, false otherwise (no detector ran, or it failed).
func detectPassRan(req *sharedv1.GatewayRequest, status string) bool {
	return req.GetKind() == sharedv1.RequestKind_REQUEST_KIND_DETECT_PII && status == "succeeded"
}

func (s *Service) resolveAttachments(ctx context.Context, req *sharedv1.GatewayRequest) (*sharedv1.GatewayRequest, error) {
	if len(req.GetAttachments()) == 0 {
		return req, nil
	}
	var resolved []*sharedv1.Attachment
	for _, attachment := range req.GetAttachments() {
		if attachment == nil || strings.TrimSpace(attachment.GetReference()) == "" {
			resolved = append(resolved, attachment)
			continue
		}
		if s.resolver == nil {
			return nil, fmt.Errorf("attachment reference %q cannot be resolved without an AttachmentResolver", attachment.GetReference())
		}
		item, err := s.resolver.Resolve(ctx, attachment)
		if err != nil {
			return nil, fmt.Errorf("resolve attachment reference: %w", err)
		}
		resolved = append(resolved, item)
	}
	copyReq := proto.Clone(req).(*sharedv1.GatewayRequest)
	copyReq.Attachments = resolved
	return copyReq, nil
}

func populateAttachmentEvidence(ev *routingv1.RouteEvidence, attachments []*sharedv1.Attachment) {
	if len(attachments) == 0 {
		return
	}
	hash := sha256.New()
	for _, attachment := range attachments {
		if attachment == nil {
			continue
		}
		data := attachment.GetInlineBytes()
		if len(data) > 0 {
			_, _ = hash.Write(data)
		}
		bytes := attachment.GetBytes()
		if bytes == 0 {
			bytes = uint64(len(data))
		}
		ev.AttachmentBytes = addAttachmentBytes(ev.AttachmentBytes, bytes)
		if attachment.GetModality() == sharedv1.Modality_MODALITY_IMAGE {
			ev.ImageCount++
		}
		ev.AttachmentDimensions = append(ev.AttachmentDimensions, &routingv1.AttachmentDimension{
			Modality:  attachment.GetModality(),
			MediaType: attachment.GetMediaType(),
			Width:     attachment.GetWidth(),
			Height:    attachment.GetHeight(),
			Bytes:     bytes,
		})
	}
	ev.AttachmentSha256 = hex.EncodeToString(hash.Sum(nil))
	ev.AttachmentsRedacted = true
}

const maxEvidenceAttachmentBytes = int64(1<<63 - 1)

// addAttachmentBytes keeps the signed evidence counter non-negative when the
// wire contract's uint64 byte count exceeds int64 or the total would overflow.
// Request validation bounds individual attachments; saturation protects the
// aggregate telemetry field from wrapping on unusually large batches.
func addAttachmentBytes(total int64, bytes uint64) int64 {
	if total < 0 {
		total = 0
	}
	if bytes > uint64(maxEvidenceAttachmentBytes) || total > maxEvidenceAttachmentBytes-int64(bytes) {
		return maxEvidenceAttachmentBytes
	}
	return total + int64(bytes)
}
