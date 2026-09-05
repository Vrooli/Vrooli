package monetization

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"testing"
)

// TestFindingCodesMatchDescriptor makes the provider's gating contract
// structural. The source scan is intentionally used here: adding a finding
// call is a contract change and must be accompanied by a descriptor entry.
func TestFindingCodesMatchDescriptor(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Join(filepath.Dir(sourceFile), "../../../../../")
	source, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "conformance.go"))
	if err != nil {
		t.Fatalf("read conformance provider: %v", err)
	}
	codes := providerFindingCodes(source)
	data, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "landing-page-business-suite", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("read phase descriptor: %v", err)
	}
	descriptorCodes := descriptorFindingCodes(t, data)
	missing, extra := findingCodeDiff(codes, descriptorCodes)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("monetization finding/descriptor sets differ: missing descriptor entries=%v, descriptor-only entries=%v", missing, extra)
	}
}

func TestFindingDescriptorMutationIsDetected(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "conformance.go"))
	if err != nil {
		t.Fatalf("read conformance provider: %v", err)
	}
	_, descriptorPath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve descriptor path")
	}
	repoRoot := filepath.Join(filepath.Dir(descriptorPath), "../../../../../")
	data, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "landing-page-business-suite", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("read phase descriptor: %v", err)
	}
	var descriptor map[string]any
	if err := json.Unmarshal(data, &descriptor); err != nil {
		t.Fatalf("decode phase descriptor: %v", err)
	}
	maturity, ok := descriptor["maturity"].(map[string]any)
	if !ok {
		t.Fatal("descriptor maturity object missing")
	}
	findings, ok := maturity["findings"].(map[string]any)
	if !ok {
		t.Fatal("descriptor findings object missing")
	}
	delete(findings, "money.live_catalog_unavailable")
	mutated, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("encode mutated descriptor: %v", err)
	}
	missing, _ := findingCodeDiff(providerFindingCodes(source), descriptorFindingCodes(t, mutated))
	if len(missing) != 1 || missing[0] != "money.live_catalog_unavailable" {
		t.Fatalf("descriptor mutation was not detected: missing=%v", missing)
	}
}

func providerFindingCodes(source []byte) map[string]struct{} {
	codes := map[string]struct{}{}
	for _, match := range regexp.MustCompile(`finding\("([^"]+)"`).FindAllSubmatch(source, -1) {
		codes[string(match[1])] = struct{}{}
	}
	return codes
}

func descriptorFindingCodes(t *testing.T, data []byte) map[string]struct{} {
	t.Helper()
	var descriptor struct {
		Maturity struct {
			Findings map[string]json.RawMessage `json:"findings"`
		} `json:"maturity"`
	}
	if err := json.Unmarshal(data, &descriptor); err != nil {
		t.Fatalf("decode phase descriptor: %v", err)
	}
	codes := make(map[string]struct{}, len(descriptor.Maturity.Findings))
	for code := range descriptor.Maturity.Findings {
		codes[code] = struct{}{}
	}
	return codes
}

func findingCodeDiff(provider, descriptor map[string]struct{}) (missing, extra []string) {
	for code := range provider {
		if _, ok := descriptor[code]; !ok {
			missing = append(missing, code)
		}
	}
	for code := range descriptor {
		if _, ok := provider[code]; !ok {
			extra = append(extra, code)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
