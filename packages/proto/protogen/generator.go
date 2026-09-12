// Package protogen owns protobuf generation as a cross-platform Go pipeline.
// It never mutates the committed generated tree while tools are running:
// output is built beside it and published only after validation.
package protogen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

type ToolRunner func(ctx context.Context, dir, name string, args ...string) error

type Config struct {
	RepoRoot         string
	ProtoRoot        string
	LockPath         string
	StageParent      string
	Scenarios        []string
	Changed          bool
	Logger           io.Writer
	RunTool          ToolRunner
	LockPoll         time.Duration
	ArtifactRoot     string
	PublishArtifacts bool
}

type Generator struct {
	cfg Config
}

var processLock sync.Mutex

const protoLockHeldEnv = "VROOLI_PROTO_LOCK_HELD"

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
	if config.PublishArtifacts && strings.TrimSpace(config.ArtifactRoot) == "" {
		config.ArtifactRoot = DefaultArtifactRoot("")
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
	var scope []string
	var excluded []string
	if g.cfg.Changed && len(g.cfg.Scenarios) > 0 {
		changed, changedErr := genmanifest.ChangedScenarios(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot})
		if changedErr != nil {
			return fmt.Errorf("resolve changed schema scope: %w", changedErr)
		}
		requested, requestedErr := g.resolveScope(allScenarios)
		if requestedErr != nil {
			return requestedErr
		}
		allowed := make(map[string]bool, len(requested))
		for _, scenario := range requested {
			allowed[scenario] = true
		}
		for _, scenario := range changed {
			if allowed[scenario] {
				scope = append(scope, scenario)
			}
		}
		sort.Strings(scope)
		logf(g.cfg.Logger, "protogen: scope rule=changed intersect requested import closure")
	} else if g.cfg.Changed {
		scope, err = genmanifest.ChangedScenarios(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot})
		if err != nil {
			return fmt.Errorf("resolve changed schema scope: %w", err)
		}
	} else if len(g.cfg.Scenarios) > 0 {
		scope, err = g.resolveScope(allScenarios)
		if err != nil {
			return err
		}
	} else {
		scope = allScenarios
		logf(g.cfg.Logger, "protogen: scope rule=full schema tree")
		valid, failures := g.validateFleetOwners(ctx, allScenarios)
		if len(failures) > 0 {
			scope = valid
			excluded = make([]string, 0, len(failures))
			for owner, ownerErr := range failures {
				excluded = append(excluded, owner)
				logf(g.cfg.Logger, "protogen: excluding schema owner %q: %v", owner, annotateBufError("validate", 1, ownerErr))
			}
			sort.Strings(excluded)
			logf(g.cfg.Logger, "protogen: scope rule=fleet minus invalid owners and their import dependents")
		}
	}
	logf(g.cfg.Logger, "protogen: publishing scenarios: %s", strings.Join(scope, ", "))
	// A changed run with no changed lock is already complete. In particular,
	// do not rebuild the descriptor or invoke buf just to discover that every
	// generated byte is still current.
	if g.cfg.Changed && len(scope) == 0 {
		return nil
	}
	inputFiles, sourceDigest, err := g.captureInputState(scope)
	if err != nil {
		return fmt.Errorf("capture Proto input state: %w", err)
	}

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
	scoped := g.cfg.Changed || len(g.cfg.Scenarios) > 0 || len(excluded) > 0
	parentArtifactID, err := g.seedGenerationBase(ctx, stageGen, scoped)
	if err != nil {
		return err
	}
	if err := g.seedGeneratedMetadata(stageGen); err != nil {
		return err
	}

	args := generateArgs(scope, stage, scoped)
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", args...); err != nil {
		return annotateBufError("generate", len(scope), err)
	}
	if err := markPythonPackages(stageGen); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(stageGen, "descriptor"), 0o755); err != nil {
		return fmt.Errorf("create staged descriptor directory: %w", err)
	}
	// The descriptor image is a whole-tree artifact: program-runtime's binding registry,
	// code-facts and proto-health all resolve against it. Building it scoped and publishing that
	// over the committed image truncated it to the scoped packages and silently emptied every
	// other scenario's bindings, so it is always built unscoped. A scoped run still tolerates a
	// failure here — an unrelated broken schema elsewhere in the tree must not block it — but
	// then publishes no descriptor at all rather than a partial one.
	descriptorBuilt := true
	if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", bufCommandArgs("build", "-o", filepath.Join(stageGen, "descriptor", "image.binpb"))...); err != nil {
		if !scoped {
			return annotateBufError("build descriptor", len(scope), err)
		}
		descriptorBuilt = false
		logf(g.cfg.Logger, "protogen: whole-tree descriptor unavailable during a scoped run for %d scenario(s); committed descriptor left untouched: %v", len(scope), annotateBufError("build descriptor", len(scope), err))
	}
	if err := g.writeManifests(stageGen, scope); err != nil {
		return err
	}
	if _, currentDigest, err := g.captureInputState(scope); err != nil {
		return fmt.Errorf("recheck Proto input state: %w", err)
	} else if currentDigest != sourceDigest {
		return fmt.Errorf("%w: captured=%s current=%s", ErrArtifactSourceDrift, sourceDigest, currentDigest)
	}
	if g.cfg.PublishArtifacts {
		if err := g.publishArtifact(ctx, stageGen, sourceDigest, inputFiles, scope, parentArtifactID); err != nil {
			return err
		}
		// The repository tree is a compatibility view only. Replace it from the
		// already-validated snapshot as one directory transaction so legacy
		// readers cannot observe the generator's per-file staging sequence.
		if err := g.publishSelectedCompatibility(ctx); err != nil {
			return err
		}
	} else if err := g.publish(stageGen, scope, !scoped, descriptorBuilt); err != nil {
		return err
	}
	if len(excluded) > 0 {
		return fmt.Errorf("protogen: generated %d healthy scenario(s); excluded invalid owner(s): %s", len(scope), strings.Join(excluded, ", "))
	}
	return nil
}

func (g *Generator) captureInputState(scenarios []string) ([]string, string, error) {
	filesByPath := map[string]struct{}{}
	digestsByPath := map[string]string{}
	type closure struct {
		Scenario string `json:"scenario"`
		Digest   string `json:"digest"`
	}
	closures := make([]closure, 0, len(scenarios))
	for _, scenario := range scenarios {
		files, digest, err := genmanifest.InputClosure(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot}, scenario)
		if err != nil {
			return nil, "", fmt.Errorf("resolve input closure for %s: %w", scenario, err)
		}
		for _, file := range files {
			filesByPath[file] = struct{}{}
		}
		closures = append(closures, closure{Scenario: scenario, Digest: digest})
	}
	// Closure traversal covers imported schema bytes. These are the other
	// generation inputs whose edits must also invalidate a candidate while the
	// shared worktree is in flight.
	for _, relative := range []string{"buf.yaml", "buf.gen.yaml", "buf.lock", "cmd/protogen", "protogen", "genmanifest", "vendor"} {
		root := filepath.Join(g.cfg.ProtoRoot, filepath.FromSlash(relative))
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) && path == root {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("unsupported generation input %s", path)
			}
			relativePath, err := filepath.Rel(g.cfg.ProtoRoot, path)
			if err != nil {
				return err
			}
			relativePath = filepath.ToSlash(relativePath)
			digest, err := artifactFileDigest(path)
			if err != nil {
				return err
			}
			filesByPath[relativePath] = struct{}{}
			digestsByPath[relativePath] = digest
			return nil
		}); err != nil {
			return nil, "", err
		}
	}
	files := make([]string, 0, len(filesByPath))
	for file := range filesByPath {
		files = append(files, file)
	}
	sort.Strings(files)
	var fingerprints []struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
	}
	for _, file := range files {
		if digest, ok := digestsByPath[file]; ok {
			fingerprints = append(fingerprints, struct {
				Path   string `json:"path"`
				Digest string `json:"digest"`
			}{Path: file, Digest: digest})
		}
	}
	encoded, err := json.Marshal(struct {
		Closures     []closure `json:"closures"`
		Fingerprints []struct {
			Path   string `json:"path"`
			Digest string `json:"digest"`
		} `json:"fingerprints"`
	}{Closures: closures, Fingerprints: fingerprints})
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(encoded)
	return files, "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (g *Generator) seedGenerationBase(ctx context.Context, stageGen string, scoped bool) (string, error) {
	if !scoped {
		return "", nil
	}
	if g.cfg.PublishArtifacts {
		store, err := NewArtifactStore(g.cfg.ArtifactRoot)
		if err != nil {
			return "", err
		}
		if snapshot, resolveErr := store.Resolve(ctx); resolveErr == nil {
			parent := snapshot.ArtifactID
			defer snapshot.Close()
			if err := copyTree(ctx, snapshot.GenRoot(), stageGen); err != nil {
				return "", fmt.Errorf("seed staged tree from selected Proto artifact: %w", err)
			}
			return parent, nil
		} else if !errors.Is(resolveErr, ErrArtifactNotFound) {
			logf(g.cfg.Logger, "protogen: selected Proto artifact unavailable; seeding compatibility output: %v", resolveErr)
		}
	}
	// This is only a first-publication bootstrap. Once a runtime artifact
	// exists, scoped generation is based on that immutable snapshot instead of
	// files another agent may be editing in the shared checkout.
	if err := copyTree(ctx, filepath.Join(g.cfg.ProtoRoot, "gen"), stageGen); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("seed staged tree from compatibility output: %w", err)
	}
	return "", nil
}

func (g *Generator) publishArtifact(ctx context.Context, stageGen, sourceDigest string, inputFiles, scope []string, parentArtifactID string) error {
	toolchain, err := genmanifest.ToolchainFingerprint(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot})
	if err != nil {
		return fmt.Errorf("fingerprint Proto toolchain: %w", err)
	}
	toolchainBytes, err := json.Marshal(toolchain)
	if err != nil {
		return err
	}
	toolchainSum := sha256.Sum256(toolchainBytes)
	metadata, err := BuildArtifactMetadata(stageGen, sourceDigest, "sha256:"+hex.EncodeToString(toolchainSum[:]), inputFiles, scope, parentArtifactID)
	if err != nil {
		return fmt.Errorf("build Proto artifact metadata: %w", err)
	}
	metadata.ValidationRef = "validation-receipt.json"
	metadata.ValidationStatus = "passed"
	store, err := NewArtifactStore(g.cfg.ArtifactRoot)
	if err != nil {
		return err
	}
	_, err = store.Publish(ctx, Candidate{
		Metadata:   metadata,
		SourceRoot: stageGen,
		ValidationReceipt: &ArtifactValidationReceipt{
			Status: "passed",
			Checks: []string{"generation", "output-digests", "input-stability", "manifest-generation"},
		},
	})
	if err != nil {
		return fmt.Errorf("publish Proto artifact: %w", err)
	}
	logf(g.cfg.Logger, "protogen: published artifact id=%s source_digest=%s scope=%s", metadata.ArtifactID, sourceDigest, strings.Join(scope, ","))
	return nil
}

func (g *Generator) publishSelectedCompatibility(ctx context.Context) error {
	store, err := NewArtifactStore(g.cfg.ArtifactRoot)
	if err != nil {
		return err
	}
	snapshot, err := store.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("resolve published Proto artifact for compatibility view: %w", err)
	}
	defer snapshot.Close()
	if err := store.MaterializeCompatibilityView(ctx, snapshot, filepath.Join(g.cfg.ProtoRoot, "gen")); err != nil {
		return fmt.Errorf("publish Proto compatibility view: %w", err)
	}
	return nil
}

// validateFleetOwners gives an unscoped run a per-owner failure boundary. A
// broken leaf must not make buf generate the entire fleet impossible, while a
// dependent of that leaf must also be excluded because its input closure is
// not valid. The caller still returns a non-zero result after publishing the
// healthy owners, so the defect remains visible to CI and operators.
func (g *Generator) validateFleetOwners(ctx context.Context, owners []string) ([]string, map[string]error) {
	valid := make([]string, 0, len(owners))
	failures := map[string]error{}
	for _, owner := range owners {
		args := bufCommandArgs("build", "--path", filepath.ToSlash(filepath.Join("schemas", owner)))
		if err := g.cfg.RunTool(ctx, g.cfg.ProtoRoot, "buf", args...); err != nil {
			failures[owner] = err
			continue
		}
		valid = append(valid, owner)
	}
	if len(failures) == 0 {
		return valid, failures
	}
	for _, owner := range owners {
		if _, failed := failures[owner]; failed {
			continue
		}
		files, _, err := genmanifest.InputClosure(genmanifest.Options{RepoRoot: g.cfg.RepoRoot, ProtoRoot: g.cfg.ProtoRoot}, owner)
		if err != nil {
			failures[owner] = err
			continue
		}
		for _, file := range files {
			parts := strings.Split(filepath.ToSlash(file), "/")
			if len(parts) < 2 || parts[0] != "schemas" {
				continue
			}
			if _, failed := failures[parts[1]]; failed {
				failures[owner] = fmt.Errorf("input closure contains invalid schema owner %q", parts[1])
				break
			}
		}
	}
	filtered := valid[:0]
	for _, owner := range valid {
		if _, failed := failures[owner]; !failed {
			filtered = append(filtered, owner)
		}
	}
	return filtered, failures
}

func generateArgs(scope []string, stage string, scoped bool) []string {
	args := bufCommandArgs("generate", "--output", stage)
	if !scoped {
		return args
	}
	for _, scenario := range scope {
		args = append(args, "--path", filepath.ToSlash(filepath.Join("schemas", scenario)))
	}
	return args
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

func (g *Generator) publish(stageGen string, scenarios []string, full bool, publishDescriptor bool) error {
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
	if !publishDescriptor {
		// The whole-tree build failed during a scoped run; the committed image stays as it is.
		return nil
	}
	return publishFile(filepath.Join(stageGen, "descriptor", "image.binpb"), filepath.Join(targetGen, "descriptor", "image.binpb"))
}

func (g *Generator) publishSharedImports(stageGen, targetGen string, scenarios []string) error {
	for _, root := range []string{"typescript"} {
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
			// Merge, never prune: publishSharedImports runs only for a scoped
			// generation, whose staging tree holds just the shared types the
			// selected scenarios import.
			if err := publishDirectoryMerge(filepath.Join(stageRoot, entry.Name()), filepath.Join(targetRoot, entry.Name())); err != nil {
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

// publishDirectoryMerge publishes every staged file into target without
// removing target files the staging tree lacks.
//
// Use it wherever the staging tree is not authoritative for the whole target.
// A scoped generation only materializes the shared types that the selected
// scenarios happen to import, so pruning against that tree deletes every
// shared type the rest of the repository depends on — a full generation is
// then the only way to notice, because the loss is silent.
func publishDirectoryMerge(source, target string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
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
	return os.RemoveAll(source)
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
	if strings.EqualFold(strings.TrimSpace(os.Getenv(protoLockHeldEnv)), "true") {
		// Lifecycle setup holds the same cross-process Proto lock across
		// compatibility-view installation and consumer build. The generator is
		// invoked as a child of that owner and must not deadlock trying to
		// reacquire its own lock.
		return func() {}, nil
	}
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
	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("%s: %w", message, err)
		}
		return err
	}
	return nil
}

var bufOwnerPathRE = regexp.MustCompile(`(?:^|\s)(schemas/([^/]+)/[^:\s()]+)(?::(\d+))?(?::(\d+))?[:\s]+(.+)`)

func annotateBufError(operation string, scopeCount int, err error) error {
	message := strings.TrimSpace(err.Error())
	if match := bufOwnerPathRE.FindStringSubmatch(message); len(match) == 6 {
		line := match[3]
		if line == "" {
			line = "unknown"
		}
		return fmt.Errorf("buf %s failed for schema owner %q at %s:%s: %s (scope contained %d scenario(s)): %w", operation, match[2], match[1], line, strings.TrimSpace(match[5]), scopeCount, err)
	}
	return fmt.Errorf("buf %s failed (scope contained %d scenario(s)): %s: %w", operation, scopeCount, message, err)
}

func logf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
