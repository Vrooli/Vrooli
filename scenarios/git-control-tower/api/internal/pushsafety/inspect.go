// Package pushsafety inspects immutable outgoing history and prepares isolated
// recovery artifacts. It never writes the source repository or publishes refs.
package pushsafety

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const WarningBytes int64 = 50 * 1024 * 1024
const GitHubLimitBytes int64 = 100 * 1024 * 1024
const MaxCommits = 100

// Commands is the process boundary. Tests inject a recording fake; production
// supplies a bounded runner with credential handling and sanitized Git env.
type Commands interface {
	Run(context.Context, string, []byte, ...string) ([]byte, error)
}

type File struct {
	OID     string   `json:"oid"`
	Bytes   int64    `json:"bytes"`
	Paths   []string `json:"paths"`
	Commits []string `json:"commits"`
	Blocked bool     `json:"blocked"`
}
type Report struct {
	StagedComplete bool     `json:"-"`
	StagedReason   string   `json:"-"`
	StagedFiles    []File   `json:"-"`
	Complete       bool     `json:"complete"`
	State          string   `json:"state"` // clear, blocked, unknown
	Reason         string   `json:"reason"`
	Head           string   `json:"head"`
	Base           string   `json:"base"`
	Remote         string   `json:"remote"`
	Branch         string   `json:"branch"`
	URL            string   `json:"-"` // may contain credentials; never persist or expose
	Limit          int64    `json:"limit"`
	Commits        []string `json:"commits"`
	Files          []File   `json:"files"`
	Fingerprint    string   `json:"fingerprint"`
	CanPrepare     bool     `json:"can_prepare"`
	RecoveryReason string   `json:"recovery_reason"`
}

func gitString(ctx context.Context, c Commands, repo string, args ...string) (string, error) {
	b, e := c.Run(ctx, repo, nil, args...)
	return strings.TrimSpace(string(b)), e
}
func IsGitHub(raw string) bool {
	if strings.HasPrefix(raw, "git@github.com:") {
		return true
	}
	u, e := url.Parse(raw)
	return e == nil && strings.EqualFold(u.Hostname(), "github.com") && (u.Scheme == "https" || u.Scheme == "ssh")
}
func validOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}

// Inspect binds a report to HEAD and the live push destination. Missing remote
// objects, authentication failures and budget exhaustion remain unknown.
func Inspect(ctx context.Context, c Commands, repo, remote, branch string) Report {
	r := Report{State: "unknown", Remote: remote, Branch: branch}
	fail := func(reason string) Report { r.Reason = reason; return r }
	if remote == "" || strings.HasPrefix(remote, "-") || branch == "" {
		return fail("Select a remote and branch before checking outgoing history.")
	}
	if _, e := c.Run(ctx, repo, nil, "check-ref-format", "refs/heads/"+branch); e != nil {
		return fail("The destination branch is invalid.")
	}
	head, e := gitString(ctx, c, repo, "rev-parse", "--verify", "HEAD^{commit}")
	if e != nil || !validOID(head) {
		return fail("Cannot resolve the source commit.")
	}
	r.Head = head
	urls, e := gitString(ctx, c, repo, "remote", "get-url", "--push", "--all", remote)
	if e != nil || urls == "" || strings.Contains(urls, "\n") {
		return fail("One verified push URL is required; multiple push destinations need separate checks.")
	}
	r.URL = urls
	if IsGitHub(urls) {
		r.Limit = GitHubLimitBytes
	}
	refs, e := gitString(ctx, c, repo, "ls-remote", "--heads", urls, "refs/heads/"+branch)
	if e != nil {
		return fail("Cannot verify the remote tip. Check authentication and connectivity, then refresh. Your local commits are unchanged.")
	}
	if refs != "" {
		fields := strings.Fields(refs)
		if len(fields) != 2 || fields[1] != "refs/heads/"+branch || !validOID(fields[0]) {
			return fail("The remote returned an unexpected branch reference.")
		}
		r.Base = fields[0]
		if _, e = c.Run(ctx, repo, nil, "cat-file", "-e", r.Base+"^{commit}"); e != nil {
			return fail("The live remote tip is not available locally. Fetch the branch and check again.")
		}
	}
	rangeSpec := head
	if r.Base != "" {
		rangeSpec = r.Base + ".." + head
	}
	commits, e := gitString(ctx, c, repo, "rev-list", "--reverse", "--topo-order", "--max-count=101", rangeSpec)
	if e != nil {
		return fail("Outgoing commit enumeration failed.")
	}
	if commits != "" {
		r.Commits = strings.Fields(commits)
	}
	if len(r.Commits) > MaxCommits {
		return fail("Outgoing history exceeds the 100-commit inspection budget; no complete safety result is available.")
	}
	objects, e := c.Run(ctx, repo, nil, "rev-list", "--objects", "--no-object-names", rangeSpec)
	if e != nil {
		return fail("Outgoing object enumeration failed.")
	}
	metadata, e := c.Run(ctx, repo, objects, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if e != nil {
		return fail("Outgoing object sizes could not be read.")
	}
	expected := map[string]bool{}
	for _, oid := range strings.Fields(string(objects)) {
		if !validOID(oid) {
			return fail("Invalid outgoing object identity.")
		}
		expected[oid] = true
	}
	large := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(metadata)), "\n") {
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 || !validOID(f[0]) {
			return fail("Outgoing object metadata is incomplete.")
		}
		if !expected[f[0]] {
			return fail("Unexpected or duplicate outgoing object metadata.")
		}
		delete(expected, f[0])
		n, e := strconv.ParseInt(f[2], 10, 64)
		if e != nil || n < 0 {
			return fail("Outgoing object size is invalid.")
		}
		if f[1] == "blob" && n > WarningBytes {
			large[f[0]] = len(r.Files)
			r.Files = append(r.Files, File{OID: f[0], Bytes: n, Blocked: r.Limit > 0 && n > r.Limit})
		}
	}
	if len(expected) > 0 {
		return fail("Outgoing object metadata is incomplete.")
	}
	// Tree records use NUL delimiters, so tabs/newlines and pathspec characters
	// in filenames remain literal. Scan every commit, including later deletions.
	if len(large) > 0 {
		for _, commit := range r.Commits {
			tree, e := c.Run(ctx, repo, nil, "ls-tree", "-r", "-z", commit)
			if e != nil {
				return fail("Affected file paths could not be completely inspected.")
			}
			for _, record := range strings.Split(string(tree), "\x00") {
				if record == "" {
					continue
				}
				meta, path, ok := strings.Cut(record, "\t")
				f := strings.Fields(meta)
				if !ok || len(f) != 3 {
					return fail("Affected tree metadata is incomplete.")
				}
				if i, ok := large[f[2]]; ok {
					r.Files[i].Paths = appendUnique(r.Files[i].Paths, path)
					r.Files[i].Commits = appendUnique(r.Files[i].Commits, commit)
				}
			}
		}
	}
	r.State = "clear"
	r.Reason = "No outgoing file exceeds the detected host limit. Other remote policies may still reject the push."
	if r.Limit == 0 {
		r.State = "unknown"
		r.Reason = "This host's file-size policy is unknown. Large outgoing files are shown for review."
	}
	for _, f := range r.Files {
		if f.Blocked {
			r.State = "blocked"
			r.Reason = "Push blocked by oversized files in outgoing history. Your local commits and working files are unchanged."
		}
	}
	r.RecoveryReason = "Recovery preparation requires oversized files and a verified existing remote base."
	if r.State == "blocked" && r.Base != "" {
		_, ancestorErr := c.Run(ctx, repo, nil, "merge-base", "--is-ancestor", r.Base, r.Head)
		merges, mergeErr := gitString(ctx, c, repo, "rev-list", "--min-parents=2", rangeSpec)
		if ancestorErr == nil && mergeErr == nil && merges == "" {
			r.CanPrepare = true
			r.RecoveryReason = "Prepare replacements in a separate repository. Applying them requires a coordinated pause and a separate reviewed operation."
		} else {
			r.RecoveryReason = "Automatic preparation supports only linear unpublished history descending from the live remote tip. Merge or diverged history needs a separate review."
		}
	}
	r.Complete = true
	end, e := gitString(ctx, c, repo, "rev-parse", "--verify", "HEAD^{commit}")
	if e != nil || end != r.Head {
		r.State = "unknown"
		r.Complete = false
		r.CanPrepare = false
		r.Reason = "The source commit changed during inspection. Refresh the preview."
	}
	for i := range r.Files {
		sort.Strings(r.Files[i].Paths)
	}
	payload, _ := json.Marshal(r)
	sum := sha256.Sum256(append(payload, []byte(r.URL+"\x00"+repo)...))
	r.Fingerprint = fmt.Sprintf("%x", sum)
	return r
}

func appendUnique(xs []string, x string) []string {
	for _, v := range xs {
		if v == x {
			return xs
		}
	}
	return append(xs, x)
}

func verifySizes(raw []byte, limit int64) error {
	if limit <= 0 {
		return fmt.Errorf("host size limit is unknown")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 || !validOID(f[0]) {
			return fmt.Errorf("incomplete candidate object metadata")
		}
		n, e := strconv.ParseInt(f[2], 10, 64)
		if e != nil || n < 0 {
			return fmt.Errorf("invalid candidate object size")
		}
		if f[1] == "blob" && n > limit {
			return fmt.Errorf("candidate still contains an oversized blob")
		}
	}
	return nil
}
