// Package gitcoedit is the git-co-edit signal. Reads `git log` over a
// lookback window and scores domains containing files that frequently
// co-edit with the chunk's file. Premise: if A and B change together in
// 70% of commits over the last 90 days, they probably belong together.
//
// Self-disables when `git` is unavailable. Default weight 0.6 per
// SIGNAL_LADDER.md.
package gitcoedit

import (
	"context"
	"fmt"
	"strings"

	"architecture-cartographer/internal/git"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// DefaultLookback is the default `--since` window passed to `git log`.
const DefaultLookback = "90.days.ago"

// MinCoEditCommits is the floor below which we don't draw conclusions.
const MinCoEditCommits = 3

// Signal is the production git-co-edit signal.
type Signal struct {
	runner   git.Runner
	lookback string
}

// New returns the production signal wired to the given runner.
func New(runner git.Runner) *Signal {
	return &Signal{runner: runner, lookback: DefaultLookback}
}

// WithLookback returns a copy of the signal with a custom --since value.
func (s *Signal) WithLookback(lb string) *Signal {
	cp := *s
	cp.lookback = lb
	return &cp
}

func (Signal) Name() string             { return "git-co-edit" }
func (Signal) DefaultWeight() float64   { return 0.6 }

func (s *Signal) IsAvailable(ctx context.Context) (bool, string) {
	if s.runner == nil {
		return false, "no git runner registered"
	}
	if !s.runner.IsAvailable(ctx) {
		return false, "git binary not available on PATH"
	}
	return true, ""
}

func (s *Signal) Score(ctx context.Context, gctx signals.GraphContext, chunk graph.Chunk) []signals.Score {
	if chunk.Path == "" {
		return nil
	}
	if ok, _ := s.IsAvailable(ctx); !ok {
		return nil
	}
	// `git log --since=<lookback> --name-only --pretty=format:%H -- <path>`
	logOut, err := s.runner.Log(ctx, "--since="+s.lookback, "--name-only", "--pretty=format:%H", "--", chunk.Path)
	if err != nil || strings.TrimSpace(logOut) == "" {
		return nil
	}
	commits := parseCommits(logOut)
	if len(commits) < MinCoEditCommits {
		return nil
	}
	// Count co-edit frequency by other file.
	coEdit := make(map[string]int)
	totalCommits := 0
	for _, c := range commits {
		totalCommits++
		for _, f := range c.Files {
			if f == chunk.Path {
				continue
			}
			coEdit[f]++
		}
	}
	if len(coEdit) == 0 {
		return nil
	}

	// Map each co-edited file to a domain.
	domainScore := make(map[string]int)
	for f, count := range coEdit {
		dom := domainForFilePath(f, gctx)
		if dom == "" {
			continue
		}
		domainScore[dom] += count
	}
	if len(domainScore) == 0 {
		return nil
	}

	// Normalize to [0, 1] using totalCommits * top-co-edit as the divisor.
	maxCount := 0
	for _, v := range domainScore {
		if v > maxCount {
			maxCount = v
		}
	}
	divisor := totalCommits
	if maxCount > divisor {
		divisor = maxCount
	}
	if divisor == 0 {
		return nil
	}

	var out []signals.Score
	for dom, count := range domainScore {
		value := float64(count) / float64(divisor)
		if value > 1 {
			value = 1
		}
		out = append(out, signals.Score{
			Signal: "git-co-edit",
			Domain: dom,
			Value:  value,
			Reason: fmt.Sprintf("co-edited with files in %q across %d commit(s)", dom, totalCommits),
			Evidence: []signals.Evidence{{
				Kind:    "git_co_edit",
				Summary: fmt.Sprintf("%d co-edits with %q files over %s", count, dom, s.lookback),
				Locator: chunk.Path,
				Weight:  value,
			}},
		})
	}
	return out
}

type commit struct {
	Hash  string
	Files []string
}

func parseCommits(raw string) []commit {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var out []commit
	var cur *commit
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			if cur != nil && len(cur.Files) > 0 {
				out = append(out, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			cur = &commit{Hash: l}
			continue
		}
		// Heuristic: hashes are 40-char lowercase hex; lines that look
		// like file paths fall into the Files slice.
		if isHash(l) {
			if len(cur.Files) > 0 {
				out = append(out, *cur)
			}
			cur = &commit{Hash: l}
			continue
		}
		cur.Files = append(cur.Files, l)
	}
	if cur != nil && len(cur.Files) > 0 {
		out = append(out, *cur)
	}
	return out
}

func isHash(s string) bool {
	if len(s) < 7 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func domainForFilePath(path string, gctx signals.GraphContext) string {
	for _, d := range gctx.Manifest.Domains {
		for _, glob := range d.Paths {
			if matches(path, glob) {
				return d.Name
			}
		}
	}
	return ""
}

func matches(path, glob string) bool {
	switch {
	case glob == "**":
		return true
	case strings.HasSuffix(glob, "/**"):
		prefix := glob[:len(glob)-3]
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	default:
		return path == glob
	}
}
