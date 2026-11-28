package campaigns

import (
	"context"
	"testing"
)

func TestReadinessReporterDoesNotResolvePeerDuringConstruction(t *testing.T) {
	setContentDeskReporterIdentity(t)
	t.Setenv("DEPLOYMENT_MANAGER_API_BASE_URL", "")
	reporter, err := newReadinessReporter(context.Background())
	if err != nil || reporter == nil {
		t.Fatalf("reporter = %v, error = %v; want a lazy resolver-backed reporter", reporter, err)
	}
}

func setContentDeskReporterIdentity(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"_TOKEN": "token", "_POLICY_VERSION": "1", "_SCENARIO": "content-desk",
		"_PROFILE_ID": "profile", "_CANDIDATE_COMMIT": "commit", "_ARTIFACT_DIGEST": "digest",
		"_TARGETS": "linux", "_CHANNEL": "local",
	} {
		t.Setenv(readinessPrefix+key, value)
	}
}
