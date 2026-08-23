package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCoverageUsesDeclaredArtifactKind(t *testing.T) {
	root := t.TempDir()
	writeArtifact(t, filepath.Join(root, "coverage", "coverage-summary.json"), `{"total":{"lines":{"total":10,"covered":9}},"src/App.tsx":{"lines":{"total":10,"covered":9}}}`)

	coverage, ok := ReadCoverage(root, []Artifact{{Kind: "istanbul-summary", Path: "coverage/coverage-summary.json"}})
	if !ok || coverage["src/App.tsx"].Total != 10 || coverage["src/App.tsx"].Covered != 9 {
		t.Fatalf("coverage=%+v ok=%v", coverage, ok)
	}
}

func TestReadCoverageRejectsOversizedArtifact(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "coverage", "summary.json")
	writeArtifact(t, path, strings.Repeat("x", maxCoverageArtifactBytes+1))
	if _, ok := ReadCoverage(root, []Artifact{{Kind: "istanbul-summary", Path: "coverage/summary.json"}}); ok {
		t.Fatal("oversized coverage artifact was accepted")
	}
}

func TestReadCoverageFallsThroughMissingArtifact(t *testing.T) {
	root := t.TempDir()
	writeArtifact(t, filepath.Join(root, "coverage", "lcov.info"), "SF:src/a.ts\nLF:4\nLH:2\nend_of_record\n")
	coverage, ok := ReadCoverage(root, []Artifact{
		{Kind: "istanbul-summary", Path: "coverage/missing.json"},
		{Kind: "lcov", Path: "coverage/lcov.info"},
	})
	if !ok || coverage["src/a.ts"].Covered != 2 {
		t.Fatalf("coverage=%+v ok=%v", coverage, ok)
	}
}

func TestReadCoverageParsesCoberturaDeclaredArtifact(t *testing.T) {
	root := t.TempDir()
	writeArtifact(t, filepath.Join(root, "coverage.xml"), `<coverage><packages><package><classes><class filename="worker.py"><lines><line number="1" hits="1"/><line number="2" hits="0"/></lines></class></classes></package></packages></coverage>`)
	coverage, ok := ReadCoverage(root, []Artifact{{Kind: "cobertura", Path: "coverage.xml"}})
	if !ok || coverage["worker.py"].Covered != 1 || coverage["worker.py"].Total != 2 {
		t.Fatalf("coverage=%+v ok=%v", coverage, ok)
	}
}

func TestReadCoverageRejectsMalformedDeclaredArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		path     string
		contents string
	}{
		{name: "go malformed record", kind: "go-cover-profile", path: "coverage.out", contents: "mode: atomic\nnot-a-cover-record\n"},
		{name: "lcov missing total", kind: "lcov", path: "coverage/lcov.info", contents: "SF:src/a.ts\nLH:2\nend_of_record\n"},
		{name: "lcov invalid number", kind: "lcov", path: "coverage/lcov.info", contents: "SF:src/a.ts\nLF:nope\nLH:2\nend_of_record\n"},
		{name: "lcov incomplete record", kind: "lcov", path: "coverage/lcov.info", contents: "SF:src/a.ts\nLF:4\nLH:2\n"},
		{name: "cobertura negative line", kind: "cobertura", path: "coverage.xml", contents: `<coverage><packages><package><classes><class filename="a.py"><lines><line number="-1" hits="1"/></lines></class></classes></package></packages></coverage>`},
		{name: "istanbul impossible metric", kind: "istanbul-summary", path: "coverage/coverage-summary.json", contents: `{"src/a.ts":{"lines":{"total":1,"covered":2}}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeArtifact(t, filepath.Join(root, tc.path), tc.contents)
			if _, ok := ReadCoverage(root, []Artifact{{Kind: tc.kind, Path: tc.path}}); ok {
				t.Fatalf("malformed %s artifact was accepted", tc.kind)
			}
		})
	}
}

func TestDefaultCoverageArtifactsFollowAdapterLanguage(t *testing.T) {
	if got := DefaultCoverageArtifacts("python"); len(got) != 1 || got[0].Kind != "cobertura" {
		t.Fatalf("python artifacts=%+v", got)
	}
	if got := DefaultCoverageArtifacts("rust"); len(got) != 1 || got[0].Kind != "lcov" {
		t.Fatalf("rust artifacts=%+v", got)
	}
	if got := DefaultCoverageArtifacts("bash"); len(got) != 0 {
		t.Fatalf("bash artifacts=%+v, want none", got)
	}
}

func writeArtifact(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
