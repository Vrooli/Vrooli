package sources

import (
	"context"
	"fmt"
	"path/filepath"
)

// Assumed resource-minio CLI surface (reconcile with actual resource CLI):
//
//	resource-minio mirror --source <bucket/prefix> --dest <local-dir>
//	  Mirrors the contents of <bucket/prefix> from the configured MinIO /
//	  S3-compatible store into <local-dir>, creating it if necessary.
//	  Credentials are self-sourced by the resource CLI from vault.
//	  No secrets appear in argv.
//
//	resource-minio mirror --source <local-dir> --dest <bucket/prefix>
//	  Mirrors a local directory back into the object store at <bucket/prefix>.
//	  Credentials are self-sourced. No secrets in argv.

const (
	minioBinary    = "resource-minio"
	minioSubMirror = "mirror"
	minioFlagSrc   = "--source"
	minioFlagDst   = "--dest"
	minioMirrorDir = "mirror"
)

// objectStorageCapturer captures a bucket/prefix from MinIO (S3-compatible)
// via the resource-minio CLI. Both Capture and Restore use mirror semantics:
// the local copy and the remote bucket converge to the same set of objects.
type objectStorageCapturer struct {
	runner CommandRunner
}

// Compile-time guarantee.
var _ Capturer = (*objectStorageCapturer)(nil)

func newObjectStorageCapturer(r CommandRunner) *objectStorageCapturer {
	return &objectStorageCapturer{runner: r}
}

func (c *objectStorageCapturer) Kind() SourceKind { return KindObjectStorage }

// Capture runs:
//
//	resource-minio mirror --source <spec.Locator> --dest <StageDir>/mirror
//
// spec.Locator is a bucket or bucket/prefix path. No secrets appear in argv.
func (c *objectStorageCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, minioMirrorDir)
	_, err := c.runner.Run(ctx, minioBinary,
		minioSubMirror,
		minioFlagSrc, spec.Locator,
		minioFlagDst, dst,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("object capture %q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst}, nil
}

// Restore runs:
//
//	resource-minio mirror --source <spec.ArtifactPath> --dest <spec.Target>
//
// spec.Target is the destination bucket/prefix. No secrets appear in argv.
func (c *objectStorageCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	_, err := c.runner.Run(ctx, minioBinary,
		minioSubMirror,
		minioFlagSrc, spec.ArtifactPath,
		minioFlagDst, spec.Target,
	)
	if err != nil {
		return fmt.Errorf("object restore %q: %w", spec.Target, err)
	}
	return nil
}
