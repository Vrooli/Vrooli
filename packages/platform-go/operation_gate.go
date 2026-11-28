package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OperationGate is a revocable native pre-exec condition. It is not a dispatch
// claim and must never be used as permission to replay an OperationIntent.
// The domain persists it before submission and revokes it before interpreting
// complete native absence as safe retirement. Pending/visible units still need
// an exact native stop and recursive quiescence proof.
type OperationGate struct {
	ReadyPath string `json:"ready_path"`
}

func (g OperationGate) validate() error {
	if !filepath.IsAbs(g.ReadyPath) || filepath.Clean(g.ReadyPath) != g.ReadyPath || strings.ContainsAny(g.ReadyPath, "\x00\r\n") || len(g.ReadyPath) > 4096 {
		return errors.New("platform: operation gate requires a bounded canonical absolute path")
	}
	name := filepath.Base(g.ReadyPath)
	if !strings.HasSuffix(name, ".ready") || !operationAttemptPattern.MatchString(strings.TrimSuffix(name, ".ready")) {
		return errors.New("platform: operation gate requires an exact attempt identity")
	}
	return nil
}
func (g OperationGate) openRoot() (*os.Root, error) {
	if err := g.validate(); err != nil {
		return nil, err
	}
	dir := filepath.Dir(g.ReadyPath)
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("platform: operation gate directory must be private and must not be a symlink")
	}
	return os.OpenRoot(dir)
}

func (g OperationGate) validateLocation() error {
	root, err := g.openRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	ready, _ := g.names()
	data, err := gateFile(root, ready)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if string(data) != "ready\n" {
		return errors.New("platform: invalid operation gate readiness marker")
	}
	return nil
}
func (g OperationGate) names() (string, string) {
	name := filepath.Base(g.ReadyPath)
	return name, strings.TrimSuffix(name, ".ready") + ".revoked"
}

func (g OperationGate) lock() (func(), error) {
	ready, _ := g.names()
	path := filepath.Join(filepath.Dir(g.ReadyPath), strings.TrimSuffix(ready, ".ready")+".lock")
	file, err := OpenLockFile(path, 0600)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	unlock, err := LockFileContext(ctx, file)
	if err != nil {
		file.Close()
		return nil, err
	}
	if err := VerifyLockFile(file, path); err != nil {
		unlock()
		file.Close()
		return nil, err
	}
	return func() { unlock(); file.Close() }, nil
}
func syncGateRoot(root *os.Root) error {
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
func gateFile(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 256 {
		return nil, errors.New("platform: operation gate must be a bounded private regular file")
	}
	return root.ReadFile(name)
}

// PrepareOperationGate exclusively creates a never-reused per-attempt gate.
// The caller creates and owns dir. A durable tombstone forbids reopening a
// revoked gate; the API does not remove or recycle tombstones.
func PrepareOperationGate(dir, attemptID string) (OperationGate, error) {
	return prepareOperationGate(dir, attemptID, nil)
}

func prepareOperationGate(dir, attemptID string, afterCheck func()) (OperationGate, error) {
	gate := OperationGate{ReadyPath: filepath.Join(dir, attemptID+".ready")}
	if !operationAttemptPattern.MatchString(attemptID) {
		return OperationGate{}, errors.New("platform: invalid operation gate attempt")
	}
	root, err := gate.openRoot()
	if err != nil {
		return OperationGate{}, err
	}
	defer root.Close()
	unlock, err := gate.lock()
	if err != nil {
		return OperationGate{}, err
	}
	defer unlock()
	ready, revoked := gate.names()
	if _, err := root.Lstat(revoked); err == nil {
		return OperationGate{}, errors.New("platform: operation gate was revoked")
	} else if !errors.Is(err, os.ErrNotExist) {
		return OperationGate{}, err
	}
	if afterCheck != nil {
		afterCheck()
	}
	file, err := root.OpenFile(ready, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return OperationGate{}, err
	}
	_, writeErr := file.WriteString("ready\n")
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr, syncGateRoot(root)); err != nil {
		return OperationGate{}, err
	}
	return gate, nil
}

// Revoke is durable and idempotent. The tombstone is committed before ready is
// removed, so another Prepare cannot race revocation and recreate authority.
func (g OperationGate) Revoke() error {
	root, err := g.openRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	unlock, err := g.lock()
	if err != nil {
		return err
	}
	defer unlock()
	ready, revoked := g.names()
	// A prior process may have exited after creating the old marker but before
	// finishing its write. Revocation only removes execution authority, so a
	// bounded private regular marker may safely converge to canonical denial.
	// Aliases and non-private files remain errors; no external inode is edited.
	if _, err := gateFile(root, revoked); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := publishGateRevocation(root, revoked); err != nil {
		return err
	}
	if _, err := gateFile(root, ready); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := root.Remove(ready); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncGateRoot(root)
}

func publishGateRevocation(root *os.Root, name string) error {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	temporary := name + ".pending-" + hex.EncodeToString(nonce)
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	_, writeErr := file.WriteString("revoked\n")
	syncErr, closeErr := file.Sync(), file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := root.Rename(temporary, name); err != nil {
		return err
	}
	return syncGateRoot(root)
}

// Revoked proves durable revocation and readiness-file absence, not native
// quiescence. The domain separately observes the exact unit and cgroup.
func (g OperationGate) Revoked() (bool, error) {
	root, err := g.openRoot()
	if err != nil {
		return false, err
	}
	defer root.Close()
	unlock, err := g.lock()
	if err != nil {
		return false, err
	}
	defer unlock()
	ready, revoked := g.names()
	data, err := gateFile(root, revoked)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if string(data) != "revoked\n" {
		return false, errors.New("platform: invalid operation gate tombstone")
	}
	if _, err := root.Lstat(ready); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

// systemdExecArgument quotes one literal systemd-run ExecCondition argument.
// systemd-run handles unit-specifier escaping itself when encoding this D-Bus
// property; doubling percent here would change the actual filename. This is
// command argument quoting only and never invokes a shell.
func systemdExecArgument(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return fmt.Sprintf("\"%s\"", value)
}
