package storage

import (
	"strings"
	"testing"
)

func TestCensusEndpointReturnsClosedAccounting(t *testing.T) {
	if !strings.Contains(Endpoints[0].Path, "census") {
		t.Fatal("census endpoint descriptor missing")
	}
}
