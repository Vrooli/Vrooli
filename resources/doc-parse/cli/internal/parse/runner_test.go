package parse

import (
	"encoding/json"
	"testing"
)

func TestRequestIncludesCapabilities(t *testing.T) {
	data, err := json.Marshal(Request{Path: "/input/record.pdf", Capabilities: []string{"content", "geometry"}})
	if err != nil {
		t.Fatal(err)
	}
	var got Request
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Path != "/input/record.pdf" || len(got.Capabilities) != 2 || got.Capabilities[1] != "geometry" {
		t.Fatalf("request = %+v", got)
	}
}
