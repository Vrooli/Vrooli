package bundle

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

type fakeRemoteBundles struct {
	files map[string]struct {
		size int64
		mt   int64
	}
	rmCalls int
}

type runnerFunc func(context.Context, ssh.Config, string, ssh.RunOptions) (ssh.Result, error)

func (f runnerFunc) Run(ctx context.Context, cfg ssh.Config, cmd string, opts ssh.RunOptions) (ssh.Result, error) {
	return f(ctx, cfg, cmd, opts)
}

func (r *fakeRemoteBundles) runner() ssh.Runner {
	return runnerFunc(func(_ context.Context, _ ssh.Config, cmd string, _ ssh.RunOptions) (ssh.Result, error) {
		if strings.Contains(cmd, "stat --printf") {
			var b strings.Builder
			for name, meta := range r.files {
				b.WriteString(strconv.FormatInt(meta.size, 10))
				b.WriteByte('\t')
				b.WriteString(name)
				b.WriteByte('\t')
				b.WriteString(strconv.FormatInt(meta.mt, 10))
				b.WriteByte('\n')
			}
			return ssh.Result{Stdout: b.String(), ExitCode: 0}, nil
		}

		if strings.Contains(cmd, " rm -f -- ") {
			r.rmCalls++
			fields := strings.Fields(cmd)
			for i := 0; i < len(fields); i++ {
				if fields[i] != "--" {
					continue
				}
				for j := i + 1; j < len(fields); j++ {
					fn := strings.Trim(fields[j], "'")
					delete(r.files, fn)
				}
				break
			}
			return ssh.Result{Stdout: "", ExitCode: 0}, nil
		}

		return ssh.Result{Stdout: "", ExitCode: 0}, nil
	})
}

func TestGCVPSBundles_DryRunReturnsPlanAndDoesNotDelete(t *testing.T) {
	r := &fakeRemoteBundles{
		files: map[string]struct {
			size int64
			mt   int64
		}{
			"mini-vrooli_app_" + strings.Repeat("a", 64) + ".tar.gz": {size: 100, mt: 1000},
			"mini-vrooli_app_" + strings.Repeat("b", 64) + ".tar.gz": {size: 200, mt: 2000},
			"mini-vrooli_app_" + strings.Repeat("c", 64) + ".tar.gz": {size: 300, mt: 3000},
		},
	}

	cfg := ssh.NewConfig("example.com", 22, "root", "/tmp/key")
	resp := GCVPSBundles(context.Background(), r.runner(), cfg, "/root/Vrooli", domain.VPSBundleGCRequest{
		ScenarioID: "app",
		KeepLatest: 2,
		DryRun:     true,
	})

	if !resp.OK || !resp.DryRun {
		t.Fatalf("expected ok dry-run, got ok=%v dry=%v err=%q", resp.OK, resp.DryRun, resp.Error)
	}
	if resp.DeletedCount != 1 {
		t.Fatalf("expected 1 planned deletion, got %d", resp.DeletedCount)
	}
	if r.rmCalls != 0 {
		t.Fatalf("expected no rm calls in dry-run, got %d", r.rmCalls)
	}
	if len(r.files) != 3 {
		t.Fatalf("expected no deletion, got %d files", len(r.files))
	}
}

func TestGCVPSBundles_ExecDeletesInBatches(t *testing.T) {
	files := make(map[string]struct {
		size int64
		mt   int64
	})
	now := time.Now().Unix()
	for i := 0; i < 120; i++ {
		sha := fmt.Sprintf("%064x", i)
		name := "mini-vrooli_app_" + sha + ".tar.gz"
		files[name] = struct {
			size int64
			mt   int64
		}{size: 10, mt: now + int64(i)}
	}
	r := &fakeRemoteBundles{files: files}

	cfg := ssh.NewConfig("example.com", 22, "root", "/tmp/key")
	resp := GCVPSBundles(context.Background(), r.runner(), cfg, "/root/Vrooli", domain.VPSBundleGCRequest{
		ScenarioID: "app",
		KeepLatest: 2,
		DryRun:     false,
	})
	if !resp.OK {
		t.Fatalf("expected ok, got %q", resp.Error)
	}
	if resp.DeletedCount != 118 {
		t.Fatalf("expected 118 deleted, got %d", resp.DeletedCount)
	}
	// 118 deletions with batch size 50 => 3 rm calls.
	if r.rmCalls != 3 {
		t.Fatalf("expected 3 rm calls (batching), got %d", r.rmCalls)
	}
	if len(r.files) != 2 {
		t.Fatalf("expected 2 files kept, got %d", len(r.files))
	}
}

func TestGCVPSBundles_RefusesUnsafeFilenames(t *testing.T) {
	unsafeSHA := strings.Repeat("a", 62) + ".." // 64 chars, includes ".." => refused by isSafeBundleFilename
	r := &fakeRemoteBundles{
		files: map[string]struct {
			size int64
			mt   int64
		}{
			"mini-vrooli_app_" + strings.Repeat("a", 64) + ".tar.gz": {size: 100, mt: 2000}, // keep
			"mini-vrooli_app_" + unsafeSHA + ".tar.gz":               {size: 100, mt: 1000}, // should be selected for deletion (older) but refused
		},
	}

	cfg := ssh.NewConfig("example.com", 22, "root", "/tmp/key")
	resp := GCVPSBundles(context.Background(), r.runner(), cfg, "/root/Vrooli", domain.VPSBundleGCRequest{
		ScenarioID: "app",
		KeepLatest: 1,
		DryRun:     false,
	})
	if resp.OK {
		t.Fatalf("expected refusal due to unsafe filename, got ok=true")
	}
	if !strings.Contains(resp.Error, "unsafe filename") {
		t.Fatalf("expected unsafe filename error, got %q", resp.Error)
	}
	if r.rmCalls != 0 {
		t.Fatalf("expected no rm calls, got %d", r.rmCalls)
	}
}
