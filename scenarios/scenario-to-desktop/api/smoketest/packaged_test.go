package smoketest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-api/procmetrics"
)

func TestMergePackagedProcessRolesReplacesUnavailablePlaceholders(t *testing.T) {
	placeholders := []PackagedProcessRole{
		{Role: string(procmetrics.RoleWebviewRenderer), GenericRole: string(procmetrics.RoleWebviewRenderer), Availability: LaunchAvailability{State: "unavailable", Reason: "pending census"}},
		{Role: string(procmetrics.RoleWebviewGPU), GenericRole: string(procmetrics.RoleWebviewGPU), Availability: LaunchAvailability{State: "unavailable", Reason: "pending census"}},
		{Role: string(procmetrics.RoleWebviewOther), GenericRole: string(procmetrics.RoleWebviewOther), Availability: LaunchAvailability{State: "unavailable", Reason: "pending census"}},
	}
	tree := &procmetrics.ProcessTreeReport{
		Supported: true,
		Roles: map[procmetrics.ProcessRole]procmetrics.RoleSummary{
			procmetrics.RoleWebviewRenderer: {Role: procmetrics.RoleWebviewRenderer, ProcessCount: 2, MemoryStatsAvailable: true, RSSBytes: 8 * 1048576, PSSBytes: 6 * 1048576, USSBytes: 4 * 1048576},
			procmetrics.RoleWebviewGPU:      {Role: procmetrics.RoleWebviewGPU},
			procmetrics.RoleWebviewOther:    {Role: procmetrics.RoleWebviewOther},
		},
	}
	merged := mergePackagedProcessRoles(placeholders, tree)
	if len(merged) != 3 {
		t.Fatalf("merged process roles = %d, want 3: %+v", len(merged), merged)
	}
	for _, role := range merged {
		if role.Role == string(procmetrics.RoleWebviewRenderer) {
			if role.ProcessCount != 2 || role.Availability.State != "available" || role.GenericRole != string(procmetrics.RoleWebviewRenderer) {
				t.Fatalf("renderer role = %+v", role)
			}
		}
		if role.Availability.State != "available" {
			t.Errorf("supported census left %s unavailable: %+v", role.Role, role)
		}
	}
}

func TestPackagedLaunchPhasesRequireAvailableMarks(t *testing.T) {
	sample := PackagedLaunchSample{
		SchemaVersion: PackagedLaunchSchemaVersion,
		SampleID:      "capture-1",
		CapturedAt:    time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		Marks: []PackagedLaunchMark{
			{Name: "process_spawn", ElapsedNs: 0, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
			{Name: "first_window", ElapsedNs: 300_000_000, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
			{Name: "first_contentful_paint", ElapsedNs: 400_000_000, Availability: LaunchAvailability{State: "available", Reason: "renderer observation"}},
			{Name: "app_interactive", ElapsedNs: 900_000_000, Availability: LaunchAvailability{State: "available", Reason: "declared mark"}},
			{Name: "runtime_spawn", ElapsedNs: 500_000_000, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
			{Name: "runtime_ready", ElapsedNs: 700_000_000, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
			{Name: "backend_ready", ElapsedNs: 700_000_000, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
		},
	}
	if err := sample.Validate(); err != nil {
		t.Fatal(err)
	}
	phases := PackagedLaunchPhases(sample)
	if !phases[0].Available || phases[0].DurationMs != 300 {
		t.Fatalf("first window phase = %+v", phases[0])
	}
	if !phases[1].Available || phases[1].DurationMs != 500 {
		t.Fatalf("interactive phase = %+v", phases[1])
	}
	if !phases[2].Available || phases[2].DurationMs != 200 {
		t.Fatalf("runtime phase = %+v", phases[2])
	}
	if !phases[3].Available || phases[3].DurationMs != 200 {
		t.Fatalf("backend phase = %+v", phases[3])
	}

	sample.Marks[3].Availability = LaunchAvailability{State: "unavailable", Reason: "renderer closed early"}
	if phases = PackagedLaunchPhases(sample); phases[1].Available || phases[1].Reason != "renderer closed early" {
		t.Fatalf("unavailable interactive mark was presented as measured: %+v", phases[1])
	}
}

func TestLaunchPhaseDurationsReadPackagedLaunchJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.json")
	sample := PackagedLaunchSample{
		SchemaVersion: PackagedLaunchSchemaVersion,
		SampleID:      "capture-1",
		CapturedAt:    time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		Marks: []PackagedLaunchMark{
			{Name: "process_spawn", Availability: LaunchAvailability{State: "available", Reason: "measured"}},
			{Name: "first_window", ElapsedNs: 20_000_000, Availability: LaunchAvailability{State: "available", Reason: "measured"}},
		},
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	phases, err := launchPhaseDurations(path, LaunchRunDemo)
	if err != nil {
		t.Fatal(err)
	}
	if !phases[0].Available || phases[0].DurationMs != 20 {
		t.Fatalf("packaged sample phase = %+v", phases[0])
	}
}

func TestMeasuredUserDataIsFreshForColdAndReusedForWarmRelaunch(t *testing.T) {
	cold := &Status{Condition: "cold"}
	if protocol, demo := smokeUserDataDirPath("run", "protocol", cold), smokeUserDataDirPath("run", "demo", cold); protocol == demo {
		t.Fatalf("cold protocol and measured launch share user data: %q", protocol)
	}
	warm := &Status{Condition: "warm"}
	if protocol, demo := smokeUserDataDirPath("run", "protocol", warm), smokeUserDataDirPath("run", "demo", warm); protocol != demo {
		t.Fatalf("warm relaunch uses different user data: %q != %q", protocol, demo)
	}
}

func TestPackagedLaunchSampleRetainsJourneyEvidence(t *testing.T) {
	sample := PackagedLaunchSample{SchemaVersion: PackagedLaunchSchemaVersion, SampleID: "sample-journey", Strategy: "electron-sidecar", CapturedAt: time.Now(), JourneyEvidence: &PackagedJourneyEvidence{JourneyID: "hello-desktop-main", CaptureID: "capture-1", Disposition: "pass"}}
	if err := sample.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PackagedLaunchSample
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.JourneyEvidence == nil || decoded.JourneyEvidence.JourneyID != "hello-desktop-main" || decoded.JourneyEvidence.CaptureID != "capture-1" || decoded.JourneyEvidence.Disposition != "pass" {
		t.Fatalf("journey evidence lost across packaged sample JSON: %+v", decoded.JourneyEvidence)
	}
}

func TestMeasuredLaunchAttachesObservedJourneyReceipt(t *testing.T) {
	sample := &PackagedLaunchSample{}
	status := &Status{JourneyCaptureID: "capture-2", JourneyDisposition: "failed", JourneyDegradedReason: "assertion_failed"}
	attachMeasuredJourneyEvidence(sample, "hello-desktop-main", status)
	if sample.JourneyEvidence == nil || sample.JourneyEvidence.JourneyID != "hello-desktop-main" || sample.JourneyEvidence.CaptureID != "capture-2" || sample.JourneyEvidence.Disposition != "failed" || sample.JourneyEvidence.DegradedReason != "assertion_failed" {
		t.Fatalf("journey receipt was not attached from smoke status: %+v", sample.JourneyEvidence)
	}
}

func TestPackagedLaunchRejectsLeakingProcessLifetimeEvidence(t *testing.T) {
	sample := PackagedLaunchSample{
		SchemaVersion: PackagedLaunchSchemaVersion,
		SampleID:      "sample-process-lifetime",
		CapturedAt:    time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		ProcessLifetime: []PackagedProcessLifetimeCheck{{
			Stage: "demo", SessionID: "launch-1", Passed: false,
			Remaining: []int{1234}, Assertion: "post-launch descendant assertion failed",
		}},
	}
	if err := sample.Validate(); err == nil {
		t.Fatal("Validate accepted a packaged launch with a surviving descendant")
	}
}

func TestPackagedLaunchRetainsApplicationFailureAlongsidePassingLifetimeEvidence(t *testing.T) {
	sample := PackagedLaunchSample{
		SchemaVersion:   PackagedLaunchSchemaVersion,
		SampleID:        "sample-journey-failed",
		CapturedAt:      time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		SmokeTestStatus: "failed",
		SmokeTestError:  "desktop evidence journey failed",
		ProcessLifetime: []PackagedProcessLifetimeCheck{{Stage: "demo", SessionID: "launch-3", Passed: true}},
	}
	if err := sample.Validate(); err != nil {
		t.Fatalf("Validate rejected an honest capture with a separate application failure: %v", err)
	}
}

func TestProcessLifetimeCheckPassesWhenAppExitsWithErrorButNoDescendantsSurvive(t *testing.T) {
	result := &ExecutionResult{ProcessSessionID: "launch-2"}
	check := processLifetimeCheck("protocol", result, fmt.Errorf("application exited with status 1"))
	if !check.Passed || check.SessionID != "launch-2" || len(check.Remaining) != 0 {
		t.Fatalf("process lifetime check = %+v, want pass based on zero surviving descendants", check)
	}
}
