// Package caller carries the identity of the process that sent a request
// through the request's context, so every record of inference the gateway
// writes can say who asked for it.
//
// The identity is observed, not declared: the gateway resolves its loopback
// peer to the process holding the client socket and names that process by its
// Vrooli scope (see packages/capacity/callerid). Resolution starts when a
// handler calls Observe and runs beside the request, so it costs the request
// nothing; the caller is alive throughout because it is waiting for the
// response.
package caller

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/callerid"
)

// Unknown is the label of a caller that could not be identified.
const Unknown = callerid.Unknown

// resolver is shared by every request so a keep-alive connection's caller is
// found once per socket, not once per request.
var resolver = callerid.NewResolver()

type observationKey struct{}

// observation is one request's caller, resolved in the background.
type observation struct {
	done  chan struct{}
	label string
}

// Observe starts identifying the process at remoteAddr, the request's peer,
// and returns a context carrying the pending answer. The server's own address
// comes from the HTTP server's context value.
func Observe(ctx context.Context, remoteAddr string) context.Context {
	local, _ := ctx.Value(http.LocalAddrContextKey).(net.Addr)
	remote, err := net.ResolveTCPAddr("tcp", strings.TrimSpace(remoteAddr))
	if local == nil || err != nil {
		return WithLabel(ctx, Unknown)
	}
	obs := &observation{done: make(chan struct{})}
	go func() {
		defer close(obs.done)
		obs.label = resolver.Identify(local, remote)
	}()
	return context.WithValue(ctx, observationKey{}, obs)
}

// WithLabel returns a context whose caller is already known.
func WithLabel(ctx context.Context, label string) context.Context {
	obs := &observation{done: make(chan struct{}), label: strings.TrimSpace(label)}
	if obs.label == "" {
		obs.label = Unknown
	}
	close(obs.done)
	return context.WithValue(ctx, observationKey{}, obs)
}

// Ready returns the caller when it is already known. ok is false while the
// identification is still running and when the context never observed one.
func Ready(ctx context.Context) (string, bool) {
	obs, _ := ctx.Value(observationKey{}).(*observation)
	if obs == nil {
		return "", false
	}
	select {
	case <-obs.done:
		return obs.label, true
	default:
		return "", false
	}
}

// Pending reports whether the context's caller identification is still
// running.
func Pending(ctx context.Context) bool {
	obs, _ := ctx.Value(observationKey{}).(*observation)
	if obs == nil {
		return false
	}
	select {
	case <-obs.done:
		return false
	default:
		return true
	}
}

// Await returns the caller, waiting at most wait for the identification to
// finish. It returns Unknown when the context never observed a caller or the
// identification did not finish in time.
func Await(ctx context.Context, wait time.Duration) string {
	obs, _ := ctx.Value(observationKey{}).(*observation)
	if obs == nil {
		return Unknown
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-obs.done:
		return obs.label
	case <-timer.C:
		return Unknown
	}
}

// Scenarios names the scenario a caller's scope belongs to. A scenario
// service's scope unit is "<scenario>-<lifecycle step>", and neither part is
// delimited, so the scenario is the longest known scenario name the unit
// starts with. Known names come from the repository's scenario directories.
type Scenarios struct {
	names []string // longest first
}

// NewScenarios returns a namer over the given scenario names.
func NewScenarios(names []string) Scenarios {
	sorted := make([]string, 0, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			sorted = append(sorted, name)
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })
	return Scenarios{names: sorted}
}

// LoadScenarios lists the scenario directories under repoRoot.
func LoadScenarios(repoRoot string) Scenarios {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return NewScenarios(nil)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return NewScenarios(names)
}

// Of returns the scenario label belongs to. A scenario scope that matches no
// known scenario keeps its whole unit; resource, agent and host callers are
// named by kind and unit, for example "agent:codex".
func (s Scenarios) Of(label string) string {
	kind, unit, _, ok := callerid.Parse(label)
	if !ok {
		return Unknown
	}
	if kind != callerid.KindScenario {
		return kind + ":" + unit
	}
	for _, name := range s.names {
		if unit == name || strings.HasPrefix(unit, name+"-") {
			return name
		}
	}
	return unit
}
