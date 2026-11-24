package workload

import (
	"strings"
	"testing"
	"time"
)

func fixtureProfile() Profile {
	return Profile{
		Network: NetworkSandbox2, ChainID: "fixture-sandbox-chain", Service: "web", Image: "ghcr.io/example/web:v1.2.3",
		CPUUnits: "0.5", Memory: "512Mi",
		Storage:   []Volume{{Size: "512Mi"}, {Name: "data", Size: "2Gi", Persistent: true, Class: "beta2"}},
		Ports:     []Port{{Container: 8080, External: 443, Protocol: "TCP"}},
		Placement: Placement{Name: "akash", Region: "fixture-region"},
		MaxPrice:  Rate{Denom: "uact", Amount: 10000}, HealthProbe: HealthProbe{Port: 8080, Path: "/healthz"},
	}
}

func TestRenderProducesStableSDLAndDigestFromFixtureProfile(t *testing.T) {
	firstProfile := fixtureProfile()
	firstProfile.Ports = append(firstProfile.Ports, Port{Container: 9000, External: 9000, Protocol: "tcp"})
	firstProfile.Storage = append(firstProfile.Storage, Volume{Name: "zeta", Size: "1Gi", Persistent: true, Class: "beta1"})
	originalProtocol := firstProfile.Ports[0].Protocol
	first, err := Render(firstProfile)
	if err != nil {
		t.Fatal(err)
	}
	// Source ordering is not semantic; canonical output stays byte-identical.
	secondProfile := fixtureProfile()
	secondProfile.Ports = []Port{{Container: 9000, External: 9000, Protocol: "tcp"}, {Container: 8080, External: 443, Protocol: "tcp"}}
	secondProfile.Storage = []Volume{{Name: "zeta", Size: "1Gi", Persistent: true, Class: "beta1"}, {Name: "data", Size: "2Gi", Persistent: true, Class: "beta2"}, {Size: "512Mi"}}
	second, err := Render(secondProfile)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || first.SDL != second.SDL {
		t.Fatal("reordered fixture fields changed canonical SDL or digest")
	}
	if firstProfile.Ports[0].Protocol != originalProtocol {
		t.Fatal("Render mutated the caller's profile slices")
	}
	if second.Digest == "" || len(second.Digest) != 64 {
		t.Fatalf("unexpected digest length %q", second.Digest)
	}
	if !strings.Contains(first.SDL, "version: \"2.0\"") || !strings.Contains(first.SDL, "persistent: true") || !strings.Contains(first.SDL, "amount: 10000") {
		t.Fatalf("render omitted SDL contract fields:\n%s", first.SDL)
	}
	if strings.Contains(first.SDL, "healthz") {
		t.Fatal("health probe metadata must not be rendered as an unsupported SDL field")
	}
}

func TestRenderRefusesIncompleteUnsafeOrWrongNetworkProfiles(t *testing.T) {
	cases := map[string]func(*Profile){
		"wrong network":        func(p *Profile) { p.Network = "mainnet" },
		"missing chain":        func(p *Profile) { p.ChainID = "" },
		"floating image":       func(p *Profile) { p.Image = "nginx:latest" },
		"missing ephemeral":    func(p *Profile) { p.Storage = []Volume{{Name: "data", Size: "1Gi", Persistent: true, Class: "beta1"}} },
		"duplicate port":       func(p *Profile) { p.Ports = append(p.Ports, p.Ports[0]) },
		"health port mismatch": func(p *Profile) { p.HealthProbe.Port = 3000 },
		"invalid price":        func(p *Profile) { p.MaxPrice.Amount = 0 },
		"no global port":       func(p *Profile) { p.Ports = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := fixtureProfile()
			mutate(&p)
			if _, err := Render(p); err == nil {
				t.Fatal("expected invalid profile refusal")
			}
		})
	}
}

func fixtureSelectedBid() (SelectedBid, BidScope) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	owner := "akash1" + strings.Repeat("a", 39)
	provider := "akash1" + strings.Repeat("b", 39)
	digest := strings.Repeat("c", 64)
	selected := SelectedBid{
		Tuple:    BidTuple{Network: NetworkSandbox2, ChainID: "fixture-sandbox-chain", Owner: owner, DSEQ: 1234, GSEQ: 1, OSEQ: 2, Provider: provider},
		IntentID: "fixture-intent", WorkloadDigest: digest, Price: Rate{Denom: "uact", Amount: 9000}, ObservedAt: now.Add(-time.Minute), ValidUntilHeight: 1200,
	}
	scope := BidScope{Network: NetworkSandbox2, ChainID: "fixture-sandbox-chain", Owner: owner, IntentID: "fixture-intent", WorkloadDigest: digest, MaxPrice: Rate{Denom: "uact", Amount: 10000}, CurrentHeight: 1100, Now: now, MaxAge: 5 * time.Minute}
	return selected, scope
}

func TestSelectedBidBindsCompleteTupleAndFreshMatchingScope(t *testing.T) {
	selected, scope := fixtureSelectedBid()
	if err := selected.Tuple.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := selected.ValidateFor(scope); err != nil {
		t.Fatalf("valid fixture tuple rejected: %v", err)
	}
	if selected.Tuple.Key() == "" || !strings.Contains(selected.Tuple.Key(), ":1234:1:2:") {
		t.Fatalf("tuple key did not preserve sequence components: %q", selected.Tuple.Key())
	}
}

func TestSelectedBidRefusesMissingMismatchedWrongChainStaleExpiredAndOverCap(t *testing.T) {
	cases := map[string]func(*SelectedBid, *BidScope){
		"incomplete tuple":          func(b *SelectedBid, _ *BidScope) { b.Tuple.Provider = "" },
		"wrong network":             func(b *SelectedBid, _ *BidScope) { b.Tuple.Network = "mainnet" },
		"wrong chain":               func(b *SelectedBid, _ *BidScope) { b.Tuple.ChainID = "another-chain" },
		"wrong owner":               func(b *SelectedBid, _ *BidScope) { b.Tuple.Owner = "akash1" + strings.Repeat("z", 39) },
		"wrong intent":              func(b *SelectedBid, _ *BidScope) { b.IntentID = "other-intent" },
		"wrong workload":            func(b *SelectedBid, _ *BidScope) { b.WorkloadDigest = strings.Repeat("d", 64) },
		"malformed expected digest": func(_ *SelectedBid, s *BidScope) { s.WorkloadDigest = "not-a-digest" },
		"stale observation":         func(b *SelectedBid, _ *BidScope) { b.ObservedAt = b.ObservedAt.Add(-10 * time.Minute) },
		"expired height":            func(b *SelectedBid, s *BidScope) { b.ValidUntilHeight = s.CurrentHeight },
		"over max price":            func(b *SelectedBid, _ *BidScope) { b.Price.Amount = 10001 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b, s := fixtureSelectedBid()
			mutate(&b, &s)
			if err := b.ValidateFor(s); err == nil {
				t.Fatal("expected selected-bid refusal")
			}
		})
	}
}

func TestDuplicateSelectedBidTupleIsRefused(t *testing.T) {
	bid, _ := fixtureSelectedBid()
	if err := ValidateUniqueBidTuples([]SelectedBid{bid, bid}); err == nil {
		t.Fatal("expected duplicate selected-bid tuple refusal")
	}
}
