package packagegov

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// A pnpm install fails or recreates node_modules when the store it would use is
// not the store node_modules was linked against: pnpm records the store in
// node_modules/.modules.yaml and treats any other store as a reason to purge.
// Non-interactively that is either an abort (ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY)
// or, under CI=true, a silent wipe. Which store is "the effective one" depends on
// the user's pnpm config, and tools that predate it (an earlier analyzer pinned
// store/v10 while the user config selects store/v3) leave node_modules behind
// on the other one. The store is an implementation detail of an existing
// node_modules, so a governed pnpm command follows it instead of switching.

var pnpmStoreVersionDir = regexp.MustCompile(`^v[0-9]+$`)

const pnpmStoreProbeTimeout = 30 * time.Second

// pnpmStoreRunner reports `<exe> store path` for a directory, with the same
// executable the guarded command will run; tests replace it.
var pnpmStoreRunner = func(ctx context.Context, exe, dir string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, exe, "store", "path")
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pnpm store path: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// pnpmTouchesNodeModules reports whether argv is a pnpm command that reads or
// rewrites node_modules and so can trigger a purge.
func pnpmTouchesNodeModules(argv []string) bool {
	if len(argv) < 2 || filepath.Base(argv[0]) != "pnpm" {
		return false
	}
	for _, arg := range argv[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		switch arg {
		case "install", "i", "add", "remove", "rm", "uninstall", "un", "update", "up", "upgrade":
			return true
		}
		return false
	}
	return false
}

// recordedPnpmStore returns the store directory node_modules was linked
// against, or "" when node_modules does not exist or records none.
func recordedPnpmStore(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "node_modules", ".modules.yaml"))
	if err != nil {
		return ""
	}
	// pnpm writes this file as YAML, or as JSON-style YAML with quoted keys.
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.Trim(strings.TrimSpace(key), `"'`) == "storeDir" {
			return strings.Trim(strings.TrimSpace(value), `"',`)
		}
	}
	return ""
}

// PnpmStoreMismatchError reports node_modules linked against a pnpm store
// layout that this pnpm cannot select, so reusing it is impossible and pnpm
// would recreate node_modules. It is a distinct type because the owner of the
// consumer, not this package, knows whether recreating it is safe: a stopped
// consumer's node_modules is not serving anything.
type PnpmStoreMismatchError struct {
	Dir       string // directory that contains node_modules
	Recorded  string
	Effective string
	Detail    string
}

func (e *PnpmStoreMismatchError) Error() string {
	return fmt.Sprintf("%s/node_modules was linked against pnpm store %s and pnpm now selects %s; %s, so it cannot be reused. The lifecycle preserved the existing tree because a runtime may be serving from it. Resolve this setup state through Vrooli's scenario lifecycle after the owning runtime is no longer serving", e.Dir, e.Recorded, e.Effective, e.Detail)
}

// GuardPnpmStore returns env adjusted so a pnpm command that touches
// node_modules in dir uses the store node_modules is already linked against. It
// changes nothing for any other command, for a directory without node_modules,
// and when the stores already agree. The note says what was adjusted, for the
// caller's log. It fails, naming both stores, when the recorded store cannot be
// selected, rather than letting pnpm purge node_modules.
func GuardPnpmStore(ctx context.Context, argv []string, dir string, env []string) (adjusted []string, note string, err error) {
	if !pnpmTouchesNodeModules(argv) {
		return env, "", nil
	}
	recorded := recordedPnpmStore(dir)
	if recorded == "" {
		return env, "", nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, pnpmStoreProbeTimeout)
	defer cancel()
	effective, err := pnpmStoreRunner(probeCtx, argv[0], dir, env)
	if err != nil {
		return env, "", err
	}
	recorded, effective = filepath.Clean(recorded), filepath.Clean(effective)
	if recorded == effective {
		return env, "", nil
	}
	base, version := filepath.Dir(recorded), filepath.Base(recorded)
	if !pnpmStoreVersionDir.MatchString(version) {
		return env, "", &PnpmStoreMismatchError{Dir: dir, Recorded: recorded, Effective: effective, Detail: "the recorded store is not a versioned store directory"}
	}
	adjusted = overlayEnv(env, "npm_config_store_dir", base)
	confirmed, err := pnpmStoreRunner(probeCtx, argv[0], dir, adjusted)
	if err != nil {
		return env, "", err
	}
	if filepath.Clean(confirmed) != recorded {
		return env, "", &PnpmStoreMismatchError{Dir: dir, Recorded: recorded, Effective: effective, Detail: fmt.Sprintf("selecting the recorded store (npm_config_store_dir=%s) gave %s instead", base, confirmed)}
	}
	return adjusted, fmt.Sprintf("%s/node_modules is linked against pnpm store %s (pnpm now selects %s); reusing it so pnpm does not recreate node_modules", dir, recorded, effective), nil
}

// overlayEnv returns env with key set, replacing any earlier value.
func overlayEnv(env []string, key, value string) []string {
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if name, _, _ := strings.Cut(item, "="); name != key {
			out = append(out, item)
		}
	}
	return append(out, key+"="+value)
}

const pnpmPurgeAbort = "ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY"

// ExplainPnpmPurgeAbort replaces pnpm's terse refusal to purge node_modules
// without a terminal with what happened and what to do. It returns err
// unchanged for any other failure.
func ExplainPnpmPurgeAbort(err error, stderr string, dir string) error {
	if err == nil || !strings.Contains(stderr, pnpmPurgeAbort) {
		return err
	}
	return fmt.Errorf("%w: pnpm needs to recreate %s/node_modules (its settings or store changed since it was installed); nothing was changed. The lifecycle must repair this tree when it can safely do so without disrupting a serving runtime", err, dir)
}
