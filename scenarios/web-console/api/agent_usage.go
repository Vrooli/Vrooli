package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	capsH "web-console/handlers/capabilities"
)

// usageReadingMaxAge bounds how long a reading describes the current standing.
// It matches Agent Manager's pricing.QuotaFreshnessMaxAge, which this module
// cannot import, so the launcher and the owner of the data agree.
const usageReadingMaxAge = 6 * time.Hour

// readAgentUsage never projects local provider readings onto a remote target.
// Agent Manager's service environment is an estimate of a future shell's
// credentials, not a verified binding to that shell's account.
func (s *Server) readAgentUsage(ctx context.Context, targetID, agentID string) (capsH.AgentUsageResult, error) {
	targetID, agentID = strings.TrimSpace(targetID), strings.TrimSpace(agentID)
	if targetID == "" {
		return capsH.AgentUsageResult{}, fmt.Errorf("target is required")
	}
	if _, ok := agentResource(agentID); !ok {
		return capsH.AgentUsageResult{}, fmt.Errorf("unknown agent")
	}
	result := capsH.AgentUsageResult{TargetID: targetID, AgentID: agentID, Match: "unavailable"}
	if targetID != "local" {
		if _, ok := s.targetByID(targetID); !ok {
			return capsH.AgentUsageResult{}, fmt.Errorf("%w: %s", capsH.ErrTargetNotFound, targetID)
		}
		result.Reason = "Usage reading is not available for this machine"
		return result, nil
	}
	// Allow the provider owner's bounded ten-second read to finish while still
	// ensuring an unavailable service cannot hold the launcher open forever.
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		result.Reason = "Usage service is unavailable"
		return result, nil
	}
	return fetchLocalAgentUsage(ctx, http.DefaultClient, base, result), nil
}

func fetchLocalAgentUsage(ctx context.Context, client *http.Client, base string, result capsH.AgentUsageResult) capsH.AgentUsageResult {
	u, err := url.Parse(strings.TrimRight(base, "/") + "/api/v1/pricing/live-quota")
	if err != nil {
		result.Reason = "Usage service is unavailable"
		return result
	}
	q := u.Query()
	q.Set("agent", result.AgentID)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		result.Reason = "Usage service is unavailable"
		return result
	}
	resp, err := client.Do(req)
	if err != nil {
		result.Reason = "Usage service is unavailable"
		return result
	}
	defer resp.Body.Close()
	var data struct {
		Match, Reason, Source, ObservedAt string
		Windows                           []localUsageWindow `json:"windows"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&data) != nil {
		result.Reason = "Usage service returned an invalid response"
		return result
	}
	// This adapter has no launch binding; never promote a service-level
	// confidence claim to a verified future account.
	if data.Match != "likely" {
		result.Reason = data.Reason
		if result.Reason == "" {
			result.Reason = "Usage is unavailable"
		}
		return result
	}
	return localUsageResult(result, data.Reason, data.Source, data.ObservedAt, data.Windows, time.Now())
}

// localUsageWindow is one window in Agent Manager's live-quota response.
// ObservedAt is set when this window's reading is older than the response.
type localUsageWindow struct {
	Pool          string   `json:"pool"`
	WindowMinutes int64    `json:"windowMinutes"`
	UsedPercent   *float64 `json:"usedPercent"`
	ResetsAt      string   `json:"resetsAt"`
	ObservedAt    string   `json:"observedAt"`
}

// localUsageResult keeps each window that still describes the current standing
// and, when none does, says why. Windows can have different ages.
func localUsageResult(result capsH.AgentUsageResult, reason, source, responseObserved string, windows []localUsageWindow, now time.Time) capsH.AgentUsageResult {
	var newest, newestStale time.Time
	resetPassed := false
	for _, w := range windows {
		if w.UsedPercent == nil || math.IsNaN(*w.UsedPercent) || *w.UsedPercent < 0 || *w.UsedPercent > 100 || w.WindowMinutes <= 0 || len(w.Pool) > 128 {
			continue
		}
		observedAt := w.ObservedAt
		if observedAt == "" {
			observedAt = responseObserved
		}
		observed, err := time.Parse(time.RFC3339, observedAt)
		if err != nil || observed.After(now.Add(time.Minute)) {
			continue
		}
		if now.Sub(observed) > usageReadingMaxAge {
			if observed.After(newestStale) {
				newestStale = observed
			}
			continue
		}
		if w.ResetsAt != "" {
			reset, err := time.Parse(time.RFC3339, w.ResetsAt)
			if err != nil {
				continue
			}
			if !reset.After(now) {
				resetPassed = true
				continue
			}
		}
		if observed.After(newest) {
			newest = observed
		}
		result.Windows = append(result.Windows, capsH.AgentUsageWindow{Pool: w.Pool, WindowMinutes: w.WindowMinutes, UsedPercent: *w.UsedPercent, ResetsAt: w.ResetsAt, ObservedAt: observedAt})
	}
	if len(result.Windows) == 0 {
		switch {
		case resetPassed:
			result.Reason = "The usage window has reset and no new reading has arrived yet"
		case !newestStale.IsZero():
			result.Reason = fmt.Sprintf("The latest usage reading is %d hours old", int(now.Sub(newestStale).Hours()))
		default:
			result.Reason = "No current usage windows are available"
		}
		return result
	}
	result.Match, result.Source, result.ObservedAt = "likely", source, newest.UTC().Format(time.RFC3339)
	result.Reason = strings.TrimSpace(reason + " The launch may use a different account.")
	return result
}
