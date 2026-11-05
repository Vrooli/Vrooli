// Package drills owns cross-layer failure-drill orchestration. The driver owns
// injection; this package owns the arm/run/assert/disarm transaction.
package drills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	drv "github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	drillv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/drills"
	drillconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/drills/drillsconnect"
)

type Deps struct {
	DriverURL, AdminSecret string
	SessionBroker          *autosession.Manager
	HTTPClient             *http.Client
	Logger                 *logrus.Logger
}
type service struct{ deps Deps }

func Module(d Deps) connectx.ServiceMount {
	if d.DriverURL == "" {
		d.DriverURL = os.Getenv(drv.PlaywrightDriverEnv)
		if d.DriverURL == "" {
			d.DriverURL = drv.DefaultDriverURL
		}
	}
	if d.AdminSecret == "" {
		d.AdminSecret = os.Getenv(drv.PlaywrightDriverAdminSecretEnv)
	}
	if d.HTTPClient == nil {
		d.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if d.Logger == nil {
		panic("drills.Module requires Logger")
	}
	path, handler := drillconnect.NewFailureDrillServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

func (s *service) ListDrills(context.Context, *connect.Request[drillv1.ListDrillsRequest]) (*connect.Response[drillv1.ListDrillsResponse], error) {
	return connect.NewResponse(&drillv1.ListDrillsResponse{Drills: []*drillv1.DrillDefinition{
		{Name: drillv1.DrillName_DRILL_NAME_DRIVER_UNAVAILABLE, Description: "One controlled admission failure leaves no session."},
		{Name: drillv1.DrillName_DRILL_NAME_PARTIAL_INITIALIZATION, Description: "Failure after registration reconciles the session."},
		{Name: drillv1.DrillName_DRILL_NAME_CAPACITY, Description: "Token-scoped capacity rejection preserves normal admission."},
		{Name: drillv1.DrillName_DRILL_NAME_EXPIRY, Description: "TTL expiry and disarm leave no fault residue."},
	}}), nil
}

func (s *service) RunDrill(ctx context.Context, req *connect.Request[drillv1.RunDrillRequest]) (*connect.Response[drillv1.RunDrillResponse], error) {
	name := req.Msg.GetName()
	fault, ok := map[drillv1.DrillName]string{drillv1.DrillName_DRILL_NAME_DRIVER_UNAVAILABLE: "driver_unavailable", drillv1.DrillName_DRILL_NAME_PARTIAL_INITIALIZATION: "fail_after_session_registration", drillv1.DrillName_DRILL_NAME_CAPACITY: "capacity_lease", drillv1.DrillName_DRILL_NAME_EXPIRY: "driver_unavailable"}[name]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported drill"))
	}
	token := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	ttl := 5000
	if name == drillv1.DrillName_DRILL_NAME_EXPIRY {
		ttl = 100
	}
	arm := map[string]any{"token": token, "fault": fault, "ttl_ms": ttl}
	if name == drillv1.DrillName_DRILL_NAME_CAPACITY {
		arm["lease_count"] = 1
	}
	pre, err := s.snapshot(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err := s.post(ctx, "/test-control/faults/arm", arm, nil); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	cleanup := "not attempted"
	defer func() {
		if cleanup == "not attempted" {
			_ = s.post(context.Background(), "/test-control/faults/disarm", map[string]string{"token": token}, nil)
		}
	}()
	var expected bool
	var outcome string
	if name == drillv1.DrillName_DRILL_NAME_EXPIRY {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			snap, snapErr := s.snapshot(ctx)
			if snapErr == nil && len(snap.Faults) == 0 {
				expected = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		outcome = "fault expired"
	} else {
		status, startErr := s.startSession(ctx, token)
		expected = startErr == nil && ((name == drillv1.DrillName_DRILL_NAME_CAPACITY && status == http.StatusTooManyRequests) || (name != drillv1.DrillName_DRILL_NAME_CAPACITY && status >= 500))
		outcome = fmt.Sprintf("session start status %d", status)
	}
	post, postErr := s.snapshot(ctx)
	if err := s.post(ctx, "/test-control/faults/disarm", map[string]string{"token": token}, nil); err != nil {
		cleanup = "disarm failed: " + err.Error()
	} else {
		cleanup = "disarmed"
	}
	auditEvent := "consumed"
	if name == drillv1.DrillName_DRILL_NAME_EXPIRY {
		auditEvent = "expired"
	}
	auditObserved := auditContains(post.Audit, fault, auditEvent)
	state := "unavailable"
	if s.deps.SessionBroker != nil {
		state = s.deps.SessionBroker.CircuitBreakerState()
	}
	breakerClosed := name != drillv1.DrillName_DRILL_NAME_CAPACITY || state == "closed"
	assertions := []*drillv1.DrillAssertion{{Name: "expected controlled outcome", Passed: expected, Detail: outcome}, {Name: "driver seam observed", Passed: auditObserved, Detail: fault + " " + auditEvent}, {Name: "capacity does not open breaker", Passed: breakerClosed, Detail: state}, {Name: "fault residue removed", Passed: postErr == nil && len(post.Faults) == 0, Detail: fmt.Sprintf("pre=%d post=%d", len(pre.Faults), len(post.Faults))}}
	passed := expected && auditObserved && breakerClosed && postErr == nil && len(post.Faults) == 0
	evidence, _ := json.Marshal(map[string]any{"precondition": pre, "postcondition": post, "outcome": outcome})
	return connect.NewResponse(&drillv1.RunDrillResponse{Verdict: &drillv1.DrillVerdict{Name: name, Passed: passed, ExpectedFailureObserved: expected, CleanupCompleted: cleanup == "disarmed", PrimaryOutcome: outcome, CleanupOutcome: cleanup, Assertions: assertions, EvidenceJson: string(evidence)}}), nil
}

func (s *service) post(ctx context.Context, path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, s.deps.DriverURL+path, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Playwright-Admin-Secret", s.deps.AdminSecret)
	resp, err := s.deps.HTTPClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("driver %s returned %d", path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

type faultSnapshot struct {
	Faults []json.RawMessage `json:"faults"`
	Audit  []faultAuditEvent `json:"audit"`
}
type faultAuditEvent struct {
	Fault string `json:"fault"`
	Event string `json:"event"`
}

func auditContains(events []faultAuditEvent, fault, event string) bool {
	for _, candidate := range events {
		if candidate.Fault == fault && candidate.Event == event {
			return true
		}
	}
	return false
}

func (s *service) snapshot(ctx context.Context) (faultSnapshot, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, s.deps.DriverURL+"/test-control/faults", nil)
	if err != nil {
		return faultSnapshot{}, err
	}
	r.Header.Set("X-Playwright-Admin-Secret", s.deps.AdminSecret)
	resp, err := s.deps.HTTPClient.Do(r)
	if err != nil {
		return faultSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return faultSnapshot{}, fmt.Errorf("snapshot returned %d", resp.StatusCode)
	}
	var v faultSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return faultSnapshot{}, err
	}
	return v, nil
}
func (s *service) startSession(ctx context.Context, token string) (int, error) {
	if s.deps.SessionBroker == nil {
		return 0, errors.New("session broker unavailable")
	}
	owned, err := s.deps.SessionBroker.CreateForDrill(ctx, &drv.CreateSessionRequest{ExecutionID: "drill-" + uuid.NewString(), WorkflowID: "failure-drill", Options: drv.SessionOptions{Viewport: drv.Viewport{Width: 100, Height: 100}, ReuseMode: "fresh", FrameScale: "css"}}, token)
	if err == nil {
		if _, closeErr := owned.CloseWithArtifacts(ctx); closeErr != nil {
			return 0, fmt.Errorf("close successful drill session: %w", closeErr)
		}
		return http.StatusOK, nil
	}
	var driverErr *drv.Error
	if errors.As(err, &driverErr) {
		return driverErr.Status, nil
	}
	return 0, err
}
