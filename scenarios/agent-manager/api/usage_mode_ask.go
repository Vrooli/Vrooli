package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/usagemodes"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/operatordecision"
)

// UsageModeAskPath is where a caller asks the operator to start a timed mode.
const UsageModeAskPath = "/api/v1/usage/modes/ask"

// UsageModeAutoSavePath lists the auto-save rules (GET) and turns one on or off
// (POST). A rule starts Save usage by itself when a window is projected to run
// out, so only a verified human may write it.
const UsageModeAutoSavePath = "/api/v1/usage/modes/auto-save"

// handleUsageModeAsk raises a decision; it never starts a mode. The mode starts
// only when a verified human answers it (usagemodes.Notifier.Apply), so any
// caller may ask and none can start one.
func (s *Server) handleUsageModeAsk(w http.ResponseWriter, r *http.Request) {
	if s.usageModeNotifier == nil {
		http.Error(w, "usage mode decisions are unavailable", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Runner string `json:"runner"`
		Length string `json:"length"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "body must be JSON {runner, length}", http.StatusBadRequest)
		return
	}
	length, err := time.ParseDuration(body.Length)
	if err != nil {
		http.Error(w, "length must be a duration such as 30m", http.StatusBadRequest)
		return
	}
	id, err := s.usageModeNotifier.AskToStart(r.Context(), body.Runner, length)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isInvalidModeAsk(err):
			status = http.StatusBadRequest
		case errors.Is(err, operatordecision.ErrFlood):
			status = http.StatusTooManyRequests // enough undecided asks already wait for the operator
		}
		http.Error(w, err.Error(), status)
		return
	}
	state, note, _ := s.usageModeNotifier.AskState(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"decision_id": id, "state": state, "note": note})
}

func (s *Server) handleUsageModeAskState(w http.ResponseWriter, r *http.Request) {
	state, note, ok := s.usageModeNotifier.AskState(r.URL.Query().Get("id"))
	if !ok {
		http.Error(w, "unknown decision", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"state": state, "note": note})
}

func isInvalidModeAsk(err error) bool { return errors.Is(err, usagemodes.ErrInvalid) }

type autoSaveRuleBody struct {
	Runner    string     `json:"runner"`
	Since     time.Time  `json:"since"`
	By        string     `json:"by"`
	Starts    int        `json:"starts"`
	LastStart *time.Time `json:"last_start,omitempty"`
}

func autoSaveRulesBody(rules []usagemodes.AutoSaveRule) []autoSaveRuleBody {
	out := make([]autoSaveRuleBody, 0, len(rules))
	for _, rule := range rules {
		item := autoSaveRuleBody{Runner: rule.Runner, Since: rule.Since, By: rule.By, Starts: rule.Starts}
		if !rule.LastStart.IsZero() {
			last := rule.LastStart
			item.LastStart = &last
		}
		out = append(out, item)
	}
	return out
}

// handleUsageModeAutoSave reads or writes the auto-save rules. A write needs a
// verified independent human with agent-manager:write, the same gate as starting
// a mode: an agent can neither turn a rule on nor off.
func (s *Server) handleUsageModeAutoSave(w http.ResponseWriter, r *http.Request) {
	if s.usageModeNotifier == nil {
		http.Error(w, "usage mode rules are unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodPost {
		principal, err := authn.RequireIndependentHuman(r.Context())
		if err == nil {
			_, err = authn.RequireCapability(r.Context(), "agent-manager:write")
		}
		if err != nil {
			http.Error(w, "auto-save rules can be changed only by a verified operator, not an agent", http.StatusForbidden)
			return
		}
		var body struct {
			Runner  string `json:"runner"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
			http.Error(w, "body must be JSON {runner, enabled}", http.StatusBadRequest)
			return
		}
		if err := s.usageModeNotifier.SetAutoSave(body.Runner, body.Enabled, principal.Subject); err != nil {
			status := http.StatusInternalServerError
			if isInvalidModeAsk(err) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"rules": autoSaveRulesBody(s.usageModeNotifier.AutoSaveRules())})
}
