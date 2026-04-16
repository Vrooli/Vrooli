package main

import "testing"

func TestAppConstants(t *testing.T) {
	if appName != "elo-swipe" {
		t.Fatalf("appName = %q, want elo-swipe", appName)
	}
	if appVersion == "" {
		t.Fatal("appVersion should not be empty")
	}
}
