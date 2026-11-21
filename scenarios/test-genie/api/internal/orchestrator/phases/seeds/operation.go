package seeds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
)

// OperationRecord is private, reconstructible native process ownership. An
// intent must be persisted before dispatch; a ref is never guessed from a PID.
type OperationRecord struct {
	Intent            platform.OperationIntent `json:"intent"`
	Ref               *platform.OperationRef   `json:"ref,omitempty"`
	Quiescent         bool                     `json:"quiescent"`
	Gate              *platform.OperationGate  `json:"gate"`
	WorkerPath        string                   `json:"worker_path,omitempty"`
	InputPath         string                   `json:"input_path,omitempty"`
	GrantPath         string                   `json:"grant_path,omitempty"`
	LogPath           string                   `json:"log_path,omitempty"`
	Released          bool                     `json:"released,omitempty"`
	AttemptID         string                   `json:"attempt_id"`
	DispatchCommitted bool                     `json:"dispatch_committed"`
}

func StopOperation(ctx context.Context, record *OperationRecord) error {
	if record == nil {
		return nil
	}
	if record.Quiescent {
		return retireWorkerFiles(record)
	}
	if record.Gate == nil {
		return errors.New("seed native operation has no revocable pre-exec gate")
	}
	if err := record.Gate.Revoke(); err != nil {
		return err
	}
	if revoked, err := record.Gate.Revoked(); err != nil || !revoked {
		return errors.Join(errors.New("seed native gate revocation is unverified"), err)
	}
	if !record.DispatchCommitted {
		record.Quiescent = true
		return retireWorkerFiles(record)
	}
	ref := record.Ref
	if ref == nil {
		found, err := platform.LookupOwnedOperation(ctx, record.Intent)
		if errors.Is(err, platform.ErrOperationOutcomeLost) || errors.Is(err, platform.ErrOperationNotFound) {
			absent, absenceErr := platform.OwnedOperationIntentAbsent(ctx, record.Intent)
			if absenceErr != nil {
				return absenceErr
			}
			if absent {
				record.Quiescent = true
				return retireWorkerFiles(record)
			}
		}
		if err != nil {
			return err
		}
		record.Ref = &found
		ref = &found
	}
	receipt, err := platform.StopOwnedOperation(ctx, *ref)
	if err != nil {
		return err
	}
	if !receipt.NativeQuiescent {
		return platform.ErrOperationNotQuiescent
	}
	record.Quiescent = true
	return retireWorkerFiles(record)
}

func retireWorkerFiles(record *OperationRecord) error {
	if record.WorkerPath == "" && record.InputPath == "" && record.GrantPath == "" {
		return nil
	}
	if record.Gate == nil {
		return errors.New("seed worker input retirement has no gate identity")
	}
	root := filepath.Dir(record.Gate.ReadyPath)
	attempt := record.AttemptID
	if attempt == "" {
		attempt = record.Intent.AttemptID
	}
	worker, input, grant, _ := privateWorkerPaths(root, attempt)
	if record.WorkerPath != worker || record.InputPath != input || record.GrantPath != grant {
		return errors.New("seed worker input retirement identity mismatch")
	}
	var result error
	for _, path := range []string{worker, input, grant, filepath.Join(root, attempt+".claim.lock")} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}

// ReleaseOperation runs only after the domain has persisted Quiescent. A lost
// release acknowledgement may converge through revoked-gate plus exact native
// absence; it never reconstructs an executable success outcome.
func ReleaseOperation(ctx context.Context, record *OperationRecord) error {
	if record == nil || record.Released {
		return nil
	}
	if !record.Quiescent || record.Gate == nil {
		return errors.New("seed operation release lacks durable quiescence and gate identity")
	}
	if !record.DispatchCommitted {
		record.Released = true
		return nil
	}
	if revoked, err := record.Gate.Revoked(); err != nil || !revoked {
		return errors.Join(errors.New("seed operation release gate is not revoked"), err)
	}
	if record.Ref != nil {
		err := platform.ReleaseOwnedOperation(ctx, *record.Ref)
		if err == nil {
			record.Released = true
			return nil
		}
		if !errors.Is(err, platform.ErrOperationOutcomeLost) {
			return err
		}
	}
	absent, err := platform.OwnedOperationIntentAbsent(ctx, record.Intent)
	if err != nil {
		return err
	}
	if !absent {
		return platform.ErrOperationNotQuiescent
	}
	record.Released = true
	return nil
}

func (m *FileManager) runOwnedScript(ctx context.Context, command string, args, env []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return err
	}
	wall := 2 * time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		wall = time.Until(deadline)
		if wall <= 0 {
			return context.DeadlineExceeded
		}
	}
	attempt := uuid.NewString()
	operationRoot := m.operationRoot
	if operationRoot == "" {
		resolver, err := storage.NewResolver(storage.ResolverConfig{})
		if err != nil {
			return err
		}
		namespace, err := storage.ScenarioNamespace("test-genie")
		if err != nil {
			return err
		}
		operationRoot, err = resolver.Path(storage.Options{ScenarioID: namespace}, storage.ClassState, filepath.Join("seed-processes", "gates"))
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(operationRoot, 0700); err != nil {
		return err
	}
	gate, err := platform.PrepareOperationGate(operationRoot, attempt)
	if err != nil {
		return err
	}
	workerPath, inputPath, grantPath, logPath := privateWorkerPaths(operationRoot, attempt)
	record := &OperationRecord{AttemptID: attempt, Gate: &gate, WorkerPath: workerPath, InputPath: inputPath, GrantPath: grantPath, LogPath: logPath}
	// Persist the private-file plan before copying an image or child input.
	// No native dispatch is possible until the later committed record exists.
	if m.operationChanged != nil {
		if err := m.operationChanged(record); err != nil {
			return errors.Join(err, gate.Revoke())
		}
	}
	workerSHA, err := freezeWorker(workerPath)
	if err != nil {
		return errors.Join(err, gate.Revoke())
	}
	input := workerInput{Version: 1, Path: path, Args: args, Env: env, Dir: m.scenarioDir, Deadline: time.Now().Add(wall)}
	if err := writeWorkerJSON(inputPath, input); err != nil {
		return errors.Join(err, gate.Revoke())
	}
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return errors.Join(err, gate.Revoke())
	}
	inputHash := sha256.Sum256(inputBytes)
	inputSHA := hex.EncodeToString(inputHash[:])
	spec := platform.OperationSpec{Path: workerPath, Args: []string{workerFlag, inputPath, inputSHA}, Env: []string{"PATH=/usr/bin:/bin"}, Dir: m.scenarioDir, Limits: platform.OperationLimits{MemoryMaxBytes: 2 << 30, TasksMax: 512, CPUWeight: 50, WallTime: wall}, Gate: &gate}
	intent, err := platform.NewOperationIntent("test-genie-seed", attempt, spec)
	if err != nil {
		return errors.Join(err, gate.Revoke())
	}
	record.Intent = intent
	if err := writeWorkerJSON(grantPath, workerGrant{Version: 1, Intent: intent, Gate: gate, InputSHA: inputSHA, WorkerSHA: workerSHA}); err != nil {
		return errors.Join(err, gate.Revoke(), retireWorkerFiles(record))
	}
	// Standalone suite callers still persist intent before native dispatch.
	// The Seed Service supplies its own enclosing session journal instead.
	var standalonePath string
	recorder := m.operationChanged
	if recorder == nil {
		standalonePath = filepath.Join(operationRoot, intent.AttemptID+".json")
		recorder = func(r *OperationRecord) error {
			data, err := json.Marshal(r)
			if err != nil {
				return err
			}
			return storage.WriteFileAtomic(standalonePath, data, 0600)
		}
	}
	if recorder != nil {
		record.DispatchCommitted = true
		if err = recorder(record); err != nil {
			return errors.Join(err, gate.Revoke())
		}
	}
	operation, startErr := platform.StartOwnedOperation(ctx, intent, spec)
	if operation != nil {
		record.Ref = &operation.Ref
		if recorder != nil {
			err = recorder(record)
		}
	}
	if startErr == nil && err == nil {
		err = operation.Wait(ctx)
		if err == nil {
			observation, observeErr := platform.InspectOwnedOperation(ctx, operation.Ref)
			err = observeErr
			if observeErr == nil && (observation.Populated || !observation.ExitKnown || observation.ExecMainCode != 1 || observation.ExecMainStatus != 0 || observation.Result != "success") {
				err = fmt.Errorf("seed native operation failed: result=%s code=%d status=%d", observation.Result, observation.ExecMainCode, observation.ExecMainStatus)
			}
		}
	}
	cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stopErr := StopOperation(cleanup, record)
	if recorder != nil {
		stopErr = errors.Join(stopErr, recorder(record))
	}
	if stopErr == nil {
		stopErr = ReleaseOperation(cleanup, record)
		if recorder != nil {
			stopErr = errors.Join(stopErr, recorder(record))
		}
	}
	if stopErr == nil && standalonePath != "" {
		stopErr = os.Remove(standalonePath)
	}
	return errors.Join(startErr, err, stopErr)
}
