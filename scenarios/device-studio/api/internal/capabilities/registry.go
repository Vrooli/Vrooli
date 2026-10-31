// Package capabilities declares the optional scenario dependencies exposed by
// the generated scenario's machine-readable capability surface.
package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type (
	Def      = capabilityregistry.Def
	Registry = capabilityregistry.Registry
	Status   = capabilityregistry.Status
)

const (
	DependencyScenario      = capabilityregistry.DependencyScenario
	StatusAvailable         = capabilityregistry.StatusAvailable
	StatusUnavailable       = capabilityregistry.StatusUnavailable
	ActionKindScenarioStart = capabilityregistry.ActionKindScenarioStart
)

var Known = []Def{
	{
		ID: "audio-tools", Name: "Audio Tools",
		Description:    "Optional shared voice input and audio output for this scenario.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "audio-tools",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Audio Tools",
		OperatorCommand: "vrooli scenario start audio-tools --json",
		Features:        []string{"voice-input", "voice-output"},
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional audio capability depends on the selected provider and host media path"},
	},
	{
		ID: "ai-gateway", Name: "AI Gateway",
		Description:    "Optional hosted or local model inference. Device Studio projects can still be imported, edited, and rendered without it.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "ai-gateway",
		Enabled: true, Required: false,
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start AI Gateway",
		OperatorCommand: "vrooli scenario start ai-gateway --json",
		Features:        []string{"model-inference"},
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional; provider availability and credentials are configured separately"},
	},
	{
		ID: "workspace-sandbox", Name: "Workspace Sandbox",
		Description:    "Optional isolated workspace execution. Device Studio local project and render operations remain available without it.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "workspace-sandbox",
		Enabled: true, Required: false,
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Workspace Sandbox",
		OperatorCommand: "vrooli scenario start workspace-sandbox --json",
		Features:        []string{"workspace-isolation"},
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional workspace execution capability"},
	},
}

type ScenarioChecker struct{}

func (ScenarioChecker) Check(context.Context) (capabilityregistry.Status, string) {
	output, err := exec.Command("vrooli", "scenario", "status", "audio-tools", "--json").Output()
	if err != nil {
		return capabilityregistry.StatusUnavailable, "audio-tools is unavailable; start it with the operator action"
	}
	var payload struct {
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return capabilityregistry.StatusUnavailable, "audio-tools status was not valid JSON"
	}
	if strings.EqualFold(payload.Scenario.Status, "running") || strings.EqualFold(payload.Scenario.Status, "healthy") {
		return capabilityregistry.StatusAvailable, "audio-tools is healthy"
	}
	return capabilityregistry.StatusUnavailable, "audio-tools is not running; start it with the operator action"
}

// LoopbackScenarioChecker observes a scenario's cheap health endpoint through
// the canonical scenario discovery owner. It pins only loopback addresses and
// never uses environment proxies or follows redirects, so an optional service
// cannot silently resolve to or borrow a remote host.
type LoopbackScenarioChecker struct {
	Slug    string
	Resolve func(context.Context, string) (string, error)
	Client  *http.Client
}

func (c LoopbackScenarioChecker) Check(ctx context.Context) (capabilityregistry.Status, string) {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	resolve := c.Resolve
	if resolve == nil {
		resolve = discovery.ResolveScenarioURLDefault
	}
	base, err := resolve(checkCtx, c.Slug)
	if err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " is unavailable from this app; local project work remains available"
	}
	endpoint, err := loopbackHealthURL(checkCtx, base)
	if err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " is unavailable from this app; local project work remains available"
	}
	client := c.Client
	if client == nil {
		client = loopbackHealthClient()
	}
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " availability could not be checked; local project work remains available"
	}
	resp, err := client.Do(req)
	if err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " is unavailable from this app; local project work remains available"
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return capabilityregistry.StatusUnavailable, c.Slug + " health check is unavailable; local project work remains available"
	}
	return capabilityregistry.StatusAvailable, c.Slug + " responded to a local health check"
}

func loopbackHealthClient() *http.Client {
	return &http.Client{
		Timeout:       1500 * time.Millisecond,
		Transport:     &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 750 * time.Millisecond}).DialContext},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func loopbackHealthURL(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("invalid discovered scenario address")
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if host == "" || err != nil || port < 1 || port > 65535 {
		return "", errors.New("invalid discovered scenario address")
	}
	var chosen netip.Addr
	if ip, parseErr := netip.ParseAddr(host); parseErr == nil {
		if !ip.IsLoopback() {
			return "", errors.New("discovered scenario address is not loopback")
		}
		chosen = ip
	} else {
		if host != "localhost" {
			return "", errors.New("discovered scenario address is not loopback")
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return "", errors.New("loopback address could not be resolved")
		}
		for _, ip := range ips {
			if !ip.IsLoopback() {
				return "", errors.New("localhost resolved outside loopback")
			}
		}
		chosen = ips[0]
	}
	u.Host = net.JoinHostPort(chosen.String(), strconv.Itoa(port))
	u.Path = "/health"
	u.RawPath = ""
	return u.String(), nil
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]capabilityregistry.Checker{
		"audio-tools":       ScenarioChecker{},
		"ai-gateway":        LoopbackScenarioChecker{Slug: "ai-gateway"},
		"workspace-sandbox": LoopbackScenarioChecker{Slug: "workspace-sandbox"},
	}, 5*time.Second)
}
