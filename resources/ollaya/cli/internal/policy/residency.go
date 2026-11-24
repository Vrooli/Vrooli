package policy

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// Residency decides how long a loaded model stays in memory after its last
// decision, the same contract the Ollama resource's residency block declares.
// The primary models of resident roles stay loaded because constant traffic
// would only turn an unload into cold starts. Every other model unloads after
// IdleUnloadSeconds without a request, so a model used in occasional bursts
// does not hold accelerator memory between them. A model's own
// placement.keep_alive is more specific and wins.
//
// The activity edge applies it: every decision that reaches the daemon carries
// the keep_alive this block resolves to, so the daemon-wide OLLAYA_KEEP_ALIVE
// matters only when the policy cannot be read.
type Residency struct {
	ResidentRoles     []string `json:"resident_roles"`
	IdleUnloadSeconds int      `json:"idle_unload_seconds"`
	Rationale         string   `json:"rationale,omitempty"`
}

// KeepAliveForever is the daemon's keep_alive for "never unload".
const KeepAliveForever = -1

// minIdleUnloadSeconds keeps an idle unload from turning ordinary gaps inside
// one burst of decisions into cold loads.
const minIdleUnloadSeconds = 60

func (r Residency) validate(p Policy) error {
	var errs []error
	if r.IdleUnloadSeconds < 0 {
		errs = append(errs, errors.New("residency.idle_unload_seconds must not be negative"))
	}
	if r.IdleUnloadSeconds > 0 && r.IdleUnloadSeconds < minIdleUnloadSeconds {
		errs = append(errs, fmt.Errorf("residency.idle_unload_seconds %d is below %d; a shorter dwell turns gaps inside one burst into cold loads", r.IdleUnloadSeconds, minIdleUnloadSeconds))
	}
	seen := map[string]bool{}
	for _, name := range r.ResidentRoles {
		name = strings.TrimSpace(name)
		role, ok := p.Roles[name]
		switch {
		case !ok:
			errs = append(errs, fmt.Errorf("residency.resident_roles names unknown role %q", name))
		case p.Upstreams[role.Upstream].Locality != LocalityLocal:
			errs = append(errs, fmt.Errorf("residency.resident_roles names hosted role %q; only local models hold memory", name))
		}
		if seen[name] {
			errs = append(errs, fmt.Errorf("residency.resident_roles lists %q twice", name))
		}
		seen[name] = true
	}
	return errors.Join(errs...)
}

// residentModels is every model that stays loaded: each resident role's model
// and, for a router, the targets the daemon loads to serve it.
func (p Policy) residentModels() map[string]bool {
	out := map[string]bool{}
	var add func(string)
	add = func(name string) {
		canonical := modelretention.CanonicalName(name)
		if canonical == "" || out[canonical] {
			return
		}
		out[canonical] = true
		for _, route := range p.Models[name].Routes {
			add(route)
		}
	}
	for _, role := range p.Residency.ResidentRoles {
		add(p.Roles[strings.TrimSpace(role)].Model)
	}
	return out
}

// KeepAliveSeconds returns the keep_alive (in seconds) a decision for model
// carries: the model's own placement.keep_alive when it declares one,
// KeepAliveForever for a resident model, IdleUnloadSeconds otherwise. ok is
// false when nothing applies (an unknown model under a policy with no idle
// unload), in which case the daemon default stays in force. model may carry a
// tag and an @sha256 pin; both are matched by canonical name.
func (p Policy) KeepAliveSeconds(model string) (int, bool) {
	canonical := modelretention.CanonicalName(model)
	if canonical == "" {
		return 0, false
	}
	for name, entry := range p.Models {
		if modelretention.CanonicalName(name) != canonical || entry.Placement == nil || entry.Placement.KeepAlive == "" {
			continue
		}
		if d, err := time.ParseDuration(entry.Placement.KeepAlive); err == nil && d > 0 {
			return int(math.Ceil(d.Seconds())), true
		}
	}
	if p.residentModels()[canonical] {
		return KeepAliveForever, true
	}
	if p.Residency.IdleUnloadSeconds <= 0 {
		return 0, false
	}
	return p.Residency.IdleUnloadSeconds, true
}
