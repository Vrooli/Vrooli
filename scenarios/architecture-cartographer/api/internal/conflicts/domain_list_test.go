package conflicts

import (
	"reflect"
	"testing"
)

func TestSortedUniqueDomainNames(t *testing.T) {
	got := SortedUniqueDomainNames(" billing ", "", "orders", "billing", "\t", "accounts")
	want := []string{"accounts", "billing", "orders"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SortedUniqueDomainNames() = %v, want %v", got, want)
	}
}
