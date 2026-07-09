package templatevalidation

import "time"

const (
	Provider = "template-manager"
	Phase    = "templates"

	CodeProvenanceMissing        = "PROVENANCE_MISSING"
	CodeTemplateUnknown          = "TEMPLATE_UNKNOWN"
	CodeTemplateVersionLag       = "TEMPLATE_VERSION_LAG"
	CodeTemplateManifestDrift    = "TEMPLATE_MANIFEST_DRIFT"
	CodeOrientationStateMissing  = "ORIENTATION_STATE_MISSING"
	CodeInheritedDebtOutstanding = "INHERITED_DEBT_OUTSTANDING"
)

type Severity string

const (
	SeverityError Severity = "ERROR"
	SeverityWarn  Severity = "WARNING"
	SeverityInfo  Severity = "INFO"
)

type Finding struct {
	Code        string
	Severity    Severity
	Title       string
	Message     string
	Location    string
	Remediation string
	Autofix     bool
}

type Provenance struct {
	TemplateID      string
	TemplateVersion string
	GeneratedAt     string
	ManifestSHA     string
	ContentSHA      string
	Adopted         bool
}

type Report struct {
	Scenario    string
	RootPath    string
	Provenance  Provenance
	CurrentTime time.Time
	Findings    []Finding
}

func (r Report) Passed() bool {
	for _, finding := range r.Findings {
		if finding.Severity == SeverityError {
			return false
		}
	}
	return true
}
