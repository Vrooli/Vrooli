//go:build linux

package resources

import (
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func verifyOpenRGBReleaseRuntimePrerequisites(ctx context.Context, bundle string) error {
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return err
	}
	payload := filepath.Join(bundle, filepath.FromSlash(p.Payload))
	if err := p.verifyPayloadABI(payload); err != nil {
		return err
	}
	main, err := elf.Open(payload)
	if err != nil {
		return err
	}
	defer main.Close()
	interpreter := ""
	for _, program := range main.Progs {
		if program.Type == elf.PT_INTERP {
			data, err := io.ReadAll(io.LimitReader(program.Open(), 4097))
			if err != nil || len(data) > 4096 {
				return errors.Join(errors.New("OpenRGB ELF interpreter is unbounded"), err)
			}
			interpreter = strings.TrimRight(string(data), "\x00")
		}
	}
	expected := "/lib64/ld-linux-x86-64.so.2"
	if p.Arch == "arm64" {
		expected = "/lib/ld-linux-aarch64.so.1"
	}
	if interpreter != expected {
		return errors.New("OpenRGB release uses an unexpected native interpreter")
	}
	if err := verifyOpenRGBSystemELF(ctx, expected, p); err != nil {
		return fmt.Errorf("OpenRGB native loader prerequisite: %w", err)
	}
	providers := map[string]map[string]bool{}
	return filepath.WalkDir(bundle, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		var magic [4]byte
		n, _ := f.Read(magic[:])
		f.Close()
		if n != 4 || magic != [4]byte{0x7f, 'E', 'L', 'F'} {
			return nil
		}
		image, err := elf.Open(path)
		if err != nil {
			return err
		}
		defer image.Close()
		libraries, err := image.ImportedLibraries()
		if err != nil {
			return err
		}
		needs, err := openRGBELFVersionNeeds(image)
		if err != nil {
			return err
		}
		for _, library := range libraries {
			if _, err := os.Lstat(filepath.Join(bundle, "usr", "lib", library)); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return err
			}
			exports, ok := providers[library]
			if !ok {
				system, err := openRGBSystemLibrary(p.Arch, library)
				if err != nil {
					return err
				}
				if err := verifyOpenRGBSystemELF(ctx, system, p); err != nil {
					return err
				}
				provider, err := elf.Open(system)
				if err != nil {
					return err
				}
				exports, err = openRGBELFVersionDefinitions(provider)
				provider.Close()
				if err != nil {
					return err
				}
				providers[library] = exports
			}
			for _, version := range needs[library] {
				if !exports[version] {
					return fmt.Errorf("OpenRGB %s requires system %s export version %s; the pinned release closure requires glibc2.39/GLIBCXX3.4.32/CXXABI1.3.15-compatible native providers", filepath.Base(path), library, version)
				}
			}
		}
		return nil
	})
}

func openRGBSystemLibrary(arch, name string) (string, error) {
	if filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00") {
		return "", errors.New("unsafe native ELF dependency name")
	}
	triplet := "x86_64-linux-gnu"
	if arch == "arm64" {
		triplet = "aarch64-linux-gnu"
	}
	for _, dir := range []string{filepath.Join("/lib", triplet), filepath.Join("/usr/lib", triplet), "/lib64", "/usr/lib64", "/lib", "/usr/lib"} {
		path := filepath.Join(dir, name)
		if _, err := os.Lstat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("OpenRGB native OS dependency %s is missing from supported system library roots", name)
}

func verifyOpenRGBSystemELF(ctx context.Context, path string, p openRGBReleaseProfile) error {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(real) {
		return errors.New("native system dependency is not absolute")
	}
	for at := real; ; at = filepath.Dir(at) {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(at)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 || info.Mode().Perm()&0o22 != 0 || at == real && !info.Mode().IsRegular() {
			return errors.New("native system loader/library has writable or non-root provenance")
		}
		if at == string(filepath.Separator) {
			break
		}
	}
	return p.verifyImageABI(real, false)
}
