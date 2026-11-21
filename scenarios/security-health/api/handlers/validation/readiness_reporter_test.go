package validation

import (
	"context"
	"testing"

	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

func TestReadinessReporterGroupDefersLocalDiscoveryUntilEachReport(t *testing.T) {
	for key, value := range map[string]string{
		"_TOKEN": "token", "_POLICY_VERSION": "1", "_SCENARIO": "security-health",
		"_PROFILE_ID": "profile", "_CANDIDATE_COMMIT": "commit", "_ARTIFACT_DIGEST": "digest",
		"_TARGETS": "linux", "_CHANNEL": "local", "_URL": "",
	} {
		t.Setenv(securityHealthReadinessPrefix+key, value)
	}
	group, ok := newReadinessReporter(context.Background()).(readinessReporterGroup)
	if !ok || len(group) != 2 {
		t.Fatalf("reporter group = %#v; want two resolver-backed reports", group)
	}
	for i, reporter := range group {
		if _, ok := reporter.(*readinessreporter.Reporter); !ok {
			t.Fatalf("reporter %d type = %T; want resolver-backed readiness reporter", i, reporter)
		}
	}
}
