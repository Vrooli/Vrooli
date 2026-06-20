package intent

import "testing"

const sample = `{
  "service": {"name": "demo", "displayName": "Demo"},
  "cli": {"enabled": true, "command": "demo"},
  "ports": {"api": {"env_var": "API_PORT", "range": "15000-19999"}, "ui": {"env_var": "UI_PORT", "range": "20000-24999"}},
  "lifecycle": {
    "health": {"endpoints": {"api": "/health"}, "checks": [{"name": "api_endpoint", "type": "http", "target": "http://localhost:${API_PORT}/health", "critical": true}], "startup_grace_period": 15000},
    "setup": {"condition": {"checks": [{"type": "binaries", "targets": ["api/demo-api"]}, {"type": "ui-bundle", "bundle_path": "ui/dist/index.html", "source_dir": "ui/src"}]}, "steps": [{"name": "build-api", "run": "cd api && go build ."}]},
    "develop": {"steps": [{"name": "start-api", "run": "cd api && ./demo-api", "background": true}]}
  },
  "dependencies": {"resources": {"postgres": {"startup_policy": "must_start", "freshness_policy": "reuse_running"}}}
}`

// [REQ:SH-GT-003]
func TestParse(t *testing.T) {
	in, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if in.Name != "demo" || !in.CLIEnabled || in.CLICommand != "demo" {
		t.Fatalf("service/cli fields wrong: %+v", in)
	}
	if len(in.Ports) != 2 || in.Ports["api"].EnvVar != "API_PORT" {
		t.Fatalf("ports wrong: %+v", in.Ports)
	}
	if len(in.Lifecycle.Health.Checks) != 1 || !in.Lifecycle.Health.Checks[0].Critical {
		t.Fatalf("health checks wrong: %+v", in.Lifecycle.Health)
	}
	if in.Lifecycle.Health.StartupGracePeriod != 15000 {
		t.Fatalf("grace period = %d", in.Lifecycle.Health.StartupGracePeriod)
	}
	bins := in.FreshCheckByType("binaries")
	if len(bins) != 1 || len(bins[0].Targets) != 1 {
		t.Fatalf("binaries fresh-check wrong: %+v", bins)
	}
	bundles := in.FreshCheckByType("ui-bundle")
	if len(bundles) != 1 || bundles[0].BundlePath != "ui/dist/index.html" {
		t.Fatalf("ui-bundle fresh-check wrong: %+v", bundles)
	}
	if len(in.Deps.Resources) != 1 || in.Deps.Resources["postgres"].StartupPolicy != "must_start" {
		t.Fatalf("deps wrong: %+v", in.Deps)
	}
	if in.Raw == nil {
		t.Fatal("raw document must be retained")
	}
}
