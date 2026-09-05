package conformance

import (
	"fmt"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Violation is a typed contract failure suitable for API and test reporting.
type Violation struct {
	Path   string
	Reason string
}

func (v Violation) Error() string { return v.Path + ": " + v.Reason }

// Validate checks the semantic requirements shared by all evidence producers.
// The generated protobuf descriptors enforce wire-level rules; this harness
// enforces cross-field and producer-facing rules that descriptors cannot express.
func Validate(verdict *commonv1.TargetVerdict) []Violation {
	if verdict == nil {
		return []Violation{{Path: "verdict", Reason: "required"}}
	}
	var violations []Violation
	if verdict.Target == nil {
		return []Violation{{Path: "target", Reason: "required"}}
	}
	t := verdict.Target
	for path, value := range map[string]string{"target.ramp": t.Ramp, "target.platform": t.Platform, "target.os": t.Os, "run_id": verdict.RunId} {
		if strings.TrimSpace(value) == "" {
			violations = append(violations, Violation{Path: path, Reason: "must not be blank"})
		}
	}
	if t.DeviceKind == commonv1.DeviceKind_DEVICE_KIND_UNSPECIFIED {
		violations = append(violations, Violation{Path: "target.device_kind", Reason: "must be specified"})
	}
	if verdict.Disposition == commonv1.Disposition_DISPOSITION_UNSPECIFIED {
		violations = append(violations, Violation{Path: "disposition", Reason: "must be specified"})
	}
	if t.DeviceKind != commonv1.DeviceKind_DEVICE_KIND_HOST && t.DeviceKind != commonv1.DeviceKind_DEVICE_KIND_PHYSICAL && t.BridgeNodeId != nil {
		violations = append(violations, Violation{Path: "target.bridge_node_id", Reason: "only host or physical evidence may identify a bridge node"})
	}
	for i, ref := range verdict.Refs {
		if ref == nil {
			violations = append(violations, Violation{Path: fmt.Sprintf("refs[%d]", i), Reason: "must not be nil"})
			continue
		}
		for path, value := range map[string]string{fmt.Sprintf("refs[%d].producer", i): ref.Producer, fmt.Sprintf("refs[%d].artifact_id", i): ref.ArtifactId, fmt.Sprintf("refs[%d].kind", i): ref.Kind, fmt.Sprintf("refs[%d].checksum", i): ref.Checksum} {
			if strings.TrimSpace(value) == "" {
				violations = append(violations, Violation{Path: path, Reason: "must not be blank"})
			}
		}
		if ref.SizeBytes < 0 {
			violations = append(violations, Violation{Path: fmt.Sprintf("refs[%d].size_bytes", i), Reason: "must be non-negative"})
		}
		if ref.CreatedAt == nil {
			violations = append(violations, Violation{Path: fmt.Sprintf("refs[%d].created_at", i), Reason: "required"})
		}
	}
	return violations
}
