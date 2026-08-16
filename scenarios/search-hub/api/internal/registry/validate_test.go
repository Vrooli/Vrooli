package registry_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/types/known/durationpb"

	"search-hub/internal/registry"
)

// validActive returns a fully-populated, valid ACTIVE descriptor that tests
// mutate to exercise one failing rule at a time.
func validActive() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "cli-health.commands",
		ProviderGroup: "cli-health",
		Bucket:        registryv1.Bucket_BUCKET_DO,
		Type:          "command",
		Description:   "CLI commands searchable by what they do.",
		Lifecycle:     registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		Endpoint: &registryv1.Endpoint{
			Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
				ScenarioId: "cli-health",
				Path:       "/vrooli.cli_health.v1.search.SearchService/Search",
				Method:     registryv1.HttpMethod_HTTP_METHOD_POST,
			}},
		},
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results",
			IdField:     "name",
			TitleField:  "name",
			ScoreField:  "score",
			ScoreScale:  registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
		StatusEndpoint: &registryv1.Endpoint{
			Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
				ScenarioId: "cli-health", Path: "/status",
			}},
		},
	}
}

func TestValidateLifecycleEnum(t *testing.T) {
	for _, lifecycle := range []registryv1.Lifecycle{
		registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		registryv1.Lifecycle_LIFECYCLE_FIXTURE,
		registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL,
	} {
		d := validActive()
		d.Lifecycle = lifecycle
		if err := registry.Validate(d); err != nil {
			t.Fatalf("lifecycle %s: %v", lifecycle, err)
		}
	}
	d := validActive()
	d.Lifecycle = registryv1.Lifecycle(99)
	if err := registry.Validate(d); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("unknown lifecycle error = %v", err)
	}
	d.Lifecycle = registryv1.Lifecycle_LIFECYCLE_UNSPECIFIED
	if err := registry.Validate(d); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("unspecified lifecycle error = %v", err)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	d := validActive()
	d.State = registryv1.ProviderState_PROVIDER_STATE_UNSPECIFIED
	d.Scope = registryv1.Scope_SCOPE_UNSPECIFIED
	d.ProviderId = "  cli-health.commands  "

	registry.Normalize(d)

	require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_ACTIVE, d.State)
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, d.Scope)
	require.Equal(t, registryv1.Lifecycle_LIFECYCLE_PRODUCTION, d.Lifecycle)
	require.Equal(t, registry.DefaultFreshnessBudget, d.GetFreshnessBudget().AsDuration())
	require.Equal(t, "cli-health.commands", d.ProviderId, "identity fields trimmed")
}

func TestValidateActiveOK(t *testing.T) {
	d := validActive()
	registry.Normalize(d)
	require.NoError(t, registry.Validate(d))
}

func TestValidateStatusEndpointDefaultsIndexTimestampDeclaration(t *testing.T) {
	d := validActive()
	d.StatusEndpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: "cli-health", Path: "/status"}}}
	registry.Normalize(d)
	require.Equal(t, "last_indexed_at", d.GetIndexTimestampField())
	require.NoError(t, registry.Validate(d))
	d.IndexTimestampField = "index.last_reconcile_at"
	require.NoError(t, registry.Validate(d))
}

func TestValidateActiveFailures(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*registryv1.ProviderDescriptor)
		wantField string
	}{
		{"missing provider_id", func(d *registryv1.ProviderDescriptor) { d.ProviderId = "" }, "provider_id"},
		{"missing provider_group", func(d *registryv1.ProviderDescriptor) { d.ProviderGroup = "" }, "provider_group"},
		{"missing type", func(d *registryv1.ProviderDescriptor) { d.Type = "" }, "type"},
		{"missing description", func(d *registryv1.ProviderDescriptor) { d.Description = "   " }, "description"},
		{"unspecified bucket", func(d *registryv1.ProviderDescriptor) { d.Bucket = registryv1.Bucket_BUCKET_UNSPECIFIED }, "bucket"},
		{"nil endpoint", func(d *registryv1.ProviderDescriptor) { d.Endpoint = nil }, "endpoint"},
		{"endpoint missing scenario_id", func(d *registryv1.ProviderDescriptor) {
			d.Endpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{Path: "/x"}}}
		}, "endpoint.http_json.scenario_id"},
		{"endpoint missing path", func(d *registryv1.ProviderDescriptor) {
			d.Endpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: "x"}}}
		}, "endpoint.http_json.path"},
		{"nil result_mapping", func(d *registryv1.ProviderDescriptor) { d.ResultMapping = nil }, "result_mapping"},
		{"mapping missing results_path", func(d *registryv1.ProviderDescriptor) { d.ResultMapping.ResultsPath = "" }, "result_mapping.results_path"},
		{"mapping missing id_field", func(d *registryv1.ProviderDescriptor) { d.ResultMapping.IdField = "" }, "result_mapping.id_field"},
		{"mapping missing title_field", func(d *registryv1.ProviderDescriptor) { d.ResultMapping.TitleField = "" }, "result_mapping.title_field"},
		{"mapping missing score_field", func(d *registryv1.ProviderDescriptor) { d.ResultMapping.ScoreField = "" }, "result_mapping.score_field"},
		{"filter_value without filter_field", func(d *registryv1.ProviderDescriptor) { d.ResultMapping.FilterValue = "surface" }, "result_mapping.filter_field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := validActive()
			registry.Normalize(d)
			tc.mutate(d)
			err := registry.Validate(d)
			require.Error(t, err)
			var invalid registry.ErrInvalidDescriptor
			require.True(t, errors.As(err, &invalid))
			require.Equal(t, tc.wantField, invalid.Field)
		})
	}
}

func TestValidateRejectsNegativeFreshnessBudget(t *testing.T) {
	d := validActive()
	d.FreshnessBudget = durationpb.New(-time.Hour)
	registry.Normalize(d)
	err := registry.Validate(d)
	var invalid registry.ErrInvalidDescriptor
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "freshness_budget", invalid.Field)
}

func TestValidateProductionRegistrationRequiresStatusEndpoint(t *testing.T) {
	d := validActive()
	d.StatusEndpoint = nil
	registry.Normalize(d)
	err := registry.ValidateProductionRegistration(d)
	var invalid registry.ErrInvalidDescriptor
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "status_endpoint", invalid.Field)
}

func TestValidateCapabilityGap(t *testing.T) {
	base := func() *registryv1.ProviderDescriptor {
		return &registryv1.ProviderDescriptor{
			ProviderId:    "code.symbols",
			ProviderGroup: "code-reference",
			Bucket:        registryv1.Bucket_BUCKET_REUSE,
			Type:          "code",
			Description:   "Source symbols searchable by intent.",
			Lifecycle:     registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
			State:         registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP,
			IntendedHome:  "code-reference",
		}
	}

	t.Run("valid gap needs no endpoint/mapping", func(t *testing.T) {
		d := base()
		registry.Normalize(d)
		require.NoError(t, registry.Validate(d))
	})

	t.Run("gap requires intended_home", func(t *testing.T) {
		d := base()
		d.IntendedHome = ""
		registry.Normalize(d)
		err := registry.Validate(d)
		var invalid registry.ErrInvalidDescriptor
		require.True(t, errors.As(err, &invalid))
		require.Equal(t, "intended_home", invalid.Field)
	})

	t.Run("gap must not carry an endpoint", func(t *testing.T) {
		d := base()
		d.Endpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: "x", Path: "/y"}}}
		registry.Normalize(d)
		err := registry.Validate(d)
		var invalid registry.ErrInvalidDescriptor
		require.True(t, errors.As(err, &invalid))
		require.Equal(t, "endpoint", invalid.Field)
	})
}
