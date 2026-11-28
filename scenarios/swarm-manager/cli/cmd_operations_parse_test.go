package main

import "testing"

func TestParseOperationsFlags_WithFilters(t *testing.T) {
	args := []string{
		"--window", " PT24H ",
		"--status", "active",
		"--status", "blocked",
		"--lane", "fix",
		"--mode", "yolo",
		"--owner-type", "agent",
		"--q", " search text ",
		"--json",
	}
	query, jsonOut, err := parseOperationsFlags("operations list", true, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !jsonOut {
		t.Error("expected jsonOut true")
	}
	if query.Get("window") != "PT24H" {
		t.Errorf("window = %q", query.Get("window"))
	}
	if got := query["status"]; len(got) != 2 || got[0] != "active" || got[1] != "blocked" {
		t.Errorf("status = %v", got)
	}
	if query.Get("lane") != "fix" {
		t.Errorf("lane = %q", query.Get("lane"))
	}
	if query.Get("mode") != "yolo" {
		t.Errorf("mode = %q", query.Get("mode"))
	}
	if query.Get("owner_type") != "agent" {
		t.Errorf("owner_type = %q", query.Get("owner_type"))
	}
	if query.Get("q") != "search text" {
		t.Errorf("q = %q", query.Get("q"))
	}
}

func TestParseOperationsFlags_FiltersDisabled(t *testing.T) {
	// When includeFilters is false, the --q flag is not registered and using
	// it is a parse error; filter flags are also absent.
	query, jsonOut, err := parseOperationsFlags("operations brief", false, []string{"--window", "PT1H"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jsonOut {
		t.Error("expected jsonOut false")
	}
	if query.Get("window") != "PT1H" {
		t.Errorf("window = %q", query.Get("window"))
	}
	if _, has := query["status"]; has {
		t.Error("status should not be set when filters disabled")
	}
}

func TestParseOperationsFlags_ParseError(t *testing.T) {
	if _, _, err := parseOperationsFlags("operations list", true, []string{"--unknown"}); err == nil {
		t.Error("expected parse error for unknown flag")
	}
}
