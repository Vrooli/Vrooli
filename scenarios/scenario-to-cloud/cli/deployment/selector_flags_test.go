package deployment

import (
	"flag"
	"testing"
)

func TestSelectorFlagsRejectsTargetWithHostOrDomain(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	selector := registerSelectorFlags(fs)
	if err := fs.Parse([]string{"--target", "vrooli.com", "--host", "138.197.95.182"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	_, err := selector.toSelector()
	if err == nil {
		t.Fatal("expected an error when target and host are both set")
	}
}

func TestSelectorFlagsAcceptsTargetOnly(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	selector := registerSelectorFlags(fs)
	if err := fs.Parse([]string{"--target", "https://vrooli.com/health", "--scenario", "landing-page-business-suite"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	parsed, err := selector.toSelector()
	if err != nil {
		t.Fatalf("toSelector: %v", err)
	}
	if parsed.Target != "https://vrooli.com/health" {
		t.Fatalf("expected target to be preserved, got %q", parsed.Target)
	}
	if parsed.ScenarioID != "landing-page-business-suite" {
		t.Fatalf("expected scenario id, got %q", parsed.ScenarioID)
	}
}
