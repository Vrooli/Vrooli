package adapters

import "encoding/json"

// ArchitectureDrift and ProjectionDrift are adapter-produced findings. They
// intentionally contain only normalized evidence; the validation kernel adds
// scenario identity, severity, and API-specific fields.
type ArchitectureDrift struct {
	Code         string
	File         string
	Message      string
	Evidence     string
	Expected     string
	Observed     string
	WhyItMatters string
	Remediation  string
}

type ProjectionPolicy struct {
	CoverageFloor     float64
	Environment       string
	SetupFiles        []string
	CoverageProvider  string
	CoverageReporters []string
	CoverageInclude   []string
	CoverageExclude   []string
	ReportOnFailure   bool
}

type ProjectionInput struct {
	RootPath string
	Policy   ProjectionPolicy
}

type ProjectionDrift struct {
	File        string
	Message     string
	Evidence    string
	Expected    string
	Observed    string
	Remediation string
}

type ProjectionCheck struct {
	Key         string
	Owner       string
	File        string
	PolicyValue string
	NativeValue string
	Pass        bool
	Remediation string
}

// Analyzer is the optional static-analysis capability of an adapter. An
// adapter may implement only planning and artifact collection; the kernel must
// never infer architecture rules from language alone.
type Analyzer interface {
	Adapter
	ValidatePolicySettings(map[string]json.RawMessage) error
	DefaultProjectionPolicy() ProjectionPolicy
	ProjectionPolicyFromSettings(map[string]json.RawMessage) ProjectionPolicy
	AnalyzeArchitecture(string) []ArchitectureDrift
	AnalyzeProjection(ProjectionInput) []ProjectionDrift
	AnalyzeProjectionChecks(ProjectionInput) []ProjectionCheck
}
