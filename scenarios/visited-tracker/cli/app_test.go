package main

import (
	"os"
	"reflect"
	"testing"
)

func TestNormalizeArgsCampaignAliases(t *testing.T) {
	app := &App{}

	args, err := app.normalizeArgs([]string{"campaign"})
	if err != nil {
		t.Fatalf("normalizeArgs returned error: %v", err)
	}

	expected := []string{"campaigns", "list"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("expected %v, got %v", expected, args)
	}
}

func TestNormalizeArgsCampaignFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantArgs     []string
		wantCampaign string
	}{
		{
			name:         "CampaignIDEquals",
			args:         []string{"--campaign-id=abc", "coverage"},
			wantArgs:     []string{"coverage"},
			wantCampaign: "abc",
		},
		{
			name:         "CampaignIDSeparate",
			args:         []string{"--campaign", "def", "coverage"},
			wantArgs:     []string{"coverage"},
			wantCampaign: "def",
		},
		{
			name:         "CampaignAlias",
			args:         []string{"campaign", "get"},
			wantArgs:     []string{"campaigns", "get"},
			wantCampaign: "",
		},
		{
			name:         "CampaignsDefaultList",
			args:         []string{"campaigns"},
			wantArgs:     []string{"campaigns", "list"},
			wantCampaign: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VISITED_TRACKER_CAMPAIGN_ID", "")
			app := &App{}

			args, err := app.normalizeArgs(tc.args)
			if err != nil {
				t.Fatalf("normalizeArgs returned error: %v", err)
			}

			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("expected args %v, got %v", tc.wantArgs, args)
			}

			if tc.wantCampaign != "" {
				if got := os.Getenv("VISITED_TRACKER_CAMPAIGN_ID"); got != tc.wantCampaign {
					t.Fatalf("expected campaign env %q, got %q", tc.wantCampaign, got)
				}
			}
		})
	}
}

func TestNormalizeArgsMissingFlagValue(t *testing.T) {
	app := &App{}

	if _, err := app.normalizeArgs([]string{"--campaign-id"}); err == nil {
		t.Fatal("expected error for missing campaign-id value")
	}
	if _, err := app.normalizeArgs([]string{"--campaign"}); err == nil {
		t.Fatal("expected error for missing campaign value")
	}
}

func TestSetCampaignID(t *testing.T) {
	t.Setenv("VISITED_TRACKER_CAMPAIGN_ID", "")
	app := &App{}

	app.setCampaignID("  test-id  ")
	if app.campaignID != "test-id" {
		t.Fatalf("expected campaignID to be set, got %q", app.campaignID)
	}
	if got := os.Getenv("VISITED_TRACKER_CAMPAIGN_ID"); got != "test-id" {
		t.Fatalf("expected campaign env to be set, got %q", got)
	}

	app.setCampaignID("   ")
	if got := os.Getenv("VISITED_TRACKER_CAMPAIGN_ID"); got != "test-id" {
		t.Fatalf("expected campaign env to remain unchanged, got %q", got)
	}
}
