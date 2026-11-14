package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// runtimeReleaseStore is the lifecycle's artifact owner, not a second process
// launcher. It retains code only; namespaces, credentials and data stay canonical.
func (r *Runner) runtimeReleaseStore() *retainedruntime.Store {
	return retainedruntime.NewStore(filepath.Join(r.Home, ".vrooli", "state", "scenario-releases"))
}

func (r *Runner) SelectedRuntimeReleaseID(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	key, err := scenarioruntime.ParseInstanceKey(name, "")
	if err != nil {
		return "", err
	}
	selection, err := r.runtimeReleaseStore().ReadSelection(key.Scenario, key.Variant)
	return selection.ReleaseID, err
}

// CaptureRuntimeRelease preserves today's declared artifact closure. It does not
// retrospectively prove loaded interpreted modules, healthy recovery, or database
// compatibility. Those remain qualification obligations of the scenario owner.
func (r *Runner) CaptureRuntimeRelease(ctx context.Context, name, customPath string) (retainedruntime.Release, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, "")
	if err != nil {
		return retainedruntime.Release{}, err
	}
	unlock, err := r.acquireScenarioLock(key.Slug())
	if err != nil {
		return retainedruntime.Release{}, err
	}
	defer unlock()
	item, err := r.loadScenario(key.Scenario, customPath)
	if err != nil {
		return retainedruntime.Release{}, err
	}
	item.Variant = key.Variant
	if selected, err := r.SelectedRuntimeReleaseID(ctx, key.Slug()); err != nil {
		return retainedruntime.Release{}, err
	} else if selected != "" {
		release, err := r.runtimeReleaseStore().Verify(ctx, key.Scenario, key.Variant, selected)
		if err != nil {
			return retainedruntime.Release{}, err
		}
		if _, err = r.bindRetainedRuntime(item, release); err != nil {
			return retainedruntime.Release{}, fmt.Errorf("selected runtime binding validation: %w", err)
		}
		return release, nil
	}
	if item, err = r.withSourceLayout(item); err != nil {
		return retainedruntime.Release{}, err
	}
	if item.SourcePath() != item.Path {
		return retainedruntime.Release{}, fmt.Errorf("capture runtime requires the canonical source, not a baseline split")
	}
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil {
		return retainedruntime.Release{}, err
	}
	if !view.Authoritative || view.Instance.Status != scenarioruntime.StatusRunning || view.Instance.BuildIdentity == "" {
		return retainedruntime.Release{}, fmt.Errorf("capture runtime requires an authoritative running instance with recorded build identity")
	}
	if !r.isRegistryRuntimeServingHealthy(item, view) {
		return retainedruntime.Release{}, fmt.Errorf("capture runtime requires current serving health; capture is not a recovery qualification")
	}
	manifest, err := os.ReadFile(item.ServicePath)
	if err != nil {
		return retainedruntime.Release{}, err
	}
	forbidden := []string{"data", "logs", "tmp", "api/data"}
	for _, c := range item.Manifest.Components {
		for _, p := range append(append([]string(nil), c.Run.DataDirs...), c.Run.LogDir) {
			if strings.TrimSpace(p) != "" {
				forbidden = append(forbidden, filepath.ToSlash(filepath.Clean(p)))
			}
		}
	}
	release, err := r.runtimeReleaseStore().Capture(ctx, retainedruntime.CaptureSpec{Scenario: key.Scenario, Variant: key.Variant, SourceDir: item.Path, BuildIdentity: view.Instance.BuildIdentity, SourceRuntimeIdentity: view.Instance.InstanceID, ServiceManifest: manifest, ForbiddenPaths: forbidden})
	if err != nil {
		return release, err
	}
	if _, err = r.bindRetainedRuntime(item, release); err != nil {
		return release, fmt.Errorf("captured bytes remain unselected: %w", err)
	}
	return release, nil
}

func (r *Runner) VerifyRuntimeRelease(ctx context.Context, name, id string) (retainedruntime.Release, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, "")
	if err != nil {
		return retainedruntime.Release{}, err
	}
	release, err := r.runtimeReleaseStore().Verify(ctx, key.Scenario, key.Variant, id)
	if err != nil {
		return retainedruntime.Release{}, err
	}
	// The verified release owns its capture source path, including captures made
	// with --path. Byte hashes alone do not validate current owner compatibility
	// or Node resolution outside the immutable tree.
	item, err := r.loadScenario(key.Scenario, release.SourceDir)
	if err != nil {
		return retainedruntime.Release{}, err
	}
	item.Variant = key.Variant
	if _, err = r.bindRetainedRuntime(item, release); err != nil {
		return retainedruntime.Release{}, fmt.Errorf("retained runtime binding validation: %w", err)
	}
	return release, nil
}

// selectRequestedRuntime is called under the ordinary scenario lock, after the
// durable start operation exists and before any build, provider fence or stop.
func (r *Runner) selectRequestedRuntime(name string, opts StartOptions) error {
	if opts.RetainedReleaseID == "" {
		if opts.ExpectedRetainedReleaseID != "" {
			return fmt.Errorf("expected runtime release requires an explicit selection")
		}
		return nil
	}
	if opts.ExpectedRetainedReleaseID == "" {
		return fmt.Errorf("runtime release selection requires its expected current release (source when none)")
	}
	if opts.lifecycleOperationID == "" {
		return fmt.Errorf("runtime release selection requires a durable start-operation identity")
	}
	id, expected := opts.RetainedReleaseID, opts.ExpectedRetainedReleaseID
	if id == "source" {
		id = ""
	}
	if expected == "source" {
		expected = ""
	}
	if id != "" && opts.ForceSetup {
		return fmt.Errorf("retained runtime cannot be rebuilt; select source explicitly first")
	}
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return err
	}
	item.Variant = opts.Variant
	if r.Engagements != nil {
		info, engaged, e := r.Engagements.Engagement(name)
		if e != nil {
			return e
		}
		if engaged || info.Slug != "" {
			return fmt.Errorf("runtime release selection conflicts with a baseline engagement")
		}
	}
	if id != "" {
		release, e := r.runtimeReleaseStore().Verify(ctxOrBackground(opts.Context), name, opts.Variant, id)
		if e != nil {
			return e
		}
		if _, e = r.bindRetainedRuntime(item, release); e != nil {
			return e
		}
	}
	_, err = r.runtimeReleaseStore().Select(ctxOrBackground(opts.Context), name, opts.Variant, id, expected, opts.lifecycleOperationID)
	return err
}

// retainedSourceLayout resolves a durable selection without mutating it. A
// missing/tampered release fails closed; it never falls back to the dirty source.
func (r *Runner) retainedSourceLayout(item scenario.Scenario) (scenario.Scenario, bool, error) {
	key := scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize()
	selection, err := r.runtimeReleaseStore().ReadSelection(key.Scenario, key.Variant)
	if err != nil {
		return item, false, err
	}
	if selection.ReleaseID == "" {
		if item.RetainedReleaseID != "" {
			data, readErr := os.ReadFile(item.ServicePath)
			if readErr != nil {
				return item, false, readErr
			}
			var manifest scenario.ServiceManifest
			if readErr = json.Unmarshal(data, &manifest); readErr != nil {
				return item, false, readErr
			}
			item.Manifest, item.SourceDir = manifest, ""
			item.RetainedReleaseID, item.RetainedBuildIdentity = "", ""
		}
		return item, false, nil
	}
	if r.Engagements != nil {
		info, engaged, e := r.Engagements.Engagement(item.Slug)
		if e != nil {
			return item, false, e
		}
		if engaged || info.Slug != "" {
			return item, false, fmt.Errorf("selected runtime conflicts with baseline engagement")
		}
	}
	release, err := r.runtimeReleaseStore().Verify(context.Background(), key.Scenario, key.Variant, selection.ReleaseID)
	if err != nil {
		return item, false, err
	}
	item, err = r.bindRetainedRuntime(item, release)
	return item, true, err
}

func normalizedJSON(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func (r *Runner) bindRetainedRuntime(item scenario.Scenario, release retainedruntime.Release) (scenario.Scenario, error) {
	if filepath.Clean(item.Path) != filepath.Clean(release.SourceDir) {
		return item, fmt.Errorf("retained runtime belongs to a different canonical scenario path")
	}
	if release.BuildIdentity == "" {
		return item, fmt.Errorf("retained runtime lacks its original build identity")
	}
	current, err := os.ReadFile(item.ServicePath)
	if err != nil {
		return item, err
	}
	a, err := normalizedJSON(current)
	if err != nil {
		return item, err
	}
	hash := sha256.Sum256(a)
	if hex.EncodeToString(hash[:]) != release.ServiceManifestSHA256 {
		return item, fmt.Errorf("runtime launch/data policy changed; owner compatibility review is required before selecting this release")
	}
	var contract struct {
		Version       int    `json:"version"`
		StateContract string `json:"state_contract"`
	}
	declaration := filepath.Join(item.Path, ".vrooli", "runtime-release.json")
	info, err := os.Lstat(declaration)
	if err != nil {
		return item, err
	}
	if !info.Mode().IsRegular() {
		return item, fmt.Errorf("runtime closure declaration must be a regular file")
	}
	data, err := os.ReadFile(declaration)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(data, &contract); err != nil {
		return item, err
	}
	if contract.Version != 1 || strings.TrimSpace(contract.StateContract) == "" || contract.StateContract != release.StateContract {
		return item, fmt.Errorf("runtime state compatibility contract changed or is absent; data is never restored by artifact recovery")
	}
	var frozen scenario.ServiceManifest
	if err = json.Unmarshal(current, &frozen); err != nil {
		return item, err
	}
	item.Manifest = frozen
	item.SourceDir = release.RuntimeDir
	for _, phase := range []scenario.Phase{frozen.Lifecycle.Develop, frozen.Lifecycle.Production, frozen.Lifecycle.Stop} {
		for _, step := range phase.Steps {
			if len(step.Exec) > 0 || step.Condition != nil {
				return item, fmt.Errorf("retained runtime does not support executable or conditional lifecycle hooks: %s", step.Name)
			}
		}
	}
	interpreters := map[string]string{}
	for _, v := range release.Interpreters {
		interpreters[v.Command] = v.Path
	}
	for name, c := range item.Manifest.Components {
		if len(c.Run.DataDirs) != 0 || strings.TrimSpace(c.Run.LogDir) != "" {
			return item, fmt.Errorf("component %s declares relative mutable directories; qualify canonical external state before retaining this runtime", name)
		}
		if c.Run.Condition != nil {
			return item, fmt.Errorf("component %s has unsupported conditional runtime code", name)
		}
		spec, ok := builderRegistry[strings.TrimSpace(c.Build.Kind)]
		if !ok || spec.Reserved {
			return item, fmt.Errorf("component %s has unsupported runtime artifact builder", name)
		}
		targets, e := componentBuildTargets(name, item.SourcePath(), item.Slug, c, spec, runtimeGOOS())
		if e != nil {
			return item, e
		}
		for _, target := range targets {
			if e := retainedPathInside(item.SourcePath(), target.Output); e != nil {
				return item, fmt.Errorf("component %s output: %w", name, e)
			}
			if _, e = os.Stat(target.Output); e != nil {
				return item, fmt.Errorf("component %s runtime artifact absent: %w", name, e)
			}
		}
		if isAssetComponent(c) {
			continue
		}
		cwd, e := componentWorkingDir(item.SourcePath(), c.Run.CWD)
		if e != nil {
			return item, e
		}
		if fi, e := os.Stat(cwd); e != nil || !fi.IsDir() {
			return item, fmt.Errorf("component %s runtime working directory is absent", name)
		}
		if e := retainedPathInside(item.SourcePath(), cwd); e != nil {
			return item, fmt.Errorf("component %s cwd: %w", name, e)
		}
		if len(c.Run.Argv) == 0 {
			return item, fmt.Errorf("component %s has no runtime command", name)
		}
		command := c.Run.Argv[0]
		if command == "{{bin."+name+"}}" {
			if len(targets) == 0 {
				return item, fmt.Errorf("component %s has no retained executable", name)
			}
			if e := validateRetainedNativeExecutable(targets[0].Output); e != nil {
				return item, fmt.Errorf("component %s: %w", name, e)
			}
			if info, e := os.Stat(targets[0].Output); e != nil || !recognizeArtifact(targets[0].Output, info).Runnable {
				return item, fmt.Errorf("component %s retained executable is not runnable", name)
			}
		} else if pinned, ok := interpreters[command]; ok {
			if e := validateRetainedNativeExecutable(pinned); e != nil {
				return item, fmt.Errorf("component %s interpreter: %w", name, e)
			}
			if command != "node" {
				return item, fmt.Errorf("component %s interpreter %s has no qualified runtime packaging contract", name, command)
			}
			for _, key := range []string{"NODE_OPTIONS", "NODE_PATH"} {
				if os.Getenv(key) != "" || frozen.Environment[key] != "" || c.Run.Env[key] != "" {
					return item, fmt.Errorf("component %s cannot use external Node loader override %s", name, key)
				}
			}
			c.Run.Argv = append([]string(nil), c.Run.Argv...)
			c.Run.Argv[0] = pinned
			if len(c.Run.Argv) < 2 || strings.HasPrefix(c.Run.Argv[1], "-") {
				return item, fmt.Errorf("component %s interpreter requires a retained entry file, not inline code", name)
			}
			if filepath.IsAbs(c.Run.Argv[1]) || strings.ContainsAny(c.Run.Argv[1], "$\x00") || strings.Contains(c.Run.Argv[1], "{{") {
				return item, fmt.Errorf("component %s interpreter entry must be a literal retained relative path", name)
			}
			script := filepath.Join(cwd, c.Run.Argv[1])
			relative, e := filepath.Rel(item.SourcePath(), script)
			if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return item, fmt.Errorf("component %s interpreter entry escapes retained runtime", name)
			}
			if info, e := os.Stat(script); e != nil || !info.Mode().IsRegular() {
				return item, fmt.Errorf("component %s interpreter entry is missing", name)
			}
			if e := retainedPathInside(item.SourcePath(), script); e != nil {
				return item, e
			}
			if e := validateRetainedNodeRuntime(item.SourcePath(), cwd, script); e != nil {
				return item, fmt.Errorf("component %s Node closure: %w", name, e)
			}
			c.Run.Argv = append([]string{pinned, "--no-global-search-paths", script}, c.Run.Argv[2:]...)
		} else {
			return item, fmt.Errorf("component %s runtime command is not a retained binary or pinned interpreter", name)
		}
		item.Manifest.Components[name] = c
	}
	item.RetainedReleaseID = release.ID
	item.RetainedBuildIdentity = "sha256:" + release.ID
	return item, nil
}

// retainedPathInside checks physical paths, not only lexical declarations. A
// supported component never resolves an output, entry file or cwd outside the
// exact tree whose content was verified.
func retainedPathInside(root, path string) error {
	r, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(r, p)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("runtime path escapes retained tree: %s", path)
	}
	return nil
}
