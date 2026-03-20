package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestScanScheme(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow("s1", "Test Scheme", now, now),
	)

	row := db.QueryRow("SELECT id, name, created_at, updated_at FROM schemes")
	s, err := scanScheme(row)
	if err != nil {
		t.Fatalf("scanScheme: %v", err)
	}
	if s.ID != "s1" || s.Name != "Test Scheme" {
		t.Errorf("unexpected scheme: %+v", s)
	}
}

func TestScanThought(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	sid := "scheme-1"
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
			AddRow("t1", &sid, "Title", "Body", 1.0, 2.0, now, now),
	)

	row := db.QueryRow("SELECT id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at FROM thoughts")
	th, err := scanThought(row)
	if err != nil {
		t.Fatalf("scanThought: %v", err)
	}
	if th.ID != "t1" || th.Title != "Title" || th.CanvasX != 1.0 {
		t.Errorf("unexpected thought: %+v", th)
	}
}

func TestScanEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "source_id", "target_id", "label", "created_at"}).
			AddRow("e1", "src", "tgt", "causes", now),
	)

	row := db.QueryRow("SELECT id, source_id, target_id, label, created_at FROM thought_edges")
	e, err := scanEdge(row)
	if err != nil {
		t.Fatalf("scanEdge: %v", err)
	}
	if e.ID != "e1" || e.Label != "causes" {
		t.Errorf("unexpected edge: %+v", e)
	}
}

func TestScanInformation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
			AddRow("i1", "s1", "text", "hello", 0.0, 0.0, now, now),
	)

	row := db.QueryRow("SELECT id, scheme_id, type, content, canvas_x, canvas_y, created_at, updated_at FROM information")
	info, err := scanInformation(row)
	if err != nil {
		t.Fatalf("scanInformation: %v", err)
	}
	if info.ID != "i1" || info.Content != "hello" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestCollectRows_MultipleRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow("s1", "First", now, now).
			AddRow("s2", "Second", now, now),
	)

	rows, err := db.Query("SELECT id, name, created_at, updated_at FROM schemes")
	if err != nil {
		t.Fatal(err)
	}
	schemes, err := collectRows(rows, scanScheme)
	if err != nil {
		t.Fatalf("collectRows: %v", err)
	}
	if len(schemes) != 2 {
		t.Errorf("expected 2 schemes, got %d", len(schemes))
	}
}

func TestCollectRows_EmptyReturnsNonNilSlice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}),
	)

	rows, err := db.Query("SELECT id, name, created_at, updated_at FROM schemes")
	if err != nil {
		t.Fatal(err)
	}
	schemes, err := collectRows(rows, scanScheme)
	if err != nil {
		t.Fatalf("collectRows: %v", err)
	}
	if schemes == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(schemes) != 0 {
		t.Errorf("expected 0 schemes, got %d", len(schemes))
	}
}

func TestCollectRows_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Wrong number of columns triggers scan error
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow("only-one-col"),
	)

	rows, err := db.Query("SELECT id FROM schemes")
	if err != nil {
		t.Fatal(err)
	}
	_, err = collectRows(rows, scanScheme)
	if err == nil {
		t.Error("expected scan error, got nil")
	}
}

func TestCollectRows_RowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow("s1", "OK", now, now).
			RowError(0, fmt.Errorf("row iteration error")),
	)

	rows, err := db.Query("SELECT id, name, created_at, updated_at FROM schemes")
	if err != nil {
		t.Fatal(err)
	}
	_, err = collectRows(rows, scanScheme)
	if err == nil {
		t.Error("expected rows.Err(), got nil")
	}
}

// Verify that sqlmock.Row satisfies the scanner interface used by scan helpers.
func TestScanScheme_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}),
	)

	row := db.QueryRow("SELECT id, name, created_at, updated_at FROM schemes WHERE id = $1", "missing")
	_, err = scanScheme(row)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
