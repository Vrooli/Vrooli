package notes

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type censusConnector struct{ records [][]driver.Value }

func (c censusConnector) Connect(context.Context) (driver.Conn, error) {
	return censusConn{records: c.records}, nil
}
func (c censusConnector) Driver() driver.Driver { return censusDriver{records: c.records} }

type censusDriver struct{ records [][]driver.Value }

func (d censusDriver) Open(string) (driver.Conn, error) { return censusConn{records: d.records}, nil }

type censusConn struct{ records [][]driver.Value }

func (c censusConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (c censusConn) Close() error                        { return nil }
func (c censusConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (c censusConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "WHERE owner_id=$1 AND alias=$2") && len(args) == 2 && args[0].Value == "notes-owner" && args[1].Value == "notes-alias" {
		return &censusRows{records: c.records}, nil
	}
	if strings.Contains(query, "FROM notes_fleet_generations ORDER BY owner_id, alias") && len(args) == 0 {
		return &censusRows{records: c.records}, nil
	}
	{
		return nil, errors.New("Notes census query did not use its explicit owner and alias")
	}
}

type censusRows struct {
	records [][]driver.Value
	index   int
}

func (*censusRows) Columns() []string {
	return []string{"arm_id", "owner_id", "alias", "recipe_id", "snapshot_id", "snapshot_watermark", "catchup_watermark", "dimensions", "collection", "rollback_to", "state", "cursor", "version", "protected", "updated_at"}
}
func (*censusRows) Close() error { return nil }
func (r *censusRows) Next(dest []driver.Value) error {
	if r.index >= len(r.records) {
		return io.EOF
	}
	copy(dest, r.records[r.index])
	r.index++
	return nil
}

func TestCensusOwnerAliasReturnsDurableGenerationFacts(t *testing.T) {
	observed := time.Date(2026, 11, 12, 12, 30, 0, 0, time.UTC)
	rows := [][]driver.Value{
		{"arm-new", "notes-owner", "notes-alias", "recipe-new", "snap-2", "12", "18", int64(768), "notes-v2", "notes-v1", "active", "", int64(4), true, observed},
		{"arm-old", "notes-owner", "notes-alias", "recipe-old", "snap-1", "3", "11", int64(384), "notes-v1", "", "retired", "", int64(7), true, observed.Add(-time.Hour)},
	}
	db := sql.OpenDB(censusConnector{records: rows})
	defer db.Close()
	items, err := (FleetCatalog{DB: db}).CensusOwnerAlias(context.Background(), " notes-owner ", "notes-alias")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ArmID != "arm-new" || items[0].RecipeID != "recipe-new" || items[0].Dimensions != 768 || items[0].State != "active" {
		t.Fatalf("census did not preserve active owner facts: %+v", items)
	}
	if items[1].ArmID != "arm-old" || items[1].Collection != "notes-v1" || items[1].State != "retired" {
		t.Fatalf("census did not preserve predecessor generation: %+v", items)
	}
}

func TestCensusOwnerAliasFailsClosedOnMultipleActiveRows(t *testing.T) {
	observed := time.Now().UTC()
	item := func(id string) []driver.Value {
		return []driver.Value{id, "notes-owner", "notes-alias", "recipe-" + id, "snap", "1", "1", int64(768), "collection-" + id, "", "active", "", int64(1), true, observed}
	}
	db := sql.OpenDB(censusConnector{records: [][]driver.Value{item("arm-a"), item("arm-b")}})
	defer db.Close()
	if _, err := (FleetCatalog{DB: db}).CensusOwnerAlias(context.Background(), "notes-owner", "notes-alias"); err == nil {
		t.Fatal("census accepted two active generations for one alias")
	}
}

func TestCensusOwnerAliasRequiresExplicitIdentity(t *testing.T) {
	if _, err := (FleetCatalog{}).CensusOwnerAlias(context.Background(), "notes-owner", "notes-alias"); err == nil {
		t.Fatal("census accepted an absent owner database")
	}
	if _, err := (FleetCatalog{DB: sql.OpenDB(censusConnector{})}).CensusOwnerAlias(context.Background(), "", "notes-alias"); err == nil {
		t.Fatal("census accepted a missing owner identity")
	}
}

func TestCensusAllGenerationsReturnsOwnerFactsAndRejectsDuplicateActiveAlias(t *testing.T) {
	observed := time.Date(2026, 11, 12, 13, 0, 0, 0, time.UTC)
	item := func(owner, alias, id, state string) []driver.Value {
		return []driver.Value{id, owner, alias, "recipe-" + id, "snapshot-" + id, "1", "2", int64(768), "collection-" + id, "", state, "", int64(1), state == "active", observed}
	}
	db := sql.OpenDB(censusConnector{records: [][]driver.Value{
		item("owner-a", "alias-a", "active-a", "active"),
		item("owner-a", "alias-a", "old-a", "retired"),
		item("owner-b", "alias-b", "active-b", "active"),
	}})
	defer db.Close()
	items, err := (FleetCatalog{DB: db}).CensusAllGenerations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].OwnerID != "owner-a" || items[0].RecipeID != "recipe-active-a" || items[2].OwnerID != "owner-b" {
		t.Fatalf("owner-wide census did not preserve catalog facts: %+v", items)
	}
	duplicate := sql.OpenDB(censusConnector{records: [][]driver.Value{
		item("owner-a", "alias-a", "active-a", "active"),
		item("owner-a", "alias-a", "active-b", "active"),
	}})
	defer duplicate.Close()
	if _, err := (FleetCatalog{DB: duplicate}).CensusAllGenerations(context.Background()); err == nil {
		t.Fatal("owner-wide census accepted multiple active generations for one alias")
	}
}
