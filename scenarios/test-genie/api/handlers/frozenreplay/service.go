package frozenreplay

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"test-genie/internal/frozenreplay"

	frozenreplayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/frozen_replay"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/frozen_replay/frozen_replay_v1connect"
)

type Service struct {
	connectv1.UnimplementedFrozenReplayServiceHandler
	owner *frozenreplay.Service
}

func New(owner *frozenreplay.Service) *Service { return &Service{owner: owner} }

func (s *Service) CreateFrozenReplayReceipt(ctx context.Context, req *connect.Request[frozenreplayv1.CreateFrozenReplayReceiptRequest]) (*connect.Response[frozenreplayv1.CreateFrozenReplayReceiptResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetAdjudication() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("adjudication is required"))
	}
	input, err := fromWire(req.Msg.GetAdjudication())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := s.owner.Create(ctx, frozenreplay.CreateRequest{IdempotencyKey: req.Msg.GetIdempotencyKey(), Receipt: input})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&frozenreplayv1.CreateFrozenReplayReceiptResponse{Receipt: toWire(result.Receipt), Replayed: result.Replayed}), nil
}

func (s *Service) GetFrozenReplayReceipt(ctx context.Context, req *connect.Request[frozenreplayv1.GetFrozenReplayReceiptRequest]) (*connect.Response[frozenreplayv1.GetFrozenReplayReceiptResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetReceiptId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("receipt_id is required"))
	}
	receipt, err := s.owner.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&frozenreplayv1.GetFrozenReplayReceiptResponse{Receipt: toWire(receipt)}), nil
}

func fromWire(input *frozenreplayv1.FrozenReplayAdjudication) (frozenreplay.Receipt, error) {
	receipt := frozenreplay.Receipt{
		CaseID: input.GetCaseId(), ProjectID: input.GetProjectId(),
		FrozenEvaluator: input.GetFrozenEvaluatorDigest(), PreimageIdentity: input.GetPreimageIdentity(),
		CandidateSeal: input.GetCandidateSealDigest(), ReferenceIdentity: input.GetReferenceIdentity(),
	}
	for _, evidence := range input.GetEvidence() {
		variant, ok := fromVariant(evidence.GetVariant())
		if !ok {
			return frozenreplay.Receipt{}, fmt.Errorf("unknown replay variant %s", evidence.GetVariant())
		}
		status, ok := fromStatus(evidence.GetStatus())
		if !ok {
			return frozenreplay.Receipt{}, fmt.Errorf("unknown test status %s", evidence.GetStatus())
		}
		receipt.Evidence = append(receipt.Evidence, frozenreplay.RunEvidence{
			Variant: variant, RunID: evidence.GetRunId(), ProducerReceipt: evidence.GetProducerReceiptId(),
			SourceIdentity: evidence.GetSourceIdentity(), EvaluatorDigest: evidence.GetEvaluatorDigest(),
			Status: status, ArtifactDigest: evidence.GetArtifactDigest(),
		})
	}
	return receipt, nil
}

func toWire(receipt frozenreplay.Receipt) *frozenreplayv1.FrozenReplayReceipt {
	result := &frozenreplayv1.FrozenReplayReceipt{
		ReceiptId: receipt.ReceiptDigest[len("sha256:"):],
		Adjudication: &frozenreplayv1.FrozenReplayAdjudication{
			CaseId: receipt.CaseID, ProjectId: receipt.ProjectID,
			FrozenEvaluatorDigest: receipt.FrozenEvaluator, PreimageIdentity: receipt.PreimageIdentity,
			CandidateSealDigest: receipt.CandidateSeal, ReferenceIdentity: receipt.ReferenceIdentity,
		},
		ReceiptDigest: receipt.ReceiptDigest,
	}
	if receipt.Disposition == "qualified" {
		result.Disposition = frozenreplayv1.FrozenReplayDisposition_FROZEN_REPLAY_DISPOSITION_QUALIFIED
	} else {
		result.Disposition = frozenreplayv1.FrozenReplayDisposition_FROZEN_REPLAY_DISPOSITION_NOT_QUALIFIED
	}
	for _, evidence := range receipt.Evidence {
		result.Adjudication.Evidence = append(result.Adjudication.Evidence, &frozenreplayv1.FrozenReplayRunEvidence{
			Variant: toVariant(evidence.Variant), RunId: evidence.RunID, ProducerReceiptId: evidence.ProducerReceipt,
			SourceIdentity: evidence.SourceIdentity, EvaluatorDigest: evidence.EvaluatorDigest,
			Status: toStatus(evidence.Status), ArtifactDigest: evidence.ArtifactDigest,
		})
	}
	return result
}

func fromVariant(value frozenreplayv1.FrozenReplayVariant) (frozenreplay.Variant, bool) {
	switch value {
	case frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_PREIMAGE:
		return frozenreplay.Preimage, true
	case frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_CANDIDATE:
		return frozenreplay.Candidate, true
	case frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_REFERENCE:
		return frozenreplay.Reference, true
	default:
		return "", false
	}
}

func toVariant(value frozenreplay.Variant) frozenreplayv1.FrozenReplayVariant {
	switch value {
	case frozenreplay.Preimage:
		return frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_PREIMAGE
	case frozenreplay.Candidate:
		return frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_CANDIDATE
	case frozenreplay.Reference:
		return frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_REFERENCE
	default:
		return frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_UNSPECIFIED
	}
}

func fromStatus(value frozenreplayv1.FrozenReplayTestStatus) (string, bool) {
	switch value {
	case frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_PASSED:
		return "passed", true
	case frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_FAILED:
		return "failed", true
	default:
		return "", false
	}
}

func toStatus(value string) frozenreplayv1.FrozenReplayTestStatus {
	if value == "passed" {
		return frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_PASSED
	}
	return frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_FAILED
}

func mapError(err error) error {
	switch {
	case errors.Is(err, frozenreplay.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, frozenreplay.ErrIdempotencyKey):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, frozenreplay.ErrInvalidReceipt):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeUnavailable, err)
	}
}

var _ connectv1.FrozenReplayServiceHandler = (*Service)(nil)
