package designcapture

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	timelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func producerTimeline(t *testing.T) *timelinev1.ExecutionTimeline {
	t.Helper()
	raw, err := os.ReadFile("testdata/bas-target-timeline.json")
	require.NoError(t, err)
	var timeline timelinev1.ExecutionTimeline
	require.NoError(t, protojson.Unmarshal(raw, &timeline))
	return &timeline
}
func TestBASCaptureEvidenceRequiresExactSuccessfulTarget(t *testing.T) {
	op := Operation{State: Running, ProducerID: "bas-fixture", Version: 1, Request: captureRequest()}
	observation, err := observeTimeline(op, producerTimeline(t))
	require.NoError(t, err)
	require.Equal(t, Completed, observation.State)
	require.Len(t, observation.Artifacts, 2)
	require.Len(t, observation.Artifacts[1].Evidence.Regions, 5)
	for _, test := range []struct {
		name   string
		mutate func(*timelinev1.ExecutionTimeline)
	}{
		{"wrong producer", func(v *timelinev1.ExecutionTimeline) { v.ExecutionId = "other" }},
		{"failed step despite terminal completion", func(v *timelinev1.ExecutionTimeline) { v.Entries[1].Context.Success = new(bool) }},
		{"missing screenshot", func(v *timelinev1.ExecutionTimeline) { v.Entries[2].Telemetry = nil }},
		{"missing target", func(v *timelinev1.ExecutionTimeline) { v.Entries[2].Aggregates = nil }},
		{"wrong render", func(v *timelinev1.ExecutionTimeline) {
			f := v.Entries[2].Aggregates.Artifacts[0].Payload["value"].GetObjectValue().Fields["result"].GetObjectValue().Fields
			f["renderHash"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "stale"}}
		}},
		{"wrong viewport", func(v *timelinev1.ExecutionTimeline) {
			f := v.Entries[2].Aggregates.Artifacts[0].Payload["value"].GetObjectValue().Fields["result"].GetObjectValue().Fields["viewport"].GetObjectValue().Fields
			f["width"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: 1440}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			v := producerTimeline(t)
			test.mutate(v)
			_, err := observeTimeline(op, v)
			require.Error(t, err)
		})
	}
}

type observerFunc func(context.Context, Operation) (Observation, error)

func (f observerFunc) Observe(ctx context.Context, op Operation) (Observation, error) {
	return f(ctx, op)
}
func TestAttachPersistsReceiptAndNeverRedispatches(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	service := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) { return "bas-fixture", nil })}
	op, err := service.Start(ctx, "attach", captureRequest())
	require.NoError(t, err)
	_, err = service.Attach(ctx, op.ID, observerFunc(func(context.Context, Operation) (Observation, error) {
		return Observation{}, errors.New("temporary observation failure")
	}))
	require.Error(t, err)
	still, err := repo.Get(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, op.Version, still.Version)
	require.Equal(t, Running, still.State)
	calls := 0
	observer := observerFunc(func(_ context.Context, op Operation) (Observation, error) {
		calls++
		return observeTimeline(op, producerTimeline(t))
	})
	done, err := service.Attach(ctx, op.ID, observer)
	require.NoError(t, err)
	require.Equal(t, Completed, done.State)
	again, err := service.Attach(ctx, op.ID, observer)
	require.NoError(t, err)
	require.Equal(t, done, again)
	require.Equal(t, 1, calls)
}
func TestBASObserverReadsExistingProducerOnly(t *testing.T) {
	raw, err := os.ReadFile("testdata/bas-target-timeline.json")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/browser_automation_studio.v1.ExecutionsService/GetExecutionTimeline", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer server.Close()
	result, err := (BASDispatcher{BASBaseURL: server.URL}).Observe(context.Background(), Operation{State: Running, ProducerID: "bas-fixture", Version: 1, Request: captureRequest()})
	require.NoError(t, err)
	require.Equal(t, Completed, result.State)
}
