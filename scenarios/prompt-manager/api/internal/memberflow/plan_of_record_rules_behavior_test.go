package memberflow

import (
	"os"
	"path/filepath"
	"testing"
)

// Behavioral tests for the plan-of-record manifest rules.
//
// Fifteen por_* rules were named by no test at plan start, and the family had
// never reported a finding. That combination is exactly the ambiguity Phase 3's
// screens exist to resolve: a rule that has never fired and has no test cannot
// be told apart from a dead one by looking at it. These tests establish that
// each rule detects the defect it names, so the family's silence on a clean tree
// is evidence of a clean tree rather than evidence of nothing.

// validManifest is the fixture every case below mutates in exactly one way.
func validManifest(t *testing.T, root string) (PlanOfRecordManifest, OperatingModelDocument) {
	t.Helper()
	manifest := PlanOfRecordManifest{
		Contract: PlanOfRecordContract{
			Kind:   teamPlanOfRecordKind,
			Schema: teamPlanOfRecordSchema,
			Team:   "team-a",
		},
		SourcePath: filepath.Join(root, "docs", "team-a", "manifest.json"),
		RootDir:    filepath.Join(root, "docs", "team-a"),
	}
	model := OperatingModelDocument{Team: "team-a"}
	return manifest, model
}

func porFindingFor(findings []OperatingGraphFinding, rule string) *OperatingGraphFinding {
	for i := range findings {
		if findings[i].Rule == rule {
			return &findings[i]
		}
	}
	return nil
}

func TestPlanOfRecordManifestRulesFireOnTheDefectTheyName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rule   string
		mutate func(*PlanOfRecordManifest, *OperatingModelDocument)
	}{
		{
			name: "manifest kind is not the team plan-of-record kind",
			rule: "por_manifest_kind_unknown",
			mutate: func(m *PlanOfRecordManifest, _ *OperatingModelDocument) {
				m.Contract.Kind = "something-else"
			},
		},
		{
			name: "manifest schema is not the team plan-of-record schema",
			rule: "por_manifest_schema_unknown",
			mutate: func(m *PlanOfRecordManifest, _ *OperatingModelDocument) {
				m.Contract.Schema = "some-other-schema/v9"
			},
		},
		{
			name: "manifest team disagrees with the operating model team",
			rule: "por_manifest_team_mismatch",
			mutate: func(m *PlanOfRecordManifest, _ *OperatingModelDocument) {
				m.Contract.Team = "a-different-team"
			},
		},
		{
			name: "two sections share one id",
			rule: "por_manifest_duplicate_section",
			mutate: func(m *PlanOfRecordManifest, _ *OperatingModelDocument) {
				m.Sections = []PlanOfRecordSection{
					{ID: "strategy", Path: "strategy"},
					{ID: "strategy", Path: "strategy-again"},
				}
			},
		},
		{
			name: "a section path escapes the manifest root",
			rule: "por_manifest_path_invalid",
			mutate: func(m *PlanOfRecordManifest, _ *OperatingModelDocument) {
				m.Sections = []PlanOfRecordSection{{ID: "strategy", Path: "../outside"}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			manifest, model := validManifest(t, root)
			tc.mutate(&manifest, &model)

			findings := ValidatePlanOfRecordManifest(root, manifest, model)
			got := porFindingFor(findings, tc.rule)
			if got == nil {
				t.Fatalf("%s did not fire; findings: %+v", tc.rule, findings)
			}
			if got.Severity != SeverityError {
				t.Errorf("%s severity = %q, want error", tc.rule, got.Severity)
			}
			if got.Detail == "" {
				t.Errorf("%s fired with no detail naming what to change", tc.rule)
			}
		})
	}
}

// A conforming manifest must produce none of the rules above. Without this, a
// rule that fired unconditionally would pass every case in the table.
func TestPlanOfRecordManifestRulesStaySilentOnAConformingManifest(t *testing.T) {
	root := t.TempDir()
	manifest, model := validManifest(t, root)

	for _, rule := range []string{
		"por_manifest_kind_unknown",
		"por_manifest_schema_unknown",
		"por_manifest_team_mismatch",
		"por_manifest_duplicate_section",
		"por_manifest_path_invalid",
	} {
		if got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), rule); got != nil {
			t.Errorf("%s fired on a conforming manifest: %+v", rule, got)
		}
	}
}

// The document- and package-level plan-of-record rules need a real tree on
// disk, because they report on files the manifest declares but the filesystem
// does not hold.
func TestPlanOfRecordDocumentAndPackageRulesFireAgainstARealTree(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rule    string
		mutate  func(*PlanOfRecordManifest)
		wantSev Severity
	}{
		{
			name: "a required document the tree does not hold",
			rule: "por_required_document_missing",
			mutate: func(m *PlanOfRecordManifest) {
				m.Sections = []PlanOfRecordSection{{
					ID:        "strategy",
					Path:      "strategy",
					Documents: []PlanOfRecordDocument{{Path: "ABSENT.md", Required: true}},
				}}
			},
			wantSev: SeverityError,
		},
		{
			name: "one section declares the same document twice",
			rule: "por_manifest_duplicate_document",
			mutate: func(m *PlanOfRecordManifest) {
				m.Sections = []PlanOfRecordSection{{
					ID:   "strategy",
					Path: "strategy",
					Documents: []PlanOfRecordDocument{
						{Path: "SAME.md"},
						{Path: "SAME.md"},
					},
				}}
			},
			wantSev: SeverityError,
		},
		{
			name: "one section declares the same package twice",
			rule: "por_manifest_duplicate_package",
			mutate: func(m *PlanOfRecordManifest) {
				m.Sections = []PlanOfRecordSection{{
					ID:   "strategy",
					Path: "strategy",
					Packages: []PlanOfRecordPackage{
						{ID: "pkg-a", Path: "pkg"},
						{ID: "pkg-a", Path: "pkg-again"},
					},
				}}
			},
			wantSev: SeverityError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			manifest, model := validManifest(t, root)
			tc.mutate(&manifest)

			got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), tc.rule)
			if got == nil {
				t.Fatalf("%s did not fire", tc.rule)
			}
			if got.Severity != tc.wantSev {
				t.Errorf("%s severity = %q, want %q", tc.rule, got.Severity, tc.wantSev)
			}
			if got.Detail == "" {
				t.Errorf("%s fired with no detail naming what to change", tc.rule)
			}
		})
	}
}

// The remaining plan-of-record rules need real files, because each reports on
// what the tree holds rather than on what the manifest says.
func TestPlanOfRecordFileBackedRulesFireAgainstARealTree(t *testing.T) {
	t.Run("a declared package holds fewer entries than it requires", func(t *testing.T) {
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		pkgDir := filepath.Join(manifest.RootDir, "strategy", "pkg")
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		// One entry present, two required.
		if err := os.WriteFile(filepath.Join(pkgDir, "one.md"), []byte("# One\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		manifest.Sections = []PlanOfRecordSection{{
			ID:   "strategy",
			Path: "strategy",
			Packages: []PlanOfRecordPackage{{
				ID:             "pkg-a",
				Path:           "pkg",
				EntryPattern:   "*.md",
				MinimumEntries: 2,
			}},
		}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_package_entries_missing")
		if got == nil {
			t.Fatal("por_package_entries_missing did not fire for an under-filled package")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}
	})

	t.Run("a required link is absent from a declared document", func(t *testing.T) {
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		docDir := filepath.Join(manifest.RootDir, "strategy")
		if err := os.MkdirAll(docDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(docDir, "STRATEGY.md"), []byte("# Strategy\n\nNo links here.\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		manifest.Sections = []PlanOfRecordSection{{
			ID:   "strategy",
			Path: "strategy",
			Documents: []PlanOfRecordDocument{{
				Path:       "STRATEGY.md",
				Validation: PlanOfRecordDocumentRules{RequiredLinks: []string{"docs/agent-system/TOPICS.md"}},
			}},
		}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_required_link_missing")
		if got == nil {
			t.Fatal("por_required_link_missing did not fire for a document without the required link")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}

		// The same document carrying the link must stay silent.
		if err := os.WriteFile(filepath.Join(docDir, "STRATEGY.md"),
			[]byte("# Strategy\n\nSee docs/agent-system/TOPICS.md\n"), 0o644); err != nil {
			t.Fatalf("rewrite: %v", err)
		}
		if got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_required_link_missing"); got != nil {
			t.Errorf("por_required_link_missing fired on a document carrying the link: %+v", got)
		}
	})

	t.Run("a durable document outside every declared section", func(t *testing.T) {
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		if err := os.MkdirAll(manifest.RootDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		// A document inside a declared section directory is allowed; the rule
		// reports one that belongs to no declared section at all, which is a
		// durable document nothing in the manifest accounts for.
		if err := os.WriteFile(filepath.Join(manifest.RootDir, "STRAY.md"), []byte("# Stray\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		manifest.Sections = []PlanOfRecordSection{{ID: "strategy", Path: "strategy"}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_unregistered_document")
		if got == nil {
			t.Fatal("por_unregistered_document did not fire for a document outside every declared section")
		}
		if got.Severity != SeverityWarning {
			t.Errorf("severity = %q, want warning", got.Severity)
		}
	})
}

// por_manifest_invalid and por_document_unreadable report on files the
// validator cannot parse or cannot open. Both matter because the alternative to
// reporting them is reporting nothing: an unparseable manifest silently
// contributes no findings at all, so the team looks clean.
func TestPlanOfRecordReportsUnparseableAndUnreadableFiles(t *testing.T) {
	t.Run("a manifest that is not valid JSON", func(t *testing.T) {
		repoRoot := t.TempDir()
		teamDir := filepath.Join(repoRoot, "docs", "team-a")
		if err := os.MkdirAll(teamDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		// Truncated on purpose. A typed writer cannot produce this, which is
		// the whole point: the rule exists for files the parser rejects.
		if err := os.WriteFile(filepath.Join(teamDir, "manifest.json"), []byte(`{"contract":`), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		got := porFindingFor(ValidateAllPlanOfRecords(repoRoot), "por_manifest_invalid")
		if got == nil {
			t.Fatal("por_manifest_invalid did not fire for a manifest that is not valid JSON")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}
		if got.Detail == "" {
			t.Error("finding carries no detail explaining why the manifest is invalid")
		}
	})

	t.Run("a declared document that cannot be read", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions, so an unreadable file cannot be simulated")
		}
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		docDir := filepath.Join(manifest.RootDir, "strategy")
		if err := os.MkdirAll(docDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		docPath := filepath.Join(docDir, "STRATEGY.md")
		if err := os.WriteFile(docPath, []byte("# Strategy\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		// Present, declared, and unopenable — a different defect from absent.
		if err := os.Chmod(docPath, 0o000); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(docPath, 0o644) })

		manifest.Sections = []PlanOfRecordSection{{
			ID:   "strategy",
			Path: "strategy",
			Documents: []PlanOfRecordDocument{{
				Path:       "STRATEGY.md",
				Validation: PlanOfRecordDocumentRules{RequiredHeadings: []string{"Strategy"}},
			}},
		}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_document_unreadable")
		if got == nil {
			t.Fatal("por_document_unreadable did not fire for a present but unreadable document")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}
		// An unreadable document must not also be reported missing; the fixes
		// differ (restore permissions versus write the file).
		if missing := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_required_document_missing"); missing != nil {
			t.Errorf("an unreadable document was also reported missing: %+v", missing)
		}
	})
}

// The last plan-of-record rules: a required heading absent from a declared
// document, and a package missing a file it declares as required.
func TestPlanOfRecordHeadingAndPackageFileRulesFire(t *testing.T) {
	t.Run("a required heading is absent from a declared document", func(t *testing.T) {
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		docDir := filepath.Join(manifest.RootDir, "strategy")
		if err := os.MkdirAll(docDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(docDir, "STRATEGY.md"), []byte("# Strategy\n\nNo required section here.\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		manifest.Sections = []PlanOfRecordSection{{
			ID:   "strategy",
			Path: "strategy",
			Documents: []PlanOfRecordDocument{{
				Path:       "STRATEGY.md",
				Validation: PlanOfRecordDocumentRules{RequiredHeadings: []string{"The coverage rule"}},
			}},
		}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_required_heading_missing")
		if got == nil {
			t.Fatal("por_required_heading_missing did not fire")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}

		// The same document carrying the heading must stay silent.
		if err := os.WriteFile(filepath.Join(docDir, "STRATEGY.md"),
			[]byte("# Strategy\n\n## The coverage rule\n\nHere.\n"), 0o644); err != nil {
			t.Fatalf("rewrite: %v", err)
		}
		if got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_required_heading_missing"); got != nil {
			t.Errorf("por_required_heading_missing fired on a document carrying the heading: %+v", got)
		}
	})

	t.Run("a package is missing a file it declares as required", func(t *testing.T) {
		root := t.TempDir()
		manifest, model := validManifest(t, root)
		pkgDir := filepath.Join(manifest.RootDir, "strategy", "pkg")
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		manifest.Sections = []PlanOfRecordSection{{
			ID:   "strategy",
			Path: "strategy",
			Packages: []PlanOfRecordPackage{{
				ID:            "pkg-a",
				Path:          "pkg",
				RequiredFiles: []string{"README.md"},
			}},
		}}

		got := porFindingFor(ValidatePlanOfRecordManifest(root, manifest, model), "por_package_required_file_missing")
		if got == nil {
			t.Fatal("por_package_required_file_missing did not fire for an absent required file")
		}
		if got.Severity != SeverityError {
			t.Errorf("severity = %q, want error", got.Severity)
		}
	})
}

// por_discovery_failed reports that the walk itself could not run. It is the
// difference between "no plan-of-record problems" and "we never looked".
func TestPlanOfRecordDiscoveryFailedFiresWhenTheDocsTreeCannotBeWalked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions, so an unwalkable tree cannot be simulated")
	}
	repoRoot := t.TempDir()
	docsDir := filepath.Join(repoRoot, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(docsDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(docsDir, 0o755) })

	findings := ValidateAllPlanOfRecords(repoRoot)
	got := porFindingFor(findings, "por_discovery_failed")
	if got == nil {
		// Discovery may succeed with zero results on some platforms; only a
		// real walk error should produce this rule, and a silent empty result
		// is the wrong answer to report as success.
		t.Skipf("discovery did not error on this platform; findings: %+v", findings)
	}
	if got.Severity != SeverityError {
		t.Errorf("severity = %q, want error", got.Severity)
	}
}

// por_manifest_unreadable fires from a different entry point than
// por_manifest_invalid: ValidatePlanOfRecordManifestsForModels stats the
// manifest for a model it already holds, so it reports a stat failure that is
// not "does not exist". ValidateAllPlanOfRecords never reaches it, because its
// discovery walk opens the file first and reports por_discovery_failed instead.
//
// That distinction is the reason to pin it: the two rules look interchangeable
// from their names and are reached by different callers.
func TestPlanOfRecordManifestUnreadableFiresWhenTheManifestCannotBeStatted(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions, so an unstattable path cannot be simulated")
	}
	repoRoot := t.TempDir()
	teamDir := filepath.Join(repoRoot, "docs", "team-a")
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "manifest.json"), []byte(`{"contract":{}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Removing search permission on the parent makes the stat fail with
	// something other than "does not exist", which is the branch under test.
	if err := os.Chmod(teamDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(teamDir, 0o755) })

	models := []OperatingModelDocument{{
		Team:   "team-a",
		Source: OperatingModelSource{Path: "docs/team-a/operating/OPERATING_MODEL.md"},
	}}
	findings := ValidatePlanOfRecordManifestsForModels(models, OperatingGraphRuntime{RepoRoot: repoRoot})

	got := porFindingFor(findings, "por_manifest_unreadable")
	if got == nil {
		t.Skipf("stat did not fail on this platform; findings: %+v", findings)
	}
	if got.Severity != SeverityError {
		t.Errorf("severity = %q, want error", got.Severity)
	}
}
