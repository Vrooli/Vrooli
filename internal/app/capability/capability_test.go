package capabilityapp

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunHelpUsesCapabilityContract(t *testing.T) {
	var output bytes.Buffer
	if err := (&App{}).Run(&CommandContext{Stdout: &output}, []string{"--help"}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("capability help should be non-empty")
	}
}

func TestWriteCapabilityValuePreservesObjectShape(t *testing.T) {
	var output bytes.Buffer
	if err := writeCapabilityValue(&output, struct {
		State string `json:"state"`
		Count int    `json:"count"`
	}{State: "degraded", Count: 0}); err != nil {
		t.Fatalf("writeCapabilityValue: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	if got["state"] != "degraded" || got["count"] != float64(0) {
		t.Fatalf("unexpected capability output: %v", got)
	}
}
