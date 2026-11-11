package sqlcompat

import (
	"fmt"
	"testing"
)

type sqliteCode int

func (e sqliteCode) Error() string { return "driver failure" }
func (e sqliteCode) Code() int     { return int(e) }

func TestTransientErrorUsesWrappedSQLiteCodes(t *testing.T) {
	for _, code := range []int{5, 6, 517, 262, 1, 19, 11} {
		want := code&255 == 5 || code&255 == 6
		if got := TransientError(fmt.Errorf("persist: %w", sqliteCode(code))); got != want {
			t.Errorf("code%d transient%t want%t", code, got, want)
		}
	}
	if TransientError(nil) || TransientError(fmt.Errorf("SQLITE_BUSY is only untrusted text")) {
		t.Fatal("text classified as driver contention")
	}
}

func TestDBRemainsAnInterface(t *testing.T) {
	var database DB
	if database != nil {
		t.Fatal("zero-value database interface must be nil")
	}
}
