package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
)

func TestFixtureRequestAllowedOnlyOnLoopbackDevelopmentAuthority(t *testing.T) {
	tests := []struct {
		name string
		host string
		env  string
		want bool
	}{
		{name: "localhost", host: "localhost:1234", want: true},
		{name: "ipv4 loopback", host: "127.0.0.1:1234", want: true},
		{name: "ipv6 loopback", host: "[::1]:1234", want: true},
		{name: "remote host", host: "lpbs.example.test", want: false},
		{name: "production", host: "localhost:1234", env: "production", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LPBS_ENVIRONMENT", test.env)
			r := httptest.NewRequest("POST", "http://"+test.host+"/api/v1/dev/fixtures/seed", nil)
			if got := fixtureRequestAllowed(r); got != test.want {
				t.Fatalf("fixtureRequestAllowed() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFixtureSeedUsesLeaseDatabaseAndLeavesPrimaryUntouched(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	containerURL := startTestContainerDB(t)
	if err := initializeTestTemplate(containerURL); err != nil {
		t.Fatalf("initialize test template: %v", err)
	}

	sequence := testDatabaseSequence.Add(1)
	primaryName := "lpbs_fixture_primary_" + fmt.Sprint(sequence)
	leaseName := "lpbs_fixture_lease_" + fmt.Sprint(sequence)
	adminURL, err := databaseURLWithName(containerURL, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	for _, name := range []string{primaryName, leaseName} {
		if _, err := admin.Exec(`CREATE DATABASE "` + name + `" TEMPLATE "` + testTemplateDatabaseName + `"`); err != nil {
			t.Fatalf("create fixture database %s: %v", name, err)
		}
		name := name
		t.Cleanup(func() {
			cleanupAdmin, err := sql.Open("postgres", adminURL)
			if err != nil {
				return
			}
			defer cleanupAdmin.Close()
			_, _ = cleanupAdmin.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`)
		})
	}

	primaryURL, err := databaseURLWithName(containerURL, primaryName)
	if err != nil {
		t.Fatal(err)
	}
	leaseURL, err := databaseURLWithName(containerURL, leaseName)
	if err != nil {
		t.Fatal(err)
	}
	primary, err := sql.Open("postgres", primaryURL)
	if err != nil {
		t.Fatal(err)
	}
	routed := database.NewFromPrimary(primary)
	if err := routed.InstallTestPool(context.Background(), leaseURL, "fixture-lease", time.Minute); err != nil {
		_ = routed.Close()
		t.Fatalf("install fixture lease: %v", err)
	}
	t.Cleanup(func() { _ = routed.Close() })

	srv := &Server{db: primary, routedDB: routed}
	req := httptest.NewRequest(http.MethodPost, "http://localhost:1234/api/v1/dev/fixtures/seed", bytes.NewBufferString(`{"email":"lease@example.com","tier":"pro","credit_balance":25,"bundle_key":"browser_automation_studio"}`))
	req = req.WithContext(database.WithTestMode(req.Context()))
	response := httptest.NewRecorder()
	srv.fixtureSeed(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("fixture seed status = %d, body = %s", response.Code, response.Body.String())
	}

	var primaryCount int
	if err := primary.QueryRow(`SELECT COUNT(*) FROM users WHERE email = 'lease@example.com'`).Scan(&primaryCount); err != nil {
		t.Fatalf("query primary fixture: %v", err)
	}
	if primaryCount != 0 {
		t.Fatalf("primary fixture count = %d, want 0", primaryCount)
	}
	var leaseCount int
	if err := routed.QueryRowContext(database.WithTestMode(context.Background()), `SELECT COUNT(*) FROM users WHERE email = 'lease@example.com'`).Scan(&leaseCount); err != nil {
		t.Fatalf("query lease fixture: %v", err)
	}
	if leaseCount != 1 {
		t.Fatalf("lease fixture count = %d, want 1", leaseCount)
	}
}
