package facts

import (
	"context"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// FileMetricsProvider serves lexical file facts from Code Facts' authoritative
// source-file catalog. Implementations must fence a requested target to the
// active catalog generation and mark unreadable or unsupported files honestly.
type FileMetricsProvider interface {
	DescribeFileMetrics(context.Context, *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error)
}
