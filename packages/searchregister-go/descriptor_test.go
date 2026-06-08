package searchregister_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	aisearch "github.com/vrooli/aisearch-go"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// cliHealthSearchJSON mirrors scenarios/cli-health/.vrooli/search.json closely
// enough to prove the descriptor mapping (the tuning/tests blocks are present so
// the test also proves they are dropped from the descriptor).
const cliHealthSearchJSON = `{
  "version": "1.0.0",
  "providers": [{
    "provider_id": "cli-health.commands",
    "provider_group": "cli-health",
    "bucket": "BUCKET_DO",
    "type": "command",
    "description": "Vrooli CLI commands searchable by what each does.",
    "scope": "SCOPE_PROJECT",
    "endpoint": {
      "http_json": {
        "scenario_id": "cli-health",
        "path": "/vrooli.cli_health.v1.search.SearchService/Search",
        "method": "HTTP_METHOD_POST",
        "body_template": "{\"query\":\"{{query}}\",\"limit\":{{limit}}}",
        "headers": { "Content-Type": "application/json" }
      }
    },
    "status_endpoint": {
      "http_json": {
        "scenario_id": "cli-health",
        "path": "/vrooli.cli_health.v1.search.SearchService/Status",
        "method": "HTTP_METHOD_POST",
        "body_template": "{}"
      }
    },
    "result_mapping": {
      "results_path": "results",
      "id_field": "name",
      "title_field": "name",
      "score_field": "score",
      "snippet_field": "description",
      "path_field": "name",
      "score_scale": "SCORE_SCALE_COSINE_0_1"
    },
    "tuning": { "engine": "dense", "rerank_enabled": true },
    "tests": { "cases": [{ "id": "x", "query": "restart a scenario" }] }
  }]
}`

func mustParse(t *testing.T, raw string) aisearch.SearchFile {
	t.Helper()
	f, err := aisearch.ParseSearchFile([]byte(raw))
	require.NoError(t, err)
	return f
}

func TestDescriptorMapsAllDescriptorFields(t *testing.T) {
	f := mustParse(t, cliHealthSearchJSON)
	d, err := searchregister.Descriptor(f.Providers[0])
	require.NoError(t, err)

	require.Equal(t, "cli-health.commands", d.GetProviderId())
	require.Equal(t, "cli-health", d.GetProviderGroup())
	require.Equal(t, "command", d.GetType())
	require.Equal(t, registryv1.Bucket_BUCKET_DO, d.GetBucket())
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, d.GetScope())
	require.Contains(t, d.GetDescription(), "Vrooli CLI commands")

	hj := d.GetEndpoint().GetHttpJson()
	require.NotNil(t, hj)
	require.Equal(t, "cli-health", hj.GetScenarioId())
	require.Equal(t, "/vrooli.cli_health.v1.search.SearchService/Search", hj.GetPath())
	require.Equal(t, registryv1.HttpMethod_HTTP_METHOD_POST, hj.GetMethod())
	require.Equal(t, "application/json", hj.GetHeaders()["Content-Type"])

	require.Equal(t, "cli-health", d.GetStatusEndpoint().GetHttpJson().GetScenarioId())

	rm := d.GetResultMapping()
	require.NotNil(t, rm)
	require.Equal(t, "results", rm.GetResultsPath())
	require.Equal(t, "name", rm.GetIdField())
	require.Equal(t, "description", rm.GetSnippetField())
	require.Equal(t, registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1, rm.GetScoreScale())

	// State is intentionally left unset; search-hub's Normalize defaults it to
	// ACTIVE on registration. The mapping must not invent a value.
	require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_UNSPECIFIED, d.GetState())
	// query_hint/intended_home are not part of search.json's descriptor surface.
	require.Empty(t, d.GetQueryHint())
	require.Empty(t, d.GetIntendedHome())
}

// TestDescriptorMapsTuningDropsTests proves the tuning block IS carried onto the
// descriptor (proto field 15) as the RESOLVED tuning (taxonomy defaults filled),
// while the tests block is still dropped (ProviderDescriptor has no tests field;
// the corpus self-registers through EvalService.RegisterSuite instead).
func TestDescriptorMapsTuningDropsTests(t *testing.T) {
	f := mustParse(t, cliHealthSearchJSON)
	d, err := searchregister.Descriptor(f.Providers[0])
	require.NoError(t, err)

	tn := d.GetTuning()
	require.NotNil(t, tn, "tuning must be carried on the descriptor")
	require.Equal(t, "dense", tn.GetEngine())
	require.True(t, tn.GetRerankEnabled())
	// Defaults filled by ResolvedTuning (WithDefaults): model + shortlist.
	require.Equal(t, "nomic-embed-text", tn.GetEmbedModel())
	require.Equal(t, int32(50), tn.GetRerankShortlist())
	require.NotNil(t, tn.GetFloor(), "floor block always present (zero = regime default)")
}

func TestDescriptorsPreservesOrder(t *testing.T) {
	const twoProviders = `{
      "version": "1.0.0",
      "providers": [
        { "provider_id": "a.one", "provider_group": "a", "bucket": "BUCKET_DO", "type": "command",
          "description": "first", "endpoint": { "cli": { "argv_template": ["a", "{{query}}"] } },
          "result_mapping": { "results_path": "r", "id_field": "id", "title_field": "t", "score_field": "s" },
          "tuning": { "engine": "dense" }, "tests": { "cases": [] } },
        { "provider_id": "b.two", "provider_group": "b", "bucket": "BUCKET_KNOW", "type": "doc",
          "description": "second", "endpoint": { "cli": { "argv_template": ["b", "{{query}}"] } },
          "result_mapping": { "results_path": "r", "id_field": "id", "title_field": "t", "score_field": "s" },
          "tuning": { "engine": "hybrid" }, "tests": { "cases": [] } }
      ]
    }`
	f := mustParse(t, twoProviders)
	ds, err := searchregister.Descriptors(f)
	require.NoError(t, err)
	require.Len(t, ds, 2)
	require.Equal(t, "a.one", ds[0].GetProviderId())
	require.Equal(t, "b.two", ds[1].GetProviderId())
	require.Equal(t, registryv1.Bucket_BUCKET_KNOW, ds[1].GetBucket())
}

// TestDescriptorOmitsUnspecifiedEnums proves an absent bucket/scope leaves the
// enum UNSPECIFIED rather than feeding protojson an empty string (which it would
// reject). The server-side Validate is what rejects an UNSPECIFIED bucket; the
// mapper must not turn a thin file into a parse error.
// TestDescriptorMapsControlEndpoints proves the secured control-plane targets
// (reindex_endpoint / config_endpoint) round-trip from search.json onto the
// descriptor so search-hub can route the token-gated control RPCs. A provider
// that omits them leaves both unset (covered by the other descriptor tests).
func TestDescriptorMapsControlEndpoints(t *testing.T) {
	const withControl = `{
      "version": "1.0.0",
      "providers": [{
        "provider_id": "cli-health.commands",
        "provider_group": "cli-health",
        "bucket": "BUCKET_DO",
        "type": "command",
        "description": "commands",
        "endpoint": { "http_json": { "scenario_id": "cli-health", "path": "/search" } },
        "reindex_endpoint": { "http_json": { "scenario_id": "cli-health", "path": "/vrooli.search_hub.v1.control.SearchControlService/Reindex", "method": "HTTP_METHOD_POST" } },
        "config_endpoint": { "http_json": { "scenario_id": "cli-health", "path": "/vrooli.search_hub.v1.control.SearchControlService/WriteConfig", "method": "HTTP_METHOD_POST" } },
        "result_mapping": { "results_path": "results", "id_field": "name", "title_field": "name", "score_field": "score" },
        "tuning": { "engine": "dense" },
        "tests": { "cases": [] }
      }]
    }`
	f := mustParse(t, withControl)
	d, err := searchregister.Descriptor(f.Providers[0])
	require.NoError(t, err)

	ri := d.GetReindexEndpoint().GetHttpJson()
	require.NotNil(t, ri, "reindex_endpoint must be carried onto the descriptor")
	require.Equal(t, "cli-health", ri.GetScenarioId())
	require.Equal(t, "/vrooli.search_hub.v1.control.SearchControlService/Reindex", ri.GetPath())

	cf := d.GetConfigEndpoint().GetHttpJson()
	require.NotNil(t, cf, "config_endpoint must be carried onto the descriptor")
	require.Equal(t, "/vrooli.search_hub.v1.control.SearchControlService/WriteConfig", cf.GetPath())
}

// TestTuningProtoRoundTrip proves the proto<->TuningConfig converters are exact
// inverses for a fully-populated config (every factor + floor), so the config-
// write contract can carry a tuning out and back without losing a field.
func TestTuningProtoRoundTrip(t *testing.T) {
	in := aisearch.TuningConfig{
		Engine:          aisearch.EngineHybrid,
		EmbedModel:      "nomic-embed-text",
		EmbedTaskPrefix: true,
		RerankEnabled:   true,
		RerankBlend:     true,
		RerankShortlist: 80,
		Floor:           aisearch.FloorTuning{MaxGap: 0.4, HardFloor: 0.15},
	}
	out := searchregister.TuningFromProto(searchregister.TuningToProto(in))
	require.Equal(t, in, out)
}

func TestTuningFromProtoNil(t *testing.T) {
	require.Equal(t, aisearch.TuningConfig{}, searchregister.TuningFromProto(nil))
}

func TestDescriptorOmitsUnspecifiedEnums(t *testing.T) {
	p := aisearch.ProviderConfig{
		ProviderID:    "x.leaf",
		ProviderGroup: "x",
		Type:          "thing",
		Description:   "d",
	}
	d, err := searchregister.Descriptor(p)
	require.NoError(t, err)
	require.Equal(t, registryv1.Bucket_BUCKET_UNSPECIFIED, d.GetBucket())
	require.Equal(t, registryv1.Scope_SCOPE_UNSPECIFIED, d.GetScope())
}
