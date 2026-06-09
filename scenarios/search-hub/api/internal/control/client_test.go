package control_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"search-hub/internal/control"

	"github.com/vrooli/api-core/retry"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// --- fakes -----------------------------------------------------------------

type fakeResolver struct {
	url    string
	err    error
	lastID string
	calls  int
}

func (f *fakeResolver) ResolveScenarioURL(_ context.Context, id string) (string, error) {
	f.calls++
	f.lastID = id
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

// fakeControlClient implements controlconnect.SearchControlServiceClient. It
// returns the queued errors first (one per call), then canned successes, and
// records the last request of each kind for assertions.
type fakeControlClient struct {
	errs         []error
	reindexN     int
	statusN      int
	cancelN      int
	writeN       int
	writeCorpusN int
	lastReindex  *controlv1.ReindexRequest
	lastWrite    *controlv1.WriteConfigRequest
	lastCorpus   *controlv1.WriteCorpusRequest
}

func (f *fakeControlClient) next() error {
	if len(f.errs) == 0 {
		return nil
	}
	e := f.errs[0]
	f.errs = f.errs[1:]
	return e
}

func (f *fakeControlClient) Reindex(_ context.Context, req *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error) {
	f.reindexN++
	f.lastReindex = req.Msg
	if err := f.next(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.ReindexResponse{JobId: "job-42", PlannedUpserts: 5, DryRun: req.Msg.GetDryRun()}), nil
}

func (f *fakeControlClient) ReindexStatus(_ context.Context, req *connect.Request[controlv1.ReindexStatusRequest]) (*connect.Response[controlv1.ReindexStatusResponse], error) {
	f.statusN++
	if err := f.next(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.ReindexStatusResponse{JobId: req.Msg.GetJobId(), State: "succeeded"}), nil
}

func (f *fakeControlClient) ReindexCancel(_ context.Context, req *connect.Request[controlv1.ReindexCancelRequest]) (*connect.Response[controlv1.ReindexCancelResponse], error) {
	f.cancelN++
	if err := f.next(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.ReindexCancelResponse{JobId: req.Msg.GetJobId(), Cancelled: true}), nil
}

func (f *fakeControlClient) WriteConfig(_ context.Context, req *connect.Request[controlv1.WriteConfigRequest]) (*connect.Response[controlv1.WriteConfigResponse], error) {
	f.writeN++
	f.lastWrite = req.Msg
	if err := f.next(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.WriteConfigResponse{Written: true, ReindexTriggered: true, ReindexJobId: "job-99"}), nil
}

func (f *fakeControlClient) WriteCorpus(_ context.Context, req *connect.Request[controlv1.WriteCorpusRequest]) (*connect.Response[controlv1.WriteCorpusResponse], error) {
	f.writeCorpusN++
	f.lastCorpus = req.Msg
	if err := f.next(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&controlv1.WriteCorpusResponse{Written: true, Effective: req.Msg.GetCorpus()}), nil
}

// --- helpers ---------------------------------------------------------------

func descriptorWithControl() *registryv1.ProviderDescriptor {
	ep := func(path string) *registryv1.Endpoint {
		return &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId: "cli-health", Path: path, Method: registryv1.HttpMethod_HTTP_METHOD_POST,
		}}}
	}
	return &registryv1.ProviderDescriptor{
		ProviderId:      "cli-health.commands",
		ProviderGroup:   "cli-health",
		ReindexEndpoint: ep("/vrooli.search_hub.v1.control.SearchControlService/Reindex"),
		ConfigEndpoint:  ep("/vrooli.search_hub.v1.control.SearchControlService/WriteConfig"),
	}
}

func noSleepRetry(maxAttempts int) retry.Config {
	return retry.Config{MaxAttempts: maxAttempts, Sleeper: func(time.Duration) {}}
}

// --- tests -----------------------------------------------------------------

func TestReindexResolvesAndCalls(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{}
	c := control.NewClient(res, control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }))

	resp, err := c.Reindex(context.Background(), descriptorWithControl(), "tok", "web-console", true)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if res.lastID != "cli-health" {
		t.Fatalf("resolved scenario = %q, want cli-health", res.lastID)
	}
	if fc.reindexN != 1 {
		t.Fatalf("reindex called %d times, want 1", fc.reindexN)
	}
	if fc.lastReindex.GetControlToken() != "tok" || fc.lastReindex.GetScope() != "web-console" || !fc.lastReindex.GetDryRun() {
		t.Fatalf("request not threaded: %+v", fc.lastReindex)
	}
	if resp.GetJobId() != "job-42" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteConfigUsesConfigEndpoint(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{}
	c := control.NewClient(res, control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }))

	tuning := &registryv1.Tuning{Engine: "dense", RerankEnabled: true}
	resp, err := c.WriteConfig(context.Background(), descriptorWithControl(), "tok", tuning, false)
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if fc.writeN != 1 {
		t.Fatalf("write called %d times, want 1", fc.writeN)
	}
	if fc.lastWrite.GetProviderId() != "cli-health.commands" || fc.lastWrite.GetControlToken() != "tok" {
		t.Fatalf("write request not threaded: %+v", fc.lastWrite)
	}
	if fc.lastWrite.GetTuning().GetEngine() != "dense" {
		t.Fatalf("tuning not forwarded")
	}
	if !resp.GetReindexTriggered() || resp.GetReindexJobId() != "job-99" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteCorpusUsesConfigEndpoint(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{}
	c := control.NewClient(res, control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }))

	corpus := &evalv1.EvalSuite{SuiteId: "cli-health.commands.primary", Cases: []*evalv1.EvalCase{{CaseId: "c1", Query: "q"}}}
	resp, err := c.WriteCorpus(context.Background(), descriptorWithControl(), "tok", corpus, false)
	if err != nil {
		t.Fatalf("WriteCorpus: %v", err)
	}
	if fc.writeCorpusN != 1 {
		t.Fatalf("WriteCorpus called %d times, want 1", fc.writeCorpusN)
	}
	if fc.lastCorpus.GetProviderId() != "cli-health.commands" || fc.lastCorpus.GetControlToken() != "tok" {
		t.Fatalf("corpus request not threaded: %+v", fc.lastCorpus)
	}
	if got := fc.lastCorpus.GetCorpus(); got == nil || len(got.GetCases()) != 1 {
		t.Fatalf("corpus not forwarded: %+v", fc.lastCorpus)
	}
	if !resp.GetWritten() || len(resp.GetEffective().GetCases()) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteCorpusNoControlPlane(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{}
	c := control.NewClient(res, control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }))

	// A descriptor with no config_endpoint cannot accept a corpus write-back.
	d := &registryv1.ProviderDescriptor{ProviderId: "x.leaf"}
	_, err := c.WriteCorpus(context.Background(), d, "tok", &evalv1.EvalSuite{}, false)
	if !errors.Is(err, control.ErrNoControlPlane) {
		t.Fatalf("want ErrNoControlPlane, got %v", err)
	}
	if fc.writeCorpusN != 0 {
		t.Fatalf("client must not be called without a control endpoint")
	}
}

func TestNoControlPlane(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{}
	c := control.NewClient(res, control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }))

	// A descriptor with no reindex_endpoint is not sweep-tunable.
	d := &registryv1.ProviderDescriptor{ProviderId: "x.leaf"}
	_, err := c.Reindex(context.Background(), d, "tok", "", false)
	if !errors.Is(err, control.ErrNoControlPlane) {
		t.Fatalf("want ErrNoControlPlane, got %v", err)
	}
	if res.calls != 0 {
		t.Fatalf("resolver must not be called without a control endpoint")
	}
}

func TestPermanentErrorNotRetried(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{errs: []error{
		connect.NewError(connect.CodePermissionDenied, errors.New("bad token")),
		connect.NewError(connect.CodePermissionDenied, errors.New("bad token")),
	}}
	c := control.NewClient(res,
		control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }),
		control.WithRetry(noSleepRetry(5)),
	)
	_, err := c.Reindex(context.Background(), descriptorWithControl(), "tok", "", false)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("want PermissionDenied, got %v", connect.CodeOf(err))
	}
	if fc.reindexN != 1 {
		t.Fatalf("permanent error must not retry: called %d times", fc.reindexN)
	}
}

func TestTransientErrorRetriedThenSucceeds(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{errs: []error{
		connect.NewError(connect.CodeUnavailable, errors.New("down")),
		connect.NewError(connect.CodeUnavailable, errors.New("down")),
		// third attempt succeeds
	}}
	c := control.NewClient(res,
		control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }),
		control.WithRetry(noSleepRetry(5)),
	)
	resp, err := c.Reindex(context.Background(), descriptorWithControl(), "tok", "", false)
	if err != nil {
		t.Fatalf("expected success after transient retries, got %v", err)
	}
	if fc.reindexN != 3 {
		t.Fatalf("expected 3 attempts (2 transient + 1 ok), got %d", fc.reindexN)
	}
	if resp.GetJobId() != "job-42" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestTransientErrorExhaustsAttempts(t *testing.T) {
	res := &fakeResolver{url: "http://localhost:1234"}
	fc := &fakeControlClient{errs: []error{
		connect.NewError(connect.CodeUnavailable, errors.New("down")),
		connect.NewError(connect.CodeUnavailable, errors.New("down")),
		connect.NewError(connect.CodeUnavailable, errors.New("down")),
	}}
	c := control.NewClient(res,
		control.WithClientFactory(func(string) controlconnect.SearchControlServiceClient { return fc }),
		control.WithRetry(noSleepRetry(3)),
	)
	_, err := c.Reindex(context.Background(), descriptorWithControl(), "tok", "", false)
	if err == nil {
		t.Fatalf("expected failure after exhausting attempts")
	}
	if fc.reindexN != 3 {
		t.Fatalf("expected 3 attempts, got %d", fc.reindexN)
	}
}
