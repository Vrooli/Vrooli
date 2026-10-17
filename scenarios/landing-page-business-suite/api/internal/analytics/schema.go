// Package analytics owns storage declarations for conversion analytics.
package analytics

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the analytics-owned declarative SQL schema.
func Schema() string { return schema }

// IndexesSchema is applied after api-core reconciles additive columns. Keeping
// indexes separate prevents a pre-existing metrics_events table from failing
// on an index that names a column the reconciliation step is about to add.
func IndexesSchema() string {
	return `
CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_slug);
CREATE INDEX IF NOT EXISTS idx_metrics_events_type ON metrics_events(event_type);
CREATE INDEX IF NOT EXISTS idx_metrics_events_created ON metrics_events(created_at);
CREATE INDEX IF NOT EXISTS idx_metrics_events_session ON metrics_events(session_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_events_event_id ON metrics_events(event_id) WHERE event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_metrics_events_country ON metrics_events(country_code);
CREATE INDEX IF NOT EXISTS idx_metrics_events_referrer_kind ON metrics_events(referrer_kind);
CREATE INDEX IF NOT EXISTS idx_metrics_events_campaign ON metrics_events(utm_campaign);
CREATE INDEX IF NOT EXISTS idx_metrics_events_device ON metrics_events(device_class);
CREATE INDEX IF NOT EXISTS idx_metrics_events_created_type ON metrics_events(created_at, event_type);
`
}
