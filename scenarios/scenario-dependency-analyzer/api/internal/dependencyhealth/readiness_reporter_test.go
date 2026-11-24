package dependencyhealth

import (
	"context"
	"testing"

	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

func TestReadinessReporterDefersLocalDiscoveryUntilReport(t *testing.T) {
	prefix := "SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS"
	for key, value := range map[string]string{
		"_TOKEN": "token", "_POLICY_VERSION": "1", "_SCENARIO": "scenario-dependency-analyzer",
		"_PROFILE_ID": "profile", "_CANDIDATE_COMMIT": "commit", "_ARTIFACT_DIGEST": "digest",
		"_TARGETS": "linux", "_CHANNEL": "local", "_URL": "",
	} {
		t.Setenv(prefix+key, value)
	}
	reporter, ok := newReadinessReporter(context.Background()).(*readinessreporter.Reporter)
	if !ok || reporter == nil {
		t.Fatalf("reporter = %T; want resolver-backed readiness reporter without a startup lookup", reporter)
	}
}
