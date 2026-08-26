package main

// remote_targets.go owns the web-console → vrooli-bridge federation seam.
// Browser clients continue to speak the stable terminal JSON protocol; this
// server-side adapter is the only place that holds the Bridge owner and
// re-authentication credentials and translates to the binary session.Frame
// protocol. A node never receives browser credentials.

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	sharedsession "github.com/vrooli/api-core/operatorsession"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	"web-console/internal/capabilities"
)

type remoteTerminalTarget struct {
	ID              string                   `json:"id"`
	Kind            string                   `json:"kind"`
	Label           string                   `json:"label"`
	OS              string                   `json:"os,omitempty"`
	Arch            string                   `json:"arch,omitempty"`
	Revision        string                   `json:"revision,omitempty"`
	Status          string                   `json:"status,omitempty"`
	Online          bool                     `json:"online"`
	LastSeenAt      time.Time                `json:"last_seen_at,omitempty"`
	Available       bool                     `json:"available"`
	DispatchReason  string                   `json:"dispatch_reason,omitempty"`
	OperatorAction  string                   `json:"operator_action,omitempty"`
	Availability    string                   `json:"availability,omitempty"`
	SurvivesRestart bool                     `json:"survives_restart"`
	ReadinessFacts  []remoteReadinessFact    `json:"-"`
	Capability      capabilities.CheckResult `json:"-"`
	BaseURL         string                   `json:"-"`
	NodeID          string                   `json:"-"`
	OwnerToken      string                   `json:"-"`
	ReauthToken     string                   `json:"-"`
}

type remoteReadinessFact struct {
	Key    string
	Label  string
	Passed bool
	Detail string
}

func configuredRemoteTarget() remoteTerminalTarget {
	ownerToken, reauthToken := resolveBridgeOwnerCredentials()
	t := remoteTerminalTarget{
		ID:          "bridge-node:" + strings.TrimSpace(getEnvOrDefault("WEB_CONSOLE_BRIDGE_NODE_ID", "")),
		Kind:        "bridge-node",
		Label:       getEnvOrDefault("WEB_CONSOLE_BRIDGE_LABEL", "Bridge node"),
		BaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_URL")), "/"),
		NodeID:      strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_NODE_ID")),
		OwnerToken:  ownerToken,
		ReauthToken: reauthToken,
	}
	result := (&capabilities.BridgeChecker{BaseURL: t.BaseURL, OwnerToken: t.OwnerToken, ReauthToken: t.ReauthToken}).CheckResult(context.Background())
	t.Capability = result
	t.DispatchReason = result.Message
	t.OperatorAction = result.ActionLabel
	if result.Status != capabilities.StatusAvailable {
		t.Availability = "unavailable"
		return t
	}
	t.Available = true
	t.Availability = "dispatchable"
	t.DispatchReason = ""
	t.OperatorAction = ""
	t.ReadinessFacts = []remoteReadinessFact{{Key: "bridge", Label: "Bridge configured", Passed: true, Detail: result.Message}}
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
	return strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN")), strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN"))
}

func hasExplicitAuthScheme(value, scheme string) bool {
	prefix := strings.TrimSpace(scheme) + " "
	return strings.HasPrefix(strings.TrimSpace(value), prefix)
}

// bridgeOwnerTransport keeps Bridge credentials on the web-console server.
// They are injected only into the server-to-server registry request and never
// become browser-visible target fields.
type bridgeOwnerTransport struct {
	base   http.RoundTripper
	owner  string
	reauth string
}

func (t bridgeOwnerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", currentOwnerAuthorization(t.owner))
	if strings.TrimSpace(t.reauth) != "" {
		clone.Header.Set("X-Bridge-Owner-Reauth", t.reauth)
	}
	return t.base.RoundTrip(clone)
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

func targetFromRegistryNode(base remoteTerminalTarget, node *registryv1.Node) remoteTerminalTarget {
	target := base
	target.ID = "bridge-node:" + node.GetId()
	target.Kind = targetKind(node)
	target.Label = node.GetName()
	if target.Label == "" {
		target.Label = node.GetId()
	}
	target.NodeID = node.GetId()
	target.OS = node.GetOs()
	target.Arch = node.GetArch()
	target.Revision = node.GetRevision()
	target.Status = node.GetStatus().String()
	target.Online = node.GetOnline()
	if node.GetLastSeenAt() != nil {
		target.LastSeenAt = node.GetLastSeenAt().AsTime()
	}
	target.ReadinessFacts = readinessFactsForNode(node)
	target.Available = node.GetDispatchable() && node.GetKind() == registryv1.NodeKind_NODE_KIND_AGENT
	if !target.Available {
		if failure := readinessFailure(node); failure != "" {
			target.DispatchReason = failure
		} else {
			target.DispatchReason = "session backend unavailable for " + target.Kind
		}
		target.OperatorAction = recoveryActionForNode(node, target.DispatchReason)
	}
	target.Availability = targetStateForNode(node, target.Available)
	return target
}

func configuredRemoteTargets() []remoteTerminalTarget {
	base := configuredRemoteTarget()
	if !base.Available {
		return []remoteTerminalTarget{base}
	}
	client := &http.Client{Timeout: 3 * time.Second, Transport: bridgeOwnerTransport{
		base: http.DefaultTransport, owner: base.OwnerToken, reauth: base.ReauthToken,
	}}
	registryClient := registryconnect.NewNodeRegistryServiceClient(client, base.BaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := registryClient.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil || response == nil || response.Msg == nil {
		base.Available = false
		base.DispatchReason = "Bridge registry unavailable"
		base.OperatorAction = "Check Bridge health and refresh the catalog"
		base.Availability = "unavailable"
		return []remoteTerminalTarget{base}
	}
	targets := make([]remoteTerminalTarget, 0, len(response.Msg.GetNodes()))
	for _, node := range response.Msg.GetNodes() {
		if node != nil {
			targets = append(targets, targetFromRegistryNode(base, node))
		}
	}
	if len(targets) == 0 {
		base.Available = false
		base.DispatchReason = "no registered Bridge nodes"
		base.OperatorAction = "Register a node with vrooli-bridge, then refresh this catalog"
		base.Availability = "unavailable"
		return []remoteTerminalTarget{base}
	}
	return targets
}

func (s *Server) remoteTargets() []remoteTerminalTarget {
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

// currentOwnerAuthorization refreshes an enrolled local session immediately
// before a Bridge request. LocalSession credentials are intentionally
// short-lived; retaining the token captured at Web Console startup would make
// an otherwise healthy operator surface fail after the 15-minute TTL. The
// fallback preserves explicit test/configuration tokens when this process has
// no local enrollment to mint from.
func currentOwnerAuthorization(value string) string {
	value = strings.TrimSpace(value)
	if !hasExplicitAuthScheme(value, sharedsession.LocalSessionScheme) {
		return ownerAuthorization(value)
	}
	store, err := sharedsession.DefaultFileStore()
	if err != nil {
		return ownerAuthorization(value)
	}
	resolution, err := (sharedsession.LocalResolver{Store: store}).Resolve()
	if err != nil || strings.TrimSpace(resolution.Token) == "" {
		return ownerAuthorization(value)
	}
	return sharedsession.LocalSessionScheme + " " + resolution.Token
}

func websocketScheme(scheme string) string {
	if scheme == "https" {
		return "wss"
	}
	return "ws"
}
