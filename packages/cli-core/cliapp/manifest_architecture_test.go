package cliapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archManifest builds a minimal valid manifest. commandField and topField are
// each either empty or a leading-comma JSON fragment (e.g.
// `, "architecture": {...}`), so the base stays valid JSON when they are empty.
func archManifest(commandField, topField string) []byte {
	return []byte(`{
  "name": "demo",
  "groups": [
    {
      "name": "notes",
      "commands": [
        {
          "name": "list",
          "binding": { "kind": "connect-rpc", "service": "NotesService", "method": "ListNotes" },
          "governance": { "effect": "read", "run_eligible": true }` + commandField + `
        }
      ]
    }
  ]` + topField + `
}`)
}

func TestParseManifest_ArchitecturePrimitive(t *testing.T) {
	raw := archManifest(`, "architecture": { "primitive": "proto_list" }`, "")
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := m.Groups[0].Commands[0].Architecture.CommandArchitecture()
	if got.Primitive != PrimitiveProtoList {
		t.Fatalf("expected proto_list primitive, got %q", got.Primitive)
	}
}

func TestParseManifest_RejectsUnknownPrimitive(t *testing.T) {
	raw := archManifest(`, "architecture": { "primitive": "bogus" }`, "")
	if _, err := ParseManifest(raw); err == nil {
		t.Fatalf("expected parse error for unknown primitive")
	}
}

func TestParseManifest_RejectsExceptionWithoutReason(t *testing.T) {
	raw := archManifest(`, "architecture": { "exception": { "class": "durable_run", "reason": "" } }`, "")
	if _, err := ParseManifest(raw); err == nil {
		t.Fatalf("expected parse error for exception missing reason")
	}
}

// A special-case class (durable_run/upload/streaming/passthrough) must be
// declared as an exception, never as architecture.primitive (plan decision D4).
// ParseManifest rejects it early via CommandArchitecture.Validate.
func TestParseManifest_RejectsSpecialCasePrimitiveDeclaration(t *testing.T) {
	raw := archManifest(`, "architecture": { "primitive": "durable_run" }`, "")
	_, err := ParseManifest(raw)
	if err == nil || !strings.Contains(err.Error(), "must be declared as an exception") {
		t.Fatalf("expected special-case-primitive rejection, got %v", err)
	}
}

func TestParseManifest_TopLevelException(t *testing.T) {
	top := `,
  "exceptions": [
    { "command": "execute", "class": "durable_run", "reason": "server-owned run lifecycle" }
  ]`
	raw := archManifest("", top)
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(m.Exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(m.Exceptions))
	}
	got := m.Exceptions[0].CommandArchitecture()
	if got.Exception != ExceptionDurableRun || got.ExceptionReason == "" {
		t.Fatalf("exception not parsed: %+v", got)
	}
}

func TestParseManifest_RejectsExceptionUnknownClass(t *testing.T) {
	top := `,
  "exceptions": [
    { "command": "execute", "class": "not_a_class", "reason": "x" }
  ]`
	raw := archManifest("", top)
	if _, err := ParseManifest(raw); err == nil {
		t.Fatalf("expected parse error for unknown exception class")
	}
}

func TestParseManifest_LegacyManifestStaysValid(t *testing.T) {
	// No architecture metadata at all — must still parse and classify as zero.
	raw := archManifest("", "")
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("legacy manifest should stay valid: %v", err)
	}
	if !m.Groups[0].Commands[0].Architecture.CommandArchitecture().IsZero() {
		t.Fatalf("missing architecture should classify as zero")
	}
}

// TestSchemaEnumsMatchVocabulary guards against drift between the cli-core
// vocabulary SSOT (architecture.go) and the cli-manifest schema enums. If a new
// primitive/exception class is added to one but not the other, this fails.
func TestSchemaEnumsMatchVocabulary(t *testing.T) {
	repoRoot, err := findRepoRootFromCWD()
	if err != nil {
		t.Skipf("repo root not found (running outside repo): %v", err)
	}
	schemaPath := filepath.Join(repoRoot, ".vrooli", "schemas", "cli-manifest.schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Skipf("schema not readable: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	arch := nested(doc, "$defs", "Architecture", "properties")
	primEnum := stringEnum(t, nested(arch, "primitive")["enum"])
	// The schema's architecture.primitive enum mirrors the DECLARABLE (normal)
	// classes only; special-case classes are declared as exceptions (D4).
	assertSameSet(t, "primitive", primEnum, primitiveClassStrings())

	exc := nested(arch, "exception", "properties")
	classEnum := stringEnum(t, nested(exc, "class")["enum"])
	assertSameSet(t, "exception", classEnum, exceptionClassStrings())
}

func primitiveClassStrings() []string {
	out := make([]string, 0, len(declarablePrimitiveClasses))
	for _, p := range declarablePrimitiveClasses {
		out = append(out, string(p))
	}
	return out
}

func exceptionClassStrings() []string {
	out := make([]string, 0, len(exceptionClasses))
	for _, e := range exceptionClasses {
		out = append(out, string(e))
	}
	return out
}

// --- small test helpers ---

func nested(m map[string]any, keys ...string) map[string]any {
	cur := m
	for _, k := range keys {
		next, ok := cur[k].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		cur = next
	}
	return cur
}

func stringEnum(t *testing.T, v any) []string {
	t.Helper()
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("enum is not an array: %T", v)
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		out = append(out, e.(string))
	}
	return out
}

func assertSameSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	set := func(xs []string) map[string]bool {
		m := map[string]bool{}
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	g, w := set(got), set(want)
	for x := range w {
		if !g[x] {
			t.Errorf("%s: vocabulary has %q but schema enum does not", label, x)
		}
	}
	for x := range g {
		if !w[x] {
			t.Errorf("%s: schema enum has %q but vocabulary does not", label, x)
		}
	}
}
