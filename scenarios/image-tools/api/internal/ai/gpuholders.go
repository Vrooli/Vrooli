package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// gpuHolder is one process that held GPU memory at an instant.
type gpuHolder struct {
	PID    int
	MiB    int
	Detail string // the process's command, trimmed
}

// holderQueryTimeout bounds the nvidia-smi call so a wedged driver cannot hold a failed job.
const holderQueryTimeout = 3 * time.Second

// parseGPUHolders reads `nvidia-smi --query-compute-apps=pid,used_memory
// --format=csv,noheader,nounits` output. Lines it cannot read are skipped.
func parseGPUHolders(output string, describe func(pid int) string) []gpuHolder {
	var holders []gpuHolder
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, ",")
		if len(fields) != 2 {
			continue
		}
		pid, pidErr := strconv.Atoi(strings.TrimSpace(fields[0]))
		mib, memErr := strconv.Atoi(strings.TrimSpace(fields[1]))
		if pidErr != nil || memErr != nil {
			continue
		}
		holders = append(holders, gpuHolder{PID: pid, MiB: mib, Detail: describe(pid)})
	}
	return holders
}

// processCommand is the first 80 characters of a process's command line, "" when it is gone.
func processCommand(pid int) string {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	command := strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
	if len(command) > 80 {
		command = command[:80]
	}
	return command
}

// queryGPUHolders lists the processes holding GPU memory right now. It returns nil
// when there is no NVML (no nvidia-smi, or the query fails): the caller reports what it has.
func queryGPUHolders(ctx context.Context) []gpuHolder {
	ctx, cancel := context.WithTimeout(ctx, holderQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi", "--query-compute-apps=pid,used_memory", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil
	}
	return parseGPUHolders(string(out), processCommand)
}

// describeGPUHolders renders holders for a failure message, largest first.
func describeGPUHolders(holders []gpuHolder) string {
	if len(holders) == 0 {
		return ""
	}
	sorted := append([]gpuHolder(nil), holders...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].MiB > sorted[j-1].MiB; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	parts := make([]string, 0, len(sorted))
	for _, h := range sorted {
		part := fmt.Sprintf("pid %d %d MiB", h.PID, h.MiB)
		if h.Detail != "" {
			part += " (" + h.Detail + ")"
		}
		parts = append(parts, part)
	}
	return "GPU memory held when it failed: " + strings.Join(parts, "; ")
}
