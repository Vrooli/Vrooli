package campaigns

import (
	"bytes"
	"encoding/json"
	"testing"

	"visited-tracker/cli/internal/support"
)

func TestFilterCampaignsByTag(t *testing.T) {
	tagA, tagB := "alpha", "beta"
	campaigns := []support.Campaign{
		{ID: "one", Tag: &tagA},
		{ID: "two", Tag: &tagB},
		{ID: "untagged"},
	}

	filtered := filterCampaignsByTag(campaigns, " alpha ")
	if len(filtered) != 1 || filtered[0].ID != "one" {
		t.Fatalf("filtered campaigns = %#v, want only alpha", filtered)
	}
	if got := filterCampaignsByTag(campaigns, ""); len(got) != len(campaigns) {
		t.Fatalf("empty tag returned %d campaigns, want all %d", len(got), len(campaigns))
	}
}

func TestCampaignsJSONContainsStructuredCampaignRecords(t *testing.T) {
	tag := "hunt"
	campaigns := []support.Campaign{{ID: "campaign-1", Name: "round", Tag: &tag, TotalFiles: 4}}
	var output bytes.Buffer
	if err := writeCampaignsJSON(&output, campaigns); err != nil {
		t.Fatal(err)
	}

	var result struct {
		Campaigns []map[string]any `json:"campaigns"`
		Count     int              `json:"count"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if result.Count != 1 || len(result.Campaigns) != 1 {
		t.Fatalf("result count/records = %d/%d, want 1/1", result.Count, len(result.Campaigns))
	}
	if result.Campaigns[0]["id"] != "campaign-1" || result.Campaigns[0]["tag"] != "hunt" {
		t.Fatalf("campaign JSON = %#v, want id and tag fields", result.Campaigns[0])
	}
}
