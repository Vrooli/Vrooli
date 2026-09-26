package deliveryramp

import "testing"

func TestSourceDistributionRefRequiresExactIdentity(t *testing.T) {
	if !(SourceDistributionRef{DistributionID: "d1", Scenario: "demo", SourceDigest: "sha256:s", ArtifactDigest: "sha256:a"}).Valid() {
		t.Fatal("complete source distribution identity should be valid")
	}
	if (SourceDistributionRef{DistributionID: "d1", Scenario: "demo", SourceDigest: "sha256:s"}).Valid() {
		t.Fatal("missing artifact identity must be invalid")
	}
}
