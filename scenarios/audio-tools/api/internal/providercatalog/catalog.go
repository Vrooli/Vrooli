// Package providercatalog owns audio-tools' declarative provider catalogue: one
// record per provider/engine-ish provider in catalog.seed.json, plus the loader,
// validator, accessors, and the renderers that produce the generated Go and
// TypeScript surfaces.
//
// The catalogue is the single source for the provider IDs that health,
// lifecycle, settings, and usage use, and for the STT engine IDs that stream
// selection uses. Provider IDs are stable; engine IDs are the values the STT
// runtime currently reports for local streaming/batch engines. Runtime aliases
// are the remaining chain-reported identities that canonicalize to a provider,
// and probe facts are the remote readiness shape for BYOK/gateway providers, so
// adding a provider of an existing kind stays a data edit. See
// docs/concepts/ARCHITECTURE.md and docs/concepts/DOMAINS.md.
package providercatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Kind is the execution shape of a provider.
type Kind string

const (
	KindScenario      Kind = "scenario"
	KindLocalResource Kind = "local_resource"
	KindLocalInProc   Kind = "local_inprocess"
	KindBYOKAPI       Kind = "byok_api"
	KindGateway       Kind = "gateway"
	KindBrowser       Kind = "browser"
)

func (k Kind) valid() bool {
	switch k {
	case KindScenario, KindLocalResource, KindLocalInProc, KindBYOKAPI, KindGateway, KindBrowser:
		return true
	default:
		return false
	}
}

// Tier is the routing tier a provider belongs to. It mirrors the proto
// ProviderTier vocabulary without importing the proto package.
type Tier string

const (
	TierLocal   Tier = "local"
	TierBYOK    Tier = "byok"
	TierVrooli  Tier = "vrooli"
	TierBrowser Tier = "browser"
	TierNone    Tier = "none"
)

func (t Tier) valid() bool {
	switch t {
	case TierLocal, TierBYOK, TierVrooli, TierBrowser, TierNone:
		return true
	default:
		return false
	}
}

// Status is the catalogue lifecycle state. Only active providers are probed and
// rendered into capabilities.Known.
type Status string

const (
	StatusActive       Status = "active"
	StatusPlanned      Status = "planned"
	StatusExperimental Status = "experimental"
	StatusWatch        Status = "watch"
)

func (s Status) valid() bool {
	switch s {
	case StatusActive, StatusPlanned, StatusExperimental, StatusWatch:
		return true
	default:
		return false
	}
}

// Capability is the product capability a provider serves. speaker is retained
// as a distinct catalogue capability even though the diagnostics proto folds
// speaker evidence into CAPABILITY_STT.
type Capability string

const (
	CapabilitySTT       Capability = "stt"
	CapabilityTTS       Capability = "tts"
	CapabilitySummarize Capability = "summarize"
	CapabilityTranscode Capability = "transcode"
	CapabilitySpeaker   Capability = "speaker"
)

func (c Capability) valid() bool {
	switch c {
	case CapabilitySTT, CapabilityTTS, CapabilitySummarize, CapabilityTranscode, CapabilitySpeaker:
		return true
	default:
		return false
	}
}

// Action is a provider lifecycle verb advertised by ListLocalProviders.
type Action string

const (
	ActionStart     Action = "start"
	ActionStop      Action = "stop"
	ActionRestart   Action = "restart"
	ActionViewLogs  Action = "view_logs"
	ActionPullModel Action = "pull_model"
)

func (a Action) valid() bool {
	switch a {
	case ActionStart, ActionStop, ActionRestart, ActionViewLogs, ActionPullModel:
		return true
	default:
		return false
	}
}

// CheckerKind names the probe/construction shape used by the capability
// registry. It is deliberately coarse: the bootstrap composition root still
// owns the concrete checker instance.
type CheckerKind string

const (
	CheckerHTTPJSON CheckerKind = "http_json"
	CheckerCommand  CheckerKind = "command"
	CheckerResource CheckerKind = "resource"
	CheckerNone     CheckerKind = "none"
)

func (c CheckerKind) valid() bool {
	switch c {
	case CheckerHTTPJSON, CheckerCommand, CheckerResource, CheckerNone:
		return true
	default:
		return false
	}
}

// DependencyKind mirrors capabilityregistry.DependencyKind without importing it.
type DependencyKind string

const (
	DependencyScenario DependencyKind = "scenario"
	DependencyResource DependencyKind = "resource"
)

func (d DependencyKind) valid() bool {
	return d == DependencyScenario || d == DependencyResource
}

// DefActionKind mirrors capabilityregistry.ActionKind: the health-remediation
// guidance rendered into capabilities.Def. It is separate from Actions, which
// are the provider_lifecycle verbs.
type DefActionKind string

const (
	DefActionNone            DefActionKind = "none"
	DefActionOperatorCommand DefActionKind = "operator_command"
	DefActionScenarioStart   DefActionKind = "scenario_start"
	DefActionScenarioRestart DefActionKind = "scenario_restart"
	DefActionOwnerGuidance   DefActionKind = "owner_guidance"
)

func (a DefActionKind) valid() bool {
	switch a {
	case DefActionNone, DefActionOperatorCommand, DefActionScenarioStart, DefActionScenarioRestart, DefActionOwnerGuidance:
		return true
	default:
		return false
	}
}

// Criticality is the health-gating weight of a provider.
type Criticality string

const (
	CriticalityRequired Criticality = "required"
	CriticalityOptional Criticality = "optional"
)

func (c Criticality) valid() bool {
	return c == CriticalityRequired || c == CriticalityOptional
}

// PlatformSupport is a static host-support declaration. Host resource manifests
// remain the final authority; KnownForPlatform applies their verdicts.
type PlatformSupport string

const (
	PlatformSupported   PlatformSupport = "supported"
	PlatformDegraded    PlatformSupport = "degraded"
	PlatformUnsupported PlatformSupport = "unsupported"
)

func (p PlatformSupport) valid() bool {
	switch p {
	case PlatformSupported, PlatformDegraded, PlatformUnsupported:
		return true
	default:
		return false
	}
}

// Platform is the static platform verdict for a provider.
type Platform struct {
	Support PlatformSupport `json:"support"`
	Reason  string          `json:"reason,omitempty"`
}

// Presentation holds i18n keys so the UI does not grow a second provider table.
type Presentation struct {
	LabelKey       string `json:"label_key"`
	DescriptionKey string `json:"description_key"`
}

// Probe is the remote readiness probe fact for a BYOK/gateway provider: where to
// ask, which header carries the credential, and the value prefix the provider
// expects (for example "Bearer " or "Token "). It is provider fact data, never
// a credential. The secret is supplied at probe time and only ever combined
// with ValuePrefix in memory.
type Probe struct {
	URL         string `json:"url"`
	Header      string `json:"header"`
	ValuePrefix string `json:"value_prefix,omitempty"`
}

// Provider is one catalogue record.
type Provider struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Kind           Kind           `json:"kind"`
	Tier           Tier           `json:"tier"`
	Status         Status         `json:"status"`
	Capabilities   []Capability   `json:"capabilities"`
	Features       []string       `json:"features"`
	EngineIDs      []string       `json:"engine_ids,omitempty"`
	RuntimeAliases []string       `json:"runtime_aliases,omitempty"`
	ResourceSlug   string         `json:"resource_slug,omitempty"`
	DependencyKind DependencyKind `json:"dependency_kind,omitempty"`
	DependencySlug string         `json:"dependency_slug,omitempty"`
	Platform       Platform       `json:"platform"`
	Actions        []Action       `json:"actions,omitempty"`
	CheckerKind    CheckerKind    `json:"checker_kind"`
	// Probe is the remote readiness probe fact for a BYOK/gateway provider.
	// ProbeAbsentReason documents why an active remote provider cannot be
	// credential-probed instead of silently omitting the fact.
	Probe             *Probe        `json:"probe,omitempty"`
	ProbeAbsentReason string        `json:"probe_absent_reason,omitempty"`
	Models            []string      `json:"models,omitempty"`
	DocsURL           string        `json:"docs_url,omitempty"`
	OperatorCommand   string        `json:"operator_command,omitempty"`
	Criticality       Criticality   `json:"criticality"`
	DefActionKind     DefActionKind `json:"def_action_kind,omitempty"`
	Presentation      Presentation  `json:"presentation"`
	AdapterID         string        `json:"adapter_id,omitempty"`
	License           string        `json:"license,omitempty"`
}

// seedDoc is the on-disk shape of catalog.seed.json.
type seedDoc struct {
	SchemaVersion string     `json:"schema_version"`
	Providers     []Provider `json:"providers"`
}

//go:embed catalog.seed.json
var seedBytes []byte

// Registry is the validated catalogue plus its lookup indexes.
type Registry struct {
	schemaVersion    string
	providers        []Provider
	byID             map[string]Provider
	providerByEngine map[string]string
	providerByAlias  map[string]string
}

// Load parses and validates the embedded seed. It never reads the filesystem at
// runtime; the seed ships in the binary.
func Load() (*Registry, error) {
	return LoadBytes(seedBytes)
}

// LoadBytes parses and validates a seed document. Tests use it to exercise the
// validation rules with fixture documents.
func LoadBytes(raw []byte) (*Registry, error) {
	var doc seedDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("providercatalog: parse seed: %w", err)
	}
	if strings.TrimSpace(doc.SchemaVersion) == "" {
		return nil, fmt.Errorf("providercatalog: seed has no schema_version")
	}
	r := &Registry{
		schemaVersion:    doc.SchemaVersion,
		providers:        make([]Provider, len(doc.Providers)),
		byID:             make(map[string]Provider, len(doc.Providers)),
		providerByEngine: make(map[string]string),
		providerByAlias:  make(map[string]string),
	}
	for i := range doc.Providers {
		doc.Providers[i].applyDefaults()
		r.providers[i] = cloneProvider(doc.Providers[i])
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	for _, p := range r.providers {
		r.byID[p.ID] = p
		for _, engineID := range p.EngineIDs {
			r.providerByEngine[engineID] = p.ID
		}
		for _, alias := range p.RuntimeAliases {
			r.providerByAlias[alias] = p.ID
		}
	}
	return r, nil
}

// applyDefaults fills facts implied by the provider's kind. Nothing
// capability-bearing is defaulted: a missing feature, capability, tier, or
// status is a declaration bug.
func (p *Provider) applyDefaults() {
	if p.DependencyKind == "" {
		if p.Kind == KindLocalResource {
			p.DependencyKind = DependencyResource
		} else {
			p.DependencyKind = DependencyScenario
		}
	}
	if p.DependencySlug == "" {
		if p.DependencyKind == DependencyResource && p.ResourceSlug != "" {
			p.DependencySlug = p.ResourceSlug
		} else if p.DependencyKind == DependencyScenario {
			p.DependencySlug = "audio-tools"
		}
	}
	if p.Platform.Support == "" {
		p.Platform.Support = PlatformSupported
	}
	if p.DefActionKind == "" {
		p.DefActionKind = DefActionNone
	}
	if p.Criticality == "" {
		p.Criticality = CriticalityOptional
	}
	if p.CheckerKind == "" {
		p.CheckerKind = CheckerNone
	}
}

// validate enforces the cross-field invariants of the catalogue. It returns the
// first violation; the seed is a build-time source, so a failure is a defect.
func (r *Registry) validate() error {
	if len(r.providers) == 0 {
		return fmt.Errorf("providercatalog: seed declares no providers")
	}
	seenIDs := make(map[string]struct{}, len(r.providers))
	seenEngineIDs := make(map[string]string)
	seenAdapters := make(map[string]string)
	for i := range r.providers {
		p := &r.providers[i]
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("providercatalog: provider with empty id at index %d", i)
		}
		if _, dup := seenIDs[p.ID]; dup {
			return fmt.Errorf("providercatalog: duplicate provider id %q", p.ID)
		}
		seenIDs[p.ID] = struct{}{}
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("providercatalog: provider %q has no name", p.ID)
		}
		if strings.TrimSpace(p.Description) == "" {
			return fmt.Errorf("providercatalog: provider %q has no description", p.ID)
		}
		if !p.Kind.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid kind %q", p.ID, p.Kind)
		}
		if !p.Tier.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid tier %q", p.ID, p.Tier)
		}
		if want := tierForKind(p.Kind); p.Tier != want {
			return fmt.Errorf("providercatalog: provider %q kind %q requires tier %q, got %q", p.ID, p.Kind, want, p.Tier)
		}
		if !p.Status.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid status %q", p.ID, p.Status)
		}
		if !p.DependencyKind.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid dependency_kind %q", p.ID, p.DependencyKind)
		}
		if strings.TrimSpace(p.DependencySlug) == "" {
			return fmt.Errorf("providercatalog: provider %q has no dependency_slug", p.ID)
		}
		if !p.Platform.Support.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid platform support %q", p.ID, p.Platform.Support)
		}
		if p.Platform.Support != PlatformSupported && strings.TrimSpace(p.Platform.Reason) == "" {
			return fmt.Errorf("providercatalog: provider %q platform %q requires a reason", p.ID, p.Platform.Support)
		}
		if !p.CheckerKind.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid checker_kind %q", p.ID, p.CheckerKind)
		}
		if !p.Criticality.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid criticality %q", p.ID, p.Criticality)
		}
		if !p.DefActionKind.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid def_action_kind %q", p.ID, p.DefActionKind)
		}
		if strings.TrimSpace(p.Presentation.LabelKey) == "" || strings.TrimSpace(p.Presentation.DescriptionKey) == "" {
			return fmt.Errorf("providercatalog: provider %q has incomplete presentation keys", p.ID)
		}
		if p.Status == StatusActive {
			if len(p.Features) == 0 {
				return fmt.Errorf("providercatalog: active provider %q declares no features", p.ID)
			}
			if len(p.Capabilities) == 0 {
				return fmt.Errorf("providercatalog: active provider %q declares no capabilities", p.ID)
			}
			if p.Kind == KindLocalResource && strings.TrimSpace(p.ResourceSlug) == "" {
				return fmt.Errorf("providercatalog: active local provider %q has no resource_slug", p.ID)
			}
		} else {
			if p.CheckerKind != CheckerNone {
				return fmt.Errorf("providercatalog: %s provider %q must not be probed (checker_kind=%q)", p.Status, p.ID, p.CheckerKind)
			}
			if len(p.Actions) > 0 {
				return fmt.Errorf("providercatalog: %s provider %q must not advertise lifecycle actions", p.Status, p.ID)
			}
		}
		for _, capability := range p.Capabilities {
			if !capability.valid() {
				return fmt.Errorf("providercatalog: provider %q has invalid capability %q", p.ID, capability)
			}
		}
		for _, feature := range p.Features {
			if _, ok := featureCapabilities[feature]; !ok {
				return fmt.Errorf("providercatalog: provider %q declares unknown feature %q", p.ID, feature)
			}
		}
		for _, action := range p.Actions {
			if !action.valid() {
				return fmt.Errorf("providercatalog: provider %q has invalid action %q", p.ID, action)
			}
			if p.Kind != KindLocalResource {
				return fmt.Errorf("providercatalog: provider %q kind %q must not advertise lifecycle actions", p.ID, p.Kind)
			}
			if action == ActionPullModel && p.ID != "ollama" {
				return fmt.Errorf("providercatalog: provider %q must not advertise pull_model; only ollama does", p.ID)
			}
		}
		for _, engineID := range p.EngineIDs {
			if strings.TrimSpace(engineID) == "" {
				return fmt.Errorf("providercatalog: provider %q has an empty engine_id", p.ID)
			}
			if !hasCapability(p, CapabilitySTT) || p.Kind != KindLocalResource {
				return fmt.Errorf("providercatalog: provider %q declares engine_id %q but is not a local_resource STT provider", p.ID, engineID)
			}
			if owner, dup := seenEngineIDs[engineID]; dup {
				return fmt.Errorf("providercatalog: engine_id %q is declared by both %q and %q", engineID, owner, p.ID)
			}
			seenEngineIDs[engineID] = p.ID
		}
		if p.Status == StatusActive && p.Kind == KindLocalResource && hasCapability(p, CapabilitySTT) && len(p.EngineIDs) == 0 {
			return fmt.Errorf("providercatalog: active local STT provider %q declares no engine_ids", p.ID)
		}
		if p.AdapterID != "" {
			if p.Status != StatusActive || !hasCapability(p, CapabilityTTS) {
				return fmt.Errorf("providercatalog: provider %q declares adapter_id %q but is not an active TTS provider", p.ID, p.AdapterID)
			}
			if owner, dup := seenAdapters[p.AdapterID]; dup {
				return fmt.Errorf("providercatalog: adapter_id %q is declared by both %q and %q", p.AdapterID, owner, p.ID)
			}
			seenAdapters[p.AdapterID] = p.ID
		}
		if err := validateProbe(p); err != nil {
			return err
		}
	}
	// Runtime aliases are a cross-provider namespace, so validate them after
	// every provider id and engine id is known.
	seenAliases := make(map[string]string)
	for i := range r.providers {
		p := &r.providers[i]
		for _, alias := range p.RuntimeAliases {
			if alias == "" || alias != strings.TrimSpace(alias) {
				return fmt.Errorf("providercatalog: provider %q has a blank runtime_alias", p.ID)
			}
			if alias == p.ID {
				return fmt.Errorf("providercatalog: provider %q declares its own id as a runtime_alias", p.ID)
			}
			if owner, dup := seenIDs[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by %q but collides with provider id %q", alias, p.ID, owner)
			}
			if owner, dup := seenEngineIDs[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by %q but collides with an engine_id of %q", alias, p.ID, owner)
			}
			if owner, dup := seenAliases[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by both %q and %q", alias, owner, p.ID)
			}
			seenAliases[alias] = p.ID
		}
	}
	return nil
}

// validateProbe keeps the readiness-probe fact bound to remote providers and
// requires an active remote provider to state either the probe or why it has
// none. The URL must be https because the probe sends a credential in Header.
func validateProbe(p *Provider) error {
	remote := p.Kind == KindBYOKAPI || p.Kind == KindGateway
	if p.Probe != nil {
		if !remote {
			return fmt.Errorf("providercatalog: provider %q kind %q is not remote and must not declare a probe", p.ID, p.Kind)
		}
		if !strings.HasPrefix(p.Probe.URL, "https://") {
			return fmt.Errorf("providercatalog: provider %q probe url %q must be https", p.ID, p.Probe.URL)
		}
		if strings.TrimSpace(p.Probe.Header) == "" {
			return fmt.Errorf("providercatalog: provider %q probe has no header", p.ID)
		}
		if p.ProbeAbsentReason != "" {
			return fmt.Errorf("providercatalog: provider %q declares both a probe and a probe_absent_reason", p.ID)
		}
	}
	if p.ProbeAbsentReason != "" {
		if !remote {
			return fmt.Errorf("providercatalog: provider %q kind %q is not remote and must not declare a probe_absent_reason", p.ID, p.Kind)
		}
		if strings.TrimSpace(p.ProbeAbsentReason) == "" {
			return fmt.Errorf("providercatalog: provider %q has a blank probe_absent_reason", p.ID)
		}
	}
	if p.Status == StatusActive && remote && p.Probe == nil && p.ProbeAbsentReason == "" {
		return fmt.Errorf("providercatalog: active remote provider %q must declare a probe or a probe_absent_reason", p.ID)
	}
	return nil
}

func tierForKind(kind Kind) Tier {
	switch kind {
	case KindLocalResource, KindLocalInProc:
		return TierLocal
	case KindBYOKAPI:
		return TierBYOK
	case KindGateway:
		return TierVrooli
	case KindBrowser:
		return TierBrowser
	default:
		return TierNone
	}
}

func hasCapability(p *Provider, capability Capability) bool {
	for _, declared := range p.Capabilities {
		if declared == capability {
			return true
		}
	}
	return false
}

// featureCapabilities is the closed feature vocabulary. Every feature a
// provider may declare must appear here, and every entry maps to a known
// capability so the generated capabilities helper can never emit an
// unspecified row for a declared feature.
//
// Some scenario features (tts-cache, tts-paragraph-split, audio-provider-routing)
// are grouped with the capability they serve rather than treated as separate
// capabilities. The audio-tools rollup is excluded from serviceability grouping
// by ID, so these groupings do not add or remove a health row.
var featureCapabilities = map[string]Capability{
	"voice-input":                CapabilitySTT,
	"voice-streaming":            CapabilitySTT,
	"voice-streaming-realtime":   CapabilitySTT,
	"voice-speaker-verification": CapabilitySpeaker,
	"voice-enrollment":           CapabilitySpeaker,
	"voice-output":               CapabilityTTS,
	"tts-summarization":          CapabilitySummarize,
	"tts-cache":                  CapabilityTTS,
	"tts-paragraph-split":        CapabilityTTS,
	"audio-provider-routing":     CapabilitySTT,
	"ai-command-generation":      CapabilitySummarize,
	"transcode":                  CapabilityTranscode,
}

func cloneProvider(p Provider) Provider {
	p.Capabilities = append([]Capability(nil), p.Capabilities...)
	p.Features = append([]string(nil), p.Features...)
	p.EngineIDs = append([]string(nil), p.EngineIDs...)
	p.RuntimeAliases = append([]string(nil), p.RuntimeAliases...)
	p.Actions = append([]Action(nil), p.Actions...)
	p.Models = append([]string(nil), p.Models...)
	if p.Probe != nil {
		probe := *p.Probe
		p.Probe = &probe
	}
	return p
}

// Default returns the process-wide validated catalogue. A malformed seed is a
// build-time defect, not a runtime condition.
func Default() *Registry {
	return registry
}

var registry = func() *Registry {
	r, err := Load()
	if err != nil {
		panic(err)
	}
	return r
}()

// SchemaVersion returns the seed schema version.
func (r *Registry) SchemaVersion() string { return r.schemaVersion }

// All returns every declared provider in seed order.
func (r *Registry) All() []Provider {
	out := make([]Provider, len(r.providers))
	for i := range r.providers {
		out[i] = cloneProvider(r.providers[i])
	}
	return out
}

// Active returns the providers that are probed and rendered into
// capabilities.Known, in seed order. Planned, experimental, and watch entries
// are excluded.
func (r *Registry) Active() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status == StatusActive {
			out = append(out, cloneProvider(p))
		}
	}
	return out
}

// ByID returns a provider by stable provider ID.
func (r *Registry) ByID(id string) (Provider, bool) {
	p, ok := r.byID[id]
	if !ok {
		return Provider{}, false
	}
	return cloneProvider(p), true
}

// ByCapability returns active providers that declare the capability, in seed
// order. Planned/watch entries are never returned.
func (r *Registry) ByCapability(capability Capability) []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status == StatusActive && hasCapability(&p, capability) {
			out = append(out, cloneProvider(p))
		}
	}
	return out
}

// CapabilityForFeature maps a declared feature string to its catalogue
// capability.
func (r *Registry) CapabilityForFeature(feature string) (Capability, bool) {
	capability, ok := featureCapabilities[feature]
	return capability, ok
}

// TierForProviderID returns the routing tier for a provider ID, or TierNone for
// an unknown ID.
func (r *Registry) TierForProviderID(id string) Tier {
	if p, ok := r.byID[id]; ok {
		return p.Tier
	}
	return TierNone
}

// ResourceSlugForProviderID returns the `vrooli resource <verb> <slug>` slug for
// an active local resource provider. Non-local, planned, and unknown IDs return
// false.
func (r *Registry) ResourceSlugForProviderID(id string) (string, bool) {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive || p.Kind != KindLocalResource || strings.TrimSpace(p.ResourceSlug) == "" {
		return "", false
	}
	return p.ResourceSlug, true
}

// IsLocalProvider reports whether the provider ID is an active local resource
// that lifecycle actions may target.
func (r *Registry) IsLocalProvider(id string) bool {
	_, ok := r.ResourceSlugForProviderID(id)
	return ok
}

// SupportsPullModel reports whether the active provider advertises pull_model.
// Only ollama does.
func (r *Registry) SupportsPullModel(id string) bool {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive {
		return false
	}
	for _, action := range p.Actions {
		if action == ActionPullModel {
			return true
		}
	}
	return false
}

// ActionsForProvider returns a copy of the lifecycle actions advertised by an
// active local resource provider.
func (r *Registry) ActionsForProvider(id string) []Action {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive || p.Kind != KindLocalResource {
		return nil
	}
	return append([]Action(nil), p.Actions...)
}

// LocalProviderIDs returns the active local-resource providers that own a
// lifecycle row, in seed order. Providers that share a resource slug collapse
// to the first/primary row, so ListLocalProviders never emits a duplicate
// resource (sherpa-stt and speaker-verification both map to sherpa-onnx).
func (r *Registry) LocalProviderIDs() []string {
	seenSlugs := make(map[string]struct{})
	out := make([]string, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status != StatusActive || p.Kind != KindLocalResource || len(p.Actions) == 0 || strings.TrimSpace(p.ResourceSlug) == "" {
			continue
		}
		if _, dup := seenSlugs[p.ResourceSlug]; dup {
			continue
		}
		seenSlugs[p.ResourceSlug] = struct{}{}
		out = append(out, p.ID)
	}
	return out
}

// ProviderIDForEngine maps an STT engine ID (for example sherpa-streaming) to
// the stable provider ID used by health, lifecycle, settings, and usage (for
// example sherpa-stt).
func (r *Registry) ProviderIDForEngine(engineID string) (string, bool) {
	id, ok := r.providerByEngine[engineID]
	return id, ok
}

// EngineIDsForProvider returns the STT engine IDs declared by a provider.
func (r *Registry) EngineIDsForProvider(providerID string) []string {
	p, ok := r.byID[providerID]
	if !ok {
		return nil
	}
	return append([]string(nil), p.EngineIDs...)
}

// ProviderIDForRuntimeAlias maps a runtime identity that is not an engine ID
// (for example kokoro-local) to the stable provider ID it canonicalizes to.
func (r *Registry) ProviderIDForRuntimeAlias(raw string) (string, bool) {
	id, ok := r.providerByAlias[raw]
	return id, ok
}

// RuntimeAliasesForProvider returns the runtime identities declared by a
// provider.
func (r *Registry) RuntimeAliasesForProvider(providerID string) []string {
	p, ok := r.byID[providerID]
	if !ok {
		return nil
	}
	return append([]string(nil), p.RuntimeAliases...)
}

// ProbeFor returns the remote readiness probe fact for a provider. The probe
// carries the endpoint and header shape only, never a credential.
func (r *Registry) ProbeFor(providerID string) (Probe, bool) {
	p, ok := r.byID[providerID]
	if !ok || p.Probe == nil {
		return Probe{}, false
	}
	return *p.Probe, true
}

// All returns every provider from the default catalogue.
func All() []Provider { return registry.All() }

// Active returns the active providers from the default catalogue.
func Active() []Provider { return registry.Active() }

// ByID returns a provider from the default catalogue.
func ByID(id string) (Provider, bool) { return registry.ByID(id) }

// ByCapability returns active providers for a capability from the default
// catalogue.
func ByCapability(capability Capability) []Provider { return registry.ByCapability(capability) }

// CapabilityForFeature maps a feature using the default catalogue.
func CapabilityForFeature(feature string) (Capability, bool) {
	return registry.CapabilityForFeature(feature)
}

// TierForProviderID returns the routing tier using the default catalogue.
func TierForProviderID(id string) Tier { return registry.TierForProviderID(id) }

// ResourceSlugForProviderID returns the local resource slug using the default
// catalogue.
func ResourceSlugForProviderID(id string) (string, bool) {
	return registry.ResourceSlugForProviderID(id)
}

// IsLocalProvider reports whether the ID is an active local resource using the
// default catalogue.
func IsLocalProvider(id string) bool { return registry.IsLocalProvider(id) }

// SupportsPullModel reports pull_model support using the default catalogue.
func SupportsPullModel(id string) bool { return registry.SupportsPullModel(id) }

// ActionsForProvider returns lifecycle actions using the default catalogue.
func ActionsForProvider(id string) []Action { return registry.ActionsForProvider(id) }

// LocalProviderIDs returns the lifecycle-row provider IDs using the default
// catalogue.
func LocalProviderIDs() []string { return registry.LocalProviderIDs() }

// ProviderIDForEngine maps an STT engine ID using the default catalogue.
func ProviderIDForEngine(engineID string) (string, bool) {
	return registry.ProviderIDForEngine(engineID)
}

// EngineIDsForProvider returns a provider's engine IDs using the default
// catalogue.
func EngineIDsForProvider(providerID string) []string {
	return registry.EngineIDsForProvider(providerID)
}

// ProviderIDForRuntimeAlias maps a runtime identity using the default catalogue.
func ProviderIDForRuntimeAlias(raw string) (string, bool) {
	return registry.ProviderIDForRuntimeAlias(raw)
}

// RuntimeAliasesForProvider returns a provider's runtime aliases using the
// default catalogue.
func RuntimeAliasesForProvider(providerID string) []string {
	return registry.RuntimeAliasesForProvider(providerID)
}

// ProbeFor returns a provider's readiness probe fact using the default
// catalogue.
func ProbeFor(providerID string) (Probe, bool) {
	return registry.ProbeFor(providerID)
}

// KnownFeatures returns every feature string the seed may use, sorted for
// deterministic generation.
func KnownFeatures() []string {
	out := make([]string, 0, len(featureCapabilities))
	for feature := range featureCapabilities {
		out = append(out, feature)
	}
	sort.Strings(out)
	return out
}
