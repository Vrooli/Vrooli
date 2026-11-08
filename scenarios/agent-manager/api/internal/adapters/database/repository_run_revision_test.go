package database

import (
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

func TestNarrowRunMutationsAdvanceCommittedRevisionWithoutTrigger(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	effortRef := "effort:revision-owner"
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked, WorkReferences: []*eventdomain.WorkReference{{Kind: "effort", Id: effortRef, Relationship: "orchestrator", Verified: true, Visibility: eventdomain.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventdomain.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	before := run.UpdatedAt
	for _, change := range []func() error{
		func() error {
			return repos.Runs.(repository.RunLabelUpdater).UpdateRunLabel(t.Context(), run.ID, "operator name", domain.RunLabelSourceManual)
		},
		func() error {
			return repos.Runs.(repository.RunSubjectUpdater).UpdateRunSubject(t.Context(), run.ID, []string{"recovery"})
		},
		func() error {
			_, eligible, err := repos.Runs.(repository.EffortSignalCursorRepository).QueueEffortSignalCursor(t.Context(), run.ID, effortRef, 1)
			if !eligible {
				t.Fatal("active public effort owner was not eligible")
			}
			return err
		},
	} {
		if err := change(); err != nil {
			t.Fatal(err)
		}
		stored, err := repos.Runs.Get(t.Context(), run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !stored.UpdatedAt.After(before) {
			t.Fatalf("narrow mutation did not advance revision: before=%v after=%v", before, stored.UpdatedAt)
		}
		before = stored.UpdatedAt
	}
}
