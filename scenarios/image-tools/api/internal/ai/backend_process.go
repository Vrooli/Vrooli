package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"image-tools/internal/backends"
)

// commandRunner executes a resolved command. Injected so tests assert argument
// assembly + availability without invoking absent backend binaries.
type commandRunner func(ctx context.Context, name string, args []string) error

// outputRunner executes a command and returns stdout/stderr text for providers
// whose native result is written to stdout instead of an output file.
type outputRunner func(ctx context.Context, name string, args []string) ([]byte, error)

// lookPathFunc resolves a binary on PATH. Injected for the same reason.
type lookPathFunc func(file string) (string, error)

// pythonModuleChecker checks whether a Python executable can import modules.
type pythonModuleChecker func(ctx context.Context, python string, modules []string) error

// onnxProviderChecker reports the ONNX Runtime execution providers importable
// from the configured Python executable.
type onnxProviderChecker func(ctx context.Context, python string) ([]string, error)

// pythonProgram is the sentinel program string that marks a backend as Python-
// served. Such backends run ONLY through the scenario's private uv venv
// interpreter (execProvider.pythonInterpreter); they never resolve a bare
// "python3" off the host PATH.
const pythonProgram = "python3"

// backendProcess owns software readiness and execution. It receives already-built
// arguments and never selects a model, operation or technique.
type backendProcess struct {
	name              string
	program           string
	pythonInterpreter string
	programAliases    []string
	gpuCapable        bool
	provision         string
	imports           []string
	lookPath          lookPathFunc
	checkPy           pythonModuleChecker
	checkONNX         onnxProviderChecker
	run               commandRunner
	warm              warmRunner
	gpuProbe          func(context.Context, string) bool
	gpuProbeOnce      sync.Once
	gpuProbeValue     bool
	progressScan      func(string) (float64, string, bool)
	stream            streamRunner
	probe             backends.AvailabilityCache
}

// childPeakRSSKey carries a per-execution collector through the default command
// runners. The collector is unique to one backend Execute call, so concurrent
// jobs cannot share or overwrite a process measurement.
type childPeakRSSKey struct{}

type childPeakRSSCollector struct {
	mu       sync.Mutex
	bytes    int64
	measured bool
}

func (c *childPeakRSSCollector) record(cmd *exec.Cmd) {
	bytes, ok := processPeakRSSBytes(cmd)
	if !ok {
		return
	}
	c.mu.Lock()
	c.bytes = bytes
	c.measured = true
	c.mu.Unlock()
}

func (c *childPeakRSSCollector) result() (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bytes, c.measured
}

// processPeakRSSBytes reads the peak resident set size reported by wait4 for
// this exact completed child. Linux and BSD report Maxrss in KiB; Darwin reports
// bytes. Runners without a supported per-child wait result remain unavailable.
func processPeakRSSBytes(cmd *exec.Cmd) (int64, bool) {
	if cmd == nil || cmd.ProcessState == nil {
		return 0, false
	}
	usage := reflect.ValueOf(cmd.ProcessState.SysUsage())
	if !usage.IsValid() {
		return 0, false
	}
	if usage.Kind() == reflect.Pointer {
		if usage.IsNil() {
			return 0, false
		}
		usage = usage.Elem()
	}
	if usage.Kind() != reflect.Struct {
		return 0, false
	}
	field := usage.FieldByName("Maxrss")
	if !field.IsValid() {
		return 0, false
	}
	var maxRSS int64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		maxRSS = field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := field.Uint()
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		maxRSS = int64(v)
	default:
		return 0, false
	}
	if maxRSS <= 0 {
		return 0, false
	}
	switch runtime.GOOS {
	case "darwin": // ru_maxrss is already bytes on macOS.
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		if maxRSS > int64(^uint64(0)>>1)/1024 {
			return 0, false
		}
		maxRSS *= 1024
	default:
		return 0, false
	}
	return maxRSS, true
}

func (p *backendProcess) Close() error {
	if p.warm != nil {
		return p.warm.Close()
	}
	return nil
}

// programName returns the program this provider should invoke: the first of its
// programAliases that resolves on PATH (a GPU build like "sd-gpu"), else the
// base program ("sd"). Resolution is cheap (a PATH lookup) and done per call so
// installing a GPU build at runtime is picked up without a restart.
func (p *backendProcess) programName() string {
	// Python backends invoke the scenario's private venv interpreter by absolute
	// path — and ONLY that. There is no bare-"python3" PATH fallback: that would
	// reintroduce the host-Python contamination this isolation seam prevents. An
	// empty interpreter means the venv is not provisioned yet, and Availability/
	// Execute report the backend unavailable (with remediation) rather than
	// silently running unisolated.
	if p.program == pythonProgram {
		return p.pythonInterpreter
	}
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	for _, alias := range p.programAliases {
		if _, err := lookPath(alias); err == nil {
			return alias
		}
	}
	return p.program
}

func (p *backendProcess) GPUCapable() bool {
	if !p.gpuCapable {
		return false
	}
	if p.gpuProbe == nil {
		return true
	}
	program := p.programName()
	if program == "" {
		return false
	}
	if p.lookPath != nil {
		resolved, err := p.lookPath(program)
		if err != nil {
			return false
		}
		program = resolved
	}
	if p.program == pythonProgram {
		// The private environment is provisioned asynchronously. A probe before
		// Torch is ready must not pin this provider to CPU for its whole lifetime.
		return p.gpuProbe(context.Background(), program)
	}
	p.gpuProbeOnce.Do(func() {
		p.gpuProbeValue = p.gpuProbe(context.Background(), program)
	})
	return p.gpuProbeValue
}

// Available reports whether the backend program is resolvable on PATH.
func (p *backendProcess) Available(ctx context.Context) bool {
	return p.Availability(ctx).Available
}

// Availability reports software readiness for doctor/selection. It is limited
// to host software presence; model weights and hardware fit are checked by the
// model lifecycle and capabilities layers.
func (p *backendProcess) Availability(ctx context.Context) backends.Availability {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	program := p.programName()
	a := backends.Availability{Provision: p.provision}
	if p.program == pythonProgram && program == "" {
		a.Detail = "image-tools Python venv not provisioned yet; provision the isolated runtime"
		return a
	}
	resolved, err := lookPath(program)
	if err != nil {
		a.Detail = fmt.Sprintf("%s not found on PATH", program)
		return a
	}
	a.Available = true
	a.Detail = fmt.Sprintf("%s resolved at %s", program, resolved)
	if len(p.imports) == 0 {
		return a
	}
	return p.probe.Do(ctx, resolved+"\x00"+strings.Join(p.imports, ","), func() backends.Availability {
		check := p.checkPy
		if check == nil {
			check = defaultCheckPythonModules
		}
		if err := check(ctx, resolved, p.imports); err != nil {
			a.Available = false
			a.Detail = fmt.Sprintf("%s; Python imports failed: %v", a.Detail, err)
			return a
		}
		a.Detail += "; Python imports ready: " + strings.Join(p.imports, ",")
		if p.name == "onnxruntime" {
			providers, err := p.onnxRuntimeProviders(ctx, resolved)
			if err != nil {
				a.Available = false
				a.Detail = fmt.Sprintf("%s; ONNX Runtime provider probe failed: %v", a.Detail, err)
			} else {
				a.Available = slices.Contains(providers, "CPUExecutionProvider")
				a.Detail = p.onnxRuntimeProviderDetail(resolved, providers)
			}
		}
		return a
	})
}

func (p *backendProcess) onnxRuntimeProviders(ctx context.Context, python string) ([]string, error) {
	check := p.checkONNX
	if check == nil {
		check = defaultCheckONNXRuntimeProviders
	}
	return check(ctx, python)
}

func (p *backendProcess) onnxRuntimeProviderDetail(python string, providers []string) string {
	detail := fmt.Sprintf("%s resolved at %s; Python imports ready: %s; ONNX Runtime providers: %s", p.program, python, strings.Join(p.imports, ","), strings.Join(providers, ","))
	if !slices.Contains(providers, "CPUExecutionProvider") {
		return detail + "; CPUExecutionProvider missing"
	}
	if !slices.Contains(providers, "CUDAExecutionProvider") {
		return detail + "; CUDAExecutionProvider unavailable, sidecar remains CPU-only"
	}
	return detail + "; CUDAExecutionProvider available, but this registered sidecar remains CPU-labeled until a GPU-capable ONNX provider row is promoted"
}

func defaultCheckPythonModules(ctx context.Context, python string, modules []string) error {
	script := "import " + strings.Join(modules, ", ")
	cmd := exec.CommandContext(ctx, python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func defaultCheckONNXRuntimeProviders(ctx context.Context, python string) ([]string, error) {
	script := "import json, onnxruntime as ort; print(json.dumps(ort.get_available_providers()))"
	cmd := exec.CommandContext(ctx, python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	var providers []string
	if err := json.Unmarshal(out, &providers); err != nil {
		return nil, fmt.Errorf("decode onnxruntime providers: %w", err)
	}
	return providers, nil
}

func (p *backendProcess) commandName() (string, error) {
	program := p.programName()
	if p.program == pythonProgram && program == "" {
		return "", fmt.Errorf("ai: backend %q unavailable: image-tools Python venv not provisioned — %s", p.name, p.provision)
	}
	return program, nil
}

func (p *backendProcess) execute(ctx context.Context, program string, args []string, progress func(float64, string)) (string, int64, bool, error) {
	if p.warm != nil {
		runner := "warm"
		if admitted, _ := ctx.Value(operationMemoryContextKey{}).(bool); admitted {
			runner = "operation-scoped-worker"
		}
		return runner, 0, false, p.warm.Run(ctx, program, args)
	}
	var peakRSS *childPeakRSSCollector
	usesDefaultStream := progress != nil && p.progressScan != nil
	// Only the default process runners expose a completed child process state.
	// Injected runners are useful for tests and adapters but cannot make an RSS
	// claim; persistent warm workers also have no isolatable per-job peak.
	if (usesDefaultStream && p.stream == nil) || (!usesDefaultStream && p.run == nil) {
		peakRSS = &childPeakRSSCollector{}
		ctx = context.WithValue(ctx, childPeakRSSKey{}, peakRSS)
	}
	if progress != nil && p.progressScan != nil {
		stream := p.stream
		if stream == nil {
			stream = defaultStreamRun
		}
		err := stream(ctx, program, args, func(line string) {
			if frac, message, ok := p.progressScan(line); ok {
				progress(frac, message)
			}
		})
		bytes, measured := int64(0), false
		if peakRSS != nil {
			bytes, measured = peakRSS.result()
		}
		return "stream", bytes, measured, err
	}
	run := p.run
	if run == nil {
		run = defaultRun
	}
	err := run(ctx, program, args)
	bytes, measured := int64(0), false
	if peakRSS != nil {
		bytes, measured = peakRSS.result()
	}
	return "", bytes, measured, err
}

func defaultRun(ctx context.Context, name string, args []string) error {
	_, err := defaultRunOutput(ctx, name, args)
	return err
}

func defaultRunOutput(ctx context.Context, name string, args []string) ([]byte, error) {
	cmd := inferenceCommand(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if collector, ok := ctx.Value(childPeakRSSKey{}).(*childPeakRSSCollector); ok {
		collector.record(cmd)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w (%s)", name, args, err, string(out))
	}
	return out, nil
}

// streamRunner runs a command, invoking onLine for each output fragment as it is
// produced (split on \r and \n, since progress bars redraw with \r). It still
// returns a non-nil error carrying captured output on a non-zero exit, matching
// defaultRun's error shape.
type streamRunner func(ctx context.Context, name string, args []string, onLine func(line string)) error

// defaultStreamRun is the real streamRunner. It merges stdout+stderr, scans
// fragments live (so per-step progress surfaces immediately rather than buffering
// until a newline), and accumulates everything for the failure message.
func defaultStreamRun(ctx context.Context, name string, args []string, onLine func(line string)) error {
	cmd := inferenceCommand(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%s: stdout pipe: %w", name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("%s: stderr pipe: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: start: %w", name, err)
	}

	var (
		mu  sync.Mutex
		buf strings.Builder
		wg  sync.WaitGroup
	)
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		sc.Split(scanLinesOrCR)
		for sc.Scan() {
			frag := sc.Text()
			mu.Lock()
			buf.WriteString(frag)
			buf.WriteByte('\n')
			mu.Unlock()
			if onLine != nil {
				onLine(frag)
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()

	waitErr := cmd.Wait()
	if collector, ok := ctx.Value(childPeakRSSKey{}).(*childPeakRSSCollector); ok {
		collector.record(cmd)
	}
	if waitErr != nil {
		return fmt.Errorf("%s %v: %w (%s)", name, args, waitErr, buf.String())
	}
	return nil
}

// scanLinesOrCR is a bufio.SplitFunc that breaks on either \n or \r, so a
// carriage-return-redrawn progress bar (e.g. "|====| 3/20\r") yields a fragment
// per redraw instead of buffering the whole line until the next \n.
func scanLinesOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// sdSamplerLine matches the stable-diffusion.cpp SAMPLING progress bar, which is
// the long phase users wait on. It is distinguished from the model-load /
// vae-decode byte bars (e.g. "|####| 196/196 - 2.28GB/s") by the per-iteration
// "s/it" or "it/s" rate suffix; only the sampler reports iterations.
var sdSamplerLine = regexp.MustCompile(`\|[#=> ]*\|\s*(\d+)/(\d+)\s*-\s*[0-9.]+\s*(?:s/it|it/s)`)

// scanStableDiffusionProgress parses one sd-cli output fragment into sampling
// progress. Returns ok=false for any non-sampler line so the caller streams a
// Progress update only on real forward motion.
func scanStableDiffusionProgress(line string) (frac float64, message string, ok bool) {
	m := sdSamplerLine.FindStringSubmatch(line)
	if m == nil {
		return 0, "", false
	}
	cur, err1 := strconv.Atoi(m[1])
	total, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || total <= 0 {
		return 0, "", false
	}
	if cur > total {
		cur = total
	}
	return float64(cur) / float64(total), fmt.Sprintf("sampling step %d/%d", cur, total), true
}

// probeStableDiffusionGPU reports whether the installed sd-cli has a GPU compute
// backend compiled in. stable-diffusion.cpp validates the --backend device
// assignment BEFORE loading the model, so pointing it at a non-existent model is
// a ~millisecond probe: a CPU-only build prints "backend 'cuda0' was not found"
// and exits; a GPU build gets past backend config to the model-load failure
// (which never mentions the device). A short timeout bounds the worst case.
func probeStableDiffusionGPU(ctx context.Context, program string) bool {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	missing := filepath.Join(os.TempDir(), "imgtools-gpu-probe-nonexistent.safetensors")
	out := filepath.Join(os.TempDir(), "imgtools-gpu-probe.png")
	for _, dev := range []string{"cuda0", "vulkan0"} {
		cmd := exec.CommandContext(ctx, program, "--backend", dev, "-M", "img_gen",
			"-m", missing, "-o", out, "-p", "probe", "--steps", "1")
		combined, _ := cmd.CombinedOutput()
		if !strings.Contains(string(combined), fmt.Sprintf("backend '%s' was not found", dev)) {
			return true // device backend is compiled in
		}
	}
	return false
}
