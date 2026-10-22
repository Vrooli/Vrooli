package goals

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func goalTestContext(subject string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
}

func TestSQLiteMilestonePrerequisitesBlockCompletionAndRollUpProgress(t *testing.T) {
	db, err := sql.Open("sqlite", "file:goals-prerequisites-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,created_at,updated_at,revision) VALUES ('goal-1','Ship','', 'active','milestones',0,10000,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goal_owners (goal_id,owner_subject) VALUES ('goal-1','owner-a')`); err != nil {
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
	ctx := goalTestContext("owner-a")
	if _, err := repo.UpdateMilestoneStatus(ctx, "m-2", MilestoneDone, 1); err == nil {
		t.Fatal("expected incomplete prerequisite rejection")
	} else if _, ok := err.(ErrPrerequisitesIncomplete); !ok {
		t.Fatalf("err=%T %v", err, err)
	}
	if _, err := repo.UpdateMilestoneStatus(ctx, "m-1", MilestoneDone, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateMilestoneStatus(ctx, "m-2", MilestoneDone, 1); err != nil {
		t.Fatal(err)
	}
	goals, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 || goals[0].ProgressBasisPoints != 10000 {
		t.Fatalf("goals=%#v", goals)
	}
	ctx, cancel := context.WithTimeout(goalTestContext("owner-a"), 2*time.Second)
	defer cancel()
	milestones, err := repo.ListMilestones(ctx, "goal-1")
	if err != nil || len(milestones) != 2 || len(milestones[1].PrerequisiteMilestoneIDs) != 1 {
		t.Fatalf("milestones=%#v err=%v", milestones, err)
	}
}

func TestSQLiteSetTargetDateRequiresCurrentGoalRevision(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,target_date,created_at,updated_at,revision) VALUES ('goal-target','Ship','', 'active','manual',0,10000,'2026-11-01',1,1,3)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goal_owners (goal_id,owner_subject) VALUES ('goal-target','owner-a')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	ctx := goalTestContext("owner-a")
	var conflict ErrRevisionConflict
	if err := repo.SetTargetDate(ctx, "goal-target", "2026-12-01", "Stale attempt", 2); !errors.As(err, &conflict) {
		t.Fatalf("stale update err=%v, want revision conflict", err)
	}
	var date string
	var revision int64
	read := func() {
		t.Helper()
		if err := db.QueryRow(`SELECT target_date,revision FROM goals WHERE id='goal-target'`).Scan(&date, &revision); err != nil {
			t.Fatal(err)
		}
	}
	read()
	if date != "2026-11-01" || revision != 3 {
		t.Fatalf("stale update stored %q / %d", date, revision)
	}
	history, err := repo.ListTargetDateHistory(ctx, "goal-target")
	if err != nil || len(history) != 0 {
		t.Fatalf("stale update wrote history=%#v err=%v", history, err)
	}
	if err := repo.SetTargetDate(ctx, "goal-target", "2026-12-01", "New delivery date", 3); err != nil {
		t.Fatalf("current revision update: %v", err)
	}
	if err := repo.SetTargetDate(ctx, "goal-target", "2027-01-15", "Scope expanded", 4); err != nil {
		t.Fatalf("second current revision update: %v", err)
	}
	read()
	if date != "2027-01-15" || revision != 5 {
		t.Fatalf("current update stored %q / %d", date, revision)
	}
	history, err = repo.ListTargetDateHistory(ctx, "goal-target")
	if err != nil || len(history) != 2 {
		t.Fatalf("history=%#v err=%v", history, err)
	}
	if history[0].PreviousTargetDate != "2026-11-01" || history[0].TargetDate != "2026-12-01" || history[0].Reason != "New delivery date" || history[0].Revision != 4 || history[0].ChangedAt == "" {
		t.Fatalf("first history item=%#v", history[0])
	}
	if history[1].PreviousTargetDate != "2026-12-01" || history[1].TargetDate != "2027-01-15" || history[1].Reason != "Scope expanded" || history[1].Revision != 5 {
		t.Fatalf("second history item=%#v", history[1])
	}
	var notFound ErrGoalNotFound
	if err := repo.SetTargetDate(ctx, "missing", "2026-12-01", "Missing goal", 1); !errors.As(err, &notFound) {
		t.Fatalf("missing goal update err=%v, want not found", err)
	}
	if _, err := repo.ListTargetDateHistory(ctx, "missing"); !errors.As(err, &notFound) {
		t.Fatalf("missing goal history err=%v, want not found", err)
	}
}

func TestSQLiteMeasuredGoalCreateReadRoundTripKeepsProgressUnknown(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	service := NewService(repo)
	for _, measure := range []*Measure{
		{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "one_time", Period: "once"},
		{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"},
	} {
		date := "2026-11-30"
		if measure.Recurrence == "recurring" {
			date = "2026-12-31"
		}
		created, err := service.Create(goalTestContext("owner-a"), CreateInput{Title: measure.Recurrence, TargetBasisPoints: 10000, TargetDate: date, Measure: measure})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.UpdateProgress(goalTestContext("owner-a"), created.ID, 5000, created.Revision); err == nil {
			t.Fatal("measured goals must remain unknown until observations are supported")
		}
		goals, err := service.List(goalTestContext("owner-a"))
		if err != nil {
			t.Fatal(err)
		}
		var read *Goal
		for i := range goals {
			if goals[i].ID == created.ID {
				read = &goals[i]
				break
			}
		}
		if read == nil || read.Measure == nil || *read.Measure != *measure || read.TargetDate != date || read.ProgressBasisPoints != 0 {
			t.Fatalf("round trip created=%#v read=%#v", created, read)
		}
	}
	// Existing generic goals keep their original explicit progress behavior.
	generic, err := service.Create(goalTestContext("owner-a"), CreateInput{Title: "Generic", TargetBasisPoints: 10000})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateProgress(goalTestContext("owner-a"), generic.ID, 2500, generic.Revision)
	if err != nil || updated.ProgressBasisPoints != 2500 {
		t.Fatalf("generic progress update=%#v err=%v", updated, err)
	}
}

func TestSQLiteGoalsAndMilestonesArePrivateToVerifiedOwnerAndLegacyRowsStayUnowned(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/goals.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	// These represent preexisting rows. Adding the owner table must not infer
	// ownership for either the goal or its children.
	if _, err := db.Exec(`INSERT INTO goals (id,title,status,progress_method,target_basis_points,created_at,updated_at,revision) VALUES ('legacy','Legacy goal','active','milestones',10000,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO milestones (id,goal_id,title,status,created_at,updated_at,revision) VALUES ('legacy-m','legacy','Legacy milestone','open',1,1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goal_target_date_history (goal_id,previous_target_date,target_date,reason,resulting_revision,changed_at) VALUES ('legacy','2026-10-01','2026-11-01','legacy history',2,'2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	rowSnapshot := func(query string) string {
		t.Helper()
		var row string
		if err := db.QueryRow(query).Scan(&row); err != nil {
			t.Fatal(err)
		}
		return row
	}
	legacyGoalBefore := rowSnapshot(`SELECT quote(id)||'|'||quote(title)||'|'||quote(purpose)||'|'||quote(status)||'|'||quote(progress_method)||'|'||quote(progress_basis_points)||'|'||quote(target_basis_points)||'|'||quote(target_date)||'|'||quote(completed_date)||'|'||quote(created_at)||'|'||quote(updated_at)||'|'||quote(revision) FROM goals WHERE id='legacy'`)
	legacyMilestoneBefore := rowSnapshot(`SELECT quote(id)||'|'||quote(goal_id)||'|'||quote(title)||'|'||quote(criteria)||'|'||quote(due_date)||'|'||quote(original_due_date)||'|'||quote(completed_date)||'|'||quote(status)||'|'||quote(created_at)||'|'||quote(updated_at)||'|'||quote(revision) FROM milestones WHERE id='legacy-m'`)
	legacyHistoryBefore := rowSnapshot(`SELECT quote(history_id)||'|'||quote(goal_id)||'|'||quote(previous_target_date)||'|'||quote(target_date)||'|'||quote(reason)||'|'||quote(resulting_revision)||'|'||quote(changed_at) FROM goal_target_date_history WHERE goal_id='legacy'`)
	repo := NewSQLiteRepository(db, schedule.System())
	service := NewService(repo)
	ctxA, ctxB := goalTestContext("actor-a"), goalTestContext("actor-b")
	legacyList, err := repo.List(ctxA)
	if err != nil || len(legacyList) != 0 {
		t.Fatalf("legacy visible to A: %#v err=%v", legacyList, err)
	}
	if _, err := repo.ListMilestones(ctxA, "legacy"); !errors.Is(err, ErrGoalNotFound{ID: "legacy"}) {
		t.Fatalf("legacy goal child read err=%v", err)
	}
	if _, err := repo.CreateMilestone(ctxA, Milestone{GoalID: "legacy", Title: "must not attach"}); !errors.Is(err, ErrGoalNotFound{ID: "legacy"}) {
		t.Fatalf("legacy child create err=%v", err)
	}
	if exists, err := repo.MilestonesExistForGoal(ctxA, "legacy", []string{"legacy-m"}); err != nil || exists {
		t.Fatalf("legacy prerequisite lookup exists=%v err=%v", exists, err)
	}
	if err := repo.SetTargetDate(ctxA, "legacy", "2027-01-01", "", 1); !errors.Is(err, ErrGoalNotFound{ID: "legacy"}) {
		t.Fatalf("legacy goal update err=%v", err)
	}
	if _, err := repo.UpdateProgress(ctxA, "legacy", 10, 1); err == nil {
		t.Fatal("legacy progress update succeeded")
	}
	if _, err := repo.UpdateMilestoneStatus(ctxA, "legacy-m", MilestoneDone, 1); !errors.Is(err, ErrMilestoneNotFound{ID: "legacy-m"}) {
		t.Fatalf("legacy milestone update err=%v", err)
	}
	if _, err := repo.ListTargetDateHistory(ctxA, "legacy"); !errors.Is(err, ErrGoalNotFound{ID: "legacy"}) {
		t.Fatalf("legacy history err=%v", err)
	}
	created, err := service.Create(ctxA, CreateInput{Title: "A private goal", TargetBasisPoints: 10000})
	if err != nil {
		t.Fatal(err)
	}
	milestone, err := service.CreateMilestone(ctxA, CreateMilestoneInput{GoalID: created.ID, Title: "A private milestone"})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := repo.ListMilestones(ctxA, created.ID); err != nil || len(got) != 1 || got[0].ID != milestone.ID {
		t.Fatalf("A milestone read=%#v err=%v", got, err)
	}
	if got, err := repo.List(ctxB); err != nil || len(got) != 0 {
		t.Fatalf("A goal visible to B: %#v err=%v", got, err)
	}
	if _, err := repo.ListMilestones(ctxB, created.ID); !errors.Is(err, ErrGoalNotFound{ID: created.ID}) {
		t.Fatalf("foreign goal child read err=%v", err)
	}
	if _, err := repo.UpdateProgress(ctxB, created.ID, 200, created.Revision); !errors.Is(err, ErrGoalNotFound{ID: created.ID}) {
		t.Fatalf("foreign goal progress update err=%v", err)
	}
	if _, err := repo.UpdateMilestoneStatus(ctxB, milestone.ID, MilestoneDone, milestone.Revision); !errors.Is(err, ErrMilestoneNotFound{ID: milestone.ID}) {
		t.Fatalf("foreign milestone update err=%v", err)
	}
	if err := repo.SetTargetDate(ctxB, created.ID, "2027-01-01", "", created.Revision); !errors.Is(err, ErrGoalNotFound{ID: created.ID}) {
		t.Fatalf("foreign goal target update err=%v", err)
	}
	if _, err := repo.ListTargetDateHistory(ctxB, created.ID); !errors.Is(err, ErrGoalNotFound{ID: created.ID}) {
		t.Fatalf("foreign history err=%v", err)
	}
	if _, err := service.CreateMilestone(ctxB, CreateMilestoneInput{GoalID: created.ID, Title: "foreign child"}); err == nil {
		t.Fatal("foreign actor created a child")
	}
	createdB, err := service.Create(ctxB, CreateInput{Title: "B private goal", TargetBasisPoints: 10000})
	if err != nil {
		t.Fatal(err)
	}
	milestoneB, err := service.CreateMilestone(ctxB, CreateMilestoneInput{GoalID: createdB.ID, Title: "B private milestone"})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := repo.ListMilestones(ctxB, createdB.ID); err != nil || len(got) != 1 || got[0].ID != milestoneB.ID {
		t.Fatalf("B milestone read=%#v err=%v", got, err)
	}
	if got, err := repo.List(ctxB); err != nil || len(got) != 1 || got[0].ID != createdB.ID {
		t.Fatalf("B listing=%#v err=%v", got, err)
	}
	if got, err := repo.List(ctxA); err != nil || len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("A listing=%#v err=%v", got, err)
	}
	var legacyTitle, legacyStatus, legacyProgressMethod, legacyTargetDate, legacyCompletedDate string
	var legacyMilestoneTitle, legacyMilestoneStatus string
	var legacyTarget, legacyCreated, legacyUpdated, legacyRevision int64
	var ownerCount int
	if err := db.QueryRow(`SELECT title,status,progress_method,target_basis_points,COALESCE(target_date,''),COALESCE(completed_date,''),created_at,updated_at,revision FROM goals WHERE id='legacy'`).Scan(&legacyTitle, &legacyStatus, &legacyProgressMethod, &legacyTarget, &legacyTargetDate, &legacyCompletedDate, &legacyCreated, &legacyUpdated, &legacyRevision); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT title,status,created_at,updated_at,revision FROM milestones WHERE id='legacy-m'`).Scan(&legacyMilestoneTitle, &legacyMilestoneStatus, &legacyCreated, &legacyUpdated, &legacyRevision); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM goal_owners WHERE goal_id='legacy'`).Scan(&ownerCount); err != nil {
		t.Fatal(err)
	}
	var historyCount int
	if err := db.QueryRow(`SELECT count(*) FROM goal_target_date_history WHERE goal_id='legacy' AND reason='legacy history'`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if got := rowSnapshot(`SELECT quote(id)||'|'||quote(title)||'|'||quote(purpose)||'|'||quote(status)||'|'||quote(progress_method)||'|'||quote(progress_basis_points)||'|'||quote(target_basis_points)||'|'||quote(target_date)||'|'||quote(completed_date)||'|'||quote(created_at)||'|'||quote(updated_at)||'|'||quote(revision) FROM goals WHERE id='legacy'`); got != legacyGoalBefore {
		t.Fatalf("legacy goal row changed: before=%q after=%q", legacyGoalBefore, got)
	}
	if got := rowSnapshot(`SELECT quote(id)||'|'||quote(goal_id)||'|'||quote(title)||'|'||quote(criteria)||'|'||quote(due_date)||'|'||quote(original_due_date)||'|'||quote(completed_date)||'|'||quote(status)||'|'||quote(created_at)||'|'||quote(updated_at)||'|'||quote(revision) FROM milestones WHERE id='legacy-m'`); got != legacyMilestoneBefore {
		t.Fatalf("legacy milestone row changed: before=%q after=%q", legacyMilestoneBefore, got)
	}
	if got := rowSnapshot(`SELECT quote(history_id)||'|'||quote(goal_id)||'|'||quote(previous_target_date)||'|'||quote(target_date)||'|'||quote(reason)||'|'||quote(resulting_revision)||'|'||quote(changed_at) FROM goal_target_date_history WHERE goal_id='legacy'`); got != legacyHistoryBefore {
		t.Fatalf("legacy history row changed: before=%q after=%q", legacyHistoryBefore, got)
	}
	if legacyTitle != "Legacy goal" || legacyStatus != StatusActive || legacyProgressMethod != ProgressMilestones || legacyTarget != 10000 || legacyTargetDate != "" || legacyCompletedDate != "" || legacyCreated != 1 || legacyUpdated != 1 || legacyRevision != 1 || legacyMilestoneTitle != "Legacy milestone" || legacyMilestoneStatus != MilestoneOpen || historyCount != 1 || ownerCount != 0 {
		t.Fatalf("legacy rows changed: goal=%q/%q/%q/%d/%q/%q/%d/%d/%d milestone=%q/%q history=%d owners=%d", legacyTitle, legacyStatus, legacyProgressMethod, legacyTarget, legacyTargetDate, legacyCompletedDate, legacyCreated, legacyUpdated, legacyRevision, legacyMilestoneTitle, legacyMilestoneStatus, historyCount, ownerCount)
	}
}

func TestSQLiteGoalsRequireVerifiedPrincipalForEveryGoalOperation(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	missing := context.Background()
	unverified := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "claimed", Verified: false})
	for name, ctx := range map[string]context.Context{"missing": missing, "unverified": unverified} {
		t.Run(name, func(t *testing.T) {
			if _, err := repo.List(ctx); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("List err=%v", err)
			}
			if _, err := repo.Create(ctx, Goal{Title: "x"}); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("Create err=%v", err)
			}
			if err := repo.SetTargetDate(ctx, "id", "", "", 1); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("SetTargetDate err=%v", err)
			}
			if _, err := repo.ListTargetDateHistory(ctx, "id"); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("history err=%v", err)
			}
			if _, err := repo.UpdateProgress(ctx, "id", 1, 1); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("UpdateProgress err=%v", err)
			}
			if _, err := repo.ListMilestones(ctx, "id"); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("ListMilestones err=%v", err)
			}
			if _, err := repo.CreateMilestone(ctx, Milestone{GoalID: "id"}); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("CreateMilestone err=%v", err)
			}
			if _, err := repo.UpdateMilestoneStatus(ctx, "id", MilestoneDone, 1); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("UpdateMilestoneStatus err=%v", err)
			}
			if _, err := repo.MilestonesExistForGoal(ctx, "id", []string{"child"}); !errors.Is(err, ErrUnverifiedGoalPrincipal{}) {
				t.Fatalf("prerequisite check err=%v", err)
			}
		})
	}
	var goals, owners int
	if err := db.QueryRow(`SELECT count(*) FROM goals`).Scan(&goals); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM goal_owners`).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if goals != 0 || owners != 0 {
		t.Fatalf("invalid principal wrote goal rows=%d owner rows=%d", goals, owners)
	}
}
