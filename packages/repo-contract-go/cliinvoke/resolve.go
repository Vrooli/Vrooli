package cliinvoke

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// BinaryEnvVar names the environment variable an operator or a unit file can
// set to point every invoker at one vrooli binary.
const BinaryEnvVar = "VROOLI_BIN"

// ResolveOptions narrows the fixed resolution order. Every field is optional.
type ResolveOptions struct {
	// Explicit is a path the caller was configured with (a --vrooli-bin flag,
	// a retired VROOLI_CMD_PATH override). It wins when it exists.
	Explicit string
	// RuntimeHome is the vrooli runtime home (~/.vrooli by default). When set,
	// its bin entry from the repo contract is tried before PATH.
	RuntimeHome string
	// Getenv, Stat and LookPath are test seams; nil means the os defaults.
	Getenv   func(string) string
	Stat     func(string) (os.FileInfo, error)
	LookPath func(string) (string, error)
}

// BinaryMissingError is returned when no candidate resolves. Candidates are
// listed in the order they were tried so an operator can see which seam was
// expected to supply the binary.
type BinaryMissingError struct {
	Candidates []string
}

func (e *BinaryMissingError) Error() string {
	return fmt.Sprintf("vrooli binary not found; tried %s", strings.Join(e.Candidates, ", "))
}

// ErrBinaryMissing is the sentinel that BinaryMissingError unwraps to.
var ErrBinaryMissing = errors.New("vrooli binary missing")

func (e *BinaryMissingError) Unwrap() error { return ErrBinaryMissing }

// Resolve finds the vrooli binary in one fixed order: the explicit path, the
// VROOLI_BIN environment variable, the runtime home's bin entry from the repo
// contract, then PATH. There is no newest-mtime heuristic: freshness belongs
// to the lifecycle engine, not to the caller.
func Resolve(opts ResolveOptions) (string, error) {
	getenv, stat, lookPath := opts.Getenv, opts.Stat, opts.LookPath
	if getenv == nil {
		getenv = os.Getenv
	}
	if stat == nil {
		stat = os.Stat
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	var tried []string
	isFile := func(path string) bool {
		info, err := stat(path)
		return err == nil && !info.IsDir()
	}
	for _, candidate := range []string{strings.TrimSpace(opts.Explicit), strings.TrimSpace(getenv(BinaryEnvVar))} {
		if candidate == "" {
			continue
		}
		tried = append(tried, candidate)
		if isFile(candidate) {
			return candidate, nil
		}
	}
	if home := strings.TrimSpace(opts.RuntimeHome); home != "" {
		if binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin); err == nil {
			candidate := filepath.Join(binDir, binaryName())
			tried = append(tried, candidate)
			if isFile(candidate) {
				return candidate, nil
			}
		}
	}
	tried = append(tried, "PATH")
	if path, err := lookPath(binaryName()); err == nil && strings.TrimSpace(path) != "" {
		return path, nil
	}
	return "", &BinaryMissingError{Candidates: tried}
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "vrooli.exe"
	}
	return "vrooli"
}
