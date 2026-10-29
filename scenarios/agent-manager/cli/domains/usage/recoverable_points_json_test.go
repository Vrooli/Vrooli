package usage

import (
	"strings"
	"testing"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestGroupAndLeverShareTheRecoverablePointsKey(t *testing.T) {
	out, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(&measurepb.UsageRecommendationGroup{Levers: []*measurepb.UsageLever{{}}})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), `"recoverable_points"`); n != 2 || strings.Contains(string(out), `"recoverable"`) {
		t.Fatalf("group and lever must both use recoverable_points: %s", out)
	}
}
