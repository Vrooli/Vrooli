package agentmanager

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultProfileRef_UsesManifestProfileOnly(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	ref, err := svc.profileRefFor("")
	if err != nil {
		t.Fatalf("profileRefFor returned error: %v", err)
	}
	if ref == nil {
		t.Fatal("profileRefFor returned nil for configured service")
	}
	if ref.ProfileKey != "swarm-manager/default" {
		t.Fatalf("expected manifest profile key, got %q", ref.ProfileKey)
	}
	if ref.UpdateExisting || ref.Defaults != nil {
		t.Fatalf("expected run creation to reference the reconciled manifest profile without inline defaults")
	}
}

func TestProfileRefFor_UsesExplicitProfileKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	ref, err := svc.profileRefFor("swarm-manager/deep-work")
	if err != nil {
		t.Fatalf("profileRefFor returned error: %v", err)
	}
	if ref == nil || ref.ProfileKey != "swarm-manager/deep-work" {
		t.Fatalf("profileRefFor explicit key = %+v", ref)
	}
}

func TestProfileRefFor_FailsWhenReconciledProfilesMissingExplicitKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	svc.profileIDs = map[string]string{"swarm-manager/default": "profile-default"}

	_, err := svc.profileRefFor("swarm-manager/deep-work")
	if err == nil {
		t.Fatal("expected missing reconciled profile error")
	}
}

func TestValidateRequiredProfilesAcceptsAllRequiredProfiles(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"swarm-manager/deep-work": "profile-deep-work",
		"swarm-manager/analysis":  "profile-analysis",
	})
	if err != nil {
		t.Fatalf("validateRequiredProfiles returned error: %v", err)
	}
}

func TestValidateRequiredProfilesRejectsMissingDeepWork(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":  "profile-default",
		"swarm-manager/analysis": "profile-analysis",
	})
	if err == nil || !strings.Contains(err.Error(), "swarm-manager/deep-work") {
		t.Fatalf("expected missing deep-work profile error, got %v", err)
	}
}

func TestValidateRequiredProfilesRejectsMissingAnalysis(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"swarm-manager/deep-work": "profile-deep-work",
	})
	if err == nil || !strings.Contains(err.Error(), "swarm-manager/analysis") {
		t.Fatalf("expected missing analysis profile error, got %v", err)
	}
}

func TestValidateRequiredProfilesRejectsNonOwnedProfileKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"other-scenario/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"other-scenario/analysis": "profile-analysis",
	})
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("expected non-owned profile error, got %v", err)
	}
}

func TestTruncateDescription_Short(t *testing.T) {
	desc := "short description"
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("expected unchanged description, got %q", result)
	}
}

func TestTruncateDescription_ExactLimit(t *testing.T) {
	desc := strings.Repeat("a", maxTaskDescriptionLen)
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("expected unchanged description at exact limit, got len=%d", len(result))
	}
}

func TestTruncateDescription_OverLimit(t *testing.T) {
	desc := strings.Repeat("x", maxTaskDescriptionLen+1000)
	result := truncateDescription(desc)
	if len(result) > maxTaskDescriptionLen {
		t.Fatalf("truncated description exceeds limit: len=%d, max=%d", len(result), maxTaskDescriptionLen)
	}
	if !strings.HasSuffix(result, "[truncated — full prompt provided via run request]") {
		t.Fatal("expected truncation suffix")
	}
}

func TestTruncateDescription_LargePrompt(t *testing.T) {
	// A 20KB prompt fits within the 64KB limit.
	desc := strings.Repeat("y", 20195)
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("20KB prompt should pass through unchanged, got len=%d", len(result))
	}
}

func TestTruncateDescription_ExceedsNewLimit(t *testing.T) {
	// Verify truncation still works for prompts exceeding the 64KB limit.
	desc := strings.Repeat("z", maxTaskDescriptionLen+500)
	result := truncateDescription(desc)
	if len(result) > maxTaskDescriptionLen {
		t.Fatalf("prompt exceeding 64KB not truncated: len=%d", len(result))
	}
	if !strings.HasSuffix(result, "[truncated — full prompt provided via run request]") {
		t.Fatal("expected truncation suffix")
	}
}
