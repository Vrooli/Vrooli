package physicalruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
	"testing/fstest"
)

func TestEmbeddedReviewedSmartDisplayInstallsIntoCleanOwnerAndReplaysIdempotently(t *testing.T) {
	ctx := database.WithTestMode(context.Background())
	owners := newReviewedStarterInstallTestOwners(t, ctx)
	qualification, err := NewQualificationService(owners, qualificationRenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	owners.Qualification = qualification

	preset, err := owners.InitializePhysicalStarter(ctx, reviewedSmartDisplayV7StarterID)
	if err != nil {
		t.Fatalf("install embedded reviewed starter: %v", err)
	}
	if preset.PresetID != "generic-smart-display-tabletop-still-v7" || preset.Revision != 2 {
		t.Fatalf("installer did not return the refreshed qualified preset: %+v", preset)
	}
	state, err := owners.InitializeReviewedSmartDisplayV7Candidate(ctx)
	if err != nil || state.Device.State != devices.DeviceRevisionQualified || state.Device.Revision != 2 || state.Preset.Revision != 2 {
		t.Fatalf("clean destination did not receive the exact scoped reviewed child: device=%+v preset=%+v err=%v", state.Device, state.Preset, err)
	}
	if len(state.Device.Candidate.Definition.QualificationReviews) != 1 {
		t.Fatalf("install appended %d reviews, want exactly the retained root review", len(state.Device.Candidate.Definition.QualificationReviews))
	}
	review := state.Device.Candidate.Definition.QualificationReviews[0]
	if review.Scope != "visual_setup" || review.Result != "passed" || review.Reviewer != "root" || review.NativeRenderJobID != "e83ef49d-62ee-46aa-b65c-cb6d470e2782" || len(review.DimensionStates) != 1 || review.DimensionStates[0] != "source_geometry_envelope" || len(review.AllowedMotionIDs) != 1 || review.AllowedMotionIDs[0] != "static" {
		t.Fatalf("installer changed or broadened the retained review scope: %+v", review)
	}

	secondPreset, err := owners.InitializePhysicalStarter(ctx, reviewedSmartDisplayV7StarterID)
	if err != nil || secondPreset.Revision != preset.Revision || secondPreset.SHA256 != preset.SHA256 {
		t.Fatalf("explicit repeat was not idempotent: preset=%+v err=%v", secondPreset, err)
	}
	head, err := owners.Devices.GetHead(ctx, "generic-smart-display-candidate-v7")
	if err != nil || head.Revision != 2 || len(head.Candidate.Definition.QualificationReviews) != 1 {
		t.Fatalf("repeat install appended or changed review history: head=%+v err=%v", head, err)
	}
}

func TestEmbeddedReviewedSmartDisplayPackageFailsClosedOnMissingOrChangedInputs(t *testing.T) {
	certificateBytes, err := starterAssets.ReadFile(smartDisplayV7ReviewCertificate)
	if err != nil {
		t.Fatal(err)
	}
	var certificate reviewedStarterCertificate
	if err := json.Unmarshal(certificateBytes, &certificate); err != nil {
		t.Fatal(err)
	}
	makeMapFS := func(change func(map[string][]byte)) fstest.MapFS {
		files := map[string][]byte{smartDisplayV7ReviewCertificate: certificateBytes}
		output, err := starterAssets.ReadFile(smartDisplayV7ReviewOutput)
		if err != nil {
			t.Fatal(err)
		}
		files[smartDisplayV7ReviewOutput] = output
		for _, artifact := range certificate.SourceArtifacts {
			path, ok := reviewedSmartDisplayV7SourcePath(artifact.Ref)
			if !ok {
				t.Fatalf("unknown embedded source reference %s", artifact.Ref.ArtifactID)
			}
			data, err := starterAssets.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files[path] = data
		}
		change(files)
		mapped := make(fstest.MapFS, len(files))
		for name, data := range files {
			mapped[name] = &fstest.MapFile{Data: data}
		}
		return mapped
	}
	if _, err := loadEmbeddedSmartDisplayV7Review(makeMapFS(func(files map[string][]byte) { delete(files, smartDisplayV7ReviewOutput) })); err == nil {
		t.Fatal("package loader accepted a missing retained native output")
	}
	if _, err := loadEmbeddedSmartDisplayV7Review(makeMapFS(func(files map[string][]byte) { delete(files, "assets/phone-table/textures/travertine-color.jpg") })); err == nil {
		t.Fatal("package loader accepted a missing source texture")
	}
	if _, err := loadEmbeddedSmartDisplayV7Review(makeMapFS(func(files map[string][]byte) {
		path := "assets/smart-display-v7/generic-smart-display-v7.glb"
		files[path] = append([]byte(nil), files[path]...)
		files[path][0] ^= 1
	})); err == nil {
		t.Fatal("package loader accepted a changed source model")
	}
	if _, err := loadEmbeddedSmartDisplayV7Review(makeMapFS(func(files map[string][]byte) {
		files[smartDisplayV7ReviewCertificate] = []byte(`{"version":2,"package_sha256":"stale"}`)
	})); err == nil {
		t.Fatal("package loader accepted a stale or malformed certificate")
	}
}

func TestEmbeddedReviewedStarterRegistryInstallsAndReplaysExactScopedPackages(t *testing.T) {
	ctx := database.WithTestMode(context.Background())

	ids := []string{
		"generic-round-watch-body-v4-floating-reviewed",
		"generic-television-concept-v3-reviewed",
		"generic-digital-frame-concept-v4-glassless-reviewed",
		"generic-digital-frame-wall-v5-mounted-reviewed",
		"generic-book-fold-v3-left-screen-vflip-static-reviewed",
		"laptop-screen-vflip-reviewed",
		"phone-study-hero-reviewed",
		"ipad-portrait-cupfree-reviewed",
		"monitor-study-matching-reviewed",
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			owners := newReviewedStarterInstallTestOwners(t, ctx)
			qualification, err := NewQualificationService(owners, qualificationRenderStub{})
			if err != nil {
				t.Fatal(err)
			}
			owners.Qualification = qualification
			descriptor := embeddedReviewedStarters[id]
			_, certificate, err := loadEmbeddedReviewedStarter(starterAssets, descriptor)
			if err != nil {
				t.Fatalf("load pinned package: %v", err)
			}
			preset, err := owners.InitializePhysicalStarter(ctx, id)
			if err != nil {
				t.Fatalf("install package: %v", err)
			}
			if preset.PresetID != certificate.Preset.PresetID || preset.Revision != 2 {
				t.Fatalf("install did not return refreshed reviewed preset: got %+v want %s@2", preset, certificate.Preset.PresetID)
			}
			qualified, err := owners.Devices.GetRevision(ctx, descriptor.DeviceID, 2)
			if err != nil || qualified.State != devices.DeviceRevisionQualified || !sameJSON(qualified.Candidate.Definition.QualificationReviews[0], certificate.Review) {
				t.Fatalf("destination did not retain the exact root scope: revision=%+v err=%v", qualified, err)
			}
			if len(qualified.Candidate.Definition.QualificationReviews) != 1 || qualified.Candidate.Definition.Qualification != "qualified" {
				t.Fatalf("installer broadened or duplicated review state: %+v", qualified.Candidate.Definition.QualificationReviews)
			}
			replay, err := owners.InitializePhysicalStarter(ctx, id)
			if err != nil || replay.PresetID != preset.PresetID || replay.Revision != preset.Revision || replay.SHA256 != preset.SHA256 {
				t.Fatalf("reviewed starter replay changed preset: %+v err=%v", replay, err)
			}
			head, err := owners.Devices.GetHead(ctx, descriptor.DeviceID)
			if err != nil || head.Revision != 2 || len(head.Candidate.Definition.QualificationReviews) != 1 {
				t.Fatalf("replay changed review history: head=%+v err=%v", head, err)
			}
		})
	}
}

func TestQualifiedPhoneReviewDoesNotBlockTabletOrMonitorFactoryInitializers(t *testing.T) {
	ctx := database.WithTestMode(context.Background())
	owners := newReviewedStarterInstallTestOwners(t, ctx)
	qualification, err := NewQualificationService(owners, qualificationRenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	owners.Qualification = qualification
	if _, err := owners.InitializePhysicalStarter(ctx, "phone-study-hero-reviewed"); err != nil {
		t.Fatalf("install scoped phone review: %v", err)
	}
	for _, id := range []string{"ipad-portrait-cupfree-reviewed", "monitor-study-matching-reviewed"} {
		if _, err := owners.InitializePhysicalStarter(ctx, id); err != nil {
			t.Fatalf("initialize %s with the exact reviewed phone child present: %v", id, err)
		}
	}
	phone, err := owners.Devices.GetHead(ctx, "authored-phone-neutral")
	if err != nil || phone.Revision != 2 || phone.State != devices.DeviceRevisionQualified || len(phone.Candidate.Definition.QualificationReviews) != 1 {
		t.Fatalf("dependent initializer changed or broadened phone review: %+v err=%v", phone, err)
	}
	changed := phone.Candidate
	changed.Definition.Qualification = "unqualified"
	changed.Definition.QualificationReviews = nil
	changed.Definition.Description += " changed after scoped review"
	if _, err := owners.Devices.SaveCandidate(ctx, changed, phone.Revision); err != nil {
		t.Fatalf("append changed phone head for negative replay case: %v", err)
	}
	if _, err := owners.InitializePhysicalStarter(ctx, "ipad-screen-uv-oriented-180"); !errors.Is(err, devices.ErrRevisionConflict) {
		t.Fatalf("tablet initializer accepted a changed phone head after the scoped review: %v", err)
	}
}

func TestStartupReplaysQualifiedMonitorFactoryOnlyFromExactScopedChild(t *testing.T) {
	ctx := database.WithTestMode(context.Background())
	fixture := newReviewedStarterInstallFixture(t, ctx)
	owners := fixture.owners
	qualification, err := NewQualificationService(owners, qualificationRenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	owners.Qualification = qualification
	for _, id := range []string{"phone-study-hero-reviewed", "monitor-study-matching-reviewed"} {
		if _, err := owners.InitializePhysicalStarter(ctx, id); err != nil {
			t.Fatalf("install retained %s review: %v", id, err)
		}
	}

	// Reconstruct all owner services over the same persisted database and file
	// routes to exercise the normal startup initializer after app restart.
	reopened, err := New(fixture.db, fixture.roots, fixture.media)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.InitializeReviewedStarter(ctx); err != nil {
		t.Fatalf("startup should replay fixed factory candidates beneath exact phone and monitor reviews: %v", err)
	}
	monitor, err := reopened.Devices.GetHead(ctx, monitorV2DeviceID)
	if err != nil || monitor.Revision != 2 || monitor.State != devices.DeviceRevisionQualified || len(monitor.Candidate.Definition.QualificationReviews) != 1 {
		t.Fatalf("startup changed the exact reviewed monitor child: head=%+v err=%v", monitor, err)
	}

	changed := monitor.Candidate
	changed.Definition.Qualification = "unqualified"
	changed.Definition.QualificationReviews = nil
	changed.Definition.Description += " unexpected extra head"
	if _, err := reopened.Devices.SaveCandidate(ctx, changed, monitor.Revision); err != nil {
		t.Fatalf("append malformed monitor head for fail-closed case: %v", err)
	}
	changedHead, err := reopened.Devices.GetHead(ctx, monitorV2DeviceID)
	if err != nil {
		t.Fatal(err)
	}
	changedRevision, changedSHA := changedHead.Revision, changedHead.SHA256
	if _, err := reopened.InitializeReviewedStarter(ctx); !errors.Is(err, devices.ErrRevisionConflict) {
		t.Fatalf("startup accepted an extra changed monitor head: %v", err)
	}
	availability, err := reopened.BootstrapStartersForStartup(ctx)
	if err != nil {
		t.Fatalf("startup bootstrap should preserve app availability after a user-edited optional starter: %v", err)
	}
	contains := func(want string) bool {
		for _, item := range availability.Available {
			if item == want {
				return true
			}
		}
		return false
	}
	if !contains("phone-default") || !contains("laptop-screen-vflip") || !contains("monitor-default") || contains("monitor-screen-mapped-v2") || len(availability.Issues) != 1 || availability.Issues[0].StarterID != "monitor-screen-mapped-v2" || !errors.Is(availability.Issues[0].Err, devices.ErrRevisionConflict) {
		t.Fatalf("startup availability did not truthfully degrade only the edited monitor starter: %+v", availability)
	}
	unchanged, err := reopened.Devices.GetHead(ctx, monitorV2DeviceID)
	if err != nil || unchanged.Revision != changedRevision || unchanged.SHA256 != changedSHA {
		t.Fatalf("startup bootstrap adopted or overwrote the user-edited monitor head: head=%+v err=%v", unchanged, err)
	}
}

func TestEmbeddedReviewedStarterRegistryFailsClosedOnTamperedOutputAndSource(t *testing.T) {
	for id, descriptor := range embeddedReviewedStarters {
		t.Run(id, func(t *testing.T) {
			certificateBytes, err := starterAssets.ReadFile(descriptor.Certificate)
			if err != nil {
				t.Fatal(err)
			}
			var certificate reviewedStarterCertificate
			if err := json.Unmarshal(certificateBytes, &certificate); err != nil {
				t.Fatal(err)
			}
			outputPath := descriptor.Certificate[:len(descriptor.Certificate)-len("qualification-certificate-v1.json")] + "native-output.png"
			output, err := starterAssets.ReadFile(outputPath)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{descriptor.Certificate: certificateBytes, outputPath: output}
			sourcePath := ""
			for i, artifact := range certificate.SourceArtifacts {
				path, ok := reviewedStarterSourcePath(artifact.Ref)
				if !ok {
					t.Fatalf("certificate source %s is not allowlisted", artifact.Ref.ArtifactID)
				}
				if i == 0 {
					sourcePath = path
				}
				data, err := starterAssets.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				files[path] = data
			}
			makeFS := func(mutate func(map[string][]byte)) fstest.MapFS {
				copyFiles := make(map[string][]byte, len(files))
				for path, data := range files {
					copyFiles[path] = append([]byte(nil), data...)
				}
				mutate(copyFiles)
				mapped := make(fstest.MapFS, len(copyFiles))
				for path, data := range copyFiles {
					mapped[path] = &fstest.MapFile{Data: data}
				}
				return mapped
			}
			if _, _, err := loadEmbeddedReviewedStarter(makeFS(func(f map[string][]byte) { delete(f, outputPath) }), descriptor); err == nil {
				t.Fatal("loader accepted a missing native output")
			}
			if _, _, err := loadEmbeddedReviewedStarter(makeFS(func(f map[string][]byte) { delete(f, sourcePath) }), descriptor); err == nil {
				t.Fatal("loader accepted a missing source artifact")
			}
			if _, _, err := loadEmbeddedReviewedStarter(makeFS(func(f map[string][]byte) { f[sourcePath][0] ^= 1 }), descriptor); err == nil {
				t.Fatal("loader accepted changed source bytes")
			}
		})
	}
}

func newReviewedStarterInstallTestOwners(t *testing.T, ctx context.Context) Owners {
	return newReviewedStarterInstallFixture(t, ctx).owners
}

type reviewedStarterInstallFixture struct {
	owners Owners
	db     *database.RoutedDB
	roots  *filerouting.RoutedRoots
	media  projects.SourceMediaService
}

func newReviewedStarterInstallFixture(t *testing.T, ctx context.Context) reviewedStarterInstallFixture {
	t.Helper()
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema),
		database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(initCtx, pool, providers...)
	})
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	const lease = "reviewed-starter-install-test"
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	projectRepo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(projectRepo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	mediaService := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: projectRepo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, mediaService)
	if err != nil {
		t.Fatal(err)
	}
	return reviewedStarterInstallFixture{owners: owners, db: db, roots: roots, media: mediaService}
}
