package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type result struct {
	option string
	file   string
	format string
	wallMS int64
	rssKB  int64
	bytes  int
	sha    string
	out    string
}

func main() {
	root := flag.String("resource-root", ".", "doc-parse resource root")
	native := flag.String("native", "", "native shim executable")
	wasm := flag.String("wasm", "", "WASI shim module")
	corpus := flag.String("corpus", "testdata/corpus", "fixture directory relative to resource root")
	flag.Parse()
	if *native == "" || *wasm == "" {
		fatalf("--native and --wasm are required")
	}

	var files []string
	err := filepath.WalkDir(filepath.Join(*root, *corpus), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fatalf("walk corpus: %v", err)
	}
	sort.Strings(files)

	fmt.Println("option,file,format,wall_ms,rss_kb,output_bytes,sha256")
	nativeResults := make(map[string]result, len(files))
	wasmResults := make(map[string]result, len(files))
	wasmRunner, err := newWasmRunner(*wasm)
	if err != nil {
		fatalf("compile wasm module: %v", err)
	}
	defer wasmRunner.close()
	for _, file := range files {
		rel, err := filepath.Rel(*root, file)
		if err != nil {
			fatalf("relative fixture path: %v", err)
		}
		requestPath := "/work/" + filepath.ToSlash(rel)
		got, err := runNative(*native, file, requestPath)
		if err != nil {
			fatalf("native %s: %v", rel, err)
		}
		nativeResults[rel] = got
		fmt.Printf("native,%s,%s,%d,%d,%d,%s\n", rel, got.format, got.wallMS, got.rssKB, got.bytes, got.sha)
		got, err = wasmRunner.run(*root, requestPath)
		if err != nil {
			fatalf("wasm %s: %v", rel, err)
		}
		wasmResults[rel] = got
		fmt.Printf("wasm,%s,%s,%d,%d,%d,%s\n", rel, got.format, got.wallMS, got.rssKB, got.bytes, got.sha)
		if nativeResults[rel].out != wasmResults[rel].out {
			fatalf("output mismatch for %s", rel)
		}
	}
	fmt.Printf("equality,all,%d fixtures matched byte-for-byte\n", len(files))
}

func runNative(binary, hostPath, requestPath string) (result, error) {
	request := []byte(fmt.Sprintf("{\"path\":%q}\n", hostPath))
	start := time.Now()
	cmd := exec.Command(binary)
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.Output()
	if err != nil {
		return result{}, err
	}
	out = bytes.ReplaceAll(out, []byte(hostPath), []byte(requestPath))
	rssKB := int64(0)
	if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		rssKB = usage.Maxrss
	}
	return makeResult("native", requestPath, out, time.Since(start), rssKB), nil
}

type wasmRunner struct {
	ctx      context.Context
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	sequence uint64
}

func newWasmRunner(modulePath string) (*wasmRunner, error) {
	wasm, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, err
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, err
	}
	return &wasmRunner{ctx: ctx, runtime: runtime, compiled: compiled}, nil
}

func (w *wasmRunner) close() {
	_ = w.runtime.Close(w.ctx)
}

func (w *wasmRunner) run(hostRoot, requestPath string) (result, error) {
	request := []byte(fmt.Sprintf("{\"path\":%q}\n", requestPath))
	start := time.Now()
	var out bytes.Buffer
	config := wazero.NewModuleConfig().
		WithName(fmt.Sprintf("portable-compare-%d", atomic.AddUint64(&w.sequence, 1))).
		WithStdin(bytes.NewReader(request)).
		WithStdout(&out).
		WithFSConfig(wazero.NewFSConfig().WithDirMount(hostRoot, "/work"))
	if _, err := w.runtime.InstantiateModule(w.ctx, w.compiled, config); err != nil {
		return result{}, err
	}
	return makeResult("wasm", requestPath, out.Bytes(), time.Since(start), readRSSKB()), nil
}

func readRSSKB() int64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			var value int64
			_, _ = fmt.Sscanf(line, "VmRSS: %d kB", &value)
			return value
		}
	}
	return 0
}

func makeResult(option, file string, output []byte, elapsed time.Duration, rssKB int64) result {
	text := strings.TrimSpace(string(output))
	// The shim deliberately emits stable normalized output; wall-clock is
	// measured by this harness and never becomes part of the equality payload.
	format := "unknown"
	if strings.Contains(text, `"format":"pdf"`) {
		format = "pdf"
	} else if strings.Contains(text, `"format":"document"`) {
		format = "document"
	}
	digest := sha256.Sum256([]byte(text))
	return result{option: option, file: file, format: format, wallMS: elapsed.Milliseconds(), rssKB: rssKB, bytes: len([]byte(text)), sha: hex.EncodeToString(digest[:]), out: text}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
