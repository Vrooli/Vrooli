package forensics

import (
	"context"
	"errors"
	"strings"
	"testing"

	"system-monitor-api/internal/services/journal"
)

// journalMockExec implements journal.CommandExecutor for tests of boot
// history (the journal.Reader is the real one, fed by this mock).
type journalMockExec struct {
	responses map[string][]byte
	errors    map[string]error
	def       []byte
	defErr    error
}

func (m *journalMockExec) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if e, ok := m.errors[key]; ok {
		return nil, e
	}
	if r, ok := m.responses[key]; ok {
		return r, nil
	}
	return m.def, m.defErr
}

func TestBootHistoryNoJournal(t *testing.T) {
	s := NewService(nil, nil, FileSystem{}, fixedNow)
	env := s.BootHistory(context.Background())
	if env.Available {
		t.Fatal("expected not available")
	}
}

func TestBootHistoryListBootsFails(t *testing.T) {
	exec := &journalMockExec{defErr: errors.New("boom")}
	r := journal.NewReader(exec)
	s := NewService(r, nil, FileSystem{}, fixedNow)
	env := s.BootHistory(context.Background())
	if env.Available {
		t.Fatal("expected not available")
	}
	if !strings.Contains(env.Reason, "list-boots") {
		t.Errorf("reason: %q", env.Reason)
	}
}

func TestBootHistoryClassifiesBoots(t *testing.T) {
	exec := &journalMockExec{
		responses: map[string][]byte{
			"journalctl --list-boots --no-pager -o json": []byte(
				`[{"index":-1,"boot_id":"prevboot","first_entry":1,"last_entry":2},` +
					`{"index":0,"boot_id":"currboot","first_entry":3,"last_entry":4}]`,
			),
			// previous boot has shutdown marker
			"journalctl --no-pager -o json -b prevboot -g Reached target.*Shutdown|systemd-shutdown|Shutdown initiated|Power-Off|Reboot -n 5 -r": []byte(
				`{"__REALTIME_TIMESTAMP":"2000000","MESSAGE":"Reached target Shutdown."}` + "\n",
			),
		},
	}
	r := journal.NewReader(exec)
	s := NewService(r, nil, FileSystem{}, fixedNow)
	env := s.BootHistory(context.Background())
	if !env.Available {
		t.Fatalf("expected available, reason=%q", env.Reason)
	}
	report := env.Data.(BootHistoryReport)
	if len(report.Boots) != 2 {
		t.Fatalf("got %d boots, want 2", len(report.Boots))
	}
	// current boot always clean
	if !report.Boots[1].Clean {
		t.Error("current boot should be clean")
	}
	if !report.Boots[0].Clean {
		t.Errorf("previous boot should be clean (had shutdown marker): %+v", report.Boots[0])
	}
}

func TestBootHistoryFlagsUncleanBoot(t *testing.T) {
	exec := &journalMockExec{
		responses: map[string][]byte{
			"journalctl --list-boots --no-pager -o json": []byte(
				`[{"index":-1,"boot_id":"prevboot","first_entry":1,"last_entry":2},` +
					`{"index":0,"boot_id":"currboot","first_entry":3,"last_entry":4}]`,
			),
			// previous boot returns no shutdown markers
			"journalctl --no-pager -o json -b prevboot -g Reached target.*Shutdown|systemd-shutdown|Shutdown initiated|Power-Off|Reboot -n 5 -r": []byte(""),
		},
	}
	r := journal.NewReader(exec)
	s := NewService(r, nil, FileSystem{}, fixedNow)
	env := s.BootHistory(context.Background())
	if !env.Available {
		t.Fatal("expected available")
	}
	report := env.Data.(BootHistoryReport)
	if report.Boots[0].Clean {
		t.Errorf("previous boot should be unclean: %+v", report.Boots[0])
	}
	if report.Boots[0].Reason == "" {
		t.Error("expected reason for unclean boot")
	}
}
