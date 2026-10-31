package main

// Usage modes in Web Console. Reads go straight to the shared mode file
// (cli-core/usagemode) and never touch Agent Manager, so the launcher keeps
// working when Agent Manager is down. Starting and ending a mode needs a
// verified human; agent and service identities are refused.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/usagemode"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	capsH "web-console/handlers/capabilities"
)

const (
	usageModeLabel      = "Save usage"
	usageModeMaxMinutes = 7 * 24 * 60
)

var (
	usageModeMu       sync.Mutex
	usageModeModelRE  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)
	usageModeProvider = map[string]string{"codex": "openai", "claude": "anthropic"}
	usageModeModelFlg = map[string]string{"codex": "-m", "claude": "--model"}
)

type usageModeView struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Runner    string `json:"runner"`
	StartedAt string `json:"startedAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	UntilKind string `json:"untilKind"`
	StartedBy string `json:"startedBy,omitempty"`
	// Kind is "save" for a mode that lowers the model, or "learn" for a
	// controlled comparison, which carries no settings of its own: the model
	// depends on the arm each launch is assigned.
	Kind string `json:"kind"`
	// Role, SplitPercent and SessionsOptIn describe a learn comparison.
	Role          string `json:"role,omitempty"`
	SplitPercent  int    `json:"splitPercent,omitempty"`
	SessionsOptIn bool   `json:"sessionsOptIn,omitempty"`
	Paused        bool   `json:"paused,omitempty"`
}

// usageModeAgent is what a new session of one agent will use. Model and
// ModeID are empty when no mode applies. FullPowerArgs is set only when the
// resource's normal default differs from what the mode selects.
type usageModeAgent struct {
	AgentID       string `json:"agentId"`
	Label         string `json:"label"`
	Runner        string `json:"runner"`
	LowerModel    string `json:"lowerModel"`
	UsualModel    string `json:"usualModel"`
	ModeID        string `json:"modeId,omitempty"`
	Model         string `json:"model,omitempty"`
	FullPowerArgs string `json:"fullPowerArgs,omitempty"`
	// LearnModeID is set when a Learn comparison includes this agent's operator
	// sessions. The session's model then depends on the arm its launch is
	// assigned, so Model stays empty: the launcher cannot say which model it will use.
	LearnModeID string `json:"learnModeId,omitempty"`
}

type usageModeResponse struct {
	Modes      []usageModeView  `json:"modes"`
	Agents     []usageModeAgent `json:"agents"`
	CanControl bool             `json:"canControl"`
}

type usageModeStartRequest struct {
	AgentID string `json:"agentId"`
	// Kind is "minutes", "off" or "reset".
	Kind          string `json:"kind"`
	Minutes       int    `json:"minutes"`
	WindowMinutes int    `json:"windowMinutes"`
}

type usageModePolicy struct {
	Roles map[string]struct {
		Model string `json:"model"`
	} `json:"roles"`
}

func (s *Server) mountUsageMode() {
	s.router.HandleFunc("/api/v1/usage-mode", s.handleUsageModeGet).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/usage-mode/start", s.handleUsageModeStart).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/usage-mode/end", s.handleUsageModeEnd).Methods(http.MethodPost)
}

func usageModePolicyBytes(resource string) ([]byte, error) {
	var roots []string
	for _, key := range []string{"VROOLI_RESOURCE_ROOT", "VROOLI_ROOT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			roots = append(roots, v)
		}
	}
	if wd, err := os.Getwd(); err == nil {
		for dir, i := wd, 0; i < 8; dir, i = filepath.Dir(dir), i+1 {
			roots = append(roots, dir)
		}
	}
	for _, root := range roots {
		for _, path := range []string{filepath.Join(root, "resources", resource, "model-policy.json"), filepath.Join(root, resource, "model-policy.json")} {
			if raw, err := os.ReadFile(path); err == nil {
				return raw, nil
			}
		}
	}
	return nil, fmt.Errorf("no model policy for %s", resource)
}

// usageModeAgents lists the agents whose resource declares usage tiers.
func usageModeTiers(agentID string) (usagemode.Tiers, string, string, bool) {
	resource, ok := agentResource(agentID)
	if !ok {
		return usagemode.Tiers{}, "", "", false
	}
	raw, err := usageModePolicyBytes(resource)
	if err != nil {
		return usagemode.Tiers{}, "", "", false
	}
	tiers, err := usagemode.LoadTiers(raw)
	if err != nil {
		return usagemode.Tiers{}, "", "", false
	}
	var policy usageModePolicy
	_ = json.Unmarshal(raw, &policy)
	return tiers, strings.TrimSpace(policy.Roles["code.default"].Model), usagemode.NormalizeRunner(resource), true
}

// usageModeHigherModel is the model a mode's session model stands in for, so
// full power means the tier above it even when the role default already equals
// the lower model (Claude's code.default is the mode's Sonnet). It prefers the
// shortest name, which is the alias.
func usageModeHigherModel(tiers usagemode.Tiers, fallback string) string {
	higher := ""
	for from, to := range tiers.ModelMap {
		if to != tiers.SessionModel {
			continue
		}
		if higher == "" || len(from) < len(higher) || (len(from) == len(higher) && from < higher) {
			higher = from
		}
	}
	if higher == "" {
		return fallback
	}
	return higher
}

func buildUsageModeResponse(now time.Time, canControl bool) usageModeResponse {
	resp := usageModeResponse{Modes: []usageModeView{}, Agents: []usageModeAgent{}, CanControl: canControl}
	for _, mode := range usagemode.Read(now) {
		view := usageModeView{ID: mode.ID, Label: mode.Label, Runner: mode.Scope.Runner, StartedAt: mode.StartedAt.UTC().Format(time.RFC3339), UntilKind: mode.Until.Kind, StartedBy: mode.StartedBy, Kind: "save"}
		if mode.IsLearn() && mode.Learn != nil {
			view.Kind, view.Role, view.Paused = usagemode.KindLearn, mode.Scope.Role, mode.Learn.Paused
			view.SplitPercent = int(mode.Learn.Split()*100 + 0.5)
			view.SessionsOptIn = mode.Learn.Sessions != nil && mode.Learn.Sessions.OptIn
		}
		if mode.ExpiresAt != nil {
			view.ExpiresAt = mode.ExpiresAt.UTC().Format(time.RFC3339)
		}
		resp.Modes = append(resp.Modes, view)
	}
	sort.Slice(resp.Modes, func(i, j int) bool { return resp.Modes[i].StartedAt > resp.Modes[j].StartedAt })
	for _, def := range capabilityprobe.AITools {
		tiers, usual, runner, ok := usageModeTiers(def.ID)
		if !ok || usual == "" || tiers.SessionModel == "" {
			continue
		}
		usual = usageModeHigherModel(tiers, usual)
		agent := usageModeAgent{AgentID: def.ID, Label: def.Label, Runner: runner, LowerModel: tiers.SessionModel, UsualModel: usual}
		if mode, found := usagemode.Resolve(usagemode.Query{Runner: runner, Now: now}); found {
			if settings, has := mode.SettingsFor(runner); has && settings.SessionModel != "" {
				agent.ModeID, agent.Model = mode.ID, settings.SessionModel
				if settings.SessionModel != usual && usageModeModelRE.MatchString(usual) && usageModeModelFlg[runner] != "" {
					agent.FullPowerArgs = usageModeModelFlg[runner] + " " + usual
				}
			}
		} else if learn, joined := usagemode.ResolveLearnSession(runner, now); joined {
			// A save mode takes priority over a comparison, so this branch is reached only when none applies.
			agent.LearnModeID = learn.ID
		}
		resp.Agents = append(resp.Agents, agent)
	}
	return resp
}

func writeUsageModeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeUsageModeError(w http.ResponseWriter, status int, message string) {
	writeUsageModeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) handleUsageModeGet(w http.ResponseWriter, r *http.Request) {
	_, err := authn.RequireIndependentHuman(r.Context())
	writeUsageModeJSON(w, http.StatusOK, buildUsageModeResponse(time.Now(), err == nil))
}

// requireUsageModeOperator admits only a verified independent human on a
// same-origin request: the requirement Agent Manager applies to the same
// operation. A human credential forwarded by an agent (for example the owner
// delegation inside a Web Console terminal) carries agent provenance and is
// refused, so loopback alone never makes a caller the operator. The operator's
// browser reaches here through the Web Console session, which has no agent
// provenance.
func requireUsageModeOperator(w http.ResponseWriter, r *http.Request) (string, bool) {
	if !webConsoleSameOrigin(w, r) {
		return "", false
	}
	principal, err := authn.RequireIndependentHuman(r.Context())
	if err != nil || !principal.IsHuman() {
		writeUsageModeError(w, http.StatusForbidden, "usage modes can only be changed by the signed-in operator")
		return "", false
	}
	return principal.OwnerSubject(), true
}

func (s *Server) usageResetFor(ctx context.Context, agentID string) (capsH.AgentUsageResult, error) {
	if s.usageModeQuota != nil {
		return s.usageModeQuota(ctx, agentID)
	}
	return s.readAgentUsage(ctx, "local", agentID)
}

func (s *Server) handleUsageModeStart(w http.ResponseWriter, r *http.Request) {
	operator, ok := requireUsageModeOperator(w, r)
	if !ok {
		return
	}
	var req usageModeStartRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil {
		writeUsageModeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	tiers, _, runner, found := usageModeTiers(strings.TrimSpace(req.AgentID))
	if !found {
		writeUsageModeError(w, http.StatusBadRequest, "this agent has no usage tiers")
		return
	}
	now := time.Now()
	opts := usagemode.NewModeOptions{Label: usageModeLabel, Kind: "save", Scope: usagemode.Scope{Runner: runner}, StartedBy: operator, Source: "web-console", Now: now}
	switch req.Kind {
	case "minutes":
		if req.Minutes < 1 || req.Minutes > usageModeMaxMinutes {
			writeUsageModeError(w, http.StatusBadRequest, "duration must be between one minute and seven days")
			return
		}
		opts.Duration = time.Duration(req.Minutes) * time.Minute
	case "off":
	case "reset":
		provider := usageModeProvider[runner]
		if provider == "" || req.WindowMinutes <= 0 {
			writeUsageModeError(w, http.StatusBadRequest, "no reset is known for this agent")
			return
		}
		quota, err := s.usageResetFor(r.Context(), strings.TrimSpace(req.AgentID))
		var reset time.Time
		matches := 0
		if err == nil && quota.Match == "likely" {
			for _, window := range quota.Windows {
				if window.WindowMinutes != int64(req.WindowMinutes) {
					continue
				}
				matches++
				if at, perr := time.Parse(time.RFC3339, window.ResetsAt); perr == nil && at.After(now) {
					reset = at
				}
			}
		}
		if matches != 1 { // several pools share this window: which reset applies is unknown
			reset = time.Time{}
		}
		if reset.IsZero() {
			writeUsageModeError(w, http.StatusConflict, "no reset time is known for that window")
			return
		}
		opts.ExpiresAt, opts.UntilKind, opts.Provider, opts.WindowMinutes = &reset, usagemode.UntilWindowReset, provider, req.WindowMinutes
	default:
		writeUsageModeError(w, http.StatusBadRequest, "unknown duration")
		return
	}
	mode, err := usagemode.NewMode(tiers, opts)
	if err != nil {
		writeUsageModeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	usageModeMu.Lock()
	defer usageModeMu.Unlock()
	err = usagemode.Update(func(file *usagemode.File) error {
		file.Prune(now)
		kept := []usagemode.Mode{}
		for _, existing := range file.Modes {
			if !existing.IsLearn() && usagemode.NormalizeRunner(existing.Scope.Runner) == runner && existing.Scope.Role == "" && existing.Scope.Profile == "" {
				continue // identical scope: the new mode replaces it; a comparison is never discarded
			}
			kept = append(kept, existing)
		}
		file.Modes = append(kept, mode)
		return nil
	})
	if err != nil {
		writeUsageModeError(w, http.StatusInternalServerError, "the usage mode could not be saved")
		return
	}
	writeUsageModeJSON(w, http.StatusOK, buildUsageModeResponse(time.Now(), true))
}

func (s *Server) handleUsageModeEnd(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUsageModeOperator(w, r); !ok {
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeUsageModeError(w, http.StatusBadRequest, "a mode id is required")
		return
	}
	usageModeMu.Lock()
	defer usageModeMu.Unlock()
	removed := false
	err := usagemode.Update(func(file *usagemode.File) error {
		now := time.Now()
		file.Prune(now)
		kept := []usagemode.Mode{}
		for _, existing := range file.Modes {
			if existing.ID == req.ID {
				removed = true
				continue
			}
			kept = append(kept, existing)
		}
		if !removed {
			return errUsageModeNotActive
		}
		file.Modes = kept
		return nil
	})
	if errors.Is(err, errUsageModeNotActive) {
		writeUsageModeError(w, http.StatusNotFound, "that usage mode is not active")
		return
	}
	if err != nil {
		writeUsageModeError(w, http.StatusInternalServerError, "the usage mode could not be ended")
		return
	}
	writeUsageModeJSON(w, http.StatusOK, buildUsageModeResponse(time.Now(), true))
}

var errUsageModeNotActive = errors.New("usage mode not active")
