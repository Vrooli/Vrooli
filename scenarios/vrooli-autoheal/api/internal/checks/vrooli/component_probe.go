package vrooli

// Component-level reachability.
//
// `vrooli scenario status` reports one health_status for the whole scenario,
// and that value reflects the API probe. A scenario whose UI has stopped
// answering therefore reports "healthy" and autoheal sees StatusOK -- the
// operator sees a green dashboard and a dead page.
//
// This file adds a direct reachability probe for the non-API components whose
// ports the lifecycle allocated, so "running but the UI is down" becomes a
// state autoheal can name and act on rather than a silent false green.

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Port roles reported by `vrooli scenario status --json`.
const (
	roleAPIPort = "API_PORT"
	roleUIPort  = "UI_PORT"
)

// componentProbeTimeout bounds a single component probe. This runs inside a
// scheduled check against a loopback port, so it should be short: a UI that
// needs more than this to answer a TCP connect is not serving users either.
const componentProbeTimeout = 3 * time.Second

// ComponentProbeState is the outcome of probing one component's port.
type ComponentProbeState string

const (
	// ComponentProbeOK means the component served a 200 page with a root
	// element and a loadable script bundle.
	ComponentProbeOK ComponentProbeState = "ok"
	// ComponentProbeUnreachable means the port is allocated but nothing
	// usable is answering on it, or it answers with an error status.
	ComponentProbeUnreachable ComponentProbeState = "unreachable"
	// ComponentProbeDegraded means the page answers 200 but is blank or its
	// first script bundle does not load.
	ComponentProbeDegraded ComponentProbeState = "degraded"
	// ComponentProbeSkipped means there was no port to probe, so the
	// component's reachability is unknown and must not be treated as failure.
	ComponentProbeSkipped ComponentProbeState = "skipped"
)

// ComponentProbeResult describes one component probe.
type ComponentProbeResult struct {
	Role   string
	Port   int32
	State  ComponentProbeState
	Detail string
}

// Failed reports whether the probe found an allocated port that is not serving
// a usable UI (unreachable or degraded).
func (r ComponentProbeResult) Failed() bool {
	return r.State == ComponentProbeUnreachable || r.State == ComponentProbeDegraded
}

// componentProbe is the seam used to probe a component port. Tests substitute
// a fake so check behaviour can be exercised without binding real sockets.
type componentProbe func(ctx context.Context, port int32) ComponentProbeState

// Page and bundle bodies are read only far enough to find the root element and
// the first script tag, and to see that the bundle is not empty.
const componentProbeBodyLimit = 512 << 10

var (
	uiRootMarker = regexp.MustCompile(`(?i)<div[^>]+id=["']?[\w-]+|id=["'](root|app|__next)["']`)
	uiScriptSrc  = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	uiHTMLTag    = regexp.MustCompile(`(?s)<script.*?</script>|<style.*?</style>|<[^>]*>`)
)

// probeHTTPPort reports whether a loopback UI is serving a page a user can
// use, the same question the steward's own probe asks: `GET /` must be 200, the
// page must not be blank (a root element, a script, or visible text), and its
// first local script bundle must load 200 and non-empty. A listener that
// answers 404 or 5xx is not a serving UI; a page whose bundle is gone is
// degraded, which is the blank-page outage a bare TCP or any-status probe
// reports green.
func probeHTTPPort(ctx context.Context, port int32) ComponentProbeState {
	if port <= 0 {
		return ComponentProbeSkipped
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))

	// TCP first: it distinguishes "nothing is listening" from "listening but
	// slow", and it is the cheap, unambiguous half of the signal.
	dialer := net.Dialer{Timeout: componentProbeTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return ComponentProbeUnreachable
	}
	_ = conn.Close()

	client := &http.Client{Timeout: componentProbeTimeout}
	status, page, err := fetchLoopback(ctx, client, "http://"+address+"/")
	if err != nil || status != http.StatusOK {
		// Either nothing answered HTTP on a held port, or it answered with an
		// error status: both are the wedged-UI case worth healing.
		return ComponentProbeUnreachable
	}
	text := string(page)
	var scripts []string
	for _, match := range uiScriptSrc.FindAllStringSubmatch(text, -1) {
		scripts = append(scripts, match[1])
	}
	if len(scripts) == 0 && !uiRootMarker.MatchString(text) && len(strings.TrimSpace(uiHTMLTag.ReplaceAllString(text, ""))) < 20 {
		return ComponentProbeDegraded
	}
	if bundle := firstLocalScript(scripts, address); bundle != "" {
		status, body, err := fetchLoopback(ctx, client, bundle)
		if err != nil || status != http.StatusOK || len(body) == 0 {
			return ComponentProbeDegraded
		}
	}
	return ComponentProbeOK
}

// firstLocalScript resolves the first script that this server hosts. A script
// on another origin says nothing about this UI, so it is not fetched.
func firstLocalScript(sources []string, address string) string {
	base, err := url.Parse("http://" + address + "/")
	if err != nil {
		return ""
	}
	for _, source := range sources {
		resolved, err := base.Parse(source)
		if err != nil || resolved.Host != address {
			continue
		}
		return resolved.String()
	}
	return ""
}

func fetchLoopback(ctx context.Context, client *http.Client, target string) (int, []byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, componentProbeTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, componentProbeBodyLimit))
	return response.StatusCode, body, err
}

// probeComponent runs the configured probe for one role/port pair.
func probeComponent(ctx context.Context, probe componentProbe, role string, port int32, present bool) ComponentProbeResult {
	if !present {
		return ComponentProbeResult{
			Role:   role,
			State:  ComponentProbeSkipped,
			Detail: "no " + role + " allocated for this scenario",
		}
	}
	if probe == nil {
		probe = probeHTTPPort
	}
	state := probe(ctx, port)
	result := ComponentProbeResult{Role: role, Port: port, State: state}
	switch state {
	case ComponentProbeUnreachable:
		result.Detail = fmt.Sprintf("%s %d is allocated but not answering", role, port)
	case ComponentProbeDegraded:
		result.Detail = fmt.Sprintf("%s %d answers but serves a blank page or a missing script bundle", role, port)
	case ComponentProbeOK:
		result.Detail = fmt.Sprintf("%s %d is answering", role, port)
	default:
		result.Detail = fmt.Sprintf("%s probe skipped", role)
	}
	return result
}
