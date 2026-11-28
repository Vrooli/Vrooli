package rendering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const physicalGPUDiagnosticsName = "physical-gpu-diagnostics.json"

// physicalBlenderProcess owns every Blender subprocess from admission through
// reap. Callers may stream frames while it runs, but must finish or abort it
// before returning; GPU leases are released only after Wait has reaped Blender.
type physicalBlenderProcess struct {
	ctx                context.Context
	cancel             context.CancelFunc
	root               *os.Root
	manifest           PhysicalRenderManifest
	cmd                *exec.Cmd
	stdout             *physicalBlenderLogBuffer
	stderr             *physicalBlenderLogBuffer
	started            time.Time
	startedProc        bool
	reaped             bool
	finished           bool
	exitErr            error
	gpuLease           *vroolicli.GPUCapacityLease
	gpuIdentity        vroolicli.GPUIdentity
	stopHeartbeat      chan struct{}
	heartbeatDone      chan error
	stopUsage          chan struct{}
	usageDone          chan gpuProcessUsage
	gpuUsage           gpuProcessUsage
	gpuProofErr        error
	gpuName            string
	gpuBackendSelected string
	gpuRenderCompleted bool
	gpuReleaseErr      string
}

func startPhysicalBlenderProcess(
	parent context.Context,
	root *os.Root,
	workspace, blender string,
	manifest PhysicalRenderManifest,
	extraFiles []*os.File,
	stdout, stderr *physicalBlenderLogBuffer,
) (*physicalBlenderProcess, error) {
	if parent == nil || parent.Err() != nil || root == nil || !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace {
		return nil, errors.New("physical Blender process requires a live context and fixed attempt root")
	}
	if !filepath.IsAbs(blender) || filepath.Clean(blender) != blender {
		return nil, errors.New("managed Blender path is not absolute and clean")
	}
	binary, err := readRegularPath(blender, 512<<20)
	if err != nil || digest(binary) != PhysicalBlenderBinarySHA {
		return nil, errors.New("managed Blender executable digest differs from profile")
	}
	if stdout == nil || stderr == nil {
		return nil, errors.New("physical Blender output buffers are unavailable")
	}
	if err := ensurePhysicalBlenderDirectories(workspace); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	cmd := exec.CommandContext(ctx, blender, physicalBlenderArgs(workspace)...)
	cmd.Dir = workspace
	cmd.Env = physicalBlenderEnvironment(workspace)
	cmd.ExtraFiles = extraFiles
	cmd.Stdout, cmd.Stderr = stdout, stderr
	process := &physicalBlenderProcess{
		ctx: ctx, cancel: cancel, root: root, manifest: manifest, cmd: cmd,
		stdout: stdout, stderr: stderr, started: time.Now(),
	}
	if manifest.Profile.Backend != "CPU" {
		if err := ensurePhysicalOutputAbsent(root, physicalGPUDiagnosticsName); err != nil {
			cancel()
			return nil, err
		}
		uuid, err := physicalGPUUUID(ctx)
		if err != nil {
			cancel()
			return nil, err
		}
		lease, err := vroolicli.New().AcquireGPUCapacity(ctx, vroolicli.GPUCapacityRequest{
			OwnerID: "device-studio:physical-render:" + manifest.JobID + ":" + manifest.AttemptID,
			GPUUUID: uuid, VRAMBytes: physicalGPUClaimBytes,
		})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("admit pinned GPU render: %w", err)
		}
		process.gpuLease, process.gpuIdentity = lease, lease.Identity()
		cmd.Env = append(cmd.Env,
			"CUDA_VISIBLE_DEVICES="+process.gpuIdentity.UUID,
			"NVIDIA_VISIBLE_DEVICES="+process.gpuIdentity.UUID,
			"DEVICE_STUDIO_GPU_UUID="+process.gpuIdentity.UUID,
		)
		process.stopHeartbeat = make(chan struct{})
		process.heartbeatDone = make(chan error, 1)
		go process.heartbeat()
	}
	if err := cmd.Start(); err != nil {
		process.exitErr = err
		return nil, errors.Join(err, process.finish())
	}
	process.startedProc = true
	if process.gpuLease != nil {
		process.stopUsage = make(chan struct{})
		process.usageDone = make(chan gpuProcessUsage, 1)
		go func() {
			process.usageDone <- sampleGPUProcessUsage(ctx, cmd.Process.Pid, process.gpuIdentity.UUID, process.stopUsage)
		}()
	}
	return process, nil
}

func physicalGPUUUID(ctx context.Context) (string, error) {
	if uuid := strings.TrimSpace(os.Getenv("DEVICE_STUDIO_GPU_UUID")); uuid != "" {
		return uuid, nil
	}
	inventory, err := vroolicli.New().HostInventory(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve pinned physical GPU from host inventory: %w", err)
	}
	return physicalGPUUUIDFromInventory(inventory.GetGpus())
}

func physicalGPUUUIDFromInventory(gpus []*cliv1.HostGPU) (string, error) {
	if len(gpus) == 1 && strings.TrimSpace(gpus[0].GetUuid()) != "" {
		return strings.TrimSpace(gpus[0].GetUuid()), nil
	}
	observed := make([]string, 0, len(gpus))
	for i, gpu := range gpus {
		if gpu == nil {
			observed = append(observed, fmt.Sprintf("[%d]=<nil>", i))
			continue
		}
		observed = append(observed, fmt.Sprintf("[%d]{name:%q,uuid:%q}", i, gpu.GetName(), strings.TrimSpace(gpu.GetUuid())))
	}
	return "", fmt.Errorf("GPU render requires DEVICE_STUDIO_GPU_UUID or exactly one identified GPU in host inventory; observed %d GPU records: %s", len(gpus), strings.Join(observed, ", "))
}

func (p *physicalBlenderProcess) heartbeat() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopHeartbeat:
			p.heartbeatDone <- nil
			return
		case <-p.ctx.Done():
			p.heartbeatDone <- nil
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := p.gpuLease.Heartbeat(ctx)
			cancel()
			if err != nil {
				p.cancel()
				p.heartbeatDone <- err
				return
			}
		}
	}
}

func (p *physicalBlenderProcess) wait() error { return p.finish() }

func (p *physicalBlenderProcess) abort() error {
	if p == nil {
		return nil
	}
	if p.startedProc && !p.reaped && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return p.finish()
}

func (p *physicalBlenderProcess) finish() error {
	if p == nil || p.finished {
		return nil
	}
	if p.startedProc && !p.reaped {
		p.exitErr = p.cmd.Wait()
		p.reaped = true
	}
	if p.stopUsage != nil {
		close(p.stopUsage)
		usage := <-p.usageDone
		p.gpuUsage = usage
	}
	var heartbeatErr error
	if p.stopHeartbeat != nil {
		close(p.stopHeartbeat)
		heartbeatErr = <-p.heartbeatDone
	}
	if p.cancel != nil {
		p.cancel()
	}
	if heartbeatErr != nil {
		p.exitErr = errors.Join(p.exitErr, fmt.Errorf("GPU capacity heartbeat failed; Blender was cancelled and reaped: %w", heartbeatErr))
	}
	if p.gpuLease != nil {
		p.writeGPUProof()
		if p.gpuUsage.Samples == 0 || p.gpuUsage.PeakBytes == 0 {
			p.exitErr = errors.Join(p.exitErr, errors.New("GPU render lacks exact-process VRAM peak evidence"))
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := p.gpuLease.Release(cleanupCtx); err != nil {
			p.gpuReleaseErr = err.Error()
			p.exitErr = errors.Join(p.exitErr, fmt.Errorf("release GPU capacity claim after Blender exited: %w", err))
		}
		cancel()
	}
	if err := p.writeDiagnostics(); err != nil {
		p.exitErr = errors.Join(p.exitErr, err)
	}
	p.finished = true
	if p.exitErr != nil {
		return physicalBlenderFailure(p.exitErr, p.ctx.Err(), p.stderr.data.String())
	}
	return nil
}

// These fields are filled only after the matching Blender child is reaped.
func (p *physicalBlenderProcess) writeGPUProof() {
	proof, err := readPhysicalGPUDiagnostics(p.root)
	if err != nil {
		p.gpuProofErr = err
		p.exitErr = errors.Join(p.exitErr, fmt.Errorf("read actual Blender GPU selection proof: %w", err))
		return
	}
	if err := validatePhysicalGPUSelectionProof(proof, p.gpuIdentity.UUID, p.manifest.Profile.Backend); err != nil {
		p.gpuProofErr = err
		p.exitErr = errors.Join(p.exitErr, p.gpuProofErr)
		return
	}
	p.gpuBackendSelected = proof.Backend
	p.gpuRenderCompleted = proof.RenderCompleted
	p.gpuName = proof.SelectedDevice
	if !proof.RenderCompleted && p.ctx.Err() == nil && p.exitErr == nil {
		p.gpuProofErr = errors.New("Blender exited without completing the GPU render")
		p.exitErr = errors.Join(p.exitErr, p.gpuProofErr)
	}
}

func validatePhysicalGPUSelectionProof(proof physicalGPUDiagnostics, expectedUUID, expectedBackend string) error {
	if expectedUUID == "" || expectedBackend == "" || proof.UUID != expectedUUID || proof.Backend != expectedBackend || proof.SelectedDevice == "" || len(proof.VisibleDevices) != 1 || proof.VisibleDevices[0] != proof.SelectedDevice {
		return errors.New("Blender selected backend/device did not match the pinned GPU lease")
	}
	return nil
}

func (p *physicalBlenderProcess) writeDiagnostics() error {
	d := physicalBlenderDiagnostics{
		Version: 1, Executable: filepath.Base(p.cmd.Path), Backend: p.manifest.Profile.Backend,
		Engine: p.manifest.Profile.Engine, Width: p.manifest.Output.Width, Height: p.manifest.Output.Height,
		Samples: p.manifest.Profile.Samples, Threads: p.manifest.Profile.Threads,
		WallTimeLimit: p.manifest.Profile.WallTimeSeconds, ElapsedMillis: time.Since(p.started).Milliseconds(),
		StdoutTail: strings.ToValidUTF8(p.stdout.data.String(), "\uFFFD"),
		StderrTail: strings.ToValidUTF8(p.stderr.data.String(), "\uFFFD"),
	}
	if p.gpuLease != nil {
		d.GPUUUID, d.GPUName, d.GPUIndex = p.gpuIdentity.UUID, p.gpuName, p.gpuIdentity.Index
		d.GPUClaimID, d.GPUClaimBytes = p.gpuLease.ClaimID(), physicalGPUClaimBytes
		d.GPUBackendSelected, d.GPURenderCompleted = p.gpuBackendSelected, p.gpuRenderCompleted
		d.GPUPeakProcessBytes, d.GPUSamples, d.GPUUsageProbeError = p.gpuUsage.PeakBytes, p.gpuUsage.Samples, p.gpuUsage.Error
		d.GPUReleaseErr = p.gpuReleaseErr
		if p.gpuUsage.Samples == 0 || p.gpuUsage.PeakBytes == 0 {
			d.ExitError = "GPU render lacks exact-process VRAM peak evidence"
		}
		if p.gpuProofErr != nil && d.ExitError == "" {
			d.ExitError = p.gpuProofErr.Error()
		}
	}
	if p.exitErr != nil {
		d.ExitError = p.exitErr.Error()
	}
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("encode physical Blender diagnostics: %w", err)
	}
	if err := writePhysicalRoot(p.root, physicalBlenderLogName, append(data, '\n')); err != nil {
		return fmt.Errorf("persist physical Blender diagnostics: %w", err)
	}
	return nil
}
