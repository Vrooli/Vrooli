package stagedmessages

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"git-control-tower/internal/trailers"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

const (
	continuationHistoryDepth = 50
	continuationMaxAge       = 14 * 24 * time.Hour
	continuationMaxResults   = 3
)

// ContinuationFinder suggests recent commits the staged change continues.
type ContinuationFinder interface {
	Continuations(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContinuation, error)
}

// GitContinuations reads recent first-parent history of HEAD.
type GitContinuations struct{ Now func() time.Time }

type historyCommit struct {
	Hash    string
	Time    time.Time
	Message trailers.Message
	Paths   []string
}

func (g GitContinuations) Continuations(ctx context.Context, dir string, snapshot *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContinuation, error) {
	if snapshot == nil || len(snapshot.Files) == 0 {
		return nil, nil
	}
	out, err := gitRead(ctx, dir, "-c", "core.quotePath=false", "log", "-n", strconv.Itoa(continuationHistoryDepth), "--no-merges", "--first-parent", "--name-only", "--format=%x1e%H%x1f%ct%x1f%B%x1f", "HEAD", "--")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if g.Now != nil {
		now = g.Now()
	}
	return rankContinuations(snapshot, parseHistory(string(out)), now), nil
}

func parseHistory(raw string) []historyCommit {
	var commits []historyCommit
	for _, record := range strings.Split(raw, "\x1e") {
		parts := strings.SplitN(record, "\x1f", 4)
		if len(parts) != 4 {
			continue
		}
		seconds, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			continue
		}
		commit := historyCommit{Hash: strings.TrimSpace(parts[0]), Time: time.Unix(seconds, 0).UTC(), Message: trailers.Parse(parts[2])}
		for _, line := range strings.Split(parts[3], "\n") {
			if path := strings.TrimSpace(line); path != "" {
				commit.Paths = append(commit.Paths, path)
			}
		}
		commits = append(commits, commit)
	}
	return commits
}

// rankContinuations favors commits that share most of the staged paths, are
// focused on them, and are recent. A commit another commit already continues
// is not a series head and is skipped.
func rankContinuations(snapshot *v1.StagedMessageSnapshot, commits []historyCommit, now time.Time) []*v1.StagedMessageContinuation {
	staged := map[string]bool{}
	for _, file := range snapshot.Files {
		staged[file.Path] = true
		if file.OldPath != "" {
			staged[file.OldPath] = true
		}
	}
	continued := map[string]bool{}
	for _, commit := range commits {
		for _, entry := range commit.Message.Entries {
			if strings.EqualFold(entry.Key, "Vrooli-Continues") {
				continued[strings.ToLower(strings.TrimSpace(entry.Value))] = true
			}
		}
	}
	isContinued := func(hash string) bool {
		for prefix := range continued {
			if len(prefix) >= 7 && strings.HasPrefix(hash, prefix) {
				return true
			}
		}
		return false
	}
	type scored struct {
		value *v1.StagedMessageContinuation
		score float64
	}
	var ranked []scored
	for index, commit := range commits {
		if commit.Hash == "" || now.Sub(commit.Time) > continuationMaxAge || isContinued(strings.ToLower(commit.Hash)) || !trailers.ValidPredecessor(commit.Hash) {
			continue
		}
		shared := 0
		for _, path := range commit.Paths {
			if staged[path] {
				shared++
			}
		}
		if shared == 0 {
			continue
		}
		coverage := float64(shared) / float64(len(snapshot.Files))
		focus := float64(shared) / float64(len(commit.Paths))
		if coverage > 1 {
			coverage = 1
		}
		if coverage < 0.5 && (shared < 2 || focus < 0.5) {
			continue
		}
		entries := trailers.ContinuationEntries(commit.Message, commit.Hash)
		suggestion := &v1.StagedMessageContinuation{
			Commit: commit.Hash, Subject: commit.Message.Subject, SharedPaths: int32(shared),
			CommittedAt: commit.Time.Format(time.RFC3339),
			Reason:      fmt.Sprintf("Shares %d of %d staged files", min(shared, len(snapshot.Files)), len(snapshot.Files)),
		}
		for _, entry := range entries {
			suggestion.Trailers = append(suggestion.Trailers, &v1.StagedMessageTrailer{Key: entry.Key, Value: entry.Value})
		}
		ranked = append(ranked, scored{suggestion, coverage + focus - 0.05*float64(index)})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	var result []*v1.StagedMessageContinuation
	for _, item := range ranked {
		if len(result) == continuationMaxResults {
			break
		}
		result = append(result, item.value)
	}
	return result
}
