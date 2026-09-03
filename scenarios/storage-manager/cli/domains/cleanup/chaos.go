package cleanup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
)

const (
	chaosChunk       = int64(1 << 20)
	chaosMaxDuration = 10 * time.Minute
	chaosMaxRate     = int64(100) << 30
)

type chaosResult struct {
	Root           string `json:"root"`
	Path           string `json:"path"`
	Rate           int64  `json:"requested_bytes_per_hour"`
	BytesWritten   int64  `json:"bytes_written"`
	WriterStarted  string `json:"writer_started"`
	WriterEnded    string `json:"writer_ended"`
	RecoveryRun    string `json:"recovery_run_id"`
	RecoveryState  string `json:"recovery_state"`
	Reclaimed      int64  `json:"reclaimed_bytes"`
	TargetFree     int64  `json:"target_free_bytes"`
	StoppedBecause string `json:"stopped_because"`
}

func (h *handlers) chaosCall(ctx cliapp.OperationContext) (chaosResult, error) {
	root, err := governedChaosRoot(ctx.Flag("root"))
	if err != nil {
		return chaosResult{}, err
	}
	rate, err := parseChaosRate(ctx.Flag("rate"))
	if err != nil {
		return chaosResult{}, err
	}
	duration := 8 * time.Minute
	if raw := strings.TrimSpace(ctx.Flag("duration")); raw != "" {
		duration, err = time.ParseDuration(raw)
		if err != nil {
			return chaosResult{}, fmt.Errorf("--duration: %w", err)
		}
	}
	if duration <= 0 || duration > chaosMaxDuration {
		return chaosResult{}, fmt.Errorf("--duration must be within (0,%s]", chaosMaxDuration)
	}
	// Keep the fixture visible to the governed temporary-root provider. Hidden
	// names are hard-protected by the normal cleanup safety filter and would
	// make this exercise report a false negative forever.
	path := filepath.Join(root, "vrooli-chaos-writer.bin")
	completed := false
	defer func() {
		if !completed {
			// The artifact belongs exclusively to this bounded validation
			// primitive. Do not leave it behind when sampling never produces
			// the expected rate-triggered run.
			_ = os.Remove(path)
		}
	}()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return chaosResult{}, fmt.Errorf("open chaos writer: %w", err)
	}
	started := time.Now().UTC()
	bytes, writeErr := writeChaos(context.Background(), f, rate, duration)
	closeErr := f.Close()
	ended := time.Now().UTC()
	if writeErr != nil {
		return chaosResult{}, writeErr
	}
	if closeErr != nil {
		return chaosResult{}, fmt.Errorf("close chaos writer: %w", closeErr)
	}
	runID, err := findChaosRecovery(context.Background(), h.client, root, started)
	if err != nil {
		return chaosResult{}, err
	}

	terminal, err := h.client.WaitRecovery(context.Background(), connect.NewRequest(&cleanupv1.RecoveryWaitRequest{RunId: runID}))
	if err != nil {
		return chaosResult{}, cliapp.WrapAPIError("wait for chaos recovery", err, nil)
	}
	// The chaos proof is only successful when the system reclaimed the
	// generated artifact. A completed run by itself is insufficient: it could
	// have acted on another provider while the synthetic writer remained.
	if _, statErr := os.Stat(path); statErr == nil {
		return chaosResult{}, fmt.Errorf("chaos recovery completed but writer artifact remains: %s", path)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return chaosResult{}, fmt.Errorf("verify chaos writer reclaim: %w", statErr)
	}
	completed = true
	return chaosResult{Root: root, Path: path, Rate: rate, BytesWritten: bytes, WriterStarted: started.Format(time.RFC3339Nano), WriterEnded: ended.Format(time.RFC3339Nano), RecoveryRun: runID, RecoveryState: terminal.Msg.GetStatus(), Reclaimed: terminal.Msg.GetReclaimedBytes(), TargetFree: terminal.Msg.GetTargetFreeBytes(), StoppedBecause: terminal.Msg.GetStoppedBecause()}, nil
}

func findChaosRecovery(ctx context.Context, client cleanupconnect.CleanupServiceClient, root string, after time.Time) (string, error) {
	resp, err := client.ListRecovery(ctx, connect.NewRequest(&cleanupv1.RecoveryHistoryRequest{Limit: 50}))
	if err != nil {
		return "", cliapp.WrapAPIError("list autonomous chaos recovery", err, nil)
	}
	for _, run := range resp.Msg.GetRuns() {
		if run.GetPartition() != root || !strings.Contains(strings.ToLower(run.GetTrigger()), "rate") {
			continue
		}
		if ts := run.GetStartedAt(); ts != nil && ts.AsTime().After(after.Add(-time.Second)) {
			return run.GetRunId(), nil
		}
	}
	return "", fmt.Errorf("no autonomous rate-triggered recovery was recorded for chaos root %s", root)
}

func chaosReport(_ cliapp.OperationContext, result chaosResult) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Chaos writer reclaimed %d bytes; recovery %s is %s (stopped=%s, target_free=%d).", result.Reclaimed, result.RecoveryRun, result.RecoveryState, result.StoppedBecause, result.TargetFree)}, Changes: []string{fmt.Sprintf("wrote %d bytes at %d bytes/hour under %s", result.BytesWritten, result.Rate, result.Root)}}
}

func governedChaosRoot(raw string) (string, error) {
	root := filepath.Clean(strings.TrimSpace(raw))
	if root == "." || !filepath.IsAbs(root) {
		return "", errors.New("--root must be an absolute path")
	}
	base := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home: %w", err)
		}
		base = filepath.Join(home, ".vrooli")
	}
	// The current governed temporary-root declaration is the lifecycle-owned
	// go-work namespace. Keep the chaos primitive inside that exact provider
	// root so the subsequent rate recovery can select it.
	base = filepath.Join(filepath.Clean(base), "tmp", "go-work")
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("--root must be beneath governed temporary root %s", base)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat governed chaos root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("governed chaos root is not a directory: %s", root)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve governed chaos root: %w", err)
	}
	if resolved != root {
		return "", fmt.Errorf("--root must not be a symlink: %s", root)
	}
	return root, nil
}

func parseChaosRate(raw string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if !strings.HasSuffix(s, "/h") {
		return 0, errors.New("--rate must use a per-hour suffix, for example 20gib/h")
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, "/h"))
	units := []struct {
		suffix string
		value  int64
	}{
		{"gib", 1 << 30}, {"gb", 1_000_000_000}, {"mib", 1 << 20}, {"mb", 1_000_000}, {"kib", 1 << 10}, {"kb", 1_000}, {"b", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(s, unit.suffix) {
			n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, unit.suffix)), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid rate %q", raw)
			}
			value := int64(n * float64(unit.value))
			if value <= 0 || value > chaosMaxRate {
				return 0, fmt.Errorf("--rate must be within (0,%d] bytes/hour", chaosMaxRate)
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("invalid rate %q", raw)
}

func writeChaos(ctx context.Context, dst io.Writer, rate int64, duration time.Duration) (int64, error) {
	interval := time.Duration(float64(time.Hour*time.Duration(chaosChunk)) / float64(rate))
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	deadline := time.Now().Add(duration)
	buf := make([]byte, int(chaosChunk))
	var total int64
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		n, err := writeChaosChunk(dst, buf, total)
		total += int64(n)
		if err != nil {
			return total, fmt.Errorf("write chaos data: %w", err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return total, ctx.Err()
		case <-timer.C:
		}
	}
	return total, nil
}
