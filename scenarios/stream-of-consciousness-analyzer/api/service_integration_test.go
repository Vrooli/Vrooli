package main

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// --- SchemeService tests ---

func TestSchemeService_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewSchemeService(db)
	now := time.Now()

	t.Run("with name", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO schemes`).
			WithArgs("My Scheme").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("uuid-1", "My Scheme", now, now))

		s, err := svc.Create(&CreateSchemeInput{Name: "My Scheme"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "My Scheme" {
			t.Errorf("expected name 'My Scheme', got %q", s.Name)
		}
		if s.ID != "uuid-1" {
			t.Errorf("expected id 'uuid-1', got %q", s.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty name defaults to Untitled", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO schemes`).
			WithArgs("Untitled").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("uuid-2", "Untitled", now, now))

		s, err := svc.Create(&CreateSchemeInput{Name: ""})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "Untitled" {
			t.Errorf("expected name 'Untitled', got %q", s.Name)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO schemes`).
			WillReturnError(fmt.Errorf("connection lost"))

		_, err := svc.Create(&CreateSchemeInput{Name: "fail"})
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSchemeService_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewSchemeService(db)
	now := time.Now()

	t.Run("returns schemes", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at, updated_at FROM schemes`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "First", now, now).
				AddRow("s2", "Second", now, now))

		schemes, err := svc.List()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(schemes) != 2 {
			t.Fatalf("expected 2 schemes, got %d", len(schemes))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at, updated_at FROM schemes`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}))

		schemes, err := svc.List()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(schemes) != 0 {
			t.Fatalf("expected 0 schemes, got %d", len(schemes))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("query failed"))

		_, err := svc.List()
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSchemeService_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewSchemeService(db)
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, name, created_at, updated_at FROM schemes WHERE id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))

		s, err := svc.GetByID("s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "s1" {
			t.Errorf("expected id 's1', got %q", s.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		_, err := svc.GetByID("missing")
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSchemeService_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewSchemeService(db)
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE schemes SET name`).
			WithArgs("Updated", "s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Updated", now, now))

		s, err := svc.Update("s1", &UpdateSchemeInput{Name: "Updated"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "Updated" {
			t.Errorf("expected name 'Updated', got %q", s.Name)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE schemes`).
			WillReturnError(sql.ErrNoRows)

		_, err := svc.Update("missing", &UpdateSchemeInput{Name: "X"})
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSchemeService_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewSchemeService(db)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM schemes WHERE id`).
			WithArgs("s1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := svc.Delete("s1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM schemes WHERE id`).
			WithArgs("missing").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := svc.Delete("missing")
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// --- InformationService tests ---

func TestInformationService_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewInformationService(db)
	now := time.Now()

	t.Run("with type", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO information`).
			WithArgs("scheme-1", "url", "https://example.com", 10.0, 20.0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("info-1", "scheme-1", "url", "https://example.com", 10.0, 20.0, now, now))

		info, err := svc.Create("scheme-1", &CreateInformationInput{
			Type: "url", Content: "https://example.com", CanvasX: 10, CanvasY: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Type != "url" {
			t.Errorf("expected type 'url', got %q", info.Type)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("empty type defaults to text", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO information`).
			WithArgs("scheme-1", "text", "hello", 0.0, 0.0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("info-2", "scheme-1", "text", "hello", 0.0, 0.0, now, now))

		info, err := svc.Create("scheme-1", &CreateInformationInput{Content: "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Type != "text" {
			t.Errorf("expected type 'text', got %q", info.Type)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO information`).WillReturnError(fmt.Errorf("insert failed"))

		_, err := svc.Create("scheme-1", &CreateInformationInput{Content: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInformationService_ListByScheme(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewInformationService(db)
	now := time.Now()

	t.Run("returns items", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM information WHERE scheme_id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("i1", "s1", "text", "note", 0.0, 0.0, now, now))

		items, err := svc.ListByScheme("s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("query failed"))

		_, err := svc.ListByScheme("s1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInformationService_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewInformationService(db)
	now := time.Now()
	content := "updated"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE information SET`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("i1", "s1", "text", "updated", 0.0, 0.0, now, now))

		info, err := svc.Update("i1", &UpdateInformationInput{Content: &content})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Content != "updated" {
			t.Errorf("expected content 'updated', got %q", info.Content)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE information`).WillReturnError(sql.ErrNoRows)

		_, err := svc.Update("missing", &UpdateInformationInput{Content: &content})
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInformationService_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewInformationService(db)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM information WHERE id`).
			WithArgs("i1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := svc.Delete("i1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// --- ThoughtService tests ---

func TestThoughtService_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()
	schemeID := "s1"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO thoughts`).
			WithArgs(&schemeID, "My Thought", "body text", 5.0, 10.0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("t1", &schemeID, "My Thought", "body text", 5.0, 10.0, now, now))

		th, err := svc.Create(&CreateThoughtInput{
			SchemeID: &schemeID, Title: "My Thought", Body: "body text", CanvasX: 5, CanvasY: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.Title != "My Thought" {
			t.Errorf("expected title 'My Thought', got %q", th.Title)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO thoughts`).WillReturnError(fmt.Errorf("insert failed"))

		_, err := svc.Create(&CreateThoughtInput{Title: "fail"})
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()
	schemeID := "s1"
	cols := []string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}

	t.Run("filtered by scheme", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM thoughts WHERE scheme_id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("t1", &schemeID, "Thought 1", "", 0.0, 0.0, now, now))

		thoughts, err := svc.List("s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(thoughts) != 1 {
			t.Fatalf("expected 1, got %d", len(thoughts))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("all thoughts when schemeID empty", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM thoughts ORDER BY`).
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("t1", &schemeID, "A", "", 0.0, 0.0, now, now).
				AddRow("t2", nil, "B", "", 0.0, 0.0, now, now))

		thoughts, err := svc.List("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(thoughts) != 2 {
			t.Fatalf("expected 2, got %d", len(thoughts))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error with scheme filter", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("query failed"))

		_, err := svc.List("s1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error without filter", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("query failed"))

		_, err := svc.List("")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM thoughts WHERE id`).
			WithArgs("t1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("t1", nil, "Found", "body", 0.0, 0.0, now, now))

		th, err := svc.GetByID("t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.Title != "Found" {
			t.Errorf("expected title 'Found', got %q", th.Title)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WithArgs("missing").WillReturnError(sql.ErrNoRows)

		_, err := svc.GetByID("missing")
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()
	title := "Updated"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE thoughts SET`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("t1", nil, "Updated", "", 0.0, 0.0, now, now))

		th, err := svc.Update("t1", &UpdateThoughtInput{Title: &title})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if th.Title != "Updated" {
			t.Errorf("expected title 'Updated', got %q", th.Title)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`UPDATE thoughts`).WillReturnError(sql.ErrNoRows)

		_, err := svc.Update("missing", &UpdateThoughtInput{Title: &title})
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)

	mock.ExpectExec(`DELETE FROM thoughts WHERE id`).
		WithArgs("t1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.Delete("t1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestThoughtService_CreateEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO thought_edges`).
			WithArgs("src", "tgt", "causes").
			WillReturnRows(sqlmock.NewRows([]string{"id", "source_id", "target_id", "label", "created_at"}).
				AddRow("e1", "src", "tgt", "causes", now))

		e, err := svc.CreateEdge("src", &CreateEdgeInput{TargetID: "tgt", Label: "causes"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Label != "causes" {
			t.Errorf("expected label 'causes', got %q", e.Label)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO thought_edges`).WillReturnError(fmt.Errorf("unique violation"))

		_, err := svc.CreateEdge("src", &CreateEdgeInput{TargetID: "tgt", Label: "dup"})
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_ListEdges(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)
	now := time.Now()

	t.Run("returns edges", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM thought_edges WHERE source_id`).
			WithArgs("t1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "source_id", "target_id", "label", "created_at"}).
				AddRow("e1", "t1", "t2", "relates", now))

		edges, err := svc.ListEdges("t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(edges) != 1 {
			t.Fatalf("expected 1 edge, got %d", len(edges))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("query failed"))

		_, err := svc.ListEdges("t1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestThoughtService_DeleteEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewThoughtService(db)

	mock.ExpectExec(`DELETE FROM thought_edges WHERE id`).
		WithArgs("e1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteEdge("e1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// --- ExportService tests ---

func TestExportService_ExportScheme(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewExportService(db)
	now := time.Now()
	schemeID := "s1"

	t.Run("full export with thoughts and edges", func(t *testing.T) {
		// scheme query
		mock.ExpectQuery(`SELECT id, name, created_at, updated_at FROM schemes WHERE id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Export Test", now, now))
		// information query
		mock.ExpectQuery(`SELECT .+ FROM information WHERE scheme_id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("i1", "s1", "text", "note", 0.0, 0.0, now, now))
		// thoughts query
		mock.ExpectQuery(`SELECT .+ FROM thoughts WHERE scheme_id`).
			WithArgs("s1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("t1", &schemeID, "Thought A", "", 0.0, 0.0, now, now).
				AddRow("t2", &schemeID, "Thought B", "", 0.0, 0.0, now, now))
		// edges query
		mock.ExpectQuery(`SELECT DISTINCT .+ FROM thought_edges`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "source_id", "target_id", "label", "created_at"}).
				AddRow("e1", "t1", "t2", "relates", now))

		data, err := svc.ExportScheme("s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data.Scheme.Name != "Export Test" {
			t.Errorf("expected scheme name 'Export Test', got %q", data.Scheme.Name)
		}
		if len(data.Information) != 1 {
			t.Errorf("expected 1 info item, got %d", len(data.Information))
		}
		if len(data.Thoughts) != 2 {
			t.Errorf("expected 2 thoughts, got %d", len(data.Thoughts))
		}
		if len(data.Edges) != 1 {
			t.Errorf("expected 1 edge, got %d", len(data.Edges))
		}
		if data.ExportFormat != ExportFormatVersion {
			t.Errorf("expected format %q, got %q", ExportFormatVersion, data.ExportFormat)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("scheme not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM schemes`).WillReturnError(sql.ErrNoRows)

		_, err := svc.ExportScheme("missing")
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("information query error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))
		mock.ExpectQuery(`SELECT .+ FROM information`).WillReturnError(fmt.Errorf("info query failed"))

		_, err := svc.ExportScheme("s1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("thoughts query error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))
		mock.ExpectQuery(`SELECT .+ FROM information`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}))
		mock.ExpectQuery(`SELECT .+ FROM thoughts`).WillReturnError(fmt.Errorf("thought query failed"))

		_, err := svc.ExportScheme("s1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("edges query error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))
		mock.ExpectQuery(`SELECT .+ FROM information`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}))
		mock.ExpectQuery(`SELECT .+ FROM thoughts WHERE scheme_id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}).
				AddRow("t1", &schemeID, "A", "", 0.0, 0.0, now, now))
		mock.ExpectQuery(`SELECT DISTINCT`).WillReturnError(fmt.Errorf("edges query failed"))

		_, err := svc.ExportScheme("s1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("information scan error", func(t *testing.T) {
		// Scheme succeeds, but information rows have wrong column count → collectRows scan fails
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))
		// Return rows with only 2 columns instead of 8 → scanInformation will fail
		mock.ExpectQuery(`SELECT .+ FROM information`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id"}).
				AddRow("i1", "s1"))

		_, err := svc.ExportScheme("s1")
		if err == nil {
			t.Fatal("expected scan error for information rows")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("thought scan error", func(t *testing.T) {
		// Scheme + information succeed, but thought rows have wrong column count
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Test", now, now))
		mock.ExpectQuery(`SELECT .+ FROM information`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}))
		// Return rows with only 2 columns instead of 8 → scanThought will fail
		mock.ExpectQuery(`SELECT .+ FROM thoughts`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id"}).
				AddRow("t1", &schemeID))

		_, err := svc.ExportScheme("s1")
		if err == nil {
			t.Fatal("expected scan error for thought rows")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no thoughts yields empty edges", func(t *testing.T) {
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
				AddRow("s1", "Empty", now, now))
		mock.ExpectQuery(`SELECT .+ FROM information`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "type", "content", "canvas_x", "canvas_y", "created_at", "updated_at"}))
		mock.ExpectQuery(`SELECT .+ FROM thoughts WHERE scheme_id`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "scheme_id", "title", "body", "canvas_x", "canvas_y", "created_at", "updated_at"}))

		data, err := svc.ExportScheme("s1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data.Edges) != 0 {
			t.Errorf("expected 0 edges, got %d", len(data.Edges))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// --- SuggestionService GenerateSuggestions test ---

func TestSuggestionService_GenerateSuggestions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_ = mock // no DB queries expected for the stub

	t.Run("returns empty suggestions with active provider", func(t *testing.T) {
		svc := NewSuggestionServiceWithEnv(db, func(key string) string {
			return "" // no OpenRouter key, ollama is active by default
		})

		suggestions, provider, err := svc.GenerateSuggestions("scheme-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if provider == nil {
			t.Fatal("expected a provider")
		}
		if provider.Name != "ollama" {
			t.Errorf("expected provider 'ollama', got %q", provider.Name)
		}
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions (stub), got %d", len(suggestions))
		}
	})

	t.Run("no provider available", func(t *testing.T) {
		// Create a service with all providers inactive by making ollama "inactive"
		// Since we can't directly manipulate providers, we test the mock fallback
		svc := &SuggestionService{
			db:        db,
			providers: []LLMProvider{}, // empty = no providers
		}

		_, _, err := svc.GenerateSuggestions("scheme-1")
		if err == nil {
			t.Fatal("expected error when no providers available")
		}
	})
}

// --- Schema test ---

func TestEnsureSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schemes`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := ensureSchema(db); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec(`CREATE TABLE`).WillReturnError(fmt.Errorf("migration failed"))

		if err := ensureSchema(db); err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// --- NewServer and middleware tests ---

func TestNewServer_WiresRoutes(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	srv := NewServer(db)
	if srv.router == nil {
		t.Fatal("expected router to be initialized")
	}
	if srv.schemes == nil {
		t.Fatal("expected schemes service")
	}
	if srv.information == nil {
		t.Fatal("expected information service")
	}
	if srv.thoughts == nil {
		t.Fatal("expected thoughts service")
	}
	if srv.export == nil {
		t.Fatal("expected export service")
	}
	if srv.suggestions == nil {
		t.Fatal("expected suggestions service")
	}
	if srv.Handler() == nil {
		t.Fatal("expected handler")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	srv := NewServer(db)
	handler := srv.Handler()

	t.Run("500 level logs ERROR", func(t *testing.T) {
		// No DB expectations = query will fail = 500
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/schemes", nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != 500 {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("404 level logs WARN", func(t *testing.T) {
		// Use a matched route with non-existent ID to trigger 404 through middleware.
		// gorilla/mux only runs Use() middleware for matched routes, so /nonexistent
		// would bypass loggingMiddleware entirely.
		mock.ExpectQuery(`SELECT .+ FROM schemes WHERE id`).WillReturnError(sql.ErrNoRows)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/schemes/does-not-exist", nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != 404 {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("200 level logs INFO", func(t *testing.T) {
		mock.ExpectPing()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
