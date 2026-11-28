package platform

import (
	"context"
	"debug/macho"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type darwinImageExecEngine struct {
	check  func(string, bool) error
	path   func(context.Context, string, bool) error
	format func(*os.File) error
	exec   func(string, []string, []string) error
}

func verifyDarwinImagePath(ctx context.Context, path string, directory bool, check func(string, bool) error) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return ErrOperationIdentity
	}
	parts := []string{}
	for p := path; ; p = filepath.Dir(p) {
		parts = append(parts, p)
		if p == filepath.Dir(p) {
			break
		}
	}
	for n := len(parts) - 1; n >= 0; n-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := check(parts[n], n != 0 || directory); err != nil {
			return fmt.Errorf("platform: unsealed native image path %s: %w", parts[n], err)
		}
	}
	return nil
}
func (e darwinImageExecEngine) replace(file *os.File, argv, env []string) error {
	if file == nil || len(argv) == 0 || env == nil {
		return ErrOperationIdentity
	}
	// The mapped-image observer's File.Name is intentionally not an execution
	// path. Only an explicit canonical argv[0] can be admitted by this seam.
	path := argv[0]
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrOperationIdentity
	}
	if len(argv) > maxOperationArgs+1 {
		return ErrOperationIdentity
	}
	total := 0
	for _, arg := range argv {
		if strings.ContainsRune(arg, '\x00') {
			return ErrOperationIdentity
		}
		total += len(arg)
		if total > maxOperationStringBytes {
			return ErrOperationIdentity
		}
	}
	seen := map[string]bool{}
	total = 0
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if !ok || k == "" || strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, '\x00') || seen[k] {
			return ErrOperationIdentity
		}
		seen[k] = true
		total += len(entry)
		if total > maxOperationEnvBytes {
			return ErrOperationIdentity
		}
		if strings.HasPrefix(k, "DYLD_") {
			return fmt.Errorf("platform: ambient dyld search/injection is forbidden for sealed image handoff")
		}
	}
	original, err := file.Stat()
	if err != nil {
		return err
	}
	if !original.Mode().IsRegular() || original.Mode().Perm()&0222 != 0 || original.Mode().Perm()&0111 == 0 {
		return ErrOperationIdentity
	}
	if err = e.path(context.Background(), path, false); err != nil {
		return err
	}
	if err = e.format(file); err != nil {
		return err
	}
	// The complete path is sealed against every nonroot writer. A fresh vnode
	// equality check excludes replacement since the caller verified this FD.
	named, err := os.Open(path)
	if err != nil {
		return err
	}
	defer named.Close()
	observed, err := named.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(original, observed) {
		return ErrOperationIdentity
	}
	if err = e.check(path, false); err != nil {
		return err
	}
	latest, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if latest.Mode().Perm()&0222 != 0 || !os.SameFile(original, latest) {
		return ErrOperationIdentity
	}
	return e.exec(path, append([]string(nil), argv...), append([]string{}, env...))
}
func validateDarwinNativeExecutable(file *os.File, architecture string) error {
	valid := func(f *macho.File) bool {
		return f.Type == macho.TypeExec && (architecture == "amd64" && f.Cpu == macho.CpuAmd64 || architecture == "arm64" && f.Cpu == macho.CpuArm64)
	}
	f, err := macho.NewFile(file)
	if err == nil {
		defer f.Close()
		if !valid(f) {
			return ErrOperationIdentity
		}
		return nil
	}
	fat, err := macho.NewFatFile(file)
	if err != nil {
		return fmt.Errorf("platform: native Mach-O image required; scripts/interpreters unsupported: %w", err)
	}
	defer fat.Close()
	for _, arch := range fat.Arches {
		if valid(arch.File) {
			return nil
		}
	}
	return ErrOperationIdentity
}
