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
	"architecture-cartographer/internal/signals/graphindex"
)

const name = "git-co-edit"

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

func (Signal) Name() string           { return name }
func (Signal) DefaultWeight() float64 { return 0.6 }

func (s *Signal) IsAvailable(ctx context.Context) (bool, string) {
	if s.runner == nil {
		return false, "no git runner registered"
	}
	if !s.runner.IsAvailable(ctx) {
		return false, "git binary not available on PATH"
	}
	return true, ""
}

func (s *Signal) Score(ctx context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if err := ctx.Err(); err != nil {
		return signals.Abstain(name, err.Error(), chunk.Path)
	}
	if chunk.Path == "" {
		return signals.Abstain(name, "chunk has no path", "")
	}
	if ok, reason := s.IsAvailable(ctx); !ok {
		// Defensive: aggregator already skips unavailable signals, but
		// if someone calls Score directly we still satisfy the contract.
		return signals.Abstain(name, "git unavailable: "+reason, chunk.Path)
	}
	cache, err := s.coEditCache(ctx, gctx)
	if err != nil {
		return signals.Abstain(name, "git log failed: "+err.Error(), chunk.Path)
	}
	if len(cache.commitsByPath) == 0 {
		return signals.Abstain(name, "no git history for this file in the lookback window", chunk.Path)
	}
	commits := cache.commitsByPath[chunk.Path]
	if len(commits) < MinCoEditCommits {
		return signals.Abstain(name, fmt.Sprintf("fewer than %d co-edit commits in lookback window", MinCoEditCommits), chunk.Path)
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
		return signals.Abstain(name, "no co-edited files in the lookback window", chunk.Path)
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
		return signals.Abstain(name, "co-edited files are not mapped to any derived domain", chunk.Path)
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
		return signals.Abstain(name, "zero divisor for co-edit normalization", chunk.Path)
	}

	var out []signals.Score
	for dom, count := range domainScore {
		value := float64(count) / float64(divisor)
		if value > 1 {
			value = 1
		}
		out = append(out, signals.Score{
			Signal: name,
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
	return signals.ScoreResult{Scores: out}
}

type coEditCache struct {
	commitsByPath map[string][]commit
}

func (s *Signal) coEditCache(ctx context.Context, gctx signals.GraphContext) (coEditCache, error) {
	if gctx.Caches != nil {
		value, err := gctx.Caches.GitCoEditGetOrCompute(ctx, func(ctx context.Context) (any, error) {
			return s.readCoEditCache(ctx)
		})
		if err != nil {
			return coEditCache{}, err
		}
		if cached, ok := value.(coEditCache); ok {
			return cached, nil
		}
		return coEditCache{}, fmt.Errorf("git co-edit cache has unexpected type %T", value)
	}
	return s.readCoEditCache(ctx)
}

func (s *Signal) readCoEditCache(ctx context.Context) (coEditCache, error) {
	logOut, err := s.runner.Log(ctx, "--since="+s.lookback, "--name-only", "--pretty=format:%H")
	if err != nil {
		return coEditCache{}, err
	}
	if err := ctx.Err(); err != nil {
		return coEditCache{}, err
	}
	return coEditCache{commitsByPath: commitsByPath(parseCommits(logOut))}, nil
}

func commitsByPath(commits []commit) map[string][]commit {
	out := make(map[string][]commit)
	for _, c := range commits {
		for _, f := range c.Files {
			out[f] = append(out[f], c)
		}
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
	return graphindex.DomainForPath(path, gctx.DomainMap)
}
