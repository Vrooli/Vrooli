package recovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/envkit-go"
)

func TestBrokerWalksTiersAndScrubsIdentity(t *testing.T) {
	var seen []string
	b, err := New("", func(_ context.Context, tier, _ string, child []string) error {
		seen = append(seen, tier)
		for _, value := range child {
			if len(value) >= len("VROOLI_AGENT_IDENTITY_TOKEN=") && value[:len("VROOLI_AGENT_IDENTITY_TOKEN=")] == "VROOLI_AGENT_IDENTITY_TOKEN=" {
				t.Fatal("identity token crossed recovery boundary")
			}
		}
		if tier != RecoveryTierThree {
			return errors.New("tier unavailable")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	record, err := b.Recover(context.Background(), Request{Scenario: "prompt-manager", Reason: "refusing", Requester: "autoheal"}, envkit.Env{"VROOLI_AGENT_IDENTITY_TOKEN=dead", "PATH=/bin"})
	if err != nil {
		t.Fatal(err)
	}
	if record.TierReached != RecoveryTierThree || len(seen) != 3 {
		t.Fatalf("record=%+v tiers=%v", record, seen)
	}
}

func TestBrokerRecordsExhaustionWithoutRunningTier(t *testing.T) {
	b, err := New("", func(context.Context, string, string, []string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	b.now = func() time.Time { return time.Unix(100, 0) }
	for i := 0; i < DefaultBudget; i++ {
		if _, err := b.Recover(context.Background(), Request{Scenario: "agent-manager", Requester: "test"}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Recover(context.Background(), Request{Scenario: "agent-manager", Requester: "test"}, nil); err == nil {
		t.Fatal("expected exhaustion")
	}
	records := b.Records()
	if len(records) != DefaultBudget+1 || records[len(records)-1].TierReached != RecoveryTierFour {
		t.Fatalf("records=%+v", records)
	}
}

func TestBrokerCanLoadRecordsWithoutTierRunner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	if err := os.WriteFile(path, []byte(`[{"id":"r1","scenario":"demo","tier_reached":"operator-escalation","budget_remaining":0,"outcome":"budget-exhausted"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := New(path, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := b.Records(); len(got) != 1 || got[0].ID != "r1" {
		t.Fatalf("records = %#v, want loaded record", got)
	}
	if _, err := b.Recover(context.Background(), Request{Scenario: "demo", Requester: "test"}, nil); err == nil || !strings.Contains(err.Error(), "tier runner") {
		t.Fatalf("Recover() error = %v, want missing runner error", err)
	}
}
