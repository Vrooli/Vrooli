package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/basrefs"
	"experience-manager/internal/spec"
)

// BASReferenceCheck validates the explicit spec_entry_id labels used by
// generated BAS cases. Workflow-health still owns cataloging and execution.
type BASReferenceCheck struct{}

func (BASReferenceCheck) Name() string { return "bas.reference_integrity" }

func (BASReferenceCheck) Run(_ context.Context, report spec.Report) []spec.Finding {
	if report.Spec == nil {
		return nil
	}
	refs := loadCaseSpecRefs(report.TargetPath)
	specExists := map[string]bool{}
	activePages := map[string]string{}
	for _, ref := range report.Spec.Index.Pages {
		specExists[ref.ID] = true
		if ref.Status == "active" {
			activePages[ref.ID] = "experience/" + ref.Path
		}
	}
	for _, ref := range report.Spec.Index.Components {
		specExists[ref.ID] = true
	}
	var findings []spec.Finding
	for specID, files := range refs {
		if specExists[specID] {
			continue
		}
		sort.Strings(files)
		findings = append(findings, spec.Finding{
			Code:       spec.CodeRefUnresolved,
			Severity:   spec.SeverityError,
			Message:    fmt.Sprintf("BAS case references missing experience spec entry %q", specID),
			Locations:  []string{files[0]},
			Suggestion: "Update metadata.labels.spec_entry_id or restore the referenced experience page or component.",
		})
	}
	for pageID, loc := range activePages {
		if len(refs[pageID]) > 0 {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeRouteUnspecced,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("active experience page %q has no BAS case referencing metadata.labels.spec_entry_id", pageID),
			Locations:  []string{loc},
			Suggestion: "Run experience-manager spec scaffold or add a BAS case with labels.spec_entry_id set to the page id.",
		})
	}
	sortFindings(findings)
	return findings
}

func loadCaseSpecRefs(root string) map[string][]string {
	out := map[string][]string{}
	for _, path := range basCaseFiles(root) {
		var doc struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
		}
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &doc) != nil {
			continue
		}
		id := strings.TrimSpace(doc.Metadata.Labels["spec_entry_id"])
		if id == "" {
			continue
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		out[id] = append(out[id], filepath.ToSlash(rel))
	}
	return out
}

func basCaseFiles(root string) []string {
	return basrefs.CaseFiles(root)
}

func sortFindings(findings []spec.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return strings.Join(findings[i].Locations, ",") < strings.Join(findings[j].Locations, ",")
	})
}
