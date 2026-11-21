package reactvitest

import "testing"

func TestRemoveCoverageThresholdsDeletesThePropertyAndItsComment(t *testing.T) {
	config := `export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      // Exclusions cover scaffolding only.
      exclude: ['src/**/*.test.ts'],
      // 85% is the floor every canonical-surface file clears.
      // Raise these together.
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      },
      reportOnFailure: true,
    },
  },
});
`
	want := `export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      // Exclusions cover scaffolding only.
      exclude: ['src/**/*.test.ts'],
      reportOnFailure: true,
    },
  },
});
`
	got, ok := RemoveCoverageThresholds(config)
	if !ok || got != want {
		t.Fatalf("removed=%v\n%s", ok, got)
	}
	if Parse(got, "").HasThresholds {
		t.Fatal("thresholds still parsed after removal")
	}
}

func TestRemoveCoverageThresholdsHandlesLastAndInlineProperties(t *testing.T) {
	cases := []struct{ name, config, want string }{
		{
			name:   "last property without trailing comma",
			config: "export default { test: {\n  coverage: {\n    provider: 'v8',\n    thresholds: { lines: 85, statements: 85 }\n  }\n} }\n",
			want:   "export default { test: {\n  coverage: {\n    provider: 'v8',\n  }\n} }\n",
		},
		{
			name:   "inline between properties",
			config: "export default { test: { coverage: { provider: 'v8', thresholds: { lines: 0 }, reportOnFailure: true } } }",
			want:   "export default { test: { coverage: { provider: 'v8', reportOnFailure: true } } }",
		},
		{
			name:   "non-literal value",
			config: "export default { test: {\n  coverage: {\n    thresholds: COVERAGE.thresholds,\n    provider: 'v8',\n  },\n} }\n",
			want:   "export default { test: {\n  coverage: {\n    provider: 'v8',\n  },\n} }\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RemoveCoverageThresholds(tc.config)
			if !ok || got != tc.want {
				t.Fatalf("removed=%v\n got: %q\nwant: %q", ok, got, tc.want)
			}
		})
	}
}

func TestRemoveCoverageThresholdsIgnoresThresholdsOutsideTestCoverage(t *testing.T) {
	config := "const budgets = { thresholds: { lines: 85 } }\nexport default { test: { coverage: { provider: 'v8' } } }\n"
	if got, ok := RemoveCoverageThresholds(config); ok || got != config {
		t.Fatalf("only test.coverage.thresholds is owned by the gate; got removed=%v %q", ok, got)
	}
}

const sharedNameConfig = `export default defineConfig({
  optimizeDeps: {
    include: ['react', 'zustand'],
  },
  test: {
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'istanbul',
      exclude: ['src/legacy/**',],
    },
  },
});
`

func TestConfigEditsTouchOnlyTheAddressedMember(t *testing.T) {
	got, ok := EnsureCoverageStrings(sharedNameConfig, "exclude", []string{"src/legacy/**", "src/main.tsx"})
	if !ok {
		t.Fatal("coverage.exclude not edited")
	}
	got, _ = EnsureCoverageStrings(got, "include", []string{"src/**/*.{ts,tsx}"})
	got, _ = SetCoverageProperty(got, "provider", "'v8'")
	got, _ = SetCoverageProperty(got, "reportOnFailure", "true")
	got, _ = SetTestProperty(got, "environment", "'jsdom'")
	want := `export default defineConfig({
  optimizeDeps: {
    include: ['react', 'zustand'],
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    coverage: {
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      provider: 'v8',
      exclude: ['src/legacy/**', 'src/main.tsx',],
    },
  },
});
`
	if got != want {
		t.Fatalf("edits escaped their member:\n%s", got)
	}
	projection := Parse(got, "")
	if projection.Environment != "jsdom" || projection.CoverageProvider != "v8" || !projection.ReportOnFailure || !ContainsAllStrings(projection.CoverageExclude, []string{"src/legacy/**", "src/main.tsx"}) {
		t.Fatalf("edited config does not parse as intended: %+v", projection)
	}
}

func TestConfigEditsRefuseToInventMissingBlocks(t *testing.T) {
	config := "export default defineConfig({ plugins: [] })\n"
	if got, ok := SetCoverageProperty(config, "provider", "'v8'"); ok || got != config {
		t.Fatalf("a config without test.coverage must be left alone, got %q", got)
	}
}

func TestConfigEditsSeeThroughRegexLiterals(t *testing.T) {
	config := "export default { test: {\n  server: { deps: { inline: [/@vrooli\\//] } },\n  coverage: {\n    provider: 'v8',\n    thresholds: { lines: 85 },\n  },\n} }\n"
	got, ok := RemoveCoverageThresholds(config)
	want := "export default { test: {\n  server: { deps: { inline: [/@vrooli\\//] } },\n  coverage: {\n    provider: 'v8',\n  },\n} }\n"
	if !ok || got != want {
		t.Fatalf("a regex literal containing // must not hide the rest of the line: removed=%v\n%s", ok, got)
	}
	if Parse(config, "").CoverageProvider != "v8" {
		t.Fatal("projection parsing must also see through regex literals")
	}
}
