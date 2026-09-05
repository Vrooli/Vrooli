package delivery

import (
	"strings"
	"testing"
)

func TestSchemaOwnsDeliveryRuntimeTables(t *testing.T) {
	sql := strings.ToLower(Schema())
	for _, table := range []string{"download_apps", "download_assets", "download_artifacts", "download_storage_settings"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}
