package installgateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// A governed JavaScript install (pnpm or npm) is two steps that each have one
// job.
//
//  1. Resolve, in a throwaway copy of the surface's dependency inputs, with the
//     manager's lock-only mode. The package manager itself is the resolver, so
//     overrides, release-age policy, peers, optional packages and platform
//     filters are exactly what the real install would apply. Nothing in the
//     surface is touched, so a dry run can do this and so can the
//     disk-admission planner.
//  2. Publish the resolved manifest and lockfile and link node_modules from the
//     frozen lock. Any failure restores the previous inputs and relinks.
//
// Admission measures the locked packages that node_modules does not already
// hold. Apply publishes the very bytes that admission measured.

const (
	jsResolveTimeout = 3 * time.Minute
	jsLinkTimeout    = 15 * time.Minute
	maxJSNpmrcBytes  = 64 << 10
)

// stagedManager holds what differs between package managers; everything else in a
// staged install is shared.
type stagedManager struct {
	name string
	// inputs are the files a governed install may change; package.json is first.
	inputs []string
	limits map[string]int64
	// resolveArgs and linkArgs derive the lock-only and linking commands from the
	// resolved `add`/`install` argv.
	resolveArgs func(argv []string) []string
	linkArgs    func(argv []string) []string
	// environment is the one environment for every governed process of this
	// manager. cacheDir is where the stage isolates registry metadata.
	environment func(r Resolution, s *JSStage, extra ...string) []string
	cacheEnv    string
	// prepare runs before anything is staged and may refuse (store checks).
	prepare func(ctx context.Context, r Resolution, s *JSStage) error
	// seedStage adjusts the copied inputs before resolving (release-age floor).
	seedStage func(r Resolution, dir string) error
	// repair may fix a lockfile entry the manager refuses and reports whether the
	// resolve should be retried. It works only inside the staging copy.
	repair func(ctx context.Context, rc repairContext, output string) (bool, string)
	// explain turns a failed resolve into an actionable refusal.
	explain func(r Resolution, runErr error, output string) error
	// pin completes the resolved inputs before they are measured and published
	// (for example recording the integrity of a tarball URL dependency).
	pin func(ctx context.Context, s *JSStage, r Resolution) error
	// measure sizes what the staged lock adds.
	measure func(ctx context.Context, s *JSStage, r Resolution) error
}

// maxStageRepairs bounds how many refused lock entries one resolution repairs.
const maxStageRepairs = 8

// repairContext is what a repair may use: the staging directory, the manager's
// environment there, and the request's argv (for workspace flags).
type repairContext struct {
	dir  string
	env  []string
	argv []string
}

func managerFor(r Resolution) (*stagedManager, error) {
	switch r.PackageManager {
	case "pnpm":
		return pnpmManager, nil
	case "npm":
		return npmManager, nil
	}
	return nil, fmt.Errorf("staged resolution supports pnpm and npm, not %q", r.PackageManager)
}

// JSStage is one resolved, not yet published, JavaScript dependency change.
type JSStage struct {
	manager     *stagedManager
	surfaceRoot string
	dir         string
	store       pnpmStoreInfo
	pnpmPatches []stagedPnpmPatch
	patchBytes  int64
	// base holds each input exactly as read when the stage was created; nil
	// means the file did not exist. publish refuses to run when any of them
	// changed since, so an apply never overwrites concurrent work.
	base map[string][]byte
	// next holds the resolved content of the inputs that differ from base.
	next map[string][]byte
	// Measured describes what admission sized.
	Measured JSMeasurement
	// tarballSizes holds tarball URL downloads already made while pinning, so
	// measuring does not download them again.
	tarballSizes map[string]artifactSize
	// Notes are reviewer-facing facts about the resolution (for example the
	// integrity recorded for a tarball URL dependency).
	Notes []string
	// LinkSkipped, when set, is why apply publishes the manifest and lockfile
	// but does not touch node_modules. Nothing is sized for installation.
	LinkSkipped string
}

// Release removes the staging directory. It is safe to call more than once.
func (s *JSStage) Release() {
	if s == nil || s.dir == "" {
		return
	}
	_ = os.RemoveAll(s.dir)
	s.dir = ""
}

// packageManagerRunner executes a package manager; tests replace it with a fake.
type packageManagerRunner func(ctx context.Context, name, dir string, env []string, args ...string) ([]byte, error)

var runPackageManager packageManagerRunner = func(ctx context.Context, name, dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	return cmd.CombinedOutput()
}

// stageJSResolution resolves the requested install in a copy of the inputs and
// measures what applying it would add. The caller must Release the stage.
func stageJSResolution(ctx context.Context, r Resolution) (*JSStage, error) {
	manager, err := managerFor(r)
	if err != nil {
		return nil, err
	}
	if len(r.Argv) < 2 || r.Argv[0] != manager.name {
		return nil, fmt.Errorf("%s staging requires a %s install resolution", manager.name, manager.name)
	}
	if r.HostTool || r.HostToolRefresh || r.HostSource {
		return nil, fmt.Errorf("%s staging does not apply to host tools", manager.name)
	}
	rootInfo, err := os.Lstat(r.SurfaceRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s surface root is unavailable or linked", manager.name)
	}
	if err := validateNpmRegistryFiles(r.SurfaceRoot); err != nil {
		return nil, err
	}
	stage := &JSStage{manager: manager, surfaceRoot: r.SurfaceRoot, base: map[string][]byte{}, next: map[string][]byte{}}
	success := false
	defer func() {
		if !success {
			stage.Release()
		}
	}()
	if manager.prepare != nil {
		if err := manager.prepare(ctx, r, stage); err != nil {
			return nil, err
		}
	}
	if r.LockOnly && stage.LinkSkipped == "" {
		stage.LinkSkipped = "a lock-only update was requested"
	}
	for _, name := range manager.inputs {
		data, err := readOptionalBounded(filepath.Join(r.SurfaceRoot, name), manager.limits[name])
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		if data == nil && name == "package.json" {
			return nil, fmt.Errorf("%s surface has no package.json", manager.name)
		}
		stage.base[name] = data
	}
	if manager.name == "pnpm" {
		patches, total, err := readPnpmPatchInputs(r.SurfaceRoot, stage.base["package.json"])
		if err != nil {
			return nil, err
		}
		stage.pnpmPatches = patches
		stage.patchBytes = total
	}
	if err := stage.createDirectory(r); err != nil {
		return nil, err
	}
	for name, data := range stage.base {
		if data == nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(stage.dir, name), data, 0o600); err != nil {
			return nil, fmt.Errorf("stage %s: %w", name, err)
		}
	}
	if err := stagePnpmPatchFiles(stage.dir, stage.pnpmPatches); err != nil {
		return nil, err
	}
	if npmrc, err := readOptionalBounded(filepath.Join(r.SurfaceRoot, ".npmrc"), maxJSNpmrcBytes); err != nil {
		return nil, fmt.Errorf("read .npmrc: %w", err)
	} else if npmrc != nil {
		if err := os.WriteFile(filepath.Join(stage.dir, ".npmrc"), npmrc, 0o600); err != nil {
			return nil, fmt.Errorf("stage .npmrc: %w", err)
		}
	}
	if manager.seedStage != nil {
		if err := manager.seedStage(r, stage.dir); err != nil {
			return nil, err
		}
	}
	if r.NpmOverride != nil {
		if err := SetNpmOverride(filepath.Join(stage.dir, "package.json"), r.NpmOverride.Selector, r.NpmOverride.Version); err != nil {
			return nil, fmt.Errorf("stage the override: %w", err)
		}
	}

	env := manager.environment(r, stage, manager.cacheEnv+"="+filepath.Join(stage.dir, ".sda-metadata-cache"))
	runCtx, cancel := context.WithTimeout(ctx, jsResolveTimeout)
	defer cancel()
	for _, argv := range append([][]string{r.Argv}, r.ThenArgv...) {
		args := manager.resolveArgs(argv)
		out, runErr := runPackageManager(runCtx, manager.name, stage.dir, env, args...)
		for repairs := 0; runErr != nil && manager.repair != nil && repairs < maxStageRepairs; repairs++ {
			repaired, note := manager.repair(runCtx, repairContext{dir: stage.dir, env: env, argv: argv}, string(out))
			if !repaired {
				break
			}
			stage.Notes = append(stage.Notes, note)
			out, runErr = runPackageManager(runCtx, manager.name, stage.dir, env, args...)
		}
		if runErr != nil {
			return nil, manager.explain(r, runErr, string(out))
		}
	}
	for _, name := range manager.inputs {
		data, err := readOptionalBounded(filepath.Join(stage.dir, name), manager.limits[name])
		if err != nil {
			return nil, fmt.Errorf("read resolved %s: %w", name, err)
		}
		if !bytes.Equal(data, stage.base[name]) {
			if data == nil {
				return nil, fmt.Errorf("%s removed %s while resolving", manager.name, name)
			}
			stage.next[name] = data
		}
	}
	if manager.pin != nil {
		if err := manager.pin(ctx, stage, r); err != nil {
			return nil, err
		}
	}
	if stage.LinkSkipped != "" {
		if err := stage.finishMeasurement(0, 0); err != nil {
			return nil, err
		}
		stage.Notes = append(stage.Notes, "node_modules will not be relinked: "+stage.LinkSkipped+"; apply updates package.json and the lockfile only")
	} else if err := manager.measure(ctx, stage, r); err != nil {
		return nil, err
	}
	success = true
	return stage, nil
}

// createDirectory makes the throwaway project directory. Repository surfaces
// stage next to the real surface, at the same depth, so a relative `file:`
// dependency such as ../../../packages/x resolves exactly as it will in place.
func (s *JSStage) createDirectory(r Resolution) error {
	parent := filepath.Dir(r.SurfaceRoot)
	if r.ExternalWorkspace {
		parent = os.TempDir()
	}
	sweepStaleStages(parent, time.Now())
	dir, err := os.MkdirTemp(parent, stageDirPrefix)
	if err != nil {
		return fmt.Errorf("create staging directory beside %s: %w", r.SurfaceRoot, err)
	}
	s.dir = dir
	return nil
}

const (
	stageDirPrefix = ".sda-stage-"
	// A stage lives for seconds to minutes. One that is hours old belongs to a
	// process that died before it could clean up.
	staleStageAge = time.Hour
)

var stageDirName = regexp.MustCompile(`^\.sda-stage-[0-9]+$`)

// sweepStaleStages removes abandoned staging directories next to a surface, so
// a crash or kill never leaves resolver metadata behind for good.
func sweepStaleStages(parent string, now time.Time) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !stageDirName.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(parent, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || now.Sub(info.ModTime()) < staleStageAge {
			continue
		}
		_ = os.RemoveAll(path)
	}
}

// publish writes the resolved inputs, links node_modules from the frozen
// lock, and restores the previous inputs if anything fails. Once the inputs are
// restored it relinks node_modules to the previous lock so a failed apply
// leaves the surface as it was found.
func (s *JSStage) publish(ctx context.Context, r Resolution) (string, error) {
	manager := s.manager
	if len(s.next) == 0 {
		return manager.name + " resolved no change to the dependency inputs; nothing was published.", nil
	}
	if err := s.verifyPnpmPatchInputsUnchanged(); err != nil {
		return "", err
	}
	for name, base := range s.base {
		current, err := readOptionalBounded(filepath.Join(s.surfaceRoot, name), manager.limits[name])
		if err != nil {
			return "", fmt.Errorf("re-read %s before publishing: %w", name, err)
		}
		if !bytes.Equal(current, base) {
			return "", fmt.Errorf("%s changed after the install was admitted; nothing was published, retry the request", name)
		}
	}
	var published []string
	rollback := func() error {
		var errs []error
		for _, name := range published {
			path := filepath.Join(s.surfaceRoot, name)
			if s.base[name] == nil {
				errs = append(errs, os.Remove(path))
			} else {
				errs = append(errs, writeFileAtomic(path, s.base[name]))
			}
		}
		return errors.Join(errs...)
	}
	for _, name := range manager.inputs {
		data, changed := s.next[name]
		if !changed {
			continue
		}
		published = append(published, name)
		if err := writeFileAtomic(filepath.Join(s.surfaceRoot, name), data); err != nil {
			return "", errors.Join(fmt.Errorf("publish %s: %w", name, err), rollback())
		}
	}

	if s.LinkSkipped != "" {
		return manager.name + " inputs published; node_modules was not relinked (" + s.LinkSkipped + ").", nil
	}
	env := manager.environment(r, s)
	link := func() ([]byte, error) {
		runCtx, cancel := context.WithTimeout(ctx, jsLinkTimeout)
		defer cancel()
		return runPackageManager(runCtx, manager.name, s.surfaceRoot, env, manager.linkArgs(r.Argv)...)
	}
	out, err := link()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	restoreErr := rollback()
	var note string
	if restoreErr == nil {
		relinkOut, relinkErr := link()
		if relinkErr != nil {
			note = fmt.Sprintf(" The previous inputs were restored, but relinking node_modules failed (%v); run `%s` in %s. %s", relinkErr, strings.Join(append([]string{manager.name}, manager.linkArgs(r.Argv)...), " "), s.surfaceRoot, strings.TrimSpace(string(relinkOut)))
		} else {
			note = " The previous package.json, lockfile and node_modules were restored."
		}
	} else {
		note = fmt.Sprintf(" Restoring the previous inputs failed (%v); inspect %s before retrying.", restoreErr, s.surfaceRoot)
	}
	return string(out), fmt.Errorf("%s linking failed: %w.%s", manager.name, err, note)
}

func readOptionalBounded(path string, limit int64) ([]byte, error) {
	data, err := readRegularBounded(path, limit)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// writeFileAtomic replaces path through a same-directory temporary file so a
// crash never leaves a half-written manifest or lockfile.
func writeFileAtomic(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		mode = info.Mode().Perm()
	}
	var nonce [6]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	tmp := path + ".sda-" + hex.EncodeToString(nonce[:])
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// requestedSpecText names the request in messages: the package@spec argument.
func requestedSpecText(r Resolution) string {
	if len(r.RemovedPackages) > 0 {
		return "the removal of " + r.PackageName
	}
	if len(r.UpdatedPackages) > 0 {
		return "the update of " + r.PackageName
	}
	for i := len(r.Argv) - 1; i >= 0; i-- {
		if strings.HasPrefix(r.Argv[i], r.PackageName+"@") {
			return r.Argv[i]
		}
	}
	return r.PackageName
}

// requestedSpec returns the specifier the request asked for, the text after
// "<package>@" in the package argument. Flags such as -D, --save-exact and
// --ignore-workspace may follow that argument, so it is found by name, not by
// position.
func requestedSpec(r Resolution) string {
	for i := len(r.Argv) - 1; i >= 0; i-- {
		if spec, ok := strings.CutPrefix(r.Argv[i], r.PackageName+"@"); ok {
			return spec
		}
	}
	return ""
}
