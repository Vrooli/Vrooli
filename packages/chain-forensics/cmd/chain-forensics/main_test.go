package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectOutputsTypedExampleCaseJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"case", "inspect", exampleID, "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["id"] != exampleID || decoded["private"] != true {
		t.Fatalf("unexpected case JSON identity/privacy: %#v", decoded)
	}
}

func TestPacketWritesPrivateFileWithoutOverwriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.packet.json")
	var stdout, stderr bytes.Buffer
	args := []string{"case", "packet", exampleID, "--out", path}
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("packet mode = %o, want 600", info.Mode().Perm())
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"schema_version": "chain-forensics.packet/v1"`) {
		t.Fatal("packet is missing its schema version")
	}
	stdout.Reset()
	if err := run(args, &stdout, &stderr); err == nil {
		t.Fatal("packet command overwrote an existing file")
	}
}
