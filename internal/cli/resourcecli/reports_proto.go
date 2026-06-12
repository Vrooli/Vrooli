package resourcecli

import (
	"encoding/json"
	"io"

	"google.golang.org/protobuf/proto"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// writeResourceReportJSON marshals a cli/v1 message and writes a trailing newline.
func writeResourceReportJSON(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}

// --- shared converters --------------------------------------------------------

// resourceCatalogMessage maps a catalog Resource onto the shared cliv1.Resource.
func resourceCatalogMessage(item resources.Resource) *cliv1.Resource {
	return &cliv1.Resource{
		Name:       item.Name,
		Path:       item.Path,
		Exists:     item.Exists,
		Registered: item.Registered,
		Enabled:    item.Enabled,
		Required:   item.Required,
		HasCli:     item.HasCLI,
		Config: &cliv1.ResourceConfig{
			Enabled:     item.Config.Enabled,
			Required:    item.Config.Required,
			Description: item.Config.Description,
		},
		ControlMode:     item.ControlMode,
		Driver:          item.Driver,
		Template:        item.Template,
		PortabilityTier: item.PortabilityTier,
		ManifestPath:    item.ManifestPath,
	}
}

// boolValue maps an internal *bool tri-state onto a structpb.Value (nil -> null).
func boolValue(b *bool) *structpb.Value {
	if b == nil {
		return nil
	}
	v, err := structpb.NewValue(*b)
	if err != nil {
		return nil
	}
	return v
}

// rawValue maps a json.RawMessage onto a structpb.Value (empty/invalid -> null).
func rawValue(raw json.RawMessage) *structpb.Value {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	v, err := structpb.NewValue(decoded)
	if err != nil {
		return nil
	}
	return v
}

// resourceStatusMessage maps the internal resource runtime status onto cliv1.
func resourceStatusMessage(item resources.Status) *cliv1.ResourceStatus {
	return &cliv1.ResourceStatus{
		Resource:   resourceCatalogMessage(item.Resource),
		Installed:  item.Installed,
		Running:    item.Running,
		Healthy:    boolValue(item.Healthy),
		Health:     item.Health,
		StatusCode: item.StatusCode,
		Message:    item.Message,
		ProbeError: item.ProbeError,
		Raw:        rawValue(item.Raw),
	}
}

func discoveryFailureMessages(failures []discovery.Failure) []*cliv1.DiscoveryFailure {
	out := make([]*cliv1.DiscoveryFailure, 0, len(failures))
	for _, failure := range failures {
		out = append(out, &cliv1.DiscoveryFailure{
			Kind:  failure.Kind,
			Name:  failure.Name,
			Path:  failure.Path,
			Stage: failure.Stage,
			Error: failure.Error,
		})
	}
	return out
}

func controlResultItemMessages(items []control.ResultItem) []*cliv1.ResourceControlResultItem {
	out := make([]*cliv1.ResourceControlResultItem, 0, len(items))
	for _, item := range items {
		out = append(out, &cliv1.ResourceControlResultItem{
			Name:    item.Name,
			Message: item.Message,
			Error:   item.Error,
		})
	}
	return out
}

func deprecatedResourceMessage(item resources.DeprecatedResource) *cliv1.ResourceDeprecatedResource {
	return &cliv1.ResourceDeprecatedResource{
		Name:                item.Name,
		DeprecatedAt:        item.DeprecatedAt,
		Reason:              item.Reason,
		Replacement:         item.Replacement,
		ArchivePath:         item.ArchivePath,
		ArchiveHash:         item.ArchiveHash,
		RetentionPolicyDays: int32(item.RetentionPolicyDays),
		RestoreSupported:    item.RestoreSupported,
		PurgeAfter:          item.PurgeAfter,
		PurgedAt:            item.PurgedAt,
	}
}

func blueprintArchivedResourceMessage(item resources.BlueprintArchivedResource) *cliv1.ResourceBlueprintArchivedResource {
	return &cliv1.ResourceBlueprintArchivedResource{
		Name:                item.Name,
		ArchivedAt:          item.ArchivedAt,
		Reason:              item.Reason,
		BlueprintName:       item.BlueprintName,
		ArchivePath:         item.ArchivePath,
		ArchiveHash:         item.ArchiveHash,
		RetentionPolicyDays: int32(item.RetentionPolicyDays),
		RestoreSupported:    item.RestoreSupported,
		PurgeAfter:          item.PurgeAfter,
		PurgedAt:            item.PurgedAt,
	}
}

func archiveGCItemMessages(items []resources.ArchiveGCItem) []*cliv1.ResourceArchiveGCItem {
	out := make([]*cliv1.ResourceArchiveGCItem, 0, len(items))
	for _, item := range items {
		out = append(out, &cliv1.ResourceArchiveGCItem{
			Name:        item.Name,
			ArchivePath: item.ArchivePath,
			Removed:     item.Removed,
		})
	}
	return out
}

func scenarioResourceReferenceMessages(items []resources.ScenarioResourceReference) []*cliv1.ResourceScenarioResourceReference {
	out := make([]*cliv1.ResourceScenarioResourceReference, 0, len(items))
	for _, item := range items {
		out = append(out, &cliv1.ResourceScenarioResourceReference{
			Scenario:     item.Scenario,
			Resource:     item.Resource,
			ManifestPath: item.ManifestPath,
		})
	}
	return out
}

// --- template converters ------------------------------------------------------

func templateVarMessage(v resources.ResourceTemplateVar) *cliv1.ResourceTemplateVar {
	return &cliv1.ResourceTemplateVar{
		Flag:        v.Flag,
		Description: v.Description,
		Default:     v.Default,
	}
}

func templateVarMap(vars map[string]resources.ResourceTemplateVar) map[string]*cliv1.ResourceTemplateVar {
	if vars == nil {
		return nil
	}
	out := make(map[string]*cliv1.ResourceTemplateVar, len(vars))
	for key, v := range vars {
		out[key] = templateVarMessage(v)
	}
	return out
}

func templateManifestMessage(m resources.ResourceTemplateManifest) *cliv1.ResourceTemplateManifest {
	return &cliv1.ResourceTemplateManifest{
		Name:                 m.Name,
		DisplayName:          m.DisplayName,
		Description:          m.Description,
		Driver:               m.Driver,
		RequiredVars:         templateVarMap(m.RequiredVars),
		OptionalVars:         templateVarMap(m.OptionalVars),
		Docs:                 m.Docs,
		PlatformExpectations: m.PlatformExpectations,
		Transitional:         m.Transitional,
	}
}

func templateInfoMessage(info resources.ResourceTemplateInfo) *cliv1.ResourceTemplateInfo {
	return &cliv1.ResourceTemplateInfo{
		Name:     info.Name,
		Path:     info.Path,
		Manifest: templateManifestMessage(info.Manifest),
	}
}

func templateSummaryMessage(s resources.ResourceTemplateSummary) *cliv1.ResourceTemplateSummary {
	return &cliv1.ResourceTemplateSummary{
		Name:         s.Name,
		DisplayName:  s.DisplayName,
		Driver:       s.Driver,
		Transitional: s.Transitional,
		Description:  s.Description,
	}
}

// --- response builders --------------------------------------------------------

// ResourceStatusesResponse maps the fleet `resource status --json` envelope.
func ResourceStatusesResponse(items []resources.Status, failures []discovery.Failure) *cliv1.ResourceStatusesResponse {
	resp := &cliv1.ResourceStatusesResponse{Success: true}
	for _, item := range items {
		resp.Resources = append(resp.Resources, resourceStatusMessage(item))
	}
	resp.DiscoveryFailures = discoveryFailureMessages(failures)
	return resp
}

// ResourceStatusResponse maps the single `resource status --json` envelope.
func ResourceStatusResponse(item resources.Status) *cliv1.ResourceStatusResponse {
	return &cliv1.ResourceStatusResponse{
		Success:   true,
		Name:      item.Resource.Name,
		Installed: item.Installed,
		Running:   item.Running,
		Healthy:   boolValue(item.Healthy),
		Status:    item.Message,
		Resource:  resourceStatusMessage(item),
	}
}

// ResourceInfoResponse maps the `resource info --json` envelope.
func ResourceInfoResponse(item resources.Status) *cliv1.ResourceInfoResponse {
	return &cliv1.ResourceInfoResponse{
		Success:  true,
		Resource: resourceStatusMessage(item),
	}
}

// ResourceStartAllResponse maps the `resource start-all --json` envelope.
func ResourceStartAllResponse(report control.StartReport) *cliv1.ResourceStartAllResponse {
	return &cliv1.ResourceStartAllResponse{
		Success: true,
		Report: &cliv1.ResourceStartReport{
			Started: controlResultItemMessages(report.Started),
			Failed:  controlResultItemMessages(report.Failed),
			Message: report.Message,
		},
	}
}

// ResourceStopAllResponse maps the `resource stop-all --json` envelope.
func ResourceStopAllResponse(report control.StopReport) *cliv1.ResourceStopAllResponse {
	return &cliv1.ResourceStopAllResponse{
		Success: true,
		Report: &cliv1.ResourceStopReport{
			Stopped: controlResultItemMessages(report.Stopped),
			Failed:  controlResultItemMessages(report.Failed),
			Message: report.Message,
		},
	}
}

// ResourceDeprecationResponse maps the `resource deprecate --json` envelope.
func ResourceDeprecationResponse(report resources.DeprecationReport) *cliv1.ResourceDeprecationResponse {
	return &cliv1.ResourceDeprecationResponse{
		Success: true,
		Report: &cliv1.ResourceDeprecationReport{
			Resource:   deprecatedResourceMessage(report.Resource),
			Archived:   report.Archived,
			ArchiveDir: report.ArchiveDir,
		},
	}
}

// ResourceRestoreResponse maps the `resource restore --json` envelope.
func ResourceRestoreResponse(report resources.RestoreReport) *cliv1.ResourceRestoreResponse {
	return &cliv1.ResourceRestoreResponse{
		Success: true,
		Report: &cliv1.ResourceRestoreReport{
			Resource:     deprecatedResourceMessage(report.Resource),
			Restored:     report.Restored,
			RestoredPath: report.RestoredPath,
		},
	}
}

// ResourceBlueprintArchiveResponse maps `resource archive-to-blueprint --json`.
func ResourceBlueprintArchiveResponse(report resources.BlueprintArchiveReport) *cliv1.ResourceBlueprintArchiveResponse {
	return &cliv1.ResourceBlueprintArchiveResponse{
		Success: true,
		Report: &cliv1.ResourceBlueprintArchiveReport{
			Resource:   blueprintArchivedResourceMessage(report.Resource),
			Archived:   report.Archived,
			ArchiveDir: report.ArchiveDir,
		},
	}
}

// ResourceBlueprintRestoreResponse maps `resource restore-blueprint --json`.
func ResourceBlueprintRestoreResponse(report resources.BlueprintRestoreReport) *cliv1.ResourceBlueprintRestoreResponse {
	return &cliv1.ResourceBlueprintRestoreResponse{
		Success: true,
		Report: &cliv1.ResourceBlueprintRestoreReport{
			Resource:     blueprintArchivedResourceMessage(report.Resource),
			Restored:     report.Restored,
			RestoredPath: report.RestoredPath,
		},
	}
}

// ResourceArchiveGCResponse maps `resource archive gc[-blueprints] --json`.
func ResourceArchiveGCResponse(report resources.ArchiveGCReport) *cliv1.ResourceArchiveGCResponse {
	return &cliv1.ResourceArchiveGCResponse{
		Success: true,
		Report: &cliv1.ResourceArchiveGCReport{
			Removed: archiveGCItemMessages(report.Removed),
			Skipped: archiveGCItemMessages(report.Skipped),
		},
	}
}

// ResourceListDeprecatedResponse maps `resource list-deprecated --json`.
func ResourceListDeprecatedResponse(items []resources.DeprecatedResource) *cliv1.ResourceListDeprecatedResponse {
	resp := &cliv1.ResourceListDeprecatedResponse{Success: true}
	for _, item := range items {
		resp.Resources = append(resp.Resources, deprecatedResourceMessage(item))
	}
	return resp
}

// ResourceListBlueprintArchivedResponse maps `resource list-blueprint-archived --json`.
func ResourceListBlueprintArchivedResponse(items []resources.BlueprintArchivedResource) *cliv1.ResourceListBlueprintArchivedResponse {
	resp := &cliv1.ResourceListBlueprintArchivedResponse{Success: true}
	for _, item := range items {
		resp.Resources = append(resp.Resources, blueprintArchivedResourceMessage(item))
	}
	return resp
}

// ResourceSchemaValidationResponse maps `resource schema validate --json`.
// success mirrors report.passed (WriteFieldsWithSuccess).
func ResourceSchemaValidationResponse(report resources.ResourceSchemaValidationReport) *cliv1.ResourceSchemaValidationResponse {
	body := &cliv1.ResourceSchemaValidationReport{
		Passed:         report.Passed,
		ResourceCount:  int32(report.ResourceCount),
		DefinitionPath: report.DefinitionPath,
	}
	for _, issue := range report.ArtifactIssues {
		body.ArtifactIssues = append(body.ArtifactIssues, &cliv1.ResourceSchemaArtifactIssue{
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	body.MissingReferences = scenarioResourceReferenceMessages(report.MissingReferences)
	return &cliv1.ResourceSchemaValidationResponse{
		Success: report.Passed,
		Report:  body,
	}
}

// ResourceSchemaSyncResponse maps `resource schema sync --json`.
// success mirrors report.passed (WriteFieldsWithSuccess).
func ResourceSchemaSyncResponse(report resources.ResourceSchemaSyncReport) *cliv1.ResourceSchemaSyncResponse {
	body := &cliv1.ResourceSchemaSyncReport{
		Passed:         report.Passed,
		ResourceCount:  int32(report.ResourceCount),
		DefinitionPath: report.DefinitionPath,
		WrittenPaths:   report.WrittenPaths,
	}
	body.MissingReferences = scenarioResourceReferenceMessages(report.MissingReferences)
	return &cliv1.ResourceSchemaSyncResponse{
		Success: report.Passed,
		Report:  body,
	}
}

// ResourceTemplateListResponse maps `resource template list --json`.
func ResourceTemplateListResponse(items []resources.ResourceTemplateInfo) *cliv1.ResourceTemplateListResponse {
	resp := &cliv1.ResourceTemplateListResponse{Success: true}
	for _, item := range items {
		resp.Templates = append(resp.Templates, templateInfoMessage(item))
	}
	return resp
}

// ResourceTemplateShowResponse maps `resource template show --json`.
func ResourceTemplateShowResponse(info resources.ResourceTemplateInfo) *cliv1.ResourceTemplateShowResponse {
	return &cliv1.ResourceTemplateShowResponse{
		Success:  true,
		Template: templateInfoMessage(info),
	}
}

// ResourceTemplateValidationResponse maps `resource template validate --json`.
func ResourceTemplateValidationResponse(report resources.ResourceTemplateValidationReport) *cliv1.ResourceTemplateValidationResponse {
	body := &cliv1.ResourceTemplateValidationReport{Count: int32(report.Count)}
	for _, item := range report.Templates {
		body.Templates = append(body.Templates, templateSummaryMessage(item))
	}
	return &cliv1.ResourceTemplateValidationResponse{
		Success: true,
		Report:  body,
	}
}

// ResourceTemplateGenerateResponse maps `resource template generate --json`.
func ResourceTemplateGenerateResponse(report resources.ResourceTemplateGenerateReport) *cliv1.ResourceTemplateGenerateResponse {
	return &cliv1.ResourceTemplateGenerateResponse{
		Success: true,
		Report: &cliv1.ResourceTemplateGenerateReport{
			Template:      templateSummaryMessage(report.Template),
			BlueprintName: report.BlueprintName,
			Destination:   report.Destination,
			Values:        report.Values,
			Files:         report.Files,
			DryRun:        report.DryRun,
		},
	}
}

// --- exported write helpers for the resourcehandlers package ------------------

// WriteTemplateList emits `resource template list --json`.
func WriteTemplateList(w io.Writer, items []resources.ResourceTemplateInfo) error {
	return writeResourceReportJSON(w, ResourceTemplateListResponse(items))
}

// WriteTemplateShow emits `resource template show --json`.
func WriteTemplateShow(w io.Writer, info resources.ResourceTemplateInfo) error {
	return writeResourceReportJSON(w, ResourceTemplateShowResponse(info))
}

// WriteTemplateValidationReport emits `resource template validate --json`.
func WriteTemplateValidationReport(w io.Writer, report resources.ResourceTemplateValidationReport) error {
	return writeResourceReportJSON(w, ResourceTemplateValidationResponse(report))
}

// WriteTemplateGenerateReport emits `resource template generate --json`.
func WriteTemplateGenerateReport(w io.Writer, report resources.ResourceTemplateGenerateReport) error {
	return writeResourceReportJSON(w, ResourceTemplateGenerateResponse(report))
}
