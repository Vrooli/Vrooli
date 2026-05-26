package sources

import (
	"context"
	"fmt"
	"path/filepath"
)

// Assumed resource-postgres CLI surface (reconcile with actual resource CLI):
//
//	resource-postgres dump   --database <name> --output <file>
//	  Dumps the named database to <file> using pg_dump internally.
//	  Credentials (host, port, user, password) are self-sourced by the
//	  resource CLI from its own configuration / vault integration.
//	  No secrets ever appear in argv.
//
//	resource-postgres restore --database <name> --input <file>
//	  Restores a previously dumped file into <name>. The target database
//	  must already exist or the resource CLI creates it; credentials are
//	  again self-sourced.

const (
	postgresBinary     = "resource-postgres"
	postgresSubDump    = "dump"
	postgresSubRestore = "restore"
	postgresFlagDB     = "--database"
	postgresFlagOutput = "--output"
	postgresFlagInput  = "--input"
	postgresDumpFile   = "dump.pgdump"
)

// postgresCapturer captures a PostgreSQL database by delegating to the
// resource-postgres CLI (which wraps pg_dump/pg_restore). Credentials are
// self-sourced by the resource CLI; this capturer never passes secrets in argv.
type postgresCapturer struct {
	runner CommandRunner
}

// Compile-time guarantee.
var _ Capturer = (*postgresCapturer)(nil)

func newPostgresCapturer(r CommandRunner) *postgresCapturer {
	return &postgresCapturer{runner: r}
}

func (c *postgresCapturer) Kind() SourceKind { return KindPostgres }

// Capture runs:
//
//	resource-postgres dump --database <spec.Locator> --output <StageDir>/dump.pgdump
//
// spec.Locator is the database (or schema) name; no secrets appear in argv.
func (c *postgresCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, postgresDumpFile)
	_, err := c.runner.Run(ctx, postgresBinary,
		postgresSubDump,
		postgresFlagDB, spec.Locator,
		postgresFlagOutput, dst,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("postgres capture %q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst}, nil
}

// Restore runs:
//
//	resource-postgres restore --database <spec.Target> --input <spec.ArtifactPath>
//
// spec.Target is the destination database name; no secrets appear in argv.
func (c *postgresCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	_, err := c.runner.Run(ctx, postgresBinary,
		postgresSubRestore,
		postgresFlagDB, spec.Target,
		postgresFlagInput, spec.ArtifactPath,
	)
	if err != nil {
		return fmt.Errorf("postgres restore %q: %w", spec.Target, err)
	}
	return nil
}
