//go:build livecheck

package transcriptindex_test

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/transcriptindex"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestLiveCoverageAgreesWithAnIndependentDiskScan is the A1 check: the
// coverage `usage coverage` reports from the live database (opened
// read-only) against a scan that shares no indexer code. The scan walks the
// same directories, sums the bytes of every *.jsonl modified in the period
// and counts provider turns on disk (Codex turn_context lines, Claude
// assistant lines with usage). Opt in with USAGE_COVERAGE_LIVE=1.
func TestLiveCoverageAgreesWithAnIndependentDiskScan(t *testing.T) {
	if os.Getenv("USAGE_COVERAGE_LIVE") != "1" {
		t.Skip("set USAGE_COVERAGE_LIVE=1 to compare live coverage with a disk scan")
	}
	home, _ := os.UserHomeDir()
	live := filepath.Join("..", "..", "..", "data", "agent-manager.db")
	raw, err := sqlx.Connect("sqlite", "file:"+live+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	rows, err := database.NewTranscriptIndexRepository(database.NewDB(raw, log)).ListTranscriptFiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-7 * 24 * time.Hour)
	cfg := transcriptindex.RootConfig{Home: home, RunStateRoot: filepath.Join(home, ".vrooli", "state", "vrooli", "agent-manager", "runs")}
	coverage := transcriptindex.BuildCoverage(transcriptindex.Roots(cfg), rows, since)

	byPath := map[string]transcriptindex.FileRow{}
	for _, row := range rows {
		byPath[row.Path] = row
	}
	type tally struct{ files, bytes, units, capturedUnits, indexedUnits int64 }
	disk := map[string]*tally{}
	for _, root := range transcriptindex.Roots(cfg) {
		scan := disk[root.Kind]
		if scan == nil {
			scan = &tally{}
			disk[root.Kind] = scan
		}
		_ = filepath.WalkDir(root.Path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
				return nil
			}
			info, err := entry.Info()
			if err != nil || info.ModTime().Before(since) {
				return nil
			}
			units := diskUnits(t, path)
			scan.files++
			scan.bytes += info.Size()
			scan.units += units
			if row, ok := byPath[path]; ok {
				if row.Status == transcriptindex.StatusOwned || root.RunID != "" {
					scan.capturedUnits += units
				} else if row.Status == transcriptindex.StatusIndexed || row.Status == transcriptindex.StatusPending {
					reported := row.Turns
					if row.Harness == transcriptindex.HarnessClaude {
						reported = row.UsageReports
					}
					scan.indexedUnits += reported
					scan.capturedUnits += reported
				}
			}
			return nil
		})
	}
	var totalUnits, totalCaptured int64
	for _, root := range coverage.Roots {
		scan := disk[root.Kind]
		if scan == nil {
			continue
		}
		bytesShare := float64(scan.bytes)
		if bytesShare > 0 {
			bytesShare = float64(root.IndexedBytes) / bytesShare
		}
		unitShare := 1.0
		if scan.units > 0 {
			unitShare = float64(scan.capturedUnits) / float64(scan.units)
		}
		totalUnits += scan.units
		totalCaptured += scan.capturedUnits
		t.Logf("%-24s coverage: files %d bytes %d captured %.2f%% | disk scan: files %d bytes %d (Δ %+.2f pts) turns on disk %d, captured %d (%.2f%%)",
			root.Kind, root.Files, root.Bytes, root.CapturedShare()*100, scan.files, scan.bytes, (root.CapturedShare()-bytesShare)*100, scan.units, scan.capturedUnits, unitShare*100)
		if root.Files != int(scan.files) {
			t.Errorf("%s: coverage counts %d files, the disk scan %d", root.Kind, root.Files, scan.files)
		}
	}
	share := float64(totalCaptured) / float64(max(totalUnits, 1))
	t.Logf("total: %.2f%% of %d trailing-7-day turns captured (coverage byte share %.2f%%)", share*100, totalUnits, coverage.Totals.CapturedShare()*100)
}

var (
	codexTurn    = []byte(`"type":"turn_context"`)
	claudeUsage  = []byte(`"usage":{`)
	claudeAssist = []byte(`"type":"assistant"`)
)

// diskUnits counts provider turns in a transcript without the indexer: Codex
// turn_context records, or Claude assistant lines that carry usage.
func diskUnits(t *testing.T, path string) int64 {
	handle, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer handle.Close()
	reader := bufio.NewReaderSize(handle, 1<<20)
	var units int64
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 && (bytes.Contains(line[:min(len(line), 256)], codexTurn) || (bytes.Contains(line, claudeAssist) && bytes.Contains(line, claudeUsage))) {
			units++
		}
		if err != nil {
			return units
		}
	}
}
