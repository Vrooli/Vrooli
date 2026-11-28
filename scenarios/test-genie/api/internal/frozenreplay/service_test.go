package frozenreplay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"test-genie/internal/testsqlite"
)

type allowEvidence struct{}

func (allowEvidence) VerifyEvidence(context.Context, RunEvidence) error { return nil }

type digestingEvidence struct{ allowEvidence }

func (digestingEvidence) ArtifactDigest(_ context.Context, evidence RunEvidence) (string, error) {
	return digest("owner:" + evidence.ProducerReceipt), nil
}

type rejectEvidence struct{}

func (rejectEvidence) VerifyEvidence(context.Context, RunEvidence) error {
	return errors.New("owner record does not match")
}

func TestAdjudicateSealsOnlyFailPassPass(t *testing.T) {
	input := validInput()
	receipt, err := Adjudicate(context.Background(), input, allowEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Disposition != "qualified" {
		t.Fatalf("disposition = %q, want qualified", receipt.Disposition)
	}
	if err := Verify(receipt); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	receipt.Evidence[1].Status = "failed"
	if err := Verify(receipt); err == nil {
		t.Fatal("Verify accepted a changed receipt")
	}

	input.Evidence[1].Status = "passed"
	notQualified, err := Adjudicate(context.Background(), input, allowEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	if notQualified.Disposition != "not_qualified" {
		t.Fatalf("disposition = %q, want not_qualified", notQualified.Disposition)
	}
}

func TestAdjudicateRequiresOwnerVerificationAndCompleteDistinctRuns(t *testing.T) {
	if _, err := Adjudicate(context.Background(), validInput(), rejectEvidence{}); !errors.Is(err, ErrInvalidReceipt) {
		t.Fatalf("unverified evidence error = %v", err)
	}
	input := validInput()
	input.Evidence[2].RunID = input.Evidence[0].RunID
	if _, err := Adjudicate(context.Background(), input, allowEvidence{}); !errors.Is(err, ErrInvalidReceipt) {
		t.Fatalf("duplicate run error = %v", err)
	}
}

func TestAdjudicateDerivesMissingArtifactDigestsFromOwner(t *testing.T) {
	input := validInput()
	for index := range input.Evidence {
		input.Evidence[index].ArtifactDigest = ""
	}
	receipt, err := Adjudicate(context.Background(), input, digestingEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range receipt.Evidence {
		if evidence.ArtifactDigest != digest("owner:"+evidence.ProducerReceipt) {
			t.Fatalf("artifact digest was not derived from owner: %+v", evidence)
		}
	}
}

func TestCreateGetIdempotencyAndImmutableStorage(t *testing.T) {
	db := testsqlite.Open(t)
	service, err := NewService(context.Background(), db, allowEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	request := CreateRequest{IdempotencyKey: "replay-case-0001", Receipt: validInput()}
	first, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed {
		t.Fatal("first create reported replay")
	}
	second, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Receipt.ReceiptDigest != first.Receipt.ReceiptDigest {
		t.Fatalf("idempotent create = %#v", second)
	}

	conflict := request
	conflict.Receipt.Evidence[1].Status = "passed"
	if _, err := service.Create(context.Background(), conflict); !errors.Is(err, ErrIdempotencyKey) {
		t.Fatalf("conflicting idempotency key error = %v", err)
	}

	stored, err := service.Get(context.Background(), first.Receipt.ReceiptDigest)
	if err != nil || stored.ReceiptDigest != first.Receipt.ReceiptDigest {
		t.Fatalf("Get = %#v, %v", stored, err)
	}
	if _, err := db.ExecContext(context.Background(), `UPDATE frozen_replay_receipts SET receipt_json = '{}' WHERE receipt_id = ?`, receiptID(first.Receipt)); err == nil {
		t.Fatal("immutable receipt storage accepted UPDATE")
	}
	if _, err := db.ExecContext(context.Background(), `DELETE FROM frozen_replay_receipts WHERE receipt_id = ?`, receiptID(first.Receipt)); err == nil {
		t.Fatal("immutable receipt storage accepted DELETE")
	}
}

func TestCreateWithOwnerDerivedArtifactsIsIdempotent(t *testing.T) {
	service, err := NewService(context.Background(), testsqlite.Open(t), digestingEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	request := CreateRequest{IdempotencyKey: "replay-derived-0001", Receipt: validInput()}
	for index := range request.Receipt.Evidence {
		request.Receipt.Evidence[index].ArtifactDigest = ""
	}
	first, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("idempotent create with omitted artifact digests: %v", err)
	}
	if !second.Replayed || first.Receipt.ReceiptDigest != second.Receipt.ReceiptDigest {
		t.Fatalf("owner-derived artifact create did not replay same receipt: first=%+v second=%+v", first, second)
	}
	if _, err := service.Get(context.Background(), first.Receipt.ReceiptDigest); err != nil {
		t.Fatalf("get owner-derived artifact receipt: %v", err)
	}
}

func TestHTTPCreateGetBinding(t *testing.T) {
	service, err := NewService(context.Background(), testsqlite.Open(t), allowEvidence{})
	if err != nil {
		t.Fatal(err)
	}
	handler := service.Handler()
	body, err := json.Marshal(CreateRequest{IdempotencyKey: "replay-http-0001", Receipt: validInput()})
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", created.Code, created.Body.String())
	}
	var result CreateResult
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/"+receiptID(result.Receipt), nil))
	if read.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", read.Code, read.Body.String())
	}
}

func validInput() Receipt {
	return Receipt{
		CaseID:            "case-1",
		ProjectID:         "project-1",
		FrozenEvaluator:   digest("evaluator"),
		PreimageIdentity:  digest("preimage"),
		CandidateSeal:     digest("candidate"),
		ReferenceIdentity: digest("reference"),
		Evidence: []RunEvidence{
			{Variant: Reference, RunID: "run-reference", ProducerReceipt: "receipt-reference", SourceIdentity: digest("reference"), EvaluatorDigest: digest("evaluator"), Status: "passed", ArtifactDigest: digest("artifact-reference")},
			{Variant: Preimage, RunID: "run-preimage", ProducerReceipt: "receipt-preimage", SourceIdentity: digest("preimage"), EvaluatorDigest: digest("evaluator"), Status: "failed", ArtifactDigest: digest("artifact-preimage")},
			{Variant: Candidate, RunID: "run-candidate", ProducerReceipt: "receipt-candidate", SourceIdentity: digest("candidate"), EvaluatorDigest: digest("evaluator"), Status: "passed", ArtifactDigest: digest("artifact-candidate")},
		},
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func receiptID(receipt Receipt) string {
	return receipt.ReceiptDigest[len("sha256:"):]
}
