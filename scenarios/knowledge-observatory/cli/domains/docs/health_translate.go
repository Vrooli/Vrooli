package docs

import (
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
)

// protoToHealthResponse maps the proto DocHealth response onto the
// CLI's legacy HealthResponse so RenderHealthReport keeps working
// unchanged. New finding families (content, reference, manifest) all
// flow into HealthResponse.Warnings / ContractFindings / ContentIssues
// so they appear under the existing "Findings" triage section.
func protoToHealthResponse(in *kov1.DocHealthResponse) HealthResponse {
	if in == nil {
		return HealthResponse{}
	}
	out := HealthResponse{
		ScenarioName:     in.GetScenarioName(),
		SourceTemplateID: in.GetSourceTemplateId(),
		ManifestPath:     in.GetManifestPath(),
		ManifestStatus:   in.GetManifestStatus(),
		HealthScore:      in.GetHealthScore(),
		TotalDocs:        int(in.GetTotalDocs()),
		MissingDocs:      missingDocPaths(in.GetMissingDocs()),
		ExtraDocs:        append([]string(nil), in.GetExtraDocs()...),
		TemporaryDocs:    append([]string(nil), in.GetTemporaryDocs()...),
	}
	for _, m := range in.GetMisplacedDocs() {
		out.MisplacedDocs = append(out.MisplacedDocs, AuditMisplacedDoc{
			ActualPath:   m.GetActualPath(),
			ExpectedPath: m.GetExpectedPath(),
			DocType:      m.GetDocType(),
			Severity:     severityProtoToString(m.GetSeverity()),
		})
	}
	out.ContractFindings = findingsToWarnings(in.GetContractFindings(), "contract")
	out.ContentIssues = append(out.ContentIssues, findingsToWarnings(in.GetContentFindings(), "content")...)
	out.ContentIssues = append(out.ContentIssues, findingsToWarnings(in.GetReferenceFindings(), "reference")...)
	out.ContentIssues = append(out.ContentIssues, findingsToWarnings(in.GetManifestFindings(), "manifest")...)
	out.CanAutoFix = len(out.MisplacedDocs) > 0
	return out
}

func missingDocPaths(missing []*kov1.DocHealthMissingDoc) []string {
	if len(missing) == 0 {
		return nil
	}
	out := make([]string, 0, len(missing))
	for _, m := range missing {
		if m == nil {
			continue
		}
		out = append(out, m.GetDocType())
	}
	return out
}

func findingsToWarnings(findings []*kov1.DocHealthFinding, defaultType string) []HealthWarning {
	if len(findings) == 0 {
		return nil
	}
	out := make([]HealthWarning, 0, len(findings))
	for _, f := range findings {
		if f == nil {
			continue
		}
		w := HealthWarning{
			Type:     defaultType,
			Message:  f.GetMessage(),
			Path:     f.GetPath(),
			DocType:  f.GetDocType(),
			Severity: severityProtoToString(f.GetSeverity()),
		}
		if code := f.GetCode(); code != "" {
			w.Type = code
		}
		out = append(out, w)
	}
	return out
}

func severityProtoToString(s kov1.DocHealthSeverity) string {
	switch s {
	case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_INFO:
		return "info"
	case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING:
		return "warning"
	case kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE:
		return "error"
	default:
		return "warning"
	}
}
