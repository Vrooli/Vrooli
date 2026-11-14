package config

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	platform "github.com/vrooli/platform-go"
)

// PrivatePathPolicy names the two explicit principals allowed in a native
// resource's private subtree. An empty policy means the current process only.
// It never means all users, all administrators, or all service accounts.
type PrivatePathPolicy struct {
	OwnerPrincipal      string
	RuntimePrincipal    string
	RuntimeReadOnly     bool
	RuntimeTraverseOnly bool
	NoInherit           bool
}

type privateFileTemporary struct {
	File    *os.File
	Name    string
	Publish func(string) error
	Cleanup func()
}

func verifyPrivateOwner(actual, owner, runtime, system string, p PrivatePathPolicy) error {
	if (p.RuntimeReadOnly || p.RuntimeTraverseOnly) && runtime != "" && runtime != owner && actual == runtime {
		return errors.New("read-only runtime principal must not own the private object")
	}
	if actual != owner && actual != runtime && actual != system {
		return errors.New("private path owner is outside its explicit principal policy")
	}
	return nil
}

func privateTemporaryName() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return ".vrooli-private-" + hex.EncodeToString(random[:]), nil
}

// PrivatePathOwner returns the OS security owner of the same object described
// by info. A descriptor file cannot establish its own owner by self-assertion.
func PrivatePathOwner(path string, info os.FileInfo) (string, error) {
	if info == nil {
		return "", errors.New("private path owner requires file identity")
	}
	principal, err := nativePrivatePathOwner(path, info)
	if err != nil {
		return "", err
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return "", errors.Join(errors.New("private path identity changed while checking owner"), err)
	}
	return principal, nil
}

func VerifyPrivatePath(path string, info os.FileInfo, p PrivatePathPolicy) error {
	if info == nil || info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
		return errors.New("private path is an alias or unsupported file type")
	}
	if err := verifyNativePrivatePath(path, info, p); err != nil {
		return err
	}
	return confirmPrivatePathIdentity(path, info)
}

// VerifyNativePathAccess checks native ownership and ACL authority independently
// of POSIX public-read bits. Sealed-artifact callers separately enforce their
// immutable file modes and byte/tree identities.
func VerifyNativePathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	if info == nil || info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
		return errors.New("native path is an alias or unsupported file type")
	}
	if err := verifyNativePathAccess(path, info, p); err != nil {
		return err
	}
	return confirmPrivatePathIdentity(path, info)
}
func confirmPrivatePathIdentity(path string, info os.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return errors.Join(errors.New("native private path identity changed during authority inspection"), err)
	}
	return nil
}
func ConfigureNativeReadOnlyPathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	if !p.RuntimeReadOnly || p.RuntimeTraverseOnly || !p.NoInherit {
		return errors.New("sealed native access requires an explicit read-only non-inheriting policy")
	}
	if info == nil || info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
		return errors.New("sealed native path has an unsupported type")
	}
	return configureNativeReadOnlyPathAccess(path, info, p)
}

// EnsurePrivateDirectory creates missing parents for the current owner and
// configures only path for the explicit runtime principal. Existing foreign
// grants are rejected; this does not repair or normalize unrelated ACLs.
func EnsurePrivateDirectory(path string, p PrivatePathPolicy) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("private directory requires a canonical absolute path")
	}
	if err := ensureNativePrivateDirectory(path, p); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("private directory is not a directory")
	}
	return VerifyPrivatePath(path, info, p)
}

// WritePrivateFileDurable keeps its lock within the already private owner
// subtree. It is usable by a provisioned daemon without opening the operator's
// unrelated HOME/state lock namespace.
func WritePrivateFileDurable(path string, data []byte, perm os.FileMode, p PrivatePathPolicy) error {
	return WritePrivateFileDurableStream(path, perm, p, func(w io.Writer) error { _, err := io.Copy(w, bytes.NewReader(data)); return err })
}

func RemovePrivateFileDurable(path string, p PrivatePathPolicy) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = VerifyPrivatePath(path, info, p); err != nil {
		return err
	}
	return removeNativePrivateFile(path)
}

// AcquirePrivateFileLock uses the same explicit native principal policy and
// local directory as the durable record. It never opens a HOME-based lock
// namespace for a machine broker acting on an operator's behalf.
func AcquirePrivateFileLock(ctx context.Context, path string, p PrivatePathPolicy) (func(), error) {
	if ctx == nil {
		return nil, errors.New("private lock context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := EnsurePrivateDirectory(filepath.Dir(path), p); err != nil {
		return nil, err
	}
	file, closeFile, err := openNativePrivateLock(path, p)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		closeFile()
		return nil, err
	}
	if err = VerifyPrivatePath(path, info, p); err != nil {
		closeFile()
		return nil, err
	}
	unlock, err := platform.LockFileContext(ctx, file)
	if err != nil {
		closeFile()
		return nil, err
	}
	if err = platform.VerifyLockFile(file, path); err != nil {
		unlock()
		closeFile()
		return nil, err
	}
	var once sync.Once
	return func() { once.Do(func() { unlock(); closeFile() }) }, nil
}
func WritePrivateFileDurableStream(path string, perm os.FileMode, p PrivatePathPolicy, write func(io.Writer) error) error {
	if write == nil || perm&0o077 != 0 {
		return errors.New("private writer requires an owner-only mode and callback")
	}
	parent := filepath.Dir(path)
	if err := EnsurePrivateDirectory(parent, p); err != nil {
		return err
	}
	if existing, inspectErr := os.Lstat(path); inspectErr == nil {
		if !existing.Mode().IsRegular() {
			return errors.New("private publication target is not a regular file")
		}
		if err := VerifyPrivatePath(path, existing, p); err != nil {
			return err
		}
	} else if !os.IsNotExist(inspectErr) {
		return inspectErr
	}
	lockPath := path + ".write.lock"
	lockFile, closeLock, err := openNativePrivateLock(lockPath, p)
	if err != nil {
		return err
	}
	defer closeLock()
	lockInfo, err := lockFile.Stat()
	if err != nil {
		return err
	}
	if err = VerifyPrivatePath(path+".write.lock", lockInfo, p); err != nil {
		return fmt.Errorf("private publication lock: %w", err)
	}
	release, err := platform.LockFile(lockFile, false)
	if err != nil {
		return err
	}
	defer release()
	temporary, err := createNativePrivateTemp(parent, p)
	if err != nil {
		return err
	}
	tmp, name := temporary.File, temporary.Name
	defer temporary.Cleanup()
	if err = tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err = configurePrivateWriterAccess(tmp, p); err != nil {
		tmp.Close()
		return err
	}
	if err = write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err = assignPrivateWriterOwner(tmp, p); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	expected, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return err
	}
	if err = VerifyPrivatePath(name, expected, p); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = temporary.Publish(path); err != nil {
		return err
	}
	actual, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !os.SameFile(expected, actual) {
		return errors.New("private publication identity changed")
	}
	return VerifyPrivatePath(path, actual, p)
}
