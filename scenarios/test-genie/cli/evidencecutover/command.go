// Package evidencecutover exposes the one confirmation-gated offline archive
// operation for rich evidence and the Test Genie SQLite store together.
package evidencecutover

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"test-genie/internal/cutover"
)

func Run(args []string) error { return run(args, os.Stdout) }

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evidence-cutover <plan|apply> --scenario-dir <path> --archive-dir <path> --database-path <path> --database-archive <path> [--confirm %s]", cutover.Confirmation)
	}
	fs := flag.NewFlagSet("evidence-cutover "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scenarioDir := fs.String("scenario-dir", "", "scenario directory containing coverage")
	archiveDir := fs.String("archive-dir", "", "new archive destination")
	databasePath := fs.String("database-path", "", "offline Test Genie SQLite database")
	databaseArchive := fs.String("database-archive", "", "new archive path for the SQLite database")
	confirm := fs.String("confirm", "", "required confirmation token for apply")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*scenarioDir) == "" || strings.TrimSpace(*archiveDir) == "" || strings.TrimSpace(*databasePath) == "" || strings.TrimSpace(*databaseArchive) == "" {
		return fmt.Errorf("scenario-dir, archive-dir, database-path, and database-archive are required")
	}
	plan, err := cutover.PlanOffline(*scenarioDir, *archiveDir, *databasePath, *databaseArchive)
	if err != nil {
		return err
	}
	switch args[0] {
	case "plan":
		_, err = fmt.Fprintf(out, "coverage=%s archive=%s files=%d bytes=%d digest=%s database=%s database_archive=%s database_bytes=%d database_digest=%s required_free_bytes=%d\n", plan.Evidence.CoverageRoot, plan.Evidence.ArchiveRoot, plan.Evidence.Files, plan.Evidence.Bytes, plan.Evidence.Digest, plan.Database.LivePath, plan.Database.ArchivePath, plan.Database.Bytes, plan.Database.Digest, plan.RequiredFreeBytes)
		return err
	case "apply":
		if err := cutover.ApplyOffline(plan, *confirm); err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "archived coverage to %s and database to %s; receipts written beside both archives\n", plan.Evidence.ArchiveRoot, plan.Database.ArchivePath)
		return err
	default:
		return fmt.Errorf("unknown evidence-cutover command %q", args[0])
	}
}
