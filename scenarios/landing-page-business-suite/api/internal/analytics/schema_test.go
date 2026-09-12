package analytics

import (
	"strings"
	"testing"
)

func TestSchemaOwnsMetricsEvents(t *testing.T) {
	if !strings.Contains(strings.ToLower(Schema()), "create table if not exists metrics_events") {
		t.Fatal("analytics schema must declare metrics_events")
	}
}
