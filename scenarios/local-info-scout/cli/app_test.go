package main

import "testing"

func TestAppConstants(t *testing.T) {
	if appName != "local-info-scout" {
		t.Fatalf("appName = %q, want local-info-scout", appName)
	}
	if appVersion == "" {
		t.Fatal("appVersion should not be empty")
	}
}
