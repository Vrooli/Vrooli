package main

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"landing-page-business-suite-api/internal/delivery"
)

func TestSeedDownloadDefaultsUsesExactOwnerCatalogOnly(t *testing.T) {
	apps, err := delivery.DefaultDownloadSeed()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 19 {
		t.Fatalf("seed app count = %d, want 19", len(apps))
	}
	if apps[0].AppKey != "browser-automation-studio" || apps[0].Metadata["enabled"] != false || apps[0].Metadata["catalog_status"] != "planned" {
		t.Fatalf("BAS seed availability changed: %#v", apps[0])
	}
	for _, asset := range apps[0].Platforms {
		if !asset.RequiresEntitlement {
			t.Fatalf("BAS asset %q lost entitlement gating", asset.Platform)
		}
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, app := range apps {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO download_apps")).WillReturnResult(sqlmock.NewResult(1, 1))
		for range app.Platforms {
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO download_assets")).WillReturnResult(sqlmock.NewResult(1, 1))
		}
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE download_apps")).WithArgs("business_suite").WillReturnResult(sqlmock.NewResult(0, 1))

	if err := seedDownloadDefaults(db, apps); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
