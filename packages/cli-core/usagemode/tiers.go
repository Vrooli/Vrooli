package usagemode

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Tiers is a resource's usage_tiers plus the policy facts needed to validate a
// target: excluded and restricted models, aliases and metered sources.
type Tiers struct {
	Runner       string
	ModelMap     map[string]string
	SessionModel string
	ExemptRoles  []string

	excluded   map[string]bool
	restricted map[string]bool
	aliases    map[string]string // alias -> canonical model
	metered    map[string]bool   // alias -> true when its provider is a metered source
	meteredAll bool
}

type policyDoc struct {
	Runner           string                     `json:"runner"`
	ExcludedModels   []string                   `json:"excluded_models"`
	RestrictedModels map[string]json.RawMessage `json:"restricted_models"`
	Billing          struct {
		Mode    string `json:"mode"`
		Sources []struct {
			ID          string `json:"id"`
			BillingMode string `json:"billing_mode"`
		} `json:"sources"`
	} `json:"billing"`
	ModelAliases map[string]struct {
		CanonicalModel string `json:"canonical_model"`
		Provider       string `json:"provider"`
	} `json:"model_aliases"`
	UsageTiers *struct {
		ModelMap     map[string]string `json:"model_map"`
		SessionModel string            `json:"session_model"`
		ExemptRoles  []string          `json:"exempt_roles"`
	} `json:"usage_tiers"`
}

// LoadTiers parses a resource model-policy.json.
func LoadTiers(policyJSON []byte) (Tiers, error) {
	var doc policyDoc
	if err := json.Unmarshal(policyJSON, &doc); err != nil {
		return Tiers{}, fmt.Errorf("parse model policy: %w", err)
	}
	if doc.UsageTiers == nil {
		return Tiers{}, errors.New("model policy declares no usage_tiers")
	}
	t := Tiers{
		Runner: doc.Runner, ModelMap: doc.UsageTiers.ModelMap, SessionModel: doc.UsageTiers.SessionModel,
		ExemptRoles: doc.UsageTiers.ExemptRoles,
		excluded:    map[string]bool{}, restricted: map[string]bool{}, aliases: map[string]string{}, metered: map[string]bool{},
		meteredAll: doc.Billing.Mode == "metered",
	}
	for _, model := range doc.ExcludedModels {
		t.excluded[model] = true
	}
	for model := range doc.RestrictedModels {
		t.restricted[model] = true
	}
	meteredSource := map[string]bool{}
	for _, source := range doc.Billing.Sources {
		if source.BillingMode == "metered" {
			meteredSource[source.ID] = true
		}
	}
	for alias, entry := range doc.ModelAliases {
		t.aliases[alias] = entry.CanonicalModel
		t.metered[alias] = meteredSource[entry.Provider]
	}
	return t, nil
}

// ValidateTarget rejects an excluded, restricted, metered or unknown target.
func (t Tiers) ValidateTarget(model string) error {
	switch {
	case t.excluded[model] || t.excluded[t.aliases[model]]:
		return fmt.Errorf("target %q is excluded", model)
	case t.restricted[model]:
		return fmt.Errorf("target %q is restricted", model)
	}
	if _, ok := t.aliases[model]; !ok {
		return fmt.Errorf("target %q is not in model_aliases", model)
	}
	if t.meteredAll || t.metered[model] {
		return fmt.Errorf("target %q is metered", model)
	}
	return nil
}

// norm resolves an alias or provider-qualified id to one comparable name, so
// "opus", "claude-opus-5" and "anthropic/claude-opus-5" are the same model.
func (t Tiers) norm(model string) string {
	model = strings.TrimSpace(model)
	if canonical := t.aliases[model]; canonical != "" {
		model = canonical
	}
	if _, id, ok := strings.Cut(model, "/"); ok && id != "" {
		model = id
	}
	return model
}

// Rank is a model's height in the resource's own tier order, derived from
// usage_tiers.model_map without any schema addition: a model the map lowers
// sits one above what it maps to, and a model that is only ever a target is
// the floor (rank 0). ok is false for a model the map says nothing about
// (for example a Haiku), whose tier cannot be proven.
func (t Tiers) Rank(model string) (rank int, ok bool) {
	rank, known, _ := tierRank(t.ModelMap, t.norm, model)
	return rank, known
}

// LowerThan reports whether candidate is provably a lower tier than replaced.
func (t Tiers) LowerThan(candidate, replaced string) bool {
	c, cok := t.Rank(candidate)
	r, rok := t.Rank(replaced)
	return cok && rok && c < r
}

func tierRank(modelMap map[string]string, norm func(string) string, model string) (rank int, known, cyclic bool) {
	next := map[string]string{}
	for from, to := range modelMap {
		next[norm(from)] = norm(to)
	}
	model = norm(model)
	seen := map[string]bool{model: true}
	for cur := model; ; {
		if _, isKey := next[cur]; isKey || containsValue(next, cur) {
			known = true
		}
		to, lowered := next[cur]
		if !lowered {
			break
		}
		if seen[to] {
			return 0, false, true // a cycle has no order
		}
		seen[to] = true
		rank++
		cur = to
	}
	return rank, known, false
}

func containsValue(m map[string]string, value string) bool {
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

// checkNeverRaises rejects a map entry whose target is not below its key when
// both tiers are known: a mode may only lower.
func (t Tiers) checkNeverRaises(from, to string) error {
	if _, _, cyclic := tierRank(t.ModelMap, t.norm, from); cyclic {
		return fmt.Errorf("usage_tiers model_map has a cycle through %q, so it has no lower tier", from)
	}
	f, fok := t.Rank(from)
	g, gok := t.Rank(to)
	if fok && gok && g >= f {
		return fmt.Errorf("target %q is not a lower tier than %q", to, from)
	}
	return nil
}

// concrete turns an alias into the id the harness takes: Claude Code gets the
// canonical id without its provider prefix; Codex ids are the aliases.
func (t Tiers) concrete(model string) string {
	if NormalizeRunner(t.Runner) == "claude" {
		if canonical := t.aliases[model]; canonical != "" {
			_, id, _ := strings.Cut(canonical, "/")
			return id
		}
	}
	return model
}

// OverrideModel points every mapped target and the session model at model.
// The map keys still decide which models are lowered, and model must be a
// provably lower tier than every key it replaces: an override may lower
// further than the resource's default, never raise or sidestep it.
func (t Tiers) OverrideModel(model string) (Tiers, error) {
	if len(t.ModelMap) == 0 {
		return t, fmt.Errorf("the resource declares no tier order, so %q cannot be shown to be lower", model)
	}
	mapped := make(map[string]string, len(t.ModelMap))
	for from := range t.ModelMap {
		if !t.LowerThan(model, from) {
			return t, fmt.Errorf("model %q is not a lower tier than %q; a usage mode only lowers", model, from)
		}
		mapped[from] = model
	}
	t.ModelMap, t.SessionModel = mapped, model
	return t, nil
}

// NewModeOptions describe a mode to build from tiers.
type NewModeOptions struct {
	Label     string
	Kind      string
	Scope     Scope
	StartedBy string
	Source    string
	Now       time.Time
	// Duration > 0 gives a "time" mode. Otherwise ExpiresAt with UntilKind
	// (window_reset or time) applies; with neither the mode is "off".
	Duration           time.Duration
	ExpiresAt          *time.Time
	UntilKind          string
	Provider           string
	WindowMinutes      int
	Effort             string
	ContextLimitTokens int
	ExtraExemptRoles   []string
}

// NewMode builds a mode from tiers. Every model_map target and the session
// model must pass ValidateTarget.
func NewMode(t Tiers, o NewModeOptions) (Mode, error) {
	if t.Runner == "" {
		return Mode{}, errors.New("tiers have no runner")
	}
	if len(t.ModelMap) == 0 && t.SessionModel == "" {
		return Mode{}, errors.New("tiers map no models")
	}
	if o.ContextLimitTokens < 0 || (NormalizeRunner(t.Runner) == "claude" && o.ContextLimitTokens > 0 && o.ContextLimitTokens < MinClaudeContextLimit) {
		return Mode{}, fmt.Errorf("context limit %d is below the minimum", o.ContextLimitTokens)
	}
	settings := RunnerSettings{ModelMap: map[string]string{}, Effort: o.Effort, ContextLimitTokens: o.ContextLimitTokens}
	keys := make([]string, 0, len(t.ModelMap))
	for from := range t.ModelMap {
		keys = append(keys, from)
	}
	sort.Strings(keys)
	for _, from := range keys {
		to := t.ModelMap[from]
		if err := t.ValidateTarget(to); err != nil {
			return Mode{}, err
		}
		if err := t.checkNeverRaises(from, to); err != nil {
			return Mode{}, err
		}
		settings.ModelMap[from] = t.concrete(to)
		// Full-id keys are resolved so Claude's alias remap sees ids too.
		if canonical := t.aliases[from]; canonical != "" && t.concrete(from) != from {
			settings.ModelMap[t.concrete(from)] = t.concrete(to)
		}
	}
	if t.SessionModel != "" {
		if err := t.ValidateTarget(t.SessionModel); err != nil {
			return Mode{}, err
		}
		settings.SessionModel = t.concrete(t.SessionModel)
	}
	now := o.Now
	if now.IsZero() {
		now = time.Now()
	}
	until := Until{Kind: UntilOff}
	var expires *time.Time
	switch {
	case o.Duration > 0:
		e := now.Add(o.Duration).UTC()
		expires, until = &e, Until{Kind: UntilTime}
	case o.ExpiresAt != nil:
		if !o.ExpiresAt.After(now) {
			return Mode{}, errors.New("expiry is not in the future")
		}
		e := o.ExpiresAt.UTC()
		kind := o.UntilKind
		if kind != UntilWindowReset {
			kind = UntilTime
		}
		expires, until = &e, Until{Kind: kind, Provider: o.Provider, WindowMinutes: o.WindowMinutes}
	}
	id := make([]byte, 3)
	if _, err := rand.Read(id); err != nil {
		return Mode{}, err
	}
	kind := o.Kind
	if kind == "" {
		kind = "save"
	}
	exempt := append(append([]string(nil), t.ExemptRoles...), o.ExtraExemptRoles...)
	return Mode{
		ID: "um-" + now.UTC().Format("20060102") + "-" + hex.EncodeToString(id), Label: o.Label, Kind: kind,
		Scope: o.Scope, StartedAt: now.UTC(), StartedBy: o.StartedBy, Source: o.Source,
		Until: until, ExpiresAt: expires, ExemptRoles: exempt,
		Runners: map[string]RunnerSettings{NormalizeRunner(t.Runner): settings},
	}, nil
}
