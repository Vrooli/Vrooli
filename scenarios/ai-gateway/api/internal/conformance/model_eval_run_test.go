package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type modelEvalApprovalFixture struct{ answer, scope, id string }

func (f modelEvalApprovalFixture) ReadModelEvalApproval(context.Context, string) (ModelEvalApproval, error) {
	return ModelEvalApproval{DecisionID: f.id, Answer: f.answer, Scope: f.scope, RecordedAt: time.Now().UTC()}, nil
}

type modelEvalSourceFixture struct {
	cases  []ModelEvalCase
	loaded atomic.Int32
}

func (f *modelEvalSourceFixture) LoadFrozenCases(context.Context, int) ([]ModelEvalCase, error) {
	f.loaded.Add(1)
	return append([]ModelEvalCase(nil), f.cases...), nil
}

func (f *modelEvalSourceFixture) VerifyCaseIdentity(_ context.Context, item ModelEvalCase) error {
	if item.EventID == "bad" {
		return errors.New("identity mismatch")
	}
	return nil
}

type modelEvalPrivacyFixture struct{ err error }

func (f modelEvalPrivacyFixture) ProvePromptOutputNonPersistence(context.Context) error { return f.err }

type modelEvalPolicyFixture struct {
	rightsErr error
	remote    bool
	locality  string
}

func (f modelEvalPolicyFixture) ResolveChatDefault(context.Context) (string, string, bool, error) {
	locality := f.locality
	if locality == "" {
		locality = "local"
	}
	return "ollama/qwen3:8b", locality, f.remote, nil
}

func (f modelEvalPolicyFixture) CheckCurrentRights(context.Context, string, string) error {
	return f.rightsErr
}

type modelEvalDiskFixture struct {
	err      error
	free     int64
	artifact int64
}

func (f modelEvalDiskFixture) MeasureModelEvalDisk(context.Context) (ModelEvalDiskMeasurement, error) {
	if f.err != nil {
		return ModelEvalDiskMeasurement{}, f.err
	}
	return ModelEvalDiskMeasurement{FreshAt: time.Now().UTC(), FreeBytes: f.free, ArtifactBytes: f.artifact, ArtifactSHA256: strings.Repeat("a", 64)}, nil
}

type modelEvalCapacityFixture struct {
	fail   bool
	serial *atomic.Int32
}

func (f modelEvalCapacityFixture) AcquireModelEvalClaim(_ context.Context, owner string, required int64) (ModelEvalCapacityReceipt, func(context.Context) error, error) {
	if f.fail {
		return ModelEvalCapacityReceipt{}, nil, errors.New("capacity blocked")
	}
	if f.serial != nil && f.serial.Add(1) != 1 {
		return ModelEvalCapacityReceipt{}, nil, errors.New("parallel claim")
	}
	return ModelEvalCapacityReceipt{ClaimID: "claim-" + owner, RequiredBytes: required, GrantedBytes: required, FreeBytes: required, ObservedAt: time.Now().UTC()}, func(context.Context) error {
		if f.serial != nil {
			f.serial.Add(-1)
		}
		return nil
	}, nil
}

type modelEvalDispatchFixture struct {
	active atomic.Int32
	calls  atomic.Int32
	failAt int32
}

func (f *modelEvalDispatchFixture) DispatchModelEval(_ context.Context, model, _, prompt string) (ModelEvalDispatchReceipt, error) {
	if f.active.Add(1) != 1 {
		return ModelEvalDispatchReceipt{}, errors.New("concurrent dispatch")
	}
	defer f.active.Add(-1)
	call := f.calls.Add(1)
	if call == f.failAt {
		return ModelEvalDispatchReceipt{}, errors.New("provider error contained no body")
	}
	if !strings.HasPrefix(prompt, "PRIVATE_") {
		return ModelEvalDispatchReceipt{}, errors.New("fixture prompt not supplied")
	}
	return ModelEvalDispatchReceipt{
		EffectiveModel: model, Locality: "local", Latency: 12 * time.Millisecond, RightsAllowed: true,
		Capacity: ModelEvalCapacityReceipt{ClaimID: fmt.Sprintf("claim-ai-gateway:model-eval:event-%d", call-1), RequiredBytes: modelEvalMinClaimBytes, GrantedBytes: modelEvalMinClaimBytes},
	}, nil
}

func modelEvalRunnerFixture(answer string, source *modelEvalSourceFixture, dispatch *modelEvalDispatchFixture, capacityFail bool) ModelEvalBatchRunner {
	var active atomic.Int32
	return ModelEvalBatchRunner{
		Approval: modelEvalApprovalFixture{answer: answer, id: modelEvalAdmissionID, scope: modelEvalAdmissionScope},
		Source:   source, Privacy: modelEvalPrivacyFixture{}, Policy: modelEvalPolicyFixture{},
		Disk:     modelEvalDiskFixture{free: modelEvalMinDiskBytes, artifact: 1024},
		Capacity: modelEvalCapacityFixture{fail: capacityFail, serial: &active}, Dispatch: dispatch,
	}
}

func modelEvalCases(count int) []ModelEvalCase {
	cases := make([]ModelEvalCase, count)
	for i := range cases {
		cases[i] = ModelEvalCase{
			EventID: fmt.Sprintf("event-%d", i), RunID: fmt.Sprintf("run-%d", i), Sequence: int64(i),
			EventTimestamp: time.Date(2026, 11, 15, 0, 0, i, 0, time.UTC), RequestID: fmt.Sprintf("request-%d", i),
			PolicyRole: "chat.default", TaskRole: "assistant.chat", Consumer: "agent-manager",
			Prompt: fmt.Sprintf("PRIVATE_PROMPT_%d", i),
		}
	}
	return cases
}

func TestModelEvalBatchRefusesBeforeApprovalAndAfterDefer(t *testing.T) {
	for _, answer := range []string{"", "defer", "deny"} {
		t.Run(fmt.Sprintf("answer_%q", answer), func(t *testing.T) {
			source := &modelEvalSourceFixture{cases: modelEvalCases(2)}
			dispatch := &modelEvalDispatchFixture{}
			runner := modelEvalRunnerFixture(answer, source, dispatch, false)
			_, err := runner.Run(context.Background())
			if !errors.Is(err, errModelEvalAdmissionRefused) {
				t.Fatalf("Run error=%v, want admission refusal", err)
			}
			if dispatch.calls.Load() != 0 || source.loaded.Load() != 0 {
				t.Fatalf("pre-approval work: dispatches=%d source loads=%d", dispatch.calls.Load(), source.loaded.Load())
			}
		})
	}
}

func TestModelEvalBatchRunsSequentiallyWithMetadataOnlyFixedDenominator(t *testing.T) {
	source := &modelEvalSourceFixture{cases: modelEvalCases(3)}
	dispatch := &modelEvalDispatchFixture{}
	runner := modelEvalRunnerFixture("approve", source, dispatch, false)
	receipt, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Denominator != 3 || receipt.Succeeded != 3 || receipt.Failed != 0 || receipt.Refused != 0 || receipt.Unattempted != 0 || dispatch.calls.Load() != 3 {
		t.Fatalf("unexpected fixed-denominator result: %+v calls=%d", receipt, dispatch.calls.Load())
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{"PRIVATE_PROMPT_", "PRIVATE_OUTPUT_", "prompt", "output"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("receipt contains prohibited body field %q: %s", forbidden, encoded)
		}
	}
	if len(receipt.CaseSetSHA256) != 64 || !validSHA256(receipt.CaseSetSHA256) {
		t.Fatalf("invalid frozen case-set digest %q", receipt.CaseSetSHA256)
	}
}

func TestModelEvalBatchStopsAfterProviderFailureAndPreservesDenominator(t *testing.T) {
	source := &modelEvalSourceFixture{cases: modelEvalCases(4)}
	dispatch := &modelEvalDispatchFixture{failAt: 2}
	runner := modelEvalRunnerFixture("approve", source, dispatch, false)
	receipt, err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("provider failure unexpectedly succeeded")
	}
	if receipt.Denominator != 4 || receipt.Succeeded != 1 || receipt.Failed != 1 || receipt.Refused != 0 || receipt.Unattempted != 2 || len(receipt.Cases) != 4 || dispatch.calls.Load() != 2 {
		t.Fatalf("failure denominator changed: %+v calls=%d", receipt, dispatch.calls.Load())
	}
}

func TestModelEvalBatchCapacityRefusalStopsBeforeDispatch(t *testing.T) {
	source := &modelEvalSourceFixture{cases: modelEvalCases(2)}
	dispatch := &modelEvalDispatchFixture{}
	runner := modelEvalRunnerFixture("approve", source, dispatch, true)
	receipt, err := runner.Run(context.Background())
	if !errors.Is(err, errModelEvalAdmissionRefused) {
		t.Fatalf("Run error=%v, want admission refusal", err)
	}
	if dispatch.calls.Load() != 0 || receipt.Denominator != 2 || receipt.Refused != 1 || receipt.Unattempted != 1 {
		t.Fatalf("capacity refusal did not preserve the denominator: %+v calls=%d", receipt, dispatch.calls.Load())
	}
}

func TestModelEvalBatchRightsRefusalPreservesRemainingAsUnattempted(t *testing.T) {
	source := &modelEvalSourceFixture{cases: modelEvalCases(3)}
	dispatch := &modelEvalDispatchFixture{}
	runner := modelEvalRunnerFixture("approve", source, dispatch, false)
	runner.Policy = modelEvalPolicyFixture{rightsErr: errors.New("rights unknown")}
	receipt, err := runner.Run(context.Background())
	if !errors.Is(err, errModelEvalAdmissionRefused) {
		t.Fatalf("Run error=%v, want rights refusal", err)
	}
	if dispatch.calls.Load() != 0 || receipt.Denominator != 3 || receipt.Refused != 1 || receipt.Unattempted != 2 {
		t.Fatalf("rights refusal changed the denominator: %+v calls=%d", receipt, dispatch.calls.Load())
	}
}

func TestModelEvalBatchDiskFloorAndArtifactBoundFailBeforeDispatch(t *testing.T) {
	for _, test := range []struct {
		name     string
		free     int64
		artifact int64
	}{
		{name: "below disk floor", free: modelEvalMinDiskBytes - 1, artifact: 1},
		{name: "artifact over bound", free: modelEvalMinDiskBytes, artifact: modelEvalMaxArtifact + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := &modelEvalSourceFixture{cases: modelEvalCases(2)}
			dispatch := &modelEvalDispatchFixture{}
			runner := modelEvalRunnerFixture("approve", source, dispatch, false)
			runner.Disk = modelEvalDiskFixture{free: test.free, artifact: test.artifact}
			receipt, err := runner.Run(context.Background())
			if !errors.Is(err, errModelEvalAdmissionRefused) {
				t.Fatalf("Run error=%v, want disk refusal", err)
			}
			if dispatch.calls.Load() != 0 || receipt.Denominator != 2 || receipt.Refused != 2 {
				t.Fatalf("disk refusal did not preserve cases: %+v calls=%d", receipt, dispatch.calls.Load())
			}
		})
	}
}

func TestModelEvalBatchRefusesUnsafePolicyPrivacyAndCaseIdentity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ModelEvalBatchRunner, *modelEvalSourceFixture)
	}{
		{name: "remote route", mutate: func(r *ModelEvalBatchRunner, _ *modelEvalSourceFixture) {
			r.Policy = modelEvalPolicyFixture{remote: true}
		}},
		{name: "privacy unknown", mutate: func(r *ModelEvalBatchRunner, _ *modelEvalSourceFixture) {
			r.Privacy = modelEvalPrivacyFixture{err: errors.New("non-persistence unknown")}
		}},
		{name: "source identity mismatch", mutate: func(_ *ModelEvalBatchRunner, s *modelEvalSourceFixture) { s.cases[0].EventID = "bad" }},
		{name: "missing request identity", mutate: func(_ *ModelEvalBatchRunner, s *modelEvalSourceFixture) { s.cases[0].RequestID = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := &modelEvalSourceFixture{cases: modelEvalCases(1)}
			dispatch := &modelEvalDispatchFixture{}
			runner := modelEvalRunnerFixture("approve", source, dispatch, false)
			test.mutate(&runner, source)
			_, err := runner.Run(context.Background())
			if !errors.Is(err, errModelEvalAdmissionRefused) {
				t.Fatalf("Run error=%v, want fail-closed refusal", err)
			}
			if dispatch.calls.Load() != 0 {
				t.Fatalf("dispatch count=%d before preflight passed", dispatch.calls.Load())
			}
		})
	}
}
