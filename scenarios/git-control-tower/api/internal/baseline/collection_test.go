package baseline

import (
	"strings"
	"testing"
	"time"
)

func sampleCollection() CollectionManifest {
	return CollectionManifest{
		Name: "plan-before", Branch: "agi", SchemaVersion: CollectionSchemaVersion,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		Members: []CollectionMember{
			{Scenario: "plan-manager", BaselineName: "plan-before", Required: true, Status: CollectionMemberReady, RunID: "run-pm"},
			{Scenario: "git-control-tower", BaselineName: "plan-before", Required: true, Status: CollectionMemberPending},
			{Scenario: "optional-docs", BaselineName: "plan-before", Required: false, Status: CollectionMemberSkipped},
		},
	}
}

func TestCollectionCoverageNeverTreatsPartialRequiredSetAsComplete(t *testing.T) {
	collection := sampleCollection()
	coverage := collection.Coverage()
	if coverage.Required != 2 || coverage.Ready != 1 || coverage.Pending != 1 {
		t.Fatalf("coverage = %#v", coverage)
	}
	if coverage.Complete() {
		t.Fatal("partial required collection reported complete")
	}
	collection.Members[1].Status = CollectionMemberReady
	collection.Members[1].RunID = "run-gct"
	if !collection.Coverage().Complete() {
		t.Fatal("all required ready members did not report complete")
	}
}

func TestCollectionManifestRejectsDuplicateAndReadyWithoutRun(t *testing.T) {
	collection := sampleCollection()
	collection.Members = append(collection.Members, CollectionMember{Scenario: "plan-manager", BaselineName: "other", Status: CollectionMemberPending})
	if err := collection.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	collection = sampleCollection()
	collection.Members[0].RunID = ""
	if err := collection.Validate(); err == nil || !strings.Contains(err.Error(), "requires baseline run id") {
		t.Fatalf("ready validation = %v", err)
	}
}

func TestAggregateCollectionDiffPrecedenceKeepsRequiredMemberTruth(t *testing.T) {
	collection := sampleCollection()
	collection.Members[1].Status, collection.Members[1].RunID = CollectionMemberReady, "run-gct"
	result := AggregateCollectionDiff(collection, []CollectionDiffMember{
		{Scenario: "plan-manager", Required: true, Status: "ready", Verdict: VerdictClean},
		{Scenario: "git-control-tower", Required: true, Status: "ready", Verdict: VerdictNewFailure},
	})
	if result.Verdict != CollectionDiffRegression {
		t.Fatalf("new failure aggregate = %#v", result)
	}
	result = AggregateCollectionDiff(collection, []CollectionDiffMember{
		{Scenario: "plan-manager", Required: true, Status: "pending"},
		{Scenario: "git-control-tower", Required: true, Status: "ready", Verdict: VerdictRegression},
	})
	if result.Verdict != CollectionDiffRegression {
		t.Fatalf("regression must outrank not-ready = %#v", result)
	}
	collection.Members[1].Status = CollectionMemberPending
	result = AggregateCollectionDiff(collection, nil)
	if result.Verdict != CollectionDiffNotReady {
		t.Fatalf("pending required collection = %#v", result)
	}
}
