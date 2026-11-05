package cloudtarget

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// DataBackupRequest captures one recovery point on this target.
type DataBackupRequest struct {
	Effect EffectRequest
	// RecoveryPointID defaults to the operation id + step so one (operation,
	// step) maps to exactly one recovery point.
	RecoveryPointID string
	// RecoveryPointDir overrides the canonical location beneath the
	// deployment directory (used when the cloud side pins a workdir path).
	RecoveryPointDir string
	Bindings         []recoverypoint.Binding
	Refs             recoverypoint.Refs
	KeyRef           string
	Provider         string
	ProviderRef      string
	RetentionPolicy  string
	MigrationPosture string
	Deps             DataDeps
}

// ParseDataBindings decodes the --binding JSON values (each one binding
// object) into engine bindings.
func ParseDataBindings(values []string) ([]recoverypoint.Binding, error) {
	var bindings []recoverypoint.Binding
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "[") {
			var many []recoverypoint.Binding
			if err := json.Unmarshal([]byte(raw), &many); err != nil {
				return nil, refuse(CodeInvalidArgument, "parse bindings: %v", err)
			}
			bindings = append(bindings, many...)
			continue
		}
		var one recoverypoint.Binding
		if err := json.Unmarshal([]byte(raw), &one); err != nil {
			return nil, refuse(CodeInvalidArgument, "parse binding: %v", err)
		}
		bindings = append(bindings, one)
	}
	for _, b := range bindings {
		if err := b.Validate(); err != nil {
			return nil, dataError(err)
		}
	}
	return bindings, nil
}

// DataBackup enters each binding's consistency boundary, captures it through
// its provider, seals the artifact under the referenced key and writes the
// manifest; the receipt records the recovery point identity and checksums.
func (s *Store) DataBackup(ctx context.Context, req DataBackupRequest) (recoverypoint.Manifest, EffectResult, error) {
	recoveryPointID := req.RecoveryPointID
	if recoveryPointID == "" {
		recoveryPointID = req.Effect.OperationID + "-" + req.Effect.Step
	}
	dir := req.RecoveryPointDir
	if dir == "" {
		resolved, err := s.RecoveryPointDir(req.Effect.DeploymentID, recoveryPointID)
		if err != nil {
			return recoverypoint.Manifest{}, EffectResult{}, err
		}
		dir = resolved
	}
	req.Effect.Verb = "data backup"
	req.Effect.Input = map[string]any{
		"recovery_point_id": recoveryPointID, "recovery_point_dir": dir, "bindings": req.Bindings,
		"refs": req.Refs, "key_ref": req.KeyRef, "provider": req.Provider, "provider_ref": req.ProviderRef,
		"retention_policy": req.RetentionPolicy, "migration_posture": req.MigrationPosture,
	}
	var manifest recoverypoint.Manifest
	result, err := s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		captured, err := recoverypoint.Capture(ctx, recoverypoint.CaptureRequest{
			DeploymentID: req.Effect.DeploymentID, RecoveryPointID: recoveryPointID, Dir: dir,
			Bindings: req.Bindings, Refs: req.Refs, KeyRef: req.KeyRef,
			Keys: req.Deps.keys(), Sealer: req.Deps.Sealer, Providers: req.Deps.providers(),
			Boundary: recoverypoint.ApplicationHooks{Runner: req.Deps.runner()},
			Provider: req.Provider, ProviderRef: req.ProviderRef, RetentionPolicy: req.RetentionPolicy,
			MigrationPosture: req.MigrationPosture, Now: s.now,
		})
		if err != nil {
			return map[string]any{"recovery_point_id": recoveryPointID, "recovery_point_dir": dir}, OutcomeFailed, dataError(err)
		}
		manifest = captured
		return map[string]any{
			"recovery_point_id": captured.ID, "recovery_point_dir": dir, "digest": captured.Digest,
			"captured_at": captured.CapturedAt, "bindings": captured.BindingIDs(), "checksums": captured.Checksums,
			"consistency": captured.Consistency, "refs": captured.Refs, "key_ref": captured.KeyRef,
			"encrypted": captured.Encrypted, "migration_posture": captured.MigrationPosture,
		}, OutcomeSucceeded, nil
	})
	if err != nil {
		return recoverypoint.Manifest{}, result, err
	}
	if result.Replayed {
		if replayed, readErr := recoverypoint.ReadManifest(dir); readErr == nil {
			manifest = replayed
		}
	}
	return manifest, result, nil
}

// dataError maps an engine error onto the verb error model: corruption,
// missing keys, unclean targets and missing providers are refusals (exit 2);
// everything else is a failure (exit 1).
func dataError(err error) error {
	if err == nil {
		return nil
	}
	typed := recoverypoint.AsError(err, recoverypoint.CodeCaptureFailed)
	exit := ExitFailed
	switch typed.Code {
	case recoverypoint.CodeRecoveryPointCorrupt, recoverypoint.CodeRecoveryKeyMissing, recoverypoint.CodeRestoreTargetNotClean,
		recoverypoint.CodeProviderUnavailable, recoverypoint.CodeInvalidArgument, recoverypoint.CodeRecoveryPointExists:
		exit = ExitRefused
	}
	details := typed.Details
	if typed.Blocker != "" {
		if details == nil {
			details = map[string]any{}
		}
		details["blocker"] = typed.Blocker
	}
	return &Error{Code: typed.Code, Message: typed.Message, Exit: exit, Details: details}
}
