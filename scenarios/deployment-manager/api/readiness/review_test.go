package readiness

import "testing"

func TestReviewIdentityKeyIsStableAcrossTargetOrder(t *testing.T) {
	a := ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:one", Targets: []string{"windows", "linux"}, Channel: "stable", PolicyVersion: 2}
	b := a
	b.Targets = []string{"linux", "windows", "linux"}
	keyA, err := a.Key()
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := b.Key()
	if err != nil {
		t.Fatal(err)
	}
	if keyA != keyB {
		t.Fatalf("keys differ: %s != %s", keyA, keyB)
	}
}

func TestReviewIdentityKeyIncludesEveryIdentityDimension(t *testing.T) {
	base := ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:one", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2}
	baseKey, err := base.Key()
	if err != nil {
		t.Fatal(err)
	}
	mutations := []ReviewIdentity{base, base, base, base, base, base, base}
	mutations[0].Scenario = "other"
	mutations[1].ProfileID = "p2"
	mutations[2].CandidateCommit = "def"
	mutations[3].ArtifactDigest = "sha256:two"
	mutations[4].Targets = []string{"windows"}
	mutations[5].Channel = "beta"
	mutations[6].PolicyVersion = 3
	for _, mutation := range mutations {
		key, err := mutation.Key()
		if err != nil {
			t.Fatal(err)
		}
		if key == baseKey {
			t.Fatalf("identity mutation did not change key: %+v", mutation)
		}
	}
}

func TestReleaseBindingDimensionsAreOptionalForLegacyReviewsButKeyedWhenPresent(t *testing.T) {
	legacy := ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:one", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2}
	bound := legacy
	bound.CandidateID = "candidate-1"
	bound.DestinationRevisionID = "destination-1"
	bound.AuthorizationEpoch = 1
	legacyKey, err := legacy.Key()
	if err != nil {
		t.Fatal(err)
	}
	boundKey, err := bound.Key()
	if err != nil {
		t.Fatal(err)
	}
	if legacyKey == boundKey {
		t.Fatal("release binding did not change the review key")
	}
	invalid := bound
	invalid.AuthorizationEpoch = 0
	if _, err := invalid.Key(); err == nil {
		t.Fatal("incomplete release binding was accepted")
	}
}
