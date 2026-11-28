package resourcecli

import "testing"

func TestParseWordPressRequestBoundsActionsAndIdentifiers(t *testing.T) {
	if _, err := ParseWordPressRequest(WordPressStart, nil); err == nil {
		t.Fatal("start accepted missing site ID")
	}
	if _, err := ParseWordPressRequest(WordPressApplyRecipe, []string{"--site-id", "demo"}); err == nil {
		t.Fatal("apply-recipe accepted missing recipe ID")
	}
	request, err := ParseWordPressRequest(WordPressApplyRecipe, []string{"--site-id", "demo", "--recipe-id", "forms-contact-form-7", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if request.SiteID != "demo" || request.RecipeID != "forms-contact-form-7" || !request.JSON {
		t.Fatalf("unexpected request: %+v", request)
	}
	if _, err := ParseWordPressRequest(WordPressAcquire, []string{"--site-id", "../outside"}); err == nil {
		t.Fatal("acquire accepted unsupported extra options")
	}
	grade, err := ParseWordPressRequest(WordPressGradeGround, []string{"--json"})
	if err != nil || !grade.JSON || grade.SiteID != "" {
		t.Fatalf("grade-ground should run without mutating a site: request=%+v err=%v", grade, err)
	}
	content, err := ParseWordPressRequest(WordPressVerifyContent, []string{"--site-id", "demo", "--json"})
	if err != nil || content.SiteID != "demo" || !content.JSON {
		t.Fatalf("could not parse content-ground request: request=%+v err=%v", content, err)
	}
	webhook, err := ParseWordPressRequest(WordPressFormWebhook, []string{"--site-id", "demo", "--enabled", "false"})
	if err != nil || webhook.WebhookEnabled {
		t.Fatalf("could not parse explicit webhook disable request: request=%+v err=%v", webhook, err)
	}
	if _, err := ParseWordPressRequest(WordPressFormWebhook, []string{"--site-id", "demo"}); err == nil {
		t.Fatal("form-webhook accepted an implicit enable/disable value")
	}
	migration, err := ParseWordPressRequest(WordPressMigrateContent, []string{"--source-site-id", "source", "--target-site-id", "target", "--json"})
	if err != nil || migration.SourceSiteID != "source" || migration.TargetSiteID != "target" || !migration.JSON {
		t.Fatalf("could not parse safe two-site migration: request=%+v err=%v", migration, err)
	}
	plan, err := ParseWordPressRequest(WordPressPlanContentMigration, []string{"--source-site-id", "source", "--target-site-id", "target"})
	if err != nil || plan.SourceSiteID != "source" || plan.TargetSiteID != "target" {
		t.Fatalf("could not parse read-only migration plan: request=%+v err=%v", plan, err)
	}
	export, err := ParseWordPressRequest(WordPressExportContent, []string{"--site-id", "source", "--json"})
	if err != nil || export.SiteID != "source" || !export.JSON {
		t.Fatalf("could not parse versioned content export request: request=%+v err=%v", export, err)
	}
	for _, args := range [][]string{
		{"--source-site-id", "same", "--target-site-id", "same"},
		{"--source-site-id", "../outside", "--target-site-id", "target"},
	} {
		if _, err := ParseWordPressRequest(WordPressMigrateContent, args); err == nil {
			t.Errorf("accepted unsafe or ambiguous migration args %v", args)
		}
	}
	metric, err := ParseWordPressRequest(WordPressUpdateContentMetric, []string{"--site-id", "source", "--slug", "measured-onboarding", "--value", "25"})
	if err != nil || metric.ContentMetric != 25 || metric.ContentSlug != "measured-onboarding" {
		t.Fatalf("could not parse bounded content metric update: request=%+v err=%v", metric, err)
	}
	if _, err := ParseWordPressRequest(WordPressUpdateContentMetric, []string{"--site-id", "source", "--slug", "../../outside", "--value", "25"}); err == nil {
		t.Fatal("accepted traversal post slug")
	}
	if _, err := ParseWordPressRequest(WordPressUpdateContentMetric, []string{"--site-id", "source", "--slug", "measured-onboarding", "--value", "-1"}); err == nil {
		t.Fatal("accepted negative content metric")
	}
	term, err := ParseWordPressRequest(WordPressSetContentTerm, []string{"--site-id", "source", "--slug", "measured-onboarding", "--taxonomy", "category", "--term", "Taxonomy Review", "--present", "true", "--json"})
	if err != nil || term.SiteID != "source" || term.Taxonomy != "category" || term.TermName != "Taxonomy Review" || !term.TermPresent || !term.JSON {
		t.Fatalf("could not parse bounded taxonomy update: request=%+v err=%v", term, err)
	}
	if _, err := ParseWordPressRequest(WordPressSetContentTerm, []string{"--site-id", "source", "--slug", "measured-onboarding", "--taxonomy", "custom", "--term", "Tag", "--present", "true"}); err == nil {
		t.Fatal("accepted unsupported taxonomy")
	}
	if _, err := ParseWordPressRequest(WordPressSetContentTerm, []string{"--site-id", "source", "--slug", "measured-onboarding", "--taxonomy", "tag", "--term", "bad/<markup>", "--present", "true"}); err == nil {
		t.Fatal("accepted unsafe taxonomy term")
	}
	audit, err := ParseWordPressRequest(WordPressAuditCoreContent, []string{"--site-id", "source", "--json"})
	if err != nil || audit.SiteID != "source" || !audit.JSON {
		t.Fatalf("could not parse content-audit request: request=%+v err=%v", audit, err)
	}
	auditGround, err := ParseWordPressRequest(WordPressVerifyContentAudit, []string{"--site-id", "source"})
	if err != nil || auditGround.SiteID != "source" {
		t.Fatalf("could not parse live content-audit ground request: request=%+v err=%v", auditGround, err)
	}
}
