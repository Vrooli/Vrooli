package sources

import (
	"context"
	"fmt"
	"path/filepath"
)

// Assumed resource-qdrant CLI surface (reconcile with actual resource CLI):
//
//	resource-qdrant snapshot create --collection <name> --output <file>
//	  Triggers Qdrant's built-in snapshot mechanism for <collection> and
//	  downloads the resulting snapshot to <file>. Credentials / host are
//	  self-sourced by the resource CLI from vault. No secrets in argv.
//
//	resource-qdrant snapshot restore --collection <name> --input <file>
//	  Uploads <file> as a snapshot and recovers it into <collection>.
//	  The collection must already exist or the resource CLI creates it.
//	  Credentials are self-sourced.

const (
	qdrantBinary         = "resource-qdrant"
	qdrantSubSnapshot    = "snapshot"
	qdrantSubCreate      = "create"
	qdrantSubRestore     = "restore"
	qdrantFlagCollection = "--collection"
	qdrantFlagOutput     = "--output"
	qdrantFlagInput      = "--input"
	qdrantSnapshotFile   = "snapshot.qdrant"
)

// qdrantCapturer captures a Qdrant collection via the resource-qdrant CLI,
// which uses Qdrant's native snapshot API for a consistent point-in-time copy.
type qdrantCapturer struct {
	runner CommandRunner
}

// Compile-time guarantee.
var _ Capturer = (*qdrantCapturer)(nil)

func newQdrantCapturer(r CommandRunner) *qdrantCapturer {
	return &qdrantCapturer{runner: r}
}

func (c *qdrantCapturer) Kind() SourceKind { return KindQdrant }

// Capture runs:
//
//	resource-qdrant snapshot create --collection <spec.Locator> --output <StageDir>/snapshot.qdrant
//
// spec.Locator is the Qdrant collection name. No secrets appear in argv.
func (c *qdrantCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, qdrantSnapshotFile)
	_, err := c.runner.Run(ctx, qdrantBinary,
		qdrantSubSnapshot, qdrantSubCreate,
		qdrantFlagCollection, spec.Locator,
		qdrantFlagOutput, dst,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("qdrant capture collection=%q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst}, nil
}

// Restore runs:
//
//	resource-qdrant snapshot restore --collection <spec.Target> --input <spec.ArtifactPath>
//
// spec.Target is the collection to restore into. No secrets appear in argv.
func (c *qdrantCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	_, err := c.runner.Run(ctx, qdrantBinary,
		qdrantSubSnapshot, qdrantSubRestore,
		qdrantFlagCollection, spec.Target,
		qdrantFlagInput, spec.ArtifactPath,
	)
	if err != nil {
		return fmt.Errorf("qdrant restore collection=%q: %w", spec.Target, err)
	}
	return nil
}
