package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"prompt-manager/store"
)

const (
	HeartbeatControlStatusActive       = "active"
	HeartbeatControlStatusWarningIdle  = "warning-idle-soon"
	HeartbeatControlStatusPausedAuto   = "paused-auto-idle"
	HeartbeatControlStatusPausedManual = "paused-manual"
	HeartbeatControlTeamModeInherit    = "inherit"
	HeartbeatControlTeamModeDisabled   = "disabled"
	HeartbeatControlTeamModeCustom     = "custom"
	HeartbeatControlResumeModeManual   = "manual"
	HeartbeatControlDefaultPauseDays   = 14
	HeartbeatControlDefaultWarningDays = 10
	heartbeatControlSchemaVersion      = 1
	heartbeatControlMaxThresholdDays   = 365
	heartbeatControlStoreFilename      = "heartbeat-control.json"
	heartbeatControlGlobalScope        = "global"
)

var ErrHeartbeatPaused = errors.New("heartbeat scheduling is paused")

type HeartbeatControlPolicy struct {
	Enabled                                bool   `json:"enabled"`
	PauseAfterDaysWithoutHumanEngagement   int    `json:"pauseAfterDaysWithoutHumanEngagement"`
	WarningAfterDaysWithoutHumanEngagement int    `json:"warningAfterDaysWithoutHumanEngagement"`
	ResumeMode                             string `json:"resumeMode"`
}

type HeartbeatControlState struct {
	Status                    string  `json:"status"`
	LastHumanEngagementAt     *string `json:"lastHumanEngagementAt,omitempty"`
	LastHumanEngagementReason string  `json:"lastHumanEngagementReason,omitempty"`
	LastHumanEngagementTeamID string  `json:"lastHumanEngagementTeamId,omitempty"`
	LastPausedAt              *string `json:"lastPausedAt,omitempty"`
	LastPausedReason          string  `json:"lastPausedReason,omitempty"`
	LastResumedAt             *string `json:"lastResumedAt,omitempty"`
}

type HeartbeatControlTeamOverride struct {
	Mode                                   string `json:"mode"`
	PauseAfterDaysWithoutHumanEngagement   *int   `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int   `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             string `json:"resumeMode,omitempty"`
}

type HeartbeatControlDocument struct {
	SchemaVersion int                                     `json:"schemaVersion"`
	GlobalPolicy  HeartbeatControlPolicy                  `json:"globalPolicy"`
	GlobalState   HeartbeatControlState                   `json:"globalState"`
	TeamOverrides map[string]HeartbeatControlTeamOverride `json:"teamOverrides"`
	TeamState     map[string]HeartbeatControlState        `json:"teamState"`
}

type HeartbeatControlStatusResponse struct {
	Scope                     string                           `json:"scope"`
	TeamID                    string                           `json:"teamId,omitempty"`
	Status                    string                           `json:"status"`
	EffectivePolicy           HeartbeatControlPolicy           `json:"effectivePolicy"`
	GlobalPolicy              HeartbeatControlPolicy           `json:"globalPolicy,omitempty"`
	TeamOverride              *HeartbeatControlTeamOverride    `json:"teamOverride,omitempty"`
	LastHumanEngagementAt     *string                          `json:"lastHumanEngagementAt,omitempty"`
	LastHumanEngagementReason string                           `json:"lastHumanEngagementReason,omitempty"`
	LastHumanEngagementTeamID string                           `json:"lastHumanEngagementTeamId,omitempty"`
	PausedAt                  *string                          `json:"pausedAt,omitempty"`
	PausedReason              string                           `json:"pausedReason,omitempty"`
	WarningAt                 *string                          `json:"warningAt,omitempty"`
	AutoPauseAt               *string                          `json:"autoPauseAt,omitempty"`
	ResumeHint                string                           `json:"resumeHint,omitempty"`
	Teams                     []HeartbeatControlStatusResponse `json:"teams,omitempty"`
}

type HeartbeatControlPolicyRequest struct {
	Enabled                                *bool   `json:"enabled,omitempty"`
	PauseAfterDaysWithoutHumanEngagement   *int    `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int    `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             *string `json:"resumeMode,omitempty"`
}

type HeartbeatControlTeamPolicyRequest struct {
	Mode                                   *string `json:"mode,omitempty"`
	PauseAfterDaysWithoutHumanEngagement   *int    `json:"pauseAfterDaysWithoutHumanEngagement,omitempty"`
	WarningAfterDaysWithoutHumanEngagement *int    `json:"warningAfterDaysWithoutHumanEngagement,omitempty"`
	ResumeMode                             *string `json:"resumeMode,omitempty"`
}

type HeartbeatControlPauseRequest struct {
	Reason string `json:"reason,omitempty"`
}

type HeartbeatPausedErrorResponse struct {
	Error      string `json:"error"`
	Scope      string `json:"scope"`
	TeamID     string `json:"teamId,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	ResumeHint string `json:"resumeHint"`
}

type HumanEngagementEvent struct {
	TeamID      string
	Reason      string
	Attribution store.AttributionInfo
}

type HeartbeatControlStore struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

func NewHeartbeatControlStore(runtimeDataRoot string) *HeartbeatControlStore {
	return &HeartbeatControlStore{
		path: filepath.Join(runtimeDataRoot, heartbeatControlStoreFilename),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *HeartbeatControlStore) SetNowForTests(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

func (s *HeartbeatControlStore) Status(ctx context.Context, teams []store.Team) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	changed := s.evaluateLocked(doc, s.now())
	if changed {
		if err := s.saveLocked(doc); err != nil {
			return nil, err
		}
	}
	resp := s.globalStatusLocked(doc)
	resp.Teams = make([]HeartbeatControlStatusResponse, 0, len(teams))
	for _, team := range teams {
		resp.Teams = append(resp.Teams, *s.teamStatusLocked(doc, team.ID))
	}
	return resp, nil
}

func (s *HeartbeatControlStore) TeamStatus(ctx context.Context, teamID string) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	changed := s.evaluateLocked(doc, s.now())
	if changed {
		if err := s.saveLocked(doc); err != nil {
			return nil, err
		}
	}
	return s.teamStatusLocked(doc, teamID), nil
}

func (s *HeartbeatControlStore) AllowStart(ctx context.Context, teamID string) (*HeartbeatPausedErrorResponse, error) {
	status, err := s.TeamStatus(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if status.Status == HeartbeatControlStatusPausedAuto || status.Status == HeartbeatControlStatusPausedManual {
		return &HeartbeatPausedErrorResponse{
			Error:      "heartbeat_paused",
			Scope:      status.Scope,
			TeamID:     status.TeamID,
			Status:     status.Status,
			Message:    pausedMessage(status),
			ResumeHint: status.ResumeHint,
		}, ErrHeartbeatPaused
	}
	return nil, nil
}

func (s *HeartbeatControlStore) UpdateGlobalPolicy(ctx context.Context, req HeartbeatControlPolicyRequest, attr store.AttributionInfo) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	policy := doc.GlobalPolicy
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.PauseAfterDaysWithoutHumanEngagement != nil {
		policy.PauseAfterDaysWithoutHumanEngagement = *req.PauseAfterDaysWithoutHumanEngagement
	}
	if req.WarningAfterDaysWithoutHumanEngagement != nil {
		policy.WarningAfterDaysWithoutHumanEngagement = *req.WarningAfterDaysWithoutHumanEngagement
	}
	if req.ResumeMode != nil {
		policy.ResumeMode = strings.TrimSpace(*req.ResumeMode)
	}
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	doc.GlobalPolicy = policy
	s.recordHumanEngagementLocked(doc, HumanEngagementEvent{Reason: "heartbeat-policy-updated", Attribution: attr})
	s.evaluateLocked(doc, s.now())
	if err := s.saveLocked(doc); err != nil {
		return nil, err
	}
	return s.globalStatusLocked(doc), nil
}

func (s *HeartbeatControlStore) UpdateTeamPolicy(ctx context.Context, teamID string, req HeartbeatControlTeamPolicyRequest, attr store.AttributionInfo) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	override := doc.TeamOverrides[teamID]
	if override.Mode == "" {
		override.Mode = HeartbeatControlTeamModeInherit
	}
	if req.Mode != nil {
		override.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.PauseAfterDaysWithoutHumanEngagement != nil {
		v := *req.PauseAfterDaysWithoutHumanEngagement
		override.PauseAfterDaysWithoutHumanEngagement = &v
	}
	if req.WarningAfterDaysWithoutHumanEngagement != nil {
		v := *req.WarningAfterDaysWithoutHumanEngagement
		override.WarningAfterDaysWithoutHumanEngagement = &v
	}
	if req.ResumeMode != nil {
		override.ResumeMode = strings.TrimSpace(*req.ResumeMode)
	}
	if err := validateTeamOverride(doc.GlobalPolicy, override); err != nil {
		return nil, err
	}
	doc.TeamOverrides[teamID] = override
	s.recordHumanEngagementLocked(doc, HumanEngagementEvent{TeamID: teamID, Reason: "team-heartbeat-policy-updated", Attribution: attr})
	s.evaluateLocked(doc, s.now())
	if err := s.saveLocked(doc); err != nil {
		return nil, err
	}
	return s.teamStatusLocked(doc, teamID), nil
}

func (s *HeartbeatControlStore) Pause(ctx context.Context, teamID, reason string, attr store.AttributionInfo) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	now := formatTime(s.now())
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "manual pause"
	}
	if teamID == "" {
		doc.GlobalState.Status = HeartbeatControlStatusPausedManual
		doc.GlobalState.LastPausedAt = &now
		doc.GlobalState.LastPausedReason = reason
		s.recordHumanEngagementLocked(doc, HumanEngagementEvent{Reason: "heartbeat-manual-pause", Attribution: attr})
	} else {
		state := doc.TeamState[teamID]
		state.Status = HeartbeatControlStatusPausedManual
		state.LastPausedAt = &now
		state.LastPausedReason = reason
		doc.TeamState[teamID] = state
		s.recordHumanEngagementLocked(doc, HumanEngagementEvent{TeamID: teamID, Reason: "team-heartbeat-manual-pause", Attribution: attr})
	}
	if err := s.saveLocked(doc); err != nil {
		return nil, err
	}
	if teamID == "" {
		return s.globalStatusLocked(doc), nil
	}
	return s.teamStatusLocked(doc, teamID), nil
}

func (s *HeartbeatControlStore) Resume(ctx context.Context, teamID string, attr store.AttributionInfo) (*HeartbeatControlStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	now := formatTime(s.now())
	if teamID == "" {
		doc.GlobalState.Status = HeartbeatControlStatusActive
		doc.GlobalState.LastResumedAt = &now
		doc.GlobalState.LastPausedAt = nil
		doc.GlobalState.LastPausedReason = ""
		s.recordHumanEngagementLocked(doc, HumanEngagementEvent{Reason: "heartbeat-manual-resume", Attribution: attr})
	} else {
		state := doc.TeamState[teamID]
		state.Status = HeartbeatControlStatusActive
		state.LastResumedAt = &now
		state.LastPausedAt = nil
		state.LastPausedReason = ""
		doc.TeamState[teamID] = state
		s.recordHumanEngagementLocked(doc, HumanEngagementEvent{TeamID: teamID, Reason: "team-heartbeat-manual-resume", Attribution: attr})
	}
	s.evaluateLocked(doc, s.now())
	if err := s.saveLocked(doc); err != nil {
		return nil, err
	}
	if teamID == "" {
		return s.globalStatusLocked(doc), nil
	}
	return s.teamStatusLocked(doc, teamID), nil
}

func (s *HeartbeatControlStore) RecordHumanEngagement(ctx context.Context, event HumanEngagementEvent) error {
	if event.Attribution.Kind != store.KnowledgeKindOperatorDirect {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.loadLocked()
	if err != nil {
		return err
	}
	s.recordHumanEngagementLocked(doc, event)
	s.evaluateLocked(doc, s.now())
	return s.saveLocked(doc)
}

func (s *HeartbeatControlStore) loadLocked() (*HeartbeatControlDocument, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			doc := defaultHeartbeatControlDocument(s.now())
			if err := s.saveLocked(doc); err != nil {
				return nil, err
			}
			return doc, nil
		}
		return nil, err
	}
	var doc HeartbeatControlDocument
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	normalizeDocument(&doc, s.now())
	if err := validatePolicy(doc.GlobalPolicy); err != nil {
		return nil, err
	}
	for _, override := range doc.TeamOverrides {
		if err := validateTeamOverride(doc.GlobalPolicy, override); err != nil {
			return nil, err
		}
	}
	return &doc, nil
}

func (s *HeartbeatControlStore) saveLocked(doc *HeartbeatControlDocument) error {
	normalizeDocument(doc, s.now())
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func defaultHeartbeatControlDocument(now time.Time) *HeartbeatControlDocument {
	t := formatTime(now)
	return &HeartbeatControlDocument{
		SchemaVersion: heartbeatControlSchemaVersion,
		GlobalPolicy: HeartbeatControlPolicy{
			Enabled:                                true,
			PauseAfterDaysWithoutHumanEngagement:   HeartbeatControlDefaultPauseDays,
			WarningAfterDaysWithoutHumanEngagement: HeartbeatControlDefaultWarningDays,
			ResumeMode:                             HeartbeatControlResumeModeManual,
		},
		GlobalState: HeartbeatControlState{
			Status:                    HeartbeatControlStatusActive,
			LastHumanEngagementAt:     &t,
			LastHumanEngagementReason: "initialization",
		},
		TeamOverrides: map[string]HeartbeatControlTeamOverride{},
		TeamState:     map[string]HeartbeatControlState{},
	}
}

func normalizeDocument(doc *HeartbeatControlDocument, now time.Time) {
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = heartbeatControlSchemaVersion
	}
	if doc.GlobalPolicy.PauseAfterDaysWithoutHumanEngagement == 0 {
		doc.GlobalPolicy.PauseAfterDaysWithoutHumanEngagement = HeartbeatControlDefaultPauseDays
	}
	if doc.GlobalPolicy.WarningAfterDaysWithoutHumanEngagement == 0 {
		doc.GlobalPolicy.WarningAfterDaysWithoutHumanEngagement = HeartbeatControlDefaultWarningDays
	}
	if doc.GlobalPolicy.ResumeMode == "" {
		doc.GlobalPolicy.ResumeMode = HeartbeatControlResumeModeManual
	}
	if doc.GlobalState.Status == "" {
		doc.GlobalState.Status = HeartbeatControlStatusActive
	}
	if doc.GlobalState.LastHumanEngagementAt == nil {
		t := formatTime(now)
		doc.GlobalState.LastHumanEngagementAt = &t
		doc.GlobalState.LastHumanEngagementReason = "initialization"
	}
	if doc.TeamOverrides == nil {
		doc.TeamOverrides = map[string]HeartbeatControlTeamOverride{}
	}
	if doc.TeamState == nil {
		doc.TeamState = map[string]HeartbeatControlState{}
	}
}

func validatePolicy(policy HeartbeatControlPolicy) error {
	if policy.ResumeMode != HeartbeatControlResumeModeManual {
		return fmt.Errorf("resumeMode must be %q", HeartbeatControlResumeModeManual)
	}
	if policy.PauseAfterDaysWithoutHumanEngagement < 1 || policy.PauseAfterDaysWithoutHumanEngagement > heartbeatControlMaxThresholdDays {
		return fmt.Errorf("pauseAfterDaysWithoutHumanEngagement must be between 1 and %d", heartbeatControlMaxThresholdDays)
	}
	if policy.WarningAfterDaysWithoutHumanEngagement < 1 || policy.WarningAfterDaysWithoutHumanEngagement > heartbeatControlMaxThresholdDays {
		return fmt.Errorf("warningAfterDaysWithoutHumanEngagement must be between 1 and %d", heartbeatControlMaxThresholdDays)
	}
	if policy.WarningAfterDaysWithoutHumanEngagement >= policy.PauseAfterDaysWithoutHumanEngagement {
		return fmt.Errorf("warningAfterDaysWithoutHumanEngagement must be less than pauseAfterDaysWithoutHumanEngagement")
	}
	return nil
}

func validateTeamOverride(global HeartbeatControlPolicy, override HeartbeatControlTeamOverride) error {
	mode := override.Mode
	if mode == "" {
		mode = HeartbeatControlTeamModeInherit
	}
	switch mode {
	case HeartbeatControlTeamModeInherit, HeartbeatControlTeamModeDisabled, HeartbeatControlTeamModeCustom:
	default:
		return fmt.Errorf("mode must be one of %q, %q, %q", HeartbeatControlTeamModeInherit, HeartbeatControlTeamModeDisabled, HeartbeatControlTeamModeCustom)
	}
	if override.ResumeMode != "" && override.ResumeMode != HeartbeatControlResumeModeManual {
		return fmt.Errorf("resumeMode must be %q", HeartbeatControlResumeModeManual)
	}
	if mode == HeartbeatControlTeamModeCustom {
		policy := global
		if override.PauseAfterDaysWithoutHumanEngagement != nil {
			policy.PauseAfterDaysWithoutHumanEngagement = *override.PauseAfterDaysWithoutHumanEngagement
		}
		if override.WarningAfterDaysWithoutHumanEngagement != nil {
			policy.WarningAfterDaysWithoutHumanEngagement = *override.WarningAfterDaysWithoutHumanEngagement
		}
		if override.ResumeMode != "" {
			policy.ResumeMode = override.ResumeMode
		}
		return validatePolicy(policy)
	}
	return nil
}

func (s *HeartbeatControlStore) evaluateLocked(doc *HeartbeatControlDocument, now time.Time) bool {
	changed := false
	if doc.GlobalPolicy.Enabled && doc.GlobalState.Status != HeartbeatControlStatusPausedManual {
		status, pausedReason := statusFromEngagement(doc.GlobalState.LastHumanEngagementAt, doc.GlobalPolicy, now)
		if doc.GlobalState.Status != status {
			doc.GlobalState.Status = status
			changed = true
			if status == HeartbeatControlStatusPausedAuto {
				t := formatTime(now)
				doc.GlobalState.LastPausedAt = &t
				doc.GlobalState.LastPausedReason = pausedReason
			}
		}
	} else if !doc.GlobalPolicy.Enabled && doc.GlobalState.Status != HeartbeatControlStatusPausedManual {
		doc.GlobalState.Status = HeartbeatControlStatusActive
		changed = true
	}

	for teamID, override := range doc.TeamOverrides {
		state := doc.TeamState[teamID]
		if state.Status == HeartbeatControlStatusPausedManual {
			continue
		}
		if override.Mode == HeartbeatControlTeamModeDisabled || override.Mode == HeartbeatControlTeamModeInherit || override.Mode == "" {
			if state.Status != HeartbeatControlStatusActive && state.Status != "" {
				state.Status = HeartbeatControlStatusActive
				doc.TeamState[teamID] = state
				changed = true
			}
			continue
		}
		policy := effectiveTeamPolicy(doc.GlobalPolicy, override)
		status, pausedReason := statusFromEngagement(state.LastHumanEngagementAt, policy, now)
		if state.LastHumanEngagementAt == nil {
			status, pausedReason = statusFromEngagement(doc.GlobalState.LastHumanEngagementAt, policy, now)
		}
		if state.Status != status {
			state.Status = status
			changed = true
			if status == HeartbeatControlStatusPausedAuto {
				t := formatTime(now)
				state.LastPausedAt = &t
				state.LastPausedReason = pausedReason
			}
		}
		doc.TeamState[teamID] = state
	}
	return changed
}

func statusFromEngagement(last *string, policy HeartbeatControlPolicy, now time.Time) (string, string) {
	if !policy.Enabled {
		return HeartbeatControlStatusActive, ""
	}
	lastTime, ok := parseTimePtr(last)
	if !ok {
		return HeartbeatControlStatusActive, ""
	}
	pauseAt := lastTime.AddDate(0, 0, policy.PauseAfterDaysWithoutHumanEngagement)
	if !now.Before(pauseAt) {
		return HeartbeatControlStatusPausedAuto, fmt.Sprintf("no human engagement for %d days", policy.PauseAfterDaysWithoutHumanEngagement)
	}
	warnAt := lastTime.AddDate(0, 0, policy.WarningAfterDaysWithoutHumanEngagement)
	if !now.Before(warnAt) {
		return HeartbeatControlStatusWarningIdle, ""
	}
	return HeartbeatControlStatusActive, ""
}

func (s *HeartbeatControlStore) recordHumanEngagementLocked(doc *HeartbeatControlDocument, event HumanEngagementEvent) {
	if event.Attribution.Kind != store.KnowledgeKindOperatorDirect {
		return
	}
	now := formatTime(s.now())
	reason := strings.TrimSpace(event.Reason)
	if reason == "" {
		reason = "operator-engagement"
	}
	doc.GlobalState.LastHumanEngagementAt = &now
	doc.GlobalState.LastHumanEngagementReason = reason
	doc.GlobalState.LastHumanEngagementTeamID = event.TeamID
	if doc.GlobalState.Status == HeartbeatControlStatusPausedAuto || doc.GlobalState.Status == HeartbeatControlStatusWarningIdle {
		doc.GlobalState.Status = HeartbeatControlStatusActive
		doc.GlobalState.LastPausedAt = nil
		doc.GlobalState.LastPausedReason = ""
	}
	if event.TeamID != "" {
		state := doc.TeamState[event.TeamID]
		state.LastHumanEngagementAt = &now
		state.LastHumanEngagementReason = reason
		state.LastHumanEngagementTeamID = event.TeamID
		if state.Status == HeartbeatControlStatusPausedAuto || state.Status == HeartbeatControlStatusWarningIdle {
			state.Status = HeartbeatControlStatusActive
			state.LastPausedAt = nil
			state.LastPausedReason = ""
		}
		doc.TeamState[event.TeamID] = state
	}
}

func (s *HeartbeatControlStore) globalStatusLocked(doc *HeartbeatControlDocument) *HeartbeatControlStatusResponse {
	resp := statusResponseFromState(heartbeatControlGlobalScope, "", doc.GlobalState, doc.GlobalPolicy, nil)
	resp.GlobalPolicy = doc.GlobalPolicy
	return resp
}

func (s *HeartbeatControlStore) teamStatusLocked(doc *HeartbeatControlDocument, teamID string) *HeartbeatControlStatusResponse {
	global := s.globalStatusLocked(doc)
	override := doc.TeamOverrides[teamID]
	state := doc.TeamState[teamID]
	policy := effectiveTeamPolicy(doc.GlobalPolicy, override)
	scope := "team"
	status := state.Status
	if status == "" {
		status = HeartbeatControlStatusActive
	}
	if global.Status == HeartbeatControlStatusPausedManual || global.Status == HeartbeatControlStatusPausedAuto {
		scope = heartbeatControlGlobalScope
		status = global.Status
		state.LastPausedAt = global.PausedAt
		state.LastPausedReason = global.PausedReason
	}
	if global.Status == HeartbeatControlStatusWarningIdle && status == HeartbeatControlStatusActive {
		status = HeartbeatControlStatusWarningIdle
	}
	state.Status = status
	if state.LastHumanEngagementAt == nil {
		state.LastHumanEngagementAt = global.LastHumanEngagementAt
		state.LastHumanEngagementReason = global.LastHumanEngagementReason
		state.LastHumanEngagementTeamID = global.LastHumanEngagementTeamID
	}
	resp := statusResponseFromState(scope, teamID, state, policy, &override)
	return resp
}

func statusResponseFromState(scope, teamID string, state HeartbeatControlState, policy HeartbeatControlPolicy, override *HeartbeatControlTeamOverride) *HeartbeatControlStatusResponse {
	resp := &HeartbeatControlStatusResponse{
		Scope:                     scope,
		TeamID:                    teamID,
		Status:                    state.Status,
		EffectivePolicy:           policy,
		LastHumanEngagementAt:     state.LastHumanEngagementAt,
		LastHumanEngagementReason: state.LastHumanEngagementReason,
		LastHumanEngagementTeamID: state.LastHumanEngagementTeamID,
		PausedAt:                  state.LastPausedAt,
		PausedReason:              state.LastPausedReason,
		ResumeHint:                resumeHint(scope, teamID),
	}
	if override != nil && override.Mode != "" {
		v := *override
		resp.TeamOverride = &v
	}
	if last, ok := parseTimePtr(state.LastHumanEngagementAt); ok {
		warnAt := formatTime(last.AddDate(0, 0, policy.WarningAfterDaysWithoutHumanEngagement))
		pauseAt := formatTime(last.AddDate(0, 0, policy.PauseAfterDaysWithoutHumanEngagement))
		resp.WarningAt = &warnAt
		resp.AutoPauseAt = &pauseAt
	}
	return resp
}

func effectiveTeamPolicy(global HeartbeatControlPolicy, override HeartbeatControlTeamOverride) HeartbeatControlPolicy {
	policy := global
	if override.Mode == HeartbeatControlTeamModeDisabled {
		policy.Enabled = false
		return policy
	}
	if override.Mode == HeartbeatControlTeamModeCustom {
		if override.PauseAfterDaysWithoutHumanEngagement != nil {
			policy.PauseAfterDaysWithoutHumanEngagement = *override.PauseAfterDaysWithoutHumanEngagement
		}
		if override.WarningAfterDaysWithoutHumanEngagement != nil {
			policy.WarningAfterDaysWithoutHumanEngagement = *override.WarningAfterDaysWithoutHumanEngagement
		}
		if override.ResumeMode != "" {
			policy.ResumeMode = override.ResumeMode
		}
	}
	return policy
}

func pausedMessage(status *HeartbeatControlStatusResponse) string {
	if status.PausedReason != "" {
		return fmt.Sprintf("Heartbeats are paused: %s.", status.PausedReason)
	}
	if status.LastHumanEngagementAt != nil && status.Status == HeartbeatControlStatusPausedAuto {
		return fmt.Sprintf("Heartbeats are auto-paused because no human engagement was recorded since %s.", *status.LastHumanEngagementAt)
	}
	return "Heartbeats are paused."
}

func resumeHint(scope, teamID string) string {
	if scope == "team" && teamID != "" {
		return fmt.Sprintf("Run prompt-manager team heartbeat-control %s resume or use the UI Resume button.", teamID)
	}
	return "Run prompt-manager heartbeat-control resume or use the UI Resume button."
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTimePtr(v *string) (time.Time, bool) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*v))
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func attributionFromRequest(r *http.Request, teamID string) (store.AttributionInfo, bool, error) {
	info, err := parseAttributionHeader(r.Header.Get(attributionHeaderName))
	if err != nil {
		if errors.Is(err, errAttributionRequired) {
			return store.AttributionInfo{}, false, nil
		}
		return store.AttributionInfo{}, false, err
	}
	if err := validateAttribution(info, teamID); err != nil {
		return store.AttributionInfo{}, false, err
	}
	return info, info.Kind == store.KnowledgeKindOperatorDirect, nil
}
