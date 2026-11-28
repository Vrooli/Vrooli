package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/endpoints"
)

func TestParseProcedure(t *testing.T) {
	cases := []struct {
		path           string
		wantSvc, wantM string
		wantOK         bool
	}{
		{"/vrooli.image_tools.v1.ai.AIService/ListAIOperations", "AIService", "ListAIOperations", true},
		{"/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario", "ScenarioValidationService", "ValidateScenario", true},
		{"/health", "", "", false},
		{"/api/v1/blobs/{key}", "", "", false},
		{"/vrooli.x.v1.S", "", "", false}, // no method
	}
	for _, c := range cases {
		svc, m, ok := parseProcedure(c.path)
		if ok != c.wantOK || svc != c.wantSvc || m != c.wantM {
			t.Errorf("parseProcedure(%q) = (%q,%q,%v), want (%q,%q,%v)", c.path, svc, m, ok, c.wantSvc, c.wantM, c.wantOK)
		}
	}
}

func TestValidateTransport(t *testing.T) {
	ok := []endpoints.EndpointDescriptor{
		{ID: "connect", Path: "/vrooli.x.v1.s.SvcService/Do"},
		{ID: "rest", Path: "/health", RESTException: &endpoints.RESTException{Reason: endpoints.RESTReasonOpsProbe}},
	}
	if err := validateTransport(ok); err != nil {
		t.Fatalf("validateTransport(ok) = %v", err)
	}

	bad := [][]endpoints.EndpointDescriptor{
		{{ID: "connect-with-exc", Path: "/vrooli.x.v1.s.SvcService/Do", RESTException: &endpoints.RESTException{Reason: endpoints.RESTReasonOpsProbe}}},
		{{ID: "rest-no-exc", Path: "/api/v1/foo"}},
		{{ID: "bad-reason", Path: "/api/v1/foo", RESTException: &endpoints.RESTException{Reason: "made_up"}}},
	}
	for i, eps := range bad {
		if err := validateTransport(eps); err == nil {
			t.Errorf("validateTransport(bad[%d]) = nil, want error", i)
		}
	}
}

// writeManifest writes a minimal cli/manifest.json for the coverage tests.
func writeManifest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const sampleManifest = `{
  "name": "demo",
  "groups": [
    {"name": "jobs", "commands": [
      {"name": "get", "binding": {"kind": "connect-rpc", "service": "JobsService", "method": "GetJob"}},
      {"name": "list", "binding": {"kind": "connect-rpc", "service": "JobsService", "method": "ListJobs"}}
    ]},
    {"name": "query", "commands": [
      {"name": "query", "binding": {"kind": "connect-rpc", "service": "RoutingService", "method": "Query"}}
    ]}
  ],
  "omitted": [
    {"service": "JobsService", "method": "WatchJob", "reason": "server-streaming"}
  ]
}`

func loadSample(t *testing.T) manifestView {
	t.Helper()
	v, err := loadManifest(writeManifest(t, sampleManifest))
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	return v
}

func TestValidateCLICoverage_OK(t *testing.T) {
	mf := loadSample(t)
	eps := []endpoints.EndpointDescriptor{
		{ID: "jobs_get", Path: "/vrooli.demo.v1.jobs.JobsService/GetJob"},
		{ID: "jobs_list", Path: "/vrooli.demo.v1.jobs.JobsService/ListJobs"},
		{ID: "routing_query", Path: "/vrooli.demo.v1.routing.RoutingService/Query"},
		// unbound but omitted -> allowed.
		{ID: "jobs_watch", Path: "/vrooli.demo.v1.jobs.JobsService/WatchJob"},
		// REST endpoint -> ignored by CLI coverage check.
		{
			ID: "health", Path: "/health",
			RESTException: &endpoints.RESTException{Reason: endpoints.RESTReasonOpsProbe},
		},
	}
	if err := validateCLICoverage(eps, mf); err != nil {
		t.Fatalf("validateCLICoverage(ok) = %v", err)
	}
}

func TestValidateCLICoverage_Errors(t *testing.T) {
	mf := loadSample(t)
	cases := []struct {
		name string
		eps  []endpoints.EndpointDescriptor
		want string
	}{
		{
			name: "unbound not omitted",
			eps: []endpoints.EndpointDescriptor{
				{ID: "jobs_get", Path: "/vrooli.demo.v1.jobs.JobsService/GetJob"},
				{ID: "jobs_list", Path: "/vrooli.demo.v1.jobs.JobsService/ListJobs"},
				{ID: "routing_query", Path: "/vrooli.demo.v1.routing.RoutingService/Query"},
				{ID: "ghost", Path: "/vrooli.demo.v1.jobs.JobsService/SecretJob"},
			},
			want: "no binding and no omission",
		},
		{
			name: "binding without endpoint",
			eps: []endpoints.EndpointDescriptor{
				{ID: "jobs_get", Path: "/vrooli.demo.v1.jobs.JobsService/GetJob"},
				// ListJobs + Query bindings have no endpoint here.
			},
			want: "no API endpoint exposes",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCLICoverage(c.eps, mf)
			if err == nil {
				t.Fatalf("validateCLICoverage(%s) = nil, want error containing %q", c.name, c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("validateCLICoverage(%s) error = %q, want substring %q", c.name, err.Error(), c.want)
			}
		})
	}
}

func TestGenerate_OutputStability(t *testing.T) {
	mfPath := writeManifest(t, sampleManifest)
	eps := []endpoints.EndpointDescriptor{
		{
			ID: "jobs_get", Path: "/vrooli.demo.v1.jobs.JobsService/GetJob", Method: "POST",
			Summary: "Get a job", Description: "Get <one> job & return it", Category: "jobs",
		},
		{
			ID: "jobs_list", Path: "/vrooli.demo.v1.jobs.JobsService/ListJobs", Method: "POST",
			Summary: "List jobs", Description: "List jobs", Category: "jobs",
		},
		{
			ID: "routing_query", Path: "/vrooli.demo.v1.routing.RoutingService/Query", Method: "POST",
			Summary: "Query", Description: "Query", Category: "routing",
		},
		{
			ID: "jobs_watch", Path: "/vrooli.demo.v1.jobs.JobsService/WatchJob", Method: "POST",
			Summary: "Watch", Description: "Watch", Category: "jobs",
		},
		{
			ID: "health", Path: "/health", Method: "GET", Summary: "Health", Description: "Health", Category: "ops",
			RESTException: &endpoints.RESTException{
				Reason: endpoints.RESTReasonOpsProbe, Note: "probe",
				ProtoPayloads: &endpoints.RESTProtoPayloads{
					Request:  endpoints.RESTPayload{Transport: "none", Conformance: "none"},
					Response: endpoints.RESTPayload{ProtoFullName: "vrooli.demo.v1.Health", Transport: "json", Conformance: "protojson"},
					Error:    endpoints.RESTPayload{Transport: "none", Conformance: "none"},
				},
			},
		},
	}
	out := filepath.Join(t.TempDir(), "endpoints.json")
	if err := Generate(eps, mfPath, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)

	if !strings.HasSuffix(s, "}\n") {
		t.Errorf("output must end with a trailing newline after the closing brace")
	}
	if strings.Contains(s, "cli_commands") {
		t.Errorf("output must not contain a cli_commands section")
	}
	if strings.Contains(s, "cli_mapping") {
		t.Errorf("output must not contain cli_mapping fields")
	}
	if !strings.Contains(s, `"$schema": "`+endpointsSchemaRef+`"`) {
		t.Errorf("output missing expected $schema reference")
	}
	if !strings.Contains(s, "Get <one> job & return it") {
		t.Errorf("HTML escaping must be disabled: '<', '>' and '&' should be literal")
	}
	if strings.Contains(s, `\u003c`) || strings.Contains(s, `\u003e`) || strings.Contains(s, `\u0026`) {
		t.Errorf("output contains HTML-escaped \\uXXXX sequences; SetEscapeHTML(false) expected")
	}
	if !strings.Contains(s, "\n  \"version\": \"1.0.0\"") {
		t.Errorf("output must use two-space indentation")
	}

	// Idempotent: regenerating yields byte-identical output.
	out2 := filepath.Join(t.TempDir(), "endpoints.json")
	if err := Generate(eps, mfPath, out2); err != nil {
		t.Fatalf("Generate (2nd): %v", err)
	}
	data2, _ := os.ReadFile(out2)
	if string(data2) != s {
		t.Errorf("Generate is not byte-stable across runs")
	}
}
