package vision_navigation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"

	"github.com/vrooli/browser-automation-studio/services/credits"
	"github.com/vrooli/browser-automation-studio/services/vision"
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

func (f *fakeNavigator) Navigate(_ context.Context, _ vision.NavigationRequest) (vision.NavigationHandle, error) {
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

// stubCredits implements just enough of credits.CreditService for the
// handler's CanPerformAIOperation path. All other methods are unreachable
// in these tests and panic to surface accidental call sites loudly.
type stubCredits struct {
	canPerform bool
	errCode    string
	errMsg     string
	remaining  int
	err        error
	called     bool
}

func (s *stubCredits) CanPerformAIOperation(_ context.Context, _ string, _ credits.OperationType, _ bool) (bool, string, string, int, error) {
	s.called = true
	return s.canPerform, s.errCode, s.errMsg, s.remaining, s.err
}

func (*stubCredits) CanCharge(context.Context, string, credits.OperationType) (bool, int, error) {
	panic("unused")
}

func (*stubCredits) Charge(context.Context, credits.ChargeRequest) (*credits.ChargeResult, error) {
	panic("unused")
}

func (*stubCredits) ChargeIfAllowed(context.Context, credits.ChargeRequest) (*credits.ChargeResult, error) {
	panic("unused")
}

func (*stubCredits) GetUsage(context.Context, string) (*credits.UsageSummary, error) {
	panic("unused")
}
func (*stubCredits) GetOperationCost(credits.OperationType) int { panic("unused") }
func (*stubCredits) LogFailedOperation(context.Context, credits.ChargeRequest, error) error {
	panic("unused")
}

func (*stubCredits) GetUsageHistory(context.Context, string, int, int) ([]credits.UsageSummary, bool, error) {
	panic("unused")
}

func (*stubCredits) GetOperationLog(context.Context, string, string, string, int, int) (*credits.OperationLogPage, error) {
	panic("unused")
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
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
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

func TestStartNavigation_CreditExhaustionMapsToResourceExhausted(t *testing.T) {
	nav := &fakeNavigator{
		navType:            vision.NavigatorPlaywright,
		available:          true,
		clientSourcePolicy: vision.AllSourcesPolicy(),
		creditPolicy: vision.CreditPolicy{
			RequiresCredits: true,
			OperationType:   credits.OpAIVisionNavigate,
		},
		navigateID: "nav-1",
	}
	creds := &stubCredits{
		canPerform: false,
		errCode:    "INSUFFICIENT_CREDITS",
		errMsg:     "out of credits",
	}
	client := newTestClient(t, Deps{Registry: newTestRegistry(t, nav), Credits: creds})

	_, err := client.StartNavigation(context.Background(), connect.NewRequest(&aiv1.StartNavigationRequest{
		SessionId: "sess-1",
		Prompt:    "x",
		Model:     "gpt-4o",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.True(t, creds.called)
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
