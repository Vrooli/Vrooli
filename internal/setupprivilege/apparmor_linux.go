//go:build linux

package setupprivilege

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/vrooli/vrooli/internal/apparmorprofiles"
	"github.com/vrooli/vrooli/internal/tuning"
)

func RunSystemAppArmorProfile(ctx context.Context, id string) error {
	profile, ok := apparmorprofiles.Lookup(id)
	if !ok {
		return errors.New("unsupported AppArmor profile")
	}
	commandCtx, cancel := context.WithTimeout(ctx, tuning.SetupExtendedOperationTimeout())
	defer cancel()
	if err := commandCtx.Err(); err != nil {
		return err
	}
	parser := resolveAppArmorParser(profile.Parser)
	if parser == "" {
		return errors.New("apparmor_parser is required to load the managed profile")
	}
	args := append(append([]string(nil), profile.ParserArgs...), profile.Path)
	return installAndLoadAppArmorProfile(
		func() error { return installAppArmorProfile(profile) },
		func() error { return runBoundedCommand(commandCtx, parser, args...) },
	)
}

func resolveAppArmorParser(candidates []string) string {
	for _, candidate := range candidates {
		if !filepath.IsAbs(candidate) {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return candidate
		}
	}
	return ""
}

func installAndLoadAppArmorProfile(install, load func() error) error {
	if err := install(); err != nil {
		return err
	}
	if err := load(); err != nil {
		return fmt.Errorf("profile written but apparmor_parser did not verify it: %w", err)
	}
	return nil
}

func installAppArmorProfile(profile apparmorprofiles.Profile) error {
	return installAppArmorProfileAt(apparmorprofiles.ProfileDirectory, profile, 0)
}

func installAppArmorProfileAt(parent string, profile apparmorprofiles.Profile, expectedUID uint32) error {
	dirFD, err := syscall.Open(parent, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open fixed AppArmor profile directory: %w", err)
	}
	defer syscall.Close(dirFD)
	var dir syscall.Stat_t
	if err := syscall.Fstat(dirFD, &dir); err != nil {
		return err
	}
	if dir.Uid != expectedUID || dir.Mode&0022 != 0 {
		return errors.New("fixed AppArmor profile directory has unsafe owner or mode")
	}
	name := filepath.Base(profile.Path)
	if name == "." || name == "/" || name == ".." {
		return errors.New("invalid registered AppArmor profile path")
	}
	if fd, err := syscall.Openat(dirFD, name, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0); err == nil {
		file := os.NewFile(uintptr(fd), profile.Path)
		var target syscall.Stat_t
		statErr := syscall.Fstat(fd, &target)
		if statErr != nil {
			_ = file.Close()
			return statErr
		}
		if target.Mode&syscall.S_IFMT != syscall.S_IFREG || target.Uid != expectedUID || target.Mode&0022 != 0 {
			_ = file.Close()
			return errors.New("managed AppArmor profile target is not a safe root-owned regular file")
		}
		if profile.Replacement == apparmorprofiles.RefuseDifferent {
			data, readErr := io.ReadAll(file)
			_ = file.Close()
			if readErr == nil && string(data) != profile.Content {
				return fmt.Errorf("refusing to replace different existing policy at %s", profile.Path)
			}
			if readErr != nil {
				return readErr
			}
		} else {
			_ = file.Close()
		}
	} else if !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("inspect managed AppArmor profile: %w", err)
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	tmp := fmt.Sprintf(".vrooli-apparmor-%x", random[:])
	tmpFD, err := syscall.Openat(dirFD, tmp, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer syscall.Unlinkat(dirFD, tmp)
	writeErr := writeAllFD(tmpFD, []byte(profile.Content))
	if writeErr == nil {
		writeErr = syscall.Fchmod(tmpFD, 0644)
	}
	if writeErr == nil {
		writeErr = syscall.Fsync(tmpFD)
	}
	closeErr := syscall.Close(tmpFD)
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := syscall.Renameat(dirFD, tmp, dirFD, name); err != nil {
		return fmt.Errorf("replace managed AppArmor profile: %w", err)
	}
	return nil
}

func writeAllFD(fd int, content []byte) error {
	for len(content) > 0 {
		n, err := syscall.Write(fd, content)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		content = content[n:]
	}
	return nil
}
