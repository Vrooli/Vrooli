package destinations_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sysmounts"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

type fixedDestinationProbe struct {
	inspection destinationreadiness.Inspection
}

type testVolumeOperationLease struct {
	active       bool
	acquisitions int
	acquireErr   error
	onAcquire    func()
}

type failOnceMetadataBundleWriter struct {
	destinations.BundleWriter
	fail bool
}

func (w *failOnceMetadataBundleWriter) WriteMetadata(ctx context.Context, meta destinations.BundleMetadata) error {
	if w.fail {
		w.fail = false
		return errors.New("injected one-time metadata write failure")
	}
	return w.BundleWriter.WriteMetadata(ctx, meta)
}

func (l *testVolumeOperationLease) Acquire(_ context.Context, identity destinationreadiness.DeviceIdentity) (func(), error) {
	if l.acquireErr != nil {
		return nil, l.acquireErr
	}
	if !identity.StrongIdentity() || strings.TrimSpace(identity.DevicePath) == "" {
		return nil, errors.New("test lease requires strong observed identity")
	}
	l.active = true
	l.acquisitions++
	if l.onAcquire != nil {
		l.onAcquire()
	}
	return func() { l.active = false }, nil
}

type mutableDestinationProbe struct {
	inspection destinationreadiness.Inspection
}

func (p *mutableDestinationProbe) Analyze(ctx context.Context, input destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	return fixedDestinationProbe{inspection: p.inspection}.Analyze(ctx, input)
}

func (p *mutableDestinationProbe) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return p.inspection, nil
}

func testVolumeLeaseOption() destinations.Option {
	return destinations.WithVolumeOperationLease(&testVolumeOperationLease{})
}

type fixedReadinessReport struct {
	report destinationreadiness.Report
}

func (g fixedReadinessReport) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	return g.report, nil
}

type sequenceReadinessReports struct {
	reports []destinationreadiness.Report
	calls   int
}

func (g *sequenceReadinessReports) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	index := g.calls
	g.calls++
	if index >= len(g.reports) {
		index = len(g.reports) - 1
	}
	return g.reports[index], nil
}

func (p fixedDestinationProbe) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	stateSeverity := destinationreadiness.SeverityUnknown
	if p.inspection.FilesystemState == sysmounts.FilesystemStateClean {
		stateSeverity = destinationreadiness.SeverityPass
	}
	overall := destinationreadiness.SeverityUnknown
	if p.inspection.FilesystemState == sysmounts.FilesystemStateClean {
		overall = destinationreadiness.SeverityPass
	}
	return destinationreadiness.Report{OverallSeverity: overall, Mounted: p.inspection.Mounted, Identity: p.inspection.Identity, Checks: []destinationreadiness.CheckResult{
		{Code: "destination_present", Severity: destinationreadiness.SeverityPass},
		{Code: "directory_inaccessible", Severity: destinationreadiness.SeverityPass},
		{Code: "mounted_read_write", Severity: destinationreadiness.SeverityPass},
		{Code: "filesystem_state", Severity: stateSeverity},
		{Code: "filesystem_suitability", Severity: destinationreadiness.SeverityWarning},
	}}, nil
}

func (p fixedDestinationProbe) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return p.inspection, nil
}

func TestResolveForUseRebindsBoundDestinationAfterMountRelocation(t *testing.T) {
	ctx := context.Background()
	mount := t.TempDir()
	oldMount := filepath.Join(t.TempDir(), "old")
	if err := os.Mkdir(oldMount, 0o700); err != nil {
		t.Fatal(err)
	}
	relative := filepath.Join("vrooli-backups", "bundle")
	bundleRoot := filepath.Join(mount, relative)
	repoPath := destinations.RepositoryPathFor(bundleRoot, "removable")
	oldBundle := filepath.Join(oldMount, relative)
	if err := os.MkdirAll(oldBundle, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "kopia.repository.f"), []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldRepoPath := destinations.RepositoryPathFor(oldBundle, "removable")
	if err := os.MkdirAll(oldRepoPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldRepoPath, "kopia.repository.f"), []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/test1", Mountpoint: mount, Filesystem: "ntfs", UUID: "fixture-uuid", Serial: "fixture-serial"}
	d := destinations.Destination{ID: "dest-1", Name: "removable", BackendKind: destinations.BackendFilesystem, Location: oldBundle, RepositoryLocation: oldRepoPath, RelativePath: relative, DeviceIdentity: &destinationreadiness.DeviceIdentity{DevicePath: "/dev/old", Mountpoint: oldMount, Filesystem: "ntfs", UUID: identity.UUID, Serial: identity.Serial}, SecretRef: "vrooli/kopia/removable:repository-passphrase", EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256"}
	manifestWriter := &destinations.FSBundleWriter{}
	if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	meta := destinations.BundleMetadata{DestinationID: d.ID, Name: d.Name, Backend: string(d.BackendKind), BundleRoot: d.Location, RelativePath: relative, RepositoryPath: d.RepositoryLocation, SecretRef: d.SecretRef, EncryptionAlgorithm: d.EncryptionAlgorithm}
	if err := manifestWriter.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{destinations.BundleManifestFile, destinations.BundleRecoveryFile, destinations.BundleReadmeFile} {
		contents, err := os.ReadFile(filepath.Join(oldBundle, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bundleRoot, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const unrelated = "operator-notes.txt"
	if err := os.WriteFile(filepath.Join(bundleRoot, unrelated), []byte("preserve this drive file"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := mocks.NewFakeRepository()
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	markerPath := filepath.Join(repoPath, "kopia.repository.f")
	if err := os.WriteFile(markerPath, []byte(`{"uniqueID":"different-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	wrongRepoSvc := destinations.NewService(repo, eng, manifestWriter, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), testVolumeLeaseOption())
	if _, err := wrongRepoSvc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("resolver accepted a different encrypted repository under the destination bundle")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("resolver rebound a repository before identity continuity passed: %v", eng.Calls)
	}
	if err := os.WriteFile(markerPath, []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	duplicate := d
	duplicate.ID, duplicate.Name = "dest-2", "alias"
	if _, err := repo.Create(ctx, duplicate); err != nil {
		t.Fatal(err)
	}
	collisionSvc := destinations.NewService(repo, eng, manifestWriter, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), testVolumeLeaseOption())
	if _, err := collisionSvc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("resolver accepted a second destination claiming the same volume-relative bundle")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("resolver rebound before checking destination collision: %v", eng.Calls)
	}
	if _, err := repo.Delete(ctx, "dest-2"); err != nil {
		t.Fatal(err)
	}
	unknown := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateUnknown}}
	blockedSvc := destinations.NewService(repo, eng, manifestWriter, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(unknown), testVolumeLeaseOption())
	if _, err := blockedSvc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("resolver accepted an unknown filesystem state")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("resolver touched resource-kopia before readiness passed: %v", eng.Calls)
	}
	lease := &testVolumeOperationLease{}
	eng.RepoRebindFn = func(context.Context, string, string) error {
		if !lease.active {
			t.Fatal("RepoRebind ran without the host volume lease")
		}
		return nil
	}
	svc := destinations.NewService(repo, eng, manifestWriter, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	repo.UpdateErr = errors.New("simulated catalog write interruption")
	if _, err := svc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("expected catalog projection failure after safe repository rebind")
	}
	if manifest, err := os.ReadFile(filepath.Join(bundleRoot, destinations.BundleManifestFile)); err != nil || !strings.Contains(string(manifest), bundleRoot) {
		t.Fatalf("bundle projection did not reach the resolved path before injected catalog failure: %s / %v", manifest, err)
	}
	repo.UpdateErr = nil
	resolved, err := svc.ResolveForUse(ctx, d.ID)
	if err != nil {
		t.Fatalf("ResolveForUse: %v", err)
	}
	if resolved.Location != bundleRoot || resolved.RepositoryLocation != repoPath {
		t.Fatalf("resolved paths = %q / %q, want %q / %q", resolved.Location, resolved.RepositoryLocation, bundleRoot, repoPath)
	}
	if lease.active || lease.acquisitions != 2 {
		t.Fatalf("lease after ResolveForUse = active:%v acquisitions:%d", lease.active, lease.acquisitions)
	}
	if err := svc.WithResolvedDestination(ctx, d.ID, func(current destinations.Destination) error {
		if !lease.active {
			t.Fatal("operation callback ran without the host volume lease")
		}
		if current.Location != bundleRoot {
			t.Fatalf("callback location = %q, want %q", current.Location, bundleRoot)
		}
		return nil
	}); err != nil {
		t.Fatalf("WithResolvedDestination: %v", err)
	}
	if lease.active || lease.acquisitions != 3 {
		t.Fatalf("lease after operation callback = active:%v acquisitions:%d", lease.active, lease.acquisitions)
	}
	wantCall := "RepoRebind(removable," + repoPath + ")"
	found := false
	for _, call := range eng.Calls {
		if call == wantCall {
			found = true
		}
	}
	if !found {
		t.Fatalf("RepoRebind(%s) not called: %v", repoPath, eng.Calls)
	}
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			t.Fatalf("resolver created a replacement repository: %v", eng.Calls)
		}
	}
	bytes, err := os.ReadFile(filepath.Join(bundleRoot, destinations.BundleManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["bundle_root"] != bundleRoot || manifest["repository_path"] != repoPath {
		t.Fatalf("manifest paths were not rebound: %v", manifest)
	}
	readme, err := os.ReadFile(filepath.Join(bundleRoot, destinations.BundleReadmeFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), repoPath) {
		t.Fatalf("readme path projection was not rebound: %s", readme)
	}
	if preserved, err := os.ReadFile(filepath.Join(bundleRoot, unrelated)); err != nil || string(preserved) != "preserve this drive file" {
		t.Fatalf("unrelated drive content changed: %q / %v", preserved, err)
	}
	eng.RepoDeleteFn = func(context.Context, string) error {
		if !lease.active {
			t.Fatal("RepoDelete ran without the host volume lease")
		}
		return nil
	}
	removed, err := svc.DeleteDestination(ctx, d.ID, true)
	if err != nil || !removed {
		t.Fatalf("DeleteDestination under host lease = removed:%v err:%v", removed, err)
	}
}

func TestImportFilesystemDestinationAttachesAnchoredRepositoryWithoutCreating(t *testing.T) {
	ctx := context.Background()
	mount := t.TempDir()
	bundleRoot := filepath.Join(mount, "vrooli-backups", "offsite")
	repositoryPath := destinations.RepositoryPathFor(bundleRoot, "offsite")
	if err := os.MkdirAll(repositoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryPath, "kopia.repository.f"), []byte(`{"uniqueID":"existing-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := &destinations.FSBundleWriter{}
	meta := destinations.BundleMetadata{
		DestinationID: "stable-destination-id", Name: "offsite", Backend: string(destinations.BackendFilesystem),
		BundleRoot: bundleRoot, RelativePath: "vrooli-backups/offsite", RepositoryPath: repositoryPath,
		SecretRef: "vrooli/kopia/offsite:repository-passphrase", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := writer.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	unrelatedPath := filepath.Join(bundleRoot, "operator-notes.txt")
	if err := os.WriteFile(unrelatedPath, []byte("keep this user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/test-offsite", Mountpoint: mount, Filesystem: "ext4", UUID: "offsite-volume-uuid"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, FilesystemState: sysmounts.FilesystemStateClean}}
	repo := mocks.NewFakeRepository()
	lease := &testVolumeOperationLease{}
	eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
		return []engine.Snapshot{{ID: "snapshot-from-existing-repo"}}, nil
	}}
	eng.RepoRebindFn = func(context.Context, string, string) error {
		if !lease.active {
			t.Fatal("repository reconnect ran outside the volume lease")
		}
		return nil
	}
	svc := destinations.NewService(repo, eng, writer, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	imported, err := svc.ImportExistingDestination(ctx, destinations.ImportExistingInput{
		ID: meta.DestinationID, Name: meta.Name, Backend: destinations.BackendFilesystem,
		Location: bundleRoot, CreatedAt: meta.CreatedAt, ExpectedSnapshotIDs: []string{"snapshot-from-existing-repo"},
	})
	if err != nil {
		t.Fatalf("import existing bundle: %v", err)
	}
	if imported.ID != meta.DestinationID || imported.Name != meta.Name || imported.RepositoryLocation != repositoryPath || imported.ProvisioningState != destinations.ProvisioningReady {
		t.Fatalf("imported destination = %+v", imported)
	}
	if lease.active || lease.acquisitions != 1 {
		t.Fatalf("volume lease was not acquired once and released after import: %+v", lease)
	}
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			t.Fatalf("import must never create/overwrite repository: %v", eng.Calls)
		}
	}
	if len(eng.Calls) != 3 || eng.Calls[0] != "RepoRebind(offsite,"+repositoryPath+")" {
		t.Fatalf("import engine calls = %v", eng.Calls)
	}
	stored, err := repo.GetByID(ctx, meta.DestinationID)
	if err != nil || stored.Name != meta.Name || stored.RelativePath != meta.RelativePath {
		t.Fatalf("imported catalog row = %+v err=%v", stored, err)
	}
	unrelated, err := os.ReadFile(unrelatedPath)
	if err != nil || string(unrelated) != "keep this user data" {
		t.Fatalf("import changed unrelated bundle-root data: %q err=%v", unrelated, err)
	}
}

func TestImportExistingRemoteDestinationBindsDescriptorToExpectedSnapshots(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name     string
		backend  destinations.BackendKind
		location string
		check    func(t *testing.T, spec engine.RepoSpec)
	}{
		{name: "S3", backend: destinations.BackendS3, location: "s3+https://s3.example.test/bucket/dbm?region=us-east-1", check: func(t *testing.T, spec engine.RepoSpec) {
			t.Helper()
			if spec.Backend != string(destinations.BackendS3) || spec.Bucket != "bucket" || spec.Prefix != "dbm" || spec.Region != "us-east-1" {
				t.Fatalf("S3 connect descriptor = %+v", spec)
			}
		}},
		{name: "SFTP", backend: destinations.BackendSFTP, location: "sftp://backup@example.test:22/dbm/repo?known_hosts=%2Fhome%2Foperator%2F.ssh%2Fknown_hosts", check: func(t *testing.T, spec engine.RepoSpec) {
			t.Helper()
			if spec.Backend != string(destinations.BackendSFTP) || spec.SFTPHost != "example.test" || spec.SFTPUsername != "backup" || spec.SFTPRemotePath != "/dbm/repo" {
				t.Fatalf("SFTP connect descriptor = %+v", spec)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			eng := &enginemocks.FakeKopiaEngine{
				RepoConnectFn: func(_ context.Context, spec engine.RepoSpec) error {
					tc.check(t, spec)
					return nil
				},
				SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
					return []engine.Snapshot{{ID: "required-snapshot"}, {ID: "other-snapshot"}}, nil
				},
			}
			svc := destinations.NewService(repo, eng, nil, filepath.Join(t.TempDir(), "protected"))
			got, err := svc.ImportExistingDestination(context.Background(), destinations.ImportExistingInput{
				ID: "stable-remote-id", Name: "offhost", Backend: tc.backend, Location: tc.location,
				CreatedAt: createdAt, ExpectedSnapshotIDs: []string{"required-snapshot"},
			})
			if err != nil {
				t.Fatalf("import existing remote repository: %v", err)
			}
			if got.ID != "stable-remote-id" || got.BackendKind != tc.backend || got.Location != tc.location || got.SecretRef != "vrooli/kopia/offhost:repository-passphrase" {
				t.Fatalf("imported destination = %+v", got)
			}
			if len(eng.Calls) != 3 || eng.Calls[0] != "RepoConnect(offhost,"+string(tc.backend)+")" {
				t.Fatalf("remote import calls = %v", eng.Calls)
			}
			for _, call := range eng.Calls {
				if strings.HasPrefix(call, "RepoCreate(") {
					t.Fatalf("remote import must never create a repository: %v", eng.Calls)
				}
			}
		})
	}
}

func TestImportExistingRemoteDestinationFailsClosedOnMissingCredentialOrSnapshot(t *testing.T) {
	base := destinations.ImportExistingInput{
		ID: "remote-id", Name: "offhost", Backend: destinations.BackendS3,
		Location:  "s3+https://s3.example.test/bucket/dbm?region=us-east-1",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExpectedSnapshotIDs: []string{"required-snapshot"},
	}
	t.Run("missing credential authority value", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{RepoConnectFn: func(context.Context, engine.RepoSpec) error { return errors.New("credential_missing") }}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, nil, filepath.Join(t.TempDir(), "protected"))
		if _, err := svc.ImportExistingDestination(context.Background(), base); err == nil || !strings.Contains(err.Error(), "credential_missing") {
			t.Fatalf("missing credential outcome = %v", err)
		}
		if len(eng.Calls) != 1 || repo.Creates.Load() != 0 {
			t.Fatalf("missing credentials crossed catalog persistence: calls=%v creates=%d", eng.Calls, repo.Creates.Load())
		}
	})
	t.Run("expected snapshot absent", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{SnapshotListFn: func(context.Context, string, string) ([]engine.Snapshot, error) {
			return []engine.Snapshot{{ID: "different-snapshot"}}, nil
		}}
		repo := mocks.NewFakeRepository()
		svc := destinations.NewService(repo, eng, nil, filepath.Join(t.TempDir(), "protected"))
		if _, err := svc.ImportExistingDestination(context.Background(), base); err == nil || !strings.Contains(err.Error(), "expected snapshot") {
			t.Fatalf("snapshot mismatch outcome = %v", err)
		}
		if repo.Creates.Load() != 0 {
			t.Fatal("repository catalog row persisted despite expected-snapshot mismatch")
		}
	})
}

func TestImportFilesystemDestinationRejectsRepositoryAnchorMismatchBeforeConnect(t *testing.T) {
	ctx := context.Background()
	mount := t.TempDir()
	bundleRoot := filepath.Join(mount, "backup")
	repositoryPath := destinations.RepositoryPathFor(bundleRoot, "offsite")
	if err := os.MkdirAll(repositoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryPath, "kopia.repository.f"), []byte(`{"uniqueID":"wrong-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleRoot, destinations.BundleManifestFile), []byte(`{"schema_version":1,"kind":"vrooli.data-backup-manager.destination","producer":"data-backup-manager","destination_id":"id","destination_name":"offsite","backend":"filesystem","bundle_root":"/old/backup","relative_path":"backup","repository_path":"/old/backup/repositories/offsite.kopia","repository_unique_id":"expected-repository","encryption_algorithm":"AES256-GCM","secret_ref":"vrooli/kopia/offsite:repository-passphrase","created_at":"2026-01-01T00:00:00Z","restore_hints":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/test-offsite", Mountpoint: mount, Filesystem: "ext4", UUID: "offsite-volume-uuid"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, FilesystemState: sysmounts.FilesystemStateClean}}
	eng := &enginemocks.FakeKopiaEngine{}
	svc := destinations.NewService(mocks.NewFakeRepository(), eng, &destinations.FSBundleWriter{}, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), testVolumeLeaseOption())
	if _, err := svc.ImportExistingDestination(ctx, destinations.ImportExistingInput{
		ID: "id", Name: "offsite", Backend: destinations.BackendFilesystem, Location: bundleRoot,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExpectedSnapshotIDs: []string{"snap"},
	}); err == nil {
		t.Fatal("accepted repository with mismatched identity anchor")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("identity mismatch reached resource-kopia: %v", eng.Calls)
	}
}

func TestResolveForUseRejectsBundleSymlinkEscape(t *testing.T) {
	ctx := context.Background()
	mount := t.TempDir()
	outside := t.TempDir()
	relative := "vrooli-backups"
	linkPath := filepath.Join(mount, relative)
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}
	repoPath := destinations.RepositoryPathFor(outside, "removable")
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "kopia.repository.f"), []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/test1", Mountpoint: mount, Filesystem: "ntfs", UUID: "fixture-uuid", Serial: "fixture-serial"}
	d := destinations.Destination{ID: "dest-escape", Name: "removable", BackendKind: destinations.BackendFilesystem, Location: linkPath, RepositoryLocation: repoPath, RelativePath: relative, DeviceIdentity: &identity}
	repo := mocks.NewFakeRepository()
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, FilesystemState: sysmounts.FilesystemStateClean}}
	svc := destinations.NewService(repo, eng, &destinations.FSBundleWriter{}, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), testVolumeLeaseOption())
	if _, err := svc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("resolver accepted a bundle symlink that escapes the mounted volume")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("resolver touched resource-kopia before rejecting the symlink escape: %v", eng.Calls)
	}
}

func TestWithResolvedDestinationRefusesChangedIdentityBeforeHostLeaseOrKopia(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	d := destinations.Destination{
		ID: "dest-identity", Name: "external", BackendKind: destinations.BackendFilesystem,
		Location: "/media/old/vrooli-backups/external", RepositoryLocation: "/media/old/vrooli-backups/external/repositories/external.kopia",
		RelativePath:   "vrooli-backups/external",
		DeviceIdentity: &destinationreadiness.DeviceIdentity{DevicePath: "/dev/expected", Filesystem: "ntfs", UUID: "expected-uuid", Serial: "expected-serial"},
	}
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	changed := destinationreadiness.DeviceIdentity{DevicePath: "/dev/replacement", Mountpoint: "/media/new", Filesystem: "ntfs", UUID: "replacement-uuid", Serial: "replacement-serial"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: changed, Mounted: true}}
	lease := &testVolumeOperationLease{}
	eng := &enginemocks.FakeKopiaEngine{}
	svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	if err := svc.WithResolvedDestination(ctx, d.ID, func(destinations.Destination) error {
		t.Fatal("operation callback ran after stable identity changed")
		return nil
	}); err == nil {
		t.Fatal("resolved destination despite changed volume identity")
	}
	if lease.acquisitions != 0 || len(eng.Calls) != 0 {
		t.Fatalf("changed identity crossed the lease/Kopia boundary: acquisitions=%d calls=%v", lease.acquisitions, eng.Calls)
	}
}

func TestWithResolvedDestinationFailsClosedWithoutHostLeaseProvider(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	d := destinations.Destination{
		ID: "dest-no-lease", Name: "external", BackendKind: destinations.BackendFilesystem,
		Location: "/media/drive/vrooli-backups/external", RepositoryLocation: "/media/drive/vrooli-backups/external/repositories/external.kopia",
		RelativePath:   "vrooli-backups/external",
		DeviceIdentity: &destinationreadiness.DeviceIdentity{DevicePath: "/dev/external", Mountpoint: "/media/drive", Filesystem: "ext4", UUID: "strong-uuid"},
	}
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: *d.DeviceIdentity, Mounted: true, FilesystemState: sysmounts.FilesystemStateClean}}
	eng := &enginemocks.FakeKopiaEngine{}
	svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), t.TempDir(), destinations.WithReadinessGate(probe))
	if err := svc.WithResolvedDestination(ctx, d.ID, func(destinations.Destination) error { return nil }); err == nil {
		t.Fatal("bound destination proceeded without the host volume lease provider")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("engine calls without host lease = %v", eng.Calls)
	}
}

func TestResolveForUseRejectsRepositorySymlinkEscape(t *testing.T) {
	ctx := context.Background()
	mount := t.TempDir()
	bundleRoot := filepath.Join(mount, "vrooli-backups")
	if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	outsideRepo := filepath.Join(t.TempDir(), "outside.kopia")
	if err := os.MkdirAll(outsideRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideRepo, "kopia.repository.f"), []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repoPath := destinations.RepositoryPathFor(bundleRoot, "removable")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideRepo, repoPath); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/test1", Mountpoint: mount, Filesystem: "ntfs", UUID: "fixture-uuid", Serial: "fixture-serial"}
	d := destinations.Destination{ID: "dest-repo-escape", Name: "removable", BackendKind: destinations.BackendFilesystem, Location: bundleRoot, RepositoryLocation: repoPath, RelativePath: "vrooli-backups", DeviceIdentity: &identity}
	writer := &destinations.FSBundleWriter{}
	meta := destinations.BundleMetadata{DestinationID: d.ID, Name: d.Name, Backend: string(d.BackendKind), BundleRoot: bundleRoot, RelativePath: d.RelativePath, RepositoryPath: repoPath, SecretRef: "vrooli/kopia/removable:repository-passphrase", EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256"}
	if err := writer.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	repo := mocks.NewFakeRepository()
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, FilesystemState: sysmounts.FilesystemStateClean}}
	svc := destinations.NewService(repo, eng, writer, filepath.Join(t.TempDir(), "protected"), destinations.WithReadinessGate(probe), testVolumeLeaseOption())
	if _, err := svc.ResolveForUse(ctx, d.ID); err == nil {
		t.Fatal("resolver accepted a repository symlink that escapes the destination bundle")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("resolver touched resource-kopia before rejecting repository symlink escape: %v", eng.Calls)
	}
}

func TestWithResolvedDestinationWaitHonorsContextCancellation(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	d := destinations.Destination{ID: "lock-dest", Name: "external", BackendKind: destinations.BackendS3, Location: "bucket"}
	if _, err := repo.Create(ctx, d); err != nil {
		t.Fatal(err)
	}
	svc := destinations.NewService(repo, &enginemocks.FakeKopiaEngine{}, nil, t.TempDir())
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.WithResolvedDestination(ctx, d.ID, func(destinations.Destination) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	waitCtx, cancel := context.WithCancel(ctx)
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.WithResolvedDestination(waitCtx, d.ID, func(destinations.Destination) error { return nil })
	}()
	cancel()
	select {
	case err := <-secondDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiting operation error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("destination lock wait did not honor context cancellation")
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first operation: %v", err)
	}
	if err := svc.WithResolvedDestination(ctx, d.ID, func(destinations.Destination) error { return nil }); err != nil {
		t.Fatalf("destination lock did not recover after cancellation: %v", err)
	}
}

func TestCreateDestinationRefusesUnknownReadinessEvenWhenAcknowledged(t *testing.T) {
	for _, acknowledged := range []bool{false, true} {
		t.Run(fmt.Sprintf("acknowledged_%v", acknowledged), func(t *testing.T) {
			ctx := context.Background()
			repo := mocks.NewFakeRepository()
			eng := &enginemocks.FakeKopiaEngine{}
			readiness := fixedReadinessReport{report: destinationreadiness.Report{OverallSeverity: destinationreadiness.SeverityUnknown}}
			svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), t.TempDir(), destinations.WithReadinessGate(readiness), testVolumeLeaseOption())
			_, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "external", Backend: destinations.BackendFilesystem, Location: "/mnt/external", AcknowledgeReadinessFailure: acknowledged})
			if err == nil {
				t.Fatal("created a filesystem repository while readiness was unknown")
			}
			if len(eng.Calls) != 0 {
				t.Fatalf("engine touched after unknown readiness: %v", eng.Calls)
			}
			stored, listErr := repo.List(ctx, 10)
			if listErr != nil || len(stored) != 0 {
				t.Fatalf("destination catalog changed after unknown readiness: %v / %v", stored, listErr)
			}
		})
	}
}

func TestCreateDestinationRechecksStableIdentityBeforeWriting(t *testing.T) {
	first := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: "/mnt/external", Filesystem: "ext4", UUID: "uuid-original", Serial: "serial-drive"}
	changed := first
	changed.DevicePath, changed.UUID = "/dev/sdc1", "uuid-replacement"
	gate := &sequenceReadinessReports{reports: []destinationreadiness.Report{
		{OverallSeverity: destinationreadiness.SeverityPass, Identity: first},
		{OverallSeverity: destinationreadiness.SeverityPass, Identity: changed},
	}}
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	bundle := mocks.NewFakeBundleWriter()
	svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(gate), testVolumeLeaseOption())
	_, err := svc.CreateDestination(context.Background(), destinations.CreateInput{Name: "external", Backend: destinations.BackendFilesystem, Location: "/mnt/external/vrooli-backups"})
	if err == nil {
		t.Fatal("created a repository after the mounted device identity changed")
	}
	if len(bundle.Prepared) != 0 || len(eng.Calls) != 0 {
		t.Fatalf("destination wrote before detecting changed identity: prepared=%v engine=%v", bundle.Prepared, eng.Calls)
	}
	stored, listErr := repo.List(context.Background(), 10)
	if listErr != nil || len(stored) != 0 {
		t.Fatalf("catalog changed after identity gate failed: %v / %v", stored, listErr)
	}
}

func TestCreateDestinationRechecksIdentityAfterBundlePreparation(t *testing.T) {
	first := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: "/mnt/external", Filesystem: "ext4", UUID: "uuid-original", Serial: "serial-drive"}
	changed := first
	changed.DevicePath, changed.UUID = "/dev/sdc1", "uuid-replacement"
	gate := &sequenceReadinessReports{reports: []destinationreadiness.Report{
		{OverallSeverity: destinationreadiness.SeverityPass, Identity: first},
		{OverallSeverity: destinationreadiness.SeverityPass, Identity: first},
		{OverallSeverity: destinationreadiness.SeverityPass, Identity: changed},
	}}
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	bundle := mocks.NewFakeBundleWriter()
	svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(gate), testVolumeLeaseOption())
	_, err := svc.CreateDestination(context.Background(), destinations.CreateInput{Name: "external", Backend: destinations.BackendFilesystem, Location: "/mnt/external/vrooli-backups"})
	if err == nil {
		t.Fatal("created a repository after the device changed during bundle preparation")
	}
	if len(bundle.Prepared) != 1 || len(eng.Calls) != 0 {
		t.Fatalf("repository creation crossed the changed-identity gate: prepared=%v engine=%v", bundle.Prepared, eng.Calls)
	}
	stored, listErr := repo.List(context.Background(), 10)
	if listErr != nil || len(stored) != 0 {
		t.Fatalf("catalog changed after identity gate failed: %v / %v", stored, listErr)
	}
}

func TestCreateDestinationReconcilesPendingRepoStatusWithoutRecreating(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: "/mnt/external", Filesystem: "ext4", UUID: "uuid-create-retry", Serial: "serial-create-retry"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	statusCalls := 0
	eng.RepoStatusFn = func(context.Context, string) (engine.RepoStatus, error) {
		statusCalls++
		if statusCalls == 1 {
			return engine.RepoStatus{}, errors.New("injected transient status failure")
		}
		return engine.RepoStatus{Connected: true, EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256"}, nil
	}
	bundle := mocks.NewFakeBundleWriter()
	bundle.ValidateErr = errors.New("no identity-anchored manifest")
	lease := &testVolumeOperationLease{}
	svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	in := destinations.CreateInput{Name: "external", Backend: destinations.BackendFilesystem, Location: "/mnt/external/vrooli-backups"}
	if _, err := svc.CreateDestination(context.Background(), in); err == nil {
		t.Fatal("expected first call to retain a pending create after RepoStatus failure")
	}
	pending, err := repo.GetByName(context.Background(), in.Name)
	if err != nil || pending.ProvisioningState != destinations.ProvisioningCreated || pending.EncryptionAlgorithm != "" {
		t.Fatalf("pending catalog intent = %+v / %v", pending, err)
	}
	if err := svc.WithResolvedDestination(context.Background(), pending.ID, func(destinations.Destination) error { return nil }); err == nil {
		t.Fatal("pending destination was exposed to an operational callback")
	}
	if _, err := svc.DeleteDestination(context.Background(), pending.ID, false); err == nil {
		t.Fatal("pending destination was deleted without reconciling its repository")
	}
	if _, err := svc.CreateDestination(context.Background(), in); err == nil {
		t.Fatal("retry without an identity-anchored manifest must remain fail-closed")
	}
	createCalls, rebindCalls := 0, 0
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			createCalls++
		}
		if strings.HasPrefix(call, "RepoRebind(") {
			rebindCalls++
		}
	}
	if createCalls != 1 || rebindCalls != 0 {
		t.Fatalf("create/rebind call counts = %d/%d, want 1/0; calls=%v", createCalls, rebindCalls, eng.Calls)
	}
}

func TestCreateDestinationReconcilesPendingCatalogAndMetadataFailures(t *testing.T) {
	for _, failAt := range []string{"catalog-update", "bundle-metadata"} {
		t.Run(failAt, func(t *testing.T) {
			mount := t.TempDir()
			identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: mount, Filesystem: "ext4", UUID: "uuid-create-retry-" + failAt, Serial: "serial-create-retry-" + failAt}
			probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
			repo := mocks.NewFakeRepository()
			eng := &enginemocks.FakeKopiaEngine{}
			lease := &testVolumeOperationLease{}
			eng.RepoCreateFn = func(_ context.Context, spec engine.RepoSpec) error {
				intent, intentErr := repo.GetByName(context.Background(), inNameFor(failAt))
				if intentErr != nil || intent.ProvisioningState != destinations.ProvisioningCreating {
					t.Fatalf("RepoCreate started before its durable intent: %+v / %v", intent, intentErr)
				}
				if err := os.MkdirAll(spec.Path, 0o700); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(spec.Path, "kopia.repository.f"), []byte(`{"uniqueID":"create-retry-`+failAt+`"}`), 0o600)
			}
			eng.RepoRebindFn = func(_ context.Context, name, path string) error {
				if name != inNameFor(failAt) || path != destinations.RepositoryPathFor(filepath.Join(mount, "vrooli-backups"), inNameFor(failAt)) {
					t.Fatalf("RepoRebind target = %q/%q", name, path)
				}
				if !lease.active {
					t.Fatal("RepoRebind ran without the volume lease")
				}
				return nil
			}
			baseBundle := &destinations.FSBundleWriter{}
			var bundle destinations.BundleWriter = baseBundle
			var failingBundle *failOnceMetadataBundleWriter
			if failAt == "bundle-metadata" {
				failingBundle = &failOnceMetadataBundleWriter{BundleWriter: baseBundle, fail: true}
				bundle = failingBundle
			}
			svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
			in := destinations.CreateInput{Name: inNameFor(failAt), Backend: destinations.BackendFilesystem, Location: filepath.Join(mount, "vrooli-backups")}
			if failAt == "catalog-update" {
				repo.UpdateErr = errors.New("injected catalog update failure")
			}
			if _, err := svc.CreateDestination(context.Background(), in); err == nil {
				t.Fatal("expected injected post-create failure")
			}
			pending, err := repo.GetByName(context.Background(), in.Name)
			wantState := destinations.ProvisioningCreated
			if failAt == "catalog-update" {
				wantState = destinations.ProvisioningCreating
			}
			if err != nil || pending.ProvisioningState != wantState {
				t.Fatalf("pending row = %+v / %v", pending, err)
			}
			repo.UpdateErr = nil
			if _, err := svc.CreateDestination(context.Background(), in); err == nil {
				t.Fatal("retry without a complete identity anchor must remain fail-closed")
			}
			creates := 0
			for _, call := range eng.Calls {
				if strings.HasPrefix(call, "RepoCreate(") {
					creates++
				}
			}
			if creates != 1 {
				t.Fatalf("RepoCreate calls = %d, want exactly one: %v", creates, eng.Calls)
			}
			if got, err := repo.GetByName(context.Background(), in.Name); err != nil || got.ID != pending.ID || got.ProvisioningState != wantState {
				t.Fatalf("pending row changed during failed reconciliation: %+v / %v", got, err)
			}
		})
	}
}

func inNameFor(s string) string { return "external-" + s }

func TestCreateDestinationPreservesConflictingBundleFileDuringRetry(t *testing.T) {
	mount := t.TempDir()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: mount, Filesystem: "ext4", UUID: "uuid-create-conflict", Serial: "serial-create-conflict"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	name := "external-conflict"
	root := filepath.Join(mount, "vrooli-backups")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(root, destinations.BundleReadmeFile)
	original := []byte("operator-owned notes")
	if err := os.WriteFile(readmePath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	eng.RepoCreateFn = func(_ context.Context, spec engine.RepoSpec) error {
		if err := os.MkdirAll(spec.Path, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(spec.Path, "kopia.repository.f"), []byte(`{"uniqueID":"create-conflict-repo"}`), 0o600)
	}
	lease := &testVolumeOperationLease{}
	svc := destinations.NewService(repo, eng, &destinations.FSBundleWriter{}, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	in := destinations.CreateInput{Name: name, Backend: destinations.BackendFilesystem, Location: root}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := svc.CreateDestination(context.Background(), in); err == nil {
			t.Fatal("expected conflicting bundle metadata to remain pending")
		}
	}
	if got, err := os.ReadFile(readmePath); err != nil || string(got) != string(original) {
		t.Fatalf("conflicting operator file was modified: %q / %v", got, err)
	}
	row, err := repo.GetByName(context.Background(), name)
	if err != nil || row.ProvisioningState != destinations.ProvisioningCreated {
		t.Fatalf("conflicted create state = %+v / %v", row, err)
	}
	creates, rebinds := 0, 0
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			creates++
		}
		if strings.HasPrefix(call, "RepoRebind(") {
			rebinds++
		}
	}
	if creates != 1 || rebinds != 0 {
		t.Fatalf("conflict retry create/rebind calls = %d/%d, want 1/0: %v", creates, rebinds, eng.Calls)
	}
}

func TestCreateDestinationDoesNotRetryAmbiguousRepoCreate(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: "/mnt/external", Filesystem: "ext4", UUID: "uuid-create-ambiguous", Serial: "serial-create-ambiguous"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	repo := mocks.NewFakeRepository()
	eng := &enginemocks.FakeKopiaEngine{}
	eng.RepoCreateFn = func(context.Context, engine.RepoSpec) error {
		return errors.New("outcome unknown after runner invocation")
	}
	eng.RepoRebindFn = func(context.Context, string, string) error {
		return errors.New("no exact repository registration to connect")
	}
	bundle := mocks.NewFakeBundleWriter()
	lease := &testVolumeOperationLease{}
	svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	in := destinations.CreateInput{Name: "external-ambiguous", Backend: destinations.BackendFilesystem, Location: "/mnt/external/vrooli-backups"}
	if _, err := svc.CreateDestination(context.Background(), in); err == nil {
		t.Fatal("expected ambiguous RepoCreate failure")
	}
	if _, err := svc.CreateDestination(context.Background(), in); err == nil {
		t.Fatal("expected failed connection to preserve the pending row")
	}
	creates, rebinds := 0, 0
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			creates++
		}
		if strings.HasPrefix(call, "RepoRebind(") {
			rebinds++
		}
	}
	if creates != 1 || rebinds != 0 {
		t.Fatalf("ambiguous retry create/rebind calls = %d/%d, want 1/0: %v", creates, rebinds, eng.Calls)
	}
	if deleted := strings.Contains(strings.Join(eng.Calls, " "), "RepoDelete("); deleted {
		t.Fatalf("ambiguous create triggered cleanup: %v", eng.Calls)
	}
}

func TestCreateDestinationResumesCreatedAttemptAtCurrentMountpoint(t *testing.T) {
	ctx := context.Background()
	oldMount := filepath.Join(t.TempDir(), "old")
	newMount := filepath.Join(t.TempDir(), "new")
	if err := os.MkdirAll(newMount, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "relocated-pending"
	relative := filepath.Join("vrooli-backups", "bundle")
	root := filepath.Join(newMount, relative)
	repositoryPath := destinations.RepositoryPathFor(root, name)
	if err := os.MkdirAll(repositoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryPath, "kopia.repository.f"), []byte(`{"uniqueID":"relocated-pending-repo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: newMount, Filesystem: "ext4", UUID: "uuid-relocated-pending", Serial: "serial-relocated-pending"}
	probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	repo := mocks.NewFakeRepository()
	pending := destinations.Destination{
		ID: "relocated-pending-id", Name: name, BackendKind: destinations.BackendFilesystem,
		Location: filepath.Join(oldMount, relative), RepositoryLocation: destinations.RepositoryPathFor(filepath.Join(oldMount, relative), name),
		CapPolicy: destinations.CapPolicyAlertBlock, ProvisioningState: destinations.ProvisioningCreated,
		SecretRef: "vrooli/kopia/relocated-pending:repository-passphrase", EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		RelativePath: relative, DeviceIdentity: &destinationreadiness.DeviceIdentity{DevicePath: "/dev/old", Mountpoint: oldMount, Filesystem: identity.Filesystem, UUID: identity.UUID, Serial: identity.Serial},
	}
	var err error
	if pending, err = repo.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	host, _ := os.Hostname()
	meta := destinations.BundleMetadata{DestinationID: pending.ID, Name: name, Backend: string(destinations.BackendFilesystem), BundleRoot: root, RelativePath: relative, RepositoryPath: repositoryPath, EncryptionAlgorithm: pending.EncryptionAlgorithm, SecretRef: pending.SecretRef, CreatedAt: pending.CreatedAt, Host: host, User: os.Getenv("USER")}
	bundle := &destinations.FSBundleWriter{}
	if err := bundle.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	eng := &enginemocks.FakeKopiaEngine{}
	lease := &testVolumeOperationLease{}
	svc := destinations.NewService(repo, eng, bundle, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	input := destinations.CreateInput{Name: name, Backend: destinations.BackendFilesystem, Location: filepath.Join(newMount, relative)}
	manifestPath := filepath.Join(root, destinations.BundleManifestFile)
	anchoredManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(anchoredManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	delete(manifest, "repository_unique_id")
	withoutAnchor, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, withoutAnchor, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateDestination(ctx, input); err == nil {
		t.Fatal("retry accepted a manifest without a repository identity anchor")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("missing identity anchor mutated resource-kopia: %v", eng.Calls)
	}
	if err := os.WriteFile(manifestPath, anchoredManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(root, destinations.BundleReadmeFile)
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	linkedReadme := filepath.Join(t.TempDir(), "readme.txt")
	if err := os.WriteFile(linkedReadme, readmeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(readmePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(linkedReadme, readmePath); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateDestination(ctx, input); err == nil {
		t.Fatal("retry accepted a symlink in generated bundle metadata")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("symlink metadata mutated resource-kopia: %v", eng.Calls)
	}
	if err := os.Remove(readmePath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, readmeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(repositoryPath, "kopia.repository.f")
	if err := os.WriteFile(marker, []byte(`{"uniqueID":"replacement-repo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateDestination(ctx, input); err == nil {
		t.Fatal("retry accepted a repository whose unique ID differs from its manifest anchor")
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("identity mismatch mutated resource-kopia before rejection: %v", eng.Calls)
	}
	if err := os.WriteFile(marker, []byte(`{"uniqueID":"relocated-pending-repo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ready, err := svc.CreateDestination(ctx, input)
	if err != nil {
		t.Fatalf("resume at current mountpoint: %v", err)
	}
	if ready.ID != pending.ID || ready.Location != filepath.Join(newMount, relative) || ready.ProvisioningState != destinations.ProvisioningReady {
		t.Fatalf("resumed destination = %+v", ready)
	}
	creates, rebinds := 0, 0
	for _, call := range eng.Calls {
		if strings.HasPrefix(call, "RepoCreate(") {
			creates++
		}
		if strings.HasPrefix(call, "RepoRebind(") {
			rebinds++
		}
	}
	if creates != 0 || rebinds != 1 {
		t.Fatalf("retry create/rebind calls = %d/%d, want 0/1: %v", creates, rebinds, eng.Calls)
	}
}

func TestCreateDestinationRejectsStrongIdentityLossAfterLease(t *testing.T) {
	ctx := context.Background()
	mount := filepath.Join(t.TempDir(), "external")
	if err := os.MkdirAll(mount, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "identity-downgrade"
	relative := filepath.Join("vrooli-backups", "bundle")
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: mount, Filesystem: "ext4", UUID: "uuid-strong-lease", Serial: "serial-strong-lease", TotalBytes: 5000}
	probe := &mutableDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
	repo := mocks.NewFakeRepository()
	pending := destinations.Destination{
		ID: "identity-downgrade-id", Name: name, BackendKind: destinations.BackendFilesystem,
		Location: filepath.Join(mount, relative), RepositoryLocation: destinations.RepositoryPathFor(filepath.Join(mount, relative), name),
		CapPolicy: destinations.CapPolicyAlertBlock, ProvisioningState: destinations.ProvisioningCreated,
		SecretRef: "vrooli/kopia/identity-downgrade:repository-passphrase", EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		RelativePath: relative, DeviceIdentity: &identity,
	}
	if _, err := repo.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	lease := &testVolumeOperationLease{}
	lease.onAcquire = func() {
		probe.inspection.Identity.UUID = ""
		probe.inspection.Identity.Serial = ""
	}
	eng := &enginemocks.FakeKopiaEngine{}
	svc := destinations.NewService(repo, eng, &destinations.FSBundleWriter{}, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(lease))
	input := destinations.CreateInput{Name: name, Backend: destinations.BackendFilesystem, Location: filepath.Join(mount, relative)}
	if _, err := svc.CreateDestination(ctx, input); err == nil {
		t.Fatal("pending create accepted a weak device observation after acquiring the lease")
	}
	if len(eng.Calls) != 0 || lease.active || lease.acquisitions != 1 {
		t.Fatalf("post-lease identity loss crossed the rebind boundary: calls=%v lease=%+v", eng.Calls, lease)
	}
}

func TestCreateDestinationRejectsPendingPathContainmentEscapes(t *testing.T) {
	for _, escape := range []string{"bundle", "repository"} {
		t.Run(escape, func(t *testing.T) {
			ctx := context.Background()
			mount := filepath.Join(t.TempDir(), "external")
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.MkdirAll(mount, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o700); err != nil {
				t.Fatal(err)
			}
			name := "path-escape-" + escape
			relative := filepath.Join("vrooli-backups", "bundle")
			bundleRoot := filepath.Join(mount, relative)
			if escape == "bundle" {
				if err := os.Symlink(outside, filepath.Join(mount, "vrooli-backups")); err != nil {
					t.Fatal(err)
				}
			} else {
				repositoryParent := filepath.Dir(destinations.RepositoryPathFor(bundleRoot, name))
				if err := os.MkdirAll(repositoryParent, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, destinations.RepositoryPathFor(bundleRoot, name)); err != nil {
					t.Fatal(err)
				}
			}
			identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: mount, Filesystem: "ext4", UUID: "uuid-path-escape-" + escape, Serial: "serial-path-escape-" + escape}
			probe := fixedDestinationProbe{inspection: destinationreadiness.Inspection{Identity: identity, Mounted: true, LocationExists: true, LocationIsDirectory: true, FilesystemState: sysmounts.FilesystemStateClean}}
			repo := mocks.NewFakeRepository()
			pending := destinations.Destination{
				ID: "path-escape-id-" + escape, Name: name, BackendKind: destinations.BackendFilesystem,
				Location: bundleRoot, RepositoryLocation: destinations.RepositoryPathFor(bundleRoot, name),
				CapPolicy: destinations.CapPolicyAlertBlock, ProvisioningState: destinations.ProvisioningCreated,
				SecretRef: "vrooli/kopia/" + name + ":repository-passphrase", EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
				RelativePath: relative, DeviceIdentity: &identity,
			}
			if _, err := repo.Create(ctx, pending); err != nil {
				t.Fatal(err)
			}
			eng := &enginemocks.FakeKopiaEngine{}
			svc := destinations.NewService(repo, eng, &destinations.FSBundleWriter{}, t.TempDir(), destinations.WithReadinessGate(probe), destinations.WithVolumeOperationLease(&testVolumeOperationLease{}))
			input := destinations.CreateInput{Name: name, Backend: destinations.BackendFilesystem, Location: bundleRoot}
			if _, err := svc.CreateDestination(ctx, input); err == nil {
				t.Fatalf("pending %s escape was accepted", escape)
			}
			if len(eng.Calls) != 0 {
				t.Fatalf("pending %s escape reached resource-kopia: %v", escape, eng.Calls)
			}
		})
	}
}

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

	t.Run("SFTP destination delegates key resolution to resource-kopia", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		svc := destinations.NewService(mocks.NewFakeRepository(), eng, mocks.NewFakeBundleWriter(), protected)
		location := "sftp://backup@backup.example:2222/vrooli/daily?known_hosts=%2Fhome%2Foperator%2F.ssh%2Fknown_hosts"
		got, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "sftp-offsite", Backend: destinations.BackendSFTP, Location: location})
		if err != nil {
			t.Fatalf("CreateDestination SFTP: %v", err)
		}
		if got.BackendKind != destinations.BackendSFTP || got.Location != location || got.RepositoryLocation != location {
			t.Fatalf("destination = %+v", got)
		}
		if len(eng.Repos) != 1 {
			t.Fatalf("RepoCreate calls = %d, want 1", len(eng.Repos))
		}
		want := engine.RepoSpec{Name: "sftp-offsite", Backend: engine.BackendSFTP, SFTPHost: "backup.example", SFTPPort: 2222, SFTPUsername: "backup", SFTPRemotePath: "/vrooli/daily", SFTPKnownHosts: "/home/operator/.ssh/known_hosts"}
		if eng.Repos[0] != want {
			t.Fatalf("RepoSpec = %+v, want %+v", eng.Repos[0], want)
		}
	})

	t.Run("S3-compatible provider URI reaches Kopia without credentials", func(t *testing.T) {
		eng := &enginemocks.FakeKopiaEngine{}
		svc := destinations.NewService(mocks.NewFakeRepository(), eng, mocks.NewFakeBundleWriter(), protected)
		location := "s3+https://objects.example:9443/vrooli-backups/host-a?region=eu-west-1"
		if _, err := svc.CreateDestination(ctx, destinations.CreateInput{Name: "provider-s3", Backend: destinations.BackendS3, Location: location}); err != nil {
			t.Fatalf("CreateDestination provider S3: %v", err)
		}
		if len(eng.Repos) != 1 {
			t.Fatalf("RepoCreate calls = %d, want 1", len(eng.Repos))
		}
		got := eng.Repos[0]
		if got.Bucket != "vrooli-backups" || got.Endpoint != "objects.example:9443" || got.Prefix != "host-a" || got.Region != "eu-west-1" || got.DisableTLS {
			t.Fatalf("RepoSpec = %+v, provider URI was not mapped correctly", got)
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
