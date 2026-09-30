package pushsafety

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type memoryArtifacts struct {
	record       *Artifact
	saves        int
	failSave     int
	beginErr     error
	digestCalls  int
	failDigest   int
	changeDigest int
}

func (s *memoryArtifacts) Digest(context.Context, string) (string, error) {
	s.digestCalls++
	if s.digestCalls == s.failDigest {
		return "", errors.New("bundle read failed")
	}
	if s.digestCalls == s.changeDigest {
		return strings.Repeat("e", 64), nil
	}
	return strings.Repeat("d", 64), nil
}

func (s *memoryArtifacts) Begin(context.Context, string) (string, *Artifact, error) {
	return "/artifacts", s.record, s.beginErr
}

func (s *memoryArtifacts) Save(_ string, a Artifact) error {
	s.saves++
	if s.saves == s.failSave {
		return errors.New("disk full")
	}
	copy := a
	s.record = &copy
	return nil
}

type preparationCommands struct {
	calls int
	fail  int
}

func (c *preparationCommands) Run(ctx context.Context, dir string, in []byte, args ...string) ([]byte, error) {
	c.calls++
	if c.calls == c.fail {
		return nil, errors.New("injected command failure")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	switch args[0] {
	case "write-tree", "hash-object", "rev-parse":
		return []byte(headOID), nil
	case "cat-file":
		if args[1] == "commit" {
			return []byte("tree " + blobOID + "\nparent " + baseOID + "\nauthor A <a@b> 100 +0000\ncommitter A <a@b> 100 +0000\n\nmessage\n"), nil
		}
	case "rev-list":
		return []byte(blobOID + "\n"), nil
	}
	if strings.HasPrefix(args[len(args)-1], "--batch-check") {
		return []byte(blobOID + " blob 10\n"), nil
	}
	return nil, nil
}

func preparationReport() Report {
	return Report{Complete: true, State: "blocked", CanPrepare: true, Fingerprint: strings.Repeat("a", 64), Head: headOID, Base: baseOID, Limit: GitHubLimitBytes, Commits: []string{headOID}, Files: []File{{Blocked: true, Paths: []string{"artifact"}}}}
}

// [REQ:GCT-OT-P0-006] Every external failure must retain an honest failure
// record and stop subsequent work; interrupted artifacts are never success.
func TestPreparationFaultsNeverBecomeVerified(t *testing.T) {
	successCommands := &preparationCommands{}
	s := &memoryArtifacts{}
	if a, e := Prepare(context.Background(), successCommands, s, "/source", preparationReport()); e != nil || a.State != "prepared" {
		t.Fatalf("fixture: %+v %v", a, e)
	}
	for step := 1; step <= successCommands.calls; step++ {
		t.Run(strings.Repeat("x", step), func(t *testing.T) {
			c := &preparationCommands{fail: step}
			s := &memoryArtifacts{}
			_, e := Prepare(context.Background(), c, s, "/source", preparationReport())
			if e == nil || s.record == nil || s.record.State != "failed" {
				t.Fatalf("failure %d reported success: %+v %v", step, s.record, e)
			}
			if c.calls != step {
				t.Fatal("continued after failure")
			}
			before := c.calls
			a, e := Prepare(context.Background(), c, s, "/source", preparationReport())
			if e != nil || a.State != "failed" || c.calls != before {
				t.Fatal("retry overwrote failed evidence")
			}
		})
	}
}

func TestPreparationRequiresDurableReservation(t *testing.T) {
	for _, s := range []*memoryArtifacts{{beginErr: errors.New("storage offline")}, {failSave: 1}} {
		c := &preparationCommands{}
		if _, e := Prepare(context.Background(), c, s, "/source", preparationReport()); e == nil || c.calls != 0 {
			t.Fatal("commands ran without durable intent")
		}
	}
	c := &preparationCommands{}
	s := &memoryArtifacts{record: &Artifact{State: "preparing"}}
	a, e := Prepare(context.Background(), c, s, "/source", preparationReport())
	if e != nil || a.State != "preparing" || c.calls != 0 {
		t.Fatal("interrupted/in-flight operation was duplicated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s = &memoryArtifacts{}
	if _, e = Prepare(ctx, c, s, "/source", preparationReport()); e == nil || s.record.State != "failed" {
		t.Fatal("cancellation became prepared")
	}
}

func TestDiskArtifactIntegrityAndIsolation(t *testing.T) {
	root := t.TempDir()
	s := DiskStore{Root: root}
	key := strings.Repeat("a", 64)
	if _, _, e := s.Begin(context.Background(), "../../escape"); e == nil {
		t.Fatal("accepted traversal")
	}
	dir, _, e := s.Begin(context.Background(), key)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Save(dir, Artifact{State: "prepared", Candidate: headOID, Fingerprint: key}); e != nil {
		t.Fatal(e)
	}
	_, a, e := s.Begin(context.Background(), key)
	if e != nil || a.Candidate != headOID {
		t.Fatal("lost durable record")
	}
	if e = os.WriteFile(filepath.Join(dir, "recovery.json"), []byte("broken"), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Begin(context.Background(), key); e == nil {
		t.Fatal("corrupt artifact accepted")
	}
}

type reservationTestStore struct{ DiskStore }

func (s reservationTestStore) Digest(context.Context, string) (string, error) {
	return strings.Repeat("d", 64), nil
}

func TestConcurrentPreparationReservesOneWriter(t *testing.T) {
	store := reservationTestStore{DiskStore{Root: t.TempDir()}}
	commands := make([]*preparationCommands, 12)
	var group sync.WaitGroup
	for i := range commands {
		commands[i] = &preparationCommands{}
		group.Add(1)
		go func(c *preparationCommands) {
			defer group.Done()
			_, _ = Prepare(context.Background(), c, store, "/source", preparationReport())
		}(commands[i])
	}
	group.Wait()
	writers := 0
	for _, c := range commands {
		if c.calls > 0 {
			writers++
		}
	}
	if writers != 1 {
		t.Fatalf("concurrent requests started %d writers", writers)
	}
	a, e := store.Load("/source", preparationReport().Fingerprint)
	if e != nil || a.State != "damaged" {
		t.Fatalf("missing fake bundles must not be verified: %+v %v", a, e)
	}
	if _, e = store.Load("/different-source", preparationReport().Fingerprint); e == nil {
		t.Fatal("cross-repository artifact read accepted")
	}
}

// [REQ:GCT-OT-P0-006] Reattachment must survive a changed source HEAD and
// must never present missing or changed bundle bytes as verified.
func TestRecoveryDiscoveryAndIntegrity(t *testing.T) {
	store := DiskStore{Root: t.TempDir()}
	key := strings.Repeat("b", 64)
	dir, _, e := store.Begin(context.Background(), key)
	if e != nil {
		t.Fatal(e)
	}
	a := Artifact{
		Source: "/source", Fingerprint: key, State: "prepared", Head: headOID,
		OriginalBundle: filepath.Join(dir, "original.bundle"), RepairedBundle: filepath.Join(dir, "repaired.bundle"),
	}
	for _, p := range []string{a.OriginalBundle, a.RepairedBundle} {
		if e := os.WriteFile(p, []byte("bundle bytes"), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	a.OriginalDigest, e = store.Digest(context.Background(), a.OriginalBundle)
	if e != nil {
		t.Fatal(e)
	}
	a.RepairedDigest, e = store.Digest(context.Background(), a.RepairedBundle)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Save(dir, a); e != nil {
		t.Fatal(e)
	}
	found, e := store.Load("/source", "")
	if e != nil || found.Fingerprint != key || found.State != "prepared" {
		t.Fatalf("discovery: %+v %v", found, e)
	}
	other, e := store.Load("/another", "")
	if e != nil || other.State != "absent" {
		t.Fatalf("cross-repo discovery: %+v %v", other, e)
	}
	if e := os.WriteFile(a.RepairedBundle, []byte("corrupt"), 0o600); e != nil {
		t.Fatal(e)
	}
	found, e = store.Load("/source", key)
	if e != nil || found.State != "damaged" {
		t.Fatalf("corruption accepted: %+v %v", found, e)
	}
	if e := os.Remove(a.OriginalBundle); e != nil {
		t.Fatal(e)
	}
	found, e = store.Load("/source", key)
	if e != nil || found.State != "damaged" {
		t.Fatalf("missing backup accepted: %+v %v", found, e)
	}
	// Reading status must not erase or rewrite historical evidence.
	data, e := os.ReadFile(filepath.Join(dir, "recovery.json"))
	if e != nil || !strings.Contains(string(data), `"state": "prepared"`) {
		t.Fatal("status mutated record")
	}
}

func TestRecoveryLegacyAndInterruptedRecords(t *testing.T) {
	store := DiskStore{Root: t.TempDir()}
	key := strings.Repeat("c", 64)
	dir, _, e := store.Begin(context.Background(), key)
	if e != nil {
		t.Fatal(e)
	}
	for _, state := range []string{"prepared", "preparing", "failed"} {
		if e = store.Save(dir, Artifact{Source: "/source", Fingerprint: key, State: state}); e != nil {
			t.Fatal(e)
		}
		got, e := store.Load("/source", key)
		if e != nil {
			t.Fatal(e)
		}
		want := state
		if state == "prepared" {
			want = "unverified"
		}
		if got.State != want {
			t.Fatalf("%s became %s", state, got.State)
		}
	}
}

func TestRecoveryCurrentEligibility(t *testing.T) {
	a := Artifact{State: "prepared", Fingerprint: "reviewed"}
	for _, tc := range []struct {
		name   string
		report Report
		want   string
	}{
		{"same", Report{Complete: true, Fingerprint: "reviewed"}, "prepared"},
		{"remote moved", Report{Complete: true, Fingerprint: "different"}, "stale"},
		{"offline", Report{Complete: false}, "unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckCurrent(a, tc.report)
			if got.State != tc.want {
				t.Fatalf("%+v", got)
			}
		})
	}
	if CheckCurrent(Artifact{State: "damaged"}, Report{Complete: true, Fingerprint: ""}).State != "damaged" {
		t.Fatal("current remote hid damaged backup")
	}
}

func TestRecoveryIncompleteDiscoveryIsNotAbsent(t *testing.T) {
	store := DiskStore{Root: t.TempDir()}
	key := strings.Repeat("a", 64)
	dir, _, e := store.Begin(context.Background(), key)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.Load("/source", ""); e == nil {
		t.Fatal("reservation without record was hidden")
	}
	if _, e = store.Load("/source", key); e == nil {
		t.Fatal("interrupted reservation became absent")
	}
	if e = os.WriteFile(filepath.Join(dir, "recovery.json"), []byte("broken JSON"), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e = store.Load("/source", ""); e == nil {
		t.Fatal("corrupt record was hidden")
	}
}

func TestRecoveryDigestCancellationAndSymlinks(t *testing.T) {
	store := DiskStore{Root: t.TempDir()}
	file := filepath.Join(store.Root, "bundle")
	if e := os.WriteFile(file, []byte("content"), 0o600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := store.Digest(ctx, file); e == nil {
		t.Fatal("cancelled integrity check passed")
	}
	link := filepath.Join(store.Root, "link")
	if e := os.Symlink(file, link); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Digest(context.Background(), link); e == nil {
		t.Fatal("symlink bundle accepted")
	}
}

func TestPreparationIntegrityAndFinalRecordFailures(t *testing.T) {
	for _, store := range []*memoryArtifacts{{failDigest: 1}, {failDigest: 2}, {failDigest: 3}, {failDigest: 4}, {changeDigest: 3}, {changeDigest: 4}, {failSave: 2}} {
		a, e := Prepare(context.Background(), &preparationCommands{}, store, "/source", preparationReport())
		if e == nil || a.State == "prepared" || store.record == nil || store.record.State != "failed" {
			t.Fatalf("failure became verified: %+v %v", a, e)
		}
	}
}
