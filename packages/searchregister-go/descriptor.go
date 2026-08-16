// Package searchregister is the shared self-registration bridge between a
// scenario's `.vrooli/search.json` SSOT (parsed by aisearch-go) and search-hub's
// RegistryService contract (the registry proto). A scenario that owns a
// searchable corpus calls Register at boot to push its provider descriptor(s) to
// search-hub — replacing the old model where search-hub shipped every provider's
// descriptor as an embedded build-time seed.
//
// Layering: this package is allowed to know BOTH the search.json shape
// (aisearch-go) and the registry transport (proto/connect) — that coupling is
// precisely its job. aisearch-go itself stays free of registry/transport
// vocabulary (it keeps the descriptor sub-objects as raw JSON); search-hub stays
// free of any per-provider knowledge (a provider is a registry row, never code).
// The bridge lives here so neither side has to import the other.
package searchregister

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	aisearch "github.com/vrooli/ai-go/search"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// Descriptor maps one parsed search.json provider block to the registry
// ProviderDescriptor that RegistryService.RegisterProvider accepts.
//
// The descriptor-shaped fields of search.json are protojson by construction
// (the file's endpoint/result_mapping/status_endpoint sub-objects are kept as
// raw JSON exactly so they map here without a hand-written field-by-field
// translation that could drift from the proto). The tuning and tests blocks are
// NOT part of the descriptor contract — they are intentionally dropped here;
// search-hub persists them through the richer registration payload once the
// proto carries them (a later phase), and the scenario's own boot path reads
// them directly from search.json.
//
// The tuning block IS now mapped onto the descriptor's `tuning` field (proto
// field 15, added in Phase 3): search-hub persists the incumbent tuning
// alongside the descriptor so the sweep can read it via ListProviders/Get
// without a second round-trip. The tests block is still dropped HERE — but it is
// no longer lost: it self-registers through EvalService.RegisterSuite alongside
// this descriptor push (see Register → registerCorpus, which converts the corpus
// with corpus.go's SuiteToProto). The descriptor carries routing+tuning; the
// corpus rides the sibling eval RPC. The scenario's own boot path reads both
// tuning and tests directly from search.json (the SSOT); search-hub's stored
// copies (registry tuning + eval suite) are verified mirrors, never authoritative
// (corpusStoreMirrorsFile).
//
// State is deliberately left unset: search-hub's registry Normalize defaults an
// unspecified state to ACTIVE, so a live provider's search.json need not repeat
// it. A capability-gap stub is a search-hub planning artifact, never a
// self-registering scenario, so it has no search.json to map.
func Descriptor(p aisearch.ProviderConfig) (*registryv1.ProviderDescriptor, error) {
	obj := map[string]json.RawMessage{}
	var err error
	setString := func(key, val string) error {
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", key, err)
		}
		obj[key] = b
		return nil
	}

	for key, val := range map[string]string{
		"provider_id":    p.ProviderID,
		"provider_group": p.ProviderGroup,
		"type":           p.Type,
		"description":    p.Description,
	} {
		if err := setString(key, val); err != nil {
			return nil, err
		}
	}
	// Enums map by name (e.g. "BUCKET_DO"); only emit when present so an empty
	// string never reaches protojson (which would reject it as an unknown enum
	// value). An absent bucket/scope is caught by the server-side Validate.
	if p.Bucket != "" {
		if err := setString("bucket", p.Bucket); err != nil {
			return nil, err
		}
	}
	if p.Scope != "" {
		if err := setString("scope", p.Scope); err != nil {
			return nil, err
		}
	}
	lifecycle := p.Lifecycle
	if lifecycle == "" {
		lifecycle = "production"
	}
	switch strings.ToLower(strings.TrimSpace(lifecycle)) {
	case "production":
		lifecycle = "LIFECYCLE_PRODUCTION"
	case "fixture":
		lifecycle = "LIFECYCLE_FIXTURE"
	case "experimental":
		lifecycle = "LIFECYCLE_EXPERIMENTAL"
	default:
		return nil, fmt.Errorf("provider %q has invalid lifecycle %q: want production, fixture, or experimental", p.ProviderID, lifecycle)
	}
	if err := setString("lifecycle", lifecycle); err != nil {
		return nil, err
	}
	if minimum := p.Tests.Minimum; minimum != nil {
		obj["tests_minimum"], err = json.Marshal(map[string]any{
			"reviewed_positive": minimum.ReviewedPositive,
			"negative":          minimum.Negative,
			"required_tags":     minimum.RequiredTags,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal tests_minimum: %w", err)
		}
	}
	if reason := strings.TrimSpace(p.Scoring.JunkLeakOptOutReason); reason != "" {
		obj["junk_leak_opt_out_reason"], err = json.Marshal(reason)
		if err != nil {
			return nil, fmt.Errorf("marshal junk leak opt-out reason: %w", err)
		}
	}
	if p.RoutingProfile.AnswerSpaces != nil || p.RoutingProfile.Intents != nil || p.RoutingProfile.PositiveExamples != nil || p.RoutingProfile.Exclusions != nil {
		if err := p.RoutingProfile.Validate(); err != nil {
			return nil, fmt.Errorf("provider %q: %w", p.ProviderID, err)
		}
		obj["routing_profile"], err = json.Marshal(map[string]any{
			"answer_spaces":     p.RoutingProfile.AnswerSpaces,
			"intents":           p.RoutingProfile.Intents,
			"positive_examples": p.RoutingProfile.PositiveExamples,
			"exclusions":        p.RoutingProfile.Exclusions,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal routing_profile: %w", err)
		}
	}
	// Descriptor sub-objects are already protojson; pass them through verbatim.
	if len(p.Endpoint) > 0 {
		obj["endpoint"] = p.Endpoint
	}
	if len(p.StatusEndpoint) > 0 {
		obj["status_endpoint"] = p.StatusEndpoint
		field := strings.TrimSpace(p.IndexTimestampField)
		if field == "" {
			field = "last_indexed_at"
		}
		obj["index_timestamp_field"], err = json.Marshal(field)
		if err != nil {
			return nil, fmt.Errorf("marshal index_timestamp_field: %w", err)
		}
	}
	if len(p.ResultMapping) > 0 {
		obj["result_mapping"] = p.ResultMapping
	}
	// Secured control-plane targets — the same Endpoint shape, mapped onto the
	// descriptor's reindex_endpoint/config_endpoint so search-hub can route the
	// token-gated reindex + config-write RPCs to a provider that declares them. A
	// provider with no control plane simply omits both blocks from search.json.
	if len(p.ReindexEndpoint) > 0 {
		obj["reindex_endpoint"] = p.ReindexEndpoint
	}
	if len(p.ConfigEndpoint) > 0 {
		obj["config_endpoint"] = p.ConfigEndpoint
	}

	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("assemble descriptor JSON: %w", err)
	}
	d := &registryv1.ProviderDescriptor{}
	if err := protojson.Unmarshal(raw, d); err != nil {
		return nil, fmt.Errorf("map provider %q to descriptor: %w", p.ProviderID, err)
	}
	d.Tuning = TuningToProto(p.ResolvedTuning())
	return d, nil
}

// TuningToProto maps a TuningConfig (caller should pass the resolved, defaults-
// filled value) to the registry proto Tuning message. The proto field names mirror
// the TuningConfig JSON tags 1:1 (engine, embed_*, rerank_*, floor.{max_gap,
// hard_floor, hybrid_fusion}), so this is a direct field copy — the wire carrier transports
// values; the authoritative factor taxonomy stays in aisearch-go (tuning.go). It
// is exported because the config-write contract (search-hub.v1.control) carries a
// registry.Tuning too, so both the registration push and the write-back response
// build the wire shape through this one converter.
func TuningToProto(t aisearch.TuningConfig) *registryv1.Tuning {
	return &registryv1.Tuning{
		Engine:           t.Engine,
		EmbedModel:       t.EmbedModel,
		EmbedTaskPrefix:  t.EmbedTaskPrefix,
		RerankEnabled:    t.RerankEnabled,
		RerankBlend:      t.RerankBlend,
		RerankShortlist:  int32(t.RerankShortlist),
		RerankPreference: t.RerankPreference,
		HybridFusion:     t.HybridFusion,
		Floor: &registryv1.FloorConfig{
			MaxGap:    t.Floor.MaxGap,
			HardFloor: t.Floor.HardFloor,
		},
	}
}

// TuningFromProto is the inverse of TuningToProto: it maps a registry proto Tuning
// message (e.g. the one carried by a control.WriteConfigRequest) back into the
// aisearch TuningConfig the provider validates and persists. A nil message yields
// the zero TuningConfig (the caller fills taxonomy defaults via WithDefaults). The
// proto only transports values — the taxonomy (legal ranges, defaults) is enforced
// downstream by TuningConfig.Validate / WithDefaults, never here.
func TuningFromProto(t *registryv1.Tuning) aisearch.TuningConfig {
	if t == nil {
		return aisearch.TuningConfig{}
	}
	cfg := aisearch.TuningConfig{
		Engine:           t.GetEngine(),
		EmbedModel:       t.GetEmbedModel(),
		EmbedTaskPrefix:  t.GetEmbedTaskPrefix(),
		RerankEnabled:    t.GetRerankEnabled(),
		RerankBlend:      t.GetRerankBlend(),
		RerankShortlist:  int(t.GetRerankShortlist()),
		RerankPreference: t.GetRerankPreference(),
		HybridFusion:     t.GetHybridFusion(),
	}
	if f := t.GetFloor(); f != nil {
		cfg.Floor = aisearch.FloorTuning{MaxGap: f.GetMaxGap(), HardFloor: f.GetHardFloor()}
	}
	return cfg
}

// Descriptors maps every provider in a parsed search.json file, preserving order.
func Descriptors(f aisearch.SearchFile) ([]*registryv1.ProviderDescriptor, error) {
	out := make([]*registryv1.ProviderDescriptor, 0, len(f.Providers))
	for _, p := range f.Providers {
		d, err := Descriptor(p)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
