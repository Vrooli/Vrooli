package manifest_test

import (
	"testing"

	manifest "development-toolchain-validator/internal/manifest"
)

// Evaluator is the per-plan §7.2 decision boundary. The table below
// covers ≥15 cases spanning wildcard, glob, content rules, nested
// paths, and the empty-diff happy case.
func TestEvaluate_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		manifest  manifest.Manifest
		diff      []manifest.DiffFile
		wantKind  manifest.VerdictKind
		wantViols int
		reason    string
	}{
		{
			name:     "empty diff is pass regardless of manifest",
			manifest: manifest.Manifest{AllowedPaths: []string{"src/**"}},
			diff:     nil,
			wantKind: manifest.VerdictPass,
			reason:   "no diff entries → no possible violation",
		},
		{
			name:     "wildcard allows any path",
			manifest: manifest.Manifest{WildcardAllowed: true},
			diff: []manifest.DiffFile{
				{Path: "anywhere/under/the/sun.txt"},
				{Path: "another/path"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "wildcard short-circuits path allow-list",
		},
		{
			name:      "non-wildcard disallows unmatched path",
			manifest:  manifest.Manifest{AllowedPaths: []string{"src/**"}},
			diff:      []manifest.DiffFile{{Path: "secrets/credentials.txt"}},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "secrets/ falls outside src/** allow",
		},
		{
			name:     "single-star glob does not span /",
			manifest: manifest.Manifest{AllowedPaths: []string{"src/*.ts"}},
			diff: []manifest.DiffFile{
				{Path: "src/index.ts"},
				{Path: "src/nested/deep.ts"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "src/*.ts must not match src/nested/deep.ts; only top-level",
		},
		{
			name:     "double-star glob matches nested paths",
			manifest: manifest.Manifest{AllowedPaths: []string{"src/**/*.ts"}},
			diff: []manifest.DiffFile{
				{Path: "src/index.ts"},
				{Path: "src/nested/deep.ts"},
				{Path: "src/a/b/c/d.ts"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "src/**/*.ts spans nested segments",
		},
		{
			name: "content rule must_contain satisfied",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "**/*.go", MustContain: []string{"package main"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "cmd/main.go", Content: "package main\nfunc main() {}\n"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "needle found",
		},
		{
			name: "content rule must_contain missing",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "**/*.go", MustContain: []string{"package main"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "cmd/main.go", Content: "package somethingelse"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "needle missing",
		},
		{
			name: "content rule must_not_contain violated",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "src/**", MustNotContain: []string{"TODO"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "src/index.ts", Content: "// TODO: ship it"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "forbidden substring present",
		},
		{
			name: "must_not_contain absent is pass",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "src/**", MustNotContain: []string{"TODO"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "src/index.ts", Content: "ship it now"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "forbidden substring absent",
		},
		{
			name: "content rule does not apply to non-matching path",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "**/*.go", MustContain: []string{"package main"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "README.md", Content: "no go here"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "rule's path_glob excludes README",
		},
		{
			name: "multiple rules per path each enforced",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "**/*.go", MustContain: []string{"package main"}},
					{PathGlob: "**/*.go", MustNotContain: []string{"panic("}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "main.go", Content: "package main\npanic(\"x\")"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "must_not_contain violated by panic(",
		},
		{
			name: "multiple violations across files accumulate",
			manifest: manifest.Manifest{
				AllowedPaths: []string{"src/**"},
				ContentRules: []manifest.ContentRule{
					{PathGlob: "src/**", MustNotContain: []string{"TODO"}},
				},
			},
			diff: []manifest.DiffFile{
				{Path: "secrets/x"},
				{Path: "src/a.ts", Content: "// TODO"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 2,
			reason:    "one path-violation + one content-violation",
		},
		{
			name: "allowed_paths empty + wildcard true allows everything",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				AllowedPaths:    nil,
			},
			diff:     []manifest.DiffFile{{Path: "anywhere"}},
			wantKind: manifest.VerdictPass,
			reason:   "wildcard overrides empty allowed_paths",
		},
		{
			name: "allowed_paths empty + wildcard false rejects everything",
			manifest: manifest.Manifest{
				WildcardAllowed: false,
				AllowedPaths:    nil,
			},
			diff:      []manifest.DiffFile{{Path: "anything"}},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "no allowed entries, no wildcard → reject",
		},
		{
			name: "multiple allowed_paths globs OR-combined",
			manifest: manifest.Manifest{
				AllowedPaths: []string{"src/**", "docs/**"},
			},
			diff: []manifest.DiffFile{
				{Path: "src/a.ts"},
				{Path: "docs/README.md"},
			},
			wantKind: manifest.VerdictPass,
			reason:   "match either entry counts",
		},
		{
			name: "empty must_contain entry is skipped (no false positive)",
			manifest: manifest.Manifest{
				WildcardAllowed: true,
				ContentRules: []manifest.ContentRule{
					{PathGlob: "**", MustContain: []string{""}},
				},
			},
			diff:     []manifest.DiffFile{{Path: "x", Content: "anything"}},
			wantKind: manifest.VerdictPass,
			reason:   "blank needle must not trigger a violation",
		},
		{
			name: "question mark glob matches single non-/ char",
			manifest: manifest.Manifest{
				AllowedPaths: []string{"src/?.ts"},
			},
			diff: []manifest.DiffFile{
				{Path: "src/a.ts"},
				{Path: "src/ab.ts"},
			},
			wantKind:  manifest.VerdictUnexpectedMutation,
			wantViols: 1,
			reason:    "src/?.ts matches single char, not two",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := manifest.Evaluate(tc.manifest, tc.diff)
			if got.Kind != tc.wantKind {
				t.Errorf("verdict kind = %v, want %v (reason: %s; violations: %+v)", got.Kind, tc.wantKind, tc.reason, got.Violations)
			}
			if tc.wantViols > 0 && len(got.Violations) != tc.wantViols {
				t.Errorf("violations count = %d, want %d (reason: %s; got: %+v)", len(got.Violations), tc.wantViols, tc.reason, got.Violations)
			}
		})
	}
}
