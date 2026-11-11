package platformmatrix

import (
	"strings"
	"testing"
	"time"
)

func TestParseCapabilitiesPreservesListAndRejectsDuplicates(t *testing.T) {
	got, err := ParseCapabilities("# list\n| ID · Capability | State |\n|---|---|\n| C01 · Make a picture from text | works |\n| C02 · Edit a picture | partly |\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "C01,C02" {
		t.Fatalf("unexpected order: %v", got)
	}
	if _, err := ParseCapabilities("| C01 · First | works |\n| C01 · Duplicate | works |\n"); err == nil {
		t.Fatal("duplicate capability IDs must fail closed")
	}
}

func TestBuildUsesFleetAndReceiptsWithoutDefaultSupportClaims(t *testing.T) {
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, time.UTC)
	targets := []Target{
		{ID: "linux-amd64", GOOS: "linux", GOARCH: "amd64"},
		{ID: "linux-arm64", GOOS: "linux", GOARCH: "arm64"},
		{ID: "darwin-arm64-metal", GOOS: "darwin", GOARCH: "arm64", Accelerator: "metal"},
		{ID: "windows-amd64", GOOS: "windows", GOARCH: "amd64"},
	}
	fleet := FleetSnapshot{Schema: "vrooli.fleet-platform-snapshot/v1", ObservedAt: now, Nodes: []Node{
		{ID: "linux-node", Name: "linux-node", OS: "linux", Arch: "amd64", Revision: "abcdef123", Online: true, Dispatchable: true},
		{ID: "mac-node-without-metal-fact", Name: "mac-node", OS: "darwin", Arch: "arm64", Revision: "abcdef123", Online: true, Dispatchable: true},
	}}
	good := Check{Command: "scenario package", ExitCode: 0, EvidencePath: "package.log", EvidenceSHA256: strings.Repeat("a", 64)}
	capGood := Check{Command: "scenario test capability C01", ExitCode: 0, EvidencePath: "C01.json", EvidenceSHA256: strings.Repeat("b", 64)}
	failed := Check{Command: "scenario test capability C02", ExitCode: 1, EvidencePath: "C02.json", EvidenceSHA256: strings.Repeat("c", 64)}
	receipt := Receipt{Schema: ReceiptSchema, NodeID: "linux-node", TargetID: "linux-amd64", Revision: "abcdef123", ObservedAt: now, PackageCheck: good, Capabilities: map[string]Check{"C01": capGood, "C02": failed}}
	matrix, err := Build(targets, fleet, []Receipt{receipt}, []string{"C01", "C02"}, "CAPABILITIES.md", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := matrix.Platforms[0].Cells[0]; got.Status != "qualified" || len(got.Evidence) != 2 {
		t.Fatalf("passed package + capability receipt should qualify with both evidence refs: %+v", got)
	}
	if got := matrix.Platforms[0].Cells[1]; got.Status != "failed" || len(got.Evidence) != 2 {
		t.Fatalf("failed capability receipt must remain visible as failed: %+v", got)
	}
	if got := matrix.Platforms[1].Cells[0]; got.Status != "unqualified" || got.Reason != "no node" {
		t.Fatalf("missing target node must say unqualified, no node: %+v", got)
	}
	if got := matrix.Platforms[2].Cells[0]; got.Status != "unqualified" || got.Reason != "no node" {
		t.Fatalf("macOS arm64 without observed Metal must not inherit Darwin support: %+v", got)
	}
	if got := matrix.Platforms[3].Cells[0]; got.Status != "unqualified" || got.Reason != "no node" {
		t.Fatalf("missing Windows node must say unqualified, no node: %+v", got)
	}
}

func TestBuildRejectsReceiptsWithoutEvidenceOrForIncompatibleNodes(t *testing.T) {
	now := time.Now().UTC()
	targets := []Target{{ID: "linux-amd64", GOOS: "linux", GOARCH: "amd64"}}
	fleet := FleetSnapshot{Schema: "vrooli.fleet-platform-snapshot/v1", ObservedAt: now, Nodes: []Node{{ID: "mac", OS: "darwin", Arch: "arm64", Revision: "abc", Online: true, Dispatchable: true}}}
	r := Receipt{Schema: ReceiptSchema, NodeID: "mac", TargetID: "linux-amd64", Revision: "abc", ObservedAt: now,
		PackageCheck: Check{Command: "pkg", EvidencePath: "pkg.log", EvidenceSHA256: strings.Repeat("a", 64)},
		Capabilities: map[string]Check{"C01": {Command: "test", EvidencePath: "test.json", EvidenceSHA256: strings.Repeat("b", 64)}},
	}
	if _, err := Build(targets, fleet, []Receipt{r}, []string{"C01"}, "CAPABILITIES.md", now); err == nil {
		t.Fatal("receipt from a node whose observed platform differs must be rejected")
	}
	fleet.Nodes = []Node{{ID: "linux", OS: "linux", Arch: "amd64", Revision: "abc", Online: true, Dispatchable: true}}
	r.NodeID = "linux"
	r.PackageCheck.EvidenceSHA256 = ""
	if _, err := Build(targets, fleet, []Receipt{r}, []string{"C01"}, "CAPABILITIES.md", now); err == nil {
		t.Fatal("a support receipt without a package evidence digest must be rejected")
	}
}
