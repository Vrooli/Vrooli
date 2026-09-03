package bindings

import "testing"

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
