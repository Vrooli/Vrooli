package routing

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ai-gateway/internal/providers"
	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
)

type successfulMediaExecutor struct{}

func (successfulMediaExecutor) Execute(_ context.Context, _ *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
	return &MediaExecutionResult{
		RouteEvidence: &routingv1.RouteEvidence{EventId: "route-media-1", Status: "succeeded", SelectedProvider: "resource-openrouter"},
		Outputs:       []*routingv1.MediaOutput{{Reference: "blob://output-1", MediaType: "image/png", Bytes: 42, Checksum: "sha256:test"}},
		ActualCostUSD: float64Pointer(0.015),
		ResolvedModel: "resource-resolved-model",
		Seed:          "1234",
		Warnings:      []string{"deterministic fixture"},
	}, nil
}

func (successfulMediaExecutor) Cancel(context.Context, string) error {
	return errors.New("not running")
}

func TestMediaSubmissionPersistsReceiptIsIdempotentAndFailsTruthfullyWithoutExecutor(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newMediaSchemaDB(t)
	svc := NewMediaService(db, nil)
	req := mediaRequest("media-idempotency-1")

	first, err := svc.Submit(context.Background(), req)
	require.NoError(t, err)
	require.NotEmpty(t, first.GetExecutionId())
	second, err := svc.Submit(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, first.GetExecutionId(), second.GetExecutionId())

	require.Eventually(t, func() bool {
		got, getErr := svc.Get(context.Background(), first.GetExecutionId())
		return getErr == nil && got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), first.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, "executor_unavailable", got.GetErrorCode())
	require.Empty(t, got.GetOutputs())
}

func TestMediaSubmissionRejectsNonMediaRequestKinds(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newMediaSchemaDB(t)
	svc := NewMediaService(db, nil)
	req := mediaRequest("media-invalid-kind")
	req.Request.Kind = sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION
	_, err := svc.Submit(context.Background(), req)
	require.ErrorContains(t, err, "image_generation or video_generation")
}

func TestMediaSubmissionPersistsExecutorOutputAndAccounting(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	db := newMediaSchemaDB(t)
	svc := NewMediaService(db, successfulMediaExecutor{})
	exec, err := svc.Submit(context.Background(), mediaRequest("media-success-1"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, getErr := svc.Get(context.Background(), exec.GetExecutionId())
		return getErr == nil && (got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED || got.GetStatus() == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED)
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), exec.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED, got.GetStatus(), "error=%s: %s", got.GetErrorCode(), got.GetErrorMessage())
	require.Len(t, got.GetOutputs(), 1)
	require.Equal(t, "blob://output-1", got.GetOutputs()[0].GetReference())
	require.InDelta(t, 0.015, got.GetActualCostUsd(), 1e-9)
	require.Equal(t, "resource-resolved-model", got.GetResolvedModel())
	require.Equal(t, "1234", got.GetSeed())
	require.NotNil(t, got.ActualCostUsd)
	require.Equal(t, 1, mediaKnownCost(t, db, exec.GetExecutionId()))
}

func TestMediaReceiptDistinguishesKnownZeroFromUnknownAcrossPersistenceAndProto(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown", true: "known_zero"}[known], func(t *testing.T) {
			db := newMediaSchemaDB(t)
			var cost *float64
			if known {
				cost = float64Pointer(0)
			}
			executor := mediaExecutorFunc(func(context.Context, *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
				return &MediaExecutionResult{RouteEvidence: &routingv1.RouteEvidence{Status: "succeeded"}, Outputs: []*routingv1.MediaOutput{{Reference: "file:///tmp/out.png", MediaType: "image/png"}}, ActualCostUSD: cost}, nil
			})
			svc := NewMediaService(db, executor)
			submitted, err := svc.Submit(context.Background(), mediaRequest("presence-"+strconv.FormatBool(known)))
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				got, err := svc.Get(context.Background(), submitted.GetExecutionId())
				return err == nil && terminalMediaStatus(got.GetStatus())
			}, time.Second, 10*time.Millisecond)
			got, err := svc.Get(context.Background(), submitted.GetExecutionId())
			require.NoError(t, err)
			require.Equal(t, known, got.ActualCostUsd != nil)
			require.Equal(t, boolInt(known), mediaKnownCost(t, db, got.GetExecutionId()))
			binary, err := proto.Marshal(got)
			require.NoError(t, err)
			var fromBinary routingv1.MediaExecution
			require.NoError(t, proto.Unmarshal(binary, &fromBinary))
			jsonBytes, err := protojson.Marshal(got)
			require.NoError(t, err)
			var fromJSON routingv1.MediaExecution
			require.NoError(t, protojson.Unmarshal(jsonBytes, &fromJSON))
			for _, roundTrip := range []*routingv1.MediaExecution{&fromBinary, &fromJSON} {
				require.Equal(t, known, roundTrip.ActualCostUsd != nil)
				if known {
					require.Zero(t, roundTrip.GetActualCostUsd())
				}
			}
			if known {
				require.Contains(t, string(jsonBytes), `"actualCostUsd":0`)
			} else {
				require.NotContains(t, string(jsonBytes), `actualCostUsd`)
			}
		})
	}
}

func TestMediaFailurePersistsKnownCostReturnedWithError(t *testing.T) {
	db := newMediaSchemaDB(t)
	zero := float64(0)
	executor := mediaExecutorFunc(func(context.Context, *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
		return &MediaExecutionResult{ActualCostUSD: &zero}, errors.New("local output write failed")
	})
	svc := NewMediaService(db, executor)
	submitted, err := svc.Submit(context.Background(), mediaRequest("cost-failure"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, err := svc.Get(context.Background(), submitted.GetExecutionId())
		return err == nil && terminalMediaStatus(got.GetStatus())
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), submitted.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, got.GetStatus())
	require.Equal(t, "executor_failed", got.GetErrorCode())
	require.NotNil(t, got.ActualCostUsd)
	require.Zero(t, got.GetActualCostUsd())
	require.Equal(t, 1, mediaKnownCost(t, db, got.GetExecutionId()))
}

func TestMediaResourceExecutorKeepsKnownCostWhenOutputWriteFails(t *testing.T) {
	db := newMediaSchemaDB(t)
	root := t.TempDir()
	blockedParent := filepath.Join(root, "file-not-directory")
	require.NoError(t, os.WriteFile(blockedParent, []byte("block"), 0o600))
	outputPath := filepath.Join(blockedParent, "image.png")
	runner := mediaCommandRunnerFunc(func(_ context.Context, command providers.Command) (providers.Result, error) {
		if len(command.Args) > 0 && command.Args[0] == "policy" {
			return providers.Result{Stdout: `{"role":"media.generate","model":"fixture/model"}`}, nil
		}
		return providers.Result{Stdout: `{"schema_version":1,"images":[{"data_b64":"aGVsbG8=","media_type":"image/png"}],"actual_cost_usd":0}`}, nil
	})
	executor := NewResourceOpenRouterMediaExecutor(runner)
	executor.resolveCredential = func(context.Context) (string, error) { return "fixture", nil }
	svc := NewMediaService(db, executor)
	request := mediaRequest("real-executor-output-failure")
	request.OutputReference = outputPath
	submitted, err := svc.Submit(context.Background(), request)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, err := svc.Get(context.Background(), submitted.GetExecutionId())
		return err == nil && terminalMediaStatus(got.GetStatus())
	}, time.Second, 10*time.Millisecond)
	got, err := svc.Get(context.Background(), submitted.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, got.GetStatus())
	require.Equal(t, "executor_failed", got.GetErrorCode())
	require.Empty(t, got.GetOutputs())
	require.NotNil(t, got.ActualCostUsd)
	require.Zero(t, got.GetActualCostUsd())
	require.Equal(t, 1, mediaKnownCost(t, db, got.GetExecutionId()))
}

type mediaCommandRunnerFunc func(context.Context, providers.Command) (providers.Result, error)

func (f mediaCommandRunnerFunc) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	return f(ctx, command)
}

func TestMediaEnsureSchemasAddsPresenceColumnWithoutMarkingOldZeroKnown(t *testing.T) {
	db := testdb.NewSQLite(t)
	oldSchema := strings.Replace(Schema(), "  actual_cost_known INTEGER NOT NULL DEFAULT 0,\n", "", 1)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return oldSchema })))
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)))
	_, err := db.Exec(`INSERT INTO media_executions(execution_id,idempotency_key,status,created_at,request_json,actual_cost_usd) VALUES('old','old',4,'now','{}',0)`)
	require.NoError(t, err)
	got, err := NewMediaService(db, nil).Get(context.Background(), "old")
	require.NoError(t, err)
	require.Nil(t, got.ActualCostUsd, "legacy numeric zero has no provider provenance")
}

func TestDecodeMediaImageRejectsBadEnvelopeAndRetainsCostInPartialResult(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCost bool
	}{
		{name: "unsupported version", body: `{"schema_version":2,"images":[{"data_b64":"aGVsbG8=","media_type":"image/png"}],"actual_cost_usd":0}`, wantCost: true},
		{name: "bad base64", body: `{"schema_version":1,"images":[{"data_b64":"bad!","media_type":"image/png"}],"actual_cost_usd":0}`, wantCost: true},
		{name: "malformed claimed cost", body: `{"schema_version":1,"images":[{"data_b64":"aGVsbG8=","media_type":"image/png"}],"actual_cost_usd":"0"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, cost, err := decodeMediaImage([]byte(tc.body))
			require.Error(t, err)
			require.Equal(t, tc.wantCost, cost != nil)
			if tc.wantCost {
				require.Zero(t, *cost)
			}
		})
	}
}

func mediaKnownCost(t *testing.T, db *sql.DB, id string) int {
	t.Helper()
	var known int
	require.NoError(t, db.QueryRow(`SELECT actual_cost_known FROM media_executions WHERE execution_id = ?`, id).Scan(&known))
	return known
}

func float64Pointer(value float64) *float64 { return &value }

func TestMediaRecoveryQuarantinesProviderCompletedWithoutReceipt(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "retained_request", true: "corrupt_request"}[corrupt], func(t *testing.T) {
			db, executor, request, id := mediaWithLostSuccessReceipt(t)
			if corrupt {
				_, err := db.Exec(`UPDATE media_executions SET request_json = '{' WHERE execution_id = ?`, id)
				require.NoError(t, err)
			}
			recovered := NewMediaService(db, executor)
			require.NoError(t, recovered.Recover(context.Background()))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			unknown, err := recovered.Wait(ctx, id)
			require.NoError(t, err)
			require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, unknown.GetStatus())
			require.Equal(t, "outcome_unknown", unknown.GetErrorCode())
			require.Contains(t, unknown.GetErrorMessage(), "cost unknown")
			require.EqualValues(t, 1, executor.calls.Load())
			got, err := recovered.Get(ctx, id)
			require.NoError(t, err)
			require.True(t, proto.Equal(unknown, got))
			got, err = recovered.Cancel(ctx, id)
			require.NoError(t, err)
			require.True(t, proto.Equal(unknown, got))
			require.Zero(t, executor.cancels.Load())
			_, err = recovered.Retry(ctx, id, "a-new-key")
			require.ErrorContains(t, err, "unknown")
			if !corrupt {
				got, err = recovered.Submit(ctx, request)
				require.NoError(t, err)
				require.True(t, proto.Equal(unknown, got))
				changed := proto.Clone(request).(*routingv1.SubmitMediaRequest)
				changed.Prompt = "a blue circle"
				_, err = recovered.Submit(ctx, changed)
				require.Error(t, err)
			} else {
				_, err = recovered.Submit(ctx, request)
				require.ErrorContains(t, err, "retained")
			}
			require.NoError(t, recovered.Recover(ctx))
			got, err = recovered.Get(ctx, id)
			require.NoError(t, err)
			require.True(t, proto.Equal(unknown, got))
			require.EqualValues(t, 1, executor.calls.Load())
		})
	}
}

type recordingMediaExecutor struct{ calls, cancels atomic.Int64 }

func (e *recordingMediaExecutor) Execute(ctx context.Context, request *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
	e.calls.Add(1)
	return successfulMediaExecutor{}.Execute(ctx, request)
}

func (e *recordingMediaExecutor) Cancel(context.Context, string) error {
	e.cancels.Add(1)
	return nil
}

// The fake executor completes its modeled provider side effect, then the real
// run path cannot commit success. This leaves a genuinely dispatched receipt.
func mediaWithLostSuccessReceipt(t *testing.T) (*sql.DB, *recordingMediaExecutor, *routingv1.SubmitMediaRequest, string) {
	t.Helper()
	db := newMediaSchemaDB(t)
	_, err := db.Exec(`CREATE TABLE media_receipt_write_attempts (execution_id TEXT);
		CREATE TRIGGER lose_media_result BEFORE UPDATE OF status ON media_executions
		WHEN NEW.status = 3 BEGIN
		  INSERT INTO media_receipt_write_attempts VALUES (NEW.execution_id);
		  SELECT RAISE(FAIL, 'controlled result commit failure');
		END;`)
	require.NoError(t, err)
	executor := &recordingMediaExecutor{}
	request := mediaRequest("lost-result")
	svc := NewMediaService(db, executor)
	receipt, err := svc.Submit(context.Background(), request)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var attempts int
		err := db.QueryRow(`SELECT count(*) FROM media_receipt_write_attempts WHERE execution_id = ?`, receipt.GetExecutionId()).Scan(&attempts)
		return err == nil && attempts == 1
	}, time.Second, time.Millisecond)
	got, err := svc.Get(context.Background(), receipt.GetExecutionId())
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING, got.GetStatus())
	require.EqualValues(t, 1, executor.calls.Load())
	_, err = db.Exec(`DROP TRIGGER lose_media_result`)
	require.NoError(t, err)
	return db, executor, request, receipt.GetExecutionId()
}

func TestMediaSubmissionBindsEveryImmutableField(t *testing.T) {
	mutations := map[string]func(*routingv1.SubmitMediaRequest){
		"kind": func(r *routingv1.SubmitMediaRequest) {
			r.Request.Kind = sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION
		},
		"missing_request": func(r *routingv1.SubmitMediaRequest) { r.Request = nil },
		"prompt":          func(r *routingv1.SubmitMediaRequest) { r.Prompt += " changed" },
		"request_id":      func(r *routingv1.SubmitMediaRequest) { r.Request.RequestId = "other" },
		"scenario":        func(r *routingv1.SubmitMediaRequest) { r.Request.Scenario = "other" },
		"role":            func(r *routingv1.SubmitMediaRequest) { r.Request.Role = "other" },
		"profile":         func(r *routingv1.SubmitMediaRequest) { r.Request.Profile = sharedv1.Profile_PROFILE_LOCAL_ONLY },
		"privacy": func(r *routingv1.SubmitMediaRequest) {
			r.Request.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
		},
		"budget":   func(r *routingv1.SubmitMediaRequest) { r.Request.MaxCostUsd = 1 },
		"metadata": func(r *routingv1.SubmitMediaRequest) { r.Request.Metadata = map[string]string{"extra": "value"} },
		"inputs": func(r *routingv1.SubmitMediaRequest) {
			r.Inputs = []*routingv1.MediaInput{{Reference: "different", MediaType: "image/png"}}
		},
		"count":  func(r *routingv1.SubmitMediaRequest) { r.OutputCount++ },
		"output": func(r *routingv1.SubmitMediaRequest) { r.OutputReference += ".other" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			svc := NewMediaService(newMediaSchemaDB(t), nil)
			request := mediaRequest(name)
			_, err := svc.Submit(context.Background(), request)
			require.NoError(t, err)
			changed := proto.Clone(request).(*routingv1.SubmitMediaRequest)
			mutate(changed)
			_, err = svc.Submit(context.Background(), changed)
			require.ErrorIs(t, err, ErrMediaIdempotencyConflict, "one key cannot select a different request")
		})
	}
}

func TestMediaConcurrentSubmissionDispatchesOnce(t *testing.T) {
	db := newMediaSchemaDB(t)
	executor := &recordingMediaExecutor{}
	svc := NewMediaService(db, executor)
	request := mediaRequest("concurrent")
	var wg sync.WaitGroup
	start := make(chan struct{})
	ids := make(chan string, 12)
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			execution, err := svc.Submit(context.Background(), proto.Clone(request).(*routingv1.SubmitMediaRequest))
			errs <- err
			ids <- execution.GetExecutionId()
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		require.Equal(t, first, id)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := svc.Wait(ctx, first)
	require.NoError(t, err)
	require.EqualValues(t, 1, executor.calls.Load())
}

func TestMediaRecoveryResumesOnlyQueuedReceipt(t *testing.T) {
	db := newMediaSchemaDB(t)
	request := mediaRequest("queued")
	payload, err := protojson.Marshal(request)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO media_executions(execution_id,idempotency_key,status,created_at,request_json) VALUES(?,?,?,?,?)`, "queued-id", request.GetIdempotencyKey(), 1, time.Now().UTC().Format(time.RFC3339Nano), string(payload))
	require.NoError(t, err)
	executor := &recordingMediaExecutor{}
	svc := NewMediaService(db, executor)
	require.NoError(t, svc.Recover(context.Background()))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	receipt, err := svc.Wait(ctx, "queued-id")
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED, receipt.GetStatus())
	require.EqualValues(t, 1, executor.calls.Load())
}

func TestMediaSubmissionBindsOrderedInputsAndNormalizesOnlyKey(t *testing.T) {
	svc := NewMediaService(newMediaSchemaDB(t), nil)
	request := mediaRequest("  semantic-key  ")
	request.Request.Metadata = map[string]string{"a": "1", "b": "2"}
	request.Inputs = []*routingv1.MediaInput{{Reference: "first", MediaType: "image/png"}, {Reference: "second", MediaType: "image/png"}}
	first, err := svc.Submit(context.Background(), request)
	require.NoError(t, err)
	replay := proto.Clone(request).(*routingv1.SubmitMediaRequest)
	replay.IdempotencyKey = "semantic-key"
	replay.Request.Metadata = map[string]string{"b": "2", "a": "1"}
	got, err := svc.Submit(context.Background(), replay)
	require.NoError(t, err)
	require.Equal(t, first.GetExecutionId(), got.GetExecutionId())
	replay.Inputs[0], replay.Inputs[1] = replay.Inputs[1], replay.Inputs[0]
	_, err = svc.Submit(context.Background(), replay)
	require.ErrorIs(t, err, ErrMediaIdempotencyConflict)
}

func TestMediaInsertRaceFallbackChecksRetainedIdentity(t *testing.T) {
	for _, variant := range []string{"same", "changed", "corrupt"} {
		t.Run(variant, func(t *testing.T) {
			db := newMediaSchemaDB(t)
			request := mediaRequest("race-key")
			retained := proto.Clone(request).(*routingv1.SubmitMediaRequest)
			if variant == "changed" {
				retained.Prompt = "another request"
			}
			payload, err := protojson.Marshal(retained)
			require.NoError(t, err)
			if variant == "corrupt" {
				payload = []byte("{")
			}
			_, err = db.Exec(`CREATE TABLE media_race_fixture (payload TEXT)`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO media_race_fixture VALUES (?)`, string(payload))
			require.NoError(t, err)
			// A winner survives the failed insert, deterministically exercising
			// the same retained-row check as a unique-insert race.
			_, err = db.Exec(`CREATE TRIGGER media_insert_race BEFORE INSERT ON media_executions
			 WHEN NEW.execution_id != 'race-winner' BEGIN
			 INSERT INTO media_executions(execution_id,idempotency_key,status,created_at,request_json)
			 VALUES('race-winner',NEW.idempotency_key,1,NEW.created_at,(SELECT payload FROM media_race_fixture));
			 SELECT RAISE(FAIL, 'competing insert'); END;`)
			require.NoError(t, err)
			got, err := NewMediaService(db, nil).Submit(context.Background(), request)
			switch variant {
			case "same":
				require.NoError(t, err)
				require.Equal(t, "race-winner", got.GetExecutionId())
			case "changed":
				require.ErrorIs(t, err, ErrMediaIdempotencyConflict)
			case "corrupt":
				require.ErrorIs(t, err, ErrMediaRetainedSubmissionInvalid)
			}
		})
	}
}

func TestMediaQuarantineWriteFailureDoesNotDispatch(t *testing.T) {
	db, executor, _, id := mediaWithLostSuccessReceipt(t)
	_, err := db.Exec(`CREATE TRIGGER fail_quarantine BEFORE UPDATE ON media_executions
	 WHEN NEW.error_code = 'outcome_unknown' BEGIN SELECT RAISE(FAIL, 'quarantine unavailable'); END;`)
	require.NoError(t, err)
	svc := NewMediaService(db, executor)
	require.ErrorContains(t, svc.Recover(context.Background()), "quarantine")
	got, err := svc.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING, got.GetStatus())
	require.EqualValues(t, 1, executor.calls.Load())
	_, err = db.Exec(`DROP TRIGGER fail_quarantine`)
	require.NoError(t, err)
	require.NoError(t, svc.Recover(context.Background()))
	got, err = svc.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", got.GetErrorCode())
	require.EqualValues(t, 1, executor.calls.Load())
}

type mediaExecutorFunc func(context.Context, *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error)

func (f mediaExecutorFunc) Execute(ctx context.Context, request *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
	return f(ctx, request)
}
func (mediaExecutorFunc) Cancel(context.Context, string) error { return nil }

func TestMediaStaleCompletionCannotReplaceUnknownOutcome(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[failed], func(t *testing.T) {
			db := newMediaSchemaDB(t)
			svc := NewMediaService(db, nil)
			request := mediaRequest("stale")
			payload, err := protojson.Marshal(request)
			require.NoError(t, err)
			require.NoError(t, svc.insert(context.Background(), &routingv1.MediaExecution{ExecutionId: "stale-id", IdempotencyKey: request.GetIdempotencyKey(), Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, string(payload)))
			svc.executor = mediaExecutorFunc(func(ctx context.Context, request *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
				// Model a completion arriving after startup has quarantined the
				// receipt. run's real conditional result write must lose.
				require.NoError(t, svc.Recover(ctx))
				if failed {
					return nil, errors.New("late executor failure")
				}
				return successfulMediaExecutor{}.Execute(ctx, request)
			})
			svc.run(context.Background(), "stale-id", request)
			got, err := svc.Get(context.Background(), "stale-id")
			require.NoError(t, err)
			require.Equal(t, "outcome_unknown", got.GetErrorCode())
			require.Empty(t, got.GetOutputs())
			require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED, got.GetStatus())
		})
	}
}

func TestMediaCancellationCannotResumeOrDispatch(t *testing.T) {
	db := newMediaSchemaDB(t)
	executor := &recordingMediaExecutor{}
	svc := NewMediaService(db, executor)
	request := mediaRequest("cancelled-before-dispatch")
	payload, err := protojson.Marshal(request)
	require.NoError(t, err)
	require.NoError(t, svc.insert(context.Background(), &routingv1.MediaExecution{ExecutionId: "cancelled-id", IdempotencyKey: request.GetIdempotencyKey(), Status: routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, string(payload)))
	got, err := svc.Cancel(context.Background(), "cancelled-id")
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED, got.GetStatus())
	require.NoError(t, svc.Recover(context.Background()))
	svc.run(context.Background(), "cancelled-id", request)
	got, err = svc.Wait(context.Background(), "cancelled-id")
	require.NoError(t, err)
	require.Equal(t, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED, got.GetStatus())
	require.Zero(t, executor.calls.Load())
}

func mediaRequest(key string) *routingv1.SubmitMediaRequest {
	return &routingv1.SubmitMediaRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION,
			Role:         "media.generate",
			Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
			Scenario:     "fixture",
			Operation:    "generate",
		},
		Prompt:          "a red square",
		OutputCount:     1,
		IdempotencyKey:  key,
		OutputReference: filepath.Join(os.TempDir(), "ai-gateway-media-test-output.png"),
	}
}

func newMediaSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)))
	return db
}
