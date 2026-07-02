package probes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	planmodel "plan-manager/internal/planmodel"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

// fixtureRunner dispatches by probe binary + flags, mirroring the live CLIs.
func fixtureRunner(t *testing.T) Runner {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		argv := name + " " + strings.Join(args, " ")
		switch {
		case name == "prompt-manager" && strings.Contains(argv, "--type skill"):
			return fixture(t, "prompt-manager-discover-skill.json"), nil
		case name == "prompt-manager" && strings.Contains(argv, "--type all"):
			return fixture(t, "prompt-manager-discover-all.json"), nil
		case name == "search-hub":
			return fixture(t, "search-hub-query.json"), nil
		default:
			return nil, errors.New("unexpected probe " + argv)
		}
	}
}

// TestDiscoverParsesContractFixtures pins the external JSON contracts: captured
// live output from prompt-manager and search-hub parses into typed items with
// bare-slug skill targets (D6), verbatim action show-commands, doc targets, and
// record recall commands, scores passed through.
func TestDiscoverParsesContractFixtures(t *testing.T) {
	outcomes := Discover(context.Background(), fixtureRunner(t), []string{"plan authoring"}, "architectural", Options{})
	if len(outcomes) != 3 {
		t.Fatalf("expected 3 outcomes, got %d", len(outcomes))
	}
	byProbe := map[string]Outcome{}
	for _, o := range outcomes {
		if o.Degraded {
			t.Fatalf("probe %s unexpectedly degraded: %s", o.Probe, o.Detail)
		}
		byProbe[o.Probe] = o
	}

	skills := byProbe[ProbePromptManagerSkills]
	if len(skills.Items) == 0 {
		t.Fatal("no skill items parsed")
	}
	first := skills.Items[0]
	if first.Item.Kind != planmodel.RelevantContextSkill {
		t.Fatalf("expected skill kind, got %s", first.Item.Kind)
	}
	if first.Item.Target != "implementation-plan-authoring" {
		t.Fatalf("skill target must be the bare slug, got %q", first.Item.Target)
	}
	if first.Item.Command != "" || len(first.Item.Argv) != 0 {
		t.Fatalf("skill item must not carry an assembled command (D6): %q %v", first.Item.Command, first.Item.Argv)
	}
	if first.Score == 0 {
		t.Fatal("score must pass through")
	}
	if first.Origin != "search" {
		t.Fatalf("prompt-manager source must pass through as origin, got %q", first.Origin)
	}
	if first.SizeChars == 0 {
		t.Fatal("prompt-manager contentChars must pass through")
	}
	if len(first.Tags) == 0 {
		t.Fatal("prompt-manager tags must pass through")
	}
	if first.Title != "Implementation Plan Authoring" {
		t.Fatalf("prompt-manager name must pass through as title, got %q", first.Title)
	}
	if !strings.Contains(first.Snippet, "Author durable implementation plans") {
		t.Fatalf("prompt-manager description must pass through as snippet, got %q", first.Snippet)
	}

	actions := byProbe[ProbePromptManagerActions]
	var sawAction bool
	for _, it := range actions.Items {
		if it.Item.Kind != planmodel.RelevantContextCommand {
			continue
		}
		sawAction = true
		if !strings.HasPrefix(it.Item.Command, "prompt-manager action show ") {
			t.Fatalf("action command must come verbatim from prompt-manager output, got %q", it.Item.Command)
		}
	}
	if !sawAction {
		t.Fatal("no action items parsed from --type all fixture")
	}

	recall := byProbe[ProbeSearchHubRecall]
	kinds := map[planmodel.RelevantContextKind]int{}
	var sawDocWithTitle bool
	for _, it := range recall.Items {
		kinds[it.Item.Kind]++
		if it.Item.Kind == planmodel.RelevantContextCommand && !strings.HasPrefix(it.Item.Command, "swarm-manager records get --id rec-") {
			t.Fatalf("record recall command malformed: %q", it.Item.Command)
		}
		if it.Origin == "" {
			t.Fatalf("search-hub provider group must pass through as origin: %+v", it)
		}
		if it.Title != "" && it.Snippet != "" && it.Item.Kind == planmodel.RelevantContextDoc {
			sawDocWithTitle = true
		}
	}
	for _, kind := range []planmodel.RelevantContextKind{planmodel.RelevantContextSkill, planmodel.RelevantContextDoc, planmodel.RelevantContextCommand} {
		if kinds[kind] == 0 {
			t.Fatalf("search-hub fixture should yield %s items, got %v", kind, kinds)
		}
	}
	if !sawDocWithTitle {
		t.Fatal("search-hub doc title and snippet must both survive parsing")
	}
}

// TestDiscoverDegradesPerProbe covers the D5 failure modes independently:
// hanging (timeout), non-zero exit, garbage output, and a fully-down runner —
// every case returns a typed degraded outcome and never blocks or fabricates.
func TestDiscoverDegradesPerProbe(t *testing.T) {
	hangThenFail := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "prompt-manager":
			if strings.Contains(strings.Join(args, " "), "--type skill") {
				<-ctx.Done() // hang until the per-probe timeout fires
				return nil, ctx.Err()
			}
			return nil, errors.New("exit status 1")
		default:
			return []byte("not json {"), nil
		}
	}
	start := time.Now()
	outcomes := Discover(context.Background(), hangThenFail, []string{"concept"}, "", Options{Timeout: 50 * time.Millisecond})
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("discover must not block past the per-probe timeout, took %s", elapsed)
	}
	if len(outcomes) != 3 {
		t.Fatalf("expected 3 outcomes, got %d", len(outcomes))
	}
	details := map[string]string{}
	for _, o := range outcomes {
		if !o.Degraded {
			t.Fatalf("probe %s should be degraded", o.Probe)
		}
		if len(o.Items) != 0 {
			t.Fatalf("degraded probe %s must not fabricate items", o.Probe)
		}
		details[o.Probe] = o.Detail
	}
	if !strings.Contains(details[ProbePromptManagerSkills], "timed out") {
		t.Fatalf("hanging probe must report a timeout, got %q", details[ProbePromptManagerSkills])
	}
	if !strings.Contains(details[ProbePromptManagerActions], "exit status 1") {
		t.Fatalf("failing probe must carry the exec error, got %q", details[ProbePromptManagerActions])
	}
	if !strings.Contains(details[ProbeSearchHubRecall], "unparseable output") {
		t.Fatalf("garbage output must degrade as unparseable, got %q", details[ProbeSearchHubRecall])
	}

	// Fully down: nil runner degrades every probe and still returns.
	for _, o := range Discover(context.Background(), nil, []string{"concept"}, "", Options{}) {
		if !o.Degraded || o.Detail == "" {
			t.Fatalf("nil runner must degrade with a detail: %+v", o)
		}
	}
}

// TestDiscoverRunsProbesConcurrently proves the probes do not run serially: two
// concepts x three probes each sleeping ~40ms complete far faster than the
// serial sum.
func TestDiscoverRunsProbesConcurrently(t *testing.T) {
	slowRunner := func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		select {
		case <-time.After(40 * time.Millisecond):
			return []byte(`{"results":[]}`), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	start := time.Now()
	outcomes := Discover(context.Background(), slowRunner, []string{"a", "b"}, "", Options{Timeout: time.Second})
	elapsed := time.Since(start)
	if len(outcomes) != 6 {
		t.Fatalf("expected 6 outcomes, got %d", len(outcomes))
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("6 probes at 40ms each should overlap, took %s (serial would be ~240ms)", elapsed)
	}
}

func TestDiscoverCapsConcurrentProbeFanout(t *testing.T) {
	var inFlight int32
	var maxSeen int32
	slowRunner := func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			seen := atomic.LoadInt32(&maxSeen)
			if current <= seen || atomic.CompareAndSwapInt32(&maxSeen, seen, current) {
				break
			}
		}
		defer atomic.AddInt32(&inFlight, -1)
		select {
		case <-time.After(30 * time.Millisecond):
			return []byte(`{"results":[]}`), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	outcomes := Discover(context.Background(), slowRunner, []string{"a", "b", "c"}, "", Options{Timeout: time.Second, MaxConcurrent: 2})
	if len(outcomes) != 9 {
		t.Fatalf("expected 9 outcomes, got %d", len(outcomes))
	}
	if got := atomic.LoadInt32(&maxSeen); got > 2 {
		t.Fatalf("max in-flight probes = %d, want <= 2", got)
	}
}
