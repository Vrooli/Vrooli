package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	startMarker           = "<!-- BEGIN GENERATED TUNING LEVERS -->"
	endMarker             = "<!-- END GENERATED TUNING LEVERS -->"
	documentationFileMode = 0o644
)

func main() {
	check := flag.Bool("check", false, "fail if the generated tuning reference is stale")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if err := run(filepath.Clean(*root), *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, check bool) error {
	path := filepath.Join(root, "docs", "reference", "environment-management.md")
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	start := bytes.Index(current, []byte(startMarker))
	end := bytes.Index(current, []byte(endMarker))
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf("%s must contain the generated tuning markers", path)
	}
	end += len(endMarker)
	block := []byte(startMarker + "\n" + tuning.RenderDocumentation() + endMarker)
	want := append(append(append([]byte(nil), current[:start]...), block...), current[end:]...)
	if bytes.Equal(current, want) {
		return nil
	}
	if check {
		return fmt.Errorf("%s tuning reference is stale; run make generate-tuning-docs", path)
	}
	if err := os.WriteFile(path, want, documentationFileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
