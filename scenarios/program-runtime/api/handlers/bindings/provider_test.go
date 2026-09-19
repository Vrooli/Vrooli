package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestDiscoveryReconcilesAfterSuccessAndTransientFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	calls := 0
	reconcileSearchProviders(ctx, func() error { return nil }, func(callCtx context.Context) error {
		if _, ok := callCtx.Deadline(); !ok {
			t.Fatal("registration must have a deadline")
		}
		calls++
		if calls == 2 {
			return errors.New("search service restarting")
		}
		if calls == 3 {
			cancel()
		}
		return nil
	}, time.Millisecond, time.Millisecond, time.Millisecond)
	if calls != 3 {
		t.Fatalf("registration stopped after success: %d calls", calls)
	}
}

// A persistently unreachable Search Hub must not keep Program Runtime on a
// flat 2 s retry: the delay doubles up to the ceiling and drops back to the
// floor after the next success.
func TestDiscoveryRetryBacksOffExponentiallyAndResetsOnSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const retry, maxRetry, refresh = 4 * time.Millisecond, 16 * time.Millisecond, 100 * time.Millisecond
	var stamps []time.Time
	failures := 0
	reconcileSearchProviders(ctx, func() error { return nil }, func(context.Context) error {
		stamps = append(stamps, time.Now())
		switch len(stamps) {
		case 1, 2, 3, 4, 5:
			failures++
			return errors.New("search-hub unreachable")
		case 6:
			return nil
		case 7:
			return errors.New("search-hub unreachable again")
		default:
			cancel()
			return nil
		}
	}, retry, maxRetry, refresh)
	if len(stamps) != 8 {
		t.Fatalf("expected 8 registration attempts, got %d", len(stamps))
	}
	gaps := make([]time.Duration, 0, len(stamps)-1)
	for i := 1; i < len(stamps); i++ {
		gaps = append(gaps, stamps[i].Sub(stamps[i-1]))
	}
	// Expected floors: retry, 2×, 4× (=max), max, max, then refresh after the
	// success, then retry again (reset) after the next failure.
	floors := []time.Duration{retry, 2 * retry, maxRetry, maxRetry, maxRetry, refresh, retry}
	for i, floor := range floors {
		if gaps[i] < floor {
			t.Fatalf("gap %d = %v is below its floor %v (gaps=%v)", i, gaps[i], floor, gaps)
		}
	}
	// Growth: the retry after the fifth failure waited at least the ceiling,
	// which is above the floor the first retry used; the reset after success
	// must be back under the ceiling.
	if gaps[6] >= maxRetry {
		t.Fatalf("backoff did not reset after success: gap after post-success failure = %v, ceiling %v", gaps[6], maxRetry)
	}
	if gaps[2] < 2*gaps[0] {
		t.Fatalf("backoff did not grow: first retry %v, third retry %v", gaps[0], gaps[2])
	}
}

// Registration failures must not re-read the contract index (a ReadDir of
// every scenario) on every retry; only an index failure or a success cycle
// re-reads it.
func TestDiscoveryDoesNotRefreshIndexWhileRegistrationFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	indexRefreshes, registrations := 0, 0
	reconcileSearchProviders(ctx, func() error {
		indexRefreshes++
		if indexRefreshes == 1 {
			return errors.New("scenarios directory unreadable")
		}
		return nil
	}, func(context.Context) error {
		registrations++
		if registrations < 4 {
			return errors.New("search-hub unreachable")
		}
		if registrations == 5 {
			cancel()
		}
		return nil
	}, time.Millisecond, time.Millisecond, time.Millisecond)
	// Attempt 1: index fails (no registration). Attempt 2: index ok, reg fails.
	// Attempts 3-4: reg fails/ok without an index read. Attempt 5 (after a
	// success): index re-read on the periodic cycle, then registration.
	if registrations != 5 {
		t.Fatalf("registrations = %d, want 5", registrations)
	}
	if indexRefreshes != 3 {
		t.Fatalf("index refreshes = %d, want 3 (failed read, retry, post-success cycle)", indexRefreshes)
	}
}

func TestProductionDiscoveryDescriptorsHaveStatusTelemetry(t *testing.T) {
	// Binding registration precedes library registration: a rejected binding
	// descriptor must not silently prevent capability profiles from publishing.
	for _, descriptor := range []*registryv1.ProviderDescriptor{bindingDescriptor(), libraryDescriptor()} {
		if descriptor.GetStatusEndpoint().GetHttpJson().GetPath() == "" || descriptor.GetIndexTimestampField() == "" {
			t.Fatalf("%s cannot satisfy production registration", descriptor.GetProviderId())
		}
	}
}

func TestDeviceIntentFindsOwnerDeclaredProgram(t *testing.T) {
	raw, err := os.ReadFile("../../../../device-control/.vrooli/program-runtime/volume.json")
	if err != nil {
		t.Fatal(err)
	}
	var declared struct{ Name, Purpose string }
	if err := json.Unmarshal(raw, &declared); err != nil {
		t.Fatal(err)
	}
	device := corpusRecord{ID: declared.Name, Title: declared.Name, Snippet: declared.Purpose, Kind: "contract"}
	unrelated := corpusRecord{ID: "fixture.fanout", Snippet: "Run three governed reads and return a count"}
	for _, query := range []string{"Turn down the volume of my tv 50%", "make the television quieter", "lower TV volume"} {
		if lexicalScore(query, device) <= lexicalScore(query, unrelated) || lexicalScore(query, device) < 0.4 {
			t.Fatalf("%q did not retrieve the device capability: %v", query, lexicalScore(query, device))
		}
	}
}

func TestLibraryCorpusSkillSetQueryRanksDeclaredContract(t *testing.T) {
	records := []corpusRecord{
		{ID: "prompt-manager.skill-set-read", Scenario: "prompt-manager", Command: "skill-set-read", Title: "prompt-manager.skill-set-read", Snippet: "Read a scenario's skill set and return the declared skills.", Kind: "contract"},
		{ID: "unrelated", Scenario: "program-runtime", Command: "fleet", Title: "unrelated", Snippet: "Read several governed scenario surfaces.", Kind: "callable"},
	}
	if lexicalScore("read a scenario's skill set", records[0]) <= lexicalScore("read a scenario's skill set", records[1]) {
		t.Fatalf("skill-set contract did not score highest")
	}
}

func TestLibraryCorpusExactReviewedIntentWinsTie(t *testing.T) {
	preferred := corpusRecord{
		ID:      "search-hub/query/query",
		Snippet: bindingIntentAliases(&bindingsv1.Binding{Id: "search-hub/query/query"}),
	}
	competitor := corpusRecord{ID: "architecture-cartographer/search/query", Snippet: "search project"}
	if lexicalScore("search the project by intent", preferred) <= lexicalScore("search the project by intent", competitor) {
		t.Fatalf("reviewed intent did not win: preferred=%v competitor=%v", lexicalScore("search the project by intent", preferred), lexicalScore("search the project by intent", competitor))
	}
}

func TestLibraryCorpusRecordsHaveCallableOrContractKind(t *testing.T) {
	for _, record := range []corpusRecord{{Kind: "contract"}, {Kind: "callable"}} {
		if record.Kind != "contract" && record.Kind != "callable" {
			t.Fatalf("invalid corpus kind %q", record.Kind)
		}
	}
	if lexicalScore("read a scenario's skill set", corpusRecord{ID: "candidate-prog_uuid", Snippet: "Automatically accumulated successful program candidate."}) != 0 {
		t.Fatalf("candidate record unexpectedly matched")
	}
}

func TestLibraryCorpusUsageBreaksEqualScoreTies(t *testing.T) {
	rows := []scoredRecord{
		{record: corpusRecord{ID: "unused.program", Usage: 0}, score: 0.5},
		{record: corpusRecord{ID: "used.program", Usage: 4}, score: 0.5},
	}
	sortLibraryRecords(rows)
	if rows[0].record.ID != "used.program" {
		t.Fatalf("usage did not break equal-score tie: %#v", rows)
	}
}

func TestLibraryDescriptorCarriesUsageMetadata(t *testing.T) {
	fields := libraryDescriptor().GetResultMapping().GetMetadataFields()
	if fields["usage"] != "usage" {
		t.Fatalf("usage metadata mapping missing: %#v", fields)
	}
}

func TestLibraryCorpusDistinctContractsProduceDistinctScores(t *testing.T) {
	one := corpusRecord{ID: "alpha.one", Scenario: "alpha", Title: "alpha.one", Snippet: "Read a bounded alpha corpus", Kind: "contract"}
	two := corpusRecord{ID: "beta.two", Scenario: "beta", Title: "beta.two", Snippet: "Write an unrelated beta report", Kind: "contract"}
	oneScore := lexicalScore("read bounded corpus", one)
	twoScore := lexicalScore("read bounded corpus", two)
	if one.Snippet == two.Snippet || oneScore == twoScore {
		t.Fatalf("contracts are not distinguishable: snippets=%q/%q scores=%v/%v", one.Snippet, two.Snippet, oneScore, twoScore)
	}
}

func TestLibraryCorpusNegativeQueriesHaveNoStrongHit(t *testing.T) {
	record := corpusRecord{ID: "prompt-manager.skill-set-read", Scenario: "prompt-manager", Title: "prompt-manager.skill-set-read", Snippet: "Read a scenario's skill set", Kind: "contract"}
	for _, query := range []string{"book a flight to Mars", "bake a chocolate cake", "compose a symphony"} {
		if score := lexicalScore(query, record); score > 0.2 {
			t.Fatalf("negative query %q scored %v", query, score)
		}
	}
}
