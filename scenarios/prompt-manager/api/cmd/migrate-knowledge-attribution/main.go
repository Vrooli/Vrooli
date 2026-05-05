// Migrate-knowledge-attribution rewrites every pre-Pillar-3 knowledge entry
// to the post-cutoff KnowledgeEntry shape (Caller, CallerNote, Attribution)
// and stamps each team.json with `attributionValidFrom`. The tool is a
// one-time migration: see docs/agent-system/RUNTIME_ATTRIBUTION.md for the
// contract.
//
// Usage (run from anywhere; --root must point at the prompt-manager store):
//
//	go run ./cmd/migrate-knowledge-attribution \
//	    --root=../store \
//	    --cutoff-date=2026-05-04
//
// Flags:
//
//	--root          Path to the store directory containing teams/<id>/...
//	                (required; no implicit current-dir default).
//	--cutoff-date   ISO YYYY-MM-DD value to write into team.json's
//	                attributionValidFrom. Defaults to today (UTC). Override
//	                only for testing.
//	--dry-run       Report what would change; do not write any files.
//
// Greenfield contract: idempotent — re-running on a fully migrated tree is a
// no-op (no .backup file written, no diff produced). The tool refuses to
// run if --root does not exist or doesn't look like a store directory.
//
// See migrate.go for the testable core; this file is a thin CLI wrapper.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	root := flag.String("root", "", "Path to the prompt-manager store directory (required)")
	cutoffDate := flag.String("cutoff-date", "", "ISO YYYY-MM-DD; defaults to today (UTC)")
	dryRun := flag.Bool("dry-run", false, "Report changes without writing")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		log.Fatalf("--root is required (point at scenarios/prompt-manager/store)")
	}
	resolvedRoot, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("resolve --root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(resolvedRoot, "teams")); err != nil {
		log.Fatalf("--root=%s does not contain a teams/ subdirectory: %v", resolvedRoot, err)
	}

	cutoff := strings.TrimSpace(*cutoffDate)
	if cutoff == "" {
		cutoff = time.Now().UTC().Format("2006-01-02")
	}
	if !isISODate(cutoff) {
		log.Fatalf("--cutoff-date must be YYYY-MM-DD; got %q", cutoff)
	}

	stats, err := migrateStore(migrateOpts{
		StoreRoot:  resolvedRoot,
		CutoffDate: cutoff,
		DryRun:     *dryRun,
	})
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}

	mode := "applied"
	if *dryRun {
		mode = "dry-run"
	}
	fmt.Printf("migrate-knowledge-attribution (%s) — root=%s cutoff=%s\n", mode, resolvedRoot, cutoff)
	fmt.Printf("  teams scanned:     %d\n", stats.TeamsScanned)
	fmt.Printf("  team.json updated: %d (skipped %d already-migrated)\n", stats.TeamsMigrated, stats.TeamsSkipped)
	fmt.Printf("  knowledge entries: %d scanned, %d migrated, %d already-migrated\n",
		stats.EntriesScanned, stats.EntriesMigrated, stats.EntriesSkipped)
	fmt.Printf("  .backup files:     %d\n", stats.BackupsWritten)
}

// isISODate reports whether s is a YYYY-MM-DD date that round-trips through
// time.Parse without rejection. Empty strings return false.
func isISODate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
