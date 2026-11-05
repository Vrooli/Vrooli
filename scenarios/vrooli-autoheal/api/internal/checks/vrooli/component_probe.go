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
	"net"
	"net/http"
	"strconv"
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
	// ComponentProbeOK means the component accepted a connection and, for
	// HTTP probes, answered without a transport error.
	ComponentProbeOK ComponentProbeState = "ok"
	// ComponentProbeUnreachable means the port is allocated but nothing
	// usable is answering on it.
	ComponentProbeUnreachable ComponentProbeState = "unreachable"
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

// Unreachable reports whether the probe found an allocated but dead port.
func (r ComponentProbeResult) Unreachable() bool { return r.State == ComponentProbeUnreachable }

// componentProbe is the seam used to probe a component port. Tests substitute
// a fake so check behaviour can be exercised without binding real sockets.
type componentProbe func(ctx context.Context, port int32) ComponentProbeState

// probeHTTPPort reports whether a loopback port is serving.
//
// Any HTTP status counts as reachable, including 404 and 5xx: the question
// this probe answers is "is the component still there", not "is every route
// correct". A 404 from a live dev server is a healthy UI with no index route,
// and restarting the scenario over it would be a false positive.
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

	probeCtx, cancel := context.WithTimeout(ctx, componentProbeTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, "http://"+address+"/", nil)
	if err != nil {
		return ComponentProbeUnreachable
	}
	client := &http.Client{Timeout: componentProbeTimeout}
	response, err := client.Do(request)
	if err != nil {
		// The socket accepted a connection but no HTTP response came back.
		// Something holds the port without serving, which is precisely the
		// wedged-UI case worth healing.
		return ComponentProbeUnreachable
	}
	defer func() { _ = response.Body.Close() }()
	return ComponentProbeOK
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
	case ComponentProbeOK:
		result.Detail = fmt.Sprintf("%s %d is answering", role, port)
	default:
		result.Detail = fmt.Sprintf("%s probe skipped", role)
	}
	return result
}
