package supervisioncli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestRenderJSONUsesTypedContract(t *testing.T) {
	report := sampleReport()
	var output bytes.Buffer
	if err := Render(&output, cliout.FormatJSON, report); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Members []apicoreset.Member `json:"members"`
	}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded.Members) != 1 || decoded.Members[0].AttributionChain[0].Source != "core.seed" {
		t.Fatalf("decoded response = %+v", decoded)
	}
}

func TestRenderHumanUsesSharedTableVocabulary(t *testing.T) {
	var output bytes.Buffer
	if err := Render(&output, cliout.FormatHuman, sampleReport()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Name", "Kind", "Intent", "scenario:seed"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("human output missing %q: %s", want, output.String())
		}
	}
}

func TestFilterByKind(t *testing.T) {
	report := sampleReport()
	report.Members = append(report.Members, apicoreset.Member{Name: "redis", Kind: "resource"})
	filtered := Filter(report, "resource")
	if len(filtered.Members) != 1 || filtered.Members[0].Name != "redis" || filtered.MemberCounts["resource"] != 1 {
		t.Fatalf("filtered report = %+v", filtered)
	}
}

func sampleReport() apicoreset.Report {
	return apicoreset.Report{
		Source: "computed",
		Members: []apicoreset.Member{{
			Name: "seed", Kind: "scenario", SupervisionIntent: "must_start",
			AttributionChain: []apicoreset.AttributionStep{{Name: "seed", Kind: "scenario", SupervisionIntent: "must_start", Source: "core.seed"}},
		}},
		MemberCounts: map[string]int{"scenario": 1},
	}
}

func TestRenderPreservesTrustedBaseViolationsForOperators(t *testing.T) {
	report := sampleReport()
	report.TrustedBaseViolations = map[string][]string{"agent-manager": {"workspace-sandbox"}}
	var output bytes.Buffer
	if err := Render(&output, cliout.FormatJSON, report); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Violations map[string][]string `json:"trusted_base_violations"`
	}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if got := decoded.Violations["agent-manager"]; len(got) != 1 || got[0] != "workspace-sandbox" {
		t.Fatalf("trusted-base violation disappeared from operator JSON: %s", output.String())
	}
	output.Reset()
	if err := Render(&output, cliout.FormatHuman, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "agent-manager") || !strings.Contains(output.String(), "workspace-sandbox") {
		t.Fatalf("trusted-base violation disappeared from operator display: %s", output.String())
	}
}

func TestRenderHumanPreservesUnavailableManifestEvidence(t *testing.T) {
	report := sampleReport()
	report.LoadErrors = map[string]string{"missing-peer": "manifest cannot be read"}
	var output bytes.Buffer
	if err := Render(&output, cliout.FormatHuman, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "missing-peer") || !strings.Contains(output.String(), "manifest cannot be read") {
		t.Fatalf("unreadable manifest disappeared from operator display: %s", output.String())
	}
}
