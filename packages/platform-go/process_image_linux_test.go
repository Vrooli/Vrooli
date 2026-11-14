//go:build linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessImageFixture(t *testing.T) {
	switch os.Getenv("PLATFORM_IMAGE_EXEC_PHASE") {
	case "payload":
		_, _ = os.Stdout.WriteString("verified-old-image\n")
	case "wrapper":
		path := os.Getenv("PLATFORM_IMAGE_EXEC_PATH")
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := os.Rename(path, path+".retained"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho WRONG-IMAGE\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := ReplaceProcessImage(file, []string{path, "-test.run=^TestProcessImageFixture$"}, []string{"PLATFORM_IMAGE_EXEC_PHASE=payload"}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProcessImageExecutesVerifiedHandleAfterPathReplacement(t *testing.T) {
	image, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "native-image")
	if err := os.WriteFile(path, image, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessImageFixture$")
	cmd.Env = append(os.Environ(), "PLATFORM_IMAGE_EXEC_PHASE=wrapper", "PLATFORM_IMAGE_EXEC_PATH="+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("image replacement fixture: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "verified-old-image") || strings.Contains(string(output), "WRONG-IMAGE") {
		t.Fatalf("executed pathname replacement: %s", output)
	}
}

func TestProcessImageRejectsScriptsWithoutAnInterpreterContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := ReplaceProcessImage(file, []string{path}, []string{}); err == nil {
		t.Fatal("script was accepted as a native image")
	}
}
