package roadmap

import (
	"testing"

	roadmapconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap/roadmap_v1connect"
)

func TestEndpointsCoverRoadmapProcedures(t *testing.T) {
	want := map[string]bool{
		roadmapconnect.RoadmapServiceListSectorsProcedure:     false,
		roadmapconnect.RoadmapServiceUpsertSectorProcedure:    false,
		roadmapconnect.RoadmapServiceListMilestonesProcedure:  false,
		roadmapconnect.RoadmapServiceUpsertMilestoneProcedure: false,
		roadmapconnect.RoadmapServiceGetProgressProcedure:     false,
	}
	for _, endpoint := range Endpoints {
		if _, ok := want[endpoint.Path]; ok {
			want[endpoint.Path] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("missing endpoint for %s", path)
		}
	}
}
