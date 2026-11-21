package projects

import (
	"testing"

	"connectrpc.com/connect"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
)

func TestProjectConflictPreservesTypedRevisionDetails(t *testing.T) {
	err := connect.NewError(connect.CodeAborted, nil)
	detail, detailErr := connect.NewErrorDetail(&projectsv1.ProjectConflict{
		ProjectId:        "project-1",
		ExpectedRevision: 4,
		CurrentRevision:  5,
	})
	if detailErr != nil {
		t.Fatalf("NewErrorDetail() error = %v", detailErr)
	}
	err.AddDetail(detail)

	got, ok := projectConflict(err)
	if !ok {
		t.Fatal("projectConflict() did not recognize ProjectConflict detail")
	}
	if want := "project project-1 revision conflict: expected revision 4, current revision 5"; got.Error() != want {
		t.Fatalf("projectConflict() = %q, want %q", got, want)
	}
}

func TestProjectConflictIgnoresUnrelatedErrors(t *testing.T) {
	if got, ok := projectConflict(connect.NewError(connect.CodeUnavailable, nil)); ok || got != nil {
		t.Fatalf("projectConflict(unrelated error) = (%v, %t), want (nil, false)", got, ok)
	}
}
