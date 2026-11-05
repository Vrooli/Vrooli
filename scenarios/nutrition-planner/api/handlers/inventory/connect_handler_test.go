package inventory

import (
	"context"
	"database/sql"
	"io"
	"log"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/inventory"
	_ "modernc.org/sqlite"
	internal "nutrition-planner/internal/inventory"
	"nutrition-planner/internal/workspace"
)

func TestRecordEventRoundTripsQualitativeAssertion(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(workspace.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(internal.Schema()); err != nil {
		t.Fatal(err)
	}
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, schedule.System()))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner-1", Verified: true})
	savedWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Home", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(internal.NewSQLiteRepository(db), workspaces, log.New(io.Discard, "", 0))
	response, err := handler.RecordEvent(ctx, connect.NewRequest(&v1.RecordEventRequest{WorkspaceId: savedWorkspace.ID, Id: "assert-1", Kind: "assertion", ItemId: "rice", Amount: "unknown", AmountMode: "qualitative", Expression: "Some left", Storage: "pantry", BestBeforeDate: "2026-10-11", UseSoon: true}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Event.AmountMode != "qualitative" || response.Msg.Event.Expression != "Some left" || response.Msg.Event.Amount != "unknown" || response.Msg.Event.Storage != "pantry" || response.Msg.Event.BestBeforeDate != "2026-10-11" || !response.Msg.Event.UseSoon || response.Msg.Event.LotId != "assert-1" {
		t.Fatalf("response=%#v", response.Msg.Event)
	}
	listed, err := handler.ListEvents(ctx, connect.NewRequest(&v1.ListEventsRequest{WorkspaceId: savedWorkspace.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.Events) != 1 || listed.Msg.Events[0].Expression != "Some left" || listed.Msg.Events[0].Storage != "pantry" || listed.Msg.Events[0].BestBeforeDate != "2026-10-11" || !listed.Msg.Events[0].UseSoon || listed.Msg.Events[0].LotId != "assert-1" {
		t.Fatalf("listed=%#v", listed.Msg.Events)
	}
	if listed.Msg.Events[0].Amount != "unknown" || listed.Msg.Events[0].AmountMode != "qualitative" {
		t.Fatalf("persisted assertion=%#v", listed.Msg.Events[0])
	}
}
