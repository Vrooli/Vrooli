package sources

import (
	"context"
	"fmt"
	"path/filepath"
)

// Assumed resource-redis CLI surface (reconcile with actual resource CLI):
//
//	resource-redis dump   --prefix <key-prefix> --output <file>
//	  Scans keys matching <key-prefix>* using SCAN and serialises each
//	  key's value via DUMP into <file>. Credentials are self-sourced.
//	  NOTE: This is best-effort, NOT a transactional point-in-time snapshot.
//	  Keys written between the SCAN and DUMP calls may not be captured;
//	  keys deleted during the scan may produce errors that are silently
//	  skipped. For a consistent snapshot, quiesce writers before capturing.
//
//	resource-redis restore --prefix <key-prefix> --input <file>
//	  RESTOREs key-value pairs from <file> into the live Redis instance,
//	  optionally scoped to <key-prefix>. Credentials are self-sourced.

const (
	redisBinary     = "resource-redis"
	redisSubDump    = "dump"
	redisSubRestore = "restore"
	redisFlagPrefix = "--prefix"
	redisFlagOutput = "--output"
	redisFlagInput  = "--input"
	redisDumpFile   = "dump.rdb"
)

// redisCapturer captures a Redis key-prefix via the resource-redis CLI.
// IMPORTANT: this is best-effort, not a transactional point-in-time snapshot.
// Quiesce writers before capturing for strong consistency.
type redisCapturer struct {
	runner CommandRunner
}

// Compile-time guarantee.
var _ Capturer = (*redisCapturer)(nil)

func newRedisCapturer(r CommandRunner) *redisCapturer {
	return &redisCapturer{runner: r}
}

func (c *redisCapturer) Kind() SourceKind { return KindRedis }

// Capture runs:
//
//	resource-redis dump --prefix <spec.Locator> --output <StageDir>/dump.rdb
//
// spec.Locator is the key prefix to SCAN. No secrets appear in argv.
// This is best-effort: keys modified during the scan may be inconsistent.
func (c *redisCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	dst := filepath.Join(spec.StageDir, redisDumpFile)
	_, err := c.runner.Run(ctx, redisBinary,
		redisSubDump,
		redisFlagPrefix, spec.Locator,
		redisFlagOutput, dst,
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("redis capture prefix=%q: %w", spec.Locator, err)
	}
	return Artifact{Path: dst}, nil
}

// Restore runs:
//
//	resource-redis restore --prefix <spec.Target> --input <spec.ArtifactPath>
//
// spec.Target is the key-prefix scope for the restore. No secrets in argv.
func (c *redisCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	_, err := c.runner.Run(ctx, redisBinary,
		redisSubRestore,
		redisFlagPrefix, spec.Target,
		redisFlagInput, spec.ArtifactPath,
	)
	if err != nil {
		return fmt.Errorf("redis restore prefix=%q: %w", spec.Target, err)
	}
	return nil
}
