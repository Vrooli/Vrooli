package maintenance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditCronReportsBrokenDeclarationAndUndeclaredRepositoryEntry(t *testing.T) {
	root := t.TempDir()
	installedTarget := filepath.Join(root, "tools", "declared")
	if err := os.MkdirAll(filepath.Dir(installedTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installedTarget, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	declarations := []CronDeclaration{
		{Name: "installed", Schedule: "@daily", Target: "tools/declared"},
		{Name: "broken", Schedule: "*/5 * * * *", Target: "tools/missing"},
	}
	crontab := "# preserved comment\nMAILTO=ops@example.test\n@daily \"" + installedTarget + "\" --check\n0 1 * * * " + filepath.Join(root, "tools", "undeclared") + " >/dev/null\n"

	report, err := AuditCron(root, declarations, crontab)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != CronStatusIssuesFound {
		t.Fatalf("status = %q, want %q", report.Status, CronStatusIssuesFound)
	}
	assertCronFinding(t, report.Findings, CronFindingDeclaredTargetMissing, "broken")
	assertCronFinding(t, report.Findings, CronFindingDeclaredEntryMissing, "broken")
	assertCronFinding(t, report.Findings, CronFindingUndeclaredRepository, "")
	if len(report.Entries) != 2 {
		t.Fatalf("entries = %+v, want two executable entries", report.Entries)
	}
}

func TestAuditCronAcceptsDeclaredEntryAndDoesNotMatchRootPrefixCollision(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "jobs", "health")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	declarations := []CronDeclaration{{Name: "health", Schedule: "*/5 * * * *", Target: "jobs/health"}}
	crontab := "*/5 * * * * '" + target + "'\n0 0 * * * " + root + "-other/job\n"

	report, err := AuditCron(root, declarations, crontab)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != CronStatusPassed || len(report.Findings) != 0 {
		t.Fatalf("report = %+v, want clean", report)
	}
}

func TestValidateCronDeclarationsRejectsNonPortableTargets(t *testing.T) {
	for _, target := range []string{"/absolute/job", "../outside", "jobs/../../outside"} {
		err := validateCronDeclarations([]CronDeclaration{{Name: "bad", Schedule: "@daily", Target: target}})
		if err == nil {
			t.Fatalf("target %q unexpectedly accepted", target)
		}
	}
}

func assertCronFinding(t *testing.T, findings []CronFinding, code, declaration string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code && finding.Declaration == declaration {
			return
		}
	}
	t.Fatalf("missing finding code=%q declaration=%q in %+v", code, declaration, findings)
}
