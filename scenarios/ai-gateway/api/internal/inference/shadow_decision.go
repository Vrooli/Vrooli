package inference

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/proto"
)

const ShadowDecisionMode = "shadow_only"

var ErrShadowDecisionRole = errors.New("shadow decisions require the existing decide.fast role")

// ShadowDecisionRecord separates a candidate evaluation from the task result
// that produced the baseline. It contains hashes rather than the source text;
// callers decide whether and where a record is retained.
type ShadowDecisionRecord struct {
	Mode                    string
	CandidateRole           string
	TaskResultSHA256        string
	CandidateRequestSHA256  string
	CandidateResponse       *inferencev1.DecideResponse
	TaskResultChanged       bool
	BrowserActionDispatched bool
}

// ShadowDecisionOutcome returns the original result byte-for-byte alongside
// an independently evaluated candidate. This helper is internal to the
// Gateway service package and is not exposed as an RPC or HTTP endpoint.
type ShadowDecisionOutcome struct {
	TaskResult []byte
	Record     ShadowDecisionRecord
}

// EvaluateShadowDecision evaluates a candidate through the existing Gateway
// decision service while preserving the baseline task result. It accepts only
// decide.fast; browser.decide remains fail-closed until an existing owner
// provides a verified run/grant/session binding. The caller's context, source,
// privacy policy and provider route are passed through unchanged.
func (s *Service) EvaluateShadowDecision(ctx context.Context, taskResult []byte, request DecisionRequest) (*ShadowDecisionOutcome, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	if taskResult == nil {
		return nil, errors.New("shadow decision requires an existing task result")
	}
	if request.Role != "decide.fast" {
		return nil, ErrShadowDecisionRole
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&inferencev1.DecideRequest{
		Role: request.Role, State: request.State, StateJson: request.StateJSON,
		Questions: request.Questions, Profile: request.Profile, PrivacyClass: request.PrivacyClass,
		PrivacyProvenance: request.PrivacyProvenance, PinResourceRole: request.PinResourceRole,
	})
	if err != nil {
		return nil, err
	}
	response := s.Decide(ctx, request)
	if response == nil {
		return nil, errors.New("Gateway returned no shadow decision response")
	}
	resultDigest := sha256.Sum256(taskResult)
	requestDigest := sha256.Sum256(encoded)
	return &ShadowDecisionOutcome{
		TaskResult: bytes.Clone(taskResult),
		Record: ShadowDecisionRecord{
			Mode:                    ShadowDecisionMode,
			CandidateRole:           request.Role,
			TaskResultSHA256:        hex.EncodeToString(resultDigest[:]),
			CandidateRequestSHA256:  hex.EncodeToString(requestDigest[:]),
			CandidateResponse:       proto.Clone(response).(*inferencev1.DecideResponse),
			TaskResultChanged:       false,
			BrowserActionDispatched: false,
		},
	}, nil
}
