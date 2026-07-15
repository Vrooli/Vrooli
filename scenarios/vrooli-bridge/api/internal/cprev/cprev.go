// Package cprev resolves the control plane's exact git commit and the revision an
// onboarding/provisioning op should pin to. Its reason for existing is a single
// failure mode (locked decision 11a): drift between the control plane and the
// fleet nodes it provisions. Every op therefore pins to the control plane's
// current commit by DEFAULT, and a loud preflight refuses to dispatch a commit
// the nodes could never fetch because it was never pushed.
//
// One Resolver owns four concerns so onboarding, provisioning, and fleet roll all
// behave identically:
//
//   - default: an empty requested revision becomes the control plane's commit;
//   - sentinel: the "@cp" token expands to the same commit, usable anywhere a
//     revision is accepted;
//   - validate: a ref carrying a shell metacharacter (e.g. a "HEAD~1" relative
//     ref) is rejected at the API boundary with a friendly error, BEFORE it can
//     reach the node agent's privsep filter and fail there as an opaque
//     rejection;
//   - preflight: the resolved commit must exist on the clone remote, or dispatch
//     fails with explicit "push first" guidance naming the commit and remote.
//
// The commit source is chosen deliberately: the PRIMARY source is a live
// `git -C <repo> rev-parse HEAD` of the control plane's own checkout, so the
// default always tracks what the operator has actually committed locally right
// now. An ldflags-embedded GitCommit (mirroring internal/setup/setup.go's
// build-commit pattern) is a FALLBACK for a deployed binary running outside a git
// checkout. git wins because a running control plane can have committed past its
// build.
//
// Every git call goes through the GitRunner seam so tests drive resolution and
// preflight with a fake instead of a real repository.
package cprev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SentinelControlPlane is the revision token that expands to the control plane's
// current commit. It is accepted anywhere a revision is (onboard target,
// provision target/rollback, fleet roll target).
const SentinelControlPlane = "@cp"

// shellMetachars mirrors the node agent's privsep filter
// (agent/internal/privsep/privsep.go) EXACTLY. The two are intentionally
// duplicated because the agent is a separate Go module; keeping the sets
// identical means a ref this boundary accepts is one privsep will also accept, so
// the friendly rejection here never diverges from the agent's hard rejection.
const shellMetachars = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~ "

// GitCommit is the control plane's build commit, injected at build time via
// -ldflags "-X vrooli-bridge/internal/cprev.GitCommit=<sha>" (the pattern
// internal/setup/setup.go uses for the main binary). It is only a FALLBACK for a
// deployed binary with no reachable git checkout; the live `git rev-parse HEAD`
// is preferred. Empty when the build did not wire it.
var GitCommit = ""

// defaultAdvertisementTTL bounds how long a single `git ls-remote` result is
// reused. A preflight per op start is intended; this keeps a fleet roll (one
// Sync per node) from issuing one network round-trip per node while still
// noticing a push that lands mid-roll within a few seconds.
const defaultAdvertisementTTL = 30 * time.Second

// GitResult is one git invocation's outcome. ExitCode carries git's own exit
// status so callers can distinguish a meaningful non-zero (e.g. `merge-base
// --is-ancestor` reporting "not an ancestor" with code 1) from a real execution
// failure (err != nil, e.g. git not installed).
type GitResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// GitRunner runs one git command in dir and returns its result. err is non-nil
// ONLY when git could not be executed at all (missing binary, context cancelled);
// a git command that ran and exited non-zero returns err == nil with ExitCode
// set. Production wires execGitRunner; tests inject a fake.
type GitRunner interface {
	Run(ctx context.Context, dir string, args ...string) (GitResult, error)
}

// Resolver defaults, expands, validates, and preflights revisions against one
// control-plane checkout + clone remote. It is safe for concurrent use.
type Resolver struct {
	repoDir  string
	remote   string
	embedded string
	git      GitRunner
	ttl      time.Duration
	now      func() time.Time

	mu      sync.Mutex
	advSHAs map[string]struct{}
	advAt   time.Time
}

// Option customises a Resolver.
type Option func(*Resolver)

// WithGitRunner overrides the git-execution seam (tests).
func WithGitRunner(g GitRunner) Option { return func(r *Resolver) { r.git = g } }

// WithRepoDir sets the control-plane checkout git commands run against. Empty
// means "let git discover it" (commands run in the process working directory,
// which is inside the monorepo in production).
func WithRepoDir(dir string) Option {
	return func(r *Resolver) { r.repoDir = strings.TrimSpace(dir) }
}

// WithRemote overrides the clone remote name preflight checks against
// (default "origin").
func WithRemote(name string) Option {
	return func(r *Resolver) {
		if n := strings.TrimSpace(name); n != "" {
			r.remote = n
		}
	}
}

// WithEmbeddedCommit overrides the build-commit fallback (tests).
func WithEmbeddedCommit(sha string) Option {
	return func(r *Resolver) { r.embedded = strings.TrimSpace(sha) }
}

// WithAdvertisementTTL overrides the ls-remote cache lifetime (tests).
func WithAdvertisementTTL(d time.Duration) Option {
	return func(r *Resolver) {
		if d > 0 {
			r.ttl = d
		}
	}
}

// WithClock overrides the time source (tests).
func WithClock(now func() time.Time) Option { return func(r *Resolver) { r.now = now } }

// New constructs a Resolver. In production main.go passes no repo dir (git
// discovers the monorepo root from the working directory) and the default
// "origin" remote; the embedded fallback defaults to the package GitCommit var.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		remote:   "origin",
		embedded: strings.TrimSpace(GitCommit),
		ttl:      defaultAdvertisementTTL,
		now:      time.Now,
	}
	if dir := strings.TrimSpace(os.Getenv("BRIDGE_CP_REPO_DIR")); dir != "" {
		r.repoDir = dir
	}
	if name := strings.TrimSpace(os.Getenv("BRIDGE_CP_GIT_REMOTE")); name != "" {
		r.remote = name
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.git == nil {
		r.git = execGitRunner{}
	}
	return r
}

// ControlPlaneCommit returns the control plane's current commit: the live
// `git rev-parse HEAD` of its checkout, falling back to the embedded build
// commit. It returns ErrNoControlPlaneCommit when neither is available, so a
// caller relying on the default (empty or "@cp") gets a clear error rather than a
// silent empty revision.
func (r *Resolver) ControlPlaneCommit(ctx context.Context) (string, error) {
	res, err := r.git.Run(ctx, r.repoDir, "rev-parse", "HEAD")
	if err == nil && res.ExitCode == 0 {
		if sha := strings.TrimSpace(res.Stdout); sha != "" {
			return sha, nil
		}
	}
	if r.embedded != "" {
		return r.embedded, nil
	}
	return "", ErrNoControlPlaneCommit{}
}

// Expand defaults + expands + validates a revision WITHOUT preflighting it. An
// empty request stays empty (so a first-provision rollback target is preserved);
// the "@cp" sentinel expands to the control plane's commit; any other value is
// returned as-is. The result is metachar-validated so a bad ref is rejected at
// the boundary. Use it for a rollback target — a recovery point that is where the
// node already was, so it must not be gated on remote containment.
func (r *Resolver) Expand(ctx context.Context, requested string) (string, error) {
	req := strings.TrimSpace(requested)
	if req == "" {
		return "", nil
	}
	resolved, err := r.expandSentinel(ctx, req)
	if err != nil {
		return "", err
	}
	if err := ValidateRef(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

// Resolve is the full pipeline for a DISPATCH target: default an empty request to
// the control plane's commit, expand "@cp", validate the ref, and preflight that
// the resolved commit exists on the clone remote. It returns the exact commit an
// op will pin to, or a typed error (ErrUnsafeRevision / ErrNotPushed /
// ErrNoControlPlaneCommit) the handler translates to a friendly Connect error.
func (r *Resolver) Resolve(ctx context.Context, requested string) (string, error) {
	req := strings.TrimSpace(requested)

	var resolved string
	if req == "" {
		commit, err := r.ControlPlaneCommit(ctx)
		if err != nil {
			return "", err
		}
		resolved = commit
	} else {
		expanded, err := r.expandSentinel(ctx, req)
		if err != nil {
			return "", err
		}
		resolved = expanded
	}

	if err := ValidateRef(resolved); err != nil {
		return "", err
	}
	if err := r.preflight(ctx, resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

// ResolveWorkingTree is the working-tree-mode pipeline: default an empty request
// to the control plane's commit, expand "@cp", and validate the ref — but SKIP
// the pushed preflight. In working-tree mode the control plane ships its LOCAL
// tree to the node over SSH rather than having the node fetch a commit, so the
// base commit is deliberately allowed to be unpushed (that is the whole point of
// the mode). It returns the base commit the shipped tree sits on, or a typed
// error (ErrUnsafeRevision / ErrNoControlPlaneCommit). The ErrNotPushed hard
// failure that Resolve enforces is intentionally NOT reachable here; Resolve
// (pinned mode) keeps it.
func (r *Resolver) ResolveWorkingTree(ctx context.Context, requested string) (string, error) {
	req := strings.TrimSpace(requested)

	var resolved string
	if req == "" {
		commit, err := r.ControlPlaneCommit(ctx)
		if err != nil {
			return "", err
		}
		resolved = commit
	} else {
		expanded, err := r.expandSentinel(ctx, req)
		if err != nil {
			return "", err
		}
		resolved = expanded
	}

	if err := ValidateRef(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

// expandSentinel replaces the "@cp" token with the control-plane commit and
// leaves every other token untouched.
func (r *Resolver) expandSentinel(ctx context.Context, req string) (string, error) {
	if req == SentinelControlPlane {
		return r.ControlPlaneCommit(ctx)
	}
	return req, nil
}

// ValidateRef rejects a revision token carrying a shell metacharacter, mirroring
// the node agent's privsep filter so the rejection happens here (friendly) rather
// than deep in the privileged helper (opaque). It is exported so the handler and
// tests can assert the boundary directly.
func ValidateRef(revision string) error {
	rev := strings.TrimSpace(revision)
	if rev == "" {
		return ErrUnsafeRevision{Revision: revision, Reason: "empty"}
	}
	if i := strings.IndexAny(rev, shellMetachars); i >= 0 {
		return ErrUnsafeRevision{
			Revision: rev,
			Reason:   fmt.Sprintf("contains disallowed character %q (relative refs like HEAD~1 are not allowed; pass an exact revision or a branch/tag name)", string(rev[i])),
		}
	}
	return nil
}

// preflight confirms the resolved commit exists on the clone remote. The check is
// deliberately layered so it is both cheap and correct:
//
//  1. one `git ls-remote <remote>` (cached, TTL-bounded) advertises the remote's
//     ref SHAs — an exact match means the commit is a remote branch/tag tip, the
//     overwhelmingly common "did you push your branch" case;
//  2. otherwise the commit may still be an older ancestor of a remote ref, so it
//     is tested with `git merge-base --is-ancestor` against each advertised SHA
//     the local object store actually holds.
//
// A confirmed miss returns ErrNotPushed (hard failure, push-first guidance). When
// the check CANNOT be run at all — git missing, remote unreachable, no advertised
// refs — preflight degrades to allow-with-no-error rather than blocking every op
// on an infrastructure hiccup: the drift this guards against is a LOCAL unpushed
// commit, which a reachable remote surfaces cleanly; a broken remote is a
// separate operational problem that should not wedge the whole fleet path.
func (r *Resolver) preflight(ctx context.Context, commit string) error {
	shas, ok := r.advertisedSHAs(ctx)
	if !ok {
		// Indeterminate (git/remote unavailable): degrade to allow. The caller
		// logs; see doc comment.
		return nil
	}
	if len(shas) == 0 {
		return nil
	}
	if _, exact := shas[commit]; exact {
		return nil
	}
	for sha := range shas {
		res, err := r.git.Run(ctx, r.repoDir, "merge-base", "--is-ancestor", commit, sha)
		if err != nil {
			continue // this ref's object is not local; try the next
		}
		if res.ExitCode == 0 {
			return nil // commit is an ancestor of an advertised ref → pushed
		}
	}
	return ErrNotPushed{Commit: commit, Remote: r.remote}
}

// advertisedSHAs returns the set of ref SHAs the clone remote advertises, cached
// for ttl. ok is false when the advertisement cannot be obtained (git failure) so
// the caller can distinguish "remote has no refs" (empty set, ok) from "could not
// ask the remote" (ok false → indeterminate).
func (r *Resolver) advertisedSHAs(ctx context.Context) (map[string]struct{}, bool) {
	r.mu.Lock()
	if r.advSHAs != nil && r.now().Sub(r.advAt) < r.ttl {
		cached := r.advSHAs
		r.mu.Unlock()
		return cached, true
	}
	r.mu.Unlock()

	res, err := r.git.Run(ctx, r.repoDir, "ls-remote", r.remote)
	if err != nil || res.ExitCode != 0 {
		return nil, false
	}
	shas := parseLsRemote(res.Stdout)

	r.mu.Lock()
	r.advSHAs = shas
	r.advAt = r.now()
	r.mu.Unlock()
	return shas, true
}

// parseLsRemote collects the SHA column from `git ls-remote` output. Each line is
// "<sha>\t<ref>"; malformed lines are skipped.
func parseLsRemote(out string) map[string]struct{} {
	shas := make(map[string]struct{})
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if sha := strings.TrimSpace(fields[0]); sha != "" {
			shas[sha] = struct{}{}
		}
	}
	return shas
}

// ---- typed errors (translated to friendly Connect codes at the handler) ----

// ErrNoControlPlaneCommit — the control plane's commit could not be determined
// (no git checkout and no embedded build commit), so an omitted/"@cp" revision
// cannot be defaulted. The operator must pass an explicit revision.
type ErrNoControlPlaneCommit struct{}

func (ErrNoControlPlaneCommit) Error() string {
	return "cannot determine the control plane's current commit (no git checkout and no embedded build commit); pass an explicit revision"
}

// ErrUnsafeRevision — a revision token was rejected at the API boundary before
// dispatch (mirrors the node agent's privsep filter).
type ErrUnsafeRevision struct {
	Revision string
	Reason   string
}

func (e ErrUnsafeRevision) Error() string {
	return fmt.Sprintf("invalid revision %q: %s", e.Revision, e.Reason)
}

// ErrNotPushed — the resolved commit is not present on the clone remote, so a
// node could never fetch it. The message names the commit and remote and tells
// the operator to push first.
type ErrNotPushed struct {
	Commit string
	Remote string
}

func (e ErrNotPushed) Error() string {
	return fmt.Sprintf(
		"commit %s is not on remote %q, so fleet nodes cannot fetch it; push it first (e.g. `git push %s HEAD`) or pass a revision that is already pushed",
		e.Commit, e.Remote, e.Remote,
	)
}

// ---- production git runner ----

// execGitRunner runs real git commands. A non-zero exit is reported via
// GitResult.ExitCode with a nil error; only a failure to execute git at all
// (missing binary, cancelled context) returns a non-nil error.
type execGitRunner struct{}

var _ GitRunner = execGitRunner{}

func (execGitRunner) Run(ctx context.Context, dir string, args ...string) (GitResult, error) {
	full := args
	if strings.TrimSpace(dir) != "" {
		full = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	res := GitResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return res, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// git ran and exited non-zero — a meaningful result, not an exec failure.
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	// git could not be executed (missing binary, cancelled): indeterminate.
	return res, err
}
