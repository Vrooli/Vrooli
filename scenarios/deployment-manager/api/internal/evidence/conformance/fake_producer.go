package conformance

import (
	"fmt"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FakeProducer is a reference producer used by conformance tests and examples.
// It emits references only; it never embeds artifact bytes or paths.
type FakeProducer struct{ Producer string }

func (p FakeProducer) Verdict(target *commonv1.EvidenceTarget, disposition commonv1.Disposition, runID string) (*commonv1.TargetVerdict, error) {
	if p.Producer == "" {
		return nil, fmt.Errorf("producer is required")
	}
	return &commonv1.TargetVerdict{
		Target: target, Disposition: disposition, RunId: runID,
		Refs: []*commonv1.EvidenceRef{{Producer: p.Producer, ArtifactId: "artifact-1", Kind: "video/mp4", Checksum: "sha256:test", SizeBytes: 1, CreatedAt: timestamppb.New(time.Now().UTC())}},
	}, nil
}
