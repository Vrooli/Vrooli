package frozenreplay

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	core "test-genie/internal/frozenreplay"
	"test-genie/internal/testsqlite"

	frozenreplayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/frozen_replay"
)

type acceptedEvidence struct{}

func (acceptedEvidence) VerifyEvidence(context.Context, core.RunEvidence) error { return nil }

func TestConnectCreateGetFrozenReplayReceipt(t *testing.T) {
	owner, err := core.NewService(context.Background(), testsqlite.Open(t), acceptedEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	service := New(owner)
	created, err := service.CreateFrozenReplayReceipt(context.Background(), connect.NewRequest(&frozenreplayv1.CreateFrozenReplayReceiptRequest{
		IdempotencyKey: "connect-replay-0001",
		Adjudication: &frozenreplayv1.FrozenReplayAdjudication{
			CaseId: "case-1", ProjectId: "project-1", FrozenEvaluatorDigest: digest("evaluator"),
			PreimageIdentity: digest("preimage"), CandidateSealDigest: digest("candidate"), ReferenceIdentity: digest("reference"),
			Evidence: []*frozenreplayv1.FrozenReplayRunEvidence{
				{Variant: frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_PREIMAGE, RunId: "run-before", ProducerReceiptId: "receipt-before", SourceIdentity: digest("preimage"), EvaluatorDigest: digest("evaluator"), Status: frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_FAILED, ArtifactDigest: digest("result-before")},
				{Variant: frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_CANDIDATE, RunId: "run-candidate", ProducerReceiptId: "receipt-candidate", SourceIdentity: digest("candidate"), EvaluatorDigest: digest("evaluator"), Status: frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_PASSED, ArtifactDigest: digest("result-candidate")},
				{Variant: frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_REFERENCE, RunId: "run-reference", ProducerReceiptId: "receipt-reference", SourceIdentity: digest("reference"), EvaluatorDigest: digest("evaluator"), Status: frozenreplayv1.FrozenReplayTestStatus_FROZEN_REPLAY_TEST_STATUS_PASSED, ArtifactDigest: digest("result-reference")},
			},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if created.Msg.GetReceipt().GetDisposition() != frozenreplayv1.FrozenReplayDisposition_FROZEN_REPLAY_DISPOSITION_QUALIFIED {
		t.Fatalf("disposition = %s", created.Msg.GetReceipt().GetDisposition())
	}
	got, err := service.GetFrozenReplayReceipt(context.Background(), connect.NewRequest(&frozenreplayv1.GetFrozenReplayReceiptRequest{ReceiptId: created.Msg.GetReceipt().GetReceiptId()}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetReceipt().GetReceiptDigest() != created.Msg.GetReceipt().GetReceiptDigest() {
		t.Fatalf("get digest = %s, want %s", got.Msg.GetReceipt().GetReceiptDigest(), created.Msg.GetReceipt().GetReceiptDigest())
	}
}

func TestConnectRejectsUnknownVariant(t *testing.T) {
	owner, err := core.NewService(context.Background(), testsqlite.Open(t), acceptedEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	service := New(owner)
	_, err = service.CreateFrozenReplayReceipt(context.Background(), connect.NewRequest(&frozenreplayv1.CreateFrozenReplayReceiptRequest{Adjudication: &frozenreplayv1.FrozenReplayAdjudication{
		Evidence: []*frozenreplayv1.FrozenReplayRunEvidence{{Variant: frozenreplayv1.FrozenReplayVariant_FROZEN_REPLAY_VARIANT_UNSPECIFIED}},
	}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error = %v, want invalid argument", err)
	}
}

func digest(value string) string {
	return "sha256:" + fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}
