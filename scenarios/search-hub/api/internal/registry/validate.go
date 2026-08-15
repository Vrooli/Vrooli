package registry

import (
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// Normalize applies the descriptor's defaults in place before validation and
// persistence: an unspecified state becomes ACTIVE, an unspecified scope
// becomes PROJECT. Whitespace is trimmed from the identity fields so two
// descriptors that differ only by trailing spaces don't create distinct rows.
// Normalize never rejects — Validate is the gate.
func Normalize(d *registryv1.ProviderDescriptor) {
	if d == nil {
		return
	}
	d.ProviderId = strings.TrimSpace(d.ProviderId)
	d.ProviderGroup = strings.TrimSpace(d.ProviderGroup)
	d.Type = strings.TrimSpace(d.Type)
	if d.State == registryv1.ProviderState_PROVIDER_STATE_UNSPECIFIED {
		d.State = registryv1.ProviderState_PROVIDER_STATE_ACTIVE
	}
	if d.Scope == registryv1.Scope_SCOPE_UNSPECIFIED {
		d.Scope = registryv1.Scope_SCOPE_PROJECT
	}
	if d.StatusEndpoint != nil && strings.TrimSpace(d.IndexTimestampField) == "" {
		d.IndexTimestampField = "last_indexed_at"
	}
}

// Validate enforces the descriptor invariants the router depends on. It assumes
// Normalize has already run (callers — the store — run Normalize first). The
// rules differ by state:
//
//   - Always: provider_id, provider_group, type, description, bucket are
//     required (the classifier routes on description; bucket is the routing
//     facet, so neither may be unspecified).
//   - ACTIVE leaves must be callable: a present endpoint and a result_mapping
//     with at least a results_path and id/title/score selectors, so the
//     generic adapter can map responses with no provider-specific code.
//   - CAPABILITY_GAP stubs are intentionally NOT callable: they must declare an
//     intended_home and must NOT carry an endpoint (they are Track-A TODOs that
//     surface in `providers list`, not live providers).
//
// The first failing rule is returned as ErrInvalidDescriptor.
func Validate(d *registryv1.ProviderDescriptor) error {
	if d == nil {
		return ErrInvalidDescriptor{Field: "descriptor", Reason: "required"}
	}
	if d.ProviderId == "" {
		return ErrInvalidDescriptor{Field: "provider_id", Reason: "required"}
	}
	if d.ProviderGroup == "" {
		return ErrInvalidDescriptor{Field: "provider_group", Reason: "required"}
	}
	if d.Type == "" {
		return ErrInvalidDescriptor{Field: "type", Reason: "required"}
	}
	if strings.TrimSpace(d.Description) == "" {
		return ErrInvalidDescriptor{Field: "description", Reason: "required (the classifier routes on it)"}
	}
	if d.Bucket == registryv1.Bucket_BUCKET_UNSPECIFIED {
		return ErrInvalidDescriptor{Field: "bucket", Reason: "must be one of DO/REUSE/KNOW/STATE"}
	}
	switch d.Lifecycle {
	case registryv1.Lifecycle_LIFECYCLE_PRODUCTION, registryv1.Lifecycle_LIFECYCLE_FIXTURE, registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL:
	default:
		return ErrInvalidDescriptor{Field: "lifecycle", Reason: "must be LIFECYCLE_PRODUCTION, LIFECYCLE_FIXTURE, or LIFECYCLE_EXPERIMENTAL"}
	}

	switch d.State {
	case registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP:
		if strings.TrimSpace(d.IntendedHome) == "" {
			return ErrInvalidDescriptor{Field: "intended_home", Reason: "required for a capability_gap stub"}
		}
		if d.Endpoint != nil {
			return ErrInvalidDescriptor{Field: "endpoint", Reason: "a capability_gap stub must not declare an endpoint"}
		}
	default: // ACTIVE
		if err := validateEndpoint(d.Endpoint); err != nil {
			return err
		}
		if err := validateResultMapping(d.ResultMapping); err != nil {
			return err
		}
	}
	return nil
}

func validateEndpoint(e *registryv1.Endpoint) error {
	if e == nil || e.Kind == nil {
		return ErrInvalidDescriptor{Field: "endpoint", Reason: "required for an active provider"}
	}
	switch k := e.Kind.(type) {
	case *registryv1.Endpoint_HttpJson:
		if k.HttpJson == nil || strings.TrimSpace(k.HttpJson.ScenarioId) == "" {
			return ErrInvalidDescriptor{Field: "endpoint.http_json.scenario_id", Reason: "required (the resolver supplies the base URL)"}
		}
		if strings.TrimSpace(k.HttpJson.Path) == "" {
			return ErrInvalidDescriptor{Field: "endpoint.http_json.path", Reason: "required"}
		}
	case *registryv1.Endpoint_Cli:
		if k.Cli == nil || len(k.Cli.ArgvTemplate) == 0 {
			return ErrInvalidDescriptor{Field: "endpoint.cli.argv_template", Reason: "required for a CLI endpoint"}
		}
	default:
		return ErrInvalidDescriptor{Field: "endpoint", Reason: "unsupported endpoint kind"}
	}
	return nil
}

func validateResultMapping(m *registryv1.ResultMapping) error {
	if m == nil {
		return ErrInvalidDescriptor{Field: "result_mapping", Reason: "required for an active provider"}
	}
	if strings.TrimSpace(m.ResultsPath) == "" {
		return ErrInvalidDescriptor{Field: "result_mapping.results_path", Reason: "required (JSON path to the result array)"}
	}
	if strings.TrimSpace(m.IdField) == "" {
		return ErrInvalidDescriptor{Field: "result_mapping.id_field", Reason: "required"}
	}
	if strings.TrimSpace(m.TitleField) == "" {
		return ErrInvalidDescriptor{Field: "result_mapping.title_field", Reason: "required"}
	}
	if strings.TrimSpace(m.ScoreField) == "" {
		return ErrInvalidDescriptor{Field: "result_mapping.score_field", Reason: "required"}
	}
	// filter_field/filter_value are jointly optional: a filter_value without a
	// filter_field can never match, so reject that combination early.
	if strings.TrimSpace(m.FilterValue) != "" && strings.TrimSpace(m.FilterField) == "" {
		return ErrInvalidDescriptor{Field: "result_mapping.filter_field", Reason: "required when filter_value is set"}
	}
	return nil
}
