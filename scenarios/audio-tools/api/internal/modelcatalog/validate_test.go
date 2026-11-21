package modelcatalog

import "testing"

func TestValidationRejectsDuplicateIDs(t *testing.T) {
	first := validModel("duplicate", CapabilitySTT)
	second := validModel("duplicate", CapabilitySTT)
	issues := New(fixture(first, second)).ValidationIssues()
	if !hasIssue(issues, IssueDuplicateID, "duplicate") {
		t.Fatalf("expected duplicate_id, got %v", issues)
	}
}

func TestValidationRequiresLicenseNotesForNonCommercial(t *testing.T) {
	m := validModel("nc", CapabilityTTS)
	m.Commercial = CommercialNo
	m.LicenseNotes = ""
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueMissingLicenseNotes, "nc") {
		t.Fatal("expected missing_license_notes for commercial_use=no")
	}

	unknown := validModel("unknown-licence", CapabilityTTS)
	unknown.Commercial = CommercialUnknown
	unknown.LicenseNotes = ""
	if !hasIssue(New(fixture(unknown)).ValidationIssues(), IssueMissingLicenseNotes, "unknown-licence") {
		t.Fatal("expected missing_license_notes for commercial_use=unknown")
	}
}

func TestValidationRejectsNonAvailableDefaultWithoutOptIn(t *testing.T) {
	m := validModel("experimental-default", CapabilitySTT)
	m.Readiness = ReadinessExperimental
	m.Roles = []string{"default"}
	m.Notes = "experimental path"

	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidDefault, "experimental-default") {
		t.Fatal("expected invalid_default for an experimental default without allow_default")
	}

	m.AllowDefault = true
	if hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidDefault, "experimental-default") {
		t.Fatal("allow_default should make the default role valid")
	}
}

func TestValidationRejectsEmptyProviderIDs(t *testing.T) {
	m := validModel("no-provider", CapabilityTTS)
	m.ProviderIDs = nil
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueMissingProviderID, "no-provider") {
		t.Fatal("expected missing_provider_id")
	}
}

func TestValidationRejectsUnknownProviderID(t *testing.T) {
	m := validModel("unknown-provider", CapabilityTTS)
	m.ProviderIDs = []string{"retired-provider"}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnknownProviderID, "unknown-provider") {
		t.Fatal("expected unknown_provider_id")
	}
}

func TestValidationRejectsEngineOnNonSTT(t *testing.T) {
	m := validModel("tts-with-engine", CapabilityTTS)
	m.EngineIDs = []string{"whisper-local"}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnexpectedEngineID, "tts-with-engine") {
		t.Fatal("expected unexpected_engine_id")
	}
}

func TestValidationRejectsEmptyLanguages(t *testing.T) {
	m := validModel("no-language", CapabilityTTS)
	m.Languages = nil
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidLanguage, "no-language") {
		t.Fatal("expected invalid_language")
	}
}

func TestValidationRejectsIncoherentHardware(t *testing.T) {
	m := validModel("both-cpu-and-gpu", CapabilityTTS)
	m.Hardware.CPUCapable = true
	m.Hardware.GPURequired = true
	m.Hardware.MinVRAMGB = 8
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidHardware, "both-cpu-and-gpu") {
		t.Fatal("expected invalid_hardware")
	}
}

func TestValidationRejectsStalePolicy(t *testing.T) {
	seed := fixture(validModel("stt", CapabilitySTT))
	seed.SelectionPolicy.RoleOrder = []string{"default", "fast"}
	issues := New(seed).ValidationIssues()
	found := false
	for _, issue := range issues {
		if issue.Code == IssueInvalidPolicy {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_policy, got %v", issues)
	}
}
