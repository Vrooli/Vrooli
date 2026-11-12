package operatorstate

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vrooli/vrooli/internal/config"
)

func (s *Service) nativePrivateStorage() bool {
	return s.cfg.LogicalOperatorHome != "" && (runtime.GOOS == "darwin" || runtime.GOOS == "windows")
}

func (s *Service) nativeStatePolicy() (config.PrivatePathPolicy, error) {
	home := s.cfg.LogicalOperatorHome
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return config.PrivatePathPolicy{}, errors.New("operator state requires a canonical logical operator home")
	}
	info, err := os.Lstat(home)
	if err != nil {
		return config.PrivatePathPolicy{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return config.PrivatePathPolicy{}, errors.New("operator home is an alias or non-directory")
	}
	owner, err := config.PrivatePathOwner(home, info)
	return config.PrivatePathPolicy{OwnerPrincipal: owner}, err
}

func (s *Service) prepareNativeStateDirectory(path string, p config.PrivatePathPolicy) error {
	var missing []string
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("operator state has an aliased parent")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, parent)
		if filepath.Dir(parent) == parent {
			return errors.New("operator state has no existing owner namespace")
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := config.ConfigurePrivateDirectoryAccess(missing[i], p); err != nil {
			return err
		}
	}
	return config.ConfigurePrivateDirectoryAccess(filepath.Dir(path), p)
}

func (s *Service) acquireStateFileLock(ctx context.Context, path string) (func(), error) {
	if !s.nativePrivateStorage() {
		return acquireStateFileLock(ctx, path)
	}
	p, err := s.nativeStatePolicy()
	if err != nil {
		return nil, err
	}
	if err = s.prepareNativeStateDirectory(path, p); err != nil {
		return nil, err
	}
	return config.AcquirePrivateFileLock(ctx, filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".lock"), p)
}

func (s *Service) writeNativeStateFile(path string, data []byte) error {
	p, err := s.nativeStatePolicy()
	if err != nil {
		return err
	}
	return config.WritePrivateFileDurable(path, data, 0o600, p)
}

func (s *Service) readStateFile(path string) ([]byte, error) {
	if !s.nativePrivateStorage() {
		return os.ReadFile(path)
	}
	p, err := s.nativeStatePolicy()
	if err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("operator state is not a regular private file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, info) {
		return nil, errors.New("operator state changed while opening")
	}
	if err = config.VerifyPrivatePath(path, info, p); err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}
