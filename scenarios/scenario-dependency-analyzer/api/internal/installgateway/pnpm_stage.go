package installgateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
	"gopkg.in/yaml.v3"
)

const (
	maxPnpmManifestBytes  = 1 << 20
	maxPnpmLockBytes      = 16 << 20
	maxPnpmWorkspaceBytes = 64 << 10
)

// pnpmManager is the pnpm side of a staged install. Resolution is
// `pnpm add|remove --lockfile-only`; linking is `pnpm install --frozen-lockfile`,
// which cannot re-resolve or rewrite the lock.
var pnpmManager = &stagedManager{
	name:   "pnpm",
	inputs: []string{"package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml"},
	limits: map[string]int64{"package.json": maxPnpmManifestBytes, "pnpm-lock.yaml": maxPnpmLockBytes, "pnpm-workspace.yaml": maxPnpmWorkspaceBytes},
	resolveArgs: func(argv []string) []string {
		return append([]string{argv[1], "--lockfile-only"}, argv[2:]...)
	},
	linkArgs: func(argv []string) []string {
		args := []string{"install", "--frozen-lockfile", "--ignore-scripts"}
		for _, flag := range []string{"--ignore-workspace", "--ignore-pnpmfile"} {
			if containsArg(argv, flag) {
				args = append(args, flag)
			}
		}
		return args
	},
	environment: func(r Resolution, s *JSStage, extra ...string) []string {
		return pnpmEnvironment(r, s.store.Base, extra...)
	},
	cacheEnv: "npm_config_cache_dir",
	prepare: func(ctx context.Context, r Resolution, s *JSStage) error {
		store, err := effectivePnpmStore(ctx, r.SurfaceRoot, "")
		if err != nil {
			return err
		}
		s.store = store
		mismatch, recorded, err := pnpmModulesStoreMismatch(r.SurfaceRoot, store)
		if err != nil || mismatch == "" {
			return err
		}
		// node_modules belongs to the store it was linked against, so follow it
		// when pnpm can select that store: nothing is recreated and the link can
		// run. Only a store pnpm cannot select (an older pnpm major's layout)
		// leaves node_modules to the lifecycle; the manifest and lockfile are the
		// governed artifacts and are still updated.
		if followed, ok := followRecordedStore(ctx, r.SurfaceRoot, recorded); ok {
			s.store = followed
			s.Notes = append(s.Notes, fmt.Sprintf("node_modules is linked against pnpm store %s, not the one pnpm now selects (%s); the install uses the recorded store so node_modules is not recreated", followed.Versioned, store.Versioned))
			return nil
		}
		s.LinkSkipped = mismatch
		return nil
	},
	seedStage: func(r Resolution, dir string) error {
		if r.ExternalWorkspace {
			return nil
		}
		return ensureReleaseAge(dir)
	},
	repair:  repairPnpmLock,
	pin:     pinPnpmTarballIntegrity,
	explain: explainPnpmResolveFailure,
	measure: measurePnpmStage,
}

// pnpmEnvironment is the single environment for every governed pnpm process:
// the store is the one the scenario lifecycle uses, the registry is the public
// one the size probe understands, lifecycle prompts are off, and the toolchain
// width floor matches the lifecycle's.
func pnpmEnvironment(r Resolution, storeBase string, extra ...string) []string {
	overlay := managerCacheEnvironment("pnpm", storeBase, "")
	// CI must not be true: in CI pnpm recreates node_modules without asking
	// whenever its layout settings differ, which destroys a surface. Without it,
	// and without a terminal, pnpm aborts before removing anything.
	overlay = append(overlay, "CI=false")
	if r.ExternalWorkspace {
		// Retain the shared pnpm release-age policy without creating workspace
		// membership/configuration that a later operation could execute.
		overlay = append(overlay, "npm_config_minimum_release_age=10080", "npm_config_ignore_pnpmfile=true", "npm_config_manage_package_manager_versions=false")
	}
	overlay = append(overlay, extra...)
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	return overlayInstallEnvironment(envkit.Toolchain(env, envkit.ToolchainOptions{}), overlay)
}

type pnpmStoreInfo struct {
	// Base is the store-dir value pnpm appends its store version to.
	Base string
	// Versioned is the directory node_modules is linked against.
	Versioned string
}

var pnpmStoreVersionDir = regexp.MustCompile(`^v[0-9]+$`)

// effectivePnpmStore asks pnpm which store it would use in dir. The answer
// honours the user's pnpm rc, the project's .npmrc and any environment, so the
// governed install links against the same store as the scenario lifecycle.
func effectivePnpmStore(ctx context.Context, dir, baseOverride string) (pnpmStoreInfo, error) {
	env := envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	if baseOverride != "" {
		env = append(env, "npm_config_store_dir="+baseOverride)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := runPackageManager(probeCtx, "pnpm", dir, env, "store", "path")
	if err != nil {
		return pnpmStoreInfo{}, fmt.Errorf("resolve the pnpm store path with `pnpm store path`: %w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	versioned := filepath.Clean(strings.TrimSpace(lines[len(lines)-1]))
	if !filepath.IsAbs(versioned) {
		return pnpmStoreInfo{}, fmt.Errorf("`pnpm store path` returned a non-absolute path %q", versioned)
	}
	base := versioned
	if pnpmStoreVersionDir.MatchString(filepath.Base(versioned)) {
		base = filepath.Dir(versioned)
	}
	return pnpmStoreInfo{Base: base, Versioned: versioned}, nil
}

// pnpmModulesStoreMismatch reports, when node_modules is linked against a
// different store than the one pnpm will use, why the install must not link.
// pnpm answers that situation by recreating node_modules, which is destructive
// and was the observed failure of the previous gateway (and is what an older
// pnpm major's layout looks like after the fleet moves to a newer one).
func pnpmModulesStoreMismatch(surfaceRoot string, store pnpmStoreInfo) (message, recorded string, err error) {
	data, err := readRegularBounded(filepath.Join(surfaceRoot, "node_modules", ".modules.yaml"), 8<<20)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("read node_modules/.modules.yaml: %w", err)
	}
	var modules struct {
		StoreDir string `yaml:"storeDir"`
	}
	if err := yaml.Unmarshal(data, &modules); err != nil {
		return "", "", fmt.Errorf("parse node_modules/.modules.yaml: %w", err)
	}
	if modules.StoreDir == "" || filepath.Clean(modules.StoreDir) == store.Versioned {
		return "", "", nil
	}
	return fmt.Sprintf("node_modules is linked against pnpm store %s but pnpm resolves %s, and pnpm would recreate node_modules to switch", modules.StoreDir, store.Versioned), filepath.Clean(modules.StoreDir), nil
}

// followRecordedStore reports whether pnpm can be made to select the store
// node_modules was linked against, by naming its base directory.
func followRecordedStore(ctx context.Context, dir, recorded string) (pnpmStoreInfo, bool) {
	if !pnpmStoreVersionDir.MatchString(filepath.Base(recorded)) {
		return pnpmStoreInfo{}, false
	}
	followed, err := effectivePnpmStore(ctx, dir, filepath.Dir(recorded))
	if err != nil || followed.Versioned != recorded {
		return pnpmStoreInfo{}, false
	}
	return followed, true
}

var missingTarballIntegrity = regexp.MustCompile(`ERR_PNPM_MISSING_TARBALL_INTEGRITY[\s\p{Zs}]+Cannot install package "([^"]+)"`)

// repairMissingTarballIntegrity handles a lock entry for a tarball URL that an
// older pnpm wrote without an integrity field and the current pnpm refuses to
// install. Removing exactly the named entries lets pnpm re-resolve them from
// the manifest (pnpm may still record no integrity, which pinPnpmTarballIntegrity
// then supplies from the downloaded bytes).
func repairMissingTarballIntegrity(dir, output string) (bool, string) {
	match := missingTarballIntegrity.FindStringSubmatch(output)
	if match == nil {
		return false, ""
	}
	key := match[1]
	lockPath := filepath.Join(dir, "pnpm-lock.yaml")
	lock, err := readOptionalBounded(lockPath, maxPnpmLockBytes)
	if err != nil || lock == nil {
		return false, ""
	}
	_, spec, ok := splitPnpmPackageKey(key)
	if !ok || !strings.HasPrefix(spec, "https://") {
		return false, ""
	}
	repaired, removed, err := removePnpmLockEntries(lock, key, spec)
	if err != nil || !removed {
		return false, ""
	}
	if err := os.WriteFile(lockPath, repaired, 0o600); err != nil {
		return false, ""
	}
	return true, fmt.Sprintf("lockfile entry %s had no integrity and pnpm refuses it; it was re-resolved from the manifest in the staging copy", key)
}

// repairPnpmLock applies the repair that matches pnpm's refusal, if any.
func repairPnpmLock(ctx context.Context, rc repairContext, output string) (bool, string) {
	if repaired, note := repairMissingTarballIntegrity(rc.dir, output); repaired {
		return true, note
	}
	return repairImmatureLockedVersion(ctx, rc, output)
}

var immatureVersion = regexp.MustCompile(`ERR_PNPM_NO_MATURE_MATCHING_VERSION[\s\p{Zs}]+Version ([^\s]+) \(released ([^)]*)\) of ([^\s]+) does not meet`)

// repairImmatureLockedVersion handles a lockfile that already holds a version
// younger than the release-age policy (written when the policy was not applied,
// for example by a raw pnpm run). The policy is not relaxed: pnpm itself
// re-resolves the package to the newest version that satisfies it, in the
// staging copy, and the preview shows the lockfile change. A request for a
// package that is too new matches nothing in the lock and stays refused.
func repairImmatureLockedVersion(ctx context.Context, rc repairContext, output string) (bool, string) {
	match := immatureVersion.FindStringSubmatch(output)
	if match == nil {
		return false, ""
	}
	version, released, name := match[1], match[2], match[3]
	lock, err := readOptionalBounded(filepath.Join(rc.dir, "pnpm-lock.yaml"), maxPnpmLockBytes)
	if err != nil || lock == nil {
		return false, ""
	}
	packages, err := readPnpmLockPackages(lock)
	if err != nil {
		return false, ""
	}
	if _, locked := packages[name+"@"+version]; !locked {
		return false, ""
	}
	args := []string{"update", name, "--lockfile-only"}
	for _, flag := range []string{"--workspace-root", "--ignore-workspace"} {
		if containsArg(rc.argv, flag) {
			args = append(args, flag)
		}
	}
	if _, err := runPackageManager(ctx, "pnpm", rc.dir, rc.env, args...); err != nil {
		return false, ""
	}
	return true, fmt.Sprintf("the lockfile held %s@%s, released %s, which is younger than the release-age policy; pnpm re-resolved %s to a version that satisfies the policy", name, version, released, name)
}

func removePnpmLockEntries(lock []byte, key, spec string) ([]byte, bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(lock, &doc); err != nil {
		return nil, false, err
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("pnpm lockfile is not a mapping")
	}
	removed := false
	deleteKey := func(mapping *yaml.Node, name string) {
		for i := 0; i+1 < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == name {
				mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
				removed = true
				return
			}
		}
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		section, value := root.Content[i].Value, root.Content[i+1]
		switch section {
		case "packages", "snapshots":
			if value.Kind == yaml.MappingNode {
				deleteKey(value, key)
			}
		case "importers":
			for j := 0; j+1 < len(value.Content); j += 2 {
				importer := value.Content[j+1]
				for k := 0; k+1 < len(importer.Content); k += 2 {
					deps := importer.Content[k+1]
					if deps.Kind != yaml.MappingNode {
						continue
					}
					for d := 0; d+1 < len(deps.Content); {
						entry := deps.Content[d+1]
						if entry.Kind == yaml.MappingNode && mappingScalar(entry, "version") == spec {
							deps.Content = append(deps.Content[:d], deps.Content[d+2:]...)
							removed = true
							continue
						}
						d += 2
					}
				}
			}
		}
	}
	if !removed {
		return nil, false, nil
	}
	encoded, err := encodePnpmLock(&doc)
	return encoded, removed, err
}

func mappingScalar(mapping *yaml.Node, name string) string {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == name {
			return mapping.Content[i+1].Value
		}
	}
	return ""
}

// explainPnpmResolveFailure turns pnpm's output into the refusal an agent can
// act on, and states that nothing was changed.
func explainPnpmResolveFailure(r Resolution, runErr error, output string) error {
	tail := strings.TrimSpace(output)
	if len(tail) > 2000 {
		tail = tail[len(tail)-2000:]
	}
	hint := ""
	switch {
	case strings.Contains(output, "ERR_PNPM_NO_MATURE_MATCHING_VERSION"), strings.Contains(output, "ERR_PNPM_NO_MATCHING_VERSION"):
		hint = " The package or version is not published, or is newer than the seven-day release-age policy; pick an older version."
	case strings.Contains(output, "ERR_PNPM_FETCH_404"):
		hint = " The registry has no such package or version. The package named in the output may be a dependency rather than the request; a shared @vrooli/* package the closure needs must already be declared on the surface as a file: dependency."
	case strings.Contains(output, "ERR_PNPM_MISSING_TARBALL_INTEGRITY"):
		hint = " A lockfile entry that pnpm refuses could not be re-resolved automatically; it must be repaired by the surface owner."
	case strings.Contains(output, "ERR_PNPM_OUTDATED_LOCKFILE"), strings.Contains(output, "ERR_PNPM_LOCKFILE_CONFIG_MISMATCH"):
		hint = " The surface's lockfile disagrees with its package.json; refresh it with the scenario lifecycle (`vrooli scenario setup <scenario>`) and retry."
	}
	return fmt.Errorf("pnpm could not resolve %s in a staging copy of %s (%v); the surface was not changed.%s\n%s", requestedSpecText(r), r.SurfaceRoot, runErr, hint, tail)
}
