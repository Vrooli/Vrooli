package discovery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	repocontract "github.com/vrooli/repo-contract-go"
)

// bridgeTargetResolver is the default provider adapter. It speaks only the
// public vrooli-bridge CLI surface; discovery keeps the target model and does
// not import Bridge's service internals.
type bridgeTargetResolver struct {
	runner CommandRunner
	path   string
}

type bridgeNodeList struct {
	Targets []targetmodel.Target `json:"targets"`
	Nodes   []struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		NodeID       string   `json:"node_id"`
		Online       bool     `json:"online"`
		Dispatchable bool     `json:"dispatchable"`
		Scopes       []string `json:"scopes"`
		Revoked      bool     `json:"revoked"`
	} `json:"nodes"`
}

func (r bridgeTargetResolver) ResolveTarget(ctx context.Context, name string) (targetmodel.Target, error) {
	raw, err := r.runner(ctx, r.path, "nodes", "list", "--json")
	if err != nil {
		return targetmodel.Target{}, err
	}
	var response bridgeNodeList
	if err := json.Unmarshal(raw, &response); err != nil {
		return targetmodel.Target{}, fmt.Errorf("decode bridge node list: %w", err)
	}
	for _, target := range response.Targets {
		if target.ID == name || target.Label == name || target.NodeID == name {
			return target, nil
		}
	}
	for _, node := range response.Nodes {
		if node.Name != name && node.ID != name && node.NodeID != name {
			continue
		}
		id := strings.TrimSpace(node.ID)
		if id == "" {
			id = strings.TrimSpace(node.NodeID)
		}
		return targetmodel.Target{
			ID: id, NodeID: strings.TrimSpace(node.NodeID), Label: node.Name,
			Platform: "remote", Available: node.Online && node.Dispatchable,
			Revoked: node.Revoked, Scopes: append([]string(nil), node.Scopes...),
			Transport:   targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: id, Available: node.Online && node.Dispatchable},
			BridgeTrust: &targetmodel.BridgeTrust{Registered: !node.Revoked, Online: node.Online, DispatchAuthorized: node.Dispatchable},
		}, nil
	}
	return targetmodel.Target{}, fmt.Errorf("node %q is not paired", name)
}

type bridgeRelay struct {
	runner CommandRunner
	path   string
}

type bridgeRelayEnvelope struct {
	CorrelationID string `json:"correlation_id"`
	Outcome       string `json:"outcome"`
	Data          string `json:"data"`
	Reason        string `json:"reason"`
	ExitCode      int32  `json:"exit_code"`
	TotalBytes    uint64 `json:"total_bytes"`
}

func (r bridgeRelay) Call(ctx context.Context, request RelayRequest) (RelayResponse, error) {
	args := []string{"relay", "call", "--json", "--node-id", request.NodeID, "--scenario", request.Scenario, "--command", request.Command}
	args = append(args, "--args")
	args = append(args, request.Args...)
	raw, err := r.runner(ctx, r.path, args...)
	if err != nil {
		return RelayResponse{}, err
	}
	var envelope bridgeRelayEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return RelayResponse{}, fmt.Errorf("decode bridge relay response: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(envelope.Data)
	if err != nil && envelope.Data != "" {
		return RelayResponse{}, fmt.Errorf("decode bridge relay data: %w", err)
	}
	return RelayResponse{
		CorrelationID: envelope.CorrelationID, Outcome: envelope.Outcome,
		Data: data, Reason: envelope.Reason, ExitCode: envelope.ExitCode,
		TotalBytes: envelope.TotalBytes,
	}, nil
}

func defaultCommandScope(command string) (string, bool) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return conservativeCommandScope(command)
	}
	catalog, err := scopecatalog.Build(root)
	if err != nil {
		return conservativeCommandScope(command)
	}
	scope, ok := catalog.LookupVerb(command)
	if !ok {
		return conservativeCommandScope(command)
	}
	return scope.Value, true
}

// conservativeCommandScope keeps the default adapter usable when an
// installation is running without a readable catalog artifact. It only admits
// the read-only status probe used by the default discovery operation; callers
// requiring other commands must provide a healthy catalog or an explicit
// CommandScopeResolver.
func conservativeCommandScope(command string) (string, bool) {
	if strings.TrimSpace(command) == "scenario status" {
		return "vrooli:read", true
	}
	return "", false
}
