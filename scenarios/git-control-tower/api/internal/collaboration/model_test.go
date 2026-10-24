package collaboration

import "testing"

func TestHostIdentityIncludesInstanceAndRepository(t *testing.T) {
	base := Change{Host: HostIdentity{Kind: HostGitHub, InstanceURL: "https://github.example", RepositoryID: "org/repo"}, Number: "42", HeadRevision: "deadbeef"}
	a, err := base.IdentityDigest()
	if err != nil {
		t.Fatal(err)
	}
	base.Host.InstanceURL = "https://github.com"
	b, err := base.IdentityDigest()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("different host instances must not share change identity")
	}
}

func TestUnsupportedAndDisconnectedCapabilitiesRemainDistinct(t *testing.T) {
	statuses := []CapabilityStatus{{Capability: CapabilityReviews, Standing: StandingUnsupported}, {Capability: CapabilityChecks, Standing: StandingDisconnected}}
	if statuses[0].Standing == statuses[1].Standing {
		t.Fatal("capability standing collapsed")
	}
}

func TestPublicationRequiresHumanExactPreconditions(t *testing.T) {
	if err := (DraftPublication{DraftDigest: "sha256:d", Destination: "comment:42", ExpectedRevision: "head-1", AuthorityStanding: "agent"}).Validate(); err == nil {
		t.Fatal("agent publication must be refused")
	}
	if err := (DraftPublication{DraftDigest: "sha256:d", Destination: "comment:42", ExpectedRevision: "head-1", AuthorityStanding: "human_verified"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
