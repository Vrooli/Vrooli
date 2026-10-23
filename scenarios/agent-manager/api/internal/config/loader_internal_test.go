package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadUsesValidEnvironmentAndKeepsSafeFallbacks(t *testing.T) {
	t.Setenv("API_PORT", " 9090 ")
	t.Setenv("SERVER_READ_TIMEOUT", "12s")
	t.Setenv("DB_MAX_OPEN_CONNS", "41")
	t.Setenv("DB_MAX_IDLE_CONNS", "invalid")
	t.Setenv("POLICY_REQUIRE_SANDBOX", "false")
	t.Setenv("POLICY_REQUIRE_APPROVAL", "not-a-bool")
	t.Setenv("RUNNER_MAX_TURNS", "0")

	got := Load()
	if got.Server.Port != "9090" || got.Server.ReadTimeout != 12*time.Second {
		t.Fatalf("server config = %+v", got.Server)
	}
	if got.Database.MaxOpenConns != 41 || got.Database.MaxIdleConns != 5 {
		t.Fatalf("database config = %+v", got.Database)
	}
	if got.Policy.RequireSandbox || !got.Policy.RequireApproval {
		t.Fatalf("policy defaults = %+v", got.Policy)
	}
	if got.Runners.MaxTurns != 0 {
		t.Fatalf("runner max turns = %d, want explicit zero", got.Runners.MaxTurns)
	}
}

func TestEnvironmentHelpersRejectInvalidValuesAndTrimLists(t *testing.T) {
	t.Setenv("CONFIG_TEST_INT", "not-an-int")
	t.Setenv("CONFIG_TEST_BOOL", "not-a-bool")
	t.Setenv("CONFIG_TEST_DURATION", "bad")
	t.Setenv("CONFIG_TEST_LIST", " one, , two ,  ")
	if got := getEnvInt("CONFIG_TEST_INT"); got != 0 {
		t.Fatalf("invalid integer = %d", got)
	}
	if _, ok := envBoolOpt("CONFIG_TEST_BOOL"); ok {
		t.Fatal("invalid optional boolean must not be accepted")
	}
	if got := getEnvDuration("CONFIG_TEST_DURATION"); got != 0 {
		t.Fatalf("invalid duration = %s", got)
	}
	got := envStringList("CONFIG_TEST_LIST")
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("parsed list = %#v", got)
	}
}

func TestApplyEnvOverridesPreservesZeroOnlySettingsAndCompatibilityPrecedence(t *testing.T) {
	levers := DefaultLevers()
	t.Setenv("AGENT_MANAGER_EXECUTION_DEFAULT_TIMEOUT", "2m")
	t.Setenv("AGENT_MANAGER_EXECUTION_DEFAULT_MAX_TURNS", "12")
	t.Setenv("AGENT_MANAGER_SAFETY_REQUIRE_SANDBOX", "false")
	t.Setenv("AGENT_MANAGER_SAFETY_DENY_PATH_PATTERNS", " secrets/**, .env* ")
	t.Setenv("AGENT_MANAGER_CONCURRENCY_QUEUE_WAIT_TIMEOUT", "0s")
	t.Setenv("AGENT_MANAGER_APPROVAL_ALLOW_PARTIAL", "false")
	t.Setenv("AGENT_MANAGER_RUNNERS_STARTUP_GRACE_PERIOD", "0s")
	t.Setenv("AGENT_MANAGER_SERVER_PORT", "7777")
	t.Setenv("API_PORT", "8888")
	t.Setenv("AGENT_MANAGER_STORAGE_DATABASE_URL", "postgres://configured")
	t.Setenv("DATABASE_URL", "postgres://legacy-wins")

	applyEnvOverrides(&levers)
	if levers.Execution.DefaultTimeout != 2*time.Minute || levers.Execution.DefaultMaxTurns != 12 {
		t.Fatalf("execution overrides = %+v", levers.Execution)
	}
	if levers.Safety.RequireSandboxByDefault || len(levers.Safety.DenyPathPatterns) != 2 {
		t.Fatalf("safety overrides = %+v", levers.Safety)
	}
	if levers.Concurrency.QueueWaitTimeout != 0 || levers.Approval.AllowPartialApproval || levers.Runners.StartupGracePeriod != 0 {
		t.Fatalf("zero-capable overrides were not retained: concurrency=%+v approval=%+v runners=%+v", levers.Concurrency, levers.Approval, levers.Runners)
	}
	if levers.Server.Port != "8888" || levers.Storage.DatabaseURL != "postgres://legacy-wins" {
		t.Fatalf("compatibility precedence = server=%q database=%q", levers.Server.Port, levers.Storage.DatabaseURL)
	}
}

// The transcript-import interval keeps its historical env name and the new
// scan-roots list is applied additively; unset env leaves the defaults (6h, no
// extra roots).
func TestApplyEnvOverridesTranscriptImport(t *testing.T) {
	t.Run("overrides from env", func(t *testing.T) {
		levers := DefaultLevers()
		t.Setenv("AGENT_MANAGER_TRANSCRIPT_IMPORT_INTERVAL", "15m")
		t.Setenv("AGENT_MANAGER_TRANSCRIPT_IMPORT_SCAN_ROOTS", " /srv/remote-transcripts , ~/extra/sessions ,  ")
		applyEnvOverrides(&levers)
		if levers.TranscriptImport.Interval != 15*time.Minute {
			t.Fatalf("interval = %v, want 15m", levers.TranscriptImport.Interval)
		}
		if got := levers.TranscriptImport.ScanRoots; len(got) != 2 || got[0] != "/srv/remote-transcripts" || got[1] != "~/extra/sessions" {
			t.Fatalf("scan roots = %#v", got)
		}
	})
	t.Run("unset env keeps defaults", func(t *testing.T) {
		levers := DefaultLevers()
		applyEnvOverrides(&levers)
		if levers.TranscriptImport.Interval != 6*time.Hour {
			t.Fatalf("interval = %v, want 6h default", levers.TranscriptImport.Interval)
		}
		if len(levers.TranscriptImport.ScanRoots) != 0 {
			t.Fatalf("scan roots = %#v, want none by default", levers.TranscriptImport.ScanRoots)
		}
	})
	t.Run("invalid interval is ignored (safe fallback)", func(t *testing.T) {
		levers := DefaultLevers()
		t.Setenv("AGENT_MANAGER_TRANSCRIPT_IMPORT_INTERVAL", "not-a-duration")
		applyEnvOverrides(&levers)
		if levers.TranscriptImport.Interval != 6*time.Hour {
			t.Fatalf("interval = %v, want 6h after invalid env", levers.TranscriptImport.Interval)
		}
	})
}

func TestTranscriptImportLeversValidate(t *testing.T) {
	ok := TranscriptImportLevers{Interval: 15 * time.Minute, ScanRoots: []string{"/srv/x"}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid levers rejected: %v", err)
	}
	tooFast := TranscriptImportLevers{Interval: time.Second}
	if err := tooFast.Validate(); err == nil {
		t.Fatal("interval below 1m must be rejected")
	}
	emptyRoot := TranscriptImportLevers{Interval: time.Hour, ScanRoots: []string{"  "}}
	if err := emptyRoot.Validate(); err == nil {
		t.Fatal("empty scan root must be rejected")
	}
}

func TestLeversForProfileChangesOnlyDocumentedOperationalDefaults(t *testing.T) {
	development := LeversForProfile(ProfileDevelopment)
	testingProfile := LeversForProfile(ProfileTesting)
	production := LeversForProfile(ProfileProduction)
	defaults := DefaultLevers()
	if development.Execution.DefaultTimeout != 10*time.Minute || development.Concurrency.MaxConcurrentRuns != 3 || development.Storage.EventRetentionDays != 7 {
		t.Fatalf("development profile = %+v", development)
	}
	if testingProfile.Execution.DefaultTimeout != time.Minute || testingProfile.Execution.DefaultMaxTurns != 10 || testingProfile.Concurrency.QueueWaitTimeout != 10*time.Second || testingProfile.Server.ReadTimeout != 5*time.Second {
		t.Fatalf("testing profile = %+v", testingProfile)
	}
	if !reflect.DeepEqual(production, defaults) {
		t.Fatalf("production profile must retain defaults")
	}
}
