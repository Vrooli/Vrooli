// Package providercatalog owns audio-tools' declarative provider catalogue: one
// record per provider/engine-ish provider in catalog.seed.json, plus the loader,
// validator, accessors, and the renderers that produce the generated Go and
// TypeScript surfaces.
//
// The catalogue is the single source for the provider IDs that health,
// lifecycle, settings, and usage use, and for the STT engine IDs used by
// streaming selection and explicit unary batch routing. Provider IDs are
// stable; engine IDs are the values the STT runtime reports. Runtime aliases
// are the remaining chain-reported identities that canonicalize to a provider,
// and probe facts are the remote readiness shape for BYOK/gateway providers, so
// adding a provider of an existing kind stays a data edit. See
// docs/concepts/ARCHITECTURE.md and docs/concepts/DOMAINS.md.
package providercatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
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
	KindFullDuplex    Kind = "full_duplex"
	KindBrowser       Kind = "browser"
)

func (k Kind) valid() bool {
	switch k {
	case KindScenario, KindLocalResource, KindLocalInProc, KindBYOKAPI, KindGateway, KindFullDuplex, KindBrowser:
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
	CapabilityDuplex    Capability = "duplex"
)

func (c Capability) valid() bool {
	switch c {
	case CapabilitySTT, CapabilityTTS, CapabilitySummarize, CapabilityTranscode, CapabilitySpeaker, CapabilityDuplex:
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

// CheckerKind names the coarse probe category projected to provider clients.
// Local readiness/liveness construction is selected separately by HealthProfile.
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

// TTSRuntime declares the service configuration needed to route a local TTS
// provider, including whether playback speed is native or post-processed.
type TTSRuntime struct {
	BackendID     string `json:"backend_id"`
	URLField      string `json:"url_field"`
	HealthPath    string `json:"health_path"`
	ReadinessKind string `json:"readiness_kind,omitempty"`
	ModelRole     string `json:"model_role"`
	CacheBackend  string `json:"cache_backend"`
	// SpeedMode declares whether the backend applies requested playback speed
	// itself or whether Audio Tools must synthesize WAV and adjust it locally.
	SpeedMode    string `json:"speed_mode,omitempty"`
	AutoPriority int    `json:"auto_priority,omitempty"`
	// ConditionalAuto permits a candidate model/provider to be tried by auto
	// routing when its runtime is available. It does not promote the model or
	// provider and is restricted by bootstrap to planned providers, candidate
	// models, and explicit commercial permission.
	ConditionalAuto bool `json:"conditional_auto,omitempty"`
	// SharedRuntime declares an optional alternate runtime for this provider's
	// model/protocol (for example sherpa serving the Kokoro model). It keeps
	// provider/model/voice identity on the parent declaration.
	SharedRuntime *TTSSharedRuntime `json:"shared_runtime,omitempty"`
}

type TTSSharedRuntime struct {
	BackendID   string `json:"backend_id"`
	URLField    string `json:"url_field"`
	HealthPath  string `json:"health_path"`
	CachePrefix string `json:"cache_prefix"`
	ModelRole   string `json:"model_role"`
	SpeedMode   string `json:"speed_mode,omitempty"`
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

// ConsentSource identifies the authority's source key for a consent receipt.
type ConsentSource string

// ConsentRequirement declares the source-bound human consent contract for a
// provider. Full-duplex sessions must revalidate the receipt while active.
type ConsentRequirement struct {
	Source                   ConsentSource `json:"source"`
	RevalidateActiveSessions bool          `json:"revalidate_active_sessions"`
}

// HealthProfile selects an executable health-check strategy. Endpoint paths
// for the generic HTTP strategy live in HealthProbe so adding a provider that
// uses an existing HTTP shape is declaration-only.
type HealthProfile string

const (
	// HealthProfileNone is an explicit declaration that this provider has no
	// bootstrap health checker. It must use checker_kind=none and cannot join
	// readiness or liveness aggregates.
	HealthProfileNone             HealthProfile = "none"
	HealthProfileWhisperSilent    HealthProfile = "whisper-silent"
	HealthProfileHTTP             HealthProfile = "http"
	HealthProfileOllamaModelAware HealthProfile = "ollama-model-aware"
	HealthProfileRequestScoped    HealthProfile = "request-scoped"
	HealthProfileTranscode        HealthProfile = "transcode"
)

func (p HealthProfile) Valid() bool {
	switch p {
	case HealthProfileNone, HealthProfileWhisperSilent, HealthProfileHTTP, HealthProfileOllamaModelAware, HealthProfileRequestScoped, HealthProfileTranscode:
		return true
	default:
		return false
	}
}

// HealthBaseURLField selects a configured local service URL. Catalog data can
// select only one of these allowlisted fields, never an arbitrary URL.
type HealthBaseURLField string

const (
	HealthBaseURLWhisper HealthBaseURLField = "WhisperURL"
	HealthBaseURLKyutai  HealthBaseURLField = "KyutaiURL"
	HealthBaseURLSherpa  HealthBaseURLField = "SherpaURL"
	HealthBaseURLOllama  HealthBaseURLField = "OllamaURL"
)

func (f HealthBaseURLField) Valid() bool {
	switch f {
	case HealthBaseURLWhisper, HealthBaseURLKyutai, HealthBaseURLSherpa, HealthBaseURLOllama:
		return true
	default:
		return false
	}
}

// HealthProbe declares generic HTTP readiness and liveness endpoints. BaseURL
// names a constrained Env field, never an arbitrary URL from catalog data.
type HealthProbe struct {
	BaseURLField          HealthBaseURLField `json:"base_url_field"`
	ReadinessPath         string             `json:"readiness_path"`
	LivenessPath          string             `json:"liveness_path"`
	AggregateLivenessPath string             `json:"aggregate_liveness_path,omitempty"`
}

// Provider is one catalogue record.
type Provider struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Kind         Kind         `json:"kind"`
	Tier         Tier         `json:"tier"`
	Status       Status       `json:"status"`
	Capabilities []Capability `json:"capabilities"`
	Features     []string     `json:"features"`
	EngineIDs    []string     `json:"engine_ids,omitempty"`
	// STTEvalAdapterID binds experiment evaluation to a reusable STT adapter.
	// Provider identity and engine identity remain catalogue data; this key
	// selects the executable protocol implementation.
	STTEvalAdapterID string         `json:"stt_eval_adapter_id,omitempty"`
	RuntimeAliases   []string       `json:"runtime_aliases,omitempty"`
	ResourceSlug     string         `json:"resource_slug,omitempty"`
	DependencyKind   DependencyKind `json:"dependency_kind,omitempty"`
	DependencySlug   string         `json:"dependency_slug,omitempty"`
	Platform         Platform       `json:"platform"`
	Actions          []Action       `json:"actions,omitempty"`
	CheckerKind      CheckerKind    `json:"checker_kind"`
	// Probe is the remote readiness probe fact for a BYOK/gateway provider.
	// ProbeAbsentReason documents why an active remote provider cannot be
	// credential-probed instead of silently omitting the fact.
	Probe              *Probe              `json:"probe,omitempty"`
	ProbeAbsentReason  string              `json:"probe_absent_reason,omitempty"`
	ConsentRequirement *ConsentRequirement `json:"consent_requirement,omitempty"`
	DocsURL            string              `json:"docs_url,omitempty"`
	OperatorCommand    string              `json:"operator_command,omitempty"`
	Criticality        Criticality         `json:"criticality"`
	DefActionKind      DefActionKind       `json:"def_action_kind,omitempty"`
	Presentation       Presentation        `json:"presentation"`
	AdapterID          string              `json:"adapter_id,omitempty"`
	// BYOKAdapterID selects a reusable credential protocol implementation. It
	// is independent of AdapterID, which is reserved for TTS voice mapping.
	BYOKAdapterID string `json:"byok_adapter_id,omitempty"`
	// HealthProfile selects a bootstrap checker adapter without coupling the
	// adapter to a provider ID. Aggregate flags declare membership explicitly.
	HealthProfile      HealthProfile `json:"health_profile,omitempty"`
	HealthProbe        *HealthProbe  `json:"health_probe,omitempty"`
	ReadinessAggregate bool          `json:"readiness_aggregate,omitempty"`
	LivenessAggregate  bool          `json:"liveness_aggregate,omitempty"`
	TTSRuntime         *TTSRuntime   `json:"tts_runtime,omitempty"`
	// VoiceAliases maps canonical audio-tools voice IDs to backend-specific
	// voice identifiers for this TTS adapter.
	VoiceAliases map[string]string `json:"voice_aliases,omitempty"`
	License      string            `json:"license,omitempty"`
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
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("providercatalog: parse seed: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("providercatalog: parse seed: expected one JSON document")
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
