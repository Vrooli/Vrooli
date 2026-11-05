package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/capacity"
)

const (
	lamaBuildModule         = "image_tools_sidecar.lama_export"
	lamaBuildTimeout        = 45 * time.Minute
	lamaBuildRAMBytes       = int64(16 << 30)
	lamaBuildCPUMillicores  = int64(2000)
	lamaBuildTorchThreads   = "2"
	lamaUtilsSourceSHA256   = "0a399137ac692a8c0eac6b8f1befacdd9d8f3443397d8a0b000845010c2b0cfc"
	lamaUtilsPreparedSHA256 = "929aede2965c0a5d060e0c4809c19bf2e88c7748239bf7ab59dca1cdac4388a2"
	lamaSourcePreparation   = "replace pinned training-only seed_everything import with equivalent local seed helper"
	lamaCheckpointSHA256    = "12cb5eb5d515400f262b48d4e7c1994183e170aedb4b86699d9a55b770038e16"
	lamaCheckpointKeyPrefix = "model.generator."
	lamaReferenceONNXSHA256 = "1faef5301d78db7dda502fe59966957ec4b79dd64e16f03ed96913c7a4eb68d6"
	lamaReferenceTolerance  = 0.5
)

var modelBuildLocks sync.Map // model ID -> *sync.Mutex; serializes recovery and build per model.

func lockModelBuild(modelID string) func() {
	lockValue, _ := modelBuildLocks.LoadOrStore(modelID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// BuildConfig supplies only the interpreter side of the one closed export
// profile. The module and its accepted arguments are fixed in this package.
type BuildConfig struct {
	Python        string
	PythonPath    string
	ReferenceONNX string
	Run           func(context.Context, string, []string, []string) ([]byte, error)
	Capacity      BuildCapacityBrokerFactory
	Operations    BuildOperationOwner
}

// BuildOperationOwner provides the native cgroup-backed lifecycle for the
// exporter child. The intent is persisted by InstallStore before Start.
type BuildOperationOwner interface {
	NewIntent(string, string, platform.OperationSpec) (platform.OperationIntent, error)
	Start(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error)
	Lookup(context.Context, platform.OperationIntent) (platform.OperationRef, error)
	Inspect(context.Context, platform.OperationRef) (platform.OperationObservation, error)
	Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error)
	Release(context.Context, platform.OperationRef) error
}

type buildOperationRecord struct {
	JobID      string
	ModelID    string
	Intent     platform.OperationIntent
	State      string
	Result     string
	ExitStatus int
}

func (s *InstallStore) saveBuildOperation(ctx context.Context, record buildOperationRecord) error {
	intentJSON, err := json.Marshal(record.Intent)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO model_build_operation(job_id,model_id,intent_json,state,updated_at)
VALUES(?,?,?,'starting',?) ON CONFLICT(job_id) DO NOTHING`, record.JobID, record.ModelID, string(intentJSON), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("models: persist build operation intent: %w", err)
	}
	existing, ok, err := s.getBuildOperation(ctx, record.JobID)
	if err != nil {
		return err
	}
	if !ok || existing.ModelID != record.ModelID || existing.Intent != record.Intent {
		return errors.New("models: build operation job identity already has a different intent")
	}
	return nil
}

func (s *InstallStore) getBuildOperation(ctx context.Context, jobID string) (buildOperationRecord, bool, error) {
	var record buildOperationRecord
	var intentJSON string
	if err := s.db.QueryRowContext(ctx, `SELECT job_id,model_id,intent_json,state,result,exit_status
FROM model_build_operation WHERE job_id=?`, jobID).Scan(&record.JobID, &record.ModelID, &intentJSON, &record.State, &record.Result, &record.ExitStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return buildOperationRecord{}, false, nil
		}
		return buildOperationRecord{}, false, fmt.Errorf("models: load build operation: %w", err)
	}
	if err := json.Unmarshal([]byte(intentJSON), &record.Intent); err != nil {
		return buildOperationRecord{}, false, fmt.Errorf("models: decode persisted build operation intent: %w", err)
	}
	return record, true, nil
}

func (s *InstallStore) updateBuildOperation(ctx context.Context, record buildOperationRecord) error {
	intentJSON, err := json.Marshal(record.Intent)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE model_build_operation SET state=?,result=?,exit_status=?,updated_at=?
WHERE job_id=? AND model_id=? AND intent_json=?`, record.State, record.Result, record.ExitStatus,
		time.Now().UTC().Format(time.RFC3339Nano), record.JobID, record.ModelID, string(intentJSON))
	if err != nil {
		return fmt.Errorf("models: update build operation state: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("models: persisted build operation identity changed")
	}
	return nil
}

func (s *InstallStore) pendingBuildOperation(ctx context.Context, modelID string) (buildOperationRecord, bool, error) {
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT job_id FROM model_build_operation
WHERE model_id=? AND state IN ('starting','running','outcome_unknown') ORDER BY updated_at LIMIT 1`, modelID).Scan(&jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return buildOperationRecord{}, false, nil
		}
		return buildOperationRecord{}, false, err
	}
	return s.getBuildOperation(ctx, jobID)
}

type platformBuildOperations struct{}

func (platformBuildOperations) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (platformBuildOperations) Start(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	return platform.StartOwnedOperation(ctx, intent, spec)
}
func (platformBuildOperations) Lookup(ctx context.Context, intent platform.OperationIntent) (platform.OperationRef, error) {
	return platform.LookupOwnedOperation(ctx, intent)
}
func (platformBuildOperations) Inspect(ctx context.Context, ref platform.OperationRef) (platform.OperationObservation, error) {
	return platform.InspectOwnedOperation(ctx, ref)
}
func (platformBuildOperations) Stop(ctx context.Context, ref platform.OperationRef) (platform.OperationStopReceipt, error) {
	return platform.StopOwnedOperation(ctx, ref)
}
func (platformBuildOperations) Release(ctx context.Context, ref platform.OperationRef) error {
	return platform.ReleaseOwnedOperation(ctx, ref)
}

// NewPlatformBuildOperationOwner returns the canonical cgroup-backed owner
// used to bound and reap exporter processes.
func NewPlatformBuildOperationOwner() BuildOperationOwner { return platformBuildOperations{} }

func (in *Installer) runOwnedBuild(ctx context.Context, jobID, modelID, dir, python string, args, extraEnv []string) ([]byte, error, bool) {
	owner := in.Build.Operations
	if owner == nil || in.State == nil {
		return nil, errors.New("models: native build owner and durable install store are required"), false
	}
	if pending, ok, err := in.State.pendingBuildOperation(ctx, modelID); err != nil {
		return nil, err, false
	} else if ok && pending.JobID != jobID {
		return nil, fmt.Errorf("models: build operation %s for %s remains unreconciled in state %s", pending.JobID, modelID, pending.State), false
	}
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(extraEnv))
	spec := platform.OperationSpec{
		Path: python, Args: append([]string(nil), args...), Env: []string(env), Dir: dir,
		// CPUWeight is a relative systemd scheduling weight, not a quota. The
		// shared broker separately reserves 2000 millicores, while the exporter
		// environment caps its PyTorch/BLAS worker pools at two threads.
		Limits: platform.OperationLimits{MemoryMaxBytes: lamaBuildRAMBytes, TasksMax: 4096, CPUWeight: 100, WallTime: lamaBuildTimeout},
	}
	record, exists, err := in.State.getBuildOperation(ctx, jobID)
	if err != nil {
		return nil, err, false
	}
	if exists {
		if record.ModelID != modelID {
			return nil, errors.New("models: build job is already bound to another model"), false
		}
	} else {
		intent, intentErr := owner.NewIntent("image-tools-model-build", jobID, spec)
		if intentErr != nil {
			return nil, fmt.Errorf("models: create native build intent: %w", intentErr), false
		}
		record = buildOperationRecord{JobID: jobID, ModelID: modelID, Intent: intent, State: "starting"}
		if err := in.State.saveBuildOperation(ctx, record); err != nil {
			return nil, err, false
		}
	}
	ref, err := owner.Lookup(ctx, record.Intent)
	if errors.Is(err, platform.ErrOperationNotFound) || errors.Is(err, platform.ErrOperationOutcomeLost) {
		if exists || record.State != "starting" {
			return nil, fmt.Errorf("models: persisted exporter outcome is unavailable; refusing redispatch: %w", err), false
		}
		started, startErr := owner.Start(ctx, record.Intent, spec)
		if startErr != nil {
			ref, err = owner.Lookup(ctx, record.Intent)
			if err != nil {
				persistErr := in.State.updateBuildOperation(context.WithoutCancel(ctx), buildOperationRecord{JobID: jobID, ModelID: modelID, Intent: record.Intent, State: "outcome_unknown", Result: errors.Join(startErr, err).Error()})
				return nil, errors.Join(fmt.Errorf("models: start native exporter; dispatch outcome unknown, RAM claim retained until expiry: %w", startErr), err, persistErr), true
			}
		} else if started != nil && started.Ref.NativeInvocationID != "" {
			ref = started.Ref
		} else {
			ref, err = owner.Lookup(ctx, record.Intent)
			if err != nil {
				persistErr := in.State.updateBuildOperation(context.WithoutCancel(ctx), buildOperationRecord{JobID: jobID, ModelID: modelID, Intent: record.Intent, State: "outcome_unknown", Result: err.Error()})
				return nil, errors.Join(fmt.Errorf("models: native exporter start returned no reference; RAM claim retained until expiry"), err, persistErr), true
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("models: recover native exporter: %w", err), false
	}
	if err := in.State.updateBuildOperation(ctx, buildOperationRecord{JobID: jobID, ModelID: modelID, Intent: record.Intent, State: "running"}); err != nil {
		cleanupErr, retain := in.cleanupOwnedBuild(record, ref, "outcome_unknown")
		return nil, errors.Join(fmt.Errorf("models: persist running exporter state: %w", err), cleanupErr), retain
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			cleanupErr, retain := in.cleanupOwnedBuild(record, ref, "cancelled")
			return nil, errors.Join(ctx.Err(), cleanupErr), retain
		}
		observation, inspectErr := owner.Inspect(ctx, ref)
		if inspectErr != nil {
			cleanupErr, retain := in.cleanupOwnedBuild(record, ref, "outcome_unknown")
			return nil, errors.Join(fmt.Errorf("models: inspect native exporter: %w", inspectErr), cleanupErr), retain
		}
		if observation.State == platform.OperationStoppedOutcomeKnown && observation.ExitKnown && !observation.Populated {
			if err := in.State.updateBuildOperation(ctx, buildOperationRecord{JobID: jobID, ModelID: modelID, Intent: record.Intent, State: "terminal", Result: observation.Result, ExitStatus: observation.ExecMainStatus}); err != nil {
				cleanupErr, retain := in.cleanupOwnedBuild(record, ref, "outcome_unknown")
				return nil, errors.Join(err, cleanupErr), retain
			}
			if releaseErr := owner.Release(ctx, ref); releaseErr != nil {
				return nil, fmt.Errorf("models: release terminal native exporter: %w", releaseErr), false
			}
			if observation.Result != "success" || observation.ExecMainStatus != 0 {
				return nil, fmt.Errorf("native exporter exited with result=%s status=%d", observation.Result, observation.ExecMainStatus), false
			}
			return nil, nil, false
		}
		<-ticker.C
	}
}

// recoverPendingBuild reconciles persisted operations from an earlier API/job
// lifetime before a new RAM lease is acquired or a replacement build is started.
func (in *Installer) recoverPendingBuild(ctx context.Context, modelID string) error {
	if in.Build == nil || in.Build.Operations == nil || in.State == nil {
		return nil
	}
	record, ok, err := in.State.pendingBuildOperation(ctx, modelID)
	if err != nil || !ok {
		return err
	}
	owner := in.Build.Operations
	ref, err := owner.Lookup(ctx, record.Intent)
	if errors.Is(err, platform.ErrOperationNotFound) || errors.Is(err, platform.ErrOperationOutcomeLost) {
		if updateErr := in.State.updateBuildOperation(ctx, buildOperationRecord{JobID: record.JobID, ModelID: record.ModelID, Intent: record.Intent, State: "lost", Result: err.Error()}); updateErr != nil {
			return errors.Join(fmt.Errorf("models: record missing prior build operation: %w", err), updateErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("models: recover persisted build operation: %w", err)
	}
	observation, err := owner.Inspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("models: inspect persisted build operation: %w", err)
	}
	if observation.State != platform.OperationStoppedOutcomeKnown || !observation.ExitKnown || observation.Populated {
		stopReceipt, stopErr := owner.Stop(ctx, ref)
		if stopErr != nil || !stopReceipt.NativeQuiescent {
			if stopErr == nil {
				stopErr = platform.ErrOperationNotQuiescent
			}
			return fmt.Errorf("models: prior build could not be stopped before replacement: %w", stopErr)
		}
		observation, err = owner.Inspect(ctx, ref)
		if err != nil || !observation.ExitKnown || observation.Populated {
			if err == nil {
				err = platform.ErrOperationNotQuiescent
			}
			return fmt.Errorf("models: prior build cleanup lacks terminal evidence: %w", err)
		}
	}
	if err := in.State.updateBuildOperation(ctx, buildOperationRecord{JobID: record.JobID, ModelID: record.ModelID, Intent: record.Intent, State: "recovered", Result: observation.Result, ExitStatus: observation.ExecMainStatus}); err != nil {
		return fmt.Errorf("models: persist recovered build state: %w", err)
	}
	if err := owner.Release(ctx, ref); err != nil {
		return fmt.Errorf("models: release recovered build operation: %w", err)
	}
	return nil
}

// cleanupOwnedBuild is the only post-dispatch cleanup path. A claim may be
// released only after the owner proves the exact operation tree is quiescent.
// If that proof is unavailable, stop renewing but leave the reservation to
// expire naturally; returning retain=true prevents premature RAM release.
func (in *Installer) cleanupOwnedBuild(record buildOperationRecord, ref platform.OperationRef, state string) (error, bool) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	owner := in.Build.Operations
	receipt, err := owner.Stop(cleanupCtx, ref)
	if err != nil || !receipt.NativeQuiescent {
		if err == nil {
			err = platform.ErrOperationNotQuiescent
		}
		persistErr := in.State.updateBuildOperation(cleanupCtx, buildOperationRecord{JobID: record.JobID, ModelID: record.ModelID, Intent: record.Intent, State: "outcome_unknown", Result: err.Error()})
		return errors.Join(fmt.Errorf("models: exporter cleanup cannot prove quiescence; RAM claim retained until expiry: %w", err), persistErr), true
	}
	terminal, err := owner.Inspect(cleanupCtx, ref)
	if err != nil || !terminal.ExitKnown || terminal.Populated {
		if err == nil {
			err = platform.ErrOperationNotQuiescent
		}
		persistErr := in.State.updateBuildOperation(cleanupCtx, buildOperationRecord{JobID: record.JobID, ModelID: record.ModelID, Intent: record.Intent, State: "outcome_unknown", Result: err.Error()})
		return errors.Join(fmt.Errorf("models: exporter cleanup lacks terminal evidence; RAM claim retained until expiry: %w", err), persistErr), true
	}
	if err := in.State.updateBuildOperation(cleanupCtx, buildOperationRecord{JobID: record.JobID, ModelID: record.ModelID, Intent: record.Intent, State: state, Result: terminal.Result, ExitStatus: terminal.ExecMainStatus}); err != nil {
		return errors.Join(fmt.Errorf("models: persist cleaned exporter state: %w", err), owner.Release(cleanupCtx, ref)), false
	}
	if err := owner.Release(cleanupCtx, ref); err != nil {
		return fmt.Errorf("models: release cleaned exporter operation: %w", err), false
	}
	return nil, false
}

type buildReceipt struct {
	Schema                      string  `json:"schema"`
	SourceArchiveSHA256         string  `json:"source_archive_sha256"`
	CheckpointSHA256            string  `json:"checkpoint_sha256"`
	CheckpointBytes             int64   `json:"checkpoint_bytes"`
	CheckpointKeyPrefix         string  `json:"checkpoint_key_prefix"`
	CapacityClaimBytes          int64   `json:"capacity_claim_bytes"`
	CPUClaimMillicores          int64   `json:"cpu_claim_millicores"`
	TorchThreadLimit            int     `json:"torch_thread_limit"`
	OperationCPUWeight          int     `json:"operation_cpu_weight"`
	CPUQuotaMillicores          int64   `json:"cpu_quota_millicores"`
	OutputSHA256                string  `json:"output_sha256"`
	OutputBytes                 int64   `json:"output_bytes"`
	Resolution                  []int   `json:"resolution"`
	Opset                       int     `json:"opset"`
	OutputFinite                bool    `json:"output_finite"`
	ParityMaxAbs                float64 `json:"parity_max_abs"`
	BuildElapsedMS              float64 `json:"build_elapsed_ms"`
	ProcessPeakRSSBytes         int64   `json:"process_peak_rss_bytes"`
	ProcessPeakRSSScope         string  `json:"process_peak_rss_scope"`
	ProcessPeakRSSAfterLoad     int64   `json:"process_peak_rss_after_model_load_bytes"`
	ProcessPeakRSSAfter512      int64   `json:"process_peak_rss_after_512_probe_bytes"`
	ProcessPeakRSSAfterExport   int64   `json:"process_peak_rss_after_onnx_export_bytes"`
	ProcessPeakRSSAfterParity   int64   `json:"process_peak_rss_after_runtime_parity_bytes"`
	UpstreamUtilsSourceSHA256   string  `json:"upstream_utils_source_sha256"`
	UpstreamUtilsPreparedSHA256 string  `json:"upstream_utils_prepared_sha256"`
	SourcePreparation           string  `json:"source_preparation"`
	ReferenceONNXSHA256         string  `json:"reference_onnx_sha256"`
	ReferenceProbeResolution    []int   `json:"reference_probe_resolution"`
	ReferenceProbeMaxAbs        float64 `json:"reference_probe_max_abs"`
	ReferenceProbeTolerance     float64 `json:"reference_probe_tolerance"`
}

func (in *Installer) ensureBuild(ctx context.Context, m Model, dir, jobID string, emit func(int, string)) (modelHash string, buildErr error) {
	profile := m.BuildProfile
	if profile == nil {
		return "", nil
	}
	if in.Build == nil || strings.TrimSpace(in.Build.Python) == "" || in.Build.Capacity == nil || strings.TrimSpace(in.Build.ReferenceONNX) == "" {
		return "", fmt.Errorf("models: build profile %q requires the provisioned Python environment and native RAM capacity broker", profile.Kind)
	}
	unlockBuild := lockModelBuild(m.ID)
	defer unlockBuild()
	if err := in.recoverPendingBuild(ctx, m.ID); err != nil {
		return "", err
	}
	outputPath := filepath.Join(dir, profile.OutputFilename)
	receiptPath := filepath.Join(dir, "build-provenance.json")
	if sum, ready := buildOutputReady(profile, dir); ready {
		return sum, nil
	}
	sourcePath := filepath.Join(dir, profile.SourceAsset)
	checkpointPath := filepath.Join(dir, profile.CheckpointAsset)
	if err := requireSHA256(sourcePath, profile.SourceSHA256); err != nil {
		return "", fmt.Errorf("models: build source validation: %w", err)
	}
	if err := requireSHA256(checkpointPath, profile.CheckpointSHA256); err != nil {
		return "", fmt.Errorf("models: build checkpoint validation: %w", err)
	}

	buildCtx, cancel := context.WithTimeout(ctx, lamaBuildTimeout)
	defer cancel()
	broker, err := in.Build.Capacity(buildCtx)
	if err != nil {
		return "", fmt.Errorf("models: open native capacity broker: %w", err)
	}
	if broker == nil {
		return "", fmt.Errorf("models: native capacity broker factory returned nil")
	}
	defer func() {
		if closeErr := broker.Close(); closeErr != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("models: close native capacity broker: %w", closeErr))
		}
	}()
	owner := "image-tools:model-build:" + m.ID
	lease, verdict, err := broker.AcquireNoReclaim(buildCtx, owner, lamaBuildRAMBytes, lamaBuildCPUMillicores)
	if err != nil || verdict.Kind != "grant" || lease == nil {
		if lease != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			releaseErr := lease.Release(cleanupCtx)
			cleanupCancel()
			if releaseErr != nil {
				err = errors.Join(err, fmt.Errorf("release partial RAM capacity lease: %w", releaseErr))
			}
		}
		if err != nil {
			return "", fmt.Errorf("models: reserve RAM for model build: %w", err)
		}
		return "", fmt.Errorf("models: native RAM admission %s: %s", verdict.Kind, verdict.Reason)
	}
	renewable, ok := lease.(capacity.RenewableLease)
	if !ok {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		releaseErr := lease.Release(cleanupCtx)
		cleanupCancel()
		return "", errors.Join(fmt.Errorf("models: native RAM capacity lease cannot be renewed"), releaseErr)
	}
	buildCtx, stopHeartbeat := keepBuildCapacity(buildCtx, renewable)
	retainRAMClaim := false
	defer func() {
		heartbeatErr := stopHeartbeat()
		releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer releaseCancel()
		if !retainRAMClaim {
			if releaseErr := lease.Release(releaseCtx); releaseErr != nil {
				buildErr = errors.Join(buildErr, fmt.Errorf("models: release RAM capacity lease: %w", releaseErr))
			}
		} else {
			buildErr = errors.Join(buildErr, errors.New("models: native exporter outcome unknown; RAM claim will expire without early release"))
		}
		if heartbeatErr != nil {
			buildErr = errors.Join(buildErr, fmt.Errorf("models: RAM capacity lease renewal failed: %w", heartbeatErr))
		}
	}()

	outputStage := filepath.Join(dir, ".lama-export-in-progress.onnx")
	receiptStage := filepath.Join(dir, ".lama-export-in-progress.json")
	_ = os.Remove(outputStage)
	_ = os.Remove(receiptStage)
	defer os.Remove(outputStage)
	defer os.Remove(receiptStage)
	args := []string{"--source-archive", sourcePath, "--checkpoint", checkpointPath, "--reference-onnx", in.Build.ReferenceONNX, "--output", outputStage, "--receipt", receiptStage, "--capacity-claim-bytes", fmt.Sprint(lamaBuildRAMBytes), "--cpu-claim-millicores", fmt.Sprint(lamaBuildCPUMillicores)}
	extraEnv := []string{"OMP_NUM_THREADS=" + lamaBuildTorchThreads, "MKL_NUM_THREADS=" + lamaBuildTorchThreads, "OPENBLAS_NUM_THREADS=" + lamaBuildTorchThreads, "NUMEXPR_NUM_THREADS=" + lamaBuildTorchThreads, "VECLIB_MAXIMUM_THREADS=" + lamaBuildTorchThreads}
	if in.Build.PythonPath != "" {
		extraEnv = append(extraEnv, "PYTHONPATH="+in.Build.PythonPath)
	}
	// This invocation is not user-configurable: the closed profile selects the
	// module and fixed output shape, and only the verified asset paths vary.
	fullArgs := append([]string{"-m", lamaBuildModule}, args...)
	emit(91, "exporting pinned Big-LaMa generator to fixed 1024 ONNX")
	var output []byte
	var runErr error
	if in.Build.Operations != nil {
		if strings.TrimSpace(jobID) == "" || in.State == nil {
			return "", errors.New("models: native exporter operation requires a durable install job identity and operation store")
		}
		failurePath := filepath.Join(dir, ".lama-export-failure.json")
		_ = os.Remove(failurePath)
		fullArgs = append(fullArgs, "--failure-report", failurePath)
		output, runErr, retainRAMClaim = in.runOwnedBuild(buildCtx, jobID, m.ID, dir, in.Build.Python, fullArgs, extraEnv)
		if runErr != nil {
			if data, readErr := os.ReadFile(failurePath); readErr == nil {
				output = data
			}
		}
		_ = os.Remove(failurePath)
	} else {
		run := in.Build.Run
		if run == nil {
			run = defaultBuildRunner
		}
		output, runErr = run(buildCtx, in.Build.Python, fullArgs, extraEnv)
	}
	if runErr != nil {
		detail := strings.TrimSpace(string(output))
		if len(detail) > 2048 {
			detail = detail[len(detail)-2048:]
		}
		if detail != "" {
			return "", fmt.Errorf("models: LaMa export failed: %w: %s", runErr, detail)
		}
		return "", fmt.Errorf("models: LaMa export failed: %w", runErr)
	}
	receipt, err := readBuildReceipt(receiptStage)
	if err != nil {
		return "", err
	}
	sum, size, err := hashFile(outputStage)
	if err != nil {
		return "", fmt.Errorf("models: hash exported ONNX: %w", err)
	}
	if receipt.Schema != "vrooli.image-tools.lama-onnx-build/v2" || receipt.OutputSHA256 != sum || receipt.OutputBytes != size || !receipt.OutputFinite || receipt.ParityMaxAbs > 0.05 || len(receipt.Resolution) != 2 || receipt.Resolution[0] != 1024 || receipt.Resolution[1] != 1024 || receipt.Opset != 17 || receipt.SourceArchiveSHA256 != profile.SourceSHA256 || receipt.CheckpointSHA256 != profile.CheckpointSHA256 || receipt.CheckpointBytes != 204348820 || receipt.CheckpointKeyPrefix != lamaCheckpointKeyPrefix || receipt.CapacityClaimBytes != lamaBuildRAMBytes || receipt.CPUClaimMillicores != lamaBuildCPUMillicores || receipt.TorchThreadLimit != 2 || receipt.OperationCPUWeight != 100 || receipt.CPUQuotaMillicores != 0 || receipt.ProcessPeakRSSBytes <= 0 || receipt.ProcessPeakRSSBytes > lamaBuildRAMBytes || receipt.ProcessPeakRSSScope != "single exporter process kernel high-water mark; excludes other cgroup processes" || receipt.ProcessPeakRSSAfterLoad <= 0 || receipt.ProcessPeakRSSAfter512 <= 0 || receipt.ProcessPeakRSSAfterExport <= 0 || receipt.ProcessPeakRSSAfterParity <= 0 || receipt.UpstreamUtilsSourceSHA256 != lamaUtilsSourceSHA256 || receipt.UpstreamUtilsPreparedSHA256 != lamaUtilsPreparedSHA256 || receipt.SourcePreparation != lamaSourcePreparation || receipt.ReferenceONNXSHA256 != lamaReferenceONNXSHA256 || len(receipt.ReferenceProbeResolution) != 2 || receipt.ReferenceProbeResolution[0] != 512 || receipt.ReferenceProbeResolution[1] != 512 || receipt.ReferenceProbeMaxAbs > lamaReferenceTolerance || receipt.ReferenceProbeTolerance != lamaReferenceTolerance {
		return "", fmt.Errorf("models: LaMa exporter receipt does not match the closed build profile")
	}
	if profile.OutputSHA256 != "" && profile.OutputSHA256 != sum {
		return "", fmt.Errorf("models: built ONNX checksum mismatch: expected %s got %s", profile.OutputSHA256, sum)
	}
	if err := os.Rename(outputStage, outputPath); err != nil {
		return "", fmt.Errorf("models: publish built ONNX: %w", err)
	}
	if err := os.Rename(receiptStage, receiptPath); err != nil {
		_ = os.Remove(outputPath)
		return "", fmt.Errorf("models: publish build receipt: %w", err)
	}
	return sum, nil
}

// BuildReady reports whether an installed build-profile model has a complete,
// checksum-matching export receipt. Downloaded source assets alone do not make a
// compiled model available to inference.
func (in *Installer) BuildReady(ctx context.Context, id string) bool {
	if in == nil || in.Reg == nil {
		return false
	}
	m, ok := in.Reg.ByID(id)
	if !ok || m.BuildProfile == nil || !in.Installed(ctx, id) {
		return false
	}
	if in.State != nil {
		rec, found, err := in.State.Get(ctx, id)
		if err != nil || !found || !rec.Installed {
			return false
		}
		_, ready := buildOutputReady(m.BuildProfile, rec.Path)
		return ready
	}
	_, ready := buildOutputReady(m.BuildProfile, in.modelDir(id))
	return ready
}

func buildInputsReady(profile *BuildProfile, dir string) bool {
	if profile == nil || dir == "" {
		return false
	}
	return requireSHA256(filepath.Join(dir, profile.SourceAsset), profile.SourceSHA256) == nil &&
		requireSHA256(filepath.Join(dir, profile.CheckpointAsset), profile.CheckpointSHA256) == nil
}

func buildOutputReady(profile *BuildProfile, dir string) (string, bool) {
	if profile == nil {
		return "", false
	}
	outputPath := filepath.Join(dir, profile.OutputFilename)
	receipt, err := readBuildReceipt(filepath.Join(dir, "build-provenance.json"))
	if err != nil {
		return "", false
	}
	sum, size, err := hashFile(outputPath)
	if err != nil {
		return "", false
	}
	valid := receipt.Schema == "vrooli.image-tools.lama-onnx-build/v2" &&
		receipt.OutputSHA256 == sum && receipt.OutputBytes == size && receipt.OutputFinite &&
		receipt.ParityMaxAbs <= 0.05 && len(receipt.Resolution) == 2 &&
		receipt.Resolution[0] == profile.Resolution && receipt.Resolution[1] == profile.Resolution &&
		receipt.Opset == profile.Opset && receipt.SourceArchiveSHA256 == profile.SourceSHA256 &&
		receipt.CheckpointSHA256 == profile.CheckpointSHA256 && receipt.CheckpointBytes == 204348820 &&
		receipt.CheckpointKeyPrefix == lamaCheckpointKeyPrefix &&
		receipt.CapacityClaimBytes == lamaBuildRAMBytes && receipt.CPUClaimMillicores == lamaBuildCPUMillicores && receipt.TorchThreadLimit == 2 && receipt.OperationCPUWeight == 100 && receipt.CPUQuotaMillicores == 0 && receipt.ProcessPeakRSSBytes > 0 && receipt.ProcessPeakRSSBytes <= lamaBuildRAMBytes &&
		receipt.ProcessPeakRSSScope == "single exporter process kernel high-water mark; excludes other cgroup processes" &&
		receipt.ProcessPeakRSSAfterLoad > 0 && receipt.ProcessPeakRSSAfter512 > 0 && receipt.ProcessPeakRSSAfterExport > 0 && receipt.ProcessPeakRSSAfterParity > 0 &&
		receipt.UpstreamUtilsSourceSHA256 == lamaUtilsSourceSHA256 &&
		receipt.UpstreamUtilsPreparedSHA256 == lamaUtilsPreparedSHA256 &&
		receipt.SourcePreparation == lamaSourcePreparation &&
		receipt.ReferenceONNXSHA256 == lamaReferenceONNXSHA256 && len(receipt.ReferenceProbeResolution) == 2 &&
		receipt.ReferenceProbeResolution[0] == 512 && receipt.ReferenceProbeResolution[1] == 512 &&
		receipt.ReferenceProbeMaxAbs <= lamaReferenceTolerance && receipt.ReferenceProbeTolerance == lamaReferenceTolerance &&
		(profile.OutputSHA256 == "" || profile.OutputSHA256 == sum)
	return sum, valid
}

func defaultBuildRunner(ctx context.Context, python string, args, extraEnv []string) ([]byte, error) {
	return nil, errors.New("models: direct exporter subprocesses are not supported; configure the native operation owner")
}

func readBuildReceipt(path string) (buildReceipt, error) {
	var receipt buildReceipt
	data, err := os.ReadFile(path)
	if err != nil {
		return receipt, err
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func requireSHA256(path, expected string) error {
	got, _, err := hashFile(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("SHA-256 mismatch: expected %s got %s", expected, got)
	}
	return nil
}

func keepBuildCapacity(ctx context.Context, lease capacity.RenewableLease) (context.Context, func() error) {
	ctx, cancel := context.WithCancelCause(ctx)
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		var heartbeatErr error
		defer func() { done <- heartbeatErr }()
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				heartbeatCtx, stopHeartbeat := context.WithTimeout(ctx, 5*time.Second)
				err := lease.Heartbeat(heartbeatCtx)
				stopHeartbeat()
				if err != nil {
					heartbeatErr = err
					cancel(fmt.Errorf("RAM capacity claim renewal lost: %w", err))
					return
				}
			}
		}
	}()
	return ctx, func() error {
		close(stop)
		cancel(context.Canceled)
		return <-done
	}
}
