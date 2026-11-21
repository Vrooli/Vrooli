package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"knowledge-observatory/internal/services/dochealth"
)

const (
	beginMarker = "<!-- BEGIN GENERATED DOCHEALTH CHECKS -->"
	endMarker   = "<!-- END GENERATED DOCHEALTH CHECKS -->"
)

func main() {
	root := flag.String("root", "", "scenario root containing docs/reference/cli-commands.md")
	flag.Parse()
	if err := generate(strings.TrimSpace(*root)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(root string) error {
	if root == "" {
		return fmt.Errorf("--root is required")
	}
	path := filepath.Join(root, "docs", "reference", "cli-commands.md")
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat generated docs target: %w", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated docs target: %w", err)
	}
	text := string(content)
	start := strings.Index(text, beginMarker)
	end := strings.Index(text, endMarker)
	if start < 0 || end < 0 || end <= start {
		return fmt.Errorf("generated docs markers are missing or out of order in %s", path)
	}
	start += len(beginMarker)
	updated := text[:start] + "\n\n" + dochealth.CheckDocumentationTable() + "\n\n" + text[end:]
	if updated == text {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write generated docs target: %w", err)
	}
	return nil
}
