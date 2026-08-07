package vroolicli

import (
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// hostSnapshotResponse maps hostinventory.Snapshot onto the vrooli.cli.v1 wire
// contract. A proto field rename breaks this mapping at compile time.
func hostSnapshotResponse(s hostinventory.Snapshot) *cliv1.CliHostSnapshot {
	out := &cliv1.CliHostSnapshot{
		Os:   s.OS,
		Arch: s.Arch,
		Cpu:  &cliv1.CliHostCPU{Cores: int32(s.CPU.Cores)},
		Load: &cliv1.CliHostLoad{
			Load1:           s.Load.Load1,
			Load5:           s.Load.Load5,
			Load15:          s.Load.Load15,
			RunningProcs:    int32(s.Load.RunningProcs),
			TotalProcs:      int32(s.Load.TotalProcs),
			LastPid:         int32(s.Load.LastPID),
			NormalizedLoad1: s.Load.NormalizedLoad1,
			NormalizedLoad5: s.Load.NormalizedLoad5,
		},
		Memory: &cliv1.CliHostMemory{
			TotalBytes:     int64(s.Memory.TotalBytes),
			AvailableBytes: int64(s.Memory.AvailableBytes),
			BuffersBytes:   int64(s.Memory.BuffersBytes),
			CachedBytes:    int64(s.Memory.CachedBytes),
		},
		Swap: &cliv1.CliHostSwap{
			TotalBytes: int64(s.Swap.TotalBytes),
			FreeBytes:  int64(s.Swap.FreeBytes),
		},
		DockerGpu:         &cliv1.CliHostDockerGPU{NvidiaRuntime: s.DockerGPU.NvidiaRuntime},
		NvidiaDeviceNodes: s.NvidiaDeviceNodes,
		Warnings:          s.Warnings,
		DisplayAttached:   s.DisplayAttached,
		DisplayServer:     s.DisplayServer,
		WaylandAttainable: s.Wayland.Attainable,
		WaylandReason:     s.Wayland.Reason,
		DisplayManager:    s.DisplayManager,
		SessionType:       s.SessionType,
		Seat:              s.Seat,
		ActiveSessionUser: s.ActiveSessionUser,
		AutoLoginUser:     s.AutoLoginUser,
		RemoteDesktop: &cliv1.CliHostRemoteDesktop{
			Supported:        s.RemoteDesktop.Supported,
			Observed:         s.RemoteDesktop.Observed,
			Mode:             s.RemoteDesktop.Mode,
			Active:           s.RemoteDesktop.Active,
			ListeningPort:    int32(s.RemoteDesktop.ListeningPort),
			SelectedProvider: s.RemoteDesktop.SelectedProvider,
			CredentialStore: &cliv1.CliHostCredentialStore{
				Supported:      s.RemoteDesktop.CredentialStore.Supported,
				Observed:       s.RemoteDesktop.CredentialStore.Observed,
				State:          s.RemoteDesktop.CredentialStore.State,
				ProbeSucceeded: s.RemoteDesktop.CredentialStore.ProbeSucceeded,
				Reason:         s.RemoteDesktop.CredentialStore.Reason,
			},
		},
	}
	for _, provider := range s.RemoteDesktop.Providers {
		out.RemoteDesktop.Providers = append(out.RemoteDesktop.Providers, &cliv1.CliHostRemoteDesktopProvider{
			Name:           provider.Name,
			Present:        provider.Present,
			Active:         provider.Active,
			ProbeSucceeded: provider.ProbeSucceeded,
			UserSession:    provider.UserSession,
		})
	}

	for _, gpu := range s.GPUs {
		out.Gpus = append(out.Gpus, &cliv1.CliHostGPU{
			Index:                    int32(gpu.Index),
			Uuid:                     gpu.UUID,
			Name:                     gpu.Name,
			DriverVersion:            gpu.DriverVersion,
			VramBytes:                int64(gpu.VRAMBytes),
			VramUsedBytes:            int64(gpu.VRAMUsedBytes),
			UtilizationPercent:       gpu.UtilizationPercent,
			MemoryUtilizationPercent: gpu.MemoryUtilizationPercent,
			TemperatureC:             derefFloat(gpu.TemperatureC),
			FanSpeedPercent:          derefFloat(gpu.FanSpeedPercent),
			PowerDrawW:               derefFloat(gpu.PowerDrawW),
			PowerLimitW:              derefFloat(gpu.PowerLimitW),
			SmClockMhz:               derefFloat(gpu.SMClockMHz),
			MemoryClockMhz:           derefFloat(gpu.MemoryClockMHz),
			Source:                   gpu.Source,
		})
	}

	for _, proc := range s.GPUProcesses {
		out.GpuProcesses = append(out.GpuProcesses, &cliv1.CliHostGPUProcess{
			GpuIndex:    int32(proc.GPUIndex),
			GpuUuid:     proc.GPUUUID,
			Pid:         int32(proc.PID),
			ProcessName: proc.ProcessName,
			UsedBytes:   int64(proc.UsedBytes),
		})
	}

	if len(s.RuntimeTools) > 0 {
		out.RuntimeTools = make(map[string]*cliv1.CliHostTool, len(s.RuntimeTools))
		for name, tool := range s.RuntimeTools {
			out.RuntimeTools[name] = &cliv1.CliHostTool{Present: tool.Present, Path: tool.Path}
		}
	}

	if len(s.ProbeStatuses) > 0 {
		out.ProbeStatuses = make(map[string]string, len(s.ProbeStatuses))
		for k, v := range s.ProbeStatuses {
			out.ProbeStatuses[k] = v
		}
	}

	if len(s.FieldProvenance) > 0 {
		out.FieldProvenance = make(map[string]*cliv1.CliHostProvenance, len(s.FieldProvenance))
		for field, prov := range s.FieldProvenance {
			out.FieldProvenance[field] = &cliv1.CliHostProvenance{
				SourceKind: string(prov.SourceKind),
				Source:     prov.Source,
				ObservedAt: formatTime(prov.ObservedAt),
				Confidence: prov.Confidence,
				Command:    prov.Command,
				File:       prov.File,
			}
		}
	}

	return out
}

func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// writeHostSnapshotJSON emits the host-inventory wire contract as JSON.
func writeHostSnapshotJSON(w io.Writer, s hostinventory.Snapshot) error {
	return cliout.WriteProtoJSON(w, hostSnapshotResponse(s))
}
