// Package protogen owns protobuf generation as a cross-platform Go pipeline.
// It never mutates the committed generated tree while tools are running:
// output is built beside it and published only after validation.
package protogen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

type ToolRunner func(ctx context.Context, dir, name string, args ...string) error

type Config struct {
	RepoRoot    string
	ProtoRoot   string
	LockPath    string
	StageParent string
	Scenarios   []string
	Logger      io.Writer
	RunTool     ToolRunner
	LockPoll    time.Duration
}

type Generator struct {
	cfg Config
}

var processLock sync.Mutex

// bufGenerationTimeout is intentionally longer than Buf's two-minute default.
// A full repository generation invokes several local plugins for every schema
// directory, and a fresh Intel macOS host can legitimately need more than two
// minutes before the slowest built-in Python plugin completes. Keep the bound
// finite so a genuinely wedged plugin still fails with actionable evidence.
const bufGenerationTimeout = 15 * time.Minute

func bufCommandArgs(command string, args ...string) []string {
	out := make([]string, 0, 3+len(args))
	out = append(out, command, "--timeout", bufGenerationTimeout.String())
	return append(out, args...)
}

func New(config Config) (*Generator, error) {
	config.RepoRoot = filepath.Clean(strings.TrimSpace(config.RepoRoot))
	config.ProtoRoot = filepath.Clean(strings.TrimSpace(config.ProtoRoot))
	if config.ProtoRoot == "." || config.ProtoRoot == "" {
		return nil, fmt.Errorf("protogen: proto root is required")
	}
	if config.RepoRoot == "." || config.RepoRoot == "" {
		config.RepoRoot = filepath.Clean(filepath.Join(config.ProtoRoot, "..", ".."))
	}
	if config.Logger == nil {
		config.Logger = io.Discard
	}
	if config.RunTool == nil {
		config.RunTool = runTool
	}
	if config.LockPoll <= 0 {
		config.LockPoll = 25 * time.Millisecond
	}
	if config.StageParent == "" {
		config.StageParent = DefaultStageParent(config.ProtoRoot)
	}
	return &Generator{cfg: config}, nil
}

func DefaultConfig(repoRoot string) Config {
	protoRoot := filepath.Join(repoRoot, "packages", "proto")
	return Config{RepoRoot: repoRoot, ProtoRoot: protoRoot}
}

// DefaultStageParent resolves where staging trees are created when a Config
// leaves StageParent empty. It is exported so callers that reap staging trees
// without building a Generator -- `protogen clean` -- target exactly the
// directory the generator writes to, from a single definition.
func DefaultStageParent(protoRoot string) string {
	return filepath.Dir(filepath.Clean(protoRoot))
}

// stagePrefixes names every staging directory protogen creates. Reaping keys
// on the whole set rather than one literal so a future staging kind cannot
// silently escape the retention bound.
var stagePrefixes = []string{".proto-gen-stage-", ".proto-gen-verify-", ".proto-descriptor-"}

// stageRetention bounds how long a retained staging tree survives. Retention
// exists so a failed run can be inspected, but a full staging tree is a copy of
// the entire generated fleet: without a bound, every failure permanently adds
// tens of megabytes that nothing removes. One day is long enough to debug the
// run that produced the tree and short enough that trees cannot accumulate.
const stageRetention = 24 * time.Hour

func hasStagePrefix(name string) bool {
	for _, prefix := range stagePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// stageRetainable reports whether a finished run's staging tree is worth
// keeping. Only a genuine failure is: a cancelled or timed-out run leaves a
// partial tree that says nothing the logs do not, and cancellation is the
// common case when generation runs under an agent or a test harness.
func stageRetainable(err error) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// reapStaleStages removes staging trees left behind by earlier runs. Callers
// hold the generator lock, so no live run owns a tree here. A tree that cannot
// be removed is logged and skipped: stale staging output is garbage, never a
// reason to fail the run that encountered it.
func (g *Generator) reapStaleStages(now time.Time) {
	entries, err := os.ReadDir(g.cfg.StageParent)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !hasStagePrefix(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || now.Sub(info.ModTime()) < stageRetention {
			continue
		}
		path := filepath.Join(g.cfg.StageParent, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			logf(g.cfg.Logger, "protogen: could not reap stale staging tree %s: %v", path, err)
			continue
		}
		logf(g.cfg.Logger, "protogen: reaped stale staging tree %s", path)
	}
}

// CleanStages removes every staging tree under stageParent regardless of age.
// Clean is an explicit operator request, so unlike reapStaleStages it does not
// honour stageRetention -- an operator asking to clean means all of it.
func CleanStages(stageParent string) error {
	entries, err := os.ReadDir(stageParent)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !hasStagePrefix(entry.Name()) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(stageParent, entry.Name())); err != nil {
			return fmt.Errorf("remove staging tree %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// Clean removes generated outputs while preserving the generated directory
// itself. It is intentionally exposed through the Go command so cleanup has
// identical semantics on every host.
func Clean(genRoot string) error {
	entries, err := os.ReadDir(genRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(genRoot, entry.Name())); err != nil {
			return fmt.Errorf("remove generated output %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// Generate publishes the generated tree. The named error return is what drives
// staging-tree retention in the deferred cleanup below, so every failure path
// must return through it rather than exiting by another route.
func (g *Generator) Generate(ctx context.Context) (err error) {
	unlock, err := g.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	g.reapStaleStages(time.Now())

	allScenarios, err := genmanifest.ScenarioNames(g.cfg.ProtoRoot)
	if err != nil {
		return fmt.Errorf("list schema scenarios: %w", err)
	}
	scope, err := g.resolveScope(allScenarios)
	if err != nil {
		return err
	}
	if len(g.cfg.Scenarios) == 0 {
		scope = allScenarios
	}
	logf(g.cfg.Logger, "protogen: publishing scenarios: %s", strings.Join(scope, ", "))

	stage, err := os.MkdirTemp(g.cfg.StageParent, ".proto-gen-stage-")
	if err != nil {
		return fmt.Errorf("create generation staging directory: %w", err)
	}
	defer func() {
		if stageRetainable(err) {
			logf(g.cfg.Logger, "protogen: generation failed; staging tree retained at %s", stage)
			return
		}
		_ = os.RemoveAll(stage)
	}()
	stageGen := filepath.Join(stage, "gen")
	if err := os.MkdirAll(stageGen, 0o755); err != nil {
		return fmt.Errorf("create staged generated tree: %w", err)
	}
	if err := g.seedGeneratedMetadata(stageGen); err != nil {
		return err
	}

	args := bufCommandArgs("generate", "--output", stage)
	if len(g.cfg.Scenarios) > 0 {
		for _, scenario := range scope {
			args = append(args, "--path", filepath.ToSlash(filepath.Join("schemas", scenario)))
		}
	}
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", args...); err != nil {
		return fmt.Errorf("buf generate: %w", err)
	}
	if err := markPythonPackages(stageGen); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stageGen, "descriptor"), 0o755); err != nil {
		return fmt.Errorf("create staged descriptor directory: %w", err)
	}
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", bufCommandArgs("build", "-o", filepath.Join(stageGen, "descriptor", "image.binpb"))...); err != nil {
		return fmt.Errorf("buf build descriptor: %w", err)
	}
	if err := g.writeManifests(stageGen, scope); err != nil {
		return err
	}
	if err := g.publish(stageGen, scope, len(g.cfg.Scenarios) == 0); err != nil {
		return err
	}
	return nil
}

// Verify carries the same named-error retention contract as Generate: a real
// mismatch keeps its staging tree so the diff can be inspected, a cancellation
// does not.
func (g *Generator) Verify(ctx context.Context) (err error) {
	unlock, err := g.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	g.reapStaleStages(time.Now())

	allScenarios, err := genmanifest.ScenarioNames(g.cfg.ProtoRoot)
	if err != nil {
		return fmt.Errorf("list schema scenarios: %w", err)
	}
	stage, err := os.MkdirTemp(g.cfg.StageParent, ".proto-gen-verify-")
	if err != nil {
		return fmt.Errorf("create verification staging directory: %w", err)
	}
	defer func() {
		if stageRetainable(err) {
			logf(g.cfg.Logger, "protogen: verification failed; staging tree retained at %s", stage)
			return
		}
		_ = os.RemoveAll(stage)
	}()
	stageGen := filepath.Join(stage, "gen")
	if err := os.MkdirAll(stageGen, 0o755); err != nil {
		return err
	}
	if err := g.seedGeneratedMetadata(stageGen); err != nil {
		return err
	}
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", bufCommandArgs("generate", "--output", stage)...); err != nil {
		return fmt.Errorf("buf generate: %w", err)
	}
	if err := markPythonPackages(stageGen); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stageGen, "descriptor"), 0o755); err != nil {
		return fmt.Errorf("create staged descriptor directory: %w", err)
	}
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", bufCommandArgs("build", "-o", filepath.Join(stageGen, "descriptor", "image.binpb"))...); err != nil {
		return fmt.Errorf("buf build descriptor: %w", err)
	}
	if err := g.writeManifests(stageGen, allScenarios); err != nil {
		return err
	}
	findings, err := compareTrees(stageGen, filepath.Join(g.cfg.ProtoRoot, "gen"))
	if err != nil {
		return err
	}
	if len(findings) > 0 {
		return fmt.Errorf("generated artifacts differ (%d findings): %s", len(findings), strings.Join(findings, ", "))
	}
	return nil
}

// Descriptor rebuilds only the descriptor artifact and publishes it with the
// same lock and single-file atomic rename used by the full pipeline.
func (g *Generator) Descriptor(ctx context.Context) error {
	unlock, err := g.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	stage, err := os.MkdirTemp(g.cfg.StageParent, ".proto-descriptor-")
	if err != nil {
		return fmt.Errorf("create descriptor staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	target := filepath.Join(stage, "image.binpb")
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", bufCommandArgs("build", "-o", target)...); err != nil {
		return fmt.Errorf("buf build descriptor: %w", err)
	}
	return publishFile(target, filepath.Join(g.cfg.ProtoRoot, "gen", "descriptor", "image.binpb"))
}

func (g *Generator) resolveScope(all []string) ([]string, error) {
	if len(g.cfg.Scenarios) == 0 {
		return append([]string(nil), all...), nil
	}
	known := make(map[string]struct{}, len(all))
	for _, scenario := range all {
		known[scenario] = struct{}{}
	}
	selected := make(map[string]struct{})
	for _, scenario := range g.cfg.Scenarios {
		scenario = strings.TrimSpace(scenario)
		if _, ok := known[scenario]; !ok {
			return nil, fmt.Errorf("unknown scenario %q", scenario)
		}
		selected[scenario] = struct{}{}
	}
	// A schema is a dependency of every scenario whose input closure contains
	// it. Inverting the trusted lockfile closures keeps scoped output coherent.
	for _, scenario := range all {
		files, _, err := genmanifest.InputClosure(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot}, scenario)
		if err != nil {
			return nil, fmt.Errorf("resolve imports for %s: %w", scenario, err)
		}
		for _, requested := range g.cfg.Scenarios {
			needle := filepath.Join("schemas", requested) + string(filepath.Separator)
			for _, file := range files {
				if strings.HasPrefix(filepath.FromSlash(file), needle) {
					selected[scenario] = struct{}{}
					break
				}
			}
		}
	}
	out := make([]string, 0, len(selected))
	for scenario := range selected {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out, nil
}

func (g *Generator) seedGeneratedMetadata(stageGen string) error {
	source := filepath.Join(g.cfg.ProtoRoot, "gen", "typescript", "package.json")
	dest := filepath.Join(stageGen, "typescript", "package.json")
	if raw, err := os.ReadFile(source); err == nil {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, raw, 0o644); err != nil {
			return fmt.Errorf("seed generated TypeScript metadata: %w", err)
		}
	}
	return nil
}

func markPythonPackages(genRoot string) error {
	root := filepath.Join(genRoot, "python")
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		return os.WriteFile(filepath.Join(path, "py.typed"), nil, 0o644)
	})
}

func (g *Generator) writeManifests(stageGen string, scenarios []string) error {
	for _, scenario := range scenarios {
		manifest, err := genmanifest.BuildManifest(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot, OutputRoot: stageGen}, scenario)
		if err != nil {
			return fmt.Errorf("build manifest %s: %w", scenario, err)
		}
		if err := genmanifest.WriteManifest(genmanifest.ManifestPathAt(stageGen, scenario), manifest); err != nil {
			return fmt.Errorf("write manifest %s: %w", scenario, err)
		}
	}
	return nil
}

func (g *Generator) publish(stageGen string, scenarios []string, full bool) error {
	targetGen := filepath.Join(g.cfg.ProtoRoot, "gen")
	if full {
		for _, dir := range []string{"go", "python", "typescript", "manifests"} {
			if err := publishDirectory(filepath.Join(stageGen, dir), filepath.Join(targetGen, dir)); err != nil {
				return err
			}
		}
		return publishFile(filepath.Join(stageGen, "descriptor", "image.binpb"), filepath.Join(targetGen, "descriptor", "image.binpb"))
	}
	for _, scenario := range scenarios {
		for _, rel := range genmanifest.ScenarioOutputDirs(scenario) {
			rel = strings.TrimPrefix(rel, "gen/")
			if err := publishDirectory(filepath.Join(stageGen, rel), filepath.Join(targetGen, rel)); err != nil {
				return err
			}
		}
		manifest := filepath.Base(genmanifest.ManifestPathAt(stageGen, scenario))
		if err := publishFile(filepath.Join(stageGen, "manifests", manifest), filepath.Join(targetGen, "manifests", manifest)); err != nil {
			return err
		}
	}
	if err := g.publishSharedImports(stageGen, targetGen, scenarios); err != nil {
		return err
	}
	return publishFile(filepath.Join(stageGen, "descriptor", "image.binpb"), filepath.Join(targetGen, "descriptor", "image.binpb"))
}

func (g *Generator) publishSharedImports(stageGen, targetGen string, scenarios []string) error {
	for _, root := range []string{"typescript", filepath.Join("typescript", "js")} {
		stageRoot := filepath.Join(stageGen, root)
		targetRoot := filepath.Join(targetGen, root)
		entries, err := os.ReadDir(stageRoot)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == "package.json" || isScenarioOutput(entry.Name(), scenarios) {
				continue
			}
			logf(g.cfg.Logger, "protogen: publishing shared import directory: %s", filepath.ToSlash(filepath.Join("gen", root, entry.Name())))
			if err := publishDirectory(filepath.Join(stageRoot, entry.Name()), filepath.Join(targetRoot, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func isScenarioOutput(name string, scenarios []string) bool {
	for _, scenario := range scenarios {
		if name == scenario || name == strings.ReplaceAll(scenario, "-", "_") {
			return true
		}
	}
	return false
}

func publishDirectory(source, target string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if sameDigestTree(source, target) {
		return os.RemoveAll(source)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	staged, err := treeDigests(source)
	if err != nil {
		return err
	}
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		return publishFile(path, filepath.Join(target, rel))
	}); err != nil {
		return fmt.Errorf("publish generated directory %s: %w", target, err)
	}
	if err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}
		if _, ok := staged[filepath.ToSlash(rel)]; ok {
			return nil
		}
		return os.Remove(path)
	}); err != nil {
		return fmt.Errorf("remove stale generated files from %s: %w", target, err)
	}
	return os.RemoveAll(source)
}

func publishFile(source, target string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if sameDigest(source, target) {
		return os.Remove(source)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("publish generated file %s: %w", target, err)
	}
	return nil
}

func sameDigestTree(a, b string) bool {
	aFiles, err := treeDigests(a)
	if err != nil {
		return false
	}
	bFiles, err := treeDigests(b)
	if err != nil {
		return false
	}
	return equalDigests(aFiles, bFiles)
}

func sameDigest(a, b string) bool {
	left, err := fileDigest(a)
	if err != nil {
		return false
	}
	right, err := fileDigest(b)
	return err == nil && left == right
}

func compareTrees(staged, committed string) ([]string, error) {
	left, err := treeDigests(staged)
	if err != nil {
		return nil, err
	}
	right, err := treeDigests(committed)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		keys[key] = struct{}{}
	}
	for key := range right {
		keys[key] = struct{}{}
	}
	var findings []string
	for key := range keys {
		leftDigest, leftPresent := left[key]
		rightDigest, rightPresent := right[key]
		switch {
		case leftPresent && rightPresent && leftDigest != rightDigest:
			findings = append(findings, "content-differs:"+filepath.ToSlash(filepath.Join("gen", key)))
		case leftPresent && !rightPresent:
			findings = append(findings, "untracked-generated:"+filepath.ToSlash(filepath.Join("gen", key)))
		case !leftPresent && rightPresent:
			findings = append(findings, "missing-generated:"+filepath.ToSlash(filepath.Join("gen", key)))
		}
	}
	sort.Strings(findings)
	return findings, nil
}

func treeDigests(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && path == root {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		digest, err := fileDigest(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = digest
		return nil
	})
	return out, err
}

func equalDigests(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, digest := range a {
		if b[key] != digest {
			return false
		}
	}
	return true
}

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func (g *Generator) acquireLock(ctx context.Context) (func(), error) {
	processLock.Lock()
	unlock, err := g.acquireFileLock(ctx)
	if err != nil {
		processLock.Unlock()
		return nil, err
	}
	return func() {
		unlock()
		processLock.Unlock()
	}, nil
}

func (g *Generator) acquireFileLock(ctx context.Context) (func(), error) {
	path := g.cfg.LockPath
	if path == "" {
		home, err := platform.HomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve operator home: %w", err)
		}
		path, err = platform.ResolveHomePath(home, filepath.Join(".vrooli", "locks", "proto-generation.lock"))
		if err != nil {
			return nil, fmt.Errorf("resolve generation lock: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create generation lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open generation lock: %w", err)
	}
	started := time.Now()
	waited := false
	for {
		release, lockErr := platform.LockFile(file, true)
		if lockErr == nil {
			if waited {
				logf(g.cfg.Logger, "protogen: acquired generation lock after %s", time.Since(started).Round(time.Millisecond))
			}
			if _, err := file.Seek(0, 0); err == nil {
				_ = file.Truncate(0)
				_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
				_ = file.Sync()
			}
			return func() { release(); _ = file.Close() }, nil
		}
		if !errors.Is(lockErr, platform.ErrLockUnavailable) {
			_ = file.Close()
			return nil, fmt.Errorf("acquire generation lock: %w", lockErr)
		}
		if !waited {
			waited = true
			logf(g.cfg.Logger, "protogen: waiting for generation lock held by PID %s", holderPID(file))
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, ctx.Err()
		case <-time.After(g.cfg.LockPoll):
		}
	}
}

func holderPID(file *os.File) string {
	if _, err := file.Seek(0, 0); err != nil {
		return "unknown"
	}
	var raw [64]byte
	n, _ := file.Read(raw[:])
	pid := strings.TrimSpace(string(raw[:n]))
	if pid == "" {
		return "unknown"
	}
	return pid
}

func runTool(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func logf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
