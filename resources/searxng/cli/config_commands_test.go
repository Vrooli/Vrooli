package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommandsApplyShowAndValidate(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer
	if err := runConfigApply([]string{"--config-dir", dir, "--instance-name", "Unit Test"}, &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "secret_key") && !strings.Contains(output.String(), "generated") {
		t.Fatalf("apply output unexpectedly includes secret: %s", output.String())
	}
	output.Reset()
	if err := runConfigShow([]string{"--config-dir", dir}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "[redacted]") {
		t.Fatalf("show did not redact secret: %s", output.String())
	}
	output.Reset()
	if err := runConfigValidate([]string{"--config-dir", dir}, &output); err != nil {
		t.Fatal(err)
	}
}

func TestConfigApplyRejectsEmptySecretFile(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret")
	if err := os.WriteFile(secret, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runConfigApply([]string{"--config-dir", dir, "--secret-file", secret}, &bytes.Buffer{}); err == nil {
		t.Fatal("config apply accepted an empty secret file")
	}
}
