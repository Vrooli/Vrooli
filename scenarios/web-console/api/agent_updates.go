package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	capsH "web-console/handlers/capabilities"
)

const (
	agentUpdateTTL         = 5 * time.Minute
	agentUpdateFailureTTL  = 30 * time.Second
	agentObservationMaxAge = 2 * time.Minute
	agentUpdateTimeout     = 20 * time.Second
)

// agentUpdate is an installed-tool observation, not a promise that the
// governed installer can install the newest upstream release. The executable
// path stays on the server; the browser gets only the evidence verdict.
type agentUpdate struct {
	TargetID      string               `json:"target_id"`
	AgentID       string               `json:"agent_id"`
	Resource      string               `json:"resource"`
	Installed     string               `json:"installed,omitempty"`
	Pinned        string               `json:"pinned,omitempty"`
	Latest        string               `json:"latest,omitempty"`
	Status        upstreamcheck.Status `json:"status"`
	CheckedAt     time.Time            `json:"checked_at,omitempty"`
	SourceKind    string               `json:"source_kind,omitempty"`
	SourceID      string               `json:"source_id,omitempty"`
	ProbeVerified bool                 `json:"probe_verified"`
	Note          string               `json:"note,omitempty"`
}

type agentUpdateEntry struct {
	value   agentUpdate
	expires time.Time
}
type agentUpdateFlight struct {
	done  chan struct{}
	value agentUpdate
	err   error
}
type agentUpdateCache struct {
	mu      sync.Mutex
	items   map[string]agentUpdateEntry
	pending map[string]*agentUpdateFlight
}

// read coalesces simultaneous requests for one target/agent and bounds cache
// lifetime. The cache key includes the machine ID; no local report can become
// a remote result. A caller canceling while waiting does not cancel the owner.
func (c *agentUpdateCache) read(ctx context.Context, key string, load func(context.Context) (agentUpdate, error)) (agentUpdate, error) {
	c.mu.Lock()
	if entry, ok := c.items[key]; ok && time.Now().Before(entry.expires) {
		c.mu.Unlock()
		return entry.value, nil
	}
	if flight := c.pending[key]; flight != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return agentUpdate{}, ctx.Err()
		case <-flight.done:
			return flight.value, flight.err
		}
	}
	flight := &agentUpdateFlight{done: make(chan struct{})}
	if c.pending == nil {
		c.pending = make(map[string]*agentUpdateFlight)
	}
	c.pending[key] = flight
	c.mu.Unlock()
	value, err := load(ctx)
	c.mu.Lock()
	flight.value, flight.err = value, err
	delete(c.pending, key)
	if err == nil {
		if c.items == nil {
			c.items = make(map[string]agentUpdateEntry)
		}
		for k, entry := range c.items {
			if time.Now().After(entry.expires) {
				delete(c.items, k)
			}
		}
		// A long-running console must not retain an unlimited history of
		// devices and changing installed versions.
		if len(c.items) >= 128 {
			c.items = make(map[string]agentUpdateEntry)
		}
		ttl := agentUpdateTTL
		if value.Status == upstreamcheck.StatusUnknown {
			ttl = agentUpdateFailureTTL
		}
		c.items[key] = agentUpdateEntry{value: value, expires: time.Now().Add(ttl)}
	}
	close(flight.done)
	c.mu.Unlock()
	return value, err
}

func agentResource(agentID string) (string, bool) {
	for _, def := range capabilityprobe.AITools {
		if def.ID == agentID {
			return cliutil.CodingAgentForAlias(def.Command)
		}
	}
	return "", false
}

func (s *Server) getAgentUpdate(ctx context.Context, targetID, agentID string) (agentUpdate, error) {
	targetID, agentID = strings.TrimSpace(targetID), strings.TrimSpace(agentID)
	resource, ok := agentResource(agentID)
	if !ok || targetID == "" {
		return agentUpdate{}, fmt.Errorf("unknown target or agent")
	}
	// The current catalog must still contain the selected target even if a
	// previous observation is cached.
	// The full local catalog probes every agent. This endpoint only needs the
	// selected agent, so do not trigger all five potentially slow probes for
	// each card or let those probes delay the bounded update lookup.
	var target targetConnection
	if targetID == "local" {
		target = targetConnection{Target: targetmodel.Target{ID: "local", Available: true, OS: runtime.GOOS, Architecture: runtime.GOARCH}}
	} else {
		target, ok = s.targetByID(targetID)
		if !ok {
			return agentUpdate{}, fmt.Errorf("%w: %s", capsH.ErrTargetNotFound, targetID)
		}
	}
	if !target.Available {
		return agentUpdate{TargetID: targetID, AgentID: agentID, Resource: resource, Status: upstreamcheck.StatusUnknown, Note: "The selected machine is unavailable"}, nil
	}
	var observation capabilityprobe.Observation
	if targetID == "local" {
		for _, definition := range capabilityprobe.AITools {
			if definition.ID == agentID {
				results := capabilityprobe.Probe(ctx, []capabilityprobe.Definition{definition})
				if len(results) == 1 {
					observation = results[0]
				}
				break
			}
		}
	} else {
		for _, item := range target.Capabilities {
			if item.ID == agentID {
				observation = item
				break
			}
		}
	}
	if observation.State != capabilityprobe.Ready || observation.ProbedAt.IsZero() || observation.ProbedAt.After(time.Now()) || time.Since(observation.ProbedAt) > agentObservationMaxAge {
		return agentUpdate{TargetID: targetID, AgentID: agentID, Resource: resource, Status: upstreamcheck.StatusUnknown, Note: "The installed agent has not been verified on this machine"}, nil
	}
	// Probe timestamp changes at every heartbeat. Key on the facts that affect
	// comparison, not the timestamp, to avoid a fresh network request each time.
	key := strings.Join([]string{targetID, agentID, target.OS, target.Architecture, observation.Path, observation.Version}, "\x00")
	result, err := s.agentUpdates.read(ctx, key, func(ctx context.Context) (agentUpdate, error) {
		ctx, cancel := context.WithTimeout(ctx, agentUpdateTimeout)
		defer cancel()
		var data []byte
		var err error
		if targetID == "local" {
			data, err = localAgentUpdate(ctx, resource)
		} else {
			data, err = s.remoteAgentUpdate(ctx, target, resource)
		}
		unknown := agentUpdate{TargetID: targetID, AgentID: agentID, Resource: resource, Status: upstreamcheck.StatusUnknown}
		if err != nil {
			unknown.Note = "Update check unavailable"
			return unknown, nil
		}
		var envelope struct {
			Success  bool                          `json:"success"`
			Upstream upstreamcheck.AggregateReport `json:"upstream_check"`
		}
		if len(data) > 512*1024 || json.NewDecoder(bytes.NewReader(data)).Decode(&envelope) != nil || !envelope.Success || len(envelope.Upstream.Resources) != 1 {
			unknown.Note = "Update check returned an invalid response"
			return unknown, nil
		}
		rep := envelope.Upstream.Resources[0]
		if rep.Name != resource {
			unknown.Note = "Update check returned a different resource"
			return unknown, nil
		}
		return projectAgentUpdate(unknown, rep, observation, target.OS, target.Architecture, time.Now().UTC()), nil
	})
	if err == nil && result.ProbeVerified && (result.CheckedAt.IsZero() || result.CheckedAt.After(time.Now()) || time.Since(result.CheckedAt) > agentUpdateTTL) {
		result.Status = upstreamcheck.StatusUnknown
		result.ProbeVerified = false
		result.Note = "The update observation is out of date"
	}
	return result, err
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > 512*1024-b.Len() {
		return 0, errors.New("upstream check exceeded output limit")
	}
	return b.Buffer.Write(p)
}

func localAgentUpdate(ctx context.Context, resource string) ([]byte, error) {
	var out boundedOutput
	cmd := exec.CommandContext(ctx, "vrooli", "resource", "upstream-check", resource, "--json")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (s *Server) readAgentUpdate(ctx context.Context, targetID, agentID string) (capsH.AgentUpdateResult, error) {
	update, err := s.getAgentUpdate(ctx, targetID, agentID)
	if err != nil {
		return capsH.AgentUpdateResult{}, err
	}
	checked := ""
	if !update.CheckedAt.IsZero() {
		checked = update.CheckedAt.UTC().Format(time.RFC3339)
	}
	return capsH.AgentUpdateResult{TargetID: update.TargetID, AgentID: update.AgentID, Resource: update.Resource,
		Installed: update.Installed, Pinned: update.Pinned, Latest: update.Latest, Status: string(update.Status),
		CheckedAt: checked, SourceKind: update.SourceKind, SourceID: update.SourceID, ProbeVerified: update.ProbeVerified, Note: update.Note}, nil
}

func projectAgentUpdate(out agentUpdate, rep upstreamcheck.Report, observation capabilityprobe.Observation, goos, arch string, now time.Time) agentUpdate {
	out.Installed, out.Pinned, out.Latest = rep.Installed, rep.Pinned, rep.Latest
	out.CheckedAt, out.SourceKind, out.SourceID = rep.CheckedAt, rep.Source.Kind, rep.Source.ID
	out.Note = rep.Note
	// Both independent observations must be current, identify the same target
	// platform and executable, and agree on the installed version. Older nodes
	// lacking these facts are unknown, not implicitly matched.
	if rep.CheckedAt.IsZero() || rep.CheckedAt.After(now) || now.Sub(rep.CheckedAt) > agentUpdateTTL || observation.ProbedAt.IsZero() || observation.ProbedAt.After(now) || now.Sub(observation.ProbedAt) > agentObservationMaxAge || observation.State != capabilityprobe.Ready || rep.OS == "" || rep.Arch == "" || rep.OS != goos || rep.Arch != arch || rep.ExecutablePath == "" || rep.CommandPath == "" || observation.Path == "" || filepath.Clean(rep.CommandPath) != filepath.Clean(observation.Path) || rep.Installed == "" || upstreamcheck.ExtractSemver(observation.Version) != rep.Installed {
		out.Status = upstreamcheck.StatusUnknown
		out.Note = "Cannot verify the installed executable on this machine"
		return out
	}
	out.ProbeVerified = true
	// Never trust an unrecognized or internally inconsistent status from an
	// older resource implementation.
	out.Status = upstreamcheck.Compare(rep.Installed, rep.Latest)
	if out.Status == upstreamcheck.StatusUnknown && out.Note == "" {
		out.Note = "Upstream version is unavailable or ambiguous"
	}
	return out
}

func (s *Server) remoteAgentUpdate(ctx context.Context, target targetConnection, resource string) ([]byte, error) {
	if target.NodeID == "" {
		return nil, errors.New("target has no Bridge node")
	}
	base := strings.TrimRight(strings.TrimSpace(target.BaseURL), "/")
	if base == "" {
		var err error
		base, err = resolveBridgeScenarioURL(ctx)
		if err != nil {
			return nil, err
		}
	}
	client := relayconnect.NewRelayServiceClient(authenticatedHTTPClientWithTimeout(freshOwnerAuthorization(ctx, target.OwnerToken), target.ReauthToken, agentUpdateTimeout+time.Second), base)
	response, err := client.Call(ctx, connect.NewRequest(&relayv1.RelayCallRequest{
		NodeId: target.NodeID, Scenario: "vrooli", Command: "resource upstream-check", Args: []string{resource, "--json"}, TimeoutSeconds: int64(agentUpdateTimeout.Seconds()), MaxResponseBytes: 512 * 1024,
	}))
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil || response.Msg.Outcome != relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED || response.Msg.ExitCode != 0 {
		return nil, errors.New("target check was not completed")
	}
	return response.Msg.Data, nil
}
