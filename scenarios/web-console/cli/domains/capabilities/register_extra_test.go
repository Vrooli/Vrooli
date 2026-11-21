package capabilities

import (
	"testing"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
)

func TestCapabilityRows(t *testing.T) {
	rows := capabilityRows([]*capabilitiesv1.CapabilityState{{Id: "tmux", Status: "available", Name: "tmux", Message: "ready"}})
	if len(rows) != 1 {
		t.Fatalf("capability rows = %#v", rows)
	}
}
