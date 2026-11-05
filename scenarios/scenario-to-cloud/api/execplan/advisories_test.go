package execplan

import (
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

// emailSecrets mirrors the landing-page-business-suite manifest that produced
// the live outage: both mail providers are optional, so the plan said nothing
// and the deployment went out unable to send a sign-in email.
func emailSecrets(capability string) *domain.ManifestSecrets {
	return &domain.ManifestSecrets{
		BundleSecrets: []domain.BundleSecretPlan{
			{
				ID:          "sendgrid-api-key",
				Class:       "user_prompt",
				Required:    false,
				Capability:  capability,
				Description: "Restricted SendGrid API key for transactional sign-in email delivery.",
				Target:      domain.BundleSecretTarget{Type: "env", Name: "SENDGRID_API_KEY"},
				Descriptor:  &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "sendgrid-api-key"},
			},
		},
	}
}

func TestCompileWarnsWhenACapabilityBearingSecretIsUnsatisfied(t *testing.T) {
	in := baseInputs()
	in.Manifest.Secrets = emailSecrets("customer sign-in email")

	plan := compile(t, in)

	if len(plan.Advisories) != 1 {
		t.Fatalf("advisories = %+v, want one warning", plan.Advisories)
	}
	advisory := plan.Advisories[0]
	if advisory.Severity != AdvisorySeverityWarning {
		t.Errorf("severity = %q, want warning", advisory.Severity)
	}
	if advisory.Capability != "customer sign-in email" {
		t.Errorf("capability = %q, want the declared capability", advisory.Capability)
	}
	if !strings.Contains(advisory.ID, "sendgrid-api-key") {
		t.Errorf("id = %q, want the secret named", advisory.ID)
	}
	if !strings.Contains(advisory.Hint, "secrets set") {
		t.Errorf("hint = %q, want the command that fixes it", advisory.Hint)
	}
	// The whole point: warn, do not block.
	if plan.Outcome == OutcomeNeedsInput {
		t.Error("an optional secret must not block the plan")
	}
	if plan.Handoff != nil {
		t.Error("an optional secret must not produce an input handoff")
	}
}

func TestCompileIsSilentWhenTheSecretIsSatisfied(t *testing.T) {
	in := baseInputs()
	in.Manifest.Secrets = emailSecrets("customer sign-in email")
	in.Observations.SatisfiedInputs = append(in.Observations.SatisfiedInputs, "vrooli/app:sendgrid-api-key")

	if plan := compile(t, in); len(plan.Advisories) != 0 {
		t.Errorf("advisories = %+v, want none once the secret is satisfied", plan.Advisories)
	}
}

// An undeclared capability keeps the old behavior: optional and silent. This
// keeps the field opt-in, so existing manifests do not start warning.
func TestCompileDoesNotWarnWithoutADeclaredCapability(t *testing.T) {
	in := baseInputs()
	in.Manifest.Secrets = emailSecrets("")

	if plan := compile(t, in); len(plan.Advisories) != 0 {
		t.Errorf("advisories = %+v, want none without a declared capability", plan.Advisories)
	}
}

// A required secret already blocks through the handoff; warning about it too
// would double-report the same condition.
func TestCompileDoesNotWarnAboutRequiredSecrets(t *testing.T) {
	in := baseInputs()
	secrets := emailSecrets("customer sign-in email")
	secrets.BundleSecrets[0].Required = true
	in.Manifest.Secrets = secrets

	plan := compile(t, in)

	if len(plan.Advisories) != 0 {
		t.Errorf("advisories = %+v, want none for a required secret", plan.Advisories)
	}
	if plan.Outcome != OutcomeNeedsInput {
		t.Errorf("outcome = %q, want needs_input for an unsatisfied required secret", plan.Outcome)
	}
}

// Advisories describe the conditions around a plan, not its effects. A
// warning that appears or clears between review and apply must not refuse the
// apply with a digest mismatch.
func TestAdvisoriesAreExcludedFromTheSemanticDigest(t *testing.T) {
	in := baseInputs()
	in.Manifest.Secrets = emailSecrets("customer sign-in email")
	plan := compile(t, in)
	if len(plan.Advisories) == 0 {
		t.Fatal("fixture must produce an advisory")
	}
	withAdvisory := plan.MustDigest()

	// Only the advisories differ; every effect is identical.
	plan.Advisories = nil
	if without := plan.MustDigest(); without != withAdvisory {
		t.Errorf("digest changed with the advisory removed: %s != %s", without, withAdvisory)
	}
}

func TestRenderCarriesAdvisoriesIntoThePreview(t *testing.T) {
	in := baseInputs()
	in.Manifest.Secrets = emailSecrets("card payments")

	preview := Render(compile(t, in))

	if len(preview.Advisories) != 1 || preview.Advisories[0].Capability != "card payments" {
		t.Errorf("preview advisories = %+v, want the capability warning", preview.Advisories)
	}
}
