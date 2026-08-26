package vroolicli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioexec"
)

type bridgeNodeSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Online       bool   `json:"online"`
	Dispatchable bool   `json:"dispatchable"`
}

type bridgeNodeList struct {
	Nodes []bridgeNodeSummary `json:"nodes"`
}

type bridgeRelayEnvelope struct {
	Outcome  string `json:"outcome"`
	Data     string `json:"data"`
	Reason   string `json:"reason"`
	ExitCode int32  `json:"exit_code"`
}

// remoteScenarioCall uses the public Bridge CLI surface rather than inventing
// a second Connect/authentication stack inside the project CLI. The child
// command remains typed and the node-agent performs the actual local scenario
// operation after relay admission.
func (app *App) remoteScenarioCall(ctx *CommandContext, nodeName, scenario, command string, args []string, jsonOutput bool) ([]byte, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return nil, err
	}
	bridgeCLI, err := app.resolveScenarioCLIExecutable(ctx.Root, home, "vrooli-bridge")
	if err != nil {
		return nil, fmt.Errorf("resolve vrooli-bridge CLI: %w", err)
	}

	listRaw, err := app.runBridgeJSON(ctx, bridgeCLI, "nodes", "list")
	if err != nil {
		return nil, fmt.Errorf("list bridge nodes: %w", err)
	}
	var list bridgeNodeList
	if err := json.Unmarshal(listRaw, &list); err != nil {
		return nil, fmt.Errorf("decode bridge node list: %w", err)
	}
	nodeID, err := selectBridgeNode(list.Nodes, nodeName)
	if err != nil {
		return nil, err
	}

	callArgs := []string{
		"relay", "call",
		"--node-id", nodeID,
		"--scenario", scenario,
		"--command", command,
	}
	remoteArgs := append([]string(nil), args...)
	if jsonOutput {
		remoteArgs = append(remoteArgs, "--json")
	}
	if timeout := relayTimeoutArg(remoteArgs); timeout != "" {
		// The addressed scenario command carries its own lifecycle ceiling in
		// the forwarded args. Mirror that ceiling on the outer relay transport;
		// otherwise the Bridge CLI's default HTTP deadline cancels a valid slow
		// remote build before the node can return its lifecycle result.
		callArgs = append(callArgs, "--timeout", timeout)
	}
	if len(remoteArgs) > 0 {
		// vrooli-bridge's public CLI accepts relay arguments as CSV. Keep the
		// relay contract in one place so every remotely dispatched scenario
		// verb gets identical argument handling.
		callArgs = append(callArgs, "--args", strings.Join(remoteArgs, ","))
	}
	responseRaw, err := app.runBridgeJSON(ctx, bridgeCLI, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("relay %s to %q: %w", command, nodeName, err)
	}
	var envelope bridgeRelayEnvelope
	if err := json.Unmarshal(responseRaw, &envelope); err != nil {
		return nil, fmt.Errorf("decode relay response: %w", err)
	}
	if strings.TrimSpace(envelope.Data) == "" {
		if envelope.Reason == "" {
			envelope.Reason = "relay returned no data"
		}
		return nil, fmt.Errorf("remote scenario %s failed: outcome=%s exit_code=%d reason=%s", command, envelope.Outcome, envelope.ExitCode, envelope.Reason)
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("decode relay response data: %w", err)
	}
	return payload, nil
}

func relayTimeoutArg(args []string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] != "--timeout" {
			continue
		}
		seconds, err := strconv.Atoi(strings.TrimSpace(args[index+1]))
		if err == nil && seconds > 0 {
			return strconv.Itoa(seconds)
		}
	}
	return ""
}

func (app *App) runBridgeJSON(ctx *CommandContext, executable string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	commandArgs := append(append([]string(nil), args...), "--json")
	err := app.RunScenarioSubprocess(scenarioexec.SubprocessSpec{
		Name: executable, Args: commandArgs, Dir: ctx.Root,
		Env:    app.CommandEnv(ctx.Root, ctx.Globals),
		Stdout: &stdout, Stderr: &stderr,
	})
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail != "" {
			return nil, fmt.Errorf("%w: %s", err, detail)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func selectBridgeNode(nodes []bridgeNodeSummary, name string) (string, error) {
	matches := make([]bridgeNodeSummary, 0, len(nodes))
	for _, node := range nodes {
		if node.Name == name {
			matches = append(matches, node)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("remote node %q: node_unpaired_or_revoked", name)
	}
	ready := make([]bridgeNodeSummary, 0, len(matches))
	for _, node := range matches {
		if node.Online && node.Dispatchable {
			ready = append(ready, node)
		}
	}
	if len(ready) == 1 {
		return ready[0].ID, nil
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	// NodeRegistry returns newest-first. When no duplicate is ready, use the
	// newest record so the relay can return its authoritative offline/revoked
	// reason instead of turning a readiness failure into an ambiguous-name
	// failure. Multiple ready records remain unsafe to guess between.
	return matches[0].ID, nil
}
