package journal

import _ "embed"

//go:embed schema.sql
var schema string

//go:embed cursor_schema.sql
var cursorSchema string

const seedHighWaterMark = `
INSERT OR IGNORE INTO journal_high_water_mark(id,max_rowid,recorded_at)
SELECT 1,COALESCE(MAX(position),0),CURRENT_TIMESTAMP FROM journal_entry_positions;
`

func Schema() string { return schema + "\n" + cursorSchema + seedHighWaterMark }
