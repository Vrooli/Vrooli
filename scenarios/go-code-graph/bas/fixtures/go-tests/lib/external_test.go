package lib_test

import (
	"testing"

	"github.com/vrooli/fixtures/go-tests/lib"
)

func TestExternal(t *testing.T) {
	if lib.Live() == "" {
		t.Fatal("expected live output")
	}
}
