package main

// remote_targets.go owns the web-console → vrooli-bridge federation seam.
// Browser clients continue to speak the stable terminal JSON protocol; this
// server-side adapter is the only place that holds the Bridge owner and
// re-authentication credentials and translates to the binary session.Frame
// protocol. A node never receives browser credentials.

import (
	"context"
	"os"
	"strings"
	"time"

	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/nodeclient"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

// targetConnection keeps only server-side transport credentials beside the
// shared provider-neutral target model.
type targetConnection struct {
	targetmodel.Target
	BaseURL     string
	OwnerToken  string
	ReauthToken string
}

func configuredRemoteTarget() targetConnection {
	ownerToken, reauthToken := resolveBridgeOwnerCredentials()
	t := targetConnection{Target: targetmodel.Target{
		ID:         "bridge-node:" + strings.TrimSpace(getEnvOrDefault("VROOLI_BRIDGE_NODE_ID", "")),
		Ramp:       "bridge",
		Label:      getEnvOrDefault("VROOLI_BRIDGE_LABEL", "Bridge fleet"),
		Platform:   "bridge",
		DeviceKind: "bridge-node",
		NodeID:     strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_NODE_ID")),
		Transport:  targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_NODE_ID"))},
		Health:     targetmodel.TargetHealth{Status: "unconfigured"},
	}, OwnerToken: ownerToken, ReauthToken: reauthToken}
	if strings.TrimSpace(t.OwnerToken) == "" ||
		(!hasExplicitAuthScheme(t.OwnerToken, sharedsession.LocalSessionScheme) &&
			strings.TrimSpace(t.ReauthToken) == "" && strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN")) == "") {
		t.Reason = "Bridge credentials not configured"
		t.NextAction = "Enroll this machine with Bridge, then refresh the catalog"
		t.Transport.Reason = t.Reason
		return t
	}
	t.Available = true
	t.Transport.Available = true
	t.Readiness = []targetmodel.ReadinessCheck{targetmodel.ReadinessCheckFor(targetmodel.ReadinessBridgeScope, true, "Bridge credentials available")}
	return t
}

// resolveBridgeOwnerCredentials prefers the enrolled operator session on the
// Web Console host. The enrollment store contains only a signing key and
// metadata; Resolve mints the short-lived LocalSession token immediately
// before the server talks to Bridge. Static environment credentials remain a
// compatibility fallback for deployments that have not enrolled the host.
func resolveBridgeOwnerCredentials() (string, string) {
	if store, err := sharedsession.DefaultFileStore(); err == nil {
		if resolution, resolveErr := (sharedsession.LocalResolver{Store: store}).Resolve(); resolveErr == nil && strings.TrimSpace(resolution.Token) != "" {
			return sharedsession.LocalSessionScheme + " " + resolution.Token, ""
		}
	}
	if token := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN")); token != "" {
		return "Bearer " + token, strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_REAUTH_TOKEN"))
	}
	return "", ""
}

func hasExplicitAuthScheme(value, scheme string) bool {
	prefix := strings.TrimSpace(scheme) + " "
	return strings.HasPrefix(strings.TrimSpace(value), prefix)
}

func readinessFailure(node *registryv1.Node) string {
	checks := []struct {
		ok   bool
		name string
	}{
		{node.GetRegistryRecordPresent(), "registry record"},
		{node.GetHeartbeatFresh(), "heartbeat freshness"},
		{node.GetChannelHeld(), "live channel"},
		{node.GetProtocolCompatible(), "protocol compatibility"},
		{node.GetDispatchable(), "dispatchability"},
	}
	for _, check := range checks {
		if !check.ok {
			return check.name
		}
	}
	return ""
}

func targetKind(node *registryv1.Node) string {
	switch node.GetKind() {
	case registryv1.NodeKind_NODE_KIND_SSH:
		return "ssh"
	case registryv1.NodeKind_NODE_KIND_ATTACHED:
		return "attached"
	default:
		return "bridge-node"
	}
}

func targetFromRegistryNode(base targetConnection, node *registryv1.Node) targetConnection {
	target := base
	target.ID = "bridge-node:" + node.GetId()
	target.DeviceKind = targetKind(node)
	target.Label = node.GetName()
	if target.Label == "" {
		target.Label = node.GetId()
	}
	target.OS = node.GetOs()
	target.Architecture = node.GetArch()
	target.Revision = node.GetRevision()
	target.Scopes = append([]string(nil), node.GetScopes()...)
	target.Health = targetmodel.TargetHealth{Status: node.GetStatus().String()}
	target.BridgeTrust = &targetmodel.BridgeTrust{Registered: node.GetRegistryRecordPresent(), Online: node.GetOnline(), DispatchAuthorized: node.GetDispatchable()}
	target.NodeID = node.GetId()
	if node.GetLastSeenAt() != nil {
		target.LastSeenAt = node.GetLastSeenAt().AsTime()
	}
	target.Readiness = readinessFactsForNode(node)
	target.Available = node.GetDispatchable() && node.GetKind() == registryv1.NodeKind_NODE_KIND_AGENT
	target.Transport = targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: node.GetId(), Available: target.Available}
	target.Mode = targetStateForNode(node, target.Available)
	if !target.Available {
		if failure := readinessFailure(node); failure != "" {
			target.Reason = failure
		} else {
			target.Reason = "session backend unavailable for " + target.DeviceKind
		}
		target.NextAction = recoveryActionForNode(node, target.Reason)
	}
	return target
}

// bridgeNodeClient builds the single Bridge reach this server uses, alongside
// the credential-bearing base connection every derived target inherits. It is
// shared by the target catalog and the machines surface so there is exactly one
// place that decides how this process authenticates to the control plane.
func bridgeNodeClient(ctx context.Context) (*nodeclient.Client, targetConnection) {
	base := configuredRemoteTarget()
	if !base.Available {
		return nil, base
	}
	clientToken := base.OwnerToken
	var tokenProvider func(context.Context) (string, error)
	if hasExplicitAuthScheme(clientToken, sharedsession.LocalSessionScheme) {
		// Do not pin a short-lived local session into a client. nodeclient asks
		// the provider for a fresh owner credential for each request.
		clientToken = ""
		tokenProvider = resolveLocalOwnerToken
	}
	nodeClient := nodeclient.New(nodeclient.Config{
		BridgeURL: base.BaseURL, Token: clientToken, ReauthToken: base.ReauthToken,
		TokenProvider: tokenProvider,
	})
	if base.BaseURL == "" {
		if resolved, resolveErr := nodeClient.ResolveURL(ctx); resolveErr == nil {
			base.BaseURL = resolved
		}
	}
	return nodeClient, base
}

func configuredRemoteTargets() []targetConnection {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	nodeClient, base := bridgeNodeClient(ctx)
	if nodeClient == nil {
		return []targetConnection{base}
	}
	nodes, err := nodeClient.List(ctx, 3*time.Second)
	if err != nil {
		base.Available = false
		base.Reason = "Bridge registry unavailable"
		base.NextAction = "Check Bridge health and refresh the catalog"
		return []targetConnection{base}
	}
	targets := make([]targetConnection, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			targets = append(targets, targetFromRegistryNode(base, node))
		}
	}
	if len(targets) == 0 {
		base.Available = false
		base.Reason = "no registered Bridge nodes"
		base.NextAction = "Register a node with vrooli-bridge, then refresh this catalog"
		return []targetConnection{base}
	}
	return targets
}

func (s *Server) remoteTargets() []targetConnection {
	if s.remoteTargetCatalog != nil {
		return s.remoteTargetCatalog()
	}
	return configuredRemoteTargets()
}

func ownerAuthorization(value string) string {
	value = strings.TrimSpace(value)
	if hasExplicitAuthScheme(value, "LocalSession") || hasExplicitAuthScheme(value, "Bearer") {
		return value
	}
	return "Bearer " + value
}

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

func websocketScheme(scheme string) string {
	if scheme == "https" {
		return "wss"
	}
	return "ws"
}
