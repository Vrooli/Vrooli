package installgateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	pythonLockStageMaxFiles            = 6
	pythonLockStageMaxEntries          = 2048
	pythonLockStageMaxStageFiles       = 512
	pythonLockStageFileBytes     int64 = 32 << 20
	pythonLockStagePollInterval        = 25 * time.Millisecond
	// Each bounded lock can coexist as the resolver input, generated output,
	// canonical output and canonical .in file. Reserve a little room for index
	// metadata and filesystem bookkeeping across all admitted lockfiles.
	pythonLockStagePeakBytes int64 = 800 << 20
)

// PythonLockStage is a bounded, unpublished wheel-lock resolution. It contains
// no wheels or installed packages; Apply only publishes its exact lock/input
// files after verifying that the authored files still match the staged bases.
type PythonLockStage struct {
	Dir     string
	items   []pythonLockPublication
	bases   map[string][32]byte
	missing map[string]bool
	newLock string
}

func (s *PythonLockStage) Release() {
	if s != nil && s.Dir != "" {
		_ = os.RemoveAll(s.Dir)
		s.Dir = ""
	}
}

func (s *PythonLockStage) Publish() error {
	if s == nil || s.Dir == "" {
		return fmt.Errorf("Python lock stage is unavailable")
	}
	for path, expected := range s.bases {
		data, err := os.ReadFile(path)
		if s.missing[path] {
			if !os.IsNotExist(err) {
				return fmt.Errorf("Python lock target changed after admission: %s", filepath.Base(path))
			}
			continue
		}
		if err != nil || sha256.Sum256(data) != expected {
			return fmt.Errorf("Python lock target changed after admission: %s", filepath.Base(path))
		}
	}
	if s.newLock != "" {
		if len(s.items) != 1 || s.items[0].output != s.newLock || !s.missing[s.newLock] {
			return fmt.Errorf("invalid first Python lock publication stage")
		}
		// Link publishes without replacing a lock another owner may have created
		// after admission. The source remains private and is removed by Release.
		return os.Link(s.items[0].source, s.newLock)
	}
	return publishPythonLocks(s.items, os.Rename)
}

// stagePythonLocks resolves the complete lock closure before admission. Work is
// confined to a private directory on the surface filesystem, has a fixed file
// count/size/time bound, and disables uv's persistent cache.
func stagePythonLocks(ctx context.Context, r Resolution) (*PythonLockStage, error) {
	if r.PythonBootstrap != nil {
		return stagePythonBootstrapLock(ctx, r)
	}
	if len(r.PythonLockfiles) == 0 || len(r.PythonLockfiles) > pythonLockStageMaxFiles {
		return nil, fmt.Errorf("Python lock stage requires 1-%d lockfiles", pythonLockStageMaxFiles)
	}
	version := ""
	for _, arg := range r.Argv {
		if strings.HasPrefix(arg, r.PackageName+"==") {
			version = strings.TrimPrefix(arg, r.PackageName+"==")
		}
	}
	if r.PackageName == "" || version == "" {
		return nil, fmt.Errorf("Python lock stage requires an explicit package version")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp(r.SurfaceRoot, ".sda-python-lock-stage-")
	if err != nil {
		return nil, fmt.Errorf("create bounded Python lock stage: %w", err)
	}
	stage := &PythonLockStage{Dir: dir, bases: map[string][32]byte{}, missing: map[string]bool{}}
	indexManifest := filepath.Join(r.SurfaceRoot, "resource.json")
	if data, err := readRegularBounded(indexManifest, 1<<20); err == nil {
		stage.bases[indexManifest] = sha256.Sum256(data)
	} else if os.IsNotExist(err) {
		stage.missing[indexManifest] = true
	} else {
		return nil, fmt.Errorf("read Python index manifest: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			stage.Release()
		}
	}()
	pending := make([]pythonLockPublication, 0, 2*len(r.PythonLockfiles))
	for i, lock := range r.PythonLockfiles {
		if !filepath.IsAbs(lock) || filepath.Clean(filepath.Dir(lock)) != filepath.Clean(r.SurfaceRoot) {
			return nil, fmt.Errorf("Python lock path is outside the admitted surface root")
		}
		lockData, err := readRegularBounded(lock, pythonLockStageFileBytes)
		if err != nil {
			return nil, fmt.Errorf("read Python lock %s: %w", filepath.Base(lock), err)
		}
		stage.bases[lock] = sha256.Sum256(lockData)
		inputPath := filepath.Join(dir, fmt.Sprintf("requirements-%d.in", i))
		outputPath := filepath.Join(dir, fmt.Sprintf("requirements-%d.generated.lock", i))
		lockInput := string(lockData)
		if pythonName(r.PackageName) == "onnxruntime-gpu" {
			lockInput, err = removePythonRequirement(lockInput, "onnxruntime")
			if err != nil {
				return nil, err
			}
		}
		updatedInput, err := replacePythonRequirement(lockInput, r.PackageName, version)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(inputPath, []byte(updatedInput), 0o600); err != nil {
			return nil, err
		}
		indexSet := r.PythonIndexSets[filepath.Base(lock)]
		if strings.TrimSpace(indexSet.Primary) == "" {
			indexSet.Primary = "https://pypi.org/simple"
		}
		// Keep all resolver state under the private stage. uv's persistent cache
		// remains disabled; isolated cache/home paths also prevent helpers from
		// falling back to user-level locations.
		cacheDir := filepath.Join(dir, "cache")
		tempDir := filepath.Join(dir, "tmp")
		homeDir := filepath.Join(dir, "home")
		for _, path := range []string{cacheDir, tempDir, homeDir} {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return nil, fmt.Errorf("create isolated Python lock workspace: %w", err)
			}
		}
		compileCtx := context.WithValue(ctx, installEnvironmentContextKey{}, []string{
			"HOME=" + homeDir,
			"XDG_CACHE_HOME=" + cacheDir,
			"UV_CACHE_DIR=" + filepath.Join(cacheDir, "uv"),
			"PIP_CACHE_DIR=" + filepath.Join(cacheDir, "pip"),
			"TMPDIR=" + tempDir,
			"TMP=" + tempDir,
			"TEMP=" + tempDir,
		})
		output, err := runPythonLockStageGuarded(ctx, dir, pythonLockStagePeakBytes, pythonLockStageMaxEntries, pythonLockStageFileBytes,
			func(runCtx context.Context) (string, error) {
				return compilePythonLockBounded(compileCtxWithParent(compileCtx, runCtx), r.SurfaceRoot, inputPath, outputPath, indexSet, pythonLockStageFileBytes)
			})
		if err != nil {
			return nil, fmt.Errorf("uv wheel-only lock resolution failed for %s: %w: %s", filepath.Base(lock), err, output)
		}
		generated, err := readRegularBounded(outputPath, pythonLockStageFileBytes)
		if err != nil {
			return nil, fmt.Errorf("generated Python lock exceeds stage bound: %w", err)
		}
		rel, err := filepath.Rel(r.RepositoryRoot, lock)
		if err != nil {
			return nil, err
		}
		canonical := []byte(canonicalizePythonLockOutput(generated, inputPath, filepath.ToSlash(rel)))
		if int64(len(canonical)) > pythonLockStageFileBytes {
			return nil, fmt.Errorf("generated Python lock exceeds %d-byte stage bound", pythonLockStageFileBytes)
		}
		stagedOutput := filepath.Join(dir, fmt.Sprintf("requirements-%d.canonical.lock", i))
		if err := os.WriteFile(stagedOutput, canonical, 0o600); err != nil {
			return nil, err
		}
		pending = append(pending, pythonLockPublication{source: stagedOutput, output: lock})
		input := strings.TrimSuffix(lock, ".lock") + ".in"
		if data, err := pythonInputBytes(input); err == nil {
			canonicalInput := string(data)
			if pythonName(r.PackageName) == "onnxruntime-gpu" {
				canonicalInput, err = removePythonRequirement(canonicalInput, "onnxruntime")
				if err != nil {
					return nil, err
				}
			}
			updated, err := updatePythonInput(canonicalInput, r.PackageName, version)
			if err != nil {
				return nil, err
			}
			stage.bases[input] = sha256.Sum256(data)
			stagedInput := filepath.Join(dir, fmt.Sprintf("canonical-%d.in", i))
			if err := os.WriteFile(stagedInput, []byte(updated), 0o600); err != nil {
				return nil, err
			}
			pending = append(pending, pythonLockPublication{source: stagedInput, output: input})
		} else if os.IsNotExist(err) {
			stage.missing[input] = true
		} else {
			return nil, fmt.Errorf("read canonical Python input: %w", err)
		}
	}
	stage.items = pending
	if _, err := inspectPythonLockStage(dir, pythonLockStagePeakBytes, pythonLockStageMaxEntries, pythonLockStageFileBytes); err != nil {
		return nil, err
	}
	failed = false
	return stage, nil
}

// stagePythonBootstrapLock gives first-lock resources the same bounded,
// isolated resolver and capacity admission as updates to existing lockfiles.
// It stages only a validated hashed lock; it never installs runtime packages.
func stagePythonBootstrapLock(ctx context.Context, r Resolution) (*PythonLockStage, error) {
	plan := r.PythonBootstrap
	if plan == nil || len(r.PythonLockfiles) != 1 || filepath.Base(r.PythonLockfiles[0]) != "requirements.lock" {
		return nil, fmt.Errorf("Python bootstrap requires one declared requirements.lock output")
	}
	lock := r.PythonLockfiles[0]
	if _, err := os.Lstat(lock); err == nil {
		return nil, fmt.Errorf("Python bootstrap output already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	input, err := pythonInputBytes(plan.InputPath)
	if err != nil || pythonDigest(input) != plan.InputSHA256 {
		return nil, fmt.Errorf("Python bootstrap authored input changed after approval")
	}
	manifestPath := filepath.Join(r.SurfaceRoot, "resource.json")
	manifest, err := pythonInputBytes(manifestPath)
	if err != nil || pythonDigest(manifest) != plan.ManifestSHA256 {
		return nil, fmt.Errorf("Python bootstrap index manifest changed after approval")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp(r.SurfaceRoot, ".sda-python-lock-stage-")
	if err != nil {
		return nil, fmt.Errorf("create bounded Python bootstrap stage: %w", err)
	}
	stage := &PythonLockStage{Dir: dir, bases: map[string][32]byte{plan.InputPath: sha256.Sum256(input), manifestPath: sha256.Sum256(manifest)}, missing: map[string]bool{lock: true}, newLock: lock}
	failed := true
	defer func() {
		if failed {
			stage.Release()
		}
	}()
	inputPath, outputPath := filepath.Join(dir, "requirements.in"), filepath.Join(dir, "requirements.generated.lock")
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		return nil, err
	}
	for _, path := range []string{"cache", "tmp", "home"} {
		if err := os.MkdirAll(filepath.Join(dir, path), 0o700); err != nil {
			return nil, fmt.Errorf("create isolated Python bootstrap workspace: %w", err)
		}
	}
	cacheDir, tempDir, homeDir := filepath.Join(dir, "cache"), filepath.Join(dir, "tmp"), filepath.Join(dir, "home")
	compileCtx := context.WithValue(ctx, installEnvironmentContextKey{}, []string{
		"HOME=" + homeDir, "XDG_CACHE_HOME=" + cacheDir, "UV_CACHE_DIR=" + filepath.Join(cacheDir, "uv"),
		"PIP_CACHE_DIR=" + filepath.Join(cacheDir, "pip"), "TMPDIR=" + tempDir, "TMP=" + tempDir, "TEMP=" + tempDir,
	})
	output, err := runPythonLockStageGuarded(ctx, dir, pythonLockStagePeakBytes, pythonLockStageMaxEntries, pythonLockStageFileBytes,
		func(runCtx context.Context) (string, error) {
			return compilePythonLockBounded(compileCtxWithParent(compileCtx, runCtx), r.SurfaceRoot, inputPath, outputPath, plan.Indexes, pythonLockStageFileBytes)
		})
	if err != nil {
		return nil, fmt.Errorf("uv wheel-only bootstrap failed: %w: %s", err, output)
	}
	generated, err := readRegularBounded(outputPath, pythonLockStageFileBytes)
	if err != nil {
		return nil, fmt.Errorf("generated Python lock exceeds stage bound: %w", err)
	}
	if err := validatePythonGeneratedLock(generated, plan.Requirements); err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(r.RepositoryRoot, lock)
	if err != nil {
		return nil, err
	}
	canonical := []byte(canonicalizePythonLockOutput(generated, inputPath, filepath.ToSlash(rel)))
	if int64(len(canonical)) > pythonLockStageFileBytes {
		return nil, fmt.Errorf("generated Python lock exceeds %d-byte stage bound", pythonLockStageFileBytes)
	}
	stagedLock := filepath.Join(dir, "requirements.canonical.lock")
	if err := os.WriteFile(stagedLock, canonical, 0o644); err != nil {
		return nil, err
	}
	stage.items = []pythonLockPublication{{source: stagedLock, output: lock}}
	if _, err := inspectPythonLockStage(dir, pythonLockStagePeakBytes, pythonLockStageMaxEntries, pythonLockStageFileBytes); err != nil {
		return nil, err
	}
	failed = false
	return stage, nil
}

type pythonLockStageUsage struct {
	bytes   int64
	entries int
	files   int
}

func inspectPythonLockStage(root string, maxBytes int64, maxEntries int, maxFileBytes int64) (pythonLockStageUsage, error) {
	var usage pythonLockStageUsage
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// uv creates and removes short-lived metadata directories while the
			// live guard walks the private stage. A child disappearing between
			// readdir and stat is not growth; the next poll and final scan verify
			// the stable aggregate. Missing stage root still fails in WalkDir.
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if path == root {
			return nil
		}
		usage.entries++
		if usage.entries > maxEntries {
			return fmt.Errorf("Python lock stage exceeded %d filesystem entries (observed at least %d)", maxEntries, usage.entries)
		}
		if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
			return fmt.Errorf("Python lock stage contains unsupported filesystem entry %s", filepath.Base(path))
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		usage.files++
		if usage.files > pythonLockStageMaxStageFiles {
			return fmt.Errorf("Python lock stage exceeded %d regular files", pythonLockStageMaxStageFiles)
		}
		if info.Size() > maxFileBytes {
			return fmt.Errorf("Python lock stage file %s grew to %d bytes, above %d-byte limit", filepath.Base(path), info.Size(), maxFileBytes)
		}
		usage.bytes += info.Size()
		if usage.bytes > maxBytes {
			return fmt.Errorf("Python lock stage grew beyond %d-byte peak", maxBytes)
		}
		return nil
	})
	if err != nil {
		return usage, err
	}
	return usage, nil
}

func runPythonLockStageGuarded(ctx context.Context, dir string, maxBytes int64, maxEntries int, maxFileBytes int64, run func(context.Context) (string, error)) (string, error) {
	if _, err := inspectPythonLockStage(dir, maxBytes, maxEntries, maxFileBytes); err != nil {
		return "", err
	}
	guardCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	guardDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(pythonLockStagePollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-guardCtx.Done():
				guardDone <- nil
				return
			case <-ticker.C:
				if _, err := inspectPythonLockStage(dir, maxBytes, maxEntries, maxFileBytes); err != nil {
					guardDone <- err
					cancel()
					return
				}
			}
		}
	}()
	output, runErr := run(guardCtx)
	cancel()
	guardErr := <-guardDone
	if guardErr != nil {
		return output, fmt.Errorf("Python lock stage aborted by live growth guard: %w", guardErr)
	}
	if runErr != nil {
		return output, runErr
	}
	if _, err := inspectPythonLockStage(dir, maxBytes, maxEntries, maxFileBytes); err != nil {
		return output, err
	}
	return output, nil
}

func compileCtxWithParent(values context.Context, parent context.Context) context.Context {
	return context.WithValue(parent, installEnvironmentContextKey{}, installEnvironmentFromContext(values))
}
