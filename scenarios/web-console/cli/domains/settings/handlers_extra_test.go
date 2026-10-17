package settings

import (
	"testing"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
)

func TestDefaultsRows(t *testing.T) {
	if defaultsRows(nil) != nil {
		t.Fatal("nil defaults should have no rows")
	}
	rows := defaultsRows(&settingsv1.SessionDefaults{DefaultBackend: "persistent", DefaultPolicy: &settingsv1.ExpirationPolicy{Mode: "days", Duration: "7d"}})
	if len(rows) != 3 {
		t.Fatalf("rows = %#v", rows)
	}
}
