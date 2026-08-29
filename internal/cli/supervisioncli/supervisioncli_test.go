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
