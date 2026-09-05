package main

import (
	"database/sql"
	"testing"
)

func TestDialectHelperPlaceholderSupportsMultiDigitIndices(t *testing.T) {
	helper := NewDialectHelper("postgres")
	if got := helper.Placeholder(12); got != "$12" {
		t.Fatalf("Placeholder(12) = %q, want $12", got)
	}
}

func TestNewDialectHelper_Postgres(t *testing.T) {
	h := NewDialectHelper("postgres")
	if !h.IsPostgres() {
		t.Error("Expected IsPostgres() to return true")
	}
	if h.IsSQLite() {
		t.Error("Expected IsSQLite() to return false")
	}
	if h.Dialect() != "postgres" {
		t.Errorf("Expected dialect 'postgres', got '%s'", h.Dialect())
	}
}

func TestNewDialectHelper_SQLite(t *testing.T) {
	h := NewDialectHelper("sqlite")
	if h.IsPostgres() {
		t.Error("Expected IsPostgres() to return false")
	}
	if !h.IsSQLite() {
		t.Error("Expected IsSQLite() to return true")
	}
	if h.Dialect() != "sqlite" {
		t.Errorf("Expected dialect 'sqlite', got '%s'", h.Dialect())
	}
}

func TestNewDialectHelper_DefaultsToPostgres(t *testing.T) {
	h := NewDialectHelper("")
	if !h.IsPostgres() {
		t.Error("Expected IsPostgres() to return true for empty dialect")
	}
	if h.Dialect() != "postgres" {
		t.Errorf("Expected dialect 'postgres', got '%s'", h.Dialect())
	}
}

func TestDialectHelper_NowExpr_Postgres(t *testing.T) {
	h := NewDialectHelper("postgres")
	nowExpr := h.NowExpr()
	if nowExpr != "NOW()" {
		t.Errorf("Expected 'NOW()', got '%s'", nowExpr)
	}
}

func TestDialectHelper_NowExpr_SQLite(t *testing.T) {
	h := NewDialectHelper("sqlite")
	nowExpr := h.NowExpr()
	if nowExpr != "datetime('now')" {
		t.Errorf("Expected \"datetime('now')\", got '%s'", nowExpr)
	}
}

func TestNullStringValue_Valid(t *testing.T) {
	ns := sql.NullString{String: "hello", Valid: true}
	result := NullStringValue(ns)
	if result == nil {
		t.Fatal("Expected non-nil result for valid NullString")
	}
	if *result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", *result)
	}
}

func TestNullStringValue_Invalid(t *testing.T) {
	ns := sql.NullString{Valid: false}
	result := NullStringValue(ns)
	if result != nil {
		t.Error("Expected nil result for invalid NullString")
	}
}

func TestNullStringValue_EmptyString(t *testing.T) {
	ns := sql.NullString{String: "", Valid: true}
	result := NullStringValue(ns)
	if result == nil {
		t.Fatal("Expected non-nil result for valid empty string")
	}
	if *result != "" {
		t.Errorf("Expected empty string, got '%s'", *result)
	}
}

func TestNullInt64Value_Valid(t *testing.T) {
	ni := sql.NullInt64{Int64: 42, Valid: true}
	result := NullInt64Value(ni)
	if result == nil {
		t.Fatal("Expected non-nil result for valid NullInt64")
	}
	if *result != 42 {
		t.Errorf("Expected 42, got %d", *result)
	}
}

func TestNullInt64Value_Invalid(t *testing.T) {
	ni := sql.NullInt64{Valid: false}
	result := NullInt64Value(ni)
	if result != nil {
		t.Error("Expected nil result for invalid NullInt64")
	}
}

func TestNullInt64Value_Zero(t *testing.T) {
	ni := sql.NullInt64{Int64: 0, Valid: true}
	result := NullInt64Value(ni)
	if result == nil {
		t.Fatal("Expected non-nil result for valid zero")
	}
	if *result != 0 {
		t.Errorf("Expected 0, got %d", *result)
	}
}

func TestNullFloat64Value_Valid(t *testing.T) {
	nf := sql.NullFloat64{Float64: 3.14, Valid: true}
	result := NullFloat64Value(nf)
	if result == nil {
		t.Fatal("Expected non-nil result for valid NullFloat64")
	}
	if *result != 3.14 {
		t.Errorf("Expected 3.14, got %f", *result)
	}
}

func TestNullFloat64Value_Invalid(t *testing.T) {
	nf := sql.NullFloat64{Valid: false}
	result := NullFloat64Value(nf)
	if result != nil {
		t.Error("Expected nil result for invalid NullFloat64")
	}
}

func TestNullBoolValue_Valid(t *testing.T) {
	nb := sql.NullBool{Bool: true, Valid: true}
	result := NullBoolValue(nb)
	if result == nil {
		t.Fatal("Expected non-nil result for valid NullBool")
	}
	if *result != true {
		t.Errorf("Expected true, got %v", *result)
	}
}

func TestNullBoolValue_ValidFalse(t *testing.T) {
	nb := sql.NullBool{Bool: false, Valid: true}
	result := NullBoolValue(nb)
	if result == nil {
		t.Fatal("Expected non-nil result for valid false")
	}
	if *result != false {
		t.Errorf("Expected false, got %v", *result)
	}
}

func TestNullBoolValue_Invalid(t *testing.T) {
	nb := sql.NullBool{Valid: false}
	result := NullBoolValue(nb)
	if result != nil {
		t.Error("Expected nil result for invalid NullBool")
	}
}

func TestStringToNullString_NonNil(t *testing.T) {
	s := "hello"
	ns := StringToNullString(&s)
	if !ns.Valid {
		t.Error("Expected Valid to be true")
	}
	if ns.String != "hello" {
		t.Errorf("Expected 'hello', got '%s'", ns.String)
	}
}

func TestStringToNullString_Nil(t *testing.T) {
	ns := StringToNullString(nil)
	if ns.Valid {
		t.Error("Expected Valid to be false for nil input")
	}
}

func TestStringToNullString_EmptyString(t *testing.T) {
	s := ""
	ns := StringToNullString(&s)
	if !ns.Valid {
		t.Error("Expected Valid to be true for empty string")
	}
	if ns.String != "" {
		t.Errorf("Expected empty string, got '%s'", ns.String)
	}
}

func TestInt64ToNullInt64_NonNil(t *testing.T) {
	i := int64(42)
	ni := Int64ToNullInt64(&i)
	if !ni.Valid {
		t.Error("Expected Valid to be true")
	}
	if ni.Int64 != 42 {
		t.Errorf("Expected 42, got %d", ni.Int64)
	}
}

func TestInt64ToNullInt64_Nil(t *testing.T) {
	ni := Int64ToNullInt64(nil)
	if ni.Valid {
		t.Error("Expected Valid to be false for nil input")
	}
}

func TestInt64ToNullInt64_Zero(t *testing.T) {
	i := int64(0)
	ni := Int64ToNullInt64(&i)
	if !ni.Valid {
		t.Error("Expected Valid to be true for zero")
	}
	if ni.Int64 != 0 {
		t.Errorf("Expected 0, got %d", ni.Int64)
	}
}
