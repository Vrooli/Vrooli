package strategy

import (
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestExpectedRankUsesFirstExpectedID(t *testing.T) {
	hits := []*routingv1.SearchHit{{Id: "other"}, {Id: "wanted"}, {Id: "later"}}
	if got := expectedRank([]string{"wanted", "later"}, hits); got != 2 {
		t.Fatalf("expectedRank = %d, want 2", got)
	}
	if got := expectedRank([]string{"missing"}, hits); got != 0 {
		t.Fatalf("missing expectedRank = %d, want 0", got)
	}
}

func TestPercentileUsesNearestRank(t *testing.T) {
	values := []int64{40, 10, 30, 20}
	if got := percentile(values, 50); got != 20 {
		t.Fatalf("p50 = %d, want 20", got)
	}
	if got := percentile(values, 95); got != 40 {
		t.Fatalf("p95 = %d, want 40", got)
	}
	if got := percentile(nil, 95); got != 0 {
		t.Fatalf("empty percentile = %d, want 0", got)
	}
}
