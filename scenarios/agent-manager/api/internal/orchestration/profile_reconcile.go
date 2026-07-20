package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	profileReconcileModeCreateOnly         = "create_only"
	profileReconcileModeUpdateIfUnmodified = "update_if_unmodified"
	profileReconcileModeForce              = "force"
)

type scenarioServiceProfileConfig struct {
	Dependencies struct {
		Scenarios map[string]struct {
			Enabled *bool           `json:"enabled"`
			Config  json.RawMessage `json:"config"`
		} `json:"scenarios"`
	} `json:"dependencies"`
}

// ReconcileScenarioProfiles is a thin projection over the unified declaration
// reconcile that returns only the profile-kind results. Many Go consumer
// clients call this RPC name, so it keeps working while reading exclusively
// from the new .vrooli/agent-manager/ declaration block.
func (o *Orchestrator) ReconcileScenarioProfiles(ctx context.Context, req ReconcileScenarioProfilesRequest) (*ReconcileScenarioProfilesResult, error) {
	res, err := o.ReconcileScenarioDeclarations(ctx, ReconcileScenarioDeclarationsRequest{Scenario: req.Scenario, DryRun: req.DryRun})
	if err != nil {
		return nil, err
	}
	result := &ReconcileScenarioProfilesResult{Scenario: res.Scenario, DryRun: res.DryRun}
	for _, item := range res.ProfileResults {
		result.add(item)
	}
	return result, nil
}

func (o *Orchestrator) reconcileProfileSource(ctx context.Context, scenario, scenarioRoot, source, mode string, dryRun bool) ProfileReconcileResult {
	source = strings.TrimSpace(source)
	item := ProfileReconcileResult{SourcePath: source}
	sourcePath, err := resolveProfileSourcePath(scenarioRoot, source)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	item.SourcePath = source

	data, info, err := readProfileSource(sourcePath)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	sum := sha256.Sum256(data)
	item.SourceHash = hex.EncodeToString(sum[:])

	profile, err := parseSourceProfile(data)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	item.Diagnostics = profileRestrictionDiagnostics(profile)
	item.ProfileKey = profile.ProfileKey

	if err := enforceProfileOwnership(scenario, profile.ProfileKey); err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	profile.OwnerScenario = scenario
	profile.SourcePath = source
	profile.SourceHash = item.SourceHash
	profile.LastAppliedHash = item.SourceHash
	profile.SourceUpdatedAt = info.ModTime()
	profile.LocalOverride = false
	if strings.TrimSpace(profile.CreatedBy) == "" {
		profile.CreatedBy = scenario
	}

	existing, err := o.profiles.GetByKey(ctx, profile.ProfileKey)
	if err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	if existing == nil {
		if dryRun {
			item.ProfileID = uuid.NewString()
			item.Status = ProfileReconcileStatusCreated
			item.Message = "would create profile"
			return item
		}
		profile.ID = uuid.New()
		now := time.Now()
		profile.CreatedAt = now
		profile.UpdatedAt = now
		if err := normalizeProfileInput(profile); err != nil {
			item.Status = ProfileReconcileStatusFailedValidation
			item.Message = err.Error()
			return item
		}
		if err := o.profiles.Create(ctx, profile); err != nil {
			item.Status = ProfileReconcileStatusFailedValidation
			item.Message = err.Error()
			return item
		}
		item.ProfileID = profile.ID.String()
		item.Status = ProfileReconcileStatusCreated
		return item
	}

	item.ProfileID = existing.ID.String()
	if mode == profileReconcileModeCreateOnly {
		item.Status = ProfileReconcileStatusSkipped
		item.Message = "profile exists and mode is create_only"
		return item
	}
	if existing.LocalOverride && mode != profileReconcileModeForce {
		item.Status = ProfileReconcileStatusConflictedLocalOverride
		item.Message = "profile has local overrides; use force mode to overwrite"
		return item
	}
	if existing.SourceHash == item.SourceHash && existing.LastAppliedHash == item.SourceHash && !existing.LocalOverride {
		item.Status = ProfileReconcileStatusUnchanged
		return item
	}

	profile.ID = existing.ID
	profile.CreatedAt = existing.CreatedAt
	profile.UpdatedAt = time.Now()
	if dryRun {
		item.Status = ProfileReconcileStatusUpdated
		item.Message = "would update profile"
		return item
	}
	if err := normalizeProfileInput(profile); err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	if err := o.profiles.Update(ctx, profile); err != nil {
		item.Status = ProfileReconcileStatusFailedValidation
		item.Message = err.Error()
		return item
	}
	item.Status = ProfileReconcileStatusUpdated
	return item
}

func profileRestrictionDiagnostics(profile *domain.AgentProfile) []domain.WorkflowDiagnostic {
	if profile == nil || !profile.SkipPermissionPrompt || len(profile.AllowedTools) == 0 {
		return nil
	}
	return []domain.WorkflowDiagnostic{{
		Code:     "tool_restriction_skip_permissions",
		Path:     "skipPermissionPrompt",
		Message:  "skipPermissionPrompt:true contradicts allowedTools because Claude Code bypasses its permission allowlist",
		Severity: domain.DiagnosticSeverityWarning,
	}}
}

func resolveProfileSourcePath(scenarioRoot, source string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("profile source path is required")
	}
	if filepath.IsAbs(source) {
		return "", fmt.Errorf("profile source path must be relative to scenario root")
	}
	cleanSlash := filepath.ToSlash(filepath.Clean(source))
	for _, part := range strings.Split(cleanSlash, "/") {
		if part == ".." {
			return "", fmt.Errorf("profile source path must not contain parent traversal")
		}
	}
	root, err := filepath.EvalSymlinks(scenarioRoot)
	if err != nil {
		return "", fmt.Errorf("resolve scenario root: %w", err)
	}
	candidate := filepath.Join(root, filepath.FromSlash(cleanSlash))
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve profile source: %w", err)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", fmt.Errorf("check profile source containment: %w", err)
	}
	if rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", fmt.Errorf("profile source path escapes scenario root")
	}
	return resolved, nil
}

func readProfileSource(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("stat profile source: %w", err)
	}
	if info.IsDir() {
		return nil, nil, fmt.Errorf("profile source must be a file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read profile source: %w", err)
	}
	return data, info, nil
}

func parseSourceProfile(data []byte) (*domain.AgentProfile, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse profile source JSON: %w", err)
	}
	for _, forbidden := range []string{
		"id", "createdAt", "created_at", "updatedAt", "updated_at",
		"ownerScenario", "owner_scenario", "sourcePath", "source_path",
		"sourceHash", "source_hash", "lastAppliedHash", "last_applied_hash",
		"sourceUpdatedAt", "source_updated_at", "localOverride", "local_override",
	} {
		if _, ok := raw[forbidden]; ok {
			return nil, fmt.Errorf("profile source must not set runtime field %q", forbidden)
		}
	}
	// schemaVersion discriminates the file kind in the unified declaration
	// layer; it is a file-format marker with no AgentProfile field, so strip it
	// before the strict proto unmarshal (DiscardUnknown is false).
	if _, ok := raw["schemaVersion"]; ok {
		delete(raw, "schemaVersion")
		stripped, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("strip profile source schemaVersion: %w", err)
		}
		data = stripped
	}
	var pb domainpb.AgentProfile
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("parse profile source proto JSON: %w", err)
	}
	profile := protoconv.AgentProfileFromProto(&pb)
	if err := normalizeProfileInput(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func enforceProfileOwnership(scenario, profileKey string) error {
	if strings.TrimSpace(profileKey) == "" {
		return fmt.Errorf("profileKey is required")
	}
	prefix := scenario + "/"
	if !strings.HasPrefix(profileKey, prefix) {
		return fmt.Errorf("profileKey %q must start with %q", profileKey, prefix)
	}
	return nil
}

func validProfileReconcileMode(mode string) bool {
	switch mode {
	case profileReconcileModeCreateOnly, profileReconcileModeUpdateIfUnmodified, profileReconcileModeForce:
		return true
	default:
		return false
	}
}

func (r *ReconcileScenarioProfilesResult) add(item ProfileReconcileResult) {
	r.Results = append(r.Results, item)
	switch item.Status {
	case ProfileReconcileStatusCreated:
		r.Created++
	case ProfileReconcileStatusUpdated:
		r.Updated++
	case ProfileReconcileStatusUnchanged:
		r.Unchanged++
	case ProfileReconcileStatusSkipped:
		r.Skipped++
	case ProfileReconcileStatusConflictedLocalOverride:
		r.Conflicted++
	case ProfileReconcileStatusFailedValidation:
		r.Failed++
	}
}
