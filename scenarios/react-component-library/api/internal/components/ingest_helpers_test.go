package components

import (
	"context"
	"strings"
	"testing"
)

func TestIngestContractValidationAndCanonicalization(t *testing.T) {
	reader := ingestMapReader{
		"valid.json":  `{"kind":"experience-component","z":1,"a":2}`,
		"nested.json": `{"contract":{"kind":"rcl-component-experience-contract"}}`,
		"schema.json": `{"schemaVersion":1}`,
		"wrong.json":  `{"kind":"other"}`,
		"broken.json": `{`,
	}
	for _, tc := range []struct {
		path, want string
		bad        string
	}{
		{path: "", want: ""},
		{path: "valid.json", want: `"a": 2`},
		{path: "nested.json", want: "rcl-component-experience-contract"},
		{path: "schema.json", want: "schemaVersion"},
		{path: "wrong.json", bad: "unsupported kind"},
		{path: "broken.json", bad: "decode experience contract"},
		{path: "missing.json", bad: "read experience contract"},
	} {
		got, err := readIngestExperienceContract(context.Background(), reader, "agent-buyer-range", tc.path)
		if tc.bad != "" {
			if err == nil || !strings.Contains(err.Error(), tc.bad) {
				t.Errorf("contract %q error = %v, want %q", tc.path, err, tc.bad)
			}
			continue
		}
		if err != nil || !strings.Contains(got, tc.want) {
			t.Errorf("contract %q = %q, %v; want content %q", tc.path, got, err, tc.want)
		}
	}
}

func TestIngestUnitRejectsInvalidAndEscapingCompanions(t *testing.T) {
	reader := ingestMapReader{"ui/Entry.tsx": `import { outside } from "../../../outside"; export const Entry = () => outside;`}
	_, err := readIngestUnit(context.Background(), reader, "agent-buyer-range", "ui/Entry.tsx", nil)
	if err == nil || !strings.Contains(err.Error(), "escapes its scenario") {
		t.Fatalf("escaping import error = %v", err)
	}
	_, err = readIngestUnit(context.Background(), reader, "agent-buyer-range", "ui/Entry.jsx", nil)
	if err == nil || !strings.Contains(err.Error(), "must be a .ts or .tsx") {
		t.Fatalf("unsupported entry error = %v", err)
	}
	_, err = readIngestUnit(context.Background(), ingestMapReader{"ui/Entry.tsx": "export const Entry = null;"}, "agent-buyer-range", "ui/Entry.tsx", []string{"../outside.ts"})
	if err != nil {
		t.Fatalf("explicit paths outside the scenario are ignored by closure: %v", err)
	}
}

func TestIngestUnitReportsMissingAndCollidingImports(t *testing.T) {
	missing := ingestMapReader{"src/Entry.tsx": `import { x } from "./missing"; export const Entry = x;`}
	_, err := readIngestUnit(context.Background(), missing, "agent-buyer-range", "src/Entry.tsx", nil)
	if err == nil || !strings.Contains(err.Error(), "could not be read") {
		t.Fatalf("missing companion error = %v", err)
	}

	collision := ingestMapReader{
		"src/Entry.tsx":     `import { one } from "./one/Helper"; import { two } from "./two/Helper"; export const Entry = one + two;`,
		"src/one/Helper.ts": "export const one = 1;",
		"src/two/Helper.ts": "export const two = 2;",
	}
	_, err = readIngestUnit(context.Background(), collision, "agent-buyer-range", "src/Entry.tsx", nil)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("colliding companion error = %v", err)
	}
}

func TestIngestAliasAndFindingHelpers(t *testing.T) {
	reader := ingestMapReader{
		"ui/src/hooks/useThing.ts": "export function useThing() { return null; }",
		"src/Thing.ts":             "export const Thing = true;",
	}
	if got, ok := resolveAliasCompanion(context.Background(), reader, "agent-buyer-range", "@/hooks/useThing"); !ok || !strings.Contains(got, "useThing") {
		t.Fatalf("ui/src alias = %q, %v", got, ok)
	}
	if _, ok := resolveAliasCompanion(context.Background(), reader, "agent-buyer-range", "~/../secret"); ok {
		t.Fatal("escaping alias unexpectedly resolved")
	}
	unit := []ComponentVersionFile{{Path: "Thing.tsx", Content: `import { useThing } from "@/hooks/useThing"; export function Thing() { useThing(); return null; }`}}
	if extras := readDroppedOriginCompanions(context.Background(), reader, "agent-buyer-range", unit); !strings.Contains(extras, "useThing") {
		t.Fatalf("dropped alias companion = %q", extras)
	}
	findings := []IngestFinding{{Code: "token-violation"}, {Code: "behavior-lost"}}
	if got := behaviorFindings(findings); len(got) != 1 || got[0].Code != "behavior-lost" {
		t.Fatalf("behaviorFindings() = %#v", got)
	}
	if paths := ingestFilePaths([]ComponentVersionFile{{Path: "a.ts"}, {Path: "b.tsx"}}); strings.Join(paths, ",") != "a.ts,b.tsx" {
		t.Fatalf("ingestFilePaths() = %#v", paths)
	}
}
