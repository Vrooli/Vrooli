package destinations

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	destdomain "data-backup-manager/internal/destinations"
	destmocks "data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/repohealth"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

var backupTestNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

const (
	tempA = "s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838"
	tempB = "s/4e5/7bdb040b1caeb47f4c5ba9c2867c5-sc3c89b33ffab9119144.f.tmp.dde5f267db32227a"
	tempC = "s/afb/7be9339fbf00b7cf599029ce3c07c-s4654612484971716144.f.tmp.50316f5a0d2cbe34"
	blobQ = "q/63/90189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144.f"
)

// faultFS is a real directory whose chosen entries cannot be stat-ed, like the
// unreadable leftovers on the damaged NTFS drive. It cannot write.
type faultFS struct {
	repohealth.OSFS
	mu   sync.Mutex
	root string
	fail map[string]error
}

func (f *faultFS) Lstat(p string) (fs.FileInfo, error) {
	f.mu.Lock()
	err, bad := f.fail[p]
	f.mu.Unlock()
	if bad {
		return nil, &fs.PathError{Op: "lstat", Path: p, Err: err}
	}
	return os.Lstat(p)
}

func (f *faultFS) breakEntry(rel string) {
	full := filepath.Join(f.root, filepath.FromSlash(rel))
	_ = os.MkdirAll(filepath.Dir(full), 0o755)
	_ = os.WriteFile(full, []byte("x"), 0o644)
	f.mu.Lock()
	f.fail[full] = syscall.EIO
	f.mu.Unlock()
}

func (f *faultFS) heal(rel string) {
	f.mu.Lock()
	delete(f.fail, filepath.Join(f.root, filepath.FromSlash(rel)))
	f.mu.Unlock()
}

// firstFailure is what kopia reports: it stops at the first unreadable path.
func (f *faultFS) firstFailure() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fail) == 0 {
		return nil
	}
	paths := make([]string, 0, len(f.fail))
	for p := range f.fail {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return fmt.Errorf("kopia blob stats --raw: error listing blobs: error reading directory: lstat %s: input/output error", paths[0])
}

type backupKopia struct {
	engine.KopiaEngine
	fs              *faultFS
	statusErr       error
	listErr         error
	snaps           []engine.Snapshot
	statusCalls     int
	listCalls       int
	statsCalls      int
	cacheCandidate  engine.OrphanBlobCandidate
	cacheProbeCalls int
	blobCommitCalls int
}

type disabledRecoveryPublisher struct{ *recoveryDecisionPublisher }

func (disabledRecoveryPublisher) Enabled() bool { return false }

func (k *backupKopia) RepoStatus(context.Context, string) (engine.RepoStatus, error) {
	k.statusCalls++
	return engine.RepoStatus{Connected: true}, k.statusErr
}

func (k *backupKopia) SnapshotList(context.Context, string, string) ([]engine.Snapshot, error) {
	k.listCalls++
	return k.snaps, k.listErr
}

func (k *backupKopia) RepoStats(context.Context, string) (engine.RepoStats, error) {
	k.statsCalls++
	return engine.RepoStats{}, k.fs.firstFailure()
}

func (k *backupKopia) RepoCacheProbe(_ context.Context, _ string, blobID string) (engine.OrphanBlobCandidate, error) {
	k.cacheProbeCalls++
	if k.cacheCandidate.BlobID != blobID {
		return engine.OrphanBlobCandidate{}, fmt.Errorf("candidate id mismatch")
	}
	return k.cacheCandidate, nil
}

func (k *backupKopia) RepoRestoreTemp(_ context.Context, repo, root, rel, digest string) (engine.RepoTempBlobRestore, error) {
	k.blobCommitCalls++
	k.listErr = nil
	source := filepath.Join(root, filepath.FromSlash(rel))
	final := strings.SplitN(filepath.Base(rel), ".tmp", 2)[0]
	return engine.RepoTempBlobRestore{Repository: repo, RelativePath: rel, Source: source,
		Target: filepath.Join(filepath.Dir(source), final), SHA256: digest, Bytes: k.cacheCandidate.CandidateBytes,
		SourcePreserved: true, Committed: true}, nil
}

func (k *backupKopia) RepoDamageScan(ctx context.Context, _ string) (engine.RepoDamageScan, error) {
	scan, err := (&repohealth.Scanner{FS: k.fs}).Walk(ctx, k.fs.root)
	if err != nil {
		return engine.RepoDamageScan{}, err
	}
	result := engine.RepoDamageScan{Root: scan.Root, StartedAt: scan.StartedAt.Format(time.RFC3339Nano), DurationNS: int64(scan.Duration), Files: scan.Files, Bytes: scan.Bytes, TempFiles: scan.TempFiles}
	for _, entry := range scan.Unreadable {
		result.Unreadable = append(result.Unreadable, engine.RepoDamageEntry{RelativePath: entry.RelPath, Class: string(entry.Class), BlobKind: string(entry.BlobKind), Problem: string(entry.Problem), Orphan: entry.Orphan, Detail: entry.Detail})
	}
	return result, nil
}

type fakeRepoRepair struct {
	calls     []string
	result    engine.RepoQuarantineResult
	root      string
	fs        *faultFS
	failAt    int
	failed    bool
	completed map[string]engine.RepoQuarantineResult
}

func (r *fakeRepoRepair) RepoQuarantineTempBlob(_ context.Context, repo, relativePath string) (engine.RepoQuarantineResult, error) {
	r.calls = append(r.calls, repo+":"+relativePath)
	if result, ok := r.completed[relativePath]; ok {
		return result, nil
	}
	if r.failAt > 0 && len(r.calls) == r.failAt && !r.failed {
		r.failed = true
		return engine.RepoQuarantineResult{}, fmt.Errorf("simulated interruption after prior quarantine")
	}
	result := r.result
	result.Repository = repo
	result.OriginalPath = filepath.Join(r.root, filepath.FromSlash(relativePath))
	result.QuarantinePath = filepath.Join(filepath.Dir(r.root), filepath.Base(r.root)+"-recovery-quarantine-test", filepath.FromSlash(relativePath))
	if r.fs != nil {
		r.fs.heal(relativePath)
	}
	if r.completed == nil {
		r.completed = map[string]engine.RepoQuarantineResult{}
	}
	r.completed[relativePath] = result
	return result, nil
}

type fakeHistory struct {
	mu        sync.Mutex
	plans     []PlanRecord
	runs      map[string]RunRecord
	success   map[string]PlanSuccess
	labels    map[string]TargetLabel
	active    bool
	triggered []string
}

func (h *fakeHistory) Plans(context.Context) ([]PlanRecord, error) { return h.plans, nil }

func (h *fakeHistory) LatestRun(_ context.Context, id string) (RunRecord, bool, error) {
	run, ok := h.runs[id]
	return run, ok, nil
}

func (h *fakeHistory) PlanSuccess(_ context.Context, id string) (PlanSuccess, error) {
	return h.success[id], nil
}

func (h *fakeHistory) TargetLabels(context.Context, []string) (map[string]TargetLabel, error) {
	return h.labels, nil
}
func (h *fakeHistory) ActiveRun(context.Context, string) (bool, error) { return h.active, nil }

func (h *fakeHistory) TriggerRun(_ context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.triggered = append(h.triggered, id)
	return nil
}

type backupHarness struct {
	t          *testing.T
	now        time.Time
	worker     *BackupFailureWorker
	flow       *BackupFailureFlow
	volume     *RecoveryDecisionFlow
	publisher  *recoveryDecisionPublisher
	history    *fakeHistory
	fs         *faultFS
	kopia      *backupKopia
	usage      *repohealth.UsageEngine
	store      *repohealth.FileExceptionStore
	incidents  *repohealth.FileIncidentStore
	remed      *decisionTestRemediator
	repoRepair *fakeRepoRepair
	dest       destdomain.Destination
	mounted    *bool
}

func newBackupHarness(t *testing.T, unreadable ...string) *backupHarness {
	t.Helper()
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repositories", "elements-local.kopia")
	if err := os.MkdirAll(filepath.Join(repoDir, "p", "bbb"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "p", "bbb", "7098cafebeccc21a63be88c33846b-s9e453c5eb992a889141.f"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &backupHarness{t: t, now: backupTestNow, fs: &faultFS{root: repoDir, fail: map[string]error{}}, publisher: &recoveryDecisionPublisher{}}
	for _, rel := range unreadable {
		h.fs.breakEntry(rel)
	}
	identity := destinationreadiness.DeviceIdentity{
		DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", Label: "Elements", Filesystem: "ntfs",
		UUID: "uuid-elements", Serial: "serial-elements", TotalBytes: 100,
	}
	h.dest = destdomain.Destination{
		ID: "dest-1", Name: "elements-local", BackendKind: destdomain.BackendFilesystem,
		Location: "/media/Elements/vrooli-backups", RepositoryLocation: repoDir, DeviceIdentity: &identity,
	}
	mounted := true
	h.mounted = &mounted
	h.remed = &decisionTestRemediator{mounted: &mounted, consistent: "yes"}
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: identity, mounted: &mounted}, nil).WithRemediator(h.remed)
	recoveries := &destinationreadiness.FileRecoveryStore{Root: filepath.Join(dir, "journals")}
	h.volume = newRecoveryDecisionFlow(readiness, recoveries,
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(dir, "volume-asks")}, h.publisher, true, nil)
	volumeWorker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{h.dest}}, readiness, recoveries, h.volume, nil)

	h.kopia = &backupKopia{fs: h.fs, snaps: []engine.Snapshot{{
		Source: "matthalloran8@swarminator:/home/x/api/dbm:/codex/sessions", StartTime: "2026-08-19T18:58:23.984568593Z",
	}}}
	h.store = &repohealth.FileExceptionStore{Root: filepath.Join(dir, "exceptions")}
	h.usage = &repohealth.UsageEngine{
		KopiaEngine: h.kopia, Exceptions: h.store, Scanner: &repohealth.Scanner{FS: h.fs, Workers: 2},
		Now: func() time.Time { return h.now },
		Locate: func(context.Context, string) (string, string, bool) {
			return h.dest.ID, repoDir, true
		},
	}
	h.history = &fakeHistory{
		plans: []PlanRecord{
			{ID: "plan-elements", Name: "elements-daily-runtime", Enabled: true, Schedule: "24h", TargetIDs: []string{"t-sessions", "t-state", "t-domain"}, DestinationIDs: []string{"dest-1"}, CreatedAt: backupTestNow.Add(-160 * 24 * time.Hour)},
			{ID: "plan-baseline", Name: "baseline-safety-swarm-manager", Enabled: true, Schedule: "24h", TargetIDs: []string{"t-domain"}, DestinationIDs: []string{"dest-baseline"}, CreatedAt: backupTestNow.Add(-90 * 24 * time.Hour)},
		},
		runs: map[string]RunRecord{
			"plan-elements": {
				ID: "run-1", Status: "failed", Terminal: true, FinishedAt: backupTestNow.Add(-time.Hour),
				Failures: []DestinationFailure{{DestinationID: "dest-1", Code: "repository_invalid"}},
			},
		},
		success: map[string]PlanSuccess{
			"plan-baseline": {At: backupTestNow.Add(-2 * time.Hour), ByTarget: map[string]time.Time{"t-domain": backupTestNow.Add(-2 * time.Hour)}},
		},
		labels: map[string]TargetLabel{
			"t-sessions": {Owner: "codex", Name: "sessions"}, "t-state": {Owner: "codex", Name: "state"}, "t-domain": {Owner: "swarm-manager", Name: "domain-data"},
		},
	}
	h.incidents = &repohealth.FileIncidentStore{Root: filepath.Join(dir, "incidents")}
	clock := func() time.Time { return h.now }
	h.repoRepair = &fakeRepoRepair{root: repoDir, fs: h.fs, result: engine.RepoQuarantineResult{
		Repository: "elements-local", OriginalPath: filepath.Join(repoDir, tempA),
		QuarantinePath:       repoDir + "-recovery-quarantine-test",
		FinalBlobPath:        filepath.Join(repoDir, "s", "123", "blob.f"),
		QuarantineSameVolume: true, Committed: true,
	}}
	h.flow = newBackupFailureFlow(BackupFailureFlowDeps{
		Readiness: readiness, Volume: h.volume, Recoveries: recoveries,
		Asks:      &destinationreadiness.FileDecisionStore{Root: filepath.Join(dir, "backup-asks")},
		Publisher: h.publisher, Incidents: h.incidents, Exceptions: h.store, Scanner: h.usage.Scanner, Usage: h.usage,
		RepoRepair: h.repoRepair,
		History:    h.history, Dests: &destmocks.FakeService{GetOut: h.dest, ListOut: []destdomain.Destination{h.dest}}, Enabled: true, Now: clock,
	})
	h.worker = newBackupFailureWorker(BackupFailureWorkerDeps{
		History: h.history, Dests: &destmocks.FakeService{GetOut: h.dest, ListOut: []destdomain.Destination{h.dest}},
		Readiness: readiness, Flow: h.flow, Volume: volumeWorker, Usage: h.usage, Exceptions: h.store, Incidents: h.incidents,
		Notices: &repohealth.NoticeStateStore{Path: filepath.Join(dir, "notices", "state.json")}, Publisher: h.publisher,
		Recoveries: recoveries, Now: clock,
	})
	h.kopia.fs = h.fs
	return h
}

func (h *backupHarness) reconcile() { h.worker.reconcile(context.Background()) }

func (h *backupHarness) upkeep() {
	h.worker.upkeepIncidents(context.Background(), h.history.plans, map[string]destdomain.Destination{h.dest.ID: h.dest})
}

func (h *backupHarness) restartFailureFlow() {
	incidents := &repohealth.FileIncidentStore{Root: h.incidents.Root}
	exceptions := &repohealth.FileExceptionStore{Root: h.store.Root}
	recoveries := &destinationreadiness.FileRecoveryStore{Root: h.flow.recoveries.(*destinationreadiness.FileRecoveryStore).Root}
	asks := &destinationreadiness.FileDecisionStore{Root: h.flow.asks.(*destinationreadiness.FileDecisionStore).Root}
	flow := newBackupFailureFlow(BackupFailureFlowDeps{
		Readiness: h.flow.readiness, Volume: h.flow.volume, Recoveries: recoveries, Asks: asks, Publisher: h.publisher,
		Incidents: incidents, Exceptions: exceptions, Scanner: h.usage.Scanner, Usage: h.usage, RepoRepair: h.repoRepair,
		History: h.history, Dests: &destmocks.FakeService{GetOut: h.dest, ListOut: []destdomain.Destination{h.dest}},
		Enabled: true, Now: func() time.Time { return h.now },
	})
	h.incidents, h.store, h.flow = incidents, exceptions, flow
	h.worker.incidents, h.worker.exceptions, h.worker.flow = incidents, exceptions, flow
	h.worker.recoveries = recoveries
}

func (h *backupHarness) events(eventType string) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range h.publisher.events {
		if e.EventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

func (h *backupHarness) decisions() []eventbus.DomainEvent {
	return h.events(operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind))
}

func (h *backupHarness) notices(category string) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range h.events(backupNotificationType) {
		if e.Payload["category"] == category {
			out = append(out, e)
		}
	}
	return out
}

func (h *backupHarness) openIncident() repohealth.Incident {
	h.t.Helper()
	all, err := h.incidents.List(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	for _, inc := range all {
		if inc.State.Open() {
			return inc
		}
	}
	h.t.Fatalf("no open incident among %+v", all)
	return repohealth.Incident{}
}

func (h *backupHarness) resolution(inc repohealth.Incident, answer string, edit ...func(*operatordecision.DecisionResolution)) operatordecision.DecisionResolution {
	res := operatordecision.DecisionResolution{
		AskID: inc.AskID, State: operatordecision.StateAnswered, Answer: answer,
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"incident_id": inc.ID, "generation": fmt.Sprint(inc.Generation), "destination_id": inc.DestinationID},
	}
	for _, e := range edit {
		e(&res)
	}
	return res
}

func (h *backupHarness) apply(res operatordecision.DecisionResolution) {
	h.t.Helper()
	if handled, err := h.flow.Apply(context.Background(), res); err != nil || !handled {
		h.t.Fatalf("Apply = (%v, %v), want handled", handled, err)
	}
}

func (h *backupHarness) exceptions() []repohealth.Exception {
	list, err := h.store.List(context.Background(), h.dest.ID)
	if err != nil {
		h.t.Fatal(err)
	}
	return list
}

func TestBackupFailureRaisesOneDecisionAndRepresentsItUntilAnswered(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB, tempC)
	for i := 0; i < 4; i++ {
		h.now = h.now.Add(2 * time.Minute)
		h.reconcile()
	}
	decisions := h.decisions()
	if len(decisions) != 1 {
		t.Fatalf("decision events = %d, want exactly one however many sweeps pass", len(decisions))
	}
	payload := decisions[0].Payload
	if !strings.Contains(payload["subject"].(string), "82 days") || payload["recommended"] != repohealth.OptKopiaQuarantine {
		t.Fatalf("decision subject/recommended = %v / %v", payload["subject"], payload["recommended"])
	}
	question := payload["question"].(string)
	for _, want := range []string{"3 unfinished temporary files", "2 of the 3 items", "19 Aug"} {
		if !strings.Contains(question, want) {
			t.Errorf("question omits %q: %s", want, question)
		}
	}
	inc := h.openIncident()
	if inc.State != repohealth.IncidentAwaiting || inc.Diagnosis.Cause != repohealth.CauseLeftoverTempUnreadable || len(inc.Diagnosis.Unreadable) != 3 {
		t.Fatalf("incident = %+v", inc)
	}
	if len(h.history.triggered) != 0 || h.remed.calls != 0 || len(h.exceptions()) != 0 {
		t.Fatal("raising the question must change nothing: no run, no host action, no exception")
	}
}

func TestBackupFailureDefersDiagnosisWhenDestinationReadinessCannotBeProven(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.worker.notices = nil
	dests := h.worker.dests.(*destmocks.FakeService)
	dests.ResolveErr = fmt.Errorf("filesystem readiness is unknown")
	h.reconcile()
	incidents, err := h.incidents.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(incidents) != 0 || len(h.decisions()) != 0 || len(h.history.triggered) != 0 {
		t.Fatalf("stale repository diagnosis caused an incident, notice or retry: incidents=%d notices=%d runs=%v", len(incidents), len(h.decisions()), h.history.triggered)
	}
	if h.kopia.statusCalls != 0 || h.kopia.listCalls != 0 || h.kopia.statsCalls != 0 {
		t.Fatalf("Kopia probed after destination readiness failed: status=%d list=%d stats=%d", h.kopia.statusCalls, h.kopia.listCalls, h.kopia.statsCalls)
	}
}

func TestStalledNoticeDoesNotUseCachedRepositoryEvidenceWhenDestinationCannotResolve(t *testing.T) {
	h := newBackupHarness(t)
	dests := h.worker.dests.(*destmocks.FakeService)
	dests.ResolveErr = fmt.Errorf("filesystem readiness is unknown")
	got := h.worker.repoSuccess(context.Background(), h.dest, map[string]bool{"t-sessions": true}, map[string]TargetLabel{
		"t-sessions": {Owner: "codex", Name: "sessions"},
	})
	if !got.IsZero() {
		t.Fatalf("repository evidence = %s, want unknown while destination cannot resolve", got)
	}
	if h.kopia.listCalls != 0 {
		t.Fatalf("snapshot listing occurred after destination resolution failed: %d calls", h.kopia.listCalls)
	}
}

func TestBackupFailureQuestionIsRetriedWhenTheHubWasDown(t *testing.T) {
	h := newBackupHarness(t, tempA)
	// The first sweep tries twice (diagnosis, then upkeep); the hub refuses both.
	h.publisher.failNext = 2
	h.reconcile()
	if len(h.decisions()) != 0 {
		t.Fatal("the hub refused the first publication")
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatalf("decision events = %d after the hub recovered, want the one question delivered", len(h.decisions()))
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatal("a delivered question was published again")
	}
}

func TestBackupFailureAnswerNeedsAVerifiedHuman(t *testing.T) {
	cases := map[string]func(*operatordecision.DecisionResolution){
		"agent identity":     func(r *operatordecision.DecisionResolution) { r.AnsweredByIdentityKind = "agent" },
		"no identity source": func(r *operatordecision.DecisionResolution) { r.AnsweredByIdentitySource = "" },
		"no answerer":        func(r *operatordecision.DecisionResolution) { r.AnsweredBy = "" },
		"applied by default": func(r *operatordecision.DecisionResolution) { r.ByDefault = true },
		"dismissed":          func(r *operatordecision.DecisionResolution) { r.State = operatordecision.StateDismissed },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			h := newBackupHarness(t, tempA)
			h.reconcile()
			inc := h.openIncident()
			h.apply(h.resolution(inc, repohealth.OptAccept, edit))
			if len(h.exceptions()) != 0 || len(h.history.triggered) != 0 || h.remed.calls != 0 {
				t.Fatalf("an unverified answer changed state: exceptions=%d runs=%v host=%d", len(h.exceptions()), h.history.triggered, h.remed.calls)
			}
			if got := h.openIncident(); got.State != repohealth.IncidentDeclined {
				t.Fatalf("incident state = %s, want declined", got.State)
			}
			if len(h.notices(categoryBackupNeedsAttention)) != 1 {
				t.Fatal("the operator must be told nothing was changed")
			}
		})
	}
}

func TestBackupFailureAcceptRecordsExactExceptionsAndResumesBackups(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, repohealth.OptAccept))

	got := h.exceptions()
	if len(got) != 2 {
		t.Fatalf("exceptions = %+v, want one per exact path shown", got)
	}
	paths := []string{got[0].RelPath, got[1].RelPath}
	sort.Strings(paths)
	if paths[0] != tempB || paths[1] != tempA {
		t.Fatalf("recorded paths = %v", paths)
	}
	for _, e := range got {
		if e.ApprovedBy != "operator" || e.ApprovedBySource != "cloudflare_access" || !e.ApprovedAt.Equal(backupTestNow) ||
			e.Class != repohealth.ClassLeftoverTemp || e.Problem != repohealth.ProblemIO || e.DestinationID != "dest-1" {
			t.Errorf("exception %+v lacks who/when/what", e)
		}
	}
	if len(h.history.triggered) != 1 || h.history.triggered[0] != "plan-elements" {
		t.Fatalf("triggered = %v, want the blocked plan retried", h.history.triggered)
	}
	if len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatalf("result notices = %+v", h.publisher.events)
	}
	if h.remed.calls != 0 {
		t.Fatal("accepting must not touch the drive")
	}

	// A replay of the same delivery must not trigger another run or notice.
	h.apply(h.resolution(inc, repohealth.OptAccept))
	if len(h.history.triggered) != 1 || len(h.notices(categoryBackupRecovered)) != 1 || len(h.exceptions()) != 2 {
		t.Fatal("a replayed answer was applied twice")
	}
}

func TestBackupFailureApplyingAcceptRecoversAfterRestartWithoutDuplicateRun(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	h.reconcile()
	inc := h.openIncident()
	res := h.resolution(inc, repohealth.OptAccept)
	if err := h.flow.beginApplying(context.Background(), &inc, res); err != nil {
		t.Fatal(err)
	}
	// One exception was durable before the process stopped. Add is idempotent,
	// so restart can finish the exact approved path set without replacing its
	// original approval evidence.
	if err := h.store.Add(context.Background(), repohealth.Exception{
		DestinationID: inc.DestinationID, RelPath: tempA, Class: repohealth.ClassLeftoverTemp,
		BlobKind: repohealth.BlobOther, Problem: repohealth.ProblemIO, ApprovedBy: res.AnsweredBy,
		ApprovedBySource: res.AnsweredByIdentitySource, ApprovedAt: h.now, AskID: inc.AskID,
	}); err != nil {
		t.Fatal(err)
	}
	h.restartFailureFlow()
	h.upkeep()
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplied {
		t.Fatalf("recovered incident = %+v, %v, %v", got, ok, err)
	}
	if len(h.exceptions()) != 2 || len(h.history.triggered) != 1 {
		t.Fatalf("recovery effects exceptions=%d runs=%v", len(h.exceptions()), h.history.triggered)
	}
	h.upkeep()
	if len(h.history.triggered) != 1 {
		t.Fatalf("duplicate upkeep triggered a second backup: %v", h.history.triggered)
	}
}

func TestKopiaQuarantineDecisionRevalidatesOneExactObjectAndPublishesResult(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	if !inc.Offers.KopiaQuarantine || len(inc.Diagnosis.Unreadable) != 1 || inc.Diagnosis.Unreadable[0].RelPath != tempA {
		t.Fatalf("exact-object quarantine was not offered for the incident: %+v", inc)
	}
	decisions := h.decisions()
	if len(decisions) != 1 || decisions[0].Payload["recommended"] != repohealth.OptKopiaQuarantine || !strings.Contains(decisions[0].Payload["question"].(string), "quarantine these 1 exact unreadable Kopia temp") {
		t.Fatalf("Notification Hub facts do not carry the exact quarantine option: %+v", decisions)
	}
	res := h.resolution(inc, repohealth.OptKopiaQuarantine)
	h.apply(res)
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplied {
		t.Fatalf("quarantine incident = %+v, ok=%v, err=%v", got, ok, err)
	}
	if len(h.repoRepair.calls) != 1 || h.repoRepair.calls[0] != "elements-local:"+tempA {
		t.Fatalf("Kopia repair calls = %v, want one exact repo/path call", h.repoRepair.calls)
	}
	if len(h.exceptions()) != 0 || len(h.history.triggered) != 1 || len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatalf("quarantine result exceptions=%d retries=%v notices=%d", len(h.exceptions()), h.history.triggered, len(h.notices(categoryBackupRecovered)))
	}
	if _, err := h.flow.Apply(context.Background(), res); err != nil || len(h.repoRepair.calls) != 1 {
		t.Fatalf("replayed callback should not repeat a committed quarantine: calls=%v err=%v", h.repoRepair.calls, err)
	}
}

func TestKopiaQuarantineRefusesChangedIdentityRootOrDamageSet(t *testing.T) {
	cases := []struct {
		name   string
		change func(*backupHarness)
	}{
		{
			name: "device identity changed",
			change: func(h *backupHarness) {
				service := h.flow.dests.(*destmocks.FakeService)
				current := h.dest
				identity := *h.dest.DeviceIdentity
				identity.Serial = "different-serial"
				current.DeviceIdentity = &identity
				service.ResolveOut = current
			},
		},
		{
			name: "repository root changed",
			change: func(h *backupHarness) {
				service := h.flow.dests.(*destmocks.FakeService)
				current := h.dest
				current.RepositoryLocation = filepath.Join(h.dest.RepositoryLocation, "replacement.kopia")
				service.ResolveOut = current
			},
		},
		{
			name:   "unapproved unreadable entry appeared",
			change: func(h *backupHarness) { h.fs.breakEntry(tempB) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newBackupHarness(t, tempA)
			h.reconcile()
			inc := h.openIncident()
			res := h.resolution(inc, repohealth.OptKopiaQuarantine)
			tc.change(h)
			h.apply(res)
			got, ok, err := h.incidents.Get(context.Background(), inc.ID)
			if err != nil || !ok || got.State != repohealth.IncidentApplying {
				t.Fatalf("unsafe change did not remain applying for review: %+v ok=%t err=%v", got, ok, err)
			}
			if len(h.repoRepair.calls) != 0 || len(h.notices(categoryBackupRecovered)) != 0 || len(h.history.triggered) != 0 {
				t.Fatalf("quarantine effect escaped revalidation: calls=%v notices=%d retries=%v", h.repoRepair.calls, len(h.notices(categoryBackupRecovered)), h.history.triggered)
			}
		})
	}
}

func TestKopiaQuarantineResultMustNameTheExactMovedRelativePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "elements-local.kopia")
	relative := "s/f43/blob.f.tmp.deadbeef"
	result := engine.RepoQuarantineResult{
		Repository: "elements-local", OriginalPath: filepath.Join(root, filepath.FromSlash(relative)),
		QuarantinePath: filepath.Join(filepath.Dir(root), filepath.Base(root)+"-recovery-quarantine-123", filepath.FromSlash(relative)),
		Committed:      true, QuarantineSameVolume: true,
	}
	if !validQuarantineResult(result, "elements-local", root, relative) {
		t.Fatalf("owner's exact same-volume result was rejected: %+v", result)
	}
	result.QuarantinePath = filepath.Join(filepath.Dir(root), filepath.Base(root)+"-recovery-quarantine-123", "s/f43/another.f.tmp.deadbeef")
	if validQuarantineResult(result, "elements-local", root, relative) {
		t.Fatal("owner result for a different quarantined entry was accepted")
	}
}

func TestBackupFailureApplyingRepairRecoversPendingJournalAfterRestart(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	res := h.resolution(inc, repohealth.OptRepair)
	root := filepath.Clean(h.dest.DeviceIdentity.Mountpoint)
	plans, err := h.flow.planRepairSteps(context.Background(), root, *h.dest.DeviceIdentity)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := destinationreadiness.NewRecoveryJournal(root, *h.dest.DeviceIdentity, plans, h.now)
	if err != nil {
		t.Fatal(err)
	}
	journal.ID = operatordecision.CorrelationID("dbm-backup-repair", inc.ID, fmt.Sprint(inc.Generation))
	if err := h.flow.recoveries.Save(context.Background(), journal); err != nil {
		t.Fatal(err)
	}
	inc.RepairJournalID = journal.ID
	if err := h.flow.beginApplying(context.Background(), &inc, res); err != nil {
		t.Fatal(err)
	}
	h.restartFailureFlow()
	h.upkeep()
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplied {
		t.Fatalf("recovered incident = %+v, %v, %v", got, ok, err)
	}
	if h.remed.calls != 3 || len(h.history.triggered) != 1 {
		t.Fatalf("recovery host=%d runs=%v, want three host actions and one backup", h.remed.calls, h.history.triggered)
	}
}

func TestBackupFailureApplyingRepairWithAmbiguousStepRequiresInspection(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	res := h.resolution(inc, repohealth.OptRepair)
	root := filepath.Clean(h.dest.DeviceIdentity.Mountpoint)
	plans, err := h.flow.planRepairSteps(context.Background(), root, *h.dest.DeviceIdentity)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := destinationreadiness.NewRecoveryJournal(root, *h.dest.DeviceIdentity, plans, h.now)
	if err != nil {
		t.Fatal(err)
	}
	journal.ID = operatordecision.CorrelationID("dbm-backup-repair", inc.ID, fmt.Sprint(inc.Generation))
	journal.State, journal.Current, journal.Steps[0].Status = destinationreadiness.RecoveryRunning, 0, destinationreadiness.RecoveryStepInFlight
	if err := h.flow.recoveries.Save(context.Background(), journal); err != nil {
		t.Fatal(err)
	}
	inc.RepairJournalID = journal.ID
	if err := h.flow.beginApplying(context.Background(), &inc, res); err != nil {
		t.Fatal(err)
	}
	h.restartFailureFlow()
	h.upkeep()
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplying || !strings.Contains(got.Result, "Inspect the drive") {
		t.Fatalf("ambiguous recovery = %+v, %v, %v", got, ok, err)
	}
	if h.remed.calls != 0 || len(h.history.triggered) != 0 || len(h.notices(categoryBackupNeedsAttention)) != 1 {
		t.Fatalf("ambiguous recovery actions host=%d runs=%v notices=%d", h.remed.calls, h.history.triggered, len(h.notices(categoryBackupNeedsAttention)))
	}
	h.upkeep()
	if h.remed.calls != 0 || len(h.notices(categoryBackupNeedsAttention)) != 1 {
		t.Fatalf("inspection hold was not stable: host=%d notices=%d", h.remed.calls, len(h.notices(categoryBackupNeedsAttention)))
	}
}

func TestBackupFailureRejectsStaleGenerationDecision(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	old := h.openIncident()
	if err := old.Transition(repohealth.IncidentDeclined); err != nil {
		t.Fatal(err)
	}
	if err := h.incidents.Put(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if err := old.Transition(repohealth.IncidentAwaiting); err != nil {
		t.Fatal(err)
	}
	old.AskID = backupAskID(old.ID, old.Generation)
	old.AskedAt, old.UpdatedAt = h.now.Add(time.Minute), h.now.Add(time.Minute)
	if err := h.incidents.Put(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	stale := h.resolution(old, repohealth.OptAccept)
	stale.Correlation["generation"] = fmt.Sprint(old.Generation - 1)
	stale.AskID = backupAskID(old.ID, old.Generation-1)
	if handled, err := h.flow.Apply(context.Background(), stale); !handled || err != nil {
		t.Fatalf("stale Apply = %v, %v", handled, err)
	}
	current, _, err := h.incidents.Get(context.Background(), old.ID)
	if err != nil || current.Generation != old.Generation || current.State != repohealth.IncidentAwaiting || len(h.exceptions()) != 0 || len(h.history.triggered) != 0 {
		t.Fatalf("stale answer affected current generation: incident=%+v exceptions=%d runs=%v err=%v", current, len(h.exceptions()), h.history.triggered, err)
	}
}

func TestBackupFailureExceptionLetsUsageThroughOnlyForTheApprovedPaths(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	h.reconcile()
	h.apply(h.resolution(h.openIncident(), repohealth.OptAccept))

	stats, err := h.usage.RepoStats(context.Background(), "elements-local")
	if err != nil || stats.SizeBytes < 4096 {
		t.Fatalf("usage with every unreadable entry excepted = %+v, %v", stats, err)
	}

	// A new unreadable entry is not covered: usage fails again and the monitor
	// asks a NEW question for just that entry.
	h.fs.breakEntry(tempC)
	h.usage.InvalidateScan(h.dest.RepositoryLocation)
	if _, err := h.usage.RepoStats(context.Background(), "elements-local"); err == nil || !strings.Contains(err.Error(), "not covered by an approved exception") {
		t.Fatalf("a new unreadable entry must block again, got %v", err)
	}
	h.history.runs["plan-elements"] = RunRecord{
		ID: "run-2", Status: "failed", Terminal: true, FinishedAt: h.now.Add(time.Minute),
		Failures: []DestinationFailure{{DestinationID: "dest-1", Code: "repository_invalid"}},
	}
	h.now = h.now.Add(time.Hour)
	h.reconcile()
	decisions := h.decisions()
	if len(decisions) != 2 {
		t.Fatalf("decision events = %d, want the original and one new question", len(decisions))
	}
	second := h.openIncident()
	if len(second.Diagnosis.Unreadable) != 1 || second.Diagnosis.Unreadable[0].RelPath != tempC || len(second.Diagnosis.Accepted) != 2 {
		t.Fatalf("new question covers %+v (accepted %d)", second.Diagnosis.Unreadable, len(second.Diagnosis.Accepted))
	}
}

func TestBackupFailureAcceptRechecksEachPathBeforeRecordingIt(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	h.reconcile()
	inc := h.openIncident()
	h.fs.heal(tempA)
	h.apply(h.resolution(inc, repohealth.OptAccept))
	got := h.exceptions()
	if len(got) != 1 || got[0].RelPath != tempB {
		t.Fatalf("exceptions = %+v, want only the path that is still unreadable", got)
	}

	h2 := newBackupHarness(t, tempA)
	h2.reconcile()
	inc2 := h2.openIncident()
	h2.fs.fail[filepath.Join(h2.fs.root, filepath.FromSlash(tempA))] = fs.ErrPermission
	h2.apply(h2.resolution(inc2, repohealth.OptAccept))
	if len(h2.exceptions()) != 0 || len(h2.history.triggered) != 0 {
		t.Fatalf("a path whose failure changed kind was excepted anyway: %+v", h2.exceptions())
	}
	if h2.openIncident().State != repohealth.IncidentFailed {
		t.Fatal("nothing could be applied, so the incident must stay open as failed and be asked again")
	}
}

func TestBackupFailureRepairDelegatesToTheVolumeOwnerAndRefusesWhileABackupRuns(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()

	h.history.active = true
	h.apply(h.resolution(inc, repohealth.OptRepair))
	if h.remed.calls != 0 || len(h.history.triggered) != 0 {
		t.Fatalf("the drive was taken offline while a backup ran: calls=%d", h.remed.calls)
	}
	if got := h.openIncident(); got.State != repohealth.IncidentFailed {
		t.Fatalf("state = %s", got.State)
	}

	h = newBackupHarness(t, tempA)
	h.reconcile()
	inc = h.openIncident()
	h.apply(h.resolution(inc, repohealth.OptRepair))
	var actions []destinationreadiness.PreparationAction
	for _, p := range h.remed.plans {
		actions = append(actions, p.Action)
	}
	want := []destinationreadiness.PreparationAction{destinationreadiness.ActionUnmount, destinationreadiness.ActionRepairFilesystem, destinationreadiness.ActionMountReadWrite}
	if fmt.Sprint(actions) != fmt.Sprint(want) {
		t.Fatalf("host actions = %v, want %v through the control-plane owner", actions, want)
	}
	if got := h.openIncident(); got.State != repohealth.IncidentApplied || got.RepairJournalID == "" {
		t.Fatalf("incident = %+v", got)
	}
	if len(h.history.triggered) != 1 || len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatalf("after a verified repair: runs=%v notices=%d", h.history.triggered, len(h.notices(categoryBackupRecovered)))
	}
}

func TestBackupFailureRepairIsRefusedWithoutAVerifiedHuman(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.apply(h.resolution(h.openIncident(), repohealth.OptRepair, func(r *operatordecision.DecisionResolution) { r.AnsweredByIdentityKind = "agent" }))
	if h.remed.calls != 0 {
		t.Fatal("the drive was repaired on an unverified answer")
	}
}

func TestBackupFailureFreshSetCallbackPersistsFailClosedMigrationHandoff(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, repohealth.OptFresh))
	if h.remed.calls != 0 || len(h.exceptions()) != 0 || len(h.history.triggered) != 0 {
		t.Fatal("migration must not repair, quarantine, accept exceptions, or trigger the old plan before a new destination is selected")
	}
	got := h.openIncident()
	if got.State != repohealth.IncidentAgentRequested || got.Answer != repohealth.OptFresh || got.AnsweredByIdentityKind != "verified_human" {
		t.Fatalf("state = %s", got.State)
	}
	if got.FreshMigrationStage != "awaiting_destination_selection" || !strings.Contains(got.FreshMigrationDetail, "capacity proof") {
		t.Fatalf("migration progress was not durably held on missing operator inputs: %+v", got)
	}
	if got.FreshMigrationID != repohealth.FreshMigrationID(got.ID, got.Generation) {
		t.Fatalf("migration journal id = %q, want stable incident-generation binding", got.FreshMigrationID)
	}
	notes := h.notices(categoryBackupNeedsAttention)
	if len(notes) != 1 || !strings.Contains(notes[0].Payload["body"].(string), "operator-selected destination") || !strings.Contains(notes[0].Payload["body"].(string), "old repository remains preserved") {
		t.Fatalf("the result notice must name missing migration inputs and preservation guarantee: %+v", notes)
	}
}

func TestBackupFailureFreshHandoffReplaysNoticeAfterIncidentCommit(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	res := h.resolution(inc, repohealth.OptFresh)
	res.AnsweredAt = h.now.UTC().Format(time.RFC3339Nano)
	if err := h.flow.beginApplying(context.Background(), &inc, res); err != nil {
		t.Fatal(err)
	}
	if err := inc.Transition(repohealth.IncidentAgentRequested); err != nil {
		t.Fatal(err)
	}
	inc.FreshMigrationStage = "awaiting_destination_selection"
	inc.FreshMigrationID = repohealth.FreshMigrationID(inc.ID, inc.Generation)
	inc.FreshMigrationDetail = "No destination inputs supplied."
	if err := h.incidents.Put(context.Background(), inc); err != nil {
		t.Fatal(err)
	}
	// Model a process exit after the incident journal write but before the
	// Notification Hub ask/result receipt was committed.
	h.apply(res)
	if len(h.history.triggered) != 0 || h.remed.calls != 0 {
		t.Fatal("handoff replay must publish its result without beginning migration effects")
	}
	if got := h.openIncident(); got.FreshMigrationStage != "awaiting_destination_selection" {
		t.Fatalf("replay lost durable migration progress: %+v", got)
	}
	if notes := h.notices(categoryBackupNeedsAttention); len(notes) != 1 {
		t.Fatalf("result notice count = %d, want exactly one", len(notes))
	}
}

func TestBackupFailureFreshSetRefusesUnverifiedCallback(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, repohealth.OptFresh, func(r *operatordecision.DecisionResolution) { r.AnsweredByIdentityKind = "agent" }))
	if got := h.openIncident(); got.State != repohealth.IncidentDeclined || got.Answer != repohealth.OptFresh {
		t.Fatalf("unverified callback changed incident: %+v", got)
	}
	if len(h.notices(categoryBackupNeedsAttention)) != 1 || h.remed.calls != 0 || len(h.history.triggered) != 0 {
		t.Fatal("unverified callback must be rejected without migration or repair effects")
	}
}

func TestBackupFailureDeclineKeepsTheProblemInFrontOfTheOperator(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.apply(h.resolution(h.openIncident(), repohealth.OptLeave))
	if len(h.exceptions()) != 0 || len(h.history.triggered) != 0 {
		t.Fatal("declining changed state")
	}
	h.history.runs["plan-elements"] = RunRecord{
		ID: "run-2", Status: "failed", Terminal: true, FinishedAt: h.now,
		Failures: []DestinationFailure{{DestinationID: "dest-1", Code: "repository_invalid"}},
	}

	h.now = h.now.Add(23 * time.Hour)
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatalf("asked again after only 23 hours: %d", len(h.decisions()))
	}
	h.now = h.now.Add(2 * time.Hour)
	h.reconcile()
	if len(h.decisions()) != 2 {
		t.Fatalf("decision events = %d, want the question back after a day", len(h.decisions()))
	}
	if got := h.openIncident(); got.Generation != 2 || got.State != repohealth.IncidentAwaiting {
		t.Fatalf("incident = %+v", got)
	}
}

func TestBackupFailureClosesTheIncidentWhenBackupsSucceedAgain(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.now = h.now.Add(time.Hour)
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now}
	h.reconcile()
	all, _ := h.incidents.List(context.Background())
	if len(all) != 1 || all[0].State != repohealth.IncidentResolved {
		t.Fatalf("incidents = %+v, want the one resolved", all)
	}
	if len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatal("the operator must hear that backups work again")
	}
	// An answer that arrives late is settled without changing anything.
	h.apply(h.resolution(all[0], repohealth.OptRepair))
	if h.remed.calls != 0 {
		t.Fatal("a late answer repaired a drive for a problem that no longer exists")
	}
}

func TestBackupFailureNeverQuestionsAnUnmountedDriveOrAClearedFailure(t *testing.T) {
	h := newBackupHarness(t, tempA)
	*h.mounted = false
	h.reconcile()
	if len(h.decisions()) != 0 {
		t.Fatalf("an unmounted drive belongs to the drive recovery flow, got %d backup questions", len(h.decisions()))
	}

	h = newBackupHarness(t) // nothing unreadable: every read-only check passes
	h.reconcile()
	if len(h.decisions()) != 0 || len(h.history.triggered) != 1 {
		t.Fatalf("a failure that cleared must be retried, not asked about: decisions=%d runs=%v", len(h.decisions()), h.history.triggered)
	}
}

func TestBackupFailureContentLossIsNeverOfferedForSkipping(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.fs.breakEntry(blobQ)
	h.reconcile()
	inc := h.openIncident()
	if inc.Diagnosis.Cause != repohealth.CauseContentUnreadable || inc.Offers.Accept {
		t.Fatalf("cause=%s offers=%+v, want content unreadable with no skip option", inc.Diagnosis.Cause, inc.Offers)
	}
	// Even a forged answer for the skip option records nothing.
	h.apply(h.resolution(inc, repohealth.OptAccept))
	if len(h.exceptions()) != 0 {
		t.Fatal("an exception was recorded for unreadable repository content")
	}
}

func TestDB4RepositoryDamageChoicesUseOneVerifiedDecisionClassForEIOAndMissingBlob(t *testing.T) {
	cases := []struct {
		name, unreadable string
		cause            repohealth.Cause
		wantQuarantine   bool
	}{
		{name: "unreadable orphan temp with EIO", unreadable: tempA, cause: repohealth.CauseLeftoverTempUnreadable, wantQuarantine: true},
		{name: "unreadable suffixless orphan temp with EIO", unreadable: "s/f43/abc.f.tmp", cause: repohealth.CauseLeftoverTempUnreadable, wantQuarantine: true},
		{name: "missing committed blob with EIO", unreadable: blobQ, cause: repohealth.CauseContentUnreadable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newBackupHarness(t, tc.unreadable)
			h.reconcile()
			inc := h.openIncident()
			if inc.Diagnosis.Cause != tc.cause {
				t.Fatalf("diagnosis cause = %s, want %s", inc.Diagnosis.Cause, tc.cause)
			}
			if !inc.Offers.Repair || !inc.Offers.Fresh || inc.Offers.KopiaQuarantine != tc.wantQuarantine {
				t.Fatalf("repository repair options = %+v", inc.Offers)
			}
			decisionEvents := h.decisions()
			if len(decisionEvents) != 1 {
				t.Fatalf("decision count = %d, want one", len(decisionEvents))
			}
			keys := map[string]bool{}
			options, ok := decisionEvents[0].Payload["options"].([]any)
			if !ok {
				t.Fatalf("decision options have type %T", decisionEvents[0].Payload["options"])
			}
			for _, raw := range options {
				option, ok := raw.(map[string]any)
				if !ok {
					t.Fatalf("decision option has type %T", raw)
				}
				keys[fmt.Sprint(option["key"])] = true
			}
			for _, key := range []string{repohealth.OptRepair, repohealth.OptFresh} {
				if !keys[key] {
					t.Errorf("one decision omitted %q: %v", key, keys)
				}
			}
			if keys[repohealth.OptKopiaQuarantine] != tc.wantQuarantine {
				t.Fatalf("quarantine option=%t, want %t", keys[repohealth.OptKopiaQuarantine], tc.wantQuarantine)
			}
			// The fresh-set route is deliberately a no-write operator decision.
			// Exercise the same signed, verified-human callback for both damage
			// classes, then verify one result notice and no filesystem/repo repair.
			h.apply(h.resolution(inc, repohealth.OptFresh))
			got, ok, err := h.incidents.Get(context.Background(), inc.ID)
			if err != nil || !ok || got.State != repohealth.IncidentAgentRequested {
				t.Fatalf("recorded outcome=%+v ok=%t err=%v", got, ok, err)
			}
			if h.remed.calls != 0 || len(h.repoRepair.calls) != 0 || len(h.notices(categoryBackupNeedsAttention)) != 1 {
				t.Fatalf("fresh-set callback effects: volume=%v repo=%v notices=%d", h.remed.calls, h.repoRepair.calls, len(h.notices(categoryBackupNeedsAttention)))
			}
		})
	}
}

func TestDB4SuffixlessTempUsesVerifiedQuarantineCallback(t *testing.T) {
	const temp = "s/f43/abc.f.tmp"
	h := newBackupHarness(t, temp)
	h.reconcile()
	inc := h.openIncident()
	if !inc.Offers.KopiaQuarantine || len(inc.Diagnosis.Unreadable) != 1 || inc.Diagnosis.Unreadable[0].RelPath != temp || !inc.Diagnosis.Unreadable[0].Orphan {
		t.Fatalf("suffixless exact temp was not offered for quarantine: %+v", inc)
	}
	h.apply(h.resolution(inc, repohealth.OptKopiaQuarantine))
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplied {
		t.Fatalf("suffixless quarantine incident = %+v ok=%t err=%v", got, ok, err)
	}
	if len(h.repoRepair.calls) != 1 || h.repoRepair.calls[0] != "elements-local:"+temp || len(h.history.triggered) != 1 || len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatalf("suffixless quarantine effects calls=%v retries=%v notices=%d", h.repoRepair.calls, h.history.triggered, len(h.notices(categoryBackupRecovered)))
	}
}

func TestDB4QuarantineSetResumesIdempotentlyAfterPartialApplyAndRestart(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	h.repoRepair.failAt = 2 // tempB is committed first; tempA interrupts the first callback.
	h.reconcile()
	inc := h.openIncident()
	if !inc.Offers.KopiaQuarantine || len(inc.Diagnosis.Unreadable) != 2 {
		t.Fatalf("complete exact orphan set was not offered: %+v", inc.Diagnosis)
	}
	h.apply(h.resolution(inc, repohealth.OptKopiaQuarantine))
	partial, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || partial.State != repohealth.IncidentApplying || len(partial.QuarantinedPaths) != 1 || partial.QuarantinedPaths[0] != tempB {
		t.Fatalf("partial quarantine receipt=%+v ok=%t err=%v", partial, ok, err)
	}
	h.restartFailureFlow()
	h.upkeep()
	got, ok, err := h.incidents.Get(context.Background(), inc.ID)
	if err != nil || !ok || got.State != repohealth.IncidentApplied || len(got.QuarantinedPaths) != 2 {
		t.Fatalf("resumed quarantine incident=%+v ok=%t err=%v", got, ok, err)
	}
	if len(h.repoRepair.calls) != 4 || len(h.history.triggered) != 1 || len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatalf("quarantine replay calls=%v retries=%v success notices=%d", h.repoRepair.calls, h.history.triggered, len(h.notices(categoryBackupRecovered)))
	}
	h.upkeep()
	if len(h.repoRepair.calls) != 4 || len(h.history.triggered) != 1 {
		t.Fatalf("repeated upkeep replayed completed work: calls=%v retries=%v", h.repoRepair.calls, h.history.triggered)
	}
}

func TestStandingQBlobApprovalRestoresOnlyAnExactOwnerVerifiedCandidate(t *testing.T) {
	h := newBackupHarness(t)
	h.kopia.listErr = fmt.Errorf(`BLOB not found: blob "q1234"`)
	setStandingQBlobCandidate(h)
	h.reconcile()
	if h.kopia.cacheProbeCalls != 1 || h.kopia.blobCommitCalls != 1 {
		t.Fatalf("probe/commit = %d/%d", h.kopia.cacheProbeCalls, h.kopia.blobCommitCalls)
	}
	if len(h.decisions()) != 0 {
		t.Fatal("standing q-blob authorization incorrectly raised a new decision")
	}
	if len(h.history.triggered) != 1 || h.history.triggered[0] != "plan-elements" {
		t.Fatalf("triggered plans = %v", h.history.triggered)
	}
	var recoveryNotices int
	for _, event := range h.publisher.events {
		if strings.HasPrefix(fmt.Sprint(event.Payload["dedupe_key"]), "backup-qblob-restored:") || strings.Contains(fmt.Sprint(event.Payload["title"]), "Orphan repository blob restored") {
			recoveryNotices++
		}
	}
	if recoveryNotices != 1 {
		t.Fatalf("q-blob result notices = %d, want one", recoveryNotices)
	}
}

func setStandingQBlobCandidate(h *backupHarness) {
	const final = "q/1234.f"
	const temp = final + ".tmp.abcd1234"
	h.kopia.cacheCandidate = engine.OrphanBlobCandidate{
		BlobID: "q1234", CandidatePath: filepath.Join(h.dest.RepositoryLocation, filepath.FromSlash(temp)), CandidateBytes: 17,
		CandidateSHA256: strings.Repeat("a", 64), FinalBlobAbsent: true, SourceRepositoryRead: true,
		OverlaySnapshotStatus: "snapshot_list_passed",
		Blob:                  &engine.RepoBlobInspection{BlobID: "q1234", RelativePath: final, State: "missing", OrphanTemp: true, AdjacentTemps: []string{temp}},
	}
}

func TestStandingQBlobRecoveryKeepsResultNoticePendingUntilDelivered(t *testing.T) {
	h := newBackupHarness(t)
	h.kopia.listErr = fmt.Errorf(`BLOB not found: blob "q1234"`)
	setStandingQBlobCandidate(h)
	h.publisher.failType, h.publisher.failTypeTimes = backupNotificationType, 1
	h.reconcile()
	pending, err := h.worker.notices.Pending()
	if err != nil || len(pending) != 1 || pending[0].Key != "qblob-restored:dest-1:q1234:run-1" {
		t.Fatalf("pending result notices=%+v err=%v", pending, err)
	}
	stored, err := os.ReadFile(h.worker.notices.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "/private/transient/candidate") || strings.Contains(string(stored), strings.Repeat("a", 64)) {
		t.Fatal("pending result state leaked candidate path or digest")
	}
	if len(h.decisions()) != 0 || len(h.history.triggered) != 1 {
		t.Fatalf("recovery raised a decision or skipped retry: decisions=%d retries=%v", len(h.decisions()), h.history.triggered)
	}
	h.worker.publishPendingResultNotices(context.Background())
	pending, err = h.worker.notices.Pending()
	if err != nil || len(pending) != 0 {
		t.Fatalf("result notice was not cleared after delivery: pending=%+v err=%v", pending, err)
	}
	var count int
	for _, event := range h.publisher.events {
		if strings.Contains(fmt.Sprint(event.Payload["title"]), "Orphan repository blob restored") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("delivered q-blob result notice count=%d, want one", count)
	}
}

func TestStandingQBlobRecoveryDoesNotMutateWhenResultRouteIsDisabled(t *testing.T) {
	h := newBackupHarness(t)
	h.kopia.listErr = fmt.Errorf(`BLOB not found: blob "q1234"`)
	setStandingQBlobCandidate(h)
	h.worker.publisher = disabledRecoveryPublisher{h.publisher}
	h.reconcile()
	if h.kopia.blobCommitCalls != 0 {
		t.Fatalf("commit calls=%d while notification route is disabled", h.kopia.blobCommitCalls)
	}
	if len(h.decisions()) != 1 {
		t.Fatalf("fallback decision count=%d, want one", len(h.decisions()))
	}
}

func TestStandingQBlobApprovalFailsClosedWithoutOwnerCandidateProof(t *testing.T) {
	h := newBackupHarness(t)
	h.kopia.listErr = fmt.Errorf(`BLOB not found: blob "q1234"`)
	h.kopia.cacheCandidate = engine.OrphanBlobCandidate{BlobID: "q1234", CandidateBytes: 17, CandidateSHA256: strings.Repeat("a", 64), FinalBlobAbsent: true,
		SourceRepositoryRead: true, OverlaySnapshotStatus: "snapshot_list_failed"}
	h.reconcile()
	if h.kopia.blobCommitCalls != 0 {
		t.Fatalf("commit calls = %d, want 0", h.kopia.blobCommitCalls)
	}
	if len(h.decisions()) != 1 {
		t.Fatalf("decision count = %d, want one fallback decision", len(h.decisions()))
	}
}

func TestStandingQBlobRecoveryRejectsCandidateOutsideExactTypedTemp(t *testing.T) {
	for _, mutate := range []struct {
		name string
		edit func(*engine.OrphanBlobCandidate)
	}{
		{name: "outside repository", edit: func(c *engine.OrphanBlobCandidate) { c.CandidatePath = "/outside/q/1234.f.tmp.abcd1234" }},
		{name: "not in typed adjacent set", edit: func(c *engine.OrphanBlobCandidate) { c.Blob.AdjacentTemps = []string{"q/9999.f.tmp.abcd1234"} }},
		{name: "unreadable classification", edit: func(c *engine.OrphanBlobCandidate) { c.Blob.State = "unreadable" }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			h := newBackupHarness(t)
			h.kopia.listErr = fmt.Errorf(`BLOB not found: blob "q1234"`)
			setStandingQBlobCandidate(h)
			mutate.edit(&h.kopia.cacheCandidate)
			h.reconcile()
			if h.kopia.blobCommitCalls != 0 {
				t.Fatalf("unsafe candidate was restored %d times", h.kopia.blobCommitCalls)
			}
			if len(h.decisions()) != 1 {
				t.Fatalf("unsafe candidate did not fall back to one decision: %d", len(h.decisions()))
			}
		})
	}
}

func TestBackupFailureTellsTheOperatorOncePerDayWhileBackupsAreStalled(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	stalled := h.notices(repohealth.CategoryStalled)
	if len(stalled) != 1 {
		t.Fatalf("stalled notices = %d on the first sweep, want 1", len(stalled))
	}
	body := stalled[0].Payload["body"].(string)
	for _, want := range []string{"elements-daily-runtime", "82 days", "2 of its 3", "still waiting for your answer"} {
		if !strings.Contains(stalled[0].Payload["title"].(string)+" "+body, want) {
			t.Errorf("stalled notice omits %q: %s", want, body)
		}
	}
	h.now = h.now.Add(16 * time.Minute)
	h.reconcile()
	h.now = h.now.Add(10 * time.Hour)
	h.reconcile()
	if len(h.notices(repohealth.CategoryStalled)) != 1 {
		t.Fatal("repeated within a day")
	}
	h.now = h.now.Add(15 * time.Hour)
	h.reconcile()
	if len(h.notices(repohealth.CategoryStalled)) != 2 {
		t.Fatalf("stalled notices = %d, want a second one after a day", len(h.notices(repohealth.CategoryStalled)))
	}
}

func TestBackupFailureReportsASwitchedOffPlanThatProtectsSomethingNothingElseDoes(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.history.plans = append(h.history.plans, PlanRecord{
		ID: "plan-escrow", Name: "credential-store-escrow", Enabled: false, Schedule: "12h",
		TargetIDs: []string{"t-escrow"}, DestinationIDs: []string{"dest-1"}, CreatedAt: backupTestNow.Add(-100 * 24 * time.Hour),
	})
	h.history.labels["t-escrow"] = TargetLabel{Owner: "credential-store", Name: "escrow"}
	h.reconcile()
	off := h.notices(repohealth.CategoryDisabledUnprotec)
	if len(off) != 1 || !strings.Contains(off[0].Payload["body"].(string), "credential-store-escrow") {
		t.Fatalf("switched-off notices = %+v", off)
	}
	h.now = h.now.Add(48 * time.Hour)
	h.reconcile()
	if len(h.notices(repohealth.CategoryDisabledUnprotec)) != 1 {
		t.Fatal("a switched-off plan was reported more than once")
	}
}

func TestBackupFailureWarnsDailyWhileAnAcceptedExceptionStillApplies(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.apply(h.resolution(h.openIncident(), repohealth.OptAccept))
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now.Add(time.Minute)}
	h.now = h.now.Add(20 * time.Minute)
	h.reconcile()
	warnings := h.notices(repohealth.CategoryExceptionActive)
	if len(warnings) != 1 || !strings.Contains(warnings[0].Payload["title"].(string), "1 accepted unreadable file") {
		t.Fatalf("standing warnings = %+v", warnings)
	}
	h.now = h.now.Add(25 * time.Hour)
	h.reconcile()
	if len(h.notices(repohealth.CategoryExceptionActive)) != 2 {
		t.Fatal("the standing warning must repeat daily while the exception applies")
	}
	h.fs.heal(tempA)
	h.now = h.now.Add(25 * time.Hour)
	h.reconcile()
	if len(h.notices(repohealth.CategoryExceptionActive)) != 2 {
		t.Fatal("the warning continued although the file is readable again")
	}
}

func TestBackupFailureAndVolumeFlowsClaimDifferentDecisionKinds(t *testing.T) {
	h := newBackupHarness(t, tempA)
	backupKind := operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind)
	volumeKind := operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind)
	if !h.flow.Handles(backupKind) || h.flow.Handles(volumeKind) || h.volume.Handles(backupKind) || !h.volume.Handles(volumeKind) {
		t.Fatal("each flow must claim only its own decision kind, so an answer reaches exactly one")
	}
	if handled, err := h.flow.Apply(context.Background(), operatordecision.DecisionResolution{Correlation: map[string]string{"recovery_id": "x"}}); handled || err != nil {
		t.Fatalf("a volume-recovery resolution was claimed by the backup flow: %v %v", handled, err)
	}
}

func TestBackupFailureNeverWritesToTheRepository(t *testing.T) {
	h := newBackupHarness(t, tempA, tempB)
	before := snapshotTree(t, h.dest.RepositoryLocation)
	h.reconcile()
	h.apply(h.resolution(h.openIncident(), repohealth.OptAccept))
	h.now = h.now.Add(time.Hour)
	h.reconcile()
	if after := snapshotTree(t, h.dest.RepositoryLocation); after != before {
		t.Fatalf("the repository directory changed:\nbefore: %s\nafter:  %s", before, after)
	}
}

func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s %d %s", strings.TrimPrefix(p, root), info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

func TestBackupFailureHoldsNoticesBackForAGraceAfterStartButNotTheQuestion(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.worker.noticesNotBefore = h.now.Add(backupNoticeStartupGrace)
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatal("the question must not wait for the notice grace")
	}
	if len(h.notices(repohealth.CategoryStalled)) != 0 {
		t.Fatal("a notice was sent inside the startup grace")
	}
	h.now = h.now.Add(backupNoticeStartupGrace + time.Minute)
	h.reconcile()
	if len(h.notices(repohealth.CategoryStalled)) != 1 {
		t.Fatal("the notice must be sent once the grace ends")
	}
}

func TestBackupFailureBundlesLongIdlePlansIntoOneDigest(t *testing.T) {
	h := newBackupHarness(t, tempA)
	for _, name := range []string{"manual-a", "manual-b"} {
		id := "plan-" + name
		h.history.plans = append(h.history.plans, PlanRecord{
			ID: id, Name: name, Enabled: true, TargetIDs: []string{"t-" + name}, DestinationIDs: []string{"dest-other"}, CreatedAt: backupTestNow.Add(-90 * 24 * time.Hour),
		})
		h.history.success[id] = PlanSuccess{At: backupTestNow.Add(-10 * 24 * time.Hour), ByTarget: map[string]time.Time{"t-" + name: backupTestNow.Add(-10 * 24 * time.Hour)}}
		h.history.labels["t-"+name] = TargetLabel{Owner: "o", Name: name}
	}
	h.reconcile()
	notices := h.notices(repohealth.CategoryStalled)
	if len(notices) != 2 {
		t.Fatalf("stalled notices = %d, want one for the plan with a question and one digest", len(notices))
	}
	var digest string
	for _, n := range notices {
		if strings.Contains(n.Payload["title"].(string), "2 backup plans") {
			digest = n.Payload["body"].(string)
		}
	}
	if !strings.Contains(digest, "manual-a: 10 days") || !strings.Contains(digest, "manual-b: 10 days") {
		t.Fatalf("digest = %q", digest)
	}
}

func TestBackupFailureHoldsTheQuestionOpenForTheWholeWindowWithAStableDeadline(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.publisher.failNext = 2
	h.reconcile() // the hub refuses; the question is retried below with identical facts
	h.now = h.now.Add(3 * time.Hour)
	h.reconcile()
	decisions := h.decisions()
	if len(decisions) != 1 {
		t.Fatalf("decision events = %d, want the question delivered once", len(decisions))
	}
	want := backupTestNow.Add(backupDecisionWindow).UTC().Format(time.RFC3339)
	if got := decisions[0].Payload["deadline"]; got != want {
		t.Fatalf("deadline = %v, want %s (the window from when the question was first asked, not from each retry)", got, want)
	}
}

func TestBackupFailureAsksAgainAtOnceWhenTheHubExpiresTheQuestionUnanswered(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, "", func(r *operatordecision.DecisionResolution) {
		r.State = operatordecision.StateExpired
		r.AnsweredBy, r.AnsweredByIdentityKind, r.AnsweredByIdentitySource = "", "", ""
	}))
	if len(h.exceptions()) != 0 || len(h.history.triggered) != 0 || h.remed.calls != 0 {
		t.Fatal("an expired question changed state")
	}
	if len(h.notices(categoryBackupNeedsAttention)) != 0 {
		t.Fatal("an expiry is not a refusal; no 'needs attention' notice is wanted")
	}
	if got := h.openIncident(); got.State != repohealth.IncidentLapsed {
		t.Fatalf("state = %s, want lapsed", got.State)
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 2 {
		t.Fatalf("decision events = %d, want a fresh question straight after the expiry", len(h.decisions()))
	}
	if got := h.openIncident(); got.Generation != 2 || got.State != repohealth.IncidentAwaiting {
		t.Fatalf("incident = %+v", got)
	}
}

func TestBackupFailureDoesNotAskAgainAfterAnExpiryOnceBackupsWork(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, "", func(r *operatordecision.DecisionResolution) { r.State = operatordecision.StateExpired }))
	h.now = h.now.Add(time.Hour)
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now}
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatalf("decision events = %d, want no new question once backups work", len(h.decisions()))
	}
}
