package pyenv

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// VerifyPrepared verifies an already provisioned environment against an approved
// lock digest. Unlike Ensure it never creates files, invokes uv, or repairs it.
// The caller owns approval of the runtime location and lock digest. The sentinel
// is a freshness check, not an integrity or provenance attestation of CPython.
func VerifyPrepared(spec Spec, approvedLockHash string) (Interpreter, error) {
	if !filepath.IsAbs(spec.VenvDir) || !filepath.IsAbs(spec.LockFile) || spec.BasePython != "3.12" || len(approvedLockHash) != 64 {
		return Interpreter{}, errors.New("pyenv: approved prepared CPython 3.12 required")
	}
	lock, err := readPreparedFile(spec.LockFile, 1<<20)
	if err != nil || len(strings.TrimSpace(string(lock))) == 0 || hashBytes(lock) != approvedLockHash {
		return Interpreter{}, errors.New("pyenv: prepared lock mismatch")
	}
	path := InterpreterPath(spec.VenvDir)
	recorded, sentinelErr := readPreparedFile(sentinelPath(spec.VenvDir), 128)
	if sentinelErr != nil || strings.TrimSpace(string(recorded)) != approvedLockHash {
		return Interpreter{}, errors.New("pyenv: prepared environment unavailable")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return Interpreter{}, errors.New("pyenv: prepared interpreter unavailable")
	}
	return Interpreter{Python: path, LockHash: approvedLockHash}, nil
}

// Bound freshness metadata as well as worker output. This is not a runtime
// integrity attestation; its location and immutability remain owner-approved.
func readPreparedFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errors.New("pyenv: invalid prepared metadata")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("pyenv: changed prepared metadata")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(raw)) > maximum {
		return nil, errors.New("pyenv: prepared metadata exceeds limit")
	}
	return raw, nil
}
