package control

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAuditFilterKeepsNewestMatchingRecords(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"records":[
		{"id":"old","device_id":"tv","actor":"a","created_at":"2026-11-06T10:00:00Z"},
		{"id":"legacy","device_id":"tv","actor":"a","created_at":"0001-01-01T00:00:00Z","timestamp":"2026-11-06T11:55:00Z"},
		{"id":"new","device_id":"tv","actor":"a","created_at":"2026-11-06T11:59:00Z"},
		{"id":"other-device","device_id":"phone","actor":"a","created_at":"2026-11-06T11:59:30Z"},
		{"id":"other-actor","device_id":"tv","actor":"b","created_at":"2026-11-06T11:59:40Z"}
	]}`)

	ids := func(f auditFilter) string {
		t.Helper()
		out, err := f.apply(body)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		var payload struct {
			Records []struct {
				ID string `json:"id"`
			} `json:"records"`
		}
		if err := json.Unmarshal(out, &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got := make([]string, 0, len(payload.Records))
		for _, record := range payload.Records {
			got = append(got, record.ID)
		}
		return strings.Join(got, ",")
	}

	filter, err := parseAuditFilter("tv", "a", "30m", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(filter); got != "new,legacy" {
		t.Fatalf("device+actor+since = %s, want new,legacy", got)
	}
	filter, err = parseAuditFilter("", "", "", "2", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(filter); got != "other-actor,other-device" {
		t.Fatalf("limit 2 = %s, want the two newest", got)
	}
	filter, err = parseAuditFilter("", "", "2026-11-06T11:59:00Z", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(filter); got != "other-actor,other-device,new" {
		t.Fatalf("since RFC 3339 = %s", got)
	}

	for _, bad := range [][2]string{{"since", "yesterday"}, {"limit", "0"}, {"limit", "x"}} {
		var err error
		if bad[0] == "since" {
			_, err = parseAuditFilter("", "", bad[1], "", now)
		} else {
			_, err = parseAuditFilter("", "", "", bad[1], now)
		}
		if err == nil {
			t.Errorf("--%s %s accepted", bad[0], bad[1])
		}
	}
	if (auditFilter{}).active() {
		t.Fatal("an empty filter must leave the unfiltered list untouched")
	}
}
