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
		Windows                           []struct {
			Pool          string   `json:"pool"`
			WindowMinutes int64    `json:"windowMinutes"`
			UsedPercent   *float64 `json:"usedPercent"`
			ResetsAt      string   `json:"resetsAt"`
		} `json:"windows"`
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
	observed, err := time.Parse(time.RFC3339, data.ObservedAt)
	if err != nil || observed.After(time.Now().Add(time.Minute)) || time.Since(observed) > 15*time.Minute {
		result.Reason = "The latest usage reading is out of date"
		return result
	}
	for _, w := range data.Windows {
		if w.UsedPercent == nil || math.IsNaN(*w.UsedPercent) || *w.UsedPercent < 0 || *w.UsedPercent > 100 || w.WindowMinutes <= 0 || len(w.Pool) > 128 {
			continue
		}
		if w.ResetsAt != "" {
			reset, err := time.Parse(time.RFC3339, w.ResetsAt)
			if err != nil || !reset.After(time.Now()) {
				continue
			}
		}
		result.Windows = append(result.Windows, capsH.AgentUsageWindow{Pool: w.Pool, WindowMinutes: w.WindowMinutes, UsedPercent: *w.UsedPercent, ResetsAt: w.ResetsAt})
	}
	if len(result.Windows) == 0 {
		result.Reason = "No current usage windows are available"
		return result
	}
	result.Match, result.Source, result.ObservedAt = "likely", data.Source, data.ObservedAt
	result.Reason = strings.TrimSpace(data.Reason + " The launch may use a different account.")
	return result
}
