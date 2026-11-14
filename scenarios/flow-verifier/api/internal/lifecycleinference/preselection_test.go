package lifecycleinference

import "testing"

func TestPreselectionRequiresPersistenceStateMachineAndCompleteHuntEvidence(t *testing.T) {
	candidate := Candidate{
		ID:               "alpha/api/jobs#Attempt.State",
		Persistence:      PersistenceClassification{Class: ClassPersistedLifecycle, ScanComplete: true},
		States:           []StateEvidence{{Value: "queued"}, {Value: "running"}},
		WriteSites:       []Evidence{{Path: "api/jobs.go", Line: 10}},
		TransitionOwners: []Evidence{{Path: "api/jobs.go", Line: 20}},
	}
	if got := preselectionStatus(candidate, false); got != "unknown_hunt_coverage" {
		t.Fatalf("without complete hunt source got %q", got)
	}
	if got := preselectionStatus(candidate, true); got != "owner_adjudication_candidate" {
		t.Fatalf("source-backed candidate got %q", got)
	}
	candidate.HuntRecords = []HuntReference{{EntryID: "round-32", Status: "review"}}
	if got := preselectionStatus(candidate, true); got != "prior_hunt_review_required" {
		t.Fatalf("previous hunt match got %q", got)
	}
	candidate.HuntRecords = nil
	candidate.Persistence.Class = ClassUnknown
	if got := preselectionStatus(candidate, true); got != "unknown_or_not_source_proven_persisted" {
		t.Fatalf("unknown persistence got %q", got)
	}
}
