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
		svc := destinations.NewService(repo, eng, protected)

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
		svc := destinations.NewService(repo, eng, protected)

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
		svc := destinations.NewService(repo, eng, protected)

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

	t.Run("missing name rejected", func(t *testing.T) {
		svc := destinations.NewService(mocks.NewFakeRepository(), &enginemocks.FakeKopiaEngine{}, protected)
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{Backend: destinations.BackendFilesystem, Location: "/x"})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) || invalid.Field != "name" {
			t.Fatalf("expected ErrInvalidDestination{name}, got %v", err)
		}
	})

	t.Run("missing location rejected", func(t *testing.T) {
		svc := destinations.NewService(mocks.NewFakeRepository(), &enginemocks.FakeKopiaEngine{}, protected)
		_, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "x", Backend: destinations.BackendFilesystem})
		var invalid destinations.ErrInvalidDestination
		if !errors.As(err, &invalid) || invalid.Field != "location" {
			t.Fatalf("expected ErrInvalidDestination{location}, got %v", err)
		}
	})
}
