package dochealth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type manifestCoverage struct {
	InManifest    int
	NotInManifest int
	MissingDocs   []string
	OrphanedDocs  []string
}

func checkManifestCoverage(scenarioDir, manifestRel string, requireAll bool, foundDocs []string) ([]Finding, manifestCoverage, error) {
	var (
		coverage manifestCoverage
		out      []Finding
	)
	if manifestRel == "" {
		return out, coverage, nil
	}
	manifestPath := filepath.Join(scenarioDir, manifestRel)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, coverage, nil
		}
		return out, coverage, err
	}

	var manifestDocs []string
	if perr := parseJSONArray(data, &manifestDocs); perr != nil {
		if perr := parseJSONDocsField(data, &manifestDocs); perr != nil {
			return out, coverage, fmt.Errorf("invalid manifest format: %v", perr)
		}
	}

	manifestSet := make(map[string]struct{})
	for _, doc := range manifestDocs {
		manifestSet[doc] = struct{}{}
	}

	foundSet := make(map[string]struct{})
	for _, doc := range foundDocs {
		rel, err := filepath.Rel(scenarioDir, doc)
		if err != nil {
			rel = doc
		}
		foundSet[rel] = struct{}{}
	}

	for doc := range manifestSet {
		fullPath := filepath.Join(scenarioDir, doc)
		if _, err := os.Stat(fullPath); err != nil {
			coverage.MissingDocs = append(coverage.MissingDocs, doc)
			out = append(out, Finding{
				Code:     "manifest_missing_doc",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("manifest references missing doc: %s", doc),
				Path:     doc,
			})
		} else {
			coverage.InManifest++
		}
	}

	for doc := range foundSet {
		if _, ok := manifestSet[doc]; !ok {
			coverage.OrphanedDocs = append(coverage.OrphanedDocs, doc)
			coverage.NotInManifest++
			if requireAll {
				out = append(out, Finding{
					Code:     "manifest_orphaned_doc",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("doc not in manifest: %s", doc),
					Path:     doc,
				})
			}
		}
	}

	return out, coverage, nil
}

func parseJSONArray(data []byte, out *[]string) error {
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "[") {
		return fmt.Errorf("not a JSON array")
	}
	return json.Unmarshal(data, out)
}

func parseJSONDocsField(data []byte, out *[]string) error {
	var obj struct {
		Docs []string `json:"docs"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*out = obj.Docs
	return nil
}
