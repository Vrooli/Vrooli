package skills

import (
	"encoding/json"
	"testing"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/discovery"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSkillUsageReportRoundTripsProjectedFieldIntoProto(t *testing.T) {
	report := SkillUsageReport{
		Since: "168h0m0s",
		Rows: []SkillUsageRow{{
			SkillID:   "example-skill",
			Returned:  2,
			Reads:     1,
			Projected: true,
		}},
	}

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal skill usage report: %v", err)
	}

	var response discoveryv1.GetSkillUsageResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal skill usage report into proto: %v", err)
	}
	if len(response.GetRows()) != 1 {
		t.Fatalf("rows = %d, want 1", len(response.GetRows()))
	}
	if !response.GetRows()[0].GetProjected() {
		t.Fatal("projected = false, want true")
	}
}
