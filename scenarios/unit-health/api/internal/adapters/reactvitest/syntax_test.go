package reactvitest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"path/filepath"
	"testing"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

func TestSyntaxLiveOwnerEvidenceConvertsWithoutRelinting(t *testing.T) {
	path := os.Getenv("UNIT_HEALTH_SYNTAX_FIXTURE")
	root := os.Getenv("UNIT_HEALTH_SYNTAX_ROOT")
	if path == "" || root == "" {
		t.Skip("provide freshly captured owner RPC response and source root")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var response auditv1.ObserveTestSyntaxResponse
	if err := protojson.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	files := []string{"src/test-utils/factories.test.ts"}
	rows := convertSyntax(adapters.QualityInput{Root: root, Workspace: "ui", TestKind: "unit"}, files, &response)
	if len(rows) != 3 {
		t.Fatal(rows)
	}
	for _, row := range rows {
		if row.Status != testquality.CheckedClean || row.Target.Scope != "file" || row.Target.TestID != "" {
			t.Fatalf("live owner conversion: %+v", row)
		}
	}
}

func syntaxFixture(t *testing.T) (adapters.QualityInput, *auditv1.ObserveTestSyntaxResponse) {
	t.Helper()
	root := t.TempDir()
	data := []byte("import {expect} from 'vitest'; expect(2);")
	if err := os.WriteFile(filepath.Join(root, "a.test.ts"), data, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	input := adapters.QualityInput{Root: root, Workspace: "ui", TestKind: "unit"}
	response := &auditv1.ObserveTestSyntaxResponse{SchemaVersion: "vitest-lint/v1", Observations: []*auditv1.TestSyntaxObservation{{SchemaVersion: "vitest-lint/v1", File: "a.test.ts", SourceDigest: hex.EncodeToString(digest[:]), Profile: "vitest-syntax-1.6.9", PluginVersion: "1.6.9", EslintVersion: "9.39.4", ParserVersion: "8.59.2", Status: "observed", Reason: "none", Checks: []*auditv1.TestSyntaxCheck{{Rule: "focused-test", Status: "checked_clean", Reason: "none"}, {Rule: "malformed-expectation", Status: "violation", Reason: "none"}, {Rule: "async-assertion", Status: "checked_clean", Reason: "none"}}, Diagnostics: []*auditv1.TestSyntaxDiagnostic{{CanonicalRule: "malformed-expectation", NativeRuleId: "vitest/valid-expect", MessageId: "matcherNotFound", Message: "Expect must have a corresponding matcher call", Line: 1, Column: 32, NativeSeverity: 1}}}}}
	response.Observations[0].ConfigDigest = syntaxConfigDigest
	return input, response
}
func TestSyntaxConversionPreservesFileScopeAndNativeDetail(t *testing.T) {
	in, response := syntaxFixture(t)
	rows := convertSyntax(in, []string{"a.test.ts"}, response)
	if len(rows) != 3 {
		t.Fatal(rows)
	}
	row := rows[1]
	if row.Status != testquality.Violation || row.Target.Scope != "file" || row.Target.TestID != "" || len(row.Diagnostics) != 1 || row.Diagnostics[0].Location.Column != 32 {
		t.Fatalf("lost scope or detail: %+v", row)
	}
	report, err := testquality.BuildReport("1", rows, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range report.Coverage {
		if count.Scope != "file" || count.Discovered != 1 {
			t.Fatalf("inflated tests: %+v", count)
		}
	}
}
func TestSyntaxConversionRejectsUntrustworthyEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*auditv1.ObserveTestSyntaxResponse)
		reason testquality.Reason
	}{
		{"owner missing", func(r *auditv1.ObserveTestSyntaxResponse) { r.UnavailableReason = "owner-unavailable" }, testquality.OwnerUnavailable},
		{"stale source", func(r *auditv1.ObserveTestSyntaxResponse) { r.Observations[0].SourceDigest = "old" }, testquality.StaleEvidence},
		{"future plugin", func(r *auditv1.ObserveTestSyntaxResponse) { r.Observations[0].PluginVersion = "9" }, testquality.UnsupportedVersion},
		{"missing file", func(r *auditv1.ObserveTestSyntaxResponse) { r.Observations = nil }, testquality.MissingInput},
		{"duplicate file", func(r *auditv1.ObserveTestSyntaxResponse) { r.Observations = append(r.Observations, r.Observations[0]) }, testquality.MissingInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, response := syntaxFixture(t)
			tc.change(response)
			for _, row := range convertSyntax(in, []string{"a.test.ts"}, response) {
				if row.Status != testquality.Unknown || row.Reason != tc.reason {
					t.Fatalf("untrustworthy evidence accepted: %+v", row)
				}
			}
		})
	}
}
func TestSyntaxConversionDoesNotTurnUnsupportedFocusIntoViolation(t *testing.T) {
	in, response := syntaxFixture(t)
	response.Observations[0].Checks[0].Status = "unknown"
	response.Observations[0].Checks[0].Reason = "unsupported-test-kind"
	response.Observations[0].Diagnostics = append(response.Observations[0].Diagnostics, &auditv1.TestSyntaxDiagnostic{CanonicalRule: "focused-test", NativeRuleId: "vitest/no-focused-tests", Message: "local lookalike", Line: 1, Column: 1})
	row := convertSyntax(in, []string{"a.test.ts"}, response)[0]
	if row.Status != testquality.Unknown || row.Reason != testquality.UnsupportedTestKind {
		t.Fatal(row)
	}
}
func TestSyntaxDryRunDoesNotContactOwner(t *testing.T) {
	in, _ := syntaxFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rows, reason := (Analyzer{}).CollectSyntax(ctx, in)
	if reason != testquality.ReasonNone || len(rows) != 3 {
		t.Fatalf("rows=%v reason=%s", rows, reason)
	}
	for _, row := range rows {
		if row.Reason != testquality.NotExecuted {
			t.Fatal(row)
		}
	}
}

func TestSyntaxProjectionRejectsChangedRuleSelection(t *testing.T) {
	in, response := syntaxFixture(t)
	if !syntaxProjection(response.Observations[0]).Pass {
		t.Fatal("reviewed config did not match")
	}
	for _, digest := range []string{"", "changed-rule-options"} {
		response.Observations[0].ConfigDigest = digest
		if syntaxProjection(response.Observations[0]).Pass {
			t.Fatal("unknown configuration passed projection")
		}
		for _, row := range convertSyntax(in, []string{"a.test.ts"}, response) {
			if row.Status != testquality.Unknown || row.Reason != testquality.UnsupportedVersion {
				t.Fatal(row)
			}
		}
	}
}
