package seeds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
)

const workerFlag = "--test-genie-seed-worker"
const maxWorkerImageBytes = 512 << 20

type workerInput struct {
	Version  int       `json:"version"`
	Path     string    `json:"path"`
	Args     []string  `json:"args"`
	Env      []string  `json:"env"`
	Dir      string    `json:"dir"`
	Deadline time.Time `json:"deadline"`
}
type workerGrant struct {
	Version   int                      `json:"version"`
	Intent    platform.OperationIntent `json:"intent"`
	Gate      platform.OperationGate   `json:"gate"`
	InputSHA  string                   `json:"input_sha"`
	WorkerSHA string                   `json:"worker_sha"`
	Claimed   bool                     `json:"claimed"`
}

func privateWorkerPaths(root, attempt string) (string, string, string, string) {
	return filepath.Join(root, attempt+".worker"), filepath.Join(root, attempt+".input.json"), filepath.Join(root, attempt+".grant.json"), filepath.Join(root, attempt+".log")
}
func hashFile(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, maxWorkerImageBytes+1))
	if err != nil {
		return "", err
	}
	if size > maxWorkerImageBytes {
		return "", errors.New("seed worker image exceeds owner bound")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func freezeWorker(path string) (string, error) {
	source, err := platform.OpenProcessExecutable(os.Getpid())
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0500)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(source, maxWorkerImageBytes+1))
	syncErr, closeErr := target.Sync(), target.Close()
	if size > maxWorkerImageBytes {
		copyErr = errors.Join(copyErr, errors.New("seed worker image exceeds owner bound"))
	}
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func writeWorkerJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("seed worker private input exceeds owner bound")
	}
	if err := storage.WriteFileAtomic(path, data, 0600); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func readWorkerJSON(path string, value any) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 1<<20 {
		return nil, errors.New("seed worker input is not bounded private owner state")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("seed worker input identity changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, errors.New("seed worker input exceeds owner bound")
	}
	if err := json.Unmarshal(data, value); err != nil {
		return nil, err
	}
	return data, nil
}

// RunWorker handles the frozen native worker before API or test initialization.
// Its argv contains paths and hashes only; child credentials live in a private
// input and exec.Cmd.Env, never in systemd ExecStart or public diagnostics.
func RunWorker(args []string, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != workerFlag {
		return false, 0
	}
	if len(args) != 3 {
		fmt.Fprintln(stderr, "seed native worker invocation refused")
		return true, 2
	}
	if err := executeWorker(args[1], args[2]); err != nil {
		fmt.Fprintln(stderr, "seed native worker refused or failed; inspect private owner evidence")
		return true, 1
	}
	return true, 0
}
func executeWorker(inputPath, inputSHA string) error {
	var input workerInput
	raw, err := readWorkerJSON(inputPath, &input)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != inputSHA || input.Version != 1 || !filepath.IsAbs(input.Path) || !filepath.IsAbs(input.Dir) || !time.Now().Before(input.Deadline) {
		return errors.New("seed worker private invocation identity is invalid")
	}
	attempt := strings.TrimSuffix(filepath.Base(inputPath), ".input.json")
	workerPath, expectedInput, grantPath, logPath := privateWorkerPaths(filepath.Dir(inputPath), attempt)
	if expectedInput != inputPath {
		return errors.New("seed worker input path identity mismatch")
	}
	lock, err := platform.OpenLockFile(filepath.Join(filepath.Dir(inputPath), attempt+".claim.lock"), 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	ctx, cancel := context.WithDeadline(context.Background(), input.Deadline)
	defer cancel()
	unlock, err := platform.LockFileContext(ctx, lock)
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	var grant workerGrant
	if _, err := readWorkerJSON(grantPath, &grant); err != nil {
		return err
	}
	if grant.Version != 1 || grant.Claimed || grant.InputSHA != inputSHA || grant.Intent.AttemptID != attempt || grant.Intent.ValidateIdentity() != nil || grant.Gate.ReadyPath != filepath.Join(filepath.Dir(inputPath), attempt+".ready") {
		return errors.New("seed worker grant does not match private input")
	}
	if revoked, err := grant.Gate.Revoked(); err != nil || revoked {
		return errors.New("seed worker execution gate is unavailable or revoked")
	}
	image, err := platform.OpenProcessExecutable(os.Getpid())
	if err != nil {
		return err
	}
	sha, err := hashFile(image)
	image.Close()
	if err != nil || sha != grant.WorkerSHA {
		return errors.New("seed worker image identity mismatch")
	}
	executable, err := os.Executable()
	if err != nil || executable != workerPath {
		return errors.New("seed worker is not the retained owner image")
	}
	ref, err := platform.LookupOwnedOperation(ctx, grant.Intent)
	if err != nil {
		return err
	}
	scope, err := platform.ProcessScope(os.Getpid())
	if err != nil || ref.RootPID != os.Getpid() || scope != ref.Scope.Path {
		return errors.New("seed worker is outside its admitted native operation")
	}
	observation, err := platform.InspectOwnedOperation(ctx, ref)
	if err != nil || !observation.RootAlive || !observation.Populated {
		return errors.New("seed worker native identity is unavailable")
	}
	grant.Claimed = true
	if err := writeWorkerJSON(grantPath, grant); err != nil {
		return err
	}
	unlock()
	locked = false
	// The native cgroup owns this worker and every child. Use direct child env
	// rather than a command-line env wrapper; bounded private logging prevents
	// child output from exposing credentials in the native service journal.
	log, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	writer := &privateSeedLog{file: log, remaining: 1 << 20}
	command := exec.CommandContext(ctx, input.Path, input.Args...)
	command.Dir = input.Dir
	command.Env = input.Env
	command.Stdout = writer
	command.Stderr = writer
	command.WaitDelay = time.Second
	return command.Run()
}

type privateSeedLog struct {
	mu        sync.Mutex
	file      *os.File
	remaining int
}

func (w *privateSeedLog) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	size := len(data)
	if len(data) > w.remaining {
		data = data[:w.remaining]
	}
	if len(data) > 0 {
		n, err := w.file.Write(data)
		w.remaining -= n
		if err != nil {
			return size, err
		}
	}
	return size, nil
}
