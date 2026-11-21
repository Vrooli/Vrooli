package vision_navigation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/services/vision"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// ---- Test doubles --------------------------------------------------------

type fakeNavigator struct {
	navType            vision.NavigatorType
	available          bool
	unavailableReason  string
	creditPolicy       vision.CreditPolicy
	clientSourcePolicy vision.ClientSourcePolicy
	description        string
	navigateID         string
	navigateErr        error
	lastRequest        vision.NavigationRequest
}

type fakeHandle struct {
	id, sessionID string
}

func (h *fakeHandle) ID() string                      { return h.id }
func (h *fakeHandle) SessionID() string               { return h.sessionID }
func (h *fakeHandle) Status() vision.NavigationStatus { return vision.StatusNavigating }
func (h *fakeHandle) Wait(context.Context) error      { return nil }
func (h *fakeHandle) Abort(context.Context) error     { return nil }
func (h *fakeHandle) Resume(context.Context) error    { return nil }

func (f *fakeNavigator) Navigate(_ context.Context, req vision.NavigationRequest) (vision.NavigationHandle, error) {
	f.lastRequest = req
	if f.navigateErr != nil {
		return nil, f.navigateErr
	}
	return &fakeHandle{id: f.navigateID, sessionID: "s-1"}, nil
}
func (f *fakeNavigator) CreditPolicy() vision.CreditPolicy             { return f.creditPolicy }
func (f *fakeNavigator) ClientSourcePolicy() vision.ClientSourcePolicy { return f.clientSourcePolicy }
func (f *fakeNavigator) Type() vision.NavigatorType                    { return f.navType }
func (f *fakeNavigator) IsAvailable(context.Context) bool              { return f.available }
func (f *fakeNavigator) Description() string                           { return f.description }
func (f *fakeNavigator) UnavailableReason(context.Context) string      { return f.unavailableReason }

type fakeTracker struct {
	session    *vision.NavigationSession
	exists     bool
	abortErr   error
	resumeErr  error
	lastAbort  string
	lastResume string
}

func (f *fakeTracker) GetSession(navigationID string) (*vision.NavigationSession, bool) {
	if !f.exists {
		return nil, false
	}
	return f.session, true
}

func (f *fakeTracker) AbortNavigation(_ context.Context, navigationID string) error {
	f.lastAbort = navigationID
	return f.abortErr
}

func (f *fakeTracker) ResumeNavigation(_ context.Context, navigationID string) error {
	f.lastResume = navigationID
	return f.resumeErr
}

// ---- Helpers -------------------------------------------------------------

func newTestRegistry(t *testing.T, navs ...vision.VisionNavigator) *vision.NavigatorRegistry {
	t.Helper()
	r := vision.NewNavigatorRegistry()
	for _, n := range navs {
		r.Register(n)
	}
	return r
}

func newTestClient(t *testing.T, d Deps) aiconnect.VisionNavigationServiceClient {
	t.Helper()
	if d.Logger == nil {
		l := logrus.New()
		l.SetOutput(testWriter{t})
		d.Logger = l
	}
	mount := Module(d)
	srv := testutil.StartConnectServer(t, mount.Path, mount.Handler)
	return aiconnect.NewVisionNavigationServiceClient(srv.Client(), srv.URL)
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

// ---- ListNavigators ------------------------------------------------------

func TestListNavigators_HappyPath(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		description:        "Playwright vision",
		clientSourcePolicy: vision.AllSourcesPolicy(),
		creditPolicy: vision.CreditPolicy{
			RequiresCredits: true,
			CreditsPerStep:  2,
		},
	}
	registry := newTestRegistry(t, nav)
	client := newTestClient(t, Deps{Registry: registry})

	resp, err := client.ListNavigators(context.Background(), connect.NewRequest(&aiv1.ListNavigatorsRequest{ClientSource: "ui"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Navigators, 1)
	require.Equal(t, "playwright", resp.Msg.Navigators[0].Type)
	require.True(t, resp.Msg.Navigators[0].Available)
	require.Equal(t, "playwright", resp.Msg.Default)
	require.Equal(t, int32(2), resp.Msg.Navigators[0].CreditPolicy.CreditsPerStep)
}

func TestListNavigators_EmptyRegistry(t *testing.T) {
	registry := newTestRegistry(t)
	client := newTestClient(t, Deps{Registry: registry})

	resp, err := client.ListNavigators(context.Background(), connect.NewRequest(&aiv1.ListNavigatorsRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.Navigators)
	require.Equal(t, "", resp.Msg.Default)
}

// ---- StartNavigation -----------------------------------------------------

func TestStartNavigation_HappyPath(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		navigateID:         "nav-123",
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})

	resp, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1",
		Prompt:    "Click the login button",
		Model:     "gpt-4o",
		MaxSteps:  10,
	}))
	require.NoError(t, err)
	require.Equal(t, "nav-123", resp.Msg.NavigationId)
	require.Equal(t, "started", resp.Msg.Status)
	require.Equal(t, "playwright", resp.Msg.NavigatorType)
	require.Equal(t, int32(10), resp.Msg.MaxSteps)
}

func TestStartNavigation_MissingFields(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		navigateID:         "nav-xyz",
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})

	_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		Prompt: "do thing",
		Model:  "gpt-4o",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestStartNavigation_NavigatorNotFound(t *testing.T) {
	client := newTestClient(t, Deps{Registry: newTestRegistry(t)})

	_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId:     "sess-1",
		Prompt:        "x",
		Model:         "gpt-4o",
		NavigatorType: "playwright",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestStartNavigation_NoNavigatorsAvailable(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          false,
		unavailableReason:  "driver down",
		clientSourcePolicy: vision.AllSourcesPolicy(),
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})

	_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1",
		Prompt:    "x",
		Model:     "gpt-4o",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestStartNavigation_DelegatesBillingAndCarriesStableGatewayRoot(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		creditPolicy: vision.CreditPolicy{
			RequiresCredits: true,
		},
		navigateID: "nav-1",
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})

	req := connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1",
		Prompt:    "x",
		Model:     "gpt-4o",
	})
	req.Header().Set("Idempotency-Key", "stable retry key")
	resp, err := client.StartNavigation(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "local", nav.lastRequest.AISource)
	require.Len(t, nav.lastRequest.RootOperationID, 64)
	require.NotContains(t, nav.lastRequest.RootOperationID, "stable retry key")
	require.Equal(t, nav.lastRequest.RootOperationID, resp.Header().Get("X-Vrooli-Root-Operation-ID"))
}

func TestNavigationAISourceRequiresSharedAuthorityForAccountBackedModes(t *testing.T) {
	source, token, err := navigationAISource(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, "local", source)
	require.Empty(t, token)

	byok := middleware.WithOwnKeySourceHint(context.Background(), true)
	_, _, err = navigationAISource(byok, false)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	byok = middleware.WithOwnKeySourceHint(monetization.WithAccessToken(context.Background(), "signed-session"), true)
	source, token, err = navigationAISource(byok, false)
	require.NoError(t, err)
	require.Equal(t, "own-key", source)
	require.Equal(t, "signed-session", token)
	source, token, err = navigationAISource(monetization.WithAccessToken(context.Background(), "signed-session"), true)
	require.NoError(t, err)
	require.Equal(t, "own-key", source, "a stored credential authority key preserves the saved-key route")
	require.Equal(t, "signed-session", token)

	credits := monetization.WithAccessToken(context.Background(), "signed-session")
	_, _, err = navigationAISource(credits, false)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "missing verified feature snapshot must fail closed")
	credits = monetization.WithSnapshot(credits, &monetization.Snapshot{Features: []string{entitlement.FeatureAI}})
	source, token, err = navigationAISource(credits, false)
	require.NoError(t, err)
	require.Equal(t, "vrooli-credits", source)
	require.Equal(t, "signed-session", token)
}

func TestStartNavigationCarriesGatewayConsentAndSignedSession(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		navigateID:         "nav-consent",
	}
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	svc := &service{deps: Deps{Logger: logger, Registry: newTestRegistry(t, nav), CallbackBase: "http://127.0.0.1:8080"}}
	ctx := monetization.WithAccessToken(context.Background(), "verified-session-token")
	ctx = monetization.WithSnapshot(ctx, &monetization.Snapshot{Features: []string{entitlement.FeatureAI}})
	req := connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1", Prompt: "inspect this page", Model: "remote_only",
	})
	req.Header().Set("X-Vrooli-AI-Consent-Grant", "opaque-source-bound-receipt")
	req.Header().Set("Idempotency-Key", "vision-call-1")
	response, err := svc.StartNavigation(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "vrooli-credits", nav.lastRequest.AISource)
	require.Equal(t, "verified-session-token", nav.lastRequest.AccessToken)
	require.Equal(t, "opaque-source-bound-receipt", nav.lastRequest.PrivacyConsentGrant)
	require.Equal(t, nav.lastRequest.RootOperationID, response.Header().Get("X-Vrooli-Root-Operation-ID"))
}

func TestStartNavigation_NavigateFailureMapsToUnavailable(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		navigateErr:        errors.New("driver refused"),
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})

	_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1",
		Prompt:    "x",
		Model:     "gpt-4o",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// ---- Status / Abort / Resume --------------------------------------------

func TestGetNavigationStatus_HappyPath(t *testing.T) {
	tracker := &fakeTracker{
		exists: true,
		session: &vision.NavigationSession{
			NavigationID:  "nav-1",
			SessionID:     "sess-1",
			Status:        vision.StatusNavigating,
			StepCount:     5,
			TotalTokens:   1234,
			StartedAt:     time.Now(),
			NavigatorType: vision.NavigatorPlaywright,
		},
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{
		NavigationId: "nav-1",
	}))
	require.NoError(t, err)
	require.Equal(t, "nav-1", resp.Msg.NavigationId)
	require.Equal(t, "navigating", resp.Msg.Status)
	require.Equal(t, int32(5), resp.Msg.StepCount)
}

func TestGetNavigationStatus_NotFound(t *testing.T) {
	tracker := &fakeTracker{exists: false}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	_, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetNavigationStatus_NoTracker(t *testing.T) {
	client := newTestClient(t, Deps{Registry: newTestRegistry(t)})
	_, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetNavigationStatus_MissingID(t *testing.T) {
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: &fakeTracker{exists: true}})
	_, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAbortNavigation_HappyPath(t *testing.T) {
	tracker := &fakeTracker{}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	resp, err := client.AbortNavigation(context.Background(), connect.NewRequest(&aiv1.AbortNavigationRequest{NavigationId: "nav-1"}))
	require.NoError(t, err)
	require.Equal(t, "aborting", resp.Msg.Status)
	require.Equal(t, "nav-1", tracker.lastAbort)
}

func TestAbortNavigation_NotFoundFromTracker(t *testing.T) {
	tracker := &fakeTracker{abortErr: errors.New("session not found")}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	_, err := client.AbortNavigation(context.Background(), connect.NewRequest(&aiv1.AbortNavigationRequest{NavigationId: "missing"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestResumeNavigation_NotAwaitingHuman(t *testing.T) {
	tracker := &fakeTracker{resumeErr: errors.New("navigation is not awaiting human")}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	_, err := client.ResumeNavigation(context.Background(), connect.NewRequest(&aiv1.ResumeNavigationRequest{NavigationId: "nav-1"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestResumeNavigation_HappyPath(t *testing.T) {
	tracker := &fakeTracker{}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	resp, err := client.ResumeNavigation(context.Background(), connect.NewRequest(&aiv1.ResumeNavigationRequest{NavigationId: "nav-1"}))
	require.NoError(t, err)
	require.Equal(t, "resumed", resp.Msg.Status)
	require.Equal(t, "nav-1", tracker.lastResume)
}

// ---- GetNavigationStatus: wait + step history ----------------------------

// lockedTracker mirrors the production trackers: it owns a lock, mutates the
// live session under it, and hands out Snapshot copies.
type lockedTracker struct {
	mu          sync.Mutex
	session     *vision.NavigationSession
	getObserved chan struct{}
}

func (l *lockedTracker) GetSession(string) (*vision.NavigationSession, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.session == nil {
		return nil, false
	}
	snapshot := l.session.Snapshot()
	if l.getObserved != nil {
		close(l.getObserved)
		l.getObserved = nil
	}
	return snapshot, true
}

func (l *lockedTracker) AbortNavigation(context.Context, string) error  { return nil }
func (l *lockedTracker) ResumeNavigation(context.Context, string) error { return nil }

func (l *lockedTracker) transition(status vision.NavigationStatus) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.session.SetStatus(status)
}

func newNavigatingSession() *vision.NavigationSession {
	return &vision.NavigationSession{
		NavigationID:  "nav-w",
		SessionID:     "sess-w",
		Status:        vision.StatusNavigating,
		StartedAt:     time.Now(),
		NavigatorType: vision.NavigatorPlaywright,
	}
}

func TestGetNavigationStatus_ReturnsStepsAndTerminal(t *testing.T) {
	session := newNavigatingSession()
	session.RecordStep(vision.NavigationStepRecord{ActionType: "navigate", URL: "https://example.com", Description: "open"})
	session.RecordStep(vision.NavigationStepRecord{ActionType: "click", Selector: "#go", Success: true})
	session.SetStatus(vision.StatusCompleted)
	tracker := &lockedTracker{session: session}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "nav-w"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Terminal)
	require.Equal(t, "completed", resp.Msg.Status)
	require.Len(t, resp.Msg.Steps, 2)
	require.Equal(t, int32(1), resp.Msg.Steps[0].Index)
	require.Equal(t, "navigate", resp.Msg.Steps[0].ActionType)
	require.Equal(t, "https://example.com", resp.Msg.Steps[0].Url)
	require.Equal(t, "open", resp.Msg.Steps[0].Description)
	require.Equal(t, int32(2), resp.Msg.Steps[1].Index)
	require.Equal(t, "#go", resp.Msg.Steps[1].Selector)
	require.True(t, resp.Msg.Steps[1].Success)
	require.NotNil(t, resp.Msg.Steps[1].At)
}

func TestGetNavigationStatus_NonTerminalWithoutWaitReturnsImmediately(t *testing.T) {
	tracker := &lockedTracker{session: newNavigatingSession()}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	start := time.Now()
	resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "nav-w"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Terminal)
	require.Equal(t, "navigating", resp.Msg.Status)
	require.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestGetNavigationStatus_WaitReturnsEarlyOnTerminalTransition(t *testing.T) {
	tracker := &lockedTracker{session: newNavigatingSession(), getObserved: make(chan struct{})}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	type outcome struct {
		resp *connect.Response[aiv1.GetNavigationStatusResponse]
		err  error
	}
	result := make(chan outcome, 1)
	go func() {
		resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "nav-w", WaitMillis: 10_000}))
		result <- outcome{resp, err}
	}()
	<-tracker.getObserved
	tracker.transition(vision.StatusCompleted)
	var got outcome
	select {
	case got = <-result:
	case <-time.After(2 * time.Second):
		t.Fatal("status wait did not wake on terminal transition")
	}
	resp, err := got.resp, got.err
	require.NoError(t, err)
	require.True(t, resp.Msg.Terminal)
	require.Equal(t, "completed", resp.Msg.Status)
}

func TestGetNavigationStatus_WaitWakesOnAwaitingHuman(t *testing.T) {
	tracker := &lockedTracker{session: newNavigatingSession(), getObserved: make(chan struct{})}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	type outcome struct {
		resp *connect.Response[aiv1.GetNavigationStatusResponse]
		err  error
	}
	result := make(chan outcome, 1)
	go func() {
		resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{NavigationId: "nav-w", WaitMillis: 10_000}))
		result <- outcome{resp, err}
	}()
	<-tracker.getObserved
	tracker.transition(vision.StatusAwaitingHuman)
	var got outcome
	select {
	case got = <-result:
	case <-time.After(2 * time.Second):
		t.Fatal("status wait did not wake on awaiting_human transition")
	}
	resp, err := got.resp, got.err
	require.NoError(t, err)
	require.True(t, resp.Msg.Terminal)
	require.Equal(t, "awaiting_human", resp.Msg.Status)
}

func TestGetNavigationStatus_WaitTimesOutNonTerminal(t *testing.T) {
	tracker := &lockedTracker{session: newNavigatingSession()}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	start := time.Now()
	resp, err := client.GetNavigationStatus(context.Background(), connect.NewRequest(&aiv1.GetNavigationStatusRequest{
		NavigationId: "nav-w",
		WaitMillis:   150,
	}))
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.False(t, resp.Msg.Terminal)
	require.Equal(t, "navigating", resp.Msg.Status)
	require.GreaterOrEqual(t, elapsed, 140*time.Millisecond)
	require.Less(t, elapsed, 3*time.Second)
}

func TestGetNavigationStatus_WaitRespectsContextCancel(t *testing.T) {
	tracker := &lockedTracker{session: newNavigatingSession()}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t), Tracker: tracker})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := client.GetNavigationStatus(ctx, connect.NewRequest(&aiv1.GetNavigationStatusRequest{
		NavigationId: "nav-w",
		WaitMillis:   60_000,
	}))
	require.Error(t, err)
	require.Less(t, time.Since(start), 5*time.Second, "cancelled client must not wait for wait_millis")
}

func TestClampStatusWait(t *testing.T) {
	require.Equal(t, time.Duration(0), clampStatusWait(0))
	require.Equal(t, time.Duration(0), clampStatusWait(-5))
	require.Equal(t, 250*time.Millisecond, clampStatusWait(250))
	require.Equal(t, maxStatusWait, clampStatusWait(300_000))
	require.Equal(t, maxStatusWait, clampStatusWait(9_999_999))
}

func TestStartNavigationTaskContractRejectsUnsupportedNavigatorAndInvalidPolicy(t *testing.T) {
	for _, policy := range []string{"read_only", "invalid"} {
		nav := &fakeNavigator{navType: vision.NavigatorType("alternate"), available: true, clientSourcePolicy: vision.AllSourcesPolicy(), navigateID: "nav"}
		client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav)})
		_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{SessionId: "s", Prompt: "read", Model: "m", EffectPolicy: policy}))
		require.Error(t, err)
	}
}

func TestNavigationStatusCarriesIndependentVerificationAndOutput(t *testing.T) {
	session := &vision.NavigationSession{NavigationID: "nav", Status: vision.StatusCompleted, VerifiedSuccess: true, ExtractedData: map[string]interface{}{"subjects": []interface{}{"hello"}}, FinalURL: "https://example.com/done", Error: "", Summary: "done", TotalDurationMs: 1234}
	result := navigationStatusToProto(session)
	require.True(t, result.VerifiedSuccess)
	require.Equal(t, "hello", result.ExtractedData.AsMap()["subjects"].([]interface{})[0])
	require.Equal(t, "https://example.com/done", result.FinalUrl)
	require.Equal(t, "done", result.Summary)
	require.Equal(t, int64(1234), result.TotalDurationMs)
	session.VerifiedSuccess = false
	session.VerificationError = "postcondition_failed"
	session.Error = "driver_failed"
	result = navigationStatusToProto(session)
	require.False(t, result.VerifiedSuccess)
	require.Equal(t, "postcondition_failed", result.VerificationError)
	require.Equal(t, "driver_failed", result.Error)
}
