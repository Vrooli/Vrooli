package foliage

import "testing"

// [REQ:REQ-P0-007] Foliage CLI renders weather values in a stable, readable order.
func TestWeatherRows(t *testing.T) {
	rows := weatherRows(map[string]interface{}{
		"humidity_percent": 65.0,
		"region_id":        7.0,
		"extra":            "from-provider",
		"date":             "2025-10-12",
	})

	if len(rows) != 4 {
		t.Fatalf("weatherRows length = %d, want 4: %#v", len(rows), rows)
	}
	if rows[0] != "region_id: 7" || rows[1] != "date: 2025-10-12" {
		t.Fatalf("weatherRows did not preserve priority order: %#v", rows)
	}
	if rows[3] != "extra: from-provider" {
		t.Fatalf("weatherRows did not append extra fields last: %#v", rows)
	}
}

func TestWeatherRowsEmpty(t *testing.T) {
	rows := weatherRows(nil)
	if len(rows) != 1 || rows[0] != "(no weather data)" {
		t.Fatalf("weatherRows(nil) = %#v", rows)
	}
}
