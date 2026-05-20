package api

import "testing"

func TestRoutedDrivers_FlagsNamedPgxImport(t *testing.T) {
	source := []byte(`package db

import "github.com/jackc/pgx/v5"

var _ = pgx.Connect
`)
	got := CheckRoutedDatabaseDrivers(source, "api/internal/db/pool.go")
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(got), got)
	}
}

func TestRoutedDrivers_AllowsBlankPgxImport(t *testing.T) {
	source := []byte(`package main

import (
    _ "github.com/jackc/pgx/v5"
)
`)
	got := CheckRoutedDatabaseDrivers(source, "api/main.go")
	if len(got) != 0 {
		t.Fatalf("blank import should be allowed, got %+v", got)
	}
}

func TestRoutedDrivers_FlagsSQLOpenDB(t *testing.T) {
	source := []byte(`package db

import (
    "database/sql"
    "database/sql/driver"
)

func open(c driver.Connector) *sql.DB { return sql.OpenDB(c) }
`)
	got := CheckRoutedDatabaseDrivers(source, "api/internal/db/pool.go")
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(got), got)
	}
}

func TestRoutedDrivers_ExemptsAPICore(t *testing.T) {
	source := []byte(`package database

import "modernc.org/sqlite"

var _ = sqlite.Version
`)
	got := CheckRoutedDatabaseDrivers(source, "packages/api-core/database/routed.go")
	if len(got) != 0 {
		t.Fatalf("api-core path should be exempt, got %+v", got)
	}
}

func TestRoutedDrivers_ExemptsTestFiles(t *testing.T) {
	source := []byte(`package db

import "github.com/jackc/pgx/v5"

var _ = pgx.Connect
`)
	got := CheckRoutedDatabaseDrivers(source, "api/internal/db/pool_test.go")
	if len(got) != 0 {
		t.Fatalf("_test.go should be exempt, got %+v", got)
	}
}

func TestRoutedHandle_FlagsStructField(t *testing.T) {
	source := []byte(`package server

import "database/sql"

type Server struct {
    DB *sql.DB
}
`)
	got := CheckRoutedDatabaseHandleCapture(source, "api/internal/server/server.go")
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(got), got)
	}
}

func TestRoutedHandle_FlagsPackageVar(t *testing.T) {
	source := []byte(`package main

import "database/sql"

var db *sql.DB
`)
	got := CheckRoutedDatabaseHandleCapture(source, "api/main.go")
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(got), got)
	}
}

func TestRoutedHandle_FlagsFuncParam(t *testing.T) {
	source := []byte(`package main

import "database/sql"

func handle(db *sql.DB) error { return nil }
`)
	got := CheckRoutedDatabaseHandleCapture(source, "api/internal/handlers/x.go")
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(got), got)
	}
}

func TestRoutedHandle_AllowsRoutedDBField(t *testing.T) {
	source := []byte(`package server

import "github.com/vrooli/api-core/database"

type Server struct {
    DB *database.RoutedDB
}
`)
	got := CheckRoutedDatabaseHandleCapture(source, "api/internal/server/server.go")
	if len(got) != 0 {
		t.Fatalf("RoutedDB field should be allowed, got %+v", got)
	}
}

func TestRoutedHandle_ExemptsAPICore(t *testing.T) {
	source := []byte(`package database

import "database/sql"

type pool struct { db *sql.DB }
`)
	got := CheckRoutedDatabaseHandleCapture(source, "packages/api-core/database/pool.go")
	if len(got) != 0 {
		t.Fatalf("api-core path should be exempt, got %+v", got)
	}
}
