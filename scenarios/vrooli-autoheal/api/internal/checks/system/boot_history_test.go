package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
)

// We cheat: instead of reimplementing journal arg parsing, we use the
// MockExecutor and craft the JSON output journal.Reader expects.
func newBootHistoryWithMock(avail bool, boots []journal.BootRecord, bootErr error, logsByBoot map[string]string) *BootHistoryCheck {
	mock := testutil.NewMockExecutor()
	if avail {
		mock.Responses["journalctl --version"] = testutil.MockResponse{Output: []byte("systemd 254\n")}
	} else {
		mock.Responses["journalctl --version"] = testutil.MockResponse{Error: errors.New("not found")}
	}

	if bootErr != nil {
		mock.Responses["journalctl --list-boots --no-pager -o json"] = testutil.MockResponse{Error: bootErr}
		mock.Responses["journalctl --list-boots --no-pager"] = testutil.MockResponse{Error: bootErr}
	} else if boots != nil {
		mock.Responses["journalctl --list-boots --no-pager -o json"] = testutil.MockResponse{
			Output: encodeBootsJSON(boots),
		}
	}

	mock.DefaultResponse = testutil.MockResponse{Output: []byte("")}
	for bootID, raw := range logsByBoot {
		mock.Responses["journalctl --no-pager -o json -b "+bootID+" -n 200 -r"] = testutil.MockResponse{
			Output: []byte(raw),
		}
	}

	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	return NewBootHistoryCheck(
		WithBootHistoryReader(journal.NewReader(mock)),
		WithBootHistoryNow(func() time.Time { return now }),
	)
}

func encodeBootsJSON(boots []journal.BootRecord) []byte {
	var b []byte
	b = append(b, '[')
	for i, br := range boots {
		if i > 0 {
			b = append(b, ',')
		}
		us := br.LastEntry.UnixMicro()
		fus := br.FirstEntry.UnixMicro()
		b = append(b, []byte(`{"index":`)...)
		b = append(b, []byte(itoa(br.Index))...)
		b = append(b, []byte(`,"boot_id":"`+br.BootID+`","first_entry":`+itoa64(fus)+`,"last_entry":`+itoa64(us)+`}`)...)
	}
	b = append(b, ']')
	return b
}

func itoa(n int) string     { return time.Unix(int64(n), 0).Format("") + intStr(int64(n)) }
func itoa64(n int64) string { return intStr(n) }
func intStr(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func TestBootHistoryJournalUnavailable(t *testing.T) {
	c := newBootHistoryWithMock(false, nil, nil, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING when journalctl unavailable", r.Status)
	}
	if r.Details["journalAvailable"] != false {
		t.Errorf("journalAvailable = %v", r.Details["journalAvailable"])
	}
}

func TestBootHistoryListBootsError(t *testing.T) {
	c := newBootHistoryWithMock(true, nil, errors.New("permission denied"), nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING on list error", r.Status)
	}
}

func TestBootHistoryOnlyCurrentBoot(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	c := newBootHistoryWithMock(true, []journal.BootRecord{
		{Index: 0, BootID: "current", LastEntry: now},
	}, nil, nil)
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK with only current boot", r.Status)
	}
}

func TestBootHistoryAllClean(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	cleanLogs := `{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"Reached target Shutdown"}` + "\n"
	c := newBootHistoryWithMock(true, []journal.BootRecord{
		{Index: -1, BootID: "previous", LastEntry: now.Add(-2 * time.Hour)},
		{Index: 0, BootID: "current", LastEntry: now},
	}, nil, map[string]string{
		"previous": cleanLogs,
	})
	r := c.Run(context.Background())
	if r.Status != checks.StatusOK {
		t.Errorf("Status = %s, want OK when shutdown markers present", r.Status)
	}
}

func TestBootHistorySingleUnclean(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	uncleanLogs := `{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"systemd[1]: Started Daily apt"}` + "\n"
	c := newBootHistoryWithMock(true, []journal.BootRecord{
		{Index: -1, BootID: "previous", LastEntry: now.Add(-3 * time.Hour)},
		{Index: 0, BootID: "current", LastEntry: now},
	}, nil, map[string]string{
		"previous": uncleanLogs,
	})
	r := c.Run(context.Background())
	if r.Status != checks.StatusWarning {
		t.Errorf("Status = %s, want WARNING for one unclean shutdown", r.Status)
	}
	if got := r.Details["uncleanBootsRecent24h"]; got != 1 {
		t.Errorf("uncleanBootsRecent24h = %v, want 1", got)
	}
	if got := r.Details["latestUncleanBootId"]; got != "previous" {
		t.Errorf("latestUncleanBootId = %v, want previous", got)
	}
}

func TestBootHistoryMultipleUncleanCritical(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	uncleanLogs := `{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"random non-shutdown message"}` + "\n"
	c := newBootHistoryWithMock(true, []journal.BootRecord{
		{Index: -2, BootID: "two-back", LastEntry: now.Add(-5 * time.Hour)},
		{Index: -1, BootID: "previous", LastEntry: now.Add(-2 * time.Hour)},
		{Index: 0, BootID: "current", LastEntry: now},
	}, nil, map[string]string{
		"two-back": uncleanLogs,
		"previous": uncleanLogs,
	})
	r := c.Run(context.Background())
	if r.Status != checks.StatusCritical {
		t.Errorf("Status = %s, want CRITICAL for >=2 unclean in 24h", r.Status)
	}
	if got := r.Details["latestUncleanBootId"]; got != "previous" {
		t.Errorf("latestUncleanBootId = %v, want previous", got)
	}
}

func TestBootHistoryMetadata(t *testing.T) {
	c := NewBootHistoryCheck()
	if c.ID() != "system-boot-history" {
		t.Errorf("ID = %s", c.ID())
	}
}
