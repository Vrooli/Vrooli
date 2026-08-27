package vroolicli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/nodeclient"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

const (
	remoteNodeListTimeout = 15 * time.Second
	remoteCallGrace       = 5 * time.Second
)

// remoteScenarioCall uses the shared typed node client. The project CLI only
// selects a registry record and renders the relay result; Bridge remains the
// authority for pairing, presence, scopes, and command admission.
func (app *App) remoteScenarioCall(_ *CommandContext, nodeName, scenario, command string, args []string, jsonOutput bool) ([]byte, error) {
	client := nodeclient.New(nodeclient.Config{
		Token:         firstNonEmptyEnv("VROOLI_BRIDGE_API_TOKEN", "VROOLI_API_TOKEN"),
		TokenProvider: resolveLocalOwnerToken,
	})
	nodes, err := client.List(context.Background(), remoteNodeListTimeout)
	if err != nil {
		return nil, fmt.Errorf("list bridge nodes: %w", err)
	}
	nodeID, err := selectBridgeNode(nodes, nodeName)
	if err != nil {
		return nil, err
	}

	remoteArgs := append([]string(nil), args...)
	if jsonOutput {
		remoteArgs = append(remoteArgs, "--json")
	}
	timeout := relayTimeoutArg(remoteArgs)
	callTimeout := remoteNodeListTimeout
	if timeout != "" {
		seconds, parseErr := strconv.Atoi(timeout)
		if parseErr == nil && seconds > 0 {
			callTimeout = time.Duration(seconds)*time.Second + remoteCallGrace
		}
	}
	response, err := client.Call(context.Background(), nodeclient.CallRequest{
		NodeID: nodeID, Scenario: scenario, Command: command, Args: remoteArgs,
		Timeout: callTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("relay %s to %q: %w", command, nodeName, err)
	}
	if len(response.Data) == 0 {
		reason := response.Reason
		if reason == "" {
			reason = "relay returned no data"
		}
		return nil, fmt.Errorf("remote scenario %s failed: outcome=%s exit_code=%d reason=%s", command, response.Outcome, response.ExitCode, reason)
	}
	return response.Data, nil
}

// resolveLocalOwnerToken preserves the root CLI's pre-nodeclient behavior:
// an enrolled operator mints a short-lived local Bridge session, while an
// unenrolled machine still receives Bridge's normal unauthenticated response.
func resolveLocalOwnerToken(context.Context) (string, error) {
	store, err := sharedsession.DefaultFileStore()
	if err != nil {
		return "", nil
	}
	resolution, err := (sharedsession.LocalResolver{Store: store}).Resolve()
	if err != nil || strings.TrimSpace(resolution.Token) == "" {
		return "", nil
	}
	return sharedsession.LocalSessionScheme + " " + resolution.Token, nil
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
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

func selectBridgeNode(nodes []*registryv1.Node, name string) (string, error) {
	matches := make([]*registryv1.Node, 0, len(nodes))
	for _, node := range nodes {
		if node != nil && node.GetName() == name {
			matches = append(matches, node)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("remote node %q: node_unpaired_or_revoked", name)
	}
	ready := make([]*registryv1.Node, 0, len(matches))
	for _, node := range matches {
		if node.GetOnline() && node.GetDispatchable() {
			ready = append(ready, node)
		}
	}
	if len(ready) == 1 {
		return ready[0].GetId(), nil
	}
	if len(matches) == 1 {
		return matches[0].GetId(), nil
	}
	// NodeRegistry returns newest-first. If no duplicate is ready, use the
	// newest record so Bridge can return its authoritative offline/revoked
	// reason instead of turning a readiness failure into an ambiguity error.
	return matches[0].GetId(), nil
}
