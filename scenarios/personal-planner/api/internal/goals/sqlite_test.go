package goals

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestSQLiteMilestonePrerequisitesBlockCompletionAndRollUpProgress(t *testing.T) {
	db, err := sql.Open("sqlite", "file:goals-prerequisites-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,created_at,updated_at,revision) VALUES ('goal-1','Ship','', 'active','milestones',0,10000,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO milestones (id,goal_id,title,status,created_at,updated_at,revision) VALUES ('m-1','goal-1','Foundation','open',1,1,1)`,
		`INSERT INTO milestones (id,goal_id,title,status,created_at,updated_at,revision) VALUES ('m-2','goal-1','Launch','open',1,1,1)`,
		`INSERT INTO milestone_prerequisites (milestone_id,prerequisite_milestone_id,created_at) VALUES ('m-2','m-1',1)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.System())
	if _, err := repo.UpdateMilestoneStatus(context.Background(), "m-2", MilestoneDone, 1); err == nil {
		t.Fatal("expected incomplete prerequisite rejection")
	} else if _, ok := err.(ErrPrerequisitesIncomplete); !ok {
		t.Fatalf("err=%T %v", err, err)
	}
	if _, err := repo.UpdateMilestoneStatus(context.Background(), "m-1", MilestoneDone, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateMilestoneStatus(context.Background(), "m-2", MilestoneDone, 1); err != nil {
		t.Fatal(err)
	}
	goals, err := repo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 || goals[0].ProgressBasisPoints != 10000 {
		t.Fatalf("goals=%#v", goals)
	}
	milestones, err := repo.ListMilestones(context.Background(), "goal-1")
	if err != nil || len(milestones) != 2 || len(milestones[1].PrerequisiteMilestoneIDs) != 1 {
		t.Fatalf("milestones=%#v err=%v", milestones, err)
	}
}
