package parse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Runner struct {
	ctx      context.Context
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	sequence uint64
}

func NewRunner(ctx context.Context, modulePath string) (*Runner, error) {
	wasm, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, fmt.Errorf("read parser module: %w", err)
	}
	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("instantiate WASI: %w", err)
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("compile parser module: %w", err)
	}
	return &Runner{ctx: ctx, runtime: runtime, compiled: compiled}, nil
}

func (r *Runner) Close() error { return r.runtime.Close(r.ctx) }

type Request struct {
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Run mounts only the input's parent directory and addresses the file through
// the WASI /input mount. This keeps host paths out of the module filesystem.
func (r *Runner) Run(inputPath string, capabilities []string) (json.RawMessage, error) {
	inputPath, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve input path: %w", err)
	}
	info, err := os.Stat(inputPath)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input path is not a regular file: %s", inputPath)
	}
	requestPath := "/input/" + filepath.Base(inputPath)
	request, err := json.Marshal(Request{Path: requestPath, Capabilities: capabilities})
	if err != nil {
		return nil, err
	}
	request = append(request, '\n')
	var stdout, stderr bytes.Buffer
	config := wazero.NewModuleConfig().
		WithName(fmt.Sprintf("doc-parse-%d", atomic.AddUint64(&r.sequence, 1))).
		WithStdin(bytes.NewReader(request)).
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithFSConfig(wazero.NewFSConfig().WithDirMount(filepath.Dir(inputPath), "/input"))
	module, err := r.runtime.InstantiateModule(r.ctx, r.compiled, config)
	if err != nil {
		return nil, fmt.Errorf("run parser module: %w", err)
	}
	_ = module.Close(r.ctx)
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("parser module returned no response: %s", strings.TrimSpace(stderr.String()))
	}
	lineBytes := []byte(strings.TrimSpace(stdout.String()))
	modulePathJSON, _ := json.Marshal(requestPath)
	hostPathJSON, _ := json.Marshal(inputPath)
	lineBytes = bytes.ReplaceAll(lineBytes, modulePathJSON, hostPathJSON)
	line := strings.TrimSpace(string(lineBytes))
	var response json.RawMessage
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("decode parser response: %w", err)
	}
	return response, nil
}
