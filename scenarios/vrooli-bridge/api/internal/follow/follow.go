// Package follow keeps a node on the tip of a git branch. An owner sets a
// node to follow a branch; Bridge then reads that branch's head from the
// node's source repository on a schedule and, whenever the head moves,
// dispatches a privileged provisioning op to bring the node to the new
// commit. It is the "set a machine to a branch and let it update itself" half
// of Bridge's two update paths; working-tree ships (onboarding) remain the
// other half, for uncommitted work.
//
// Following a branch never updates a node to a head it has already been sent:
// the policy records the last head it dispatched (or saw when following
// started), so a working-tree ship on top of a followed node stays in place
// until the branch actually receives a new commit. A head whose dispatch was
// refused (node offline, helper missing) is retried on the next check; a head
// that was dispatched is never re-sent, even if its provisioning op failed, so
// a bad commit cannot start a retry storm.
package follow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS node_branch_follow (
  node_id TEXT PRIMARY KEY,
  branch TEXT NOT NULL,
  repo_url TEXT NOT NULL,
  last_seen_head TEXT NOT NULL DEFAULT '',
  last_checked_at TEXT NOT NULL DEFAULT '',
  last_result TEXT NOT NULL DEFAULT '',
  last_op_id TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);`

// Schema contributes the branch-follow policy table.
func Schema() string { return schema }

// DefaultRepoURL is the source repository nodes clone from.
const DefaultRepoURL = "https://github.com/Vrooli/Vrooli.git"

// Policy is one node's branch-follow setting and the outcome of its last check.
type Policy struct {
	NodeID        string    `json:"node_id"`
	Branch        string    `json:"branch"`
	RepoURL       string    `json:"repo_url"`
	LastSeenHead  string    `json:"last_seen_head"`
	LastCheckedAt time.Time `json:"last_checked_at"`
	LastResult    string    `json:"last_result"`
	LastOpID      string    `json:"last_op_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SQLExecutor is satisfied by both *sql.DB and api-core's routed database.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// HeadResolver reads a branch's current commit from a repository.
type HeadResolver interface {
	Head(ctx context.Context, repoURL, branch string) (string, error)
}

// Provisioner dispatches a provisioning op that brings a node to a commit.
type Provisioner interface {
	Provision(ctx context.Context, nodeID, revision, actor string) (opID string, err error)
}

// NodeChecker reports whether a node exists in the registry.
type NodeChecker func(ctx context.Context, nodeID string) error

// ErrInvalid is an owner input Bridge refuses.
type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason) }

// ErrNotFollowing is returned for a node with no branch-follow policy.
var ErrNotFollowing = errors.New("node is not following a branch")

// Service owns branch-follow policies and their checks.
type Service struct {
	db    SQLExecutor
	heads HeadResolver
	prov  Provisioner
	nodes NodeChecker
	now   func() time.Time
	mu    sync.Mutex
}

// NewService constructs the branch-follow service.
func NewService(db SQLExecutor, heads HeadResolver, prov Provisioner, nodes NodeChecker, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{db: db, heads: heads, prov: prov, nodes: nodes, now: now}
}

var branchPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

func validateBranch(branch string) error {
	if !branchPattern.MatchString(branch) || strings.Contains(branch, "..") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".lock") {
		return ErrInvalid{Field: "branch", Reason: fmt.Sprintf("%q is not a plain branch name", branch)}
	}
	return nil
}

// validateRepoURL accepts only network transports git resolves without
// running a helper program (no ext::, no local paths, no option injection).
func validateRepoURL(repoURL string) error {
	switch {
	case strings.HasPrefix(repoURL, "https://"), strings.HasPrefix(repoURL, "http://"), strings.HasPrefix(repoURL, "ssh://"):
	case regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[A-Za-z0-9._/-]+$`).MatchString(repoURL):
	default:
		return ErrInvalid{Field: "repo_url", Reason: "use an https://, ssh://, or user@host:path repository URL"}
	}
	if strings.ContainsAny(repoURL, " \t\n\r") {
		return ErrInvalid{Field: "repo_url", Reason: "must not contain whitespace"}
	}
	return nil
}

// Follow sets nodeID to follow branch. The branch's current head is recorded
// as already seen, so following starts with the next commit; updateNow
// dispatches the current head immediately instead.
func (s *Service) Follow(ctx context.Context, nodeID, branch, repoURL string, updateNow bool) (Policy, error) {
	nodeID, branch, repoURL = strings.TrimSpace(nodeID), strings.TrimSpace(branch), strings.TrimSpace(repoURL)
	if repoURL == "" {
		repoURL = DefaultRepoURL
	}
	if nodeID == "" {
		return Policy{}, ErrInvalid{Field: "node_id", Reason: "is required"}
	}
	if err := validateBranch(branch); err != nil {
		return Policy{}, err
	}
	if err := validateRepoURL(repoURL); err != nil {
		return Policy{}, err
	}
	if s.nodes != nil {
		if err := s.nodes(ctx, nodeID); err != nil {
			return Policy{}, err
		}
	}
	head, err := s.heads.Head(ctx, repoURL, branch)
	if err != nil {
		return Policy{}, fmt.Errorf("read the head of %s in %s: %w", branch, repoURL, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	policy := Policy{
		NodeID: nodeID, Branch: branch, RepoURL: repoURL, LastSeenHead: head, LastCheckedAt: now, UpdatedAt: now,
		LastResult: "following " + branch + " from " + shortSHA(head) + "; the node updates when the branch receives a new commit",
	}
	if updateNow {
		policy.LastSeenHead = ""
	}
	if err := s.save(ctx, policy); err != nil {
		return Policy{}, err
	}
	if updateNow {
		s.checkLocked(ctx, &policy)
	}
	return policy, nil
}

// Unfollow stops following for nodeID.
func (s *Service) Unfollow(ctx context.Context, nodeID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM node_branch_follow WHERE node_id = ?`, strings.TrimSpace(nodeID))
	if err != nil {
		return fmt.Errorf("remove branch-follow policy: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFollowing
	}
	return nil
}

// Get returns nodeID's policy.
func (s *Service) Get(ctx context.Context, nodeID string) (Policy, error) {
	policies, err := s.query(ctx, `WHERE node_id = ?`, strings.TrimSpace(nodeID))
	if err != nil {
		return Policy{}, err
	}
	if len(policies) == 0 {
		return Policy{}, ErrNotFollowing
	}
	return policies[0], nil
}

// List returns every policy.
func (s *Service) List(ctx context.Context) ([]Policy, error) {
	return s.query(ctx, `ORDER BY node_id`)
}

// Check checks one node now.
func (s *Service) Check(ctx context.Context, nodeID string) (Policy, error) {
	policy, err := s.Get(ctx, nodeID)
	if err != nil {
		return Policy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkLocked(ctx, &policy)
	return policy, nil
}

// CheckAll checks every followed node; the scheduler calls it periodically.
func (s *Service) CheckAll(ctx context.Context) []Policy {
	policies, err := s.List(ctx)
	if err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range policies {
		s.checkLocked(ctx, &policies[i])
	}
	return policies
}

func (s *Service) checkLocked(ctx context.Context, p *Policy) {
	p.LastCheckedAt = s.now().UTC()
	defer func() { _ = s.save(ctx, *p) }()
	head, err := s.heads.Head(ctx, p.RepoURL, p.Branch)
	if err != nil {
		p.LastResult = "could not read the branch head: " + err.Error()
		return
	}
	if head == p.LastSeenHead {
		if !strings.HasPrefix(p.LastResult, "following ") && !strings.HasPrefix(p.LastResult, "dispatched ") {
			p.LastResult = "up to date with " + p.Branch + " at " + shortSHA(head)
		}
		return
	}
	opID, err := s.prov.Provision(ctx, p.NodeID, head, "bridge:follow:"+p.Branch)
	if err != nil {
		p.LastResult = "update to " + shortSHA(head) + " was not dispatched (retrying on the next check): " + err.Error()
		return
	}
	p.LastSeenHead, p.LastOpID = head, opID
	p.LastResult = "dispatched update to " + p.Branch + " at " + shortSHA(head) + " (provisioning op " + opID + ")"
}

func (s *Service) save(ctx context.Context, p Policy) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO node_branch_follow (node_id, branch, repo_url, last_seen_head, last_checked_at, last_result, last_op_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(node_id) DO UPDATE SET branch = excluded.branch, repo_url = excluded.repo_url, last_seen_head = excluded.last_seen_head,
  last_checked_at = excluded.last_checked_at, last_result = excluded.last_result, last_op_id = excluded.last_op_id, updated_at = excluded.updated_at`,
		p.NodeID, p.Branch, p.RepoURL, p.LastSeenHead, formatTime(p.LastCheckedAt), p.LastResult, p.LastOpID, formatTime(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("save branch-follow policy: %w", err)
	}
	return nil
}

func (s *Service) query(ctx context.Context, where string, args ...any) ([]Policy, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT node_id, branch, repo_url, last_seen_head, last_checked_at, last_result, last_op_id, updated_at FROM node_branch_follow `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("read branch-follow policies: %w", err)
	}
	defer rows.Close()
	var out []Policy
	for rows.Next() {
		var p Policy
		var checked, updated string
		if err := rows.Scan(&p.NodeID, &p.Branch, &p.RepoURL, &p.LastSeenHead, &checked, &p.LastResult, &p.LastOpID, &updated); err != nil {
			return nil, fmt.Errorf("read branch-follow policy: %w", err)
		}
		p.LastCheckedAt, p.UpdatedAt = parseTime(checked), parseTime(updated)
		out = append(out, p)
	}
	return out, rows.Err()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// GitHeads resolves branch heads with `git ls-remote`.
type GitHeads struct {
	Timeout time.Duration
}

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// Head returns the commit refs/heads/<branch> points at in repoURL.
func (g GitHeads) Head(ctx context.Context, repoURL, branch string) (string, error) {
	if err := validateRepoURL(repoURL); err != nil {
		return "", err
	}
	if err := validateBranch(branch); err != nil {
		return "", err
	}
	timeout := g.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ref := "refs/heads/" + branch
	// Read the branch anonymously first, with the operator's git configuration
	// neutralized: a global `url.…insteadOf` rewrite turned this public HTTPS
	// read into an SSH one and failed on a server with no agent (2026-09-15).
	// A repository that needs the operator's credentials still works through
	// the ambient retry.
	out, err := g.lsRemote(ctx, repoURL, ref, true)
	if err != nil {
		out, err = g.lsRemote(ctx, repoURL, ref, false)
	}
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == ref && shaPattern.MatchString(fields[0]) {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("branch %s does not exist in %s", branch, repoURL)
}

// lsRemote runs one `git ls-remote`; isolated drops the operator's global and
// system git configuration (URL rewrites, credential helpers).
func (g GitHeads) lsRemote(ctx context.Context, repoURL, ref string, isolated bool) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--", repoURL, ref) // #nosec G204 -- validated URL and ref, no shell.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=/bin/false")
	if isolated {
		cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git ls-remote: %s", msg)
		}
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	return string(out), nil
}
