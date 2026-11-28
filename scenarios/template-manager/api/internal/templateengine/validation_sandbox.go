package templateengine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"

	"github.com/vrooli/vrooli/packages/proto/protogen"
)

// validationSandbox is a throwaway repository root that shallow template
// validation relocates, regenerates, and lints in.
//
// Relocations declare in-repository targets (a template's proto schemas go to
// packages/proto/schemas/<id>) and post hooks that regenerate shared output.
// Run against the real checkout, validation published a throwaway schema owner
// into the shared generated tree and the shared Proto artifact store; removing
// the gen directories afterwards neither rebuilt the descriptor nor replaced
// the active artifact, so every later compatibility install restored the
// residue. In the sandbox the relocation targets, the generated tree, the
// staging trees and the artifact store are all private copies, and removing
// the sandbox is the whole cleanup.
//
// Only packages/proto is copied: it is the one tree relocations and their
// hooks write to. Everything else the contract or the hooks resolve (other
// shared packages, internal tool pins, templates) is linked read-only from
// the real checkout. The one place sandbox output must be visible in the
// checkout is Go module convergence; see lendGeneratedGo.
type validationSandbox struct {
	Root string
	// Env is the subprocess environment for relocation hooks: the caller's
	// environment, pointed at the sandbox's private Proto artifact store.
	Env []string
}

// linkedRepositoryEntries are the repository-root entries the sandbox links
// rather than copies: the repo contract's required markers that relocations
// never write to.
var linkedRepositoryEntries = []string{"templates", "resources", "cmd", "internal", "go.mod"}

func newValidationSandbox(repoRoot, root string, baseEnv []string) (*validationSandbox, error) {
	repoRoot = filepath.Clean(repoRoot)
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios"), 0o755); err != nil {
		return nil, err
	}
	if err := copyFileIfExists(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		return nil, err
	}
	if err := linkIfExists(filepath.Join(repoRoot, ".vrooli", "schemas"), filepath.Join(root, ".vrooli", "schemas")); err != nil {
		return nil, err
	}
	for _, name := range linkedRepositoryEntries {
		if err := linkIfExists(filepath.Join(repoRoot, name), filepath.Join(root, name)); err != nil {
			return nil, err
		}
	}
	if err := populateSandboxPackages(filepath.Join(repoRoot, "packages"), filepath.Join(root, "packages")); err != nil {
		return nil, fmt.Errorf("prepare sandbox packages: %w", err)
	}
	env, err := templateHookEnv(baseEnv, map[string]string{
		protogen.ArtifactRootEnv: filepath.Join(root, ".vrooli", "artifacts", "proto"),
	})
	if err != nil {
		return nil, err
	}
	return &validationSandbox{Root: root, Env: env}, nil
}

// populateSandboxPackages links every shared package except proto, which is
// copied so generation can publish into it.
func populateSandboxPackages(source, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if isProtoWorkTree(name) {
			continue
		}
		if name == "proto" {
			if err := copySandboxTree(filepath.Join(source, name), filepath.Join(destination, name)); err != nil {
				return fmt.Errorf("copy packages/proto: %w", err)
			}
			continue
		}
		if err := os.Symlink(filepath.Join(source, name), filepath.Join(destination, name)); err != nil {
			return err
		}
	}
	return nil
}

// isProtoWorkTree reports transient protogen work trees (staging trees and the
// compatibility view's backup), which are never part of a usable proto copy.
func isProtoWorkTree(name string) bool {
	return strings.HasPrefix(name, ".proto-") || name == "gen.previous"
}

// copySandboxTree copies a directory tree, preserving file modes and
// symlinks, and skipping protogen work trees at any depth.
func copySandboxTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if rel != "." && isProtoWorkTree(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case entry.IsDir():
			return os.MkdirAll(target, 0o755)
		case entry.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case entry.Type().IsRegular():
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return copyFile(path, target, info.Mode().Perm())
		default:
			return nil
		}
	})
}

func copyFileIfExists(source, destination string) error {
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return copyFile(source, destination, info.Mode().Perm())
}

func copyFile(source, destination string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func linkIfExists(source, destination string) error {
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return os.Symlink(source, destination)
}

// removeValidationTree deletes a validation temp tree. The sandbox's Proto
// artifact store makes its snapshots read-only, so directories are made
// writable before removal; links are removed, never followed.
func removeValidationTree(root string) error {
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			_ = os.Chmod(path, 0o755)
		}
		return nil
	})
	return os.RemoveAll(root)
}

// protoGenerationLock is the shared Proto generation lock; tests replace it.
var protoGenerationLock = func(ctx context.Context) (func(), error) {
	return protogen.LockGeneration(ctx, io.Discard)
}

// protoGenerationLockWait matches lifecycle's bound on waiting for the same lock.
const protoGenerationLockWait = 10 * time.Minute

// lendGeneratedGo makes the sandbox's generated Go for owners visible in the
// checkout's generated tree while check runs, and removes it again before the
// Proto generation lock is released.
//
// Module convergence (scenario-dependency-analyzer `deps reconcile`) resolves
// every in-repo module against the canonical checkout, so a generated module
// that imports its own relocated proto package converges only if the checkout's
// packages/proto module contains that package. Lending just the Go output, only
// while the lock is held, keeps every lock-respecting reader (lifecycle
// compatibility installs, protogen runs) from observing it. The artifact store,
// descriptor and manifests are never touched, and a crash mid-check leaves only
// an orphan directory that the next full generation or compatibility install
// prunes.
func (s *validationSandbox) lendGeneratedGo(ctx context.Context, repoRoot string, owners []string, check func() []templatecontracts.TemplateValidationIssue) ([]templatecontracts.TemplateValidationIssue, error) {
	var lendable []string
	for _, owner := range owners {
		if _, err := os.Stat(filepath.Join(s.Root, "packages", "proto", "gen", "go", owner)); err == nil {
			lendable = append(lendable, owner)
		}
	}
	if len(lendable) == 0 {
		return check(), nil
	}
	ctx, cancel := context.WithTimeout(ctx, protoGenerationLockWait)
	defer cancel()
	release, err := protoGenerationLock(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Proto generation lock: %w", err)
	}
	defer release()
	var lent []string
	defer func() {
		for _, path := range lent {
			_ = os.RemoveAll(path)
		}
	}()
	checkoutProto := filepath.Join(repoRoot, "packages", "proto")
	for _, owner := range lendable {
		if _, err := os.Stat(filepath.Join(checkoutProto, "schemas", owner)); err == nil {
			return nil, fmt.Errorf("validation scenario id %q collides with a schema owner in the checkout", owner)
		}
		target := filepath.Join(checkoutProto, "gen", "go", owner)
		if _, err := os.Lstat(target); err == nil {
			return nil, fmt.Errorf("checkout already holds generated output %s with no schema owner; reap it with `template-manager lifecycle destroy %s --proto-only`", target, owner)
		}
		lent = append(lent, target)
		if err := copySandboxTree(filepath.Join(s.Root, "packages", "proto", "gen", "go", owner), target); err != nil {
			return nil, fmt.Errorf("lend generated Go for %s: %w", owner, err)
		}
	}
	return check(), nil
}

// relocatedProtoOwners lists the schema owners relocations placed under the
// sandbox's packages/proto/schemas.
func relocatedProtoOwners(sandboxRoot string, relocations []templatecontracts.ResolvedRelocation) []string {
	schemas := filepath.Join(sandboxRoot, "packages", "proto", "schemas")
	var owners []string
	for _, relocation := range relocations {
		rel, err := filepath.Rel(schemas, filepath.Clean(relocation.To))
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		owners = append(owners, strings.SplitN(rel, string(filepath.Separator), 2)[0])
	}
	return owners
}

// validationContext recovers a context.Context from a handler context type.
func validationContext[C any](ctx C) context.Context {
	if typed, ok := any(ctx).(context.Context); ok && typed != nil {
		return typed
	}
	return context.Background()
}
