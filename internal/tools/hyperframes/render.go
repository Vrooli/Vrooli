package hyperframes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

type Input struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Executable struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Request struct {
	Version        int        `json:"version"`
	InputDir       string     `json:"input_dir"`
	Composition    string     `json:"composition"`
	Inputs         []Input    `json:"inputs"`
	Node           Executable `json:"node"`
	Browser        Executable `json:"browser"`
	FFmpeg         Executable `json:"ffmpeg"`
	FFprobe        Executable `json:"ffprobe"`
	TimeoutSeconds int        `json:"timeout_seconds"`
	MemoryMiB      int        `json:"memory_mib"`
}

type Receipt struct {
	Version        int               `json:"version"`
	Status         string            `json:"status"`
	Key            string            `json:"render_key"`
	OutputSHA256   string            `json:"output_sha256"`
	Installation   Installation      `json:"installation"`
	Profile        RenderProfile     `json:"profile"`
	Request        Request           `json:"request"`
	StartedAt      time.Time         `json:"started_at"`
	ElapsedSeconds float64           `json:"elapsed_seconds"`
	Reused         bool              `json:"reused"`
	ToolVersions   map[string]string `json:"tool_versions"`
	Limits         []string          `json:"limits"`
	Error          string            `json:"error,omitempty"`
	DiagnosticDir  string            `json:"diagnostic_dir,omitempty"`
}

type invocationStartKey struct{}

type runner func(context.Context, string, []string, string, []string, io.Writer, int) error

// Execute reads a bounded request and publishes a verified local output bundle.
// It never provisions tools or downloads a browser.
func Execute(ctx context.Context, requestFile, out string) (result Receipt, err error) {
	defer func() {
		if err != nil {
			result.Version = 1
			result.Status = "failed"
			result.Error = err.Error()
		}
	}()
	started := time.Now()
	ctx = context.WithValue(ctx, invocationStartKey{}, started)
	raw, err := readFileContext(ctx, requestFile, 1<<20)
	if err != nil {
		return Receipt{}, err
	}
	var req Request
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(&req); err != nil {
		return Receipt{}, err
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return Receipt{}, errors.New("request must contain one JSON value")
	}
	if err = validateRequest(req); err != nil {
		return Receipt{}, err
	}
	ctx, cancel := context.WithDeadline(ctx, started.Add(time.Duration(req.TimeoutSeconds)*time.Second))
	defer cancel()
	dir, installation, err := SelectedInstallationContext(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("managed tool required: %w", err)
	}
	return execute(ctx, req, out, dir, installation, runContained)
}

func validateRequest(req Request) error {
	if req.Version != 1 {
		return errors.New("request version must be 1")
	}
	if !filepath.IsAbs(req.InputDir) {
		return errors.New("input_dir must be absolute")
	}
	if !safeRelative(req.Composition, true) {
		return errors.New("composition must be a relative directory")
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 1800 {
		return errors.New("timeout_seconds must be 1..1800")
	}
	if req.MemoryMiB < 1024 || req.MemoryMiB > 16384 {
		return errors.New("memory_mib must be 1024..16384")
	}
	if len(req.Inputs) < 1 || len(req.Inputs) > 512 {
		return errors.New("declare 1..512 inputs")
	}
	seen := map[string]bool{}
	for _, in := range req.Inputs {
		if !safeRelative(in.Path, false) || seen[in.Path] || !validHash(in.SHA256) {
			return fmt.Errorf("invalid or duplicate input %q", in.Path)
		}
		seen[in.Path] = true
	}
	entry := filepath.ToSlash(filepath.Join(req.Composition, "index.html"))
	if !seen[entry] {
		return errors.New("composition/index.html must be declared")
	}
	for name, e := range map[string]Executable{"node": req.Node, "browser": req.Browser, "ffmpeg": req.FFmpeg, "ffprobe": req.FFprobe} {
		if !filepath.IsAbs(e.Path) || !validHash(e.SHA256) {
			return fmt.Errorf("%s must declare absolute executable path and SHA256", name)
		}
	}
	return nil
}

func validHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func safeRelative(s string, dot bool) bool {
	return s != "" && !filepath.IsAbs(s) && filepath.ToSlash(filepath.Clean(s)) == s && (dot || s != ".") && s != ".." && !strings.HasPrefix(s, "../") && !strings.ContainsAny(s, "\\\x00")
}

func verifyInputs(req Request) error { return verifyInputsContext(context.Background(), req) }

func verifyInputsContext(ctx context.Context, req Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(req.InputDir)
	if err != nil {
		return err
	}
	if root != filepath.Clean(req.InputDir) {
		return errors.New("input_dir must be canonical without symlinks")
	}
	declared := map[string]string{}
	for _, in := range req.Inputs {
		declared[in.Path] = in.SHA256
	}
	var total int64
	count := 0
	entries := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if err != nil {
			return err
		}
		entries++
		if entries > 2048 {
			return errors.New("input tree exceeds 2048 entries")
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular input %s", rel)
		}
		want, ok := declared[rel]
		if !ok {
			return fmt.Errorf("undeclared input %s", rel)
		}
		count++
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > 2<<30 {
			return errors.New("input set exceeds 2 GiB")
		}
		got, err := fileHashContext(ctx, path)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("input changed: %s", rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(declared) {
		return errors.New("declared input missing")
	}
	return nil
}

func verifyExecutables(req Request) error { return verifyExecutablesContext(context.Background(), req) }

func verifyExecutablesContext(ctx context.Context, req Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for name, e := range map[string]Executable{"node": req.Node, "browser": req.Browser, "ffmpeg": req.FFmpeg, "ffprobe": req.FFprobe} {
		got, err := fileHashContext(ctx, e.Path)
		if err != nil {
			return err
		}
		if got != e.SHA256 {
			return fmt.Errorf("%s executable changed", name)
		}
		info, err := os.Stat(e.Path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			return fmt.Errorf("%s is not executable", name)
		}
	}
	return nil
}

func renderKey(req Request, installation Installation) string {
	return renderKeyWithProfile(req, installation, fixedRenderProfile())
}

func renderKeyWithProfile(req Request, installation Installation, profile RenderProfile) string {
	// Sort a copy: input order is not part of composition meaning.
	req.Inputs = append([]Input(nil), req.Inputs...)
	sort.Slice(req.Inputs, func(i, j int) bool { return req.Inputs[i].Path < req.Inputs[j].Path })
	b, _ := json.Marshal(struct {
		Request      Request
		Installation Installation
		OS, Arch     string
		Profile      RenderProfile
	}{req, installation, runtime.GOOS, runtime.GOARCH, profile})
	return hashBytes(b)
}

func readReusable(ctx context.Context, out, key string) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	var r Receipt
	for _, path := range []string{out, filepath.Join(out, "receipt.json"), filepath.Join(out, "render.mp4")} {
		info, err := os.Lstat(path)
		if err != nil {
			return r, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return r, errors.New("reusable bundle must not use symlinks")
		}
	}
	b, err := readFileContext(ctx, filepath.Join(out, "receipt.json"), 1<<20)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	if r.Version != 1 || r.Status != "completed" || r.Key != key || r.Profile != fixedRenderProfile() {
		return r, errors.New("existing bundle does not match this render")
	}
	h, err := fileHashContext(ctx, filepath.Join(out, "render.mp4"))
	if err != nil {
		return r, err
	}
	if h != r.OutputSHA256 {
		return r, errors.New("existing render bytes changed")
	}
	r.Reused = true
	return r, nil
}

func execute(ctx context.Context, req Request, out, toolDir string, installation Installation, run runner) (receipt Receipt, err error) {
	started, ok := ctx.Value(invocationStartKey{}).(time.Time)
	if !ok {
		started = time.Now()
	}
	ctx, cancel := context.WithDeadline(ctx, started.Add(time.Duration(req.TimeoutSeconds)*time.Second))
	defer cancel()
	if err = ctx.Err(); err != nil {
		return receipt, err
	}

	if err = validateRequest(req); err != nil {
		return receipt, err
	}
	if err = verifyInputsContext(ctx, req); err != nil {
		return receipt, err
	}
	if err = verifyExecutablesContext(ctx, req); err != nil {
		return receipt, err
	}
	key := renderKey(req, installation)
	if _, e := os.Lstat(out); e == nil {
		return readReusable(ctx, out, key)
	} else if !os.IsNotExist(e) {
		return receipt, e
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return receipt, err
	}
	parent := filepath.Dir(out)
	// Refuse publication into source or tool trees, including via parent symlinks.
	if err = os.MkdirAll(parent, 0o700); err != nil {
		return receipt, err
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return receipt, err
	}
	out = filepath.Join(parent, filepath.Base(out))
	for _, root := range []string{req.InputDir, toolDir} {
		rel, e := filepath.Rel(root, out)
		if e == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return receipt, errors.New("output cannot be inside inputs or tool installation")
		}
	}
	candidate, err := os.MkdirTemp(parent, ".hyperframes-")
	if err != nil {
		return receipt, err
	}
	defer func() {
		if err == nil {
			_ = os.RemoveAll(candidate)
			return
		}
		// Preserve bounded diagnostics without retaining source snapshots or a
		// partial media file under a success-shaped destination.
		_ = os.RemoveAll(filepath.Join(candidate, "inputs"))
		_ = os.Remove(filepath.Join(candidate, "render.mp4"))
		_ = os.RemoveAll(filepath.Join(candidate, "tmp"))
		_ = os.RemoveAll(filepath.Join(candidate, "home"))
		receipt.Version = 1
		receipt.Status = "failed"
		receipt.Error = err.Error()
		receipt.Key = key
		receipt.DiagnosticDir = candidate
		if !receipt.StartedAt.IsZero() {
			receipt.ElapsedSeconds = time.Since(receipt.StartedAt).Seconds()
		}
		b, _ := json.MarshalIndent(receipt, "", "  ")
		_ = os.WriteFile(filepath.Join(candidate, "receipt.json"), append(b, '\n'), 0o600)
	}()
	inputDir := filepath.Join(candidate, "inputs")
	if err = os.Mkdir(inputDir, 0o700); err != nil {
		return receipt, err
	}
	for _, in := range req.Inputs {
		dest := filepath.Join(inputDir, filepath.FromSlash(in.Path))
		if err = os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return receipt, err
		}
		if err = copyInputContext(ctx, filepath.Join(req.InputDir, in.Path), dest, in.SHA256); err != nil {
			return receipt, err
		}
	}
	snapshot := req
	snapshot.InputDir = inputDir
	if err = verifyInputsContext(ctx, snapshot); err != nil {
		return receipt, err
	}
	for _, name := range []string{"tmp", "home"} {
		if err = os.Mkdir(filepath.Join(candidate, name), 0o700); err != nil {
			return receipt, err
		}
	}
	watchCtx, watchCancel := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	budgetFailure := make(chan error, 1)
	go func() {
		defer close(watchDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				if e := checkDiskBudgetContext(watchCtx, candidate); e != nil {
					if watchCtx.Err() != nil {
						return
					}
					budgetFailure <- e
					cancel()
					return
				}
			}
		}
	}()
	defer func() {
		watchCancel()
		<-watchDone
		if err != nil {
			select {
			case e := <-budgetFailure:
				err = fmt.Errorf("temporary-file budget: %w", e)
			default:
			}
		}
	}()
	profile := fixedRenderProfile()
	receipt = Receipt{Version: 1, Status: "running", Key: key, Installation: installation, Profile: profile, Request: req, StartedAt: started.UTC(), ToolVersions: map[string]string{}, Limits: []string{"Technical verification only; no aesthetic or listening acceptance", "Executable hashes do not pin all host shared libraries", "Linux containment only; no portable browser acquisition", "Temporary disk is sampled every second at 4 GiB; this is not a filesystem quota", "Receipt binds local declared bytes; browser network is not isolated. Reviewed local-only asset evidence is required", "512 tasks per owned scope and one capture worker; extraction concurrency and native FFmpeg automatic thread pools are not independently bounded by this owner", "Task-denial counters are sampled on owned scopes; missing or collected counters are unknown, never zero", "The owner wall deadline is enforced; HyperFrames 0.8.112 forwards extraction configuration, but this profile sets no separate FFmpeg process timeout"}}
	logFile, err := os.OpenFile(filepath.Join(candidate, "render.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return receipt, err
	}
	defer logFile.Close()
	log := &boundedWriter{Writer: logFile, remaining: profile.LogBytes}
	ctx = context.WithValue(ctx, taskObservationWriterKey{}, log)
	env := renderEnv(req)
	env = append(env, "HOME="+filepath.Join(candidate, "home"), "TMPDIR="+filepath.Join(candidate, "tmp"), "TMP="+filepath.Join(candidate, "tmp"), "TEMP="+filepath.Join(candidate, "tmp"))
	for name, spec := range map[string]Executable{"node": req.Node, "browser": req.Browser, "ffmpeg": req.FFmpeg, "ffprobe": req.FFprobe} {
		args := []string{"--version"}
		if name == "ffmpeg" || name == "ffprobe" {
			args = []string{"-version"}
		}
		var capture cappedBuffer
		if err = run(ctx, spec.Path, args, candidate, env, &capture, req.MemoryMiB); err != nil {
			return receipt, fmt.Errorf("%s version preflight: %w", name, err)
		}
		receipt.ToolVersions[name] = strings.TrimSpace(capture.String())
	}
	if !supportedNode(receipt.ToolVersions["node"]) {
		return receipt, errors.New("Node >=22.12.0 required")
	}
	entry := filepath.Join(toolDir, installation.Entry)
	comp := filepath.Join(inputDir, req.Composition)
	if err = run(ctx, req.Node.Path, []string{entry, "check", comp, "--no-browser-gpu"}, candidate, env, log, req.MemoryMiB); err != nil {
		return receipt, fmt.Errorf("composition check failed: %w", err)
	}
	video := filepath.Join(candidate, "render.mp4")
	args := profile.renderArgs(entry, comp, video)
	if err = run(ctx, req.Node.Path, args, candidate, env, log, req.MemoryMiB); err != nil {
		return receipt, fmt.Errorf("render failed: %w", err)
	}
	if err = run(ctx, req.FFmpeg.Path, []string{"-v", "error", "-threads", fmt.Sprint(profile.DecodeThreads), "-i", video, "-f", "null", "-"}, candidate, env, log, req.MemoryMiB); err != nil {
		return receipt, fmt.Errorf("full decode failed: %w", err)
	}
	var probe cappedBuffer
	if err = run(ctx, req.FFprobe.Path, []string{"-v", "error", "-show_streams", "-show_format", "-of", "json", video}, candidate, env, &probe, req.MemoryMiB); err != nil {
		return receipt, fmt.Errorf("output probe failed: %w", err)
	}
	if err = validateProbe(probe.Bytes()); err != nil {
		return receipt, err
	}
	if err = verifyInputsContext(ctx, snapshot); err != nil {
		return receipt, err
	}
	if err = verifyInputsContext(ctx, req); err != nil {
		return receipt, err
	}
	if err = verifyExecutablesContext(ctx, req); err != nil {
		return receipt, err
	}
	actual, e := TreeHashContext(ctx, toolDir)
	if e != nil {
		return receipt, e
	}
	if actual != installation.Tree {
		return receipt, errors.New("renderer installation changed during render")
	}
	if err = ctx.Err(); err != nil {
		return receipt, err
	}
	receipt.OutputSHA256, err = fileHashContext(ctx, video)
	if err != nil {
		return receipt, err
	}
	receipt.Status = "completed"
	receipt.ElapsedSeconds = time.Since(started).Seconds()
	b, _ := json.MarshalIndent(receipt, "", "  ")
	if err = os.WriteFile(filepath.Join(candidate, "receipt.json"), append(b, '\n'), 0o600); err != nil {
		return receipt, err
	}
	if err = os.WriteFile(filepath.Join(candidate, "probe.json"), probe.Bytes(), 0o600); err != nil {
		return receipt, err
	}
	if err = logFile.Close(); err != nil {
		return receipt, err
	}
	if err = os.RemoveAll(inputDir); err != nil {
		return receipt, err
	}
	watchCancel()
	<-watchDone
	if err = ctx.Err(); err != nil {
		return receipt, err
	}
	for _, name := range []string{"tmp", "home"} {
		if err = os.RemoveAll(filepath.Join(candidate, name)); err != nil {
			return receipt, err
		}
	}
	// Rename never replaces a completed nonempty bundle. A competing publication
	// is refused; the caller may inspect/reuse it on a subsequent invocation.
	if _, e := os.Lstat(out); !os.IsNotExist(e) {
		return receipt, errors.New("output destination appeared during render")
	}
	if err = os.Rename(candidate, out); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func copyInput(source, dest, want string) error {
	return copyInputContext(context.Background(), source, dest, want)
}

func copyInputContext(ctx context.Context, source, dest, want string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := openRegularContext(ctx, source, 2<<30)
	if err != nil {
		return err
	}
	defer f.Close()
	o, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(o, contextReader{ctx: ctx, reader: io.LimitReader(f, 2<<30+1)})
	closeErr := o.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	got, err := fileHashContext(ctx, dest)
	if err != nil {
		return err
	}
	if got != want {
		return errors.New("input changed while snapshotting")
	}
	return nil
}

func supportedNode(v string) bool {
	var major, minor int
	if _, err := fmt.Sscanf(strings.TrimSpace(v), "v%d.%d.", &major, &minor); err != nil {
		return false
	}
	return major > 22 || major == 22 && minor >= 12
}

func renderEnv(req Request) []string {
	profile := fixedRenderProfile()
	// Fixed environment avoids ambient Node options, HF browser overrides,
	// cloud credentials, proxy and dotenv-driven process changes.
	path := strings.Join([]string{filepath.Dir(req.Node.Path), filepath.Dir(req.FFmpeg.Path), filepath.Dir(req.FFprobe.Path), "/usr/bin", "/bin"}, string(os.PathListSeparator))
	env := []string{"PATH=" + path, "HYPERFRAMES_NO_TELEMETRY=1", "DO_NOT_TRACK=1", "HYPERFRAMES_NO_CAPTURE_DOTENV=1", "HYPERFRAMES_BROWSER_PATH=" + req.Browser.Path, "HYPERFRAMES_FFMPEG_PATH=" + req.FFmpeg.Path, "HYPERFRAMES_FFPROBE_PATH=" + req.FFprobe.Path, fmt.Sprintf("OMP_NUM_THREADS=%d", profile.OMPThreads), fmt.Sprintf("OPENBLAS_NUM_THREADS=%d", profile.BLASThreads), "LANG=C.UTF-8", "TZ=UTC"}
	for _, key := range []string{"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
		if v := os.Getenv(key); v != "" {
			env = append(env, key+"="+v)
		}
	}
	return env
}

func checkDiskBudget(dir string) error {
	return checkDiskBudgetContext(context.Background(), dir)
}

func checkDiskBudgetContext(ctx context.Context, dir string) error {
	return checkDiskBudgetWalk(ctx, dir, filepath.WalkDir)
}

func checkDiskBudgetWalk(ctx context.Context, dir string, walk func(string, fs.WalkDirFunc) error) error {
	var total int64
	entries := 0
	root := filepath.Clean(dir)
	vanishedChild := func(path string, err error) bool {
		// Chromium owns a changing temporary tree. A child can disappear
		// between ReadDir and Info (or before a directory is read). The
		// monitored root must remain available, and all other errors fail.
		return filepath.Clean(path) != root && errors.Is(err, fs.ErrNotExist)
	}
	return walk(dir, func(path string, d fs.DirEntry, err error) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if err != nil {
			if vanishedChild(path, err) {
				return nil
			}
			return err
		}
		entries++
		if entries > 100000 {
			return errors.New("temporary tree exceeds entry budget")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if vanishedChild(path, err) {
				return nil
			}
			return err
		}
		if info.Mode().IsRegular() {
			size := info.Size()
			if size < 0 || size > fixedRenderProfile().TemporaryDiskBytes-total {
				return errors.New("temporary files exceed sampled 4 GiB budget")
			}
			total += size
		}
		return nil
	})
}

type boundedWriter struct {
	io.Writer
	remaining int64
}

func (w *boundedWriter) Write(b []byte) (int, error) {
	if int64(len(b)) > w.remaining {
		return 0, errors.New("render log exceeds 8 MiB")
	}
	n, err := w.Writer.Write(b)
	w.remaining -= int64(n)
	return n, err
}

type cappedBuffer struct{ b []byte }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if len(b.b)+len(p) > 1<<20 {
		return 0, errors.New("probe output exceeds 1 MiB")
	}
	b.b = append(b.b, p...)
	return len(p), nil
}
func (b *cappedBuffer) Bytes() []byte  { return b.b }
func (b *cappedBuffer) String() string { return string(b.b) }

func validateProbe(b []byte) error {
	var p struct {
		Streams []struct {
			CodecType     string `json:"codec_type"`
			Width, Height int
			Duration      string `json:"duration"`
		}
		Format struct {
			Duration string `json:"duration"`
		}
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	var duration float64
	fmt.Sscan(p.Format.Duration, &duration)
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 || duration > 600 {
		return errors.New("render duration must be 0..600 seconds")
	}
	for _, s := range p.Streams {
		if s.CodecType == "video" && s.Width > 0 && s.Height > 0 && s.Width <= 4096 && s.Height <= 4096 {
			return nil
		}
	}
	return errors.New("output has no supported video stream")
}

func runContained(ctx context.Context, path string, args []string, dir string, env []string, output io.Writer, memoryMiB int) error {
	if runtime.GOOS != "linux" {
		return errors.New("HyperFrames containment is implemented for Linux only; runtime qualification is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	scope := fmt.Sprintf("vrooli-test-hyperframes-%d-%d", os.Getpid(), time.Now().UnixNano())
	profile := fixedRenderProfile()
	c, err := platform.ContainedCommand(platform.ContainedSpec{Path: path, Args: args, Dir: dir, Env: env, Stdout: output, Stderr: output, Scope: scope, Containment: platform.Containment{Slice: profile.Slice, MemoryMax: fmt.Sprintf("%dM", memoryMiB), TasksMax: profile.TasksMax, CPUWeight: profile.CPUWeight}})
	if err != nil {
		return err
	}
	defer c.Release()
	if err = c.StartContext(ctx); err != nil {
		return err
	}

	return waitWithTaskPressure(ctx, func(watchCtx context.Context) error {
		return waitOwned(watchCtx, c.Cmd.Wait, func(cleanup context.Context) error { return platform.StopScopeContext(cleanup, c.Scope) }, func() ([]int, error) {
			cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return platform.ScopeProcessesContext(cleanup, c.Scope)
		})
	}, func() (platform.Occupancy, error) { return platform.ScopeOccupancy(c.Scope) }, time.Duration(profile.TaskPressureSampleMS)*time.Millisecond, func(summary taskPressureSummary) error {
		return writeTaskObservation(ctx, c.Scope, path, summary)
	})
}

// waitOwned joins one owned root and verifies that its scope has no descendants.
// The injected operations let regressions exercise cancellation without a live
// manager or launching browser processes.
func waitOwned(ctx context.Context, wait func() error, stop func(context.Context) error, processes func() ([]int, error)) error {
	done := make(chan error, 1)
	go func() { done <- wait() }()
	stopBounded := func() error {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return stop(cleanup)
	}
	var result error
	select {
	case result = <-done:
	case <-ctx.Done():
		if err := stopBounded(); err != nil {
			return fmt.Errorf("owned scope cancellation failed: %w (context: %v)", err, ctx.Err())
		}
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			return errors.New("owned root did not exit after scope cancellation")
		}
		result = ctx.Err()
	}
	pids, err := processes()
	if err != nil {
		return fmt.Errorf("owned scope cleanup could not be verified: %w", err)
	}
	if len(pids) > 0 {
		if err := stopBounded(); err != nil {
			return err
		}
		pids, err = processes()
		if err != nil {
			return err
		}
		if len(pids) > 0 {
			return fmt.Errorf("owned scope still contains %d processes", len(pids))
		}
	}
	return result
}
