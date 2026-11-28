package delivery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStableS3URI(t *testing.T) {
	if got, want := StableS3URI("release-bucket", "releases/app.zip"), "s3://release-bucket/releases/app.zip"; got != want {
		t.Fatalf("StableS3URI() = %q, want %q", got, want)
	}
}

func TestValidateDirectArtifactURLRejectsOpenRedirectShapedPaths(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "same-origin relative path", url: "/downloads/app.zip"},
		{name: "HTTPS link", url: "https://downloads.example/app.zip"},
		{name: "legacy HTTP link", url: "http://downloads.example/app.zip"},
		{name: "protocol-relative redirect", url: "//downloads.evil.test/app.zip", wantErr: true},
		{name: "backslash redirect", url: "/\\downloads.evil.test/app.zip", wantErr: true},
		{name: "userinfo", url: "https://user@downloads.example/app.zip", wantErr: true},
		{name: "fragment", url: "https://downloads.example/app.zip#", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDirectArtifactURL(test.url)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateDirectArtifactURL(%q) error = %v, wantErr %v", test.url, err, test.wantErr)
			}
		})
	}
}

func TestUpsertAppRejectsNonPositiveManagedArtifactIDBeforeTransaction(t *testing.T) {
	for _, artifactID := range []int64{0, -1} {
		t.Run(fmt.Sprintf("artifact_id_%d", artifactID), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			_, err = NewCatalogService(db).UpsertApp(App{
				BundleKey: "bundle",
				AppKey:    "app",
				Name:      "App",
				Platforms: []Asset{{
					Platform:       "linux",
					ArtifactSource: "managed",
					ArtifactID:     &artifactID,
				}},
			})
			if err == nil || !strings.Contains(err.Error(), "artifact_id is required") {
				t.Fatalf("UpsertApp error = %v, want managed artifact identity validation", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("database was called before rejecting invalid artifact ID: %v", err)
			}
		})
	}
}

func TestUpsertAppRejectsManagedAssetWithoutChecksumBeforeTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	artifactID := int64(41)
	_, err = NewCatalogService(db).UpsertApp(App{BundleKey: "bundle", AppKey: "app", Name: "App", Platforms: []Asset{{Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3"}}})
	if err == nil || !strings.Contains(err.Error(), "checksum is required") {
		t.Fatalf("UpsertApp error = %v, want managed checksum validation", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database was called before rejecting managed asset without checksum: %v", err)
	}
}

func TestUpsertAssetRejectsManagedAssetWithoutChecksumBeforeDatabaseCall(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	artifactID := int64(41)
	_, err = NewCatalogService(db).UpsertAsset(t.Context(), Asset{BundleKey: "bundle", AppKey: "app", Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3"})
	if err == nil || !strings.Contains(err.Error(), "checksum is required") {
		t.Fatalf("UpsertAsset error = %v, want managed checksum validation", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database was called before rejecting managed asset without checksum: %v", err)
	}
}

func TestUpsertAppRejectsInvalidStorefrontBeforeTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = NewCatalogService(db).UpsertApp(App{
		BundleKey: "bundle",
		AppKey:    "app",
		Name:      "App",
		Storefronts: []Storefront{{
			Store: "app_store", Label: "App Store", URL: "https://apps.apple.com.evil.test/us/app/aquila/id123",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "official Apple App Store host") {
		t.Fatalf("UpsertApp error = %v, want official-host validation", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database was called before rejecting invalid storefront: %v", err)
	}
}
