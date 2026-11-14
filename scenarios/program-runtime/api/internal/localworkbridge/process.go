//go:build linux

package localworkbridge

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/vrooli/pyenv-go"
)

const maxPacket = 192 << 10

func digestMatches(raw []byte, expected string) bool {
	sum := sha256.Sum256(raw)
	return len(expected) == 64 && hex.EncodeToString(sum[:]) == expected
}

func (r *ProcessRunner) Run(ctx context.Context, req Request) (Result, error) {
	if !r.enabled {
		return Result{}, ErrDisabled
	}
	if r.provider == nil {
		return Result{}, ErrProcess
	}
	if req.Operation != "accept_native" && req.Operation != "tick" {
		return Result{}, ErrProcess
	}
	if len(req.Body) > int(MaxBodyBytes) || len(req.Signature) > 512 || len(req.EventID) > 512 || len(req.IdempotencyKey) > 512 {
		return Result{}, ErrProcess
	}
	if req.Operation == "tick" && (len(req.Body) > 0 || req.Signature != "" || req.EventID != "" || req.IdempotencyKey != "") {
		return Result{}, ErrProcess
	}
	// Every direct use is bounded, even outside Host.invoke.
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	binding, err := r.provider.Resolve(ctx)
	if err != nil {
		return Result{}, ErrProcess
	}
	now := time.Now()
	if !binding.NotBefore.Before(binding.ExpiresAt) || now.Before(binding.NotBefore) || !now.Before(binding.ExpiresAt) || binding.ExpiresAt.Sub(binding.NotBefore) > 7*24*time.Hour {
		return Result{}, ErrProcess
	}
	if len(binding.Config) > 64<<10 || !digestMatches(binding.Config, binding.ConfigSHA256) {
		return Result{}, ErrProcess
	}
	var cfg struct {
		Enabled bool `json:"enabled"`
		Export  struct {
			Enabled bool `json:"enabled"`
		} `json:"export"`
	}
	if json.Unmarshal(binding.Config, &cfg) != nil || !cfg.Enabled {
		return Result{}, ErrDisabled
	}
	ctx, expire := context.WithDeadline(ctx, binding.ExpiresAt)
	defer expire()
	interp, err := pyenv.VerifyPrepared(binding.Runtime, binding.LockSHA256)
	if err != nil || !filepath.IsAbs(binding.EntryPath) {
		return Result{}, ErrProcess
	}
	if !verifyModules(binding) {
		return Result{}, ErrProcess
	}
	packet := struct {
		Schema    string `json:"schema"`
		Operation string `json:"operation"`
		Request   struct {
			Body           string `json:"body_base64"`
			Signature      string `json:"signature"`
			EventID        string `json:"event_id"`
			IdempotencyKey string `json:"idempotency_key"`
		} `json:"request"`
		Config json.RawMessage `json:"config"`
	}{Schema: "flywheel.local-work-native-ipc.v1", Operation: req.Operation, Config: binding.Config}
	packet.Request.Body = base64.StdEncoding.EncodeToString(req.Body)
	packet.Request.Signature = req.Signature
	packet.Request.EventID = req.EventID
	packet.Request.IdempotencyKey = req.IdempotencyKey
	input, err := json.Marshal(packet)
	if err != nil || len(input) > maxPacket {
		return Result{}, ErrProcess
	}
	cmd := exec.CommandContext(ctx, interp.Python, "-I", "-S", binding.EntryPath)
	// Explicitly empty: not os.Environ(), PATH, PYTHONPATH, proxies or credentials.
	cmd.Env = []string{}
	cmd.Dir = filepath.Dir(binding.EntryPath)
	cmd.Stdin = bytes.NewReader(input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	requests, childRequests, err := os.Pipe()
	if err != nil {
		return Result{}, ErrProcess
	}
	defer requests.Close()
	defer childRequests.Close()
	childResponses, responses, err := os.Pipe()
	if err != nil {
		return Result{}, ErrProcess
	}
	defer childResponses.Close()
	defer responses.Close()
	cmd.ExtraFiles = []*os.File{childRequests, childResponses}
	stdout := &cappedOutput{limit: 128, cancel: cancel}
	stderr := &cappedOutput{limit: 4096, cancel: cancel, discard: true}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err = cmd.Start(); err != nil {
		return Result{}, ErrProcess
	}
	childRequests.Close()
	childResponses.Close()
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	done := make(chan error, 1)
	go func() { done <- r.credentials(ctx, requests, responses, req.Operation, cfg.Export.Enabled, cancel) }()
	err = cmd.Wait()
	cancel()
	requests.Close()
	responses.Close()
	credentialErr := <-done
	if err != nil || credentialErr != nil || stdout.exceeded || stderr.exceeded {
		return Result{}, ErrProcess
	}
	var result struct {
		HTTPStatus int `json:"http_status"`
	}
	dec := json.NewDecoder(bytes.NewReader(stdout.buf.Bytes()))
	dec.DisallowUnknownFields()
	if dec.Decode(&result) != nil || dec.Decode(new(any)) != io.EOF {
		return Result{}, ErrProcess
	}
	// Reject duplicate keys as well as absent fields and alternate numeric forms.
	expected, _ := json.Marshal(result)
	if !bytes.Equal(bytes.TrimSpace(stdout.buf.Bytes()), expected) {
		return Result{}, ErrProcess
	}
	switch result.HTTPStatus {
	case 204, 400, 401, 403, 404, 409, 413, 429, 503:
		return Result{HTTPStatus: result.HTTPStatus}, nil
	}
	return Result{}, ErrProcess
}

type cappedOutput struct {
	buf               bytes.Buffer
	limit             int
	count             int
	exceeded, discard bool
	cancel            context.CancelFunc
	mu                sync.Mutex
}

func (w *cappedOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.count += len(p)
	if w.count > w.limit {
		w.exceeded = true
		w.cancel()
		return len(p), nil
	}
	if !w.discard {
		w.buf.Write(p)
	}
	return len(p), nil
}

func (r *ProcessRunner) credentials(ctx context.Context, input io.Reader, output io.Writer, operation string, export bool, cancel context.CancelFunc) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 16<<10)
	seen := map[string]bool{}
	for scanner.Scan() {
		var req struct {
			Purpose string `json:"purpose"`
		}
		dec := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		dec.DisallowUnknownFields()
		if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF {
			cancel()
			return ErrProcess
		}
		canonical, _ := json.Marshal(req)
		if !bytes.Equal(scanner.Bytes(), canonical) || seen[req.Purpose] || len(seen) >= 4 {
			cancel()
			return ErrProcess
		}
		allowed := operation == "accept_native" && req.Purpose == "native_webhook_verify" || operation == "tick" && (req.Purpose == "events_reader" || req.Purpose == "runtime_collector" || export && req.Purpose == "flywheel_submit")
		if !allowed || ctx.Err() != nil {
			cancel()
			return ErrProcess
		}
		seen[req.Purpose] = true
		value, err := r.provider.Credential(ctx, req.Purpose)
		response := []byte("{\"error\":\"unavailable\"}\n")
		if err == nil && len(value) > 0 && len(value) <= 8192 && ctx.Err() == nil {
			encoded, _ := json.Marshal(struct {
				Value string `json:"value_base64"`
			}{base64.StdEncoding.EncodeToString(value)})
			response = append(encoded, '\n')
		}
		// Provider retains ownership of its buffer; do not mutate shared secrets.
		if _, err = output.Write(response); err != nil {
			cancel()
			return ErrProcess
		}
	}
	// Pipe closure after child exit/cancellation is expected, not diagnostic text.
	if scanner.Err() != nil && ctx.Err() == nil {
		cancel()
		return ErrProcess
	}
	return nil
}

// The entire imported application closure must be reviewed and pinned. This
// checks bytes before launch; the provider must additionally keep the prepared
// runtime/code tree immutable for the duration of an invocation.
var moduleNames = [...]string{"__init__.py", "adapter.py", "projection.py", "owners.py", "managed.py", "ipc.py", "ipc_entry.py"}

func verifyModules(binding ApprovedBinding) bool {
	if len(binding.ModuleSHA256) != len(moduleNames) || filepath.Base(binding.EntryPath) != "ipc_entry.py" || binding.ModuleSHA256["ipc_entry.py"] != binding.EntrySHA256 {
		return false
	}
	directory := filepath.Dir(binding.EntryPath)
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		return false
	}
	for _, name := range moduleNames {
		expected, ok := binding.ModuleSHA256[name]
		if !ok {
			return false
		}
		path := filepath.Join(directory, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return false
		}
		// Refuse link substitution and nonregular files even between Lstat and open.
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		if err != nil {
			return false
		}
		file := os.NewFile(uintptr(fd), path)
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			file.Close()
			return false
		}
		raw, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		file.Close()
		if readErr != nil || len(raw) > 1<<20 || !digestMatches(raw, expected) {
			return false
		}
	}
	return true
}
