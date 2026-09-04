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
