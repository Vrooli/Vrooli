package resourcecli

import (
	"encoding/json"

	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// writeResourceReportJSON marshals a cli/v1 message and writes a trailing newline.

// --- shared converters --------------------------------------------------------

// resourceCatalogMessage maps a catalog Resource onto the shared cliv1.Resource.
func resourceCatalogMessage(item resources.Resource) *cliv1.Resource {
	return &cliv1.Resource{
		Name:           item.Name,
		Path:           item.Path,
		Exists:         item.Exists,
		Registered:     item.Registered,
		Enabled:        item.Enabled,
		Required:       item.Required,
		DeclaresCli:    item.DeclaresCLI,
		CliInstalled:   item.CLIInstalled,
		CliStateReason: item.CLIStateReason,
		Config: &cliv1.ResourceConfig{
			Enabled:     item.Config.Enabled,
			Required:    item.Config.Required,
			Description: item.Config.Description,
		},
		ControlMode:  item.ControlMode,
		Driver:       item.Driver,
		ManifestPath: item.ManifestPath,
	}
}

// boolValue maps an internal *bool tri-state onto a structpb.Value (nil -> null).
func boolValue(b *bool) *structpb.Value {
	if b == nil {
		return nil
	}
	v, err := cliout.NewJSONValue(*b)
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
	v, err := cliout.NewJSONValue(decoded)
	if err != nil {
		return nil
	}
	return v
}

// resourceStatusMessage maps the internal resource runtime status onto cliv1.
func resourceStatusMessage(item resources.Status) *cliv1.ResourceStatus {
	observedMode := item.ObservedMode
	if observedMode == "" {
		observedMode = "not_evaluated"
	}
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

		Serving:      boolValue(item.Serving),
		DeclaredMode: item.DeclaredMode,
		ObservedMode: observedMode,
		ModeDrift:    item.ModeDrift,
		ModeReason:   item.ModeReason,
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
		Serving:   boolValue(item.Serving),
		ModeDrift: item.ModeDrift,
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
