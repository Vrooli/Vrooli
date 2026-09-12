package authz

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const matrixDocPath = "../../docs/reference/authorization-matrix.md"

// [REQ:STC-P0-013] The checked-in matrix document is generated from the
// table. Regenerate with STC_WRITE_MATRIX_DOC=1 go test ./authz/ -run TestMatrixDocMatchesTable.
func TestMatrixDocMatchesTable(t *testing.T) {
	want := RenderMarkdown()
	path := filepath.Clean(matrixDocPath)
	if os.Getenv("STC_WRITE_MATRIX_DOC") == "1" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (regenerate with STC_WRITE_MATRIX_DOC=1)", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale; regenerate with STC_WRITE_MATRIX_DOC=1 go test ./authz/ -run TestMatrixDocMatchesTable", path)
	}
}

// [REQ:STC-P0-013] The export carries every table row with its scope.
func TestMatrixHandlerExportsTable(t *testing.T) {
	rec := httptest.NewRecorder()
	MatrixHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/authz/matrix", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var matrix Matrix
	if err := json.Unmarshal(rec.Body.Bytes(), &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.SchemaVersion != MatrixSchemaVersion || matrix.Scenario != Scenario || len(matrix.Routes) != len(Table) || len(matrix.Scopes) != 3 {
		t.Fatalf("matrix = %+v", matrix)
	}
	for _, route := range matrix.Routes {
		if route.Effect != EffectPublic && route.Scope == "" {
			t.Fatalf("exported route without scope: %+v", route)
		}
	}
}
