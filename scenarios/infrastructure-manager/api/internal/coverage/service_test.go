package coverage

import (
	"context"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestSnapshotAllReliabilitySpacesResolveAndGradeNOWCells(t *testing.T) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(root, FileSpaceReader{Root: root})
	snapshot, err := service.Snapshot(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projections) != len(projectionOwners) {
		t.Fatalf("resolved projections = %d, want %d; findings=%+v", len(snapshot.Projections), len(projectionOwners), snapshot.Findings)
	}
	for _, finding := range snapshot.Findings {
		if finding.Code != "SPACE_UNAVAILABLE" {
			t.Fatalf("unexpected integrity finding: %+v", finding)
		}
	}
}
