package main

// Port identity verification.
//
// The loop adopts a port in several ways, and two of them are guesses: the
// last-known port (which the lifecycle may since have reassigned) and a blind
// probe of a hardcoded candidate list. Neither guess checked WHOSE process
// answered, so any listener on 15000/15001/18000/19761 was adopted as "the
// autoheal API".
//
// That is not theoretical. Observed 2026-09-01: two orphaned `mock-api`
// processes leaked from `TestRunnerStartStopRestart` and
// `TestStartKeepsLifecycleOwnershipWithoutALiveSupervisor` had been squatting
// 15000 and 15001 since Aug 26, reparented to init with their binaries
// deleted. The loop adopted them, and because `apiIsAlive` then reported the
// API as answering, the anti-thrash guard concluded a restart would only
// interrupt healthy work -- and refused to restart autoheal at all, forever,
// while autoheal was in fact stopped.
//
// The anti-thrash guard is right in principle: a live process that is merely
// busy must not be restarted. It just has to be asking about the right
// process. Identity is what makes "something is answering" mean "autoheal is
// answering".

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// autohealServiceMarker is the value the autoheal API reports in its /health
// payload ("service": "Vrooli Autoheal API"). Matched case-insensitively on a
// substring so a cosmetic rename does not silently disable recovery.
const autohealServiceMarker = "autoheal"

// identityProbeBodyCap bounds how much of a health body is read. The payload
// is small; anything larger is not the health endpoint.
const identityProbeBodyCap = 64 * 1024

// isAutohealAPI reports whether the process listening on the port identifies
// itself as the autoheal API.
//
// A bare 200 is not proof: any HTTP server answers 200 to something. Adoption
// requires the service to name itself.
func isAutohealAPI(port string) bool {
	healthURL, err := localHealthEndpoint(port)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, healthURL, nil) //nolint:gosec // validated loopback-only health probe
	if err != nil {
		return false
	}
	resp, err := client.Do(req) //nolint:gosec // validated loopback-only health probe
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, identityProbeBodyCap))
	if err != nil {
		return false
	}
	return bodyIdentifiesAutoheal(body)
}

// bodyIdentifiesAutoheal reports whether a /health payload names the autoheal
// service. Split out so the matching rule is directly testable.
func bodyIdentifiesAutoheal(body []byte) bool {
	var payload struct {
		Service string `json:"service"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(payload.Service), autohealServiceMarker)
}

// autohealIsAlive reports whether autoheal itself is answering on the port.
//
// This is the identity-checked replacement for a bare liveness probe, used by
// the anti-thrash guard. Any answer FROM AUTOHEAL counts as alive, including a
// degraded one -- a 503 from a service reporting its own busy database is a
// live process a restart cannot improve. An answer from anything else does
// not count at all.
func autohealIsAlive(port string) bool {
	if port == "" {
		return false
	}
	return isAutohealAPI(port)
}
