package lib

import (
	"testing"

	"github.com/vrooli/fixtures/go-tests/helper"
)

func TestLive(t *testing.T) {
	if helper.FromTest() == "" {
		t.Fatal("expected helper output")
	}
}
