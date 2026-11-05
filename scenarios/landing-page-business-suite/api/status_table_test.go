package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keep the human-facing paid-feature status table aligned with the concrete
// registration and conformance seams. This deliberately fails on the old
// overclaim instead of silently accepting documentation drift.
func TestPaidFeaturesStatusTableDoesNotOverclaimMobileOrConformance(t *testing.T) {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = filepath.Join("..", "..", "..")
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "concepts", "PAID_FEATURES.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if strings.Contains(doc, "Live for BAS and web-console; schema version 2") {
		t.Fatal("PAID_FEATURES still overclaims universal monetization-conformance coverage")
	}
	if strings.Contains(doc, "Live behind the LPBS `ReceiptValidator` seam; mobile platform SDK calls remain shell-owned") {
		t.Fatal("PAID_FEATURES still claims mobile receipt submission is live")
	}
}
