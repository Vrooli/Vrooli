package orchestration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	repocontract "github.com/vrooli/repo-contract-go"
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

type profileSourcesConfig struct {
	Profiles struct {
		Reconcile *bool    `json:"reconcile"`
		Mode      string   `json:"mode"`
		Sources   []string `json:"sources"`
	} `json:"profiles"`
	// Workflows is parsed by the workflow catalog reconciler. Keeping it as raw
	// JSON lets each subsystem remain strict about its own versioned config.
	Workflows json.RawMessage `json:"workflows,omitempty"`
}

// ReconcileScenarioProfiles reads a scenario-owned profile declaration and
// reconciles every referenced profile source into agent-manager's runtime DB.
func (o *Orchestrator) ReconcileScenarioProfiles(ctx context.Context, req ReconcileScenarioProfilesRequest) (*ReconcileScenarioProfilesResult, error) {
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		return nil, domain.NewValidationErrorWithHint("scenario", "field is required", "Provide the owning scenario slug")
	}

	contract, repoRoot, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, domain.NewConfigInvalidError("repoContract", "failed to resolve repository contract", err)
	}
	scenarioRoot, err := contract.ScenarioRoot(repoRoot, scenario)
	if err != nil {
		return nil, domain.NewValidationErrorWithHint("scenario", "invalid scenario slug", err.Error())
	}
	servicePath, err := contract.ScenarioFile(repoRoot, scenario, "service")
	if err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to resolve scenario service manifest", err)
	}

	cfg, err := readScenarioProfileConfig(servicePath)
	if err != nil {
		return nil, err
	}
	result := &ReconcileScenarioProfilesResult{
		Scenario: scenario,
		DryRun:   req.DryRun,
	}

	reconcile := true
	if cfg.Profiles.Reconcile != nil {
		reconcile = *cfg.Profiles.Reconcile
	}
	mode := strings.TrimSpace(cfg.Profiles.Mode)
	if mode == "" {
		mode = profileReconcileModeUpdateIfUnmodified
	}
	if !validProfileReconcileMode(mode) {
		return nil, domain.NewValidationErrorWithHint("config.profiles.mode", "invalid reconcile mode",
			"valid values: create_only, update_if_unmodified, force")
	}
	if !reconcile {
		for _, src := range cfg.Profiles.Sources {
			result.add(ProfileReconcileResult{
				SourcePath: src,
				Status:     ProfileReconcileStatusSkipped,
				Message:    "profile reconciliation disabled by manifest config",
			})
		}
		return result, nil
	}
	if len(cfg.Profiles.Sources) == 0 {
		result.add(ProfileReconcileResult{
			Status:  ProfileReconcileStatusSkipped,
			Message: "no profile sources declared",
		})
		return result, nil
	}

	for _, source := range cfg.Profiles.Sources {
		item := o.reconcileProfileSource(ctx, scenario, scenarioRoot, source, mode, req.DryRun)
		result.add(item)
	}
	return result, nil
}

func readScenarioProfileConfig(servicePath string) (*profileSourcesConfig, error) {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to read scenario service manifest", err)
	}
	var manifest scenarioServiceProfileConfig
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, domain.NewConfigInvalidError("serviceManifest", "failed to parse scenario service manifest", err)
	}
	dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
	if !ok {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager", "dependency is required",
			"Declare agent-manager under dependencies.scenarios with config.profiles.sources")
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.enabled", "dependency must be enabled",
			"Enable agent-manager before reconciling its scenario-owned profiles")
	}
	if len(dep.Config) == 0 {
		return &profileSourcesConfig{}, nil
	}
	var configObject map[string]json.RawMessage
	if err := json.Unmarshal(dep.Config, &configObject); err != nil || configObject == nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "must be a JSON object containing profiles", err)
	}
	profilesRaw, hasProfiles := configObject["profiles"]
	if !hasProfiles || string(profilesRaw) == "null" {
		return nil, domain.NewValidationErrorWithHint("dependencies.scenarios.agent-manager.config.profiles", "field is required",
			"Declare profiles.reconcile, profiles.mode, and profiles.sources or omit config when no scenario-owned profile is needed")
	}
	var cfg profileSourcesConfig
	decoder := json.NewDecoder(bytes.NewReader(dep.Config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "failed to parse profile config", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, domain.NewConfigInvalidError("dependencies.scenarios.agent-manager.config", "failed to parse profile config", err)
	}
	if cfg.Profiles.Reconcile == nil {
		return nil, domain.NewValidationErrorWithHint("config.profiles.reconcile", "field is required",
			"Set whether the declared scenario-owned profiles reconcile")
	}
	if !validProfileReconcileMode(strings.TrimSpace(cfg.Profiles.Mode)) {
		return nil, domain.NewValidationErrorWithHint("config.profiles.mode", "invalid reconcile mode",
			"valid values: create_only, update_if_unmodified, force")
	}
	if len(cfg.Profiles.Sources) == 0 {
		return nil, domain.NewValidationErrorWithHint("config.profiles.sources", "must declare at least one source",
			"Omit config entirely when the scenario uses only direct portable role requests")
	}
	seen := make(map[string]struct{}, len(cfg.Profiles.Sources))
	for _, source := range cfg.Profiles.Sources {
		key := strings.TrimSpace(source)
		if key == "" {
			return nil, domain.NewValidationErrorWithHint("config.profiles.sources", "profile source must not be empty", "Declare a target-relative profile JSON file")
		}
		if _, exists := seen[key]; exists {
			return nil, domain.NewValidationErrorWithHint("config.profiles.sources", "duplicate profile source", "Declare each scenario-owned source once")
		}
		seen[key] = struct{}{}
	}
	return &cfg, nil
}

// ValidateScenarioProfileConfig validates Agent Manager's declared dependency
// configuration without touching the profile repository or target files. It is
// shared by read-only conformance and mutating reconciliation to prevent the
// two surfaces from accepting different manifests.
func ValidateScenarioProfileConfig(servicePath string) error {
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return err
	}
	var manifest scenarioServiceProfileConfig
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
	if !ok || len(dep.Config) == 0 {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(dep.Config, &object); err != nil {
		return err
	}
	if _, hasProfiles := object["profiles"]; !hasProfiles {
		return nil
	}
	_, err = readScenarioProfileConfig(servicePath)
	return err
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
