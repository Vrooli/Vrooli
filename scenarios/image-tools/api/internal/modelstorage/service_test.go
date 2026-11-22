package modelstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"

	"image-tools/internal/fetch"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

var testNow = time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

// fakeDisk is a filesystem whose free space grows when weights are removed
// and shrinks when they are installed.
type fakeDisk struct {
	mu   sync.Mutex
	free int64
}

func (d *fakeDisk) avail(string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.free, nil
}

func (d *fakeDisk) add(n int64) {
	d.mu.Lock()
	d.free += n
	d.mu.Unlock()
}

type fakeInstaller struct {
	mu        sync.Mutex
	disk      *fakeDisk
	records   map[string]models.InstallRecord
	sizes     map[string]int64
	removeErr map[string]error
	removed   []string
	installs  atomic.Int32
	installed chan struct{}
	block     chan struct{}
	installFn func(id string) error
}

func (f *fakeInstaller) Installed(_ context.Context, id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.records[id].Installed
}

func (f *fakeInstaller) InstallForJob(_ context.Context, id, jobID string, emit func(int, string)) (models.InstallRecord, error) {
	f.installs.Add(1)
	if f.block != nil {
		<-f.block
	}
	if f.installFn != nil {
		if err := f.installFn(id); err != nil {
			return models.InstallRecord{}, err
		}
	}
	if !strings.Contains(jobID, ownerJobKeyPrefix) {
		return models.InstallRecord{}, fmt.Errorf("install job key %q is not attributed to auto-acquire", jobID)
	}
	emit(50, "downloading")
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := models.InstallRecord{ID: id, Installed: true, Path: "/models/" + id, InstalledAt: testNow}
	f.records[id] = rec
	f.disk.add(-f.sizes[id])
	return rec, nil
}

func (f *fakeInstaller) Remove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.removeErr[id]; err != nil {
		return err
	}
	delete(f.records, id)
	f.removed = append(f.removed, id)
	f.disk.add(f.sizes[id])
	return nil
}

func (f *fakeInstaller) list(context.Context) ([]models.InstallRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]models.InstallRecord, 0, len(f.records))
	for _, r := range f.records {
		out = append(out, r)
	}
	return out, nil
}

type harness struct {
	svc       *Service
	installer *fakeInstaller
	disk      *fakeDisk
	store     *Store
	active    map[string]bool
	inUse     map[string]bool
	enabled   map[string]bool
	seed      map[string]models.Model
	custom    []models.Model
	use       map[string]time.Time
	// runtime holds a not-ready lifecycle reason per installed model; absent
	// means runtime-ready.
	runtime map[string]string
}

func governed(id string, sizeMB int) models.Model {
	return models.Model{
		ID: id, Backend: "onnxruntime", SizeMBApprox: sizeMB, Enabled: true,
		CapabilityLabels: models.CapabilityLabels{CommercialUse: models.CommercialUseYes},
		Source:           fetch.Source{Assets: []fetch.Asset{{Filename: id + ".onnx", URL: "https://example.invalid/" + id}}},
	}
}

func newHarness(t *testing.T, freeHeadroom int64, cfg Config) *harness {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), sqlDB, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if cfg.Policy == (Policy{}) {
		cfg.Policy = testPolicy()
	}
	h := &harness{
		disk:    &fakeDisk{free: cfg.Policy.FloorBytes + freeHeadroom},
		store:   NewStore(sqlDB),
		active:  map[string]bool{},
		inUse:   map[string]bool{},
		enabled: map[string]bool{},
		seed:    map[string]models.Model{},
		use:     map[string]time.Time{},
	}
	h.installer = &fakeInstaller{disk: h.disk, records: map[string]models.InstallRecord{}, sizes: map[string]int64{}, removeErr: map[string]error{}}
	svc, err := New(Deps{
		Root:      "/models",
		Installer: h.installer,
		Seed: func(id string) (models.Model, bool) {
			m, ok := h.seed[id]
			return m, ok
		},
		Custom:            func(context.Context) ([]models.Model, error) { return h.custom, nil },
		InstallRecords:    h.installer.list,
		LastSuccessfulUse: func(context.Context) (map[string]time.Time, error) { return h.use, nil },
		ActiveModelIDs:    func(context.Context) (map[string]bool, error) { return h.active, nil },
		InUse:             func(id string) bool { return h.inUse[id] },
		Enabled: func(_ context.Context, m models.Model) (bool, error) {
			if v, ok := h.enabled[m.ID]; ok {
				return v, nil
			}
			return m.Enabled, nil
		},
		RuntimeReady: func(_ context.Context, id string) (bool, string) {
			reason, ok := h.runtime[id]
			if !ok {
				return true, "ready"
			}
			return false, reason
		},
		BackendReady: func(_ context.Context, m models.Model) error {
			if m.Backend == "iopaint" {
				return errors.New("backend iopaint cannot currently serve object_removal")
			}
			return nil
		},
		DiskAvail: h.disk.avail,
		Store:     h.store,
		Measure: func(path string) (int64, error) {
			return h.installer.sizes[filepath.Base(path)], nil
		},
		AcquireNeedBytes: func(_ context.Context, m models.Model) (int64, error) {
			return int64(m.SizeMBApprox) << 20, nil
		},
		Now:    func() time.Time { return testNow },
		Logger: log.New(io.Discard, "", 0),
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	h.svc = svc
	return h
}

// install registers installed weights of the given size, installed `age` ago.
func (h *harness) install(m models.Model, bytes int64, age time.Duration) {
	h.seed[m.ID] = m
	h.installer.sizes[m.ID] = bytes
	h.installer.records[m.ID] = models.InstallRecord{ID: m.ID, Installed: true, Path: "/models/" + m.ID, InstalledAt: testNow.Add(-age), Checksum: "sha-" + m.ID}
}

func findCandidate(t *testing.T, r Report, id string) Candidate {
	t.Helper()
	for _, c := range r.Candidates {
		if c.ModelID == id {
			return c
		}
	}
	t.Fatalf("candidate %q missing from %v", id, r.Candidates)
	return Candidate{}
}

func hasProtection(c Candidate, reason string) bool {
	for _, p := range c.Protected {
		if p == reason {
			return true
		}
	}
	return false
}

func TestPlanAppliesEveryProtection(t *testing.T) {
	h := newHarness(t, 5*gib, Config{AutoOffload: true, AutoAcquire: true})
	old := 90 * 24 * time.Hour
	h.install(governed("eligible", 100), 2*gib, old)
	h.install(governed("kept", 100), 2*gib, old)
	h.install(governed("running", 100), 2*gib, old)
	h.install(governed("leased", 100), 2*gib, old)
	h.install(governed("fresh", 100), 2*gib, 10*time.Minute)
	manual := governed("manual", 100)
	manual.Source = fetch.Source{Manual: true, DocsURL: "https://example.invalid/docs"}
	h.install(manual, 2*gib, old)
	built := governed("export", 100)
	built.BuildProfile = &models.BuildProfile{Kind: "lama"}
	h.install(built, 2*gib, old)
	h.install(governed("unknown", 100), 2*gib, 0)
	rec := h.installer.records["unknown"]
	rec.InstalledAt = time.Time{}
	h.installer.records["unknown"] = rec
	imported := governed("imported", 100)
	h.custom = []models.Model{imported}
	h.installer.sizes["imported"] = 2 * gib
	h.installer.records["imported"] = models.InstallRecord{ID: "imported", Installed: true, Path: "/models/imported", InstalledAt: testNow.Add(-old)}
	if err := h.svc.SetKeep(context.Background(), "kept", true, "showcase"); err != nil {
		t.Fatal(err)
	}
	h.active["running"] = true
	h.inUse["leased"] = true

	report, err := h.svc.Plan(context.Background(), "test", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"kept": ProtectKept, "running": ProtectActiveJob, "leased": ProtectInUse, "fresh": ProtectRecentlyInstalled,
		"manual": ProtectNotReacquirable, "imported": ProtectNotReacquirable, "export": ProtectLocallyBuilt, "unknown": ProtectUseUnknown,
	}
	for id, reason := range want {
		c := findCandidate(t, report, id)
		if !hasProtection(c, reason) || c.Selected {
			t.Errorf("%s: protected=%v selected=%t, want %s and not selected", id, c.Protected, c.Selected, reason)
		}
	}
	if c := findCandidate(t, report, "eligible"); !c.Selected || c.Eligible() == false {
		t.Fatalf("eligible model not selected under reserve deficit: %+v", c)
	}
}

func TestPlanUsesInstallTimeAsLastUseFloor(t *testing.T) {
	h := newHarness(t, 60*gib, Config{AutoOffload: true})
	h.install(governed("never-used", 10), 3*gib, 10*24*time.Hour)
	h.install(governed("used-recently", 10), 3*gib, 40*24*time.Hour)
	h.install(governed("reinstalled", 10), 3*gib, 2*24*time.Hour)
	h.use["used-recently"] = testNow.Add(-36 * time.Hour)
	h.use["reinstalled"] = testNow.Add(-20 * 24 * time.Hour) // older than its reinstall

	report, err := h.svc.Plan(context.Background(), "test", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	never := findCandidate(t, report, "never-used")
	if never.LastUseBasis != UseBasisInstall || never.Idle != 10*24*time.Hour || !never.Selected {
		t.Fatalf("never-used: basis=%s idle=%s selected=%t", never.LastUseBasis, never.Idle, never.Selected)
	}
	recent := findCandidate(t, report, "used-recently")
	if recent.LastUseBasis != UseBasisSuccessfulJob || recent.Selected {
		t.Fatalf("used-recently: basis=%s selected=%t", recent.LastUseBasis, recent.Selected)
	}
	re := findCandidate(t, report, "reinstalled")
	if re.LastUseBasis != UseBasisInstall || re.Selected {
		t.Fatalf("reinstalled: basis=%s idle=%s selected=%t", re.LastUseBasis, re.Idle, re.Selected)
	}
}

func TestOffloadApplyRemovesAndWritesReceipts(t *testing.T) {
	h := newHarness(t, 5*gib, Config{AutoOffload: true})
	h.install(governed("a", 10), 10*gib, 30*24*time.Hour)
	h.install(governed("busy", 10), 10*gib, 60*24*time.Hour)
	h.installer.removeErr["busy"] = fmt.Errorf("models: %w", models.ErrModelInUse)

	dry, receipts, err := h.svc.Offload(context.Background(), TriggerRequested, 0, nil, true)
	if err != nil || len(receipts) != 0 || len(h.installer.removed) != 0 {
		t.Fatalf("dry-run removed weights: receipts=%v removed=%v err=%v", receipts, h.installer.removed, err)
	}
	if dry.SelectedBytes != 20*gib {
		t.Fatalf("dry-run selected %d bytes", dry.SelectedBytes)
	}
	_, receipts, err = h.svc.Offload(context.Background(), TriggerRequested, 0, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	outcomes := map[string]Receipt{}
	for _, r := range receipts {
		outcomes[r.ModelID] = r
	}
	if outcomes["a"].Outcome != OutcomeRemoved || outcomes["a"].FreeAfterBytes-outcomes["a"].FreeBeforeBytes != 10*gib {
		t.Fatalf("a receipt: %+v", outcomes["a"])
	}
	if outcomes["busy"].Outcome != OutcomeSkipped || outcomes["busy"].Error != ProtectInUse {
		t.Fatalf("busy receipt: %+v", outcomes["busy"])
	}
	stored, err := h.svc.Receipts(context.Background(), 10)
	if err != nil || len(stored) != 2 {
		t.Fatalf("stored receipts=%v err=%v", stored, err)
	}
	if stored[0].Checksum == "" || stored[0].LastUseBasis != UseBasisInstall || stored[0].Mode != ModeReserveDeficit {
		t.Fatalf("receipt lacks decision evidence: %+v", stored[0])
	}
}

func TestOffloadReChecksKeepPinBeforeRemoval(t *testing.T) {
	h := newHarness(t, 5*gib, Config{AutoOffload: true})
	h.install(governed("a", 10), 10*gib, 30*24*time.Hour)
	report, err := h.svc.Plan(context.Background(), "test", 0, nil)
	if err != nil || !findCandidate(t, report, "a").Selected {
		t.Fatalf("setup: a not selected (%v)", err)
	}
	if err := h.svc.SetKeep(context.Background(), "a", true, ""); err != nil {
		t.Fatal(err)
	}
	receipts, err := h.svc.Apply(context.Background(), report)
	if err != nil || len(receipts) != 1 || receipts[0].Outcome != OutcomeSkipped || len(h.installer.removed) != 0 {
		t.Fatalf("kept model removed: receipts=%+v removed=%v err=%v", receipts, h.installer.removed, err)
	}
}

func TestSetKeepRejectsUnknownModel(t *testing.T) {
	h := newHarness(t, 50*gib, Config{})
	if err := h.svc.SetKeep(context.Background(), "nope", true, ""); !errors.Is(err, models.ErrModelNotFound) {
		t.Fatalf("err=%v, want ErrModelNotFound", err)
	}
}

func TestEnsureInstalledAcquiresGovernedModel(t *testing.T) {
	h := newHarness(t, 200*gib, Config{AutoAcquire: true, AutoOffload: true})
	h.seed["dncnn"] = governed("dncnn", 5)
	var messages []string
	err := h.svc.EnsureInstalled(context.Background(), "dncnn", "job-1", func(_ int, m string) { messages = append(messages, m) })
	if err != nil {
		t.Fatal(err)
	}
	if !h.installer.Installed(context.Background(), "dncnn") {
		t.Fatal("model not installed")
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, "acquiring weights for dncnn") {
		t.Fatalf("progress did not say it was acquiring weights: %q", joined)
	}
	if err := h.svc.EnsureInstalled(context.Background(), "dncnn", "job-2", nil); err != nil || h.installer.installs.Load() != 1 {
		t.Fatalf("installed model was re-acquired: installs=%d err=%v", h.installer.installs.Load(), err)
	}
}

func TestEnsureInstalledWaitsWhenOtherReservationsHoldTheDisk(t *testing.T) {
	// Raw headroom is ample, but another job's reservation makes shared disk
	// admission refuse the peak: the job must requeue, not fail.
	h := newHarness(t, 200*gib, Config{AutoAcquire: true, AutoOffload: true})
	h.seed["smolvlm"] = governed("smolvlm", 400)
	h.installer.installFn = func(string) error {
		return fmt.Errorf("models: reserve complete install disk peak: %w: 164 GB reserved", sharedcapacity.ErrDiskAdmissionDenied)
	}
	err := h.svc.EnsureInstalled(context.Background(), "smolvlm", "job-1", nil)
	if !errors.Is(err, ErrAcquireDiskBusy) || !errors.Is(err, jobs.ErrCapacityBusy) {
		t.Fatalf("err=%v, want a retryable shared-disk wait", err)
	}
	if receipts, _ := h.store.Receipts(context.Background(), 10); len(receipts) != 0 {
		t.Fatalf("contention offloaded weights: %+v", receipts)
	}
}

func TestEnsureInstalledRefusesIneligibleSources(t *testing.T) {
	manual := governed("manual", 5)
	manual.Source = fetch.Source{Manual: true, DocsURL: "https://example.invalid/docs"}
	disabled := governed("disabled", 5)
	nonCommercial := governed("nc", 5)
	nonCommercial.CapabilityLabels.CommercialUse = models.CommercialUseNo
	// A .pth checkpoint on the onnxruntime backend can never pass the load
	// smoke (the live dncnn seed shape); downloading it would only fail later.
	unsmokable := governed("unsmokable", 5)
	unsmokable.Source.Assets = []fetch.Asset{{Filename: "unsmokable.pth", URL: "https://example.invalid/unsmokable", Kind: fetch.ArtifactBinary}}
	// The live mi-gan shape: governed asset, but its native backend is not
	// provisioned, so the install would end needs_backend.
	noBackend := governed("no-backend", 5)
	noBackend.Backend = "iopaint"
	cases := []struct {
		name   string
		cfg    Config
		model  *models.Model
		custom bool
		want   string
	}{
		{"switch off", Config{AutoAcquire: false}, ptr(governed("m", 5)), false, "switched off"},
		{"manual source", Config{AutoAcquire: true}, &manual, false, "manual source"},
		{"custom entry", Config{AutoAcquire: true}, ptr(governed("custom", 5)), true, "custom or imported"},
		{"disabled", Config{AutoAcquire: true}, &disabled, false, "disabled"},
		{"rights", Config{AutoAcquire: true}, &nonCommercial, false, "rights do not permit local use"},
		{"no load-smoke contract", Config{AutoAcquire: true}, &unsmokable, false, "no load-smoke contract"},
		{"backend not ready", Config{AutoAcquire: true}, &noBackend, false, "without a runnable backend"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, 200*gib, tc.cfg)
			if tc.custom {
				h.custom = []models.Model{*tc.model}
			} else {
				h.seed[tc.model.ID] = *tc.model
			}
			h.enabled["disabled"] = false
			err := h.svc.EnsureInstalled(context.Background(), tc.model.ID, "job", nil)
			if !errors.Is(err, ErrAutoAcquireUnavailable) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want ErrAutoAcquireUnavailable mentioning %q", err, tc.want)
			}
			if h.installer.installs.Load() != 0 {
				t.Fatal("an ineligible model was acquired")
			}
		})
	}
}

func TestEnsureInstalledOffloadsIdleWeightsFirst(t *testing.T) {
	h := newHarness(t, 22*gib, Config{AutoAcquire: true, AutoOffload: true})
	h.install(governed("idle-big", 10), 8*gib, 20*24*time.Hour)
	h.install(governed("kept-big", 10), 8*gib, 90*24*time.Hour)
	if err := h.svc.SetKeep(context.Background(), "kept-big", true, ""); err != nil {
		t.Fatal(err)
	}
	h.seed["new"] = governed("new", 6*1024)
	h.installer.sizes["new"] = 6 * gib
	if err := h.svc.EnsureInstalled(context.Background(), "new", "job", nil); err != nil {
		t.Fatal(err)
	}
	if len(h.installer.removed) != 1 || h.installer.removed[0] != "idle-big" {
		t.Fatalf("removed %v, want only idle-big", h.installer.removed)
	}
	receipts, _ := h.svc.Receipts(context.Background(), 5)
	if len(receipts) != 1 || receipts[0].Trigger != TriggerAcquire+":new" {
		t.Fatalf("receipts=%+v", receipts)
	}
}

func TestEnsureInstalledFailsActionablyWhenStillShort(t *testing.T) {
	h := newHarness(t, 2*gib, Config{AutoAcquire: true, AutoOffload: true})
	h.install(governed("fresh", 10), 3*gib, 5*time.Minute)
	h.seed["huge"] = governed("huge", 10*1024)
	err := h.svc.EnsureInstalled(context.Background(), "huge", "job", nil)
	if !errors.Is(err, ErrAcquireInsufficientDisk) {
		t.Fatalf("err=%v, want ErrAcquireInsufficientDisk", err)
	}
	for _, hint := range []string{"floor", "models unkeep", "models storage"} {
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("error %q lacks hint %q", err, hint)
		}
	}
	if h.installer.installs.Load() != 0 || len(h.installer.removed) != 0 {
		t.Fatalf("installs=%d removed=%v", h.installer.installs.Load(), h.installer.removed)
	}
}

func TestEnsureInstalledSingleFlightsConcurrentJobs(t *testing.T) {
	h := newHarness(t, 200*gib, Config{AutoAcquire: true, AutoOffload: true})
	h.seed["m"] = governed("m", 5)
	h.installer.block = make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- h.svc.EnsureInstalled(context.Background(), "m", fmt.Sprintf("job-%d", i), nil)
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(h.installer.block)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if n := h.installer.installs.Load(); n != 1 {
		t.Fatalf("installs=%d, want one shared acquisition", n)
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{EnvAutoOffload: "off", EnvReserve: "8GiB", EnvAmple: "64GiB", EnvMinIdle: "6h", EnvInterval: "5m"}
	cfg, err := ConfigFromEnv(func(k string) string { return env[k] }, 120*gib)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoAcquire || cfg.AutoOffload || cfg.Policy.ReserveBytes != 8*gib || cfg.Policy.AmpleBytes != 64*gib ||
		cfg.Policy.MinIdle != 6*time.Hour || cfg.Interval != 5*time.Minute || cfg.Policy.FloorBytes != 120*gib {
		t.Fatalf("cfg=%+v", cfg)
	}
	for _, bad := range []map[string]string{{EnvAutoAcquire: "maybe"}, {EnvReserve: "8"}, {EnvReserve: "90GiB", EnvAmple: "64GiB"}, {EnvMinIdle: "-1h"}} {
		if _, err := ConfigFromEnv(func(k string) string { return bad[k] }, 120*gib); err == nil {
			t.Errorf("config %v accepted", bad)
		}
	}
}

func TestMeasureAllocatedCountsHardLinksOnce(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 64<<10)
	if err := os.WriteFile(filepath.Join(dir, "w.bin"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(dir, "w.bin"), filepath.Join(dir, "link.bin")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "w.bin"), filepath.Join(dir, "sym.bin")); err != nil {
		t.Fatal(err)
	}
	got, err := MeasureAllocated(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got < int64(len(data)) || got >= 2*int64(len(data)) {
		t.Fatalf("allocated=%d, want one copy of %d bytes", got, len(data))
	}
	if missing, err := MeasureAllocated(filepath.Join(dir, "absent")); err != nil || missing != 0 {
		t.Fatalf("missing path: %d %v", missing, err)
	}
}

func ptr(m models.Model) *models.Model { return &m }

func TestEnsureInstalledRevalidatesPendingSmokeWithoutDiskHeadroom(t *testing.T) {
	// Headroom is far below a full acquisition peak: revalidating weights on
	// disk must neither need it nor offload anything.
	h := newHarness(t, gib/2, Config{AutoAcquire: true, AutoOffload: true})
	manual := governed("ip2p", 7000)
	manual.Source.Manual, manual.Source.DocsURL = true, "https://example.invalid/docs"
	h.seed["ip2p"] = manual // source gates do not apply to weights on disk
	h.installer.records["ip2p"] = models.InstallRecord{ID: "ip2p", Installed: true, Path: "/models/models/ip2p", InstalledAt: testNow.Add(-48 * time.Hour)}
	h.runtime = map[string]string{"ip2p": "smoke_pending"}
	h.installer.installFn = func(id string) error { delete(h.runtime, id); return nil }
	var messages []string
	if err := h.svc.EnsureInstalled(context.Background(), "ip2p", "job-1", func(_ int, m string) { messages = append(messages, m) }); err != nil {
		t.Fatalf("revalidate: %v", err)
	}
	if h.installer.installs.Load() != 1 || !strings.Contains(strings.Join(messages, "\n"), "revalidating ip2p (smoke_pending)") {
		t.Fatalf("installs=%d messages=%q, want one announced revalidation", h.installer.installs.Load(), messages)
	}
	if receipts, _ := h.store.Receipts(context.Background(), 10); len(receipts) != 0 {
		t.Fatalf("revalidation offloaded weights: %+v", receipts)
	}
	if err := h.svc.EnsureInstalled(context.Background(), "ip2p", "job-2", nil); err != nil || h.installer.installs.Load() != 1 {
		t.Fatalf("ready model revalidated again: installs=%d err=%v", h.installer.installs.Load(), err)
	}
}

func TestRevalidationRefusesReasonsItCannotResolve(t *testing.T) {
	h := newHarness(t, 200*gib, Config{AutoAcquire: true})
	h.seed["m"] = governed("m", 5)
	for _, reason := range []string{"smoke_failed", "smoke_unsupported", "env_not_provisioned"} {
		if err := h.svc.Revalidatable(context.Background(), h.seed["m"], reason); !errors.Is(err, ErrAutoAcquireUnavailable) {
			t.Fatalf("%s: err=%v, want a refusal", reason, err)
		}
	}
	h.installer.records["m"] = models.InstallRecord{ID: "m", Installed: true, Path: "/models/models/m"}
	h.runtime = map[string]string{"m": "smoke_failed"}
	if err := h.svc.EnsureInstalled(context.Background(), "m", "job", nil); err != nil || h.installer.installs.Load() != 0 {
		t.Fatalf("a fresh failed smoke was re-run: installs=%d err=%v", h.installer.installs.Load(), err)
	}
	for _, reason := range []string{"smoke_pending", "build_not_ready", "needs_backend"} {
		if err := h.svc.Revalidatable(context.Background(), h.seed["m"], reason); err != nil {
			t.Fatalf("%s: err=%v, want revalidation allowed", reason, err)
		}
	}
}
