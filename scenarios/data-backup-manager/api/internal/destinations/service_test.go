package destinations_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

// TestCreateDestinations_FsAndS3 proves that:
//  1. A filesystem destination and an S3 destination can be created successfully.
//  2. RepoCreate was called with the right backend for each.
//  3. RepoStatus was called and its EncryptionAlgorithm is captured.
//  4. A filesystem destination whose location is under protectedRoot is rejected
//     with ErrInvalidDestination (separate-root rule).
func TestCreateDestinations_FsAndS3(t *testing.T) {
	ctx := context.Background()
	const protected = "/data/backup"

	t.Run("filesystem destination created successfully", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), protected)

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "fs-dest",
			Backend:  destinations.BackendFilesystem,
			Location: "/mnt/backup",
		})
		if err != nil {
			t.Fatalf("CreateDestination: %v", err)
		}
		if d.ID == "" {
			t.Fatal("created destination has empty id")
		}
		if d.BackendKind != destinations.BackendFilesystem {
			t.Fatalf("backend = %q, want filesystem", d.BackendKind)
		}
		// Assert RepoCreate was called with the right backend.
		found := false
		for _, call := range eng.Calls {
			if call == "RepoCreate(fs-dest,filesystem)" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("RepoCreate not called as expected; calls = %v", eng.Calls)
		}
		// Assert RepoStatus was called.
		statusFound := false
		for _, call := range eng.Calls {
			if call == "RepoStatus(fs-dest)" {
				statusFound = true
				break
			}
		}
		if !statusFound {
			t.Fatalf("RepoStatus not called; calls = %v", eng.Calls)
		}
		if d.EncryptionAlgorithm == "" {
			t.Fatal("EncryptionAlgorithm not captured from RepoStatus")
		}
	})

	t.Run("s3 destination created successfully", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), protected)

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "s3-dest",
			Backend:  destinations.BackendS3,
			Location: "my-bucket",
		})
		if err != nil {
			t.Fatalf("CreateDestination s3: %v", err)
		}
		if d.BackendKind != destinations.BackendS3 {
			t.Fatalf("backend = %q, want s3", d.BackendKind)
		}
		found := false
		for _, call := range eng.Calls {
			if call == "RepoCreate(s3-dest,s3)" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("RepoCreate(s3) not called; calls = %v", eng.Calls)
		}
	})

	t.Run("filesystem destination under protectedRoot is rejected", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), protected)

		// Exact match.
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "bad-dest",
			Backend:  destinations.BackendFilesystem,
			Location: protected,
		})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) {
			t.Fatalf("expected ErrInvalidDestination for exact-root, got %v", err)
		}

		// Sub-path.
		_, err = svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "bad-dest2",
			Backend:  destinations.BackendFilesystem,
			Location: protected + "/subdir",
		})
		if !errors.As(err, &invalid) {
			t.Fatalf("expected ErrInvalidDestination for sub-path, got %v", err)
		}

		// Outside is allowed.
		_, err = svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "ok-dest",
			Backend:  destinations.BackendFilesystem,
			Location: "/mnt/other",
		})
		if err != nil {
			t.Fatalf("destination outside protected root should succeed, got %v", err)
		}
	})

	t.Run("filesystem create writes bundle with nested repository path", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		repo := mocks.NewFakeRepository()
		bundle := mocks.NewFakeBundleWriter()
		svc := destinations.NewService(repo, eng, bundle, protected)

		d, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "elements-local",
			Backend:  destinations.BackendFilesystem,
			Location: "/mnt/backup",
		})
		if err != nil {
			t.Fatalf("CreateDestination: %v", err)
		}
		wantRepo := destinations.RepositoryPathFor("/mnt/backup", "elements-local")
		if d.RepositoryLocation != wantRepo {
			t.Fatalf("RepositoryLocation = %q, want %q", d.RepositoryLocation, wantRepo)
		}
		if d.Location != "/mnt/backup" {
			t.Fatalf("Location = %q, want bundle root /mnt/backup", d.Location)
		}
		foundRepoSpec := false
		for _, spec := range eng.Repos {
			if spec.Path == wantRepo {
				foundRepoSpec = true
			}
		}
		if !foundRepoSpec {
			t.Fatalf("RepoCreate path = %+v, want %q", eng.Repos, wantRepo)
		}
		if len(bundle.Prepared) != 1 || bundle.Prepared[0].RepositoryPath != wantRepo {
			t.Fatalf("PrepareRepository = %+v, want repo path %q", bundle.Prepared, wantRepo)
		}
		if len(bundle.Metadata) != 1 {
			t.Fatalf("WriteMetadata calls = %d, want 1", len(bundle.Metadata))
		}
		// The bundle manifest carries the engine-derived credential reference so a
		// detached drive points the operator at the passphrase location, not a
		// caller guess. It is a reference, never the secret value.
		wantRef := "vrooli/kopia/elements-local:repository-passphrase"
		if bundle.Metadata[0].SecretRef != wantRef {
			t.Fatalf("manifest secret ref = %q, want %q", bundle.Metadata[0].SecretRef, wantRef)
		}
		if d.SecretRef != wantRef {
			t.Fatalf("persisted secret ref = %q, want %q", d.SecretRef, wantRef)
		}
	})

	t.Run("s3 create does not invoke the filesystem bundle writer", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		bundle := mocks.NewFakeBundleWriter()
		svc := destinations.NewService(mocks.NewFakeRepository(), eng, bundle, protected)

		if _, err := svc.CreateDestination(ctx, destinations.CreateInput{
			Name:     "s3-dest",
			Backend:  destinations.BackendS3,
			Location: "my-bucket",
		}); err != nil {
			t.Fatalf("CreateDestination s3: %v", err)
		}
		if len(bundle.Prepared) != 0 || len(bundle.Metadata) != 0 {
			t.Fatalf("s3 backend must not touch the filesystem bundle writer: %+v / %+v", bundle.Prepared, bundle.Metadata)
		}
	})

	t.Run("non-slug name rejected", func(t *testing.T) {
		svc := destinations.NewService(mocks.NewFakeRepository(), &enginemocks.FakeKopiaEngine{}, mocks.NewFakeBundleWriter(), protected)
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "Elements Local", Backend: destinations.BackendFilesystem, Location: "/mnt/x"})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) || invalid.Field != "name" {
			t.Fatalf("expected ErrInvalidDestination{name} for non-slug name, got %v", err)
		}
	})

	t.Run("missing name rejected", func(t *testing.T) {
		svc := destinations.NewService(mocks.NewFakeRepository(), &enginemocks.FakeKopiaEngine{}, mocks.NewFakeBundleWriter(), protected)
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{Backend: destinations.BackendFilesystem, Location: "/x"})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) || invalid.Field != "name" {
			t.Fatalf("expected ErrInvalidDestination{name}, got %v", err)
		}
	})

	t.Run("missing location rejected", func(t *testing.T) {
		svc := destinations.NewService(mocks.NewFakeRepository(), &enginemocks.FakeKopiaEngine{}, mocks.NewFakeBundleWriter(), protected)
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "x", Backend: destinations.BackendFilesystem})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) || invalid.Field != "location" {
			t.Fatalf("expected ErrInvalidDestination{location}, got %v", err)
		}
	})
}

func TestReconcileCredentialReferencesRepairsCatalogAndBundle(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	legacy := destinations.Destination{
		Name:                "elements-local",
		BackendKind:         destinations.BackendFilesystem,
		Location:            "/mnt/elements",
		RepositoryLocation:  "/mnt/elements/repositories/elements-local.kopia",
		EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "secret/resources/kopia/repo/elements-local/passphrase",
	}
	if _, err := repo.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	bundle := mocks.NewFakeBundleWriter()
	svc := destinations.NewService(repo, &enginemocks.FakeKopiaEngine{}, bundle, "/protected")
	if err := svc.ReconcileCredentialReferences(ctx); err != nil {
		t.Fatalf("ReconcileCredentialReferences: %v", err)
	}
	got, err := repo.GetByName(ctx, legacy.Name)
	if err != nil {
		t.Fatal(err)
	}
	want := "vrooli/kopia/elements-local:repository-passphrase"
	if got.SecretRef != want {
		t.Fatalf("catalog SecretRef = %q, want %q", got.SecretRef, want)
	}
	if len(bundle.Refreshed) != 1 || bundle.Refreshed[0].SecretRef != want {
		t.Fatalf("bundle refreshes = %+v, want one canonical reference", bundle.Refreshed)
	}
	if err := svc.ReconcileCredentialReferences(ctx); err != nil {
		t.Fatalf("idempotent reconciliation: %v", err)
	}
	if len(bundle.Refreshed) != 1 {
		t.Fatalf("idempotent reconciliation refreshed bundle %d times", len(bundle.Refreshed))
	}
}

func TestReconcileCredentialReferencesPersistsCatalogWhenBundleIsReadOnly(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	legacy := destinations.Destination{
		Name:                "elements-local",
		BackendKind:         destinations.BackendFilesystem,
		Location:            "/media/elements",
		RepositoryLocation:  "/media/elements/repositories/elements-local.kopia",
		EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "secret/resources/kopia/repo/elements-local/passphrase",
	}
	if _, err := repo.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	bundle := mocks.NewFakeBundleWriter()
	bundle.RefreshErr = errors.New("read-only destination")
	svc := destinations.NewService(repo, &enginemocks.FakeKopiaEngine{}, bundle, "/protected")
	if err := svc.ReconcileCredentialReferences(ctx); err == nil {
		t.Fatal("ReconcileCredentialReferences succeeded despite pending bundle refresh")
	}
	got, err := repo.GetByName(ctx, legacy.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.SecretRef != "vrooli/kopia/elements-local:repository-passphrase" {
		t.Fatalf("catalog SecretRef = %q, want canonical reference", got.SecretRef)
	}
}
