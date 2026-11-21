//go:build linux

package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func openRGBStagePreparationOwner() (openRGBStagePreparation, error) {
	return openRGBStagePreparation{
		extract: extractOpenRGBStage, seal: sealOpenRGBStage, verifySeal: verifyOpenRGBAppDirSeal,
		publish: func(source, destination string) error {
			return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
		},
		syncDir: syncOpenRGBStageDirectory,
	}, nil
}

func extractOpenRGBStage(ctx context.Context, archive *os.File, destination string) error {
	profile, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return err
	}
	return extractOpenRGBStageForProfile(ctx, archive, destination, profile)
}

func extractOpenRGBStageForProfile(ctx context.Context, archive *os.File, destination string, profile openRGBReleaseProfile) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if profile.OS != "linux" || profile.validate() != nil {
		return errOpenRGBReleasePayloadUnqualified
	}

	toolInfo, err := os.Stat("/usr/bin/unsquashfs")
	if err != nil {
		return fmt.Errorf("offline OpenRGB setup requires `scenario-dependency-analyzer deps install system/squashfs-tools --scenario squashfs-tools --surface host-tool --apply`: %w", err)
	}
	stat, ok := toolInfo.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || !toolInfo.Mode().IsRegular() || toolInfo.Mode().Perm()&0o022 != 0 {
		return errors.New("untrusted native unsquashfs extraction tool")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/unsquashfs", "-no-progress", "-processors", "1", "-no-xattrs", "-offset", strconv.FormatInt(profile.SquashFSOffset, 10), "-d", destination, "/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{archive}
	// A terminated setup owner must not leave an extractor writing into the
	// pending path after the per-stage lock is released by process exit.
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
	cmd.Dir = filepath.Dir(destination)
	cmd.WaitDelay = time.Second
	var output openRGBExtractorOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract verified OpenRGB AppImage: %w: %s", errors.Join(err, ctx.Err()), output.data)
	}
	return nil
}

type openRGBExtractorOutput struct{ data []byte }

func (w *openRGBExtractorOutput) Write(data []byte) (int, error) {
	remaining := (16 << 10) - len(w.data)
	w.data = append(w.data, data[:min(len(data), remaining)]...)
	return len(data), nil
}

func prepareOpenRGBNativeArtifactParents(string) error { return nil }
