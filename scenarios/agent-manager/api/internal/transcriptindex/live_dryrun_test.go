package transcriptindex_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/transcriptindex"
)

// TestLiveDiscoveryCoverageDryRun runs the real indexer over this host's real
// transcript roots into a throwaway SQLite database and prints the coverage
// table it would report. It reads the roots only; it never touches the live
// Agent Manager database. Opt in with TRANSCRIPT_INDEX_LIVE_DRYRUN=1 (and
// optionally TRANSCRIPT_INDEX_LIVE_DAYS, default 7).
func TestLiveDiscoveryCoverageDryRun(t *testing.T) {
	if os.Getenv("TRANSCRIPT_INDEX_LIVE_DRYRUN") != "1" {
		t.Skip("set TRANSCRIPT_INDEX_LIVE_DRYRUN=1 to index the real roots into a temp database")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	days := 7
	if raw := os.Getenv("TRANSCRIPT_INDEX_LIVE_DAYS"); raw != "" {
		_, _ = fmt.Sscan(raw, &days)
	}
	r := newRig(t, 0) // summaries only: no recall events in the temp database
	r.home, r.runRoot = home, filepath.Join(home, ".vrooli", "state", "vrooli", "agent-manager", "runs")
	r.restart()
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	before := transcriptindex.BuildCoverage(transcriptindex.Roots(transcriptindex.RootConfig{Home: r.home, RunStateRoot: r.runRoot}), nil, since)
	started := time.Now()
	var read int64
	for tick := 0; tick < 1000; tick++ {
		summary, err := r.index.RunOnce(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		read += summary.BytesRead
		for _, failure := range summary.Failures {
			t.Logf("failure: %s", failure)
		}
		if summary.Deferred == 0 {
			break
		}
		// Back-fill continues newest change first; the window is complete
		// once every file modified inside it has a row (live files that grew
		// since the tick are "partially indexed", not unread).
		window, err := r.index.Coverage(context.Background(), since)
		if err != nil {
			t.Fatal(err)
		}
		unread := 0
		for _, root := range window.Roots {
			for _, issue := range root.Issues {
				if issue.Reason == "not indexed yet" {
					unread++
				}
			}
		}
		if unread == 0 {
			break
		}
	}
	elapsed := time.Since(started)
	coverage, err := r.index.Coverage(context.Background(), since)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "\nback-fill read %.2f GB in %s; coverage window %dd (since %s)\n", float64(read)/1e9, elapsed.Round(time.Second), days, since.Format(time.RFC3339))
	fmt.Fprintf(&out, "%-24s %6s %7s %6s %6s %5s %6s %6s %9s %9s %8s %9s\n", "root", "files", "indexed", "usage", "owned", "excl", "pend", "unread", "GB", "GB capt", "capt %", "reports")
	for _, root := range append(coverage.Roots, coverage.Totals) {
		share := 0.0
		if root.Bytes > 0 {
			share = 100 * float64(root.IndexedBytes) / float64(root.Bytes)
		}
		fmt.Fprintf(&out, "%-24s %6d %7d %6d %6d %5d %6d %6d %9.2f %9.2f %7.1f%% %9d\n", root.Kind, root.Files, root.Indexed, root.WithUsage, root.Owned, root.Excluded, root.Pending, root.Unreadable,
			float64(root.Bytes)/1e9, float64(root.IndexedBytes)/1e9, share, root.UsageReports)
	}
	fmt.Fprintf(&out, "before indexing (disk only): files=%d pending=%d owned=%d\n", before.Totals.Files, before.Totals.Pending, before.Totals.Owned)
	for _, root := range coverage.Roots {
		for _, issue := range root.Issues {
			if issue.Reason != "not indexed yet" {
				fmt.Fprintf(&out, "  %s: %s — %s\n", root.Kind, issue.Path, issue.Reason)
			}
		}
	}
	t.Log(out.String())
}
