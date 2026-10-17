package targets

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestHumanSummaryIncludesAvailableAndUnavailableTargetFacts(t *testing.T) {
	value := &structpb.Struct{}
	if err := protojson.Unmarshal([]byte(`{
      "targets": [
        {"os":"linux","architecture":"amd64","descriptor":{"target_id":"local-linux-amd64","display_name":"Local host","available":true}},
        {"os":"darwin","architecture":"amd64","reason":"bridge node is online but declares no supported capability","descriptor":{"target_id":"bridge:node-1","display_name":"minimouse","available":false,"missing_capability":"gui-session"}}
      ]
    }`), value); err != nil {
		t.Fatal(err)
	}

	lines := humanSummary(value)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Local host (local-linux-amd64) platform=linux/amd64 available=true",
		"minimouse (bridge:node-1) platform=darwin/amd64 available=false",
		"reason=bridge node is online but declares no supported capability",
		"missing capability or next action: gui-session",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("human summary missing %q:\n%s", want, joined)
		}
	}
}
