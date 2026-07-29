package channelmanager

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	assetstudio "channel-manager/integrations/assetstudio"
	contentdesk "channel-manager/integrations/contentdesk"
	core "channel-manager/internal/channelmanager"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	channelmanagerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager"
	channelmanagerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager/channelmanager_v1connect"
	_ "modernc.org/sqlite"
)

func testFormats() []core.Format {
	return []core.Format{{Kind: "test", MIMETypes: []string{"application/test"}, MaxBytes: 1, MaxDurationSecs: 1, MinWidth: 1, MinHeight: 1, MaxWidth: 1, MaxHeight: 1}}
}

type deliveryStub struct {
	release contentdesk.ReleaseOutcome
	metric  contentdesk.MetricSample
	err     error
}

type browserStub struct {
	id  string
	err error
}

type assetResolverStub struct {
	reference assetstudio.Reference
	err       error
	calls     []string
}

func (s *assetResolverStub) ResolveReleasedAsset(_ context.Context, assetID string) (assetstudio.Reference, error) {
	s.calls = append(s.calls, assetID)
	return s.reference, s.err
}

func (b browserStub) Dispatch(context.Context, string, string, string) (string, []string, error) {
	return b.id, nil, b.err
}

func (d *deliveryStub) DeliverRelease(_ context.Context, outcome contentdesk.ReleaseOutcome) error {
	d.release = outcome
	return d.err
}
func (d *deliveryStub) DeliverMetric(_ context.Context, sample contentdesk.MetricSample) error {
	d.metric = sample
	return d.err
}

// [REQ:CHANMGR-P0-013] [REQ:CHANMGR-P0-014] Content Desk's typed seam
// returns fail-closed eligibility and a replay-safe release receipt.
func TestEligibilityAndReleaseOverConnect(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"publish"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:connect-release?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := channelmanagerconnect.NewChannelManagerServiceClient(http.DefaultClient, server.URL)
	eligibility, err := client.GetEligibility(t.Context(), connect.NewRequest(&channelmanagerv1.GetEligibilityRequest{IdentityId: "active", Lane: "main"}))
	if err != nil || eligibility.Msg.Eligibility != "eligible" {
		t.Fatalf("eligibility=%v err=%v", eligibility, err)
	}
	first, err := client.SubmitRelease(t.Context(), connect.NewRequest(&channelmanagerv1.SubmitReleaseRequest{IdentityId: "active", Lane: "main", DraftId: "draft-connect", IdempotencyKey: "connect-key"}))
	if err != nil || first.Msg.Receipt.ActionId == "" {
		t.Fatalf("release=%v err=%v", first, err)
	}
	second, err := client.SubmitRelease(t.Context(), connect.NewRequest(&channelmanagerv1.SubmitReleaseRequest{IdentityId: "active", Lane: "main", DraftId: "draft-connect", IdempotencyKey: "connect-key"}))
	if err != nil || second.Msg.Receipt.Id != first.Msg.Receipt.Id {
		t.Fatalf("retry=%v err=%v", second, err)
	}
}

// [REQ:CHANMGR-P1-003] A release carrying asset IDs must verify each released
// Asset Studio reference. An unavailable or mismatched lookup never releases.
func TestSubmitReleaseFailsClosedWhenAssetVerificationCannotBeProven(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"publish"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:asset-release?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	resolver := &assetResolverStub{err: errors.New("asset studio unavailable")}
	router := mux.NewRouter()
	moduleWithAssetResolver(service, core.NewStore(db), nil, nil, resolver).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := channelmanagerconnect.NewChannelManagerServiceClient(http.DefaultClient, server.URL)
	request := connect.NewRequest(&channelmanagerv1.SubmitReleaseRequest{IdentityId: "active", Lane: "main", DraftId: "draft-asset", IdempotencyKey: "asset-key", AssetIds: []string{"asset-1"}})
	if _, err = client.SubmitRelease(t.Context(), request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unverified release error=%v", err)
	}
	resolver.err = nil
	resolver.reference = assetstudio.Reference{ID: "asset-1"}
	response, err := client.SubmitRelease(t.Context(), request)
	if err != nil || response.Msg.Receipt.ActionId == "" || len(resolver.calls) != 2 {
		t.Fatalf("verified release=%v err=%v calls=%v", response, err, resolver.calls)
	}
}

// [REQ:CHANMGR-P0-001] [REQ:CHANMGR-P0-015] [REQ:CHANMGR-P0-016] The
// operator transport creates an identity, exposes queued work, accepts manual
// completion evidence, and persists the resulting overview state.
func TestManualWorkflowOverHTTP(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"engage", "publish"}, Formats: testFormats()}}, []core.Program{{ID: "warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []core.Phase{{ID: "p", Allowed: []string{"engage"}}}, Provenance: core.Provenance{SourceKind: "operator", Confidence: "speculative", CapturedAt: "today", RevisitTrigger: "five runs", Sources: []string{"manual"}}}})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:handler?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	identity := map[string]any{"id": "x-1", "platform_id": "x", "purpose": "brand", "environment_ref": "device", "vault_ref": "vault://channel/x", "attestations": map[string]bool{"region": true}}
	if got := call(http.MethodPost, "/api/v1/channel-manager/identities", identity).Code; got != http.StatusCreated {
		t.Fatalf("create=%d", got)
	}
	if got := call(http.MethodPost, "/api/v1/channel-manager/identities/x-1/start", map[string]string{"program_id": "warm"}).Code; got != http.StatusOK {
		t.Fatalf("start=%d", got)
	}
	action := call(http.MethodPost, "/api/v1/channel-manager/actions", map[string]any{"identity_id": "x-1", "kind": "engage", "seed": 4})
	if action.Code != http.StatusCreated {
		t.Fatalf("enqueue=%d: %s", action.Code, action.Body.String())
	}
	var result core.Action
	_ = json.Unmarshal(action.Body.Bytes(), &result)
	if got := call(http.MethodPost, "/api/v1/channel-manager/actions/"+result.ID+"/complete", map[string]string{"evidence": "https://proof"}).Code; got != http.StatusOK {
		t.Fatalf("complete=%d", got)
	}
	observation := call(http.MethodPost, "/api/v1/channel-manager/identities/x-1/observations", map[string]any{"metric": "reach", "value": 120})
	if observation.Code != http.StatusCreated {
		t.Fatalf("observation=%d: %s", observation.Code, observation.Body.String())
	}
	overview := call(http.MethodGet, "/api/v1/channel-manager/overview", nil)
	if overview.Code != http.StatusOK || !strings.Contains(overview.Body.String(), "x-1") {
		t.Fatalf("overview=%d %s", overview.Code, overview.Body.String())
	}
}

// [REQ:CHANMGR-P0-013] [REQ:CHANMGR-P0-014] Eligibility never fails open and
// release retries return the original record rather than creating another one.
func TestEligibilityAndIdempotentReleaseOverHTTP(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"publish"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:release?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	unknown := call(http.MethodGet, "/api/v1/channel-manager/identities/missing/eligibility?lane=main", nil)
	if unknown.Code != http.StatusOK || !strings.Contains(unknown.Body.String(), "unknown") {
		t.Fatalf("unknown eligibility=%s", unknown.Body.String())
	}
	first := call(http.MethodPost, "/api/v1/channel-manager/releases", map[string]string{"identity_id": "active", "lane": "main", "draft_id": "draft-1", "idempotency_key": "release-1"})
	if first.Code != http.StatusCreated {
		t.Fatalf("first=%s", first.Body.String())
	}
	second := call(http.MethodPost, "/api/v1/channel-manager/releases", map[string]string{"identity_id": "active", "lane": "main", "draft_id": "draft-1", "idempotency_key": "release-1"})
	if second.Code != http.StatusCreated || first.Body.String() != second.Body.String() {
		t.Fatalf("retry must return original: %s / %s", first.Body.String(), second.Body.String())
	}
	var receipt core.ReleaseReceipt
	if err := json.Unmarshal(first.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	completed := call(http.MethodPost, "/api/v1/channel-manager/actions/"+receipt.ActionID+"/complete-release", map[string]string{"platform_post_id": "post-1", "published_url": "https://example.test/post-1", "first_comment_status": "succeeded"})
	if completed.Code != http.StatusOK || !strings.Contains(completed.Body.String(), "https://example.test/post-1") {
		t.Fatalf("manual release completion=%d %s", completed.Code, completed.Body.String())
	}
}

// [REQ:CHANMGR-P1-009] The operator transport returns a descriptor-derived
// preview and reports an invisible required disclosure as blocking.
func TestReleasePreviewOverHTTP(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 1, CaptionLimit: 3, ActionKinds: []string{"publish"}, DisclosureRequired: true, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:preview?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	body, _ := json.Marshal(map[string]any{"platform_id": "x", "caption": "hello", "format_kind": "test", "media_width": 1, "media_height": 1, "disclosure_visible": false})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channel-manager/releases/preview", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "required disclosure") || !strings.Contains(w.Body.String(), "hel") {
		t.Fatalf("preview=%d %s", w.Code, w.Body.String())
	}
}

// [REQ:CHANMGR-P1-001] The generated operator contract persists one opaque
// profile reference, then dispatches only a durable due action to BAS.
func TestBrowserAutomationOverConnect(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"engage"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	action, err := service.Enqueue("active", "engage", time.Now().UTC(), 1, "browser-action")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:browser-connect?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	moduleWithDependencies(service, core.NewStore(db), nil, browserStub{id: "bas-execution-1"}).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := channelmanagerconnect.NewChannelManagerServiceClient(http.DefaultClient, server.URL)
	if _, err = client.AssignAutomation(t.Context(), connect.NewRequest(&channelmanagerv1.AssignAutomationRequest{IdentityId: "active", SessionProfileRef: "profile-active", WorkflowRef: "workflow-active", EnabledActionKinds: []string{"engage"}, OperatorNote: "synthetic acceptance"})); err != nil {
		t.Fatal(err)
	}
	dispatch, err := client.DispatchBrowserAction(t.Context(), connect.NewRequest(&channelmanagerv1.DispatchBrowserActionRequest{ActionId: action.ID}))
	if err != nil || dispatch.Msg.ExecutionId != "bas-execution-1" {
		t.Fatalf("dispatch=%v err=%v", dispatch, err)
	}
}

// [REQ:CHANMGR-P1-007]
func TestDeliveryAcknowledgesDurableReleaseAndMetric(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"publish"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Release("active", "main", "draft-delivery", "delivery-key", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CompleteRelease(receipt.ActionID, "post-delivery", "https://example.test/post-delivery", "succeeded", "", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RecordMetric(receipt.ID, "sample-delivery", "impressions", 12, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:delivery?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	deliverer := &deliveryStub{}
	router := mux.NewRouter()
	moduleWithDeliverer(service, core.NewStore(db), deliverer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := channelmanagerconnect.NewChannelManagerServiceClient(http.DefaultClient, server.URL)
	if _, err = client.DeliverReleaseOutcome(t.Context(), connect.NewRequest(&channelmanagerv1.DeliverReleaseOutcomeRequest{ReleaseId: receipt.ID})); err != nil {
		t.Fatal(err)
	}
	if _, err = client.DeliverMetricSample(t.Context(), connect.NewRequest(&channelmanagerv1.DeliverMetricSampleRequest{SampleId: "sample-delivery"})); err != nil {
		t.Fatal(err)
	}
	if deliverer.release.ReceiptID != receipt.ID || deliverer.metric.ID != "sample-delivery" {
		t.Fatalf("deliveries=%#v/%#v", deliverer.release, deliverer.metric)
	}
	if receipt.DeliveryStatus != "delivered" || service.MetricSamples["sample-delivery"].DeliveryStatus != "acknowledged" {
		t.Fatalf("delivery state=%#v/%#v", receipt, service.MetricSamples["sample-delivery"])
	}
}
