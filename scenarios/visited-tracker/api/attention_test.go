package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
)

func attentionFixture(t *testing.T, count int) (*Campaign, time.Time) {
	t.Helper()
	initTestStorageRoot(t, t.TempDir())
	cleanup := setupTestLogger()
	t.Cleanup(cleanup)
	root := t.TempDir()
	for i := 0; i < count; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%02d.txt", i)), []byte(fmt.Sprintf("contents %d", i)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	c := &Campaign{ID: uuid.New(), Name: "attention", Location: &root, Patterns: []string{"*.txt"}, MaxFiles: 100, CreatedAt: now, Status: "active"}
	if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
		t.Fatal(err)
	}
	if err := saveCampaign(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c, now
}

// [REQ:VT-REQ-024] Concurrent workers must never own the same file revision.
func TestAttentionConcurrentClaimsAndReplay(t *testing.T) {
	c, now := attentionFixture(t, 8)
	var wg sync.WaitGroup
	results := make(chan *ReviewClaim, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, e := claimAttention(context.Background(), c.ID, fmt.Sprint(i), fmt.Sprint(i), 600, now)
			results <- r
			errs <- e
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[uuid.UUID]bool{}
	for r := range results {
		if r == nil || seen[r.FileID] {
			t.Fatalf("duplicate or missing claim: %+v", r)
		}
		seen[r.FileID] = true
		replay, err := claimAttention(context.Background(), c.ID, r.RequestID, r.Worker, 600, now.Add(time.Hour))
		if err != nil || replay.ID != r.ID {
			t.Fatalf("retry allocated new work: %+v %v", replay, err)
		}
	}
}

// [REQ:VT-REQ-024] Only current, unexpired ownership can grant revision-specific credit.
func TestAttentionCompletionGuardsAndIdempotency(t *testing.T) {
	for _, mode := range []string{"changed", "expired", "wrong-worker", "reviewed", "incomplete", "skipped", "excluded"} {
		t.Run(mode, func(t *testing.T) {
			c, now := attentionFixture(t, 1)
			r, err := claimAttention(context.Background(), c.ID, "request", "worker", 60, now)
			if err != nil {
				t.Fatal(err)
			}
			worker, outcome, evidence := "worker", "reviewed", "test evidence"
			expected := error(nil)
			switch mode {
			case "changed":
				if err := os.WriteFile(filepath.Join(*c.Location, r.FilePath), []byte("new revision"), 0o600); err != nil {
					t.Fatal(err)
				}
				expected = ErrClaimConflict
			case "expired":
				now = now.Add(time.Minute)
				expected = ErrClaimExpired
			case "wrong-worker":
				worker = "other"
				expected = ErrClaimConflict
			case "excluded":
				_, err = mutateCampaign(context.Background(), c.ID, func(c *Campaign) error { c.TrackedFiles[0].Excluded = true; return nil })
				if err != nil {
					t.Fatal(err)
				}
				expected = ErrClaimConflict
			case "incomplete", "skipped":
				outcome = mode
				evidence = ""
			}
			_, err = completeAttention(context.Background(), c.ID, r.ID, worker, outcome, evidence, now)
			if !errors.Is(err, expected) {
				t.Fatalf("got %v, want %v", err, expected)
			}
			if expected == nil {
				if _, err := completeAttention(context.Background(), c.ID, r.ID, worker, outcome, evidence, now.Add(time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
			stored, err := loadCampaign(c.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if mode == "reviewed" {
				want = 1
			}
			if stored.TrackedFiles[0].ReviewCount != want {
				t.Fatalf("review credit=%d want %d", stored.TrackedFiles[0].ReviewCount, want)
			}
		})
	}
}

// [REQ:VT-REQ-026] Detached stale writes cannot erase a committed mutation.
func TestCampaignConcurrentTransactionsAndStaleSave(t *testing.T) {
	c, _ := attentionFixture(t, 1)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mutateCampaign(context.Background(), c.ID, func(c *Campaign) error { c.TrackedFiles[0].VisitCount++; return nil })
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := saveCampaign(context.Background(), c); !errors.Is(err, ErrCampaignConflict) {
		t.Fatalf("stale save: %v", err)
	}
	stored, err := loadCampaign(c.ID)
	if err != nil || stored.TrackedFiles[0].VisitCount != 20 {
		t.Fatalf("lost mutation: %+v %v", stored, err)
	}
}

// [REQ:VT-REQ-027] Refresh preserves unique moves and invalidates changed revisions.
func TestAttentionRevisionAndRename(t *testing.T) {
	c, now := attentionFixture(t, 1)
	r, err := claimAttention(context.Background(), c.ID, "r", "w", 600, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := completeAttention(context.Background(), c.ID, r.ID, "w", "reviewed", "evidence", now); err != nil {
		t.Fatal(err)
	}
	c, err = loadCampaign(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	original := c.TrackedFiles[0]
	renamed := filepath.Join(*c.Location, "renamed.txt")
	if err := os.Rename(original.AbsolutePath, renamed); err != nil {
		t.Fatal(err)
	}
	result, err := syncCampaignFiles(c, c.Patterns)
	if err != nil || result.Moved != 1 {
		t.Fatalf("move: %+v %v", result, err)
	}
	if c.TrackedFiles[0].ID != original.ID || c.TrackedFiles[0].ReviewedRevision != original.ReviewedRevision {
		t.Fatal("rename lost history")
	}
	before := attentionScore(c.TrackedFiles[0], now)
	if err := os.WriteFile(renamed, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
		t.Fatal(err)
	}
	if attentionScore(c.TrackedFiles[0], now) <= before {
		t.Fatal("changed revision remains deprioritized")
	}
}

// [REQ:VT-REQ-025] Preview does not consume work or write campaign state.
func TestAttentionPreviewReadOnlyAndBounds(t *testing.T) {
	c, _ := attentionFixture(t, 2)
	rpc := &attentionRPC{}
	before, err := os.ReadFile(getCampaignPath(c.ID))
	if err != nil {
		t.Fatal(err)
	}
	res, err := rpc.Preview(context.Background(), connect.NewRequest(&attentionv1.PreviewRequest{CampaignId: c.ID.String(), Limit: 1}))
	if err != nil || len(res.Msg.Candidates) != 1 {
		t.Fatalf("preview: %+v %v", res, err)
	}
	after, err := os.ReadFile(getCampaignPath(c.ID))
	if err != nil || string(before) != string(after) {
		t.Fatal("preview mutated persisted state")
	}
	_, err = rpc.Preview(context.Background(), connect.NewRequest(&attentionv1.PreviewRequest{CampaignId: c.ID.String(), Limit: 101}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unbounded preview accepted: %v", err)
	}
}

// [REQ:VT-REQ-024] Expiry releases ownership but never certifies abandoned work.
func TestAttentionExpiryRecovery(t *testing.T) {
	c, now := attentionFixture(t, 1)
	first, err := claimAttention(context.Background(), c.ID, "first", "one", 60, now)
	if err != nil {
		t.Fatal(err)
	}
	held, err := claimAttention(context.Background(), c.ID, "second", "two", 60, now)
	if err != nil || held != nil {
		t.Fatalf("live ownership not exclusive: %+v %v", held, err)
	}
	next, err := claimAttention(context.Background(), c.ID, "second", "two", 60, now.Add(time.Minute))
	if err != nil || next == nil || next.ID == first.ID {
		t.Fatalf("expired ownership not recovered: %+v %v", next, err)
	}
	if _, err := completeAttention(context.Background(), c.ID, first.ID, "one", "reviewed", "late", now.Add(time.Minute)); !errors.Is(err, ErrClaimExpired) {
		t.Fatalf("expired owner accepted: %v", err)
	}
	stored, err := loadCampaign(c.ID)
	if err != nil || stored.TrackedFiles[0].ReviewCount != 0 {
		t.Fatalf("abandonment granted credit: %+v %v", stored, err)
	}
}

func TestAttentionArchivesTerminalClaimsAndPreservesReplay(t *testing.T) {
	c, now := attentionFixture(t, 1)
	first, err := claimAttention(context.Background(), c.ID, "first", "one", 60, now)
	if err != nil || first == nil {
		t.Fatalf("initial claim: %+v %v", first, err)
	}
	if _, err := completeAttention(context.Background(), c.ID, first.ID, "one", "incomplete", "", now); err != nil {
		t.Fatalf("complete: %v", err)
	}
	second, err := claimAttention(context.Background(), c.ID, "second", "two", 60, now.Add(time.Second))
	if err != nil || second == nil {
		t.Fatalf("claim after terminal archive: %+v %v", second, err)
	}
	stored, err := loadCampaign(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Claims) != 1 || len(stored.ArchivedClaims) != 1 || stored.ArchivedClaims[0].ID != first.ID {
		t.Fatalf("terminal claim was not archived: active=%d archived=%d", len(stored.Claims), len(stored.ArchivedClaims))
	}
	replay, err := claimAttention(context.Background(), c.ID, "first", "one", 60, now.Add(2*time.Second))
	if err != nil || replay == nil || replay.ID != first.ID || replay.Outcome != "incomplete" {
		t.Fatalf("archived replay changed identity: %+v %v", replay, err)
	}
	if _, err := completeAttention(context.Background(), c.ID, first.ID, "one", "reviewed", "late", now.Add(2*time.Second)); !errors.Is(err, ErrClaimExpired) {
		t.Fatalf("archived claim accepted new completion: %v", err)
	}
}

func TestAttentionPreviewReportsFairnessTelemetry(t *testing.T) {
	c, now := attentionFixture(t, 2)
	if _, err := mutateCampaign(context.Background(), c.ID, func(c *Campaign) error {
		for i := range c.TrackedFiles {
			c.TrackedFiles[i].FirstSeen = now.Add(-time.Hour)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	claim, err := claimAttention(context.Background(), c.ID, "claim", "worker", 600, now)
	if err != nil || claim == nil {
		t.Fatalf("claim: %+v %v", claim, err)
	}
	rpc := &attentionRPC{}
	res, err := rpc.Preview(context.Background(), connect.NewRequest(&attentionv1.PreviewRequest{CampaignId: c.ID.String(), Limit: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.EligibleCount != 1 || res.Msg.ActiveClaimCount != 1 {
		t.Fatalf("unexpected fairness counts: eligible=%d active=%d", res.Msg.EligibleCount, res.Msg.ActiveClaimCount)
	}
	if res.Msg.OldestActiveClaimAgeSeconds != 0 || res.Msg.OldestEligibleAgeSeconds == 0 {
		t.Fatalf("unexpected fairness ages: claim=%d eligible=%d", res.Msg.OldestActiveClaimAgeSeconds, res.Msg.OldestEligibleAgeSeconds)
	}
}

// [REQ:VT-REQ-027] Ambiguous renames must not transfer another file's history.
func TestAttentionAmbiguousRename(t *testing.T) {
	c, _ := attentionFixture(t, 2)
	for _, f := range c.TrackedFiles {
		if err := os.WriteFile(f.AbsolutePath, []byte("identical"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
		t.Fatal(err)
	}
	for i, f := range c.TrackedFiles {
		if err := os.Rename(f.AbsolutePath, filepath.Join(*c.Location, fmt.Sprintf("renamed-%d.txt", i))); err != nil {
			t.Fatal(err)
		}
	}
	result, err := syncCampaignFiles(c, c.Patterns)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 0 || result.Added != 2 || result.Removed != 2 {
		t.Fatalf("ambiguous identity transferred: %+v", result)
	}
}

// [REQ:VT-REQ-027] An escaping symlink fails without partially applying a snapshot.
func TestAttentionScanFailurePreservesSnapshot(t *testing.T) {
	c, _ := attentionFixture(t, 1)
	before, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(*c.Location, "escape.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := syncCampaignFiles(c, c.Patterns); err == nil {
		t.Fatal("escaping symlink accepted")
	}
	after, err := json.Marshal(c)
	if err != nil || string(before) != string(after) {
		t.Fatal("failed scan partially changed campaign")
	}
}

// [REQ:VT-REQ-025] Concurrent callers share one purpose-scoped campaign and
// incompatible settings cannot silently change another caller's work scope.
func TestAttentionEnsureCampaignConcurrentIdentity(t *testing.T) {
	c, _ := attentionFixture(t, 2)
	rpc := &attentionRPC{}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := rpc.EnsureCampaign(context.Background(), connect.NewRequest(&attentionv1.EnsureCampaignRequest{Location: *c.Location, Tag: "shared-purpose", Patterns: []string{"*.txt", "*.txt"}, MaxFiles: 100}))
			errs <- err
			if err == nil {
				ids <- res.Msg.CampaignId
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatal("duplicate campaign created")
		}
	}
	if first == "" {
		t.Fatal("no campaign created")
	}
	_, err := rpc.EnsureCampaign(context.Background(), connect.NewRequest(&attentionv1.EnsureCampaignRequest{Location: *c.Location, Tag: "shared-purpose", Patterns: []string{"*.txt"}, MaxFiles: 50}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("incompatible settings accepted: %v", err)
	}
}

// [REQ:VT-REQ-027] Explicit legacy sync patterns extend scope; they must not
// tombstone the files outside that partial request or lose their identities.
func TestAttentionLegacySyncPreservesFullScope(t *testing.T) {
	c, _ := attentionFixture(t, 1)
	original := c.TrackedFiles[0].ID
	if err := os.WriteFile(filepath.Join(*c.Location, "new.go"), []byte("package example"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"patterns":["*.go"]}`))
	request = mux.SetURLVars(request, map[string]string{"id": c.ID.String()})
	response := httptest.NewRecorder()
	structureSyncHandler(response, request)
	if response.Code != 200 {
		t.Fatalf("sync failed: %s", response.Body.String())
	}
	stored, err := loadCampaign(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Patterns) != 2 || len(stored.TrackedFiles) != 2 {
		t.Fatalf("scope lost: %+v", stored)
	}
	for _, file := range stored.TrackedFiles {
		if file.Deleted {
			t.Fatalf("narrow sync deleted %s", file.FilePath)
		}
	}
	if stored.TrackedFiles[0].ID != original {
		t.Fatal("narrow sync replaced original identity")
	}
	if _, err := syncCampaignFiles(stored, stored.Patterns); err != nil {
		t.Fatal(err)
	}
	for _, file := range stored.TrackedFiles {
		if file.Deleted {
			t.Fatal("next authoritative sync lost extended file")
		}
	}
}

func TestAttentionCancelledLockWaitDoesNotMutateCampaign(t *testing.T) {
	c, now := attentionFixture(t, 1)
	release, err := lockCampaign(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = (&attentionRPC{}).Claim(ctx, connect.NewRequest(&attentionv1.ClaimRequest{CampaignId: c.ID.String(), RequestId: "blocked", Worker: "worker", TtlSeconds: 60}))
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("claim error: %v", err)
	}
	current, err := loadCampaign(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != c.Revision || len(current.Claims) != 0 {
		t.Fatalf("cancelled waiter wrote campaign: %+v", current)
	}
	release()
	if _, err := claimAttention(context.Background(), c.ID, "next", "worker", 60, now); err != nil {
		t.Fatal(err)
	}
}

func TestAttentionExplorationBoundsRepeatedIncompleteWork(t *testing.T) {
	c, now := attentionFixture(t, 5)
	c.TrackedFiles[0].PriorityWeight = 100
	if err := saveCampaign(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	selected := map[uuid.UUID]bool{}
	for i := 0; i < 4*len(c.TrackedFiles); i++ {
		request := fmt.Sprintf("attempt-%d", i)
		claim, err := claimAttention(context.Background(), c.ID, request, "worker", 60, now)
		if err != nil || claim == nil {
			t.Fatalf("claim: %+v %v", claim, err)
		}
		selected[claim.FileID] = true
		if _, err := claimAttention(context.Background(), c.ID, request, "worker", 60, now); err != nil {
			t.Fatal(err)
		}
		if _, err := completeAttention(context.Background(), c.ID, claim.ID, "worker", "incomplete", "", now); err != nil {
			t.Fatal(err)
		}
	}
	current, err := loadCampaign(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != len(c.TrackedFiles) {
		t.Fatalf("priority work starved files: %d/%d", len(selected), len(c.TrackedFiles))
	}
	if current.AttentionSequence != uint64(4*len(c.TrackedFiles)) {
		t.Fatalf("replays advanced fairness: %d", current.AttentionSequence)
	}
	for _, file := range current.TrackedFiles {
		if file.ReviewCount != 0 || file.LastReviewed != nil {
			t.Fatal("exploration fabricated review credit")
		}
	}
}
