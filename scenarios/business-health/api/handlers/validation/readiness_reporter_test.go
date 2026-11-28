package validation

import (
	"testing"

	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

func TestReadinessReporterUsesLazyLocalResolutionWithoutURLOverride(t *testing.T) {
	setReporterIdentity(t, businessHealthReadinessPrefix, "business-health")
	t.Setenv(businessHealthReadinessPrefix+"_URL", "")
	if _, ok := newReadinessReporter(nil).(*readinessreporter.Reporter); !ok {
		t.Fatal("reporter should be constructed without resolving Deployment Manager at startup")
	}
}

func setReporterIdentity(t *testing.T, prefix, scenario string) {
	t.Helper()
	for key, value := range map[string]string{
		"_TOKEN": "token", "_POLICY_VERSION": "1", "_SCENARIO": scenario,
		"_PROFILE_ID": "profile", "_CANDIDATE_COMMIT": "commit", "_ARTIFACT_DIGEST": "digest",
		"_TARGETS": "linux", "_CHANNEL": "local",
	} {
		t.Setenv(prefix+key, value)
	}
}
