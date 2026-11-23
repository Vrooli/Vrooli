package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// Residency decides how long a loaded model stays in memory after its last
// request. The primary models of resident roles stay loaded: they serve
// constant traffic and are shared across roles, so unloading them would only
// buy cold starts. Every other model unloads after IdleUnloadSeconds without a
// request, so a large, occasionally used model does not hold GPU memory
// between uses. Disk retention is the separate Retention block.
type Residency struct {
	ResidentRoles     []string `json:"resident_roles"`
	IdleUnloadSeconds int      `json:"idle_unload_seconds"`
	Rationale         string   `json:"rationale,omitempty"`
}

// KeepAliveForever is Ollama's keep_alive value for "never unload".
const KeepAliveForever = -1

// ResidentModels maps every model that stays loaded to the resident roles it
// is the primary of.
func (p Policy) ResidentModels() map[string]string {
	out := map[string]string{}
	roles := append([]string(nil), p.Residency.ResidentRoles...)
	sort.Strings(roles)
	for _, name := range roles {
		model := strings.TrimSpace(p.Roles[strings.TrimSpace(name)].Model)
		if model == "" {
			continue
		}
		if prior, ok := out[model]; ok {
			out[model] = prior + ", " + name
		} else {
			out[model] = name
		}
	}
	return out
}

// KeepAliveSeconds returns the Ollama keep_alive a request for model should
// carry: KeepAliveForever for a resident model, IdleUnloadSeconds otherwise.
// ok is false when the policy declares no idle unload, in which case callers
// leave keep_alive to the daemon default.
func (p Policy) KeepAliveSeconds(model string) (int, bool) {
	if p.Residency.IdleUnloadSeconds <= 0 {
		return 0, false
	}
	model = modelretention.CanonicalName(model)
	if model == "" {
		return 0, false
	}
	for resident := range p.ResidentModels() {
		if modelretention.CanonicalName(resident) == model {
			return KeepAliveForever, true
		}
	}
	return p.Residency.IdleUnloadSeconds, true
}

// LoadedModelSlots is how many models Ollama may hold at once, the value of
// OLLAMA_MAX_LOADED_MODELS: every resident model plus one slot for the
// non-resident model in use. When Ollama needs a slot it evicts the idle
// runner with the shortest keep_alive, so only the extra slot turns over and a
// resident model is never evicted to make room. With no spare slot every
// non-resident load evicts a resident, which then reloads on its next request.
// Memory still gates loads on its own; this is a count, not a memory limit.
func (p Policy) LoadedModelSlots() int { return len(p.ResidentModels()) + 1 }

// AdmissionPool names the gateway admission pool a request for model waits in
// and how many runners serve that pool. Each resident model has a runner of
// its own, so its requests queue only behind each other; an embed never waits
// behind a generate on another model. Every non-resident model shares the
// turnover slot, so two large models never load at once and evict each other.
func (p Policy) AdmissionPool(model string) (string, int) {
	model = modelretention.CanonicalName(model)
	residents := p.ResidentModels()
	for resident := range residents {
		if modelretention.CanonicalName(resident) == model {
			return "model-" + poolSafe(model), 1
		}
	}
	return "turnover", p.LoadedModelSlots() - len(residents)
}

func poolSafe(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, name)
}

func (r Residency) validate(p Policy) []error {
	var errs []error
	if r.IdleUnloadSeconds < 0 {
		errs = append(errs, fmt.Errorf("residency.idle_unload_seconds must not be negative"))
	}
	if r.IdleUnloadSeconds > 0 && r.IdleUnloadSeconds < 60 {
		errs = append(errs, fmt.Errorf("residency.idle_unload_seconds %d is below 60; a shorter dwell turns ordinary request gaps into cold starts", r.IdleUnloadSeconds))
	}
	seen := map[string]bool{}
	for _, name := range r.ResidentRoles {
		name = strings.TrimSpace(name)
		if _, ok := p.Roles[name]; !ok {
			errs = append(errs, fmt.Errorf("residency.resident_roles names unknown role %q", name))
		}
		if seen[name] {
			errs = append(errs, fmt.Errorf("residency.resident_roles lists %q twice", name))
		}
		seen[name] = true
	}
	return errs
}
