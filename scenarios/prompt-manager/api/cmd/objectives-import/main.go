// Command objectives-import applies the one-way migration from the operator's
// authored objective declarations (docs/director-swarm/strategy/OBJECTIVES.md
// plus each team.json `objectivesServed` block) into the Prompt Manager
// objective authority.
//
// It is the recoverable cutover tool for plan
// aquila-objective-authority-and-team-editor Phase 3: it previews the retained
// declarations without mutating anything, and with --apply it hands the reviewed
// preview to the single objective Service, which stores a raw-source snapshot,
// applies objectives and attachments, recomputes team attachment revisions and
// writes a source-digest-keyed migration receipt. Re-running the same source is
// a no-op.
//
// The command opens the explicit database path named by --db rather than
// resolving one from the lifecycle environment. A standalone migration must not
// guess a scenario's path from ambient environment: the same identity guard that
// keeps a child from opening a supervisor's database means an unqualified
// process resolves a private class-rooted file, not the running server's.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	"prompt-manager/internal/objectives"

	// Registers the modernc SQLite driver so the migration opens the same
	// engine the scenario API uses without importing package main.
	_ "modernc.org/sqlite"
)

func main() {
	repoRoot := flag.String("repo-root", "../../..", "repository root containing docs/director-swarm/strategy/OBJECTIVES.md")
	storeDir := flag.String("store-dir", "", "prompt-manager store directory containing teams/ (default <repo-root>/scenarios/prompt-manager/store)")
	dbPath := flag.String("db", "../data/prompt-manager.db", "explicit path to the prompt-manager SQLite database")
	apply := flag.Bool("apply", false, "apply the reviewed import; without it only the preview is printed")
	reimport := flag.Bool("reimport", false, "with --apply, reconcile an already-applied digest by clearing its receipt first; the stored snapshot is preserved")
	inspect := flag.Bool("inspect", false, "open the database read-only and print the persisted authority state")
	project := flag.Bool("project", false, "render the read-only authority projection and the declaration-drift report (never writes the statement)")
	coverage := flag.Bool("coverage", false, "render the authority-backed objective coverage read (Phase 3 step 6); reads only")
	asJSON := flag.Bool("json", false, "print machine-readable JSON")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root, err := filepath.Abs(*repoRoot)
	fatal(err, "resolve repo root")
	store := *storeDir
	if store == "" {
		store = filepath.Join(root, "scenarios", "prompt-manager", "store")
	}
	dbAbs, err := filepath.Abs(*dbPath)
	fatal(err, "resolve database path")

	switch {
	case *inspect:
		runInspect(ctx, dbAbs, *asJSON)
	case *project:
		runProject(ctx, root, store, dbAbs, *asJSON)
	case *coverage:
		runCoverage(ctx, root, store, dbAbs, *asJSON)
	case *apply:
		runApply(ctx, root, store, dbAbs, *reimport, *asJSON)
	default:
		runPreview(root, store, *asJSON)
	}
}

func runPreview(repoRoot, storeDir string, asJSON bool) {
	preview, err := objectives.PreviewImportFromRepo(repoRoot, storeDir)
	fatal(err, "build preview")
	if asJSON {
		emit(preview)
		return
	}
	printPreviewSummary(preview)
}

func runApply(ctx context.Context, repoRoot, storeDir, dbPath string, reimport, asJSON bool) {
	preview, err := objectives.PreviewImportFromRepo(repoRoot, storeDir)
	fatal(err, "build preview")
	if !asJSON {
		printPreviewSummary(preview)
	}

	db, err := openReadWrite(ctx, dbPath)
	fatal(err, "open database")
	defer db.Close()
	fatal(database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(objectives.Schema)), "ensure objectives schema")

	if reimport {
		if _, err := db.ExecContext(ctx, `DELETE FROM objective_import_receipts WHERE source_digest = ?`, preview.Source.Digest); err != nil {
			fatal(err, "clear prior receipt")
		}
		fmt.Fprintf(os.Stderr, "reimport: cleared prior receipt for %s; the stored snapshot is preserved and the import will reconcile\n", preview.Source.Digest)
	}

	service := objectives.NewService(objectives.NewRepository(db), nil)
	receipt, err := service.Import(ctx, preview)
	fatal(err, "apply import")

	if asJSON {
		emit(receipt)
		return
	}
	fmt.Printf("applied=%v objectives=%d attachments=%d acksCarried=%d restatementPending=%d snapshotStored=%v digest=%s\n",
		!receipt.AlreadyApplied, receipt.ObjectivesImported, receipt.AttachmentsImported,
		receipt.AcknowledgementsCarried, receipt.RestatementPending, receipt.SnapshotStored, receipt.SourceDigest)
}

func runInspect(ctx context.Context, dbPath string, asJSON bool) {
	db, err := openReadWrite(ctx, dbPath)
	fatal(err, "open database for inspection")
	defer db.Close()

	service := objectives.NewService(objectives.NewRepository(db), nil)
	objs, err := service.ListObjectives(ctx)
	fatal(err, "list objectives")
	rels, err := service.ListRelations(ctx)
	fatal(err, "list relations")
	validation, err := service.Validate(ctx)
	fatal(err, "validate")
	receipts, err := listReceipts(ctx, db)
	fatal(err, "list receipts")

	type teamView struct {
		TeamID      string                  `json:"teamId"`
		Revision    string                  `json:"attachmentRevision"`
		Attachments []objectives.Attachment `json:"attachments"`
	}
	teamIDs, err := distinctTeamIDs(ctx, db)
	fatal(err, "list team ids")
	var views []teamView
	for _, teamID := range teamIDs {
		atts, err := service.ListTeamAttachments(ctx, teamID)
		fatal(err, "list team attachments")
		rev, err := service.TeamAttachmentRevision(ctx, teamID)
		fatal(err, "team revision")
		views = append(views, teamView{TeamID: teamID, Revision: rev, Attachments: atts})
	}

	out := map[string]any{
		"databasePath": dbPath,
		"objectives":   objs,
		"relations":    rels,
		"teams":        views,
		"validation":   validation,
		"receipts":     receipts,
	}
	if asJSON {
		emit(out)
		return
	}
	fmt.Printf("database=%s objectives=%d relations=%d teams=%d receipts=%d errors=%d warnings=%d\n",
		dbPath, len(objs), len(rels), len(views), len(receipts), validation.Errors, validation.Warnings)
	for _, o := range objs {
		fmt.Printf("  objective %s [%s] order=%d rev=%s\n", o.ID, o.Class, o.GlobalOrder, o.MeaningRevision)
	}
	for _, v := range views {
		for _, a := range v.Attachments {
			fmt.Printf("  attach team=%s obj=%s role=%s coverage=%s priority=%d pending=%v ack=%s\n",
				a.TeamID, a.ObjectiveID, a.Role, a.Coverage, a.Priority, a.RestatementPending, a.AcknowledgedRevision)
		}
	}
}

// runProject renders the read-only authority projection and reports drift
// against the retained declarations. It is the Phase 3 step 7 surface: it reads
// the statement and team.json, never writes them, and never creates an
// objectives.json (decision 02e4dd6b-2645-484c-b0e0-7c0ceb790d5d, D9/D10).
func runProject(ctx context.Context, repoRoot, storeDir, dbPath string, asJSON bool) {
	preview, err := objectives.PreviewImportFromRepo(repoRoot, storeDir)
	fatal(err, "build declaration preview")

	db, err := openReadWrite(ctx, dbPath)
	fatal(err, "open database for projection")
	defer db.Close()

	service := objectives.NewService(objectives.NewRepository(db), nil)
	projection, err := objectives.BuildProjection(ctx, service)
	fatal(err, "build projection")
	drift := objectives.CompareProjection(preview, projection)

	if asJSON {
		emit(map[string]any{
			"declarationDigest":  preview.Source.Digest,
			"projection":         projection,
			"drift":              drift,
			"projectionMarkdown": objectives.RenderProjection(projection),
			"driftMarkdown":      objectives.RenderDrift(drift),
		})
		return
	}
	fmt.Print(objectives.RenderProjection(projection))
	fmt.Println()
	fmt.Print(objectives.RenderDrift(drift))
	fmt.Printf("\ndeclaration_digest=%s has_drift=%v drift_items=%d\n", preview.Source.Digest, drift.HasDrift, len(drift.Items))
}

// runCoverage renders the authority-backed coverage read for Phase 3 step 6.
// It is the query surface the graph/CLI consumers resolve once the route
// migration is activated; it reads the authority and the retained declarations
// for drift context and writes neither.
func runCoverage(ctx context.Context, repoRoot, storeDir, dbPath string, asJSON bool) {
	db, err := openReadWrite(ctx, dbPath)
	fatal(err, "open database for coverage")
	defer db.Close()

	service := objectives.NewService(objectives.NewRepository(db), nil)
	cov, err := objectives.BuildCoverage(ctx, service, repoRoot, storeDir)
	fatal(err, "build coverage")

	if asJSON {
		emit(cov)
		return
	}
	fmt.Printf("Objective coverage from authority (%s)\n\n", cov.SourcePath)
	for _, row := range cov.Rows {
		status := "served"
		if !row.Served {
			status = "UNSERVED"
		}
		fmt.Printf("  %-4s %-12s %s\n", row.ID, row.Class, status)
	}
	if len(cov.UnattachedTeams) > 0 {
		fmt.Printf("\nTeams tracing to no objective: %s\n", strings.Join(cov.UnattachedTeams, ", "))
	}
	fmt.Printf("\n%d unserved (%d without a gap marker), %d error(s), %d warning(s), drift=%v (%d item(s))\n",
		cov.Unserved, cov.Undeclared, cov.Validation.Errors, cov.Validation.Warnings, cov.Drift.HasDrift, len(cov.Drift.Items))
}

// openReadWrite opens the explicit migration target with the canonical SQLite
// pragmas. MaxOpenConns=1 keeps the one-shot writer from opening more handles
// than the running server's single connection budget tolerates.
func openReadWrite(ctx context.Context, path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	return database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Logger:       func(format string, args ...any) { fmt.Fprintf(os.Stderr, "db: "+format+"\n", args...) },
	})
}

func listReceipts(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]json.RawMessage, error) {
	rows, err := db.QueryContext(ctx, `SELECT payload FROM objective_import_receipts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(payload))
	}
	return out, rows.Err()
}

func distinctTeamIDs(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT team_id FROM objective_attachments ORDER BY team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var teamID string
		if err := rows.Scan(&teamID); err != nil {
			return nil, err
		}
		out = append(out, teamID)
	}
	return out, rows.Err()
}

func printPreviewSummary(preview objectives.ImportPreview) {
	errors, warnings := 0, 0
	for _, c := range preview.Conflicts {
		if c.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}
	fmt.Printf("digest=%s files=%d objectives=%d attachments=%d coverageGaps=%d conflicts=%d (errors=%d warnings=%d)\n",
		preview.Source.Digest, len(preview.Source.Files), len(preview.Objectives), len(preview.Attachments),
		len(preview.CoverageGaps), len(preview.Conflicts), errors, warnings)
	for _, o := range preview.Objectives {
		fmt.Printf("  objective %s [%s] order=%d legacy=%s new=%s\n", o.ID, o.Class, o.GlobalOrder, o.LegacyMeaningRevision, o.NewMeaningRevision)
	}
	for _, c := range preview.Conflicts {
		fmt.Printf("  conflict [%s/%s] obj=%s team=%s: %s\n", c.Severity, c.Kind, c.ObjectiveID, c.TeamID, c.Detail)
	}
}

func emit(v any) {
	encoded, err := json.MarshalIndent(v, "", "  ")
	fatal(err, "encode json")
	fmt.Println(string(encoded))
}

func fatal(err error, what string) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "objectives-import: %s: %v\n", what, err)
	os.Exit(1)
}
