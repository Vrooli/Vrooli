package sqlite

import (
	"context"
	"testing"
)

func TestQueryHelpersValidateIdentifiers(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "q"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := svc.Execute(ctx, "q", "CREATE TABLE items(id INTEGER, name TEXT);"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := svc.QueryInsert(ctx, "q", "items", []string{"invalid-name=1"}); err == nil {
		t.Fatalf("expected validation error for invalid column name")
	}
	if err := svc.QueryUpdate(ctx, "q", "items", []string{"1bad=2"}, "id=1"); err == nil {
		t.Fatalf("expected validation error for column starting with digit")
	}
	if _, err := svc.QuerySelect(ctx, "q", "bad/table", "", "", 0); err == nil {
		t.Fatalf("expected validation error for bad table name")
	}
}
