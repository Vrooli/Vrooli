package runs

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestTargetKindMapsEveryDescriptorSchemaKind(t *testing.T) {
	for _, kind := range []string{
		"scenario", "asset", "resource", "tool", "safeguard", "team", "package", "control-plane", "docs", "project",
	} {
		if got := targetKind(kind); got == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED {
			t.Errorf("targetKind(%q) returned UNSPECIFIED", kind)
		}
	}
}
