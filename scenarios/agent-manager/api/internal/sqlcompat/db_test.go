package sqlcompat

import "testing"

func TestDBRemainsAnInterface(t *testing.T) {
	var database DB
	if database != nil {
		t.Fatal("zero-value database interface must be nil")
	}
}
