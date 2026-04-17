package shared

import (
	"encoding/json"
	"scenario-to-cloud/domain"
	"strings"
	"testing"
)

func TestParseManifest(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		dep := &domain.Deployment{Manifest: []byte("{")}
		_, err := ParseManifest(dep)
		if err == nil || !strings.Contains(err.Error(), "failed to parse manifest") {
			t.Fatalf("expected parse error, got %v", err)
		}
	})

	t.Run("missing vps target", func(t *testing.T) {
		m := domain.CloudManifest{
			Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
			Target:   domain.ManifestTarget{Type: "vps"},
		}
		manifestJSON, _ := json.Marshal(m)
		dep := &domain.Deployment{Manifest: manifestJSON}
		_, err := ParseManifest(dep)
		if err == nil || !strings.Contains(err.Error(), "deployment has no VPS target") {
			t.Fatalf("expected missing vps error, got %v", err)
		}
	})

	t.Run("valid manifest", func(t *testing.T) {
		m := domain.CloudManifest{
			Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
			Target: domain.ManifestTarget{
				Type: "vps",
				VPS:  &domain.ManifestVPS{Host: "138.197.95.182"},
			},
		}
		manifestJSON, _ := json.Marshal(m)
		dep := &domain.Deployment{Manifest: manifestJSON}
		got, err := ParseManifest(dep)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Target.VPS == nil || got.Target.VPS.Host != "138.197.95.182" {
			t.Fatalf("unexpected parsed VPS: %+v", got.Target.VPS)
		}
	})
}

func TestGetSSHDetails_DefaultsAndExplicit(t *testing.T) {
	user, port := GetSSHDetails(&domain.ManifestVPS{})
	if user != "root" || port != 22 {
		t.Fatalf("expected defaults root:22, got %s:%d", user, port)
	}

	user, port = GetSSHDetails(&domain.ManifestVPS{User: "ubuntu", Port: 2222})
	if user != "ubuntu" || port != 2222 {
		t.Fatalf("expected explicit ubuntu:2222, got %s:%d", user, port)
	}
}

func TestExtractErrorSummary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := ExtractErrorSummary(&domain.Deployment{}); got != "" {
			t.Fatalf("expected empty summary, got %q", got)
		}
	})

	t.Run("newline clipped", func(t *testing.T) {
		msg := "first line\nsecond line"
		got := ExtractErrorSummary(&domain.Deployment{ErrorMessage: &msg})
		if got != "first line" {
			t.Fatalf("expected first line, got %q", got)
		}
	})

	t.Run("cannot negotiate pattern", func(t *testing.T) {
		msg := `Caddy error: Cannot negotiate ALPN protocol "acme-tls/1"}`
		got := ExtractErrorSummary(&domain.Deployment{ErrorMessage: &msg})
		if !strings.Contains(got, "Cannot negotiate ALPN") {
			t.Fatalf("expected ALPN summary, got %q", got)
		}
	})

	t.Run("long message truncated", func(t *testing.T) {
		msg := strings.Repeat("x", 120)
		got := ExtractErrorSummary(&domain.Deployment{ErrorMessage: &msg})
		if !strings.HasSuffix(got, "...") {
			t.Fatalf("expected truncation suffix, got %q", got)
		}
	})
}

func TestBuildUserNoteAttachment(t *testing.T) {
	if att := BuildUserNoteAttachment(""); att != nil {
		t.Fatal("expected nil attachment for empty note")
	}

	att := BuildUserNoteAttachment("operator context")
	if att == nil {
		t.Fatal("expected attachment")
	}
	if att.Key != "user-note" || att.Content != "operator context" {
		t.Fatalf("unexpected user note attachment: %+v", att)
	}
}

func TestBuildDiagnosticChecklistAttachment_UsesContractNeutralScenarioGuidance(t *testing.T) {
	att := BuildDiagnosticChecklistAttachment()
	if att == nil {
		t.Fatal("expected attachment")
	}
	if strings.Contains(att.Content, "<workdir>/scenarios/<scenario>/.vrooli/service.json") {
		t.Fatalf("diagnostic checklist should not hard-code scenario path layout: %q", att.Content)
	}
	if !strings.Contains(att.Content, "contract-defined scenario root") {
		t.Fatalf("diagnostic checklist should reference contract-defined scenario root: %q", att.Content)
	}
}
