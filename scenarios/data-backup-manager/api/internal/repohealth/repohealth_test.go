package repohealth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/engine"

	"github.com/vrooli/api-core/operatordecision"
)

func TestClassifyNameSeparatesLeftoverTempFromRepositoryContent(t *testing.T) {
	cases := []struct {
		rel   string
		class EntryClass
		kind  BlobKind
	}{
		{tempRel1, ClassLeftoverTemp, BlobSessionMarker},
		{"p/bbb/7098cafebeccc21a63be88c33846b-s9e453c5eb992a889141.f.tmp.7bebcde20343b56d", ClassLeftoverTemp, BlobDataPack},
		{blobRel, ClassRepositoryBlob, BlobMetadataPack},
		{readable, ClassRepositoryBlob, BlobDataPack},
		{"xe40.f", ClassRepositoryBlob, BlobOther},
		{"kopia.repository.f", ClassOther, BlobOther},
		{"s/f43", ClassOther, BlobSessionMarker},
		{"s/f43/abc.f.tmp.not-hex-zz", ClassOther, BlobSessionMarker},
		{"s/f43/abc.f.tmp.8c9d", ClassOther, BlobSessionMarker},
	}
	for _, tc := range cases {
		class, kind := ClassifyName(tc.rel)
		if class != tc.class || kind != tc.kind {
			t.Errorf("ClassifyName(%q) = %s/%s, want %s/%s", tc.rel, class, kind, tc.class, tc.kind)
		}
	}
}

func TestProblemForRecognisesDriveReadErrors(t *testing.T) {
	if ProblemFor(errEIO) != ProblemIO || ProblemFor(fmt.Errorf("lstat x: input/output error")) != ProblemIO {
		t.Fatal("EIO must classify as a drive read error")
	}
	if ProblemFor(fs.ErrPermission) != ProblemPermission || ProblemFor(errors.New("boom")) != ProblemOther {
		t.Fatal("permission and unknown errors must classify separately")
	}
}

func TestScanSumsReadableBytesAndCollectsEveryUnreadableEntry(t *testing.T) {
	f := newFakeFS(readable, blobRel)
	f.addUnreadable(tempRel1, errEIO)
	f.addUnreadable(tempRel2, errEIO)
	f.addFile("p/bbb/7098cafebeccc21a63be88c33846b-s9e453c5eb992a889141.f.tmp.7bebcde20343b56d", 50)

	scan, err := (&Scanner{FS: f, Workers: 3, Now: func() time.Time { return fixedNow }}).Walk(context.Background(), root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if scan.Files != 3 || scan.Bytes != 250 || scan.TempFiles != 1 {
		t.Fatalf("files/bytes/temp = %d/%d/%d, want 3/250/1", scan.Files, scan.Bytes, scan.TempFiles)
	}
	if len(scan.Unreadable) != 2 {
		t.Fatalf("unreadable = %+v, want both siblings reported (kopia would stop at the first)", scan.Unreadable)
	}
	for _, e := range scan.Unreadable {
		if e.Class != ClassLeftoverTemp || e.Problem != ProblemIO || e.BlobKind != BlobSessionMarker {
			t.Errorf("entry %+v misclassified", e)
		}
	}
}

func TestScanTreatsAnUnlistableDirectoryAsUnreadableContent(t *testing.T) {
	f := newFakeFS(readable)
	f.set("q", fakeNode{dir: true, err: errEIO})
	scan, err := (&Scanner{FS: f}).Walk(context.Background(), root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(scan.Unreadable) != 1 || scan.Unreadable[0].RelPath != "q" || scan.Unreadable[0].Class != ClassOther {
		t.Fatalf("an unlistable directory hides everything below it and must be reported: %+v", scan.Unreadable)
	}
}

func TestScanFailsWhenTheRepositoryRootCannotBeListed(t *testing.T) {
	f := newFakeFS(readable)
	f.set(".", fakeNode{dir: true, err: errEIO})
	f.nodes[root] = fakeNode{dir: true, err: errEIO}
	if _, err := (&Scanner{FS: f}).Walk(context.Background(), root); err == nil {
		t.Fatal("a repository whose root cannot be listed must not produce a clean scan")
	}
}

func TestExceptionValidationRefusesAnythingButOneExactLeftoverTemp(t *testing.T) {
	base := approved(tempRel1)
	if err := base.Validate(); err != nil {
		t.Fatalf("a valid exception was refused: %v", err)
	}
	mutations := map[string]func(*Exception){
		"committed blob":     func(e *Exception) { e.RelPath, e.Class = blobRel, ClassRepositoryBlob },
		"other class":        func(e *Exception) { e.Class = ClassOther },
		"glob":               func(e *Exception) { e.RelPath = "s/*/*.f.tmp.8c9d63ca28444838" },
		"traversal":          func(e *Exception) { e.RelPath = "../" + tempRel1 },
		"absolute":           func(e *Exception) { e.RelPath = "/" + tempRel1 },
		"directory":          func(e *Exception) { e.RelPath = "s/f43" },
		"no approver":        func(e *Exception) { e.ApprovedBy = "" },
		"no identity source": func(e *Exception) { e.ApprovedBySource = "" },
		"no time":            func(e *Exception) { e.ApprovedAt = time.Time{} },
		"no failure kind":    func(e *Exception) { e.Problem = "" },
		"no destination":     func(e *Exception) { e.DestinationID = "" },
	}
	for name, mutate := range mutations {
		e := base
		mutate(&e)
		if err := e.Validate(); err == nil {
			t.Errorf("%s: exception was accepted but must be refused", name)
		}
	}
}

func TestFileExceptionStorePersistsAndKeepsTheFirstApproval(t *testing.T) {
	store := &FileExceptionStore{Root: filepath.Join(t.TempDir(), "exceptions")}
	ctx := context.Background()
	first := approved(tempRel1)
	if err := store.Add(ctx, first); err != nil {
		t.Fatalf("Add: %v", err)
	}
	second := approved(tempRel1)
	second.ApprovedBy = "someone-else"
	if err := store.Add(ctx, second); err != nil {
		t.Fatalf("re-Add: %v", err)
	}
	got, err := store.List(ctx, "dest-1")
	if err != nil || len(got) != 1 || got[0].ApprovedBy != "operator" || got[0].RelPath != tempRel1 {
		t.Fatalf("List = %+v, %v; want the first approval kept exactly once", got, err)
	}
	if other, _ := store.List(ctx, "dest-2"); len(other) != 0 {
		t.Fatalf("an exception for one destination leaked into another: %+v", other)
	}
	bad := approved(blobRel)
	bad.Class = ClassRepositoryBlob
	if err := store.Add(ctx, bad); err == nil {
		t.Fatal("the store must refuse to record an exception for repository content")
	}
	if err := store.Revoke(ctx, "dest-1", tempRel1); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got, _ := store.List(ctx, "dest-1"); len(got) != 0 {
		t.Fatalf("exception survived revocation: %+v", got)
	}
}

func TestEvaluateHonoursAnExceptionOnlyForTheExactPathAndFailureKind(t *testing.T) {
	f := newFakeFS(readable)
	exceptions := []Exception{approved(tempRel1)}

	unreadable := []Entry{NewEntry(tempRel1, errEIO), NewEntry(tempRel2, errEIO)}
	v := Evaluate(f, root, unreadable, exceptions)
	if len(v.Excused) != 1 || v.Excused[0].RelPath != tempRel1 {
		t.Fatalf("excused = %+v, want only the approved path", v.Excused)
	}
	if len(v.Blocking) != 1 || v.Blocking[0].RelPath != tempRel2 {
		t.Fatalf("blocking = %+v, want the sibling that was never approved", v.Blocking)
	}

	denied := []Entry{NewEntry(tempRel1, fs.ErrPermission)}
	if v := Evaluate(f, root, denied, exceptions); len(v.Blocking) != 1 {
		t.Fatalf("a different kind of failure on the approved path must block again: %+v", v)
	}
}

func TestEvaluateBlocksWhenAnApprovedPathChangesShape(t *testing.T) {
	f := newFakeFS(readable)
	f.set(tempRel1, fakeNode{dir: true})
	v := Evaluate(f, root, nil, []Exception{approved(tempRel1)})
	if len(v.Blocking) != 1 || !strings.Contains(v.Blocking[0].Detail, "directory") {
		t.Fatalf("an approved path that became a directory must block: %+v", v)
	}
}

func TestEvaluateReportsAnExceptionDormantOnceTheEntryIsGoneOrReadable(t *testing.T) {
	f := newFakeFS(readable)
	v := Evaluate(f, root, nil, []Exception{approved(tempRel1)})
	if len(v.Dormant) != 1 || len(v.Blocking) != 0 {
		t.Fatalf("a vanished path is dormant, not a blocker: %+v", v)
	}
	f.addFile(tempRel1, 10)
	if v := Evaluate(f, root, nil, []Exception{approved(tempRel1)}); len(v.Dormant) != 1 || len(v.Blocking) != 0 {
		t.Fatalf("a path that became readable is dormant: %+v", v)
	}
}

func usageEngineFor(f *fakeFS, inner *fakeEngine, ex *memExceptions) *UsageEngine {
	return &UsageEngine{
		KopiaEngine: inner,
		Locate: func(context.Context, string) (string, string, bool) {
			return "dest-1", root, true
		},
		Exceptions: ex,
		Scanner:    &Scanner{FS: f, Now: func() time.Time { return fixedNow }},
		Now:        func() time.Time { return fixedNow },
	}
}

func TestUsageEngineMeasuresDirectlyOnlyWhenEveryUnreadableEntryIsExcepted(t *testing.T) {
	f := newFakeFS(readable, blobRel)
	f.addUnreadable(tempRel1, errEIO)
	inner := &fakeEngine{statsErr: kopiaEIOError(tempRel1)}
	u := usageEngineFor(f, inner, &memExceptions{list: []Exception{approved(tempRel1)}})

	stats, err := u.RepoStats(context.Background(), "elements-local")
	if err != nil {
		t.Fatalf("RepoStats with every unreadable entry excepted: %v", err)
	}
	if stats.SizeBytes != 200 || stats.BlobCount != 2 {
		t.Fatalf("stats = %+v, want the readable bytes (200) over 2 files", stats)
	}
}

func TestUsageEngineKeepsTheErrorWhenAnotherEntryIsUnreadable(t *testing.T) {
	f := newFakeFS(readable)
	f.addUnreadable(tempRel1, errEIO)
	f.addUnreadable(tempRel2, errEIO)
	inner := &fakeEngine{statsErr: kopiaEIOError(tempRel1)}
	u := usageEngineFor(f, inner, &memExceptions{list: []Exception{approved(tempRel1)}})

	_, err := u.RepoStats(context.Background(), "elements-local")
	if text := joinedErr(t, err); !strings.Contains(text, "not covered by an approved exception") || !strings.Contains(text, "4e5/") {
		t.Fatalf("error = %q; the un-approved sibling must block and be named", text)
	}
}

func TestUsageEngineDoesNotUseAnExceptionForADifferentFailure(t *testing.T) {
	f := newFakeFS(readable)
	f.addUnreadable(tempRel1, errEIO)
	inner := &fakeEngine{statsErr: errors.New("error listing blobs: missing blob q6390189f3d3250c965c737fd82cc103e")}
	u := usageEngineFor(f, inner, &memExceptions{list: []Exception{approved(tempRel1)}})

	_, err := u.RepoStats(context.Background(), "elements-local")
	if text := joinedErr(t, err); !strings.Contains(text, "missing blob") {
		t.Fatalf("error = %q; a failure that does not name the excepted path must pass through", text)
	}
}

func TestUsageEnginePassesThroughWithoutExceptionsAndSkipsTheWalk(t *testing.T) {
	f := newFakeFS(readable)
	f.addUnreadable(tempRel1, errEIO)
	inner := &fakeEngine{statsErr: kopiaEIOError(tempRel1)}
	u := usageEngineFor(f, inner, &memExceptions{})
	if _, err := u.RepoStats(context.Background(), "elements-local"); err == nil {
		t.Fatal("without an exception the engine failure must stand")
	}
	if f.lists != 0 {
		t.Fatalf("walked the repository %d times with nothing to honour", f.lists)
	}

	ok := &fakeEngine{stats: engine.RepoStats{SizeBytes: 7}}
	if got, err := usageEngineFor(f, ok, &memExceptions{}).RepoStats(context.Background(), "x"); err != nil || got.SizeBytes != 7 {
		t.Fatalf("a healthy engine answer must be returned untouched: %+v %v", got, err)
	}
}

func TestUsageEngineReusesARecentWalk(t *testing.T) {
	f := newFakeFS(readable)
	f.addUnreadable(tempRel1, errEIO)
	inner := &fakeEngine{statsErr: kopiaEIOError(tempRel1)}
	u := usageEngineFor(f, inner, &memExceptions{list: []Exception{approved(tempRel1)}})
	for i := 0; i < 3; i++ {
		if _, err := u.RepoStats(context.Background(), "elements-local"); err != nil {
			t.Fatal(err)
		}
	}
	walkedOnce := f.lists
	f.lists = 0
	if _, err := u.RepoStats(context.Background(), "elements-local"); err != nil || f.lists != 0 {
		t.Fatalf("a fourth call inside the cache window walked again (%d listings, first pass %d)", f.lists, walkedOnce)
	}
	u.InvalidateScan(root)
	if _, err := u.RepoStats(context.Background(), "elements-local"); err != nil || f.lists == 0 {
		t.Fatal("invalidating the scan must force a fresh walk")
	}
}

func diagnoseInput(f *fakeFS, eng *fakeEngine, ex ExceptionStore) DiagnoseInput {
	return DiagnoseInput{
		Now: fixedNow, DestinationID: "dest-1", DestinationName: "elements-local", Location: "/media/Elements/vrooli-backups",
		RepositoryRoot: root,
		Volume: func(context.Context) (VolumeState, error) {
			return VolumeState{Known: true, Mounted: true, Label: "Elements"}, nil
		},
		Engine: eng, Scanner: &Scanner{FS: f, Now: func() time.Time { return fixedNow }}, Exceptions: ex,
		LastSuccess: func(context.Context, []engine.Snapshot) (time.Time, string) {
			return time.Date(2026, 8, 19, 18, 58, 23, 0, time.UTC), "repository snapshot"
		},
		Plans: func(context.Context) PlanSummary {
			return PlanSummary{TargetTotal: 14, Unprotected: 13, UnprotectedNames: []string{"codex/sessions", "codex/state"}}
		},
	}
}

func TestDiagnoseNamesTheCauseInOrderOfCost(t *testing.T) {
	snap := []engine.Snapshot{{StartTime: "2026-08-19T18:58:23Z"}, {StartTime: "2026-06-03T18:36:53Z"}}
	tests := []struct {
		name  string
		setup func(*fakeFS, *fakeEngine, *DiagnoseInput)
		want  Cause
	}{
		{"drive not mounted", func(_ *fakeFS, _ *fakeEngine, in *DiagnoseInput) {
			in.Volume = func(context.Context) (VolumeState, error) { return VolumeState{Known: true}, nil }
		}, CauseVolumeUnmounted},
		{"mounted read-only", func(_ *fakeFS, _ *fakeEngine, in *DiagnoseInput) {
			in.Volume = func(context.Context) (VolumeState, error) {
				return VolumeState{Known: true, Mounted: true, ReadOnly: true}, nil
			}
		}, CauseVolumeReadOnly},
		{"repository cannot be opened", func(_ *fakeFS, e *fakeEngine, _ *DiagnoseInput) { e.statusErr = errors.New("bad config") }, CauseRepositoryUnopenable},
		{"snapshots cannot be listed", func(_ *fakeFS, e *fakeEngine, _ *DiagnoseInput) { e.listErr = errors.New("missing manifest") }, CauseSnapshotsUnlistable},
		{"everything works now", func(_ *fakeFS, e *fakeEngine, _ *DiagnoseInput) { e.statsErr = nil }, CauseNone},
		{"only leftover temporary files unreadable", func(f *fakeFS, _ *fakeEngine, _ *DiagnoseInput) {
			f.addUnreadable(tempRel1, errEIO)
			f.addUnreadable(tempRel2, errEIO)
		}, CauseLeftoverTempUnreadable},
		{"a committed blob is unreadable too", func(f *fakeFS, _ *fakeEngine, _ *DiagnoseInput) {
			f.addUnreadable(tempRel1, errEIO)
			f.addUnreadable(blobRel, errEIO)
		}, CauseContentUnreadable},
		{"usage fails but nothing is unreadable", func(*fakeFS, *fakeEngine, *DiagnoseInput) {}, CauseUsageUnexplained},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeFS(readable, blobRel)
			eng := &fakeEngine{snaps: snap, statsErr: kopiaEIOError(tempRel1)}
			in := diagnoseInput(f, eng, &memExceptions{})
			tc.setup(f, eng, &in)
			got := Diagnose(context.Background(), in)
			if got.Cause != tc.want {
				t.Fatalf("cause = %s, want %s (%+v)", got.Cause, tc.want, got)
			}
			if got.LastSuccessSource == "" || got.DaysStopped() != 82 {
				t.Fatalf("days stopped = %d (%q), want 82 from the repository evidence", got.DaysStopped(), got.LastSuccessSource)
			}
		})
	}
}

func TestDiagnoseExcludesAcceptedEntriesFromTheQuestionedSet(t *testing.T) {
	f := newFakeFS(readable)
	f.addUnreadable(tempRel1, errEIO)
	f.addUnreadable(tempRel2, errEIO)
	eng := &fakeEngine{statsErr: kopiaEIOError(tempRel1)}
	d := Diagnose(context.Background(), diagnoseInput(f, eng, &memExceptions{list: []Exception{approved(tempRel1)}}))
	if len(d.Unreadable) != 1 || d.Unreadable[0].RelPath != tempRel2 || len(d.Accepted) != 1 {
		t.Fatalf("unreadable=%+v accepted=%+v; only the new entry may be asked about", d.Unreadable, d.Accepted)
	}
}

func TestSignatureChangesWithTheExactSetOfUnreadablePaths(t *testing.T) {
	mk := func(rels ...string) Diagnosis {
		d := Diagnosis{DestinationID: "dest-1", Cause: CauseLeftoverTempUnreadable}
		for _, r := range rels {
			d.Unreadable = append(d.Unreadable, NewEntry(r, errEIO))
		}
		return d
	}
	if mk(tempRel1, tempRel2).Signature() != mk(tempRel2, tempRel1).Signature() {
		t.Fatal("the same set in another order is the same condition")
	}
	if mk(tempRel1).Signature() == mk(tempRel1, tempRel2).Signature() {
		t.Fatal("a new unreadable path is a different condition and must raise a new question")
	}
}

var hexRun = regexp.MustCompile(`[0-9a-f]{12,}`)

func presentDiagnosis() Diagnosis {
	d := Diagnosis{
		CheckedAt: fixedNow, DestinationID: "7ebf6e6c-91f4-417e-bc49-7211e6dc53c4", DestinationName: "elements-local",
		RepositoryRoot: "/media/Elements/vrooli-backups/repositories/elements-local.kopia",
		Cause:          CauseLeftoverTempUnreadable, Volume: VolumeState{Known: true, Mounted: true, Label: "Elements"},
		RepositoryOpens: true, SnapshotsListed: true, SnapshotCount: 192,
		LastSuccessAt: time.Date(2026, 8, 19, 18, 58, 23, 0, time.UTC), LastSuccessSource: "repository snapshot",
		PriorRepairAt: time.Date(2026, 11, 4, 3, 0, 0, 0, time.UTC),
		Plans:         PlanSummary{TargetTotal: 14, Unprotected: 13, UnprotectedNames: []string{"codex/sessions", "codex/state", "claude-code/file-history"}},
	}
	for i := 0; i < 13; i++ {
		d.Unreadable = append(d.Unreadable, NewEntry(fmt.Sprintf("s/f%02d/64b7506cef44e4fbb3fc66ba7f2d%02d-sf40366ce03934d79144.f.tmp.8c9d63ca28444838", i, i), errEIO))
	}
	return d
}

func TestDecisionIsPlainAndWithinTheHubLimits(t *testing.T) {
	d := presentDiagnosis()
	facts := BuildFacts(d, OffersFor(d, true), "bf-incident", 1)
	if err := facts.Validate(); err != nil {
		t.Fatalf("hub would refuse the decision: %v", err)
	}
	if facts.Recommended != OptAccept {
		t.Fatalf("recommended = %s, want the skip option while evidence shows only leftover temp files", facts.Recommended)
	}
	visible := []string{facts.Subject, facts.Question, facts.RecommendationReason, facts.BigPicture}
	for _, o := range facts.Options {
		visible = append(visible, o.Label)
	}
	for _, text := range visible {
		if hexRun.MatchString(text) || strings.Contains(text, "7ebf6e6c") || strings.Contains(text, ".tmp.") || strings.Contains(text, "/media/") {
			t.Errorf("operator-facing text leaks an identifier or path: %q", text)
		}
	}
	for _, want := range []string{"82 days", "13 unfinished temporary files", "13 of the 14", "19 Aug"} {
		if !strings.Contains(facts.Subject+" "+facts.Question, want) {
			t.Errorf("the question omits %q:\n%s\n%s", want, facts.Subject, facts.Question)
		}
	}
	wantKeys := []string{OptAccept, OptRepair, OptFresh, OptLeave}
	for i, o := range facts.Options {
		letter := string(rune('A' + i))
		if o.Key != wantKeys[i] || !strings.HasPrefix(o.Label, letter+":") && !strings.HasPrefix(o.Label, letter+" (recommended):") {
			t.Errorf("option %d = %+v, want key %s lettered %s", i, o, wantKeys[i], letter)
		}
	}
	if !strings.Contains(facts.Options[1].Label, "4 Nov did not clear this") {
		t.Errorf("repair option must say the earlier repair did not clear it: %q", facts.Options[1].Label)
	}
	if !strings.Contains(facts.Options[2].Label, "an agent has to set it up") {
		t.Errorf("the fresh-set option must say an agent sets it up: %q", facts.Options[2].Label)
	}
	for _, rel := range []string{d.Unreadable[0].RelPath, d.RepositoryRoot, d.DestinationID} {
		if !strings.Contains(facts.TechnicalDetails, rel) {
			t.Errorf("technical details omit %q", rel)
		}
	}
	if facts.Reversible || facts.DefaultAnswer != "" {
		t.Fatal("this decision changes nothing by default and is not reversible")
	}
}

func TestDecisionOffersOnlyWhatTheCodeCanDo(t *testing.T) {
	d := presentDiagnosis()
	d.Cause = CauseContentUnreadable
	d.Unreadable = append(d.Unreadable, NewEntry(blobRel, errEIO))
	facts := BuildFacts(d, OffersFor(d, true), "i", 1)
	for _, o := range facts.Options {
		if o.Key == OptAccept {
			t.Fatal("unreadable repository content must never be offered for skipping")
		}
	}
	if facts.Recommended != OptRepair {
		t.Fatalf("recommended = %s, want repair when content is unreadable", facts.Recommended)
	}

	d.Cause = CauseLeftoverTempUnreadable
	noRepair := BuildFacts(d, OffersFor(d, false), "i", 1)
	for _, o := range noRepair.Options {
		if o.Key == OptRepair {
			t.Fatal("repair was offered although the control plane cannot plan it")
		}
	}
	if err := noRepair.Validate(); err != nil {
		t.Fatalf("decision without repair is invalid: %v", err)
	}
	if (Offers{}).Askable() || OffersFor(Diagnosis{Cause: CauseVolumeUnmounted}, true).Askable() || OffersFor(Diagnosis{Cause: CauseNone}, true).Askable() {
		t.Fatal("a cleared failure or an unmounted drive must not raise a second question")
	}
}

func TestDecisionStaysWithinLimitsForAHugeUnreadableSet(t *testing.T) {
	d := presentDiagnosis()
	for i := 0; i < 400; i++ {
		d.Unreadable = append(d.Unreadable, NewEntry(fmt.Sprintf("s/a%03d/64b7506cef44e4fbb3fc66ba7f2d%03d-sf40366ce03934d79144.f.tmp.8c9d63ca28444838", i, i), errEIO))
	}
	facts := BuildFacts(d, OffersFor(d, true), "i", 1)
	if err := facts.Validate(); err != nil {
		t.Fatalf("hub would refuse a decision with %d unreadable entries: %v", len(d.Unreadable), err)
	}
}

func TestStallThresholdIsTwiceTheScheduleButNeverUnderFortyEightHours(t *testing.T) {
	for schedule, want := range map[string]time.Duration{
		"": 48 * time.Hour, "12h": 48 * time.Hour, "24h": 48 * time.Hour, "36h": 72 * time.Hour,
		"168h": 336 * time.Hour, "0 3 * * *": 48 * time.Hour, "nonsense": 48 * time.Hour, "-5h": 48 * time.Hour,
	} {
		if got := StallThreshold(schedule); got != want {
			t.Errorf("StallThreshold(%q) = %s, want %s", schedule, got, want)
		}
	}
}

func TestStalledNoticeRepeatsOncePerDayWhileItLastsAndNamesTheOpenQuestion(t *testing.T) {
	plan := PlanStatus{
		PlanID: "p1", Name: "elements-daily-runtime", Enabled: true, Schedule: "24h",
		LastSuccessAt: fixedNow.Add(-82 * 24 * time.Hour), TargetCount: 14, Unprotected: 13,
		OpenDecision: "Backups to the Elements drive have been stopped for 82 days",
	}
	state := NoticeState{}
	first := EvaluateStalled(fixedNow, []PlanStatus{plan}, state)
	if len(first) != 1 {
		t.Fatalf("notices = %d, want 1", len(first))
	}
	n := first[0]
	if n.Category != CategoryStalled || n.Urgency != "high" {
		t.Fatalf("category/urgency = %s/%s", n.Category, n.Urgency)
	}
	for _, want := range []string{"elements-daily-runtime", "82 days", "13 of its 14", "still waiting for your answer"} {
		if !strings.Contains(n.Title+" "+n.Body, want) {
			t.Errorf("notice omits %q: %s / %s", want, n.Title, n.Body)
		}
	}
	state[n.Key] = fixedNow
	if again := EvaluateStalled(fixedNow.Add(23*time.Hour), []PlanStatus{plan}, state); len(again) != 0 {
		t.Fatal("repeated inside a day")
	}
	if next := EvaluateStalled(fixedNow.Add(24*time.Hour), []PlanStatus{plan}, state); len(next) != 1 {
		t.Fatal("not repeated after a day while the condition lasts")
	}
}

func TestStalledNoticeIgnoresHealthyDisabledAndYoungPlans(t *testing.T) {
	recent := PlanStatus{PlanID: "a", Name: "a", Enabled: true, Schedule: "24h", LastSuccessAt: fixedNow.Add(-47 * time.Hour)}
	disabled := PlanStatus{PlanID: "b", Name: "b", Enabled: false, LastSuccessAt: fixedNow.Add(-90 * 24 * time.Hour)}
	young := PlanStatus{PlanID: "c", Name: "c", Enabled: true, CreatedAt: fixedNow.Add(-time.Hour)}
	old := PlanStatus{PlanID: "d", Name: "d", Enabled: true, CreatedAt: fixedNow.Add(-5 * 24 * time.Hour)}
	got := EvaluateStalled(fixedNow, []PlanStatus{recent, disabled, young, old}, NoticeState{})
	if len(got) != 1 || !strings.Contains(got[0].Key, "d") || !strings.Contains(got[0].Body, "no successful backup on record") {
		t.Fatalf("notices = %+v, want only the plan that never succeeded and is older than the threshold", got)
	}
	slow := PlanStatus{PlanID: "e", Name: "e", Enabled: true, Schedule: "168h", LastSuccessAt: fixedNow.Add(-10 * 24 * time.Hour)}
	if got := EvaluateStalled(fixedNow, []PlanStatus{slow}, NoticeState{}); len(got) != 0 {
		t.Fatal("a weekly plan 10 days after success is within twice its schedule")
	}
}

func TestDisabledUnprotectedPlansAreReportedOnceUntilTheSetChanges(t *testing.T) {
	off := PlanStatus{PlanID: "p-off", Name: "credential-store-escrow", Enabled: false, TargetCount: 1, Unprotected: 1}
	covered := PlanStatus{PlanID: "p-covered", Name: "covered", Enabled: false, TargetCount: 2, Unprotected: 0}
	on := PlanStatus{PlanID: "p-on", Name: "on", Enabled: true, TargetCount: 1, Unprotected: 1}
	state := NoticeState{}
	n, ok := EvaluateDisabledUnprotected([]PlanStatus{off, covered, on}, state)
	if !ok || n.Category != CategoryDisabledUnprotec || !strings.Contains(n.Body, "credential-store-escrow") || strings.Contains(n.Body, "covered") {
		t.Fatalf("notice = %+v ok=%v", n, ok)
	}
	state[n.Key] = fixedNow
	if _, again := EvaluateDisabledUnprotected([]PlanStatus{off, covered, on}, state); again {
		t.Fatal("a once-only condition was repeated")
	}
	second := PlanStatus{PlanID: "p-off2", Name: "another", Enabled: false, TargetCount: 1, Unprotected: 1}
	if _, changed := EvaluateDisabledUnprotected([]PlanStatus{off, second}, state); !changed {
		t.Fatal("a changed set of switched-off plans must be reported")
	}
	if _, none := EvaluateDisabledUnprotected([]PlanStatus{covered, on}, NoticeState{}); none {
		t.Fatal("nothing is unprotected, so nothing is reported")
	}
}

func TestExceptionWarningRepeatsDailyAndNamesWhoAndWhen(t *testing.T) {
	active := []Exception{approved(tempRel1), approved(tempRel2)}
	state := NoticeState{}
	n, ok := EvaluateExceptionActive(fixedNow, "dest-1", "the Elements drive", active, state)
	if !ok || n.Category != CategoryExceptionActive || !strings.Contains(n.Title, "2 accepted unreadable files") {
		t.Fatalf("notice = %+v ok=%v", n, ok)
	}
	state[n.Key] = fixedNow
	if _, again := EvaluateExceptionActive(fixedNow.Add(time.Hour), "dest-1", "d", active, state); again {
		t.Fatal("standing warning repeated within a day")
	}
	if _, none := EvaluateExceptionActive(fixedNow, "dest-1", "d", nil, NoticeState{}); none {
		t.Fatal("no exceptions, no warning")
	}
}

func TestIncidentStoreRoundTripsAndListsInCreationOrder(t *testing.T) {
	store := &FileIncidentStore{Root: filepath.Join(t.TempDir(), "incidents")}
	ctx := context.Background()
	a := Incident{ID: "bf-a", DestinationID: "d", State: IncidentAwaiting, CreatedAt: fixedNow}
	b := Incident{ID: "bf-b", DestinationID: "d", State: IncidentResolved, CreatedAt: fixedNow.Add(-time.Hour)}
	for _, in := range []Incident{a, b} {
		if err := store.Put(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	got, ok, err := store.Get(ctx, "bf-a")
	if err != nil || !ok || got.State != IncidentAwaiting {
		t.Fatalf("Get = %+v %v %v", got, ok, err)
	}
	if _, ok, _ := store.Get(ctx, "missing"); ok {
		t.Fatal("unknown id reported present")
	}
	list, _ := store.List(ctx)
	if len(list) != 2 || list[0].ID != "bf-b" || !list[1].State.Open() || list[0].State.Open() {
		t.Fatalf("List = %+v", list)
	}
	if err := store.Put(ctx, Incident{ID: "../escape"}); err == nil {
		t.Fatal("an id that escapes the store was accepted")
	}
}

func TestNoticeStateSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "notices.json")
	if err := (&NoticeStateStore{Path: path}).Mark("stalled:p1", fixedNow); err != nil {
		t.Fatal(err)
	}
	state, err := (&NoticeStateStore{Path: path}).Load()
	if err != nil || !state["stalled:p1"].Equal(fixedNow) {
		t.Fatalf("state = %+v %v", state, err)
	}
	_ = operatordecision.MaxQuestion
}

func TestSeveralIdlePlansAreOneDigestWhilePlansWithAQuestionKeepTheirOwnNotice(t *testing.T) {
	idle := func(id string, days int) PlanStatus {
		return PlanStatus{PlanID: id, Name: "plan-" + id, Enabled: true, LastSuccessAt: fixedNow.Add(-time.Duration(days) * 24 * time.Hour), TargetCount: 2, Unprotected: 1}
	}
	asked := idle("elements", 82)
	asked.OpenDecision = "Backups to the Elements drive have been stopped for 82 days"
	state := NoticeState{}
	notices := BundleStalled(EvaluateStalled(fixedNow, []PlanStatus{idle("a", 5), idle("b", 32), asked}, state), fixedNow)
	if len(notices) != 2 {
		t.Fatalf("notices = %d, want the plan with a question plus one digest", len(notices))
	}
	var digest, own Notice
	for _, n := range notices {
		if len(n.Keys) > 0 {
			digest = n
		} else {
			own = n
		}
	}
	if !strings.Contains(own.Body, "still waiting for your answer") || strings.Contains(own.Title, "plan-a") {
		t.Fatalf("the plan with a question must keep its own notice: %+v", own)
	}
	if len(digest.Keys) != 2 || !strings.Contains(digest.Body, "plan-a: 5 days") || !strings.Contains(digest.Body, "plan-b: 32 days") {
		t.Fatalf("digest = %+v", digest)
	}
	if strings.Index(digest.Body, "plan-b") > strings.Index(digest.Body, "plan-a") {
		t.Fatal("the longest-idle plan must come first")
	}
	// Each plan's own daily cadence is tracked even though they were bundled.
	for _, key := range digest.CoveredKeys() {
		state[key] = fixedNow
	}
	if again := EvaluateStalled(fixedNow.Add(time.Hour), []PlanStatus{idle("a", 5), idle("b", 32)}, state); len(again) != 0 {
		t.Fatalf("bundled plans were notified again within a day: %+v", again)
	}
	if single := BundleStalled(EvaluateStalled(fixedNow, []PlanStatus{idle("a", 5)}, NoticeState{}), fixedNow); len(single) != 1 || len(single[0].Keys) != 0 {
		t.Fatal("a single idle plan is not bundled")
	}
}

func TestUnprotectedWordingIsGrammaticalForOneAllAndSomeItems(t *testing.T) {
	for _, tc := range []struct {
		unprotected, total int
		want               string
	}{
		{1, 1, "its only item has no other copy"},
		{2, 2, "all 2 of its items have no other copy"},
		{13, 14, "13 of its 14 items have no other copy"},
	} {
		if got := unprotectedPhrase(tc.unprotected, tc.total, "copy"); got != tc.want {
			t.Errorf("unprotectedPhrase(%d, %d) = %q, want %q", tc.unprotected, tc.total, got, tc.want)
		}
	}
}
