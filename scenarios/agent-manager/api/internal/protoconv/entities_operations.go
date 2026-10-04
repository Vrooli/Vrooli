// Responsibility: retain entities declarations within their original package.
package protoconv

import (
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"strings"
	"time"
)

// RunnerStatusToProto converts runner information to proto RunnerStatus.
func RunnerStatusToProto(runnerType domain.RunnerType, available bool, message, installHint string, models []string) *pb.RunnerStatus {
	return &pb.RunnerStatus{
		RunnerType:      RunnerTypeToProto(runnerType),
		Available:       available,
		Message:         message,
		InstallHint:     installHint,
		SupportedModels: models,
	}
}

// =============================================================================
// STOP ALL RESULT
// =============================================================================

// StopAllResultToProto converts an orchestration StopAllResult to proto StopAllResult.
func StopAllResultToProto(r *StopAllResult) *pb.StopAllResult {
	if r == nil {
		return nil
	}
	failures := make([]*pb.StopFailure, len(r.FailedIDs))
	for i, id := range r.FailedIDs {
		failures[i] = &pb.StopFailure{
			RunId: id,
			Error: "stop failed",
		}
	}
	return &pb.StopAllResult{
		StoppedCount: int32(r.Stopped),
		Failures:     failures,
	}
}

// StopAllResult mirrors orchestration.StopAllResult for import avoidance.
type StopAllResult struct {
	Stopped   int
	Failed    int
	Skipped   int
	FailedIDs []string
}

// =============================================================================
// APPROVE RESULT
// =============================================================================

// ApproveResultToProto converts an orchestration ApproveResult to proto ApproveResult.
func ApproveResultToProto(r *ApproveResult) *pb.ApproveResult {
	if r == nil {
		return nil
	}
	return &pb.ApproveResult{
		Success:      r.Success,
		FilesApplied: int32(r.Applied),
		CommitHash:   r.CommitHash,
		Message:      r.ErrorMsg,
		Remaining:    int32(r.Remaining),
		IsPartial:    r.IsPartial,
	}
}

// ApproveResult mirrors orchestration.ApproveResult for import avoidance.
type ApproveResult struct {
	Success    bool
	Applied    int
	Remaining  int
	IsPartial  bool
	CommitHash string
	AppliedAt  time.Time
	ErrorMsg   string
}

// =============================================================================
// PROBE RESULT
// =============================================================================

// ProbeResultToProto converts an orchestration ProbeResult to proto ProbeResult.
func ProbeResultToProto(r *ProbeResult) *pb.ProbeResult {
	if r == nil {
		return nil
	}
	details := make(map[string]string)
	if r.Response != "" {
		details["response"] = r.Response
	}
	return &pb.ProbeResult{
		Success:   r.Success,
		LatencyMs: r.DurationMs,
		Error:     r.Message,
		Details:   details,
	}
}

// ProbeResult mirrors orchestration.ProbeResult for import avoidance.
type ProbeResult struct {
	RunnerType domain.RunnerType
	Success    bool
	Message    string
	Response   string
	DurationMs int64
}

// =============================================================================
// RUN DIFF
// =============================================================================

// DiffResultToProto converts a sandbox DiffResult to proto RunDiff.
func DiffResultToProto(runID uuid.UUID, r *DiffResult) *pb.RunDiff {
	if r == nil {
		return nil
	}

	// Extract per-file patches from the unified diff.
	patchByPath := splitUnifiedDiff(r.UnifiedDiff)

	files := make([]*pb.FileDiff, len(r.Files))
	for i, f := range r.Files {
		patch := f.Patch
		if patch == "" {
			patch = lookupPatch(patchByPath, f.FilePath)
		}
		files[i] = &pb.FileDiff{
			Id:         UUIDToString(f.ID),
			Path:       f.FilePath,
			ChangeType: string(f.ChangeType),
			Additions:  int32(f.LinesAdded),
			Deletions:  int32(f.LinesRemoved),
			IsBinary:   false,
			Patch:      patch,
		}
	}
	return &pb.RunDiff{
		RunId:       UUIDToString(runID),
		Content:     r.UnifiedDiff,
		Files:       files,
		GeneratedAt: TimestampToProto(r.Generated),
	}
}

// lookupPatch finds a patch for filePath in the map. It first tries an exact
// match, then falls back to a suffix match. This handles the common case where
// the unified diff uses project-root-relative paths (e.g.
// "scenarios/foo/api/main.go") but file metadata uses sandbox-scope-relative
// paths (e.g. "api/main.go").
func lookupPatch(patchByPath map[string]string, filePath string) string {
	if p, ok := patchByPath[filePath]; ok {
		return p
	}
	suffix := "/" + filePath
	for k, v := range patchByPath {
		if strings.HasSuffix(k, suffix) {
			return v
		}
	}
	return ""
}

// splitUnifiedDiff splits a unified diff string into per-file patches.
// It looks for "diff --git a/... b/..." markers and maps each section
// to the file path (the "b/" side).
func splitUnifiedDiff(unified string) map[string]string {
	if unified == "" {
		return nil
	}

	result := make(map[string]string)
	lines := strings.Split(unified, "\n")

	var currentPath string
	var currentStart int
	inSection := false

	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// Flush previous section.
			if inSection && currentPath != "" {
				result[currentPath] = strings.Join(lines[currentStart:i], "\n")
			}
			// Extract the "b/" path from "diff --git a/foo b/foo".
			currentPath = extractDiffPath(line)
			currentStart = i
			inSection = true
		}
	}
	// Flush last section.
	if inSection && currentPath != "" {
		result[currentPath] = strings.Join(lines[currentStart:], "\n")
	}

	return result
}

// extractDiffPath extracts the file path from a "diff --git a/X b/Y" line.
// Returns Y without the "b/" prefix.
func extractDiffPath(line string) string {
	// Format: "diff --git a/path/to/file b/path/to/file"
	idx := strings.LastIndex(line, " b/")
	if idx < 0 {
		return ""
	}
	return line[idx+3:]
}

// DiffResult mirrors sandbox.DiffResult for import avoidance.
type DiffResult struct {
	SandboxID    uuid.UUID
	Files        []FileChange
	UnifiedDiff  string
	Generated    time.Time
	Stats        DiffStats
	ArchiveState string
}

// DiffStats mirrors the value nested in sandbox.DiffResult without importing
// the sandbox package and recreating the cycle this conversion seam avoids.
type DiffStats struct {
	FilesChanged  int
	FilesAdded    int
	FilesModified int
	FilesDeleted  int
}

// FileChange mirrors sandbox.FileChange for import avoidance.
type FileChange struct {
	ID           uuid.UUID
	FilePath     string
	ChangeType   string
	FileSize     int64
	LinesAdded   int
	LinesRemoved int
	Patch        string
}

// =============================================================================
// ORCHESTRATION RUNNER STATUS
// =============================================================================

// OrchestratorRunnerStatusToProto converts orchestration RunnerStatus to proto.
func OrchestratorRunnerStatusToProto(r *OrchestratorRunnerStatus) *pb.RunnerStatus {
	if r == nil {
		return nil
	}
	return &pb.RunnerStatus{
		RunnerType:  RunnerTypeToProto(r.Type),
		Available:   r.Available,
		Message:     r.Message,
		InstallHint: "",
		Capabilities: &pb.RunnerCapabilities{
			SpawnCapabilities:         SpawnCapabilitiesToProto(r.Capabilities.SpawnCapabilities),
			SupportsStreaming:         r.Capabilities.SupportsStreaming,
			SupportsMessages:          r.Capabilities.SupportsMessages,
			SupportsToolEvents:        r.Capabilities.SupportsToolEvents,
			SupportsCostTracking:      r.Capabilities.SupportsCostTracking,
			SupportsCancellation:      r.Capabilities.SupportsCancellation,
			SupportsContinuation:      r.Capabilities.SupportsContinuation,
			SupportsWarmIteration:     r.Capabilities.SupportsWarmIteration,
			SupportsSessionCompaction: r.Capabilities.SupportsSessionCompaction,
			SupportsImageAttachments:  r.Capabilities.SupportsImageAttachments,
			SupportsEffort:            r.Capabilities.SupportsEffort,
			EffortMappings:            r.Capabilities.EffortMappings,
			EffortModelSpecific:       r.Capabilities.EffortModelSpecific,
			SupportsRunnerDefault:     r.Capabilities.SupportsRunnerDefault,
			DynamicModelPrefixes:      r.Capabilities.DynamicModelPrefixes,
			MaxTurns:                  int32(r.Capabilities.MaxTurns),
			SupportedFeatures:         r.Capabilities.SupportedFeatures,
			AllowedExtraFlags:         r.Capabilities.AllowedExtraFlags,
			SupportsToolRestriction:   r.Capabilities.SupportsToolRestriction,
			ToolRestrictionMappings:   r.Capabilities.ToolRestrictionMappings,
		},
		SupportedModels: r.Capabilities.SupportedModels,
	}
}

func SpawnCapabilitiesToProto(capabilities []SpawnCapability) []*pb.SpawnCapability {
	result := make([]*pb.SpawnCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		result = append(result, &pb.SpawnCapability{
			ExecutionMode:   capability.ExecutionMode,
			SandboxModes:    capability.SandboxModes,
			NativeObjective: capability.NativeObjective,
		})
	}
	return result
}

func spawnCapabilitiesFromProto(capabilities []*pb.SpawnCapability) []SpawnCapability {
	result := make([]SpawnCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability == nil {
			continue
		}
		result = append(result, SpawnCapability{
			ExecutionMode:   capability.ExecutionMode,
			SandboxModes:    capability.SandboxModes,
			NativeObjective: capability.NativeObjective,
		})
	}
	return result
}

// RunnerCapabilitiesFromProto reconstructs the import-avoiding capability
// mirror. Supported models live on RunnerStatus in the wire contract, so the
// caller supplies that adjacent field for a lossless round trip.
func RunnerCapabilitiesFromProto(capabilities *pb.RunnerCapabilities, supportedModels []string) RunnerCapabilities {
	if capabilities == nil {
		return RunnerCapabilities{SupportedModels: supportedModels}
	}
	return RunnerCapabilities{
		SpawnCapabilities:         spawnCapabilitiesFromProto(capabilities.SpawnCapabilities),
		SupportsMessages:          capabilities.SupportsMessages,
		SupportsToolEvents:        capabilities.SupportsToolEvents,
		SupportsCostTracking:      capabilities.SupportsCostTracking,
		SupportsStreaming:         capabilities.SupportsStreaming,
		SupportsCancellation:      capabilities.SupportsCancellation,
		SupportsContinuation:      capabilities.SupportsContinuation,
		SupportsWarmIteration:     capabilities.SupportsWarmIteration,
		SupportsSessionCompaction: capabilities.SupportsSessionCompaction,
		SupportsImageAttachments:  capabilities.SupportsImageAttachments,
		SupportsToolRestriction:   capabilities.SupportsToolRestriction,
		ToolRestrictionMappings:   capabilities.ToolRestrictionMappings,
		SupportsEffort:            capabilities.SupportsEffort,
		EffortMappings:            capabilities.EffortMappings,
		EffortModelSpecific:       capabilities.EffortModelSpecific,
		MaxTurns:                  int(capabilities.MaxTurns),
		SupportedModels:           supportedModels,
		SupportsRunnerDefault:     capabilities.SupportsRunnerDefault,
		DynamicModelPrefixes:      capabilities.DynamicModelPrefixes,
		SupportedFeatures:         capabilities.SupportedFeatures,
		AllowedExtraFlags:         capabilities.AllowedExtraFlags,
	}
}

// OrchestratorRunnerStatusesToProto converts a slice of runner statuses.
func OrchestratorRunnerStatusesToProto(statuses []*OrchestratorRunnerStatus) []*pb.RunnerStatus {
	result := make([]*pb.RunnerStatus, len(statuses))
	for i, s := range statuses {
		result[i] = OrchestratorRunnerStatusToProto(s)
	}
	return result
}

// OrchestratorRunnerStatus mirrors orchestration.RunnerStatus for import avoidance.
type OrchestratorRunnerStatus struct {
	Type         domain.RunnerType
	Available    bool
	Message      string
	Capabilities RunnerCapabilities
}

// RunnerCapabilities mirrors runner.Capabilities for import avoidance.
type RunnerCapabilities struct {
	SpawnCapabilities         []SpawnCapability
	SupportsMessages          bool
	SupportsToolEvents        bool
	SupportsCostTracking      bool
	SupportsStreaming         bool
	SupportsCancellation      bool
	SupportsContinuation      bool
	SupportsWarmIteration     bool
	SupportsSessionCompaction bool
	SupportsImageAttachments  bool
	SupportsToolRestriction   bool
	ToolRestrictionMappings   map[string]string
	SupportsEffort            bool
	EffortMappings            map[string]string
	EffortModelSpecific       bool
	MaxTurns                  int
	SupportedModels           []string
	SupportsRunnerDefault     bool
	DynamicModelPrefixes      []string
	SupportedFeatures         []string
	AllowedExtraFlags         []string
}

type SpawnCapability struct {
	ExecutionMode   string
	SandboxModes    []string
	NativeObjective bool
}
