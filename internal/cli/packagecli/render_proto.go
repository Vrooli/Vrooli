package packagecli

import (
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// packageInfoProto maps a governed package onto the wire contract.
func packageInfoProto(p packagegov.Package) *cliv1.PackageInfo {
	return &cliv1.PackageInfo{
		Name:         p.Name,
		RootPath:     p.RootPath,
		ManifestPath: p.ManifestPath,
		Manifest:     packageManifestProto(p.Manifest),
	}
}

func packageManifestProto(m packagegov.Manifest) *cliv1.PackageManifest {
	return &cliv1.PackageManifest{
		Schema:  m.Schema,
		Version: m.Version,
		Package: packageManifestEntryProto(m.Package),
	}
}

func packageManifestEntryProto(e packagegov.ManifestEntry) *cliv1.PackageManifestEntry {
	out := &cliv1.PackageManifestEntry{
		Name:              e.Name,
		DisplayName:       e.DisplayName,
		Description:       e.Description,
		Kind:              string(e.Kind),
		Language:          e.Language,
		ModuleIdentifiers: append([]string(nil), e.ModuleIdentifiers...),
		Adoption:          packageAdoptionProto(e.Adoption),
		Lifecycle:         packageLifecycleProto(e.Lifecycle),
		Refresh:           packageRefreshPolicyProto(e.Refresh),
		Docs:              append([]string(nil), e.Docs...),
	}
	for _, g := range e.GeneratedOutputs {
		consumers := make([]string, 0, len(g.Consumers))
		for _, c := range g.Consumers {
			consumers = append(consumers, string(c))
		}
		out.GeneratedOutputs = append(out.GeneratedOutputs, &cliv1.PackageGeneratedOutput{
			Name:        g.Name,
			Identifiers: append([]string(nil), g.Identifiers...),
			Consumers:   consumers,
		})
	}
	return out
}

func packageAdoptionProto(a packagegov.AdoptionPolicy) *cliv1.PackageAdoptionPolicy {
	allowed := make([]string, 0, len(a.AllowedConsumers))
	for _, c := range a.AllowedConsumers {
		allowed = append(allowed, string(c))
	}
	modes := make([]string, 0, len(a.AdoptionModes))
	for _, m := range a.AdoptionModes {
		modes = append(modes, string(m))
	}
	return &cliv1.PackageAdoptionPolicy{
		ScenarioAdoptable: a.ScenarioAdoptable,
		AllowedConsumers:  allowed,
		AdoptionModes:     modes,
	}
}

func packageCommandSpecsProto(specs []packagegov.CommandSpec) []*cliv1.PackageCommandSpec {
	out := make([]*cliv1.PackageCommandSpec, 0, len(specs))
	for _, s := range specs {
		out = append(out, &cliv1.PackageCommandSpec{
			Name: s.Name,
			Run:  append([]string(nil), s.Run...),
		})
	}
	return out
}

func packageLifecycleProto(l packagegov.LifecyclePolicy) *cliv1.PackageLifecyclePolicy {
	return &cliv1.PackageLifecyclePolicy{
		Generate: packageCommandSpecsProto(l.Generate),
		Build:    packageCommandSpecsProto(l.Build),
	}
}

func packageRefreshPolicyProto(r packagegov.RefreshPolicy) *cliv1.PackageRefreshPolicy {
	return &cliv1.PackageRefreshPolicy{
		Strategy:                string(r.Strategy),
		RestartRunningConsumers: r.RestartRunningConsumers,
	}
}

func packageValidationIssueProto(i packagegov.ValidationIssue) *cliv1.PackageValidationIssue {
	return &cliv1.PackageValidationIssue{
		Severity:    i.Severity,
		Code:        i.Code,
		Message:     i.Message,
		Path:        i.Path,
		PackageName: i.PackageName,
	}
}

func packageValidationReportProto(r packagegov.ValidationReport) *cliv1.PackageValidationReport {
	out := &cliv1.PackageValidationReport{}
	for _, p := range r.Packages {
		out.Packages = append(out.Packages, packageInfoProto(p))
	}
	for _, i := range r.Issues {
		out.Issues = append(out.Issues, packageValidationIssueProto(i))
	}
	return out
}

func packageAuditScanStatsProto(stats packagegov.ScanStats) *cliv1.PackageAuditScanStats {
	skipped := make(map[string]int64, len(stats.SkippedByReason))
	for reason, count := range stats.SkippedByReason {
		skipped[reason] = int64(count)
	}
	return &cliv1.PackageAuditScanStats{
		FilesVisited:    int64(stats.FilesVisited),
		FilesScanned:    int64(stats.FilesScanned),
		FilesSkipped:    int64(stats.FilesSkipped),
		BytesScanned:    stats.BytesScanned,
		SkippedByReason: skipped,
		BudgetExceeded:  stats.BudgetExceeded,
	}
}

// PackageListResponse maps a ListResponse onto the wire contract
// (`vrooli package list --json`).
func PackageListResponse(resp ListResponse) *cliv1.PackageListResponse {
	out := &cliv1.PackageListResponse{Success: true}
	for _, p := range resp.Packages {
		out.Packages = append(out.Packages, packageInfoProto(p))
	}
	return out
}

// PackageInfoResponse maps a single package onto the wire contract
// (`vrooli package info <name> --json`).
func PackageInfoResponse(p packagegov.Package) *cliv1.PackageInfoResponse {
	return &cliv1.PackageInfoResponse{Success: true, Package: packageInfoProto(p)}
}

// PackageDependentsResponse maps a DependentsResponse onto the wire contract
// (`vrooli package dependents <name> --json`).
func PackageDependentsResponse(resp DependentsResponse) *cliv1.PackageDependentsResponse {
	dep := &cliv1.PackageDependents{PackageName: resp.PackageName}
	for _, d := range resp.Dependents {
		dep.Dependents = append(dep.Dependents, &cliv1.PackageDependent{
			PackageName:      d.PackageName,
			ConsumerName:     d.ConsumerName,
			ConsumerPath:     d.ConsumerPath,
			ConsumerClass:    string(d.ConsumerClass),
			AdoptionMode:     string(d.AdoptionMode),
			DependencyFile:   d.DependencyFile,
			DependencyTarget: d.DependencyTarget,
			Version:          d.Version,
		})
	}
	for _, i := range resp.Issues {
		dep.Issues = append(dep.Issues, packageValidationIssueProto(i))
	}
	return &cliv1.PackageDependentsResponse{Success: true, Dependents: dep}
}

// PackageValidateResponse maps a ValidateResponse onto the wire contract
// (`vrooli package validate --json`).
func PackageValidateResponse(resp ValidateResponse) *cliv1.PackageValidateResponse {
	return &cliv1.PackageValidateResponse{
		Success: true,
		Report:  packageValidationReportProto(resp.Report),
	}
}

// PackageRunResponse maps a RunResponse onto the wire contract
// (`vrooli package build|generate <name> --json`).
func PackageRunResponse(resp RunResponse) *cliv1.PackageRunResponse {
	return &cliv1.PackageRunResponse{
		Success: true,
		Result: &cliv1.PackageRunResult{
			PackageName: resp.PackageName,
			Action:      resp.Action,
		},
	}
}

// PackageRefreshResponse maps a RefreshResponse onto the wire contract
// (`vrooli package refresh <name> --json`).
func PackageRefreshResponse(resp RefreshResponse) *cliv1.PackageRefreshResponse {
	result := &cliv1.PackageRefreshResult{PackageName: resp.PackageName}
	for _, item := range resp.Items {
		classes := make([]string, 0, len(item.Classes))
		for _, c := range item.Classes {
			classes = append(classes, string(c))
		}
		result.Items = append(result.Items, &cliv1.PackageRefreshItem{
			Consumer:        item.Consumer,
			ConsumerClass:   string(item.Class),
			ConsumerClasses: classes,
			Action:          string(item.Action),
			Status:          item.Status,
		})
	}
	return &cliv1.PackageRefreshResponse{Success: true, Refresh: result}
}

// PackageAuditResponse maps an AuditResponse onto the wire contract
// (`vrooli package audit --json`).
func PackageAuditResponse(resp AuditResponse) *cliv1.PackageAuditResponse {
	audit := &cliv1.PackageAuditReport{
		Validation: packageValidationReportProto(resp.Report.Validation),
		ScanStats:  packageAuditScanStatsProto(resp.Report.ScanStats),
	}
	for _, i := range resp.Report.Issues {
		audit.Issues = append(audit.Issues, packageValidationIssueProto(i))
	}
	return &cliv1.PackageAuditResponse{Success: true, Audit: audit}
}

// writePackageJSON marshals a package wire-contract message and writes it.
func writePackageJSON(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}
