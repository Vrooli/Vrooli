package agentaccess

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const (
	AppMonitorCaptureTaskRevision   = "app-monitor.preview-workspace.performance-capture.v1"
	AppMonitorCaptureOracleRevision = "app-monitor.preview-workspace.metrics.v1"
	maxAppMonitorCaptureLifetime    = 10 * time.Minute
)

type AppMonitorCapturePhase string

const (
	AppMonitorCaptureBegin       AppMonitorCapturePhase = "begin"
	AppMonitorCaptureAfterMount  AppMonitorCapturePhase = "after_mount"
	AppMonitorCaptureAfterScroll AppMonitorCapturePhase = "after_scroll"
	AppMonitorCaptureAfterPicker AppMonitorCapturePhase = "after_picker"
	AppMonitorCaptureFinish      AppMonitorCapturePhase = "finish"
)

type AppMonitorCaptureBinding struct {
	CaptureID       string    `json:"captureId"`
	Phase           string    `json:"phase"`
	RunID           string    `json:"runId"`
	CallerID        string    `json:"callerId"`
	GrantReceipt    string    `json:"grantReceipt"`
	SessionID       string    `json:"sessionId"`
	PermissionRef   string    `json:"permissionRef"`
	ProfileID       string    `json:"profileId"`
	ProfileRevision string    `json:"profileRevision"`
	OwnerID         string    `json:"ownerId"`
	LeaseID         string    `json:"leaseId"`
	Backend         string    `json:"backend"`
	TaskRevision    string    `json:"taskRevision"`
	OracleRevision  string    `json:"oracleRevision"`
	ObservedAt      time.Time `json:"observedAt"`
}

// AppMonitorCaptureResponse keeps raw bounded trace bytes out of all metric
// phases. The trace appears only in the explicitly authorized finish result.
type AppMonitorCaptureResponse struct {
	Binding AppMonitorCaptureBinding          `json:"binding"`
	Metrics *driver.AppMonitorMetricsResponse `json:"metrics,omitempty"`
	Trace   *session.AppMonitorTraceArtifact  `json:"trace,omitempty"`
}

type appMonitorCaptureState struct {
	captureID string
	nextPhase int
	startedAt time.Time
	pageID    string
	trace     bool
	finished  bool
}

// CaptureAppMonitor serializes the fixed App Monitor capture lifecycle under
// the existing session lease. The Broker mints the capture ID and performs
// fresh caller, grant, profile, owner and URL checks before and after each
// driver read. No caller-controlled URL, selector, trace category or path is
// accepted here.
func (b *Broker) CaptureAppMonitor(ctx context.Context, sessionID, captureID string, phase AppMonitorCapturePhase) (AppMonitorCaptureResponse, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return AppMonitorCaptureResponse{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return AppMonitorCaptureResponse{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil || e.backend != string(sessionpermission.BackendManaged) || e.profile.ProfileMode != "fresh" {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	if phase != AppMonitorCaptureBegin && phase != AppMonitorCaptureAfterMount && phase != AppMonitorCaptureAfterScroll && phase != AppMonitorCaptureAfterPicker && phase != AppMonitorCaptureFinish {
		return AppMonitorCaptureResponse{}, errors.New("unsupported App Monitor capture phase")
	}
	credentials, err := credentialsFromContext(ctx)
	if err != nil || e.grantReceipt == "" || subtle.ConstantTimeCompare([]byte(credentials.grantReceipt), []byte(e.grantReceipt)) != 1 {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	callerID, err := b.caller.BrowserCallerID(ctx)
	if err != nil || callerID == "" || callerID != e.callerID {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	runID, err := appMonitorRunID(callerID)
	if err != nil {
		return AppMonitorCaptureResponse{}, err
	}
	ownerID, leaseID := e.session.OwnerIdentity()
	if e.session.ID() != sessionID || ownerID == "" || leaseID == "" || ownerID != e.ownerID || leaseID != e.leaseID {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	profile, policy, err := b.profiles.Resolve(ctx, e.permissionRef)
	if err != nil || policy == nil || !reflectProfilesEqual(profile, e.profile) {
		return AppMonitorCaptureResponse{}, errors.New("App Monitor capture profile is stale or revoked")
	}
	if !profile.AppMonitorMetrics && !profile.AppMonitorTrace {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	if profile.AppMonitorTrace && (profile.AppMonitorTraceMaxDurationMS <= 0 || profile.AppMonitorTraceMaxBytes <= 0) {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	if phase != AppMonitorCaptureBegin && (e.appMonitorCapture == nil || e.appMonitorCapture.finished || captureID == "" || captureID != e.appMonitorCapture.captureID) {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	if phase == AppMonitorCaptureBegin && (captureID != "" || e.appMonitorCapture != nil) {
		return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
	}
	state := e.appMonitorCapture
	if state != nil && time.Since(state.startedAt) > maxAppMonitorCaptureLifetime {
		return AppMonitorCaptureResponse{}, errors.New("App Monitor capture exceeded its session lifetime")
	}
	if phase == AppMonitorCaptureBegin {
		captureID = uuid.NewString()
		state = &appMonitorCaptureState{captureID: captureID, nextPhase: 1, startedAt: time.Now().UTC(), trace: profile.AppMonitorTrace}
		if state.trace {
			if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "app_monitor_trace"}); err != nil {
				return AppMonitorCaptureResponse{}, fmt.Errorf("authorize App Monitor trace start: %w", err)
			}
			if err := e.session.BeginAppMonitorCapture(ctx, captureID); err != nil {
				return AppMonitorCaptureResponse{}, fmt.Errorf("start App Monitor trace capture: %w", err)
			}
		}
		e.appMonitorCapture = state
		if err := reauthorizeAppMonitorCapture(ctx, e, state.trace); err != nil {
			return AppMonitorCaptureResponse{}, fmt.Errorf("reauthorize App Monitor capture start: %w", err)
		}
		return AppMonitorCaptureResponse{Binding: b.appMonitorCaptureBinding(e, runID, state, phase, time.Now().UTC())}, nil
	}
	if phase == AppMonitorCaptureAfterMount || phase == AppMonitorCaptureAfterScroll || phase == AppMonitorCaptureAfterPicker {
		wantStage := map[AppMonitorCapturePhase]int{AppMonitorCaptureAfterMount: 1, AppMonitorCaptureAfterScroll: 2, AppMonitorCaptureAfterPicker: 3}[phase]
		if state.nextPhase != wantStage || !profile.AppMonitorMetrics {
			return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
		}
		if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "app_monitor_metrics"}); err != nil {
			return AppMonitorCaptureResponse{}, err
		}
		metrics, readErr := e.session.ReadAppMonitorMetrics(ctx)
		if readErr != nil {
			return AppMonitorCaptureResponse{}, readErr
		}
		if err := validateAppMonitorMetrics(e, state, metrics); err != nil {
			return AppMonitorCaptureResponse{}, err
		}
		if err := reauthorizeAppMonitorCapture(ctx, e, false); err != nil {
			return AppMonitorCaptureResponse{}, err
		}
		if state.pageID == "" {
			state.pageID = metrics.PageID
		}
		state.nextPhase++
		return AppMonitorCaptureResponse{Binding: b.appMonitorCaptureBinding(e, runID, state, phase, parseAppMonitorTimestamp(metrics.ObservedAt)), Metrics: metrics}, nil
	}
	if phase == AppMonitorCaptureFinish {
		if state.nextPhase != 4 && !(state.nextPhase == 1 && !profile.AppMonitorMetrics) {
			return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
		}
		if state.trace && time.Since(state.startedAt) > time.Duration(profile.AppMonitorTraceMaxDurationMS)*time.Millisecond {
			return AppMonitorCaptureResponse{}, errors.New("App Monitor trace exceeded its granted duration ceiling")
		}
		if state.trace {
			if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "app_monitor_trace"}); err != nil {
				return AppMonitorCaptureResponse{}, err
			}
		}
		var artifact *session.AppMonitorTraceArtifact
		if state.trace {
			artifact, err = e.session.FinishAppMonitorCapture(ctx, state.captureID)
			if err != nil {
				return AppMonitorCaptureResponse{}, err
			}
			if err := validateAppMonitorTrace(e, state, profile, artifact); err != nil {
				return AppMonitorCaptureResponse{}, err
			}
		}
		if err := reauthorizeAppMonitorCapture(ctx, e, state.trace); err != nil {
			return AppMonitorCaptureResponse{}, err
		}
		state.finished = true
		state.nextPhase = 5
		response := AppMonitorCaptureResponse{Binding: b.appMonitorCaptureBinding(e, runID, state, phase, time.Now().UTC())}
		response.Trace = artifact
		return response, nil
	}
	return AppMonitorCaptureResponse{}, sessionpermission.ErrDenied
}

func reauthorizeAppMonitorCapture(ctx context.Context, e *entry, trace bool) error {
	kind := "app_monitor_metrics"
	if trace {
		kind = "app_monitor_trace"
	}
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: kind}); err != nil {
		return err
	}
	return nil
}

func validateAppMonitorMetrics(e *entry, state *appMonitorCaptureState, metrics *driver.AppMonitorMetricsResponse) error {
	if metrics == nil || metrics.Contract != driver.AppMonitorMetricsContract || metrics.SessionID != e.sessionID || strings.TrimSpace(metrics.PageID) == "" ||
		metrics.IframeCount < 0 || metrics.PaneCount < 0 || metrics.TabCardCount < 0 || metrics.ProfilerMarkCount < 0 || metrics.WorkspaceStorageBytes < 0 {
		return errors.New("driver returned an invalid App Monitor metric observation")
	}
	parsedAt, err := time.Parse(time.RFC3339Nano, metrics.ObservedAt)
	if err != nil || parsedAt.IsZero() || time.Since(parsedAt) > 30*time.Second || parsedAt.After(time.Now().Add(5*time.Second)) {
		return errors.New("driver returned a stale App Monitor metric observation")
	}
	u, err := url.Parse(metrics.URL)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Path != "/apps/workspace" || !e.profile.AllowsURL(metrics.URL) {
		return sessionpermission.ErrDenied
	}
	if state.pageID != "" && state.pageID != metrics.PageID {
		return sessionpermission.ErrDenied
	}
	return nil
}

func validateAppMonitorTrace(e *entry, state *appMonitorCaptureState, profile sessionpermission.Profile, artifact *session.AppMonitorTraceArtifact) error {
	if artifact == nil || artifact.CaptureID != state.captureID || artifact.SessionID != e.sessionID || artifact.PageID == "" ||
		artifact.ByteLength <= 0 || int64(artifact.ByteLength) > profile.AppMonitorTraceMaxBytes || len(artifact.TraceJSON) != artifact.ByteLength ||
		len(artifact.SHA256) != 64 || !isHexDigest(artifact.SHA256) {
		return errors.New("driver returned an invalid or over-budget App Monitor trace")
	}
	digest := sha256.Sum256([]byte(artifact.TraceJSON))
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(digest[:])), []byte(artifact.SHA256)) != 1 {
		return errors.New("App Monitor trace digest mismatch")
	}
	if state.pageID != "" && state.pageID != artifact.PageID {
		return sessionpermission.ErrDenied
	}
	if artifact.Contract != "app-monitor.preview-workspace.trace.v1" {
		return errors.New("unsupported App Monitor trace contract")
	}
	return nil
}

func (b *Broker) appMonitorCaptureBinding(e *entry, runID string, state *appMonitorCaptureState, phase AppMonitorCapturePhase, observedAt time.Time) AppMonitorCaptureBinding {
	profileJSON, _ := json.Marshal(e.profile)
	profileDigest := sha256.Sum256(profileJSON)
	ownerID, leaseID := e.session.OwnerIdentity()
	return AppMonitorCaptureBinding{
		CaptureID: state.captureID, Phase: string(phase), RunID: runID, CallerID: e.callerID,
		GrantReceipt: e.grantReceipt, SessionID: e.sessionID, PermissionRef: e.permissionRef,
		ProfileID: e.profile.ProfileID, ProfileRevision: hex.EncodeToString(profileDigest[:]),
		OwnerID: ownerID, LeaseID: leaseID, Backend: e.backend,
		TaskRevision: AppMonitorCaptureTaskRevision, OracleRevision: AppMonitorCaptureOracleRevision,
		ObservedAt: observedAt.UTC(),
	}
}

func appMonitorRunID(callerID string) (string, error) {
	const prefix = "run:"
	if !strings.HasPrefix(callerID, prefix) {
		return "", errors.New("App Monitor capture requires an Agent Manager run-bound session")
	}
	id, err := uuid.Parse(strings.TrimPrefix(callerID, prefix))
	if err != nil || id == uuid.Nil {
		return "", errors.New("App Monitor capture has an invalid run binding")
	}
	return id.String(), nil
}

func parseAppMonitorTimestamp(raw string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, raw)
	return parsed
}

func isHexDigest(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func reflectProfilesEqual(left, right sessionpermission.Profile) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
