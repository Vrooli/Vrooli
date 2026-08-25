package runtime

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/vrooli/internal/safeguards"
	"github.com/vrooli/vrooli/internal/tools"
)

// repoRoot returns the repository root derived from this test file's location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

// compileRepoSchema compiles one on-disk schema with every sibling schema
// registered, so cross-file $refs resolve. Schemas carry absolute $id values
// under two different prefixes, so each file is registered under its short
// name and both absolute forms.
func compileRepoSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	schemaDir := filepath.Join(repoRoot(t), ".vrooli", "schemas")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(schemaDir, e.Name()))
		if readErr != nil {
			t.Fatalf("read %s: %v", e.Name(), readErr)
		}
		// Bare filename plus the canonical $id base only. A third alias rooted
		// at https://vrooli.com/ (no /schemas/ segment) used to be registered
		// here, which silently absorbed a generated "../resources.schema.json"
		// ref that resolves one directory above where the file actually lives.
		// That alias made this test pass while every standards-compliant
		// validator — IDEs included — failed to compile service.schema.json.
		// Keep it out so a reintroduced parent-relative ref fails here.
		for _, id := range []string{e.Name(), "https://vrooli.com/schemas/" + e.Name()} {
			if addErr := compiler.AddResource(id, bytes.NewReader(data)); addErr != nil {
				t.Fatalf("add %s as %s: %v", e.Name(), id, addErr)
			}
		}
	}
	schema, err := compiler.Compile(name)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return schema
}

// TestRetentionBudgetShape drives the shared retentionBudget definition
// directly. Every rejection here is a mistake that would otherwise reach the
// engine as a budget that cannot be enforced.
func TestRetentionBudgetShape(t *testing.T) {
	schema := compileRepoSchema(t, "common.schema.json#/definitions/retention")

	sqliteTarget := `"target":{"kind":"sqlite_table","database":"autoheal.sqlite","table":"system_events","time_column":"occurred_at"}`
	dirTarget := `"target":{"kind":"directory","class":"cache","path":"snapshots"}`

	accepted := map[string]string{
		"age-only":        `{"budgets":{"b":{` + sqliteTarget + `,"max_age":"30d"}}}`,
		"bytes-only":      `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"2GiB"}}}`,
		"both-bounds":     `{"budgets":{"b":{` + sqliteTarget + `,"max_age":"30d","max_bytes":"2GiB"}}}`,
		"hours":           `{"budgets":{"b":{` + sqliteTarget + `,"max_age":"72h"}}}`,
		"directory":       `{"budgets":{"b":{` + dirTarget + `,"max_bytes":"5GiB"}}}`,
		"custom-pruner":   `{"budgets":{"b":{` + dirTarget + `,"max_bytes":"5GiB","pruner":"custom"}}}`,
		"with-rationale":  `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"1MiB","rationale":"host-driven ingest"}}}`,
		"multiple-budget": `{"budgets":{"a":{` + sqliteTarget + `,"max_bytes":"1MiB"},"b":{` + dirTarget + `,"max_age":"7d"}}}`,
	}
	for name, raw := range accepted {
		if err := validateAgainst(t, schema, raw); err != nil {
			t.Errorf("%s: expected accept, got %v", name, err)
		}
	}

	rejected := map[string]string{
		// The central gate: a budget with no bound is the autoheal failure mode
		// re-declared as compliant, so it must be unrepresentable.
		"no-bound":             `{"budgets":{"b":{` + sqliteTarget + `}}}`,
		"no-budgets":           `{"budgets":{}}`,
		"missing-budgets":      `{}`,
		"sqlite-no-time":       `{"budgets":{"b":{"target":{"kind":"sqlite_table","database":"a.sqlite","table":"t"},"max_age":"30d"}}}`,
		"sqlite-no-table":      `{"budgets":{"b":{"target":{"kind":"sqlite_table","database":"a.sqlite","time_column":"at"},"max_age":"30d"}}}`,
		"sqlite-no-database":   `{"budgets":{"b":{"target":{"kind":"sqlite_table","table":"t","time_column":"at"},"max_age":"30d"}}}`,
		"directory-no-path":    `{"budgets":{"b":{"target":{"kind":"directory"},"max_bytes":"1GiB"}}}`,
		"unknown-kind":         `{"budgets":{"b":{"target":{"kind":"postgres_table","table":"t"},"max_bytes":"1GiB"}}}`,
		"bytes-without-unit":   `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"2000000"}}}`,
		"bytes-decimal-unit":   `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"2GB"}}}`,
		"age-without-unit":     `{"budgets":{"b":{` + sqliteTarget + `,"max_age":"30"}}}`,
		"age-unsupported-unit": `{"budgets":{"b":{` + sqliteTarget + `,"max_age":"30m"}}}`,
		"unknown-pruner":       `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"1GiB","pruner":"scenario"}}}`,
		"unknown-class":        `{"budgets":{"b":{"target":{"kind":"directory","class":"scratch","path":"x"},"max_bytes":"1GiB"}}}`,
		"unknown-field":        `{"budgets":{"b":{` + sqliteTarget + `,"max_bytes":"1GiB","keep_n":5}}}`,
		"backslash-path":       `{"budgets":{"b":{"target":{"kind":"directory","path":"a\\b"},"max_bytes":"1GiB"}}}`,
	}
	for name, raw := range rejected {
		if err := validateAgainst(t, schema, raw); err == nil {
			t.Errorf("%s: expected rejection, got none", name)
		}
	}
}

// TestRetentionAcceptedOnEveryManifestKind confirms all four manifest schemas
// carry the block, so a scenario, resource, tool, and safeguard can each
// declare a ceiling.
//
// The base document is a real manifest from the repository rather than a
// hand-written minimum, so the assertion is that a budget can be added to
// something an author actually ships, and so the test cannot drift out of sync
// with unrelated required-field changes elsewhere in these schemas.
func TestRetentionAcceptedOnEveryManifestKind(t *testing.T) {
	root := repoRoot(t)
	budget := map[string]any{
		"budgets": map[string]any{
			"events": map[string]any{
				"target":    map[string]any{"kind": "sqlite_table", "database": "a.sqlite", "table": "events", "time_column": "occurred_at"},
				"max_bytes": "2GiB",
			},
		},
	}
	noBound := map[string]any{
		"budgets": map[string]any{
			"events": map[string]any{
				"target": map[string]any{"kind": "sqlite_table", "database": "a.sqlite", "table": "events", "time_column": "occurred_at"},
			},
		},
	}

	cases := []struct {
		schemaName string
		fsys       fs.FS
		fileName   string
	}{
		{"service.schema.json", os.DirFS(filepath.Join(root, "scenarios")), "service.json"},
		{"resource.schema.json", os.DirFS(filepath.Join(root, "resources")), "resource.json"},
		{"tool.schema.json", tools.Manifests, "tool.json"},
		{"safeguard.schema.json", safeguards.Manifests, "safeguard.json"},
	}
	for _, tc := range cases {
		schema := compileRepoSchema(t, tc.schemaName)
		base := firstValidManifest(t, schema, tc.fsys, tc.fileName)

		withBudget := withKey(base, "retention", budget)
		if err := schema.Validate(withBudget); err != nil {
			t.Errorf("%s: expected retention block accepted, got %v", tc.schemaName, err)
		}
		withoutBound := withKey(base, "retention", noBound)
		if err := schema.Validate(withoutBound); err == nil {
			t.Errorf("%s: expected unbounded budget rejected, got none", tc.schemaName)
		}
	}
}

func TestStorageEntryRequiresExplicitRegenerableIntent(t *testing.T) {
	schema := compileRepoSchema(t, "common.schema.json#/definitions/storageEntry")
	if err := schema.Validate(map[string]any{"rung": "owned", "kind": "dir", "class": "data", "regenerable": false}); err != nil {
		t.Fatalf("explicit durable entry rejected: %v", err)
	}
	err := schema.Validate(map[string]any{"rung": "owned", "kind": "dir", "class": "data"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "regenerable") {
		t.Fatalf("missing regenerable error = %v, want field name", err)
	}
}

// firstValidManifest returns a decoded manifest of the given filename that
// already validates against schema, so retention assertions layered on top
// cannot fail for an unrelated pre-existing reason.
func firstValidManifest(t *testing.T, schema *jsonschema.Schema, fsys fs.FS, fileName string) map[string]any {
	t.Helper()
	var found map[string]any
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || found != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == "dist-electron" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != fileName {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return nil
		}
		var payload map[string]any
		if json.Unmarshal(data, &payload) != nil {
			return nil
		}
		if schema.Validate(payload) == nil {
			found = payload
		}
		return nil
	})
	if found == nil {
		t.Fatalf("found no %s in the repository that validates; cannot assert retention is additive", fileName)
	}
	return found
}

// withKey returns a shallow copy of base with key set, leaving base untouched
// so one fixture serves several assertions.
func withKey(base map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[key] = value
	return out
}

// TestRetentionIsAdditiveForExistingManifests confirms the new block changed no
// existing manifest's validity. Some manifests in the repository already fail
// their schema for unrelated pre-existing reasons, so this asserts the precise
// property phase 1 owns: no manifest gains a retention-attributable error.
func TestRetentionIsAdditiveForExistingManifests(t *testing.T) {
	root := repoRoot(t)

	type source struct {
		schema   *jsonschema.Schema
		fsys     fs.FS
		dir      string
		fileName string
	}
	sources := []source{
		{compileRepoSchema(t, "tool.schema.json"), tools.Manifests, ".", "tool.json"},
		{compileRepoSchema(t, "safeguard.schema.json"), safeguards.Manifests, ".", "safeguard.json"},
		{compileRepoSchema(t, "resource.schema.json"), os.DirFS(filepath.Join(root, "resources")), ".", "resource.json"},
		{compileRepoSchema(t, "service.schema.json"), os.DirFS(filepath.Join(root, "scenarios")), ".", "service.json"},
	}

	checked := 0
	for _, src := range sources {
		err := fs.WalkDir(src.fsys, src.dir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				// Build output vendors whole copies of the catalog; validating
				// them measures the build, not the source of truth.
				if d.Name() == "node_modules" || d.Name() == "dist-electron" {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() != src.fileName {
				return nil
			}
			data, readErr := fs.ReadFile(src.fsys, p)
			if readErr != nil {
				t.Fatalf("read %s: %v", p, readErr)
			}
			var payload any
			if jsonErr := json.Unmarshal(data, &payload); jsonErr != nil {
				t.Errorf("%s: invalid JSON: %v", p, jsonErr)
				return nil
			}
			checked++
			if validErr := src.schema.Validate(payload); validErr != nil {
				if strings.Contains(validErr.Error(), "retention") {
					t.Errorf("%s gained a retention-attributable schema error: %v", p, validErr)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s manifests: %v", src.fileName, err)
		}
	}
	if checked == 0 {
		t.Fatal("validated no manifests; the walk found nothing and would pass vacuously")
	}
}

// TestValidateRetentionAgainstDurableData covers the cross-block check: where
// both blocks name the same path they must agree about what that path is.
func TestValidateRetentionAgainstDurableData(t *testing.T) {
	cases := []struct {
		name         string
		manifest     string
		wantConflict bool
	}{
		{
			name:     "agreeing sqlite file",
			manifest: `{"retention":{"budgets":{"events":{"target":{"kind":"sqlite_table","database":"autoheal.sqlite","table":"e","time_column":"at"},"max_bytes":"2GiB"}}},"durable_data":{"entries":{"db":{"path":".vrooli/data/vrooli/vrooli-autoheal/autoheal.sqlite","kind":"file","format":"sqlite","regenerable":false}}}}`,
		},
		{
			name:         "durable_data calls the sqlite target a directory",
			manifest:     `{"retention":{"budgets":{"events":{"target":{"kind":"sqlite_table","database":"autoheal.sqlite","table":"e","time_column":"at"},"max_bytes":"2GiB"}}},"durable_data":{"entries":{"db":{"path":"autoheal.sqlite","kind":"dir","regenerable":false}}}}`,
			wantConflict: true,
		},
		{
			name:         "durable_data declares a conflicting format",
			manifest:     `{"retention":{"budgets":{"events":{"target":{"kind":"sqlite_table","database":"autoheal.sqlite","table":"e","time_column":"at"},"max_bytes":"2GiB"}}},"durable_data":{"entries":{"db":{"path":"autoheal.sqlite","kind":"file","format":"json","regenerable":false}}}}`,
			wantConflict: true,
		},
		{
			name:         "directory target declared as a formatted file",
			manifest:     `{"retention":{"budgets":{"snaps":{"target":{"kind":"directory","path":"snapshots"},"max_bytes":"5GiB"}}},"durable_data":{"entries":{"s":{"path":"snapshots","kind":"file","format":"sqlite","regenerable":false}}}}`,
			wantConflict: true,
		},
		{
			name:     "agreeing directory",
			manifest: `{"retention":{"budgets":{"snaps":{"target":{"kind":"directory","path":"snapshots"},"max_bytes":"5GiB"}}},"durable_data":{"entries":{"s":{"path":"$HOME/x/snapshots","kind":"dir","regenerable":false}}}}`,
		},
		{
			name:     "different paths never conflict",
			manifest: `{"retention":{"budgets":{"events":{"target":{"kind":"sqlite_table","database":"autoheal.sqlite","table":"e","time_column":"at"},"max_bytes":"2GiB"}}},"durable_data":{"entries":{"other":{"path":"secrets.json","kind":"file","format":"json","regenerable":false}}}}`,
		},
		{
			name:     "similar suffix is not a segment suffix",
			manifest: `{"retention":{"budgets":{"events":{"target":{"kind":"sqlite_table","database":"heal.sqlite","table":"e","time_column":"at"},"max_bytes":"2GiB"}}},"durable_data":{"entries":{"other":{"path":"autoheal.sqlite","kind":"dir","regenerable":false}}}}`,
		},
		{
			name:     "retention only",
			manifest: `{"retention":{"budgets":{"events":{"target":{"kind":"directory","path":"logs"},"max_bytes":"1GiB"}}}}`,
		},
		{
			name:     "durable_data only",
			manifest: `{"durable_data":{"entries":{"db":{"path":"a.sqlite","kind":"file","format":"sqlite","regenerable":false}}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conflicts, err := ValidateRetentionAgainstDurableData([]byte(tc.manifest))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantConflict && len(conflicts) == 0 {
				t.Fatal("expected a conflict, got none")
			}
			if !tc.wantConflict && len(conflicts) != 0 {
				t.Fatalf("expected no conflict, got %v", conflicts)
			}
			for _, c := range conflicts {
				if c.Budget == "" || c.Entry == "" || c.Reason == "" {
					t.Errorf("conflict is not actionable without all three names: %+v", c)
				}
			}
		})
	}
}

// TestRepositoryManifestsHaveNoRetentionConflict runs the cross-block check
// over every manifest kind that can carry both blocks, so an author who adds a
// budget next to an existing durable_data entry gets a red build rather than a
// silently wrong declaration.
func TestRepositoryManifestsHaveNoRetentionConflict(t *testing.T) {
	root := repoRoot(t)
	for _, spec := range []struct {
		fsys     fs.FS
		fileName string
	}{
		{tools.Manifests, "tool.json"},
		{safeguards.Manifests, "safeguard.json"},
		{os.DirFS(filepath.Join(root, "resources")), "resource.json"},
		{os.DirFS(filepath.Join(root, "scenarios")), "service.json"},
	} {
		err := fs.WalkDir(spec.fsys, ".", func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if d.Name() == "node_modules" || d.Name() == "dist-electron" {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() != spec.fileName {
				return nil
			}
			data, readErr := fs.ReadFile(spec.fsys, p)
			if readErr != nil {
				t.Fatalf("read %s: %v", p, readErr)
			}
			conflicts, valErr := ValidateRetentionAgainstDurableData(data)
			if valErr != nil {
				return nil // malformed JSON is another test's failure
			}
			for _, c := range conflicts {
				t.Errorf("%s: %s", p, c)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", spec.fileName, err)
		}
	}
}

// TestReferenceDocExamplesValidate compiles every JSON example in the retention
// reference document against the schema it documents.
//
// A reference document whose examples do not validate is worse than no document:
// an author copies one, it fails, and they conclude the feature is broken. This
// keeps the doc honest as the schema evolves.
func TestReferenceDocExamplesValidate(t *testing.T) {
	root := repoRoot(t)
	docPath := filepath.Join(root, "docs", "reference", "storage-retention.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	schema := compileRepoSchema(t, "common.schema.json#/definitions/retention")

	// Examples are written as manifest fragments ("retention": { … }), which is
	// how an author actually pastes them, so they are wrapped before validation.
	blocks := jsonBlocks(string(data))
	if len(blocks) < 2 {
		t.Fatalf("found %d JSON examples in %s, want at least the sqlite_table and directory ones", len(blocks), docPath)
	}

	checked := 0
	for i, block := range blocks {
		var payload map[string]any
		if err := json.Unmarshal([]byte("{"+block+"}"), &payload); err != nil {
			t.Errorf("example %d in %s is not valid JSON: %v", i, docPath, err)
			continue
		}
		retentionBlock, ok := payload["retention"]
		if !ok {
			// Not a retention example (a Go snippet or unrelated JSON).
			continue
		}
		checked++
		if err := schema.Validate(retentionBlock); err != nil {
			t.Errorf("example %d in %s does not validate against the schema it documents: %v", i, docPath, err)
		}
	}
	if checked < 2 {
		t.Fatalf("validated %d retention examples, want at least the sqlite_table and directory ones", checked)
	}
}

// jsonBlocks returns the contents of every ```json fenced block in md.
func jsonBlocks(md string) []string {
	var out []string
	const fence = "```json\n"
	for {
		start := strings.Index(md, fence)
		if start < 0 {
			return out
		}
		md = md[start+len(fence):]
		end := strings.Index(md, "```")
		if end < 0 {
			return out
		}
		out = append(out, md[:end])
		md = md[end+3:]
	}
}
