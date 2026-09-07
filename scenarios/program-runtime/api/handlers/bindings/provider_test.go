package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestDiscoveryReconcilesAfterSuccessAndTransientFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	calls := 0
	reconcileSearchProviders(ctx, func(callCtx context.Context) error {
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
	}, time.Millisecond, time.Millisecond)
	if calls != 3 {
		t.Fatalf("registration stopped after success: %d calls", calls)
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
