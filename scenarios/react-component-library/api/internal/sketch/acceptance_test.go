package sketch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func acceptanceFixture(t *testing.T) (*Store, string, AcceptanceIntent, AcceptanceFacts) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(`{"sketch":{}}`), 0600))
	store := NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	c, err := store.SaveCandidate("demo", "home", "design-one", base.ContentHash, Document{Template: &AssetRef{Asset: "templates.collection-page", Version: "1.7.0"}})
	require.NoError(t, err)
	intent := AcceptanceIntent{DesignID: c.DesignID, CandidateHash: c.Hash, ExpectedRenderHash: strings.Repeat("a", 64), Actor: "test-operator", CritiqueIDs: []string{"review-one"}}
	f := AcceptanceFacts{PolicyVersion: AcceptancePolicy, CandidateHash: c.Hash, RenderHash: intent.ExpectedRenderHash, InputHashes: map[string]string{}}
	for _, key := range []string{"appearance", "bindings", "composition", "dependencies", "fixtures", "harness", "bundle"} {
		f.InputHashes[key] = strings.Repeat("b", 64)
	}
	for _, code := range []string{"page_freshness", "render_freshness", "render_completeness", "visual_review", "behavior_evidence", "rubric_calibration", "independent_review"} {
		f.Requirements = append(f.Requirements, AcceptanceRequirement{Code: code, Status: "passed", Detail: "Verified fixture evidence", EvidenceIDs: []string{"producer-proof"}})
	}
	return store, path, intent, f
}
func TestAcceptancePreparedIntentAndImmutableReplay(t *testing.T) {
	s, path, i, f := acceptanceFixture(t)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	id, err := acceptanceID("one")
	require.NoError(t, err)
	calls := 0
	evaluate := func(context.Context) (AcceptanceFacts, error) {
		calls++
		pending, e := s.ReadAcceptance("demo", i.DesignID, id)
		require.NoError(t, e)
		require.Equal(t, "prepared", pending.State)
		return f, nil
	}
	first, err := s.RecordAcceptance(context.Background(), "demo", "one", i, evaluate)
	require.NoError(t, err)
	require.Equal(t, "accepted", first.State)
	replay, err := s.RecordAcceptance(context.Background(), "demo", "one", i, evaluate)
	require.NoError(t, err)
	require.Equal(t, first, replay)
	require.Equal(t, 1, calls)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	i.Actor = "different"
	_, err = s.RecordAcceptance(context.Background(), "demo", "one", i, evaluate)
	require.ErrorIs(t, err, ErrAcceptanceConflict)
}
func TestAcceptanceMissingOrContradictoryProofNeverAccepts(t *testing.T) {
	cases := map[string]func(*AcceptanceFacts){
		"missing requirement":         func(f *AcceptanceFacts) { f.Requirements = f.Requirements[:6] },
		"blocked requirement":         func(f *AcceptanceFacts) { f.Requirements[0].Status = "blocked" },
		"no evidence":                 func(f *AcceptanceFacts) { f.Requirements[0].EvidenceIDs = nil },
		"empty evidence":              func(f *AcceptanceFacts) { f.Requirements[0].EvidenceIDs = []string{""} },
		"stale render":                func(f *AcceptanceFacts) { f.RenderHash = strings.Repeat("f", 64) },
		"wrong policy":                func(f *AcceptanceFacts) { f.PolicyVersion = "future" },
		"missing dependency identity": func(f *AcceptanceFacts) { delete(f.InputHashes, "dependencies") },
		"contradiction": func(f *AcceptanceFacts) {
			f.Requirements = append(f.Requirements, AcceptanceRequirement{Code: "visual_review", Status: "blocked"})
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s, _, i, f := acceptanceFixture(t)
			change(&f)
			d, err := s.RecordAcceptance(context.Background(), "demo", "one", i, func(context.Context) (AcceptanceFacts, error) { return f, nil })
			require.NoError(t, err)
			require.Equal(t, "needs_evidence", d.State)
		})
	}
}
func TestAcceptanceRechecksPageAfterEvidenceEvaluation(t *testing.T) {
	s, _, i, f := acceptanceFixture(t)
	d, err := s.RecordAcceptance(context.Background(), "demo", "one", i, func(context.Context) (AcceptanceFacts, error) {
		base, e := s.Read("demo", "home")
		require.NoError(t, e)
		_, e = s.Save("demo", "home", base.ContentHash, Document{Viewport: "phone"})
		require.NoError(t, e)
		return f, nil
	})
	require.NoError(t, err)
	require.Equal(t, "needs_evidence", d.State)
	require.Contains(t, d.Detail, "changed")
}
func TestAcceptanceResumesPreparedAfterCrashAndRecordsProducerFailure(t *testing.T) {
	s, _, i, f := acceptanceFixture(t)
	func() {
		defer func() { require.Equal(t, "crash", recover()) }()
		_, _ = s.RecordAcceptance(context.Background(), "demo", "one", i, func(context.Context) (AcceptanceFacts, error) { panic("crash") })
	}()
	d, err := s.RecordAcceptance(context.Background(), "demo", "one", i, func(context.Context) (AcceptanceFacts, error) { return f, nil })
	require.NoError(t, err)
	require.Equal(t, "accepted", d.State)
	d, err = s.RecordAcceptance(context.Background(), "demo", "failure", i, func(context.Context) (AcceptanceFacts, error) {
		return AcceptanceFacts{}, errors.New("producer unavailable")
	})
	require.NoError(t, err)
	require.Equal(t, "failed", d.State)
	read, err := s.ReadAcceptance("demo", i.DesignID, d.ID)
	require.NoError(t, err)
	require.Equal(t, d, read)
}
func TestAcceptanceConcurrentRetriesEvaluateOnce(t *testing.T) {
	s, _, i, f := acceptanceFixture(t)
	var calls atomic.Int32
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := s.RecordAcceptance(context.Background(), "demo", "same", i, func(context.Context) (AcceptanceFacts, error) { calls.Add(1); return f, nil })
			if err != nil || d.State != "accepted" {
				t.Errorf("concurrent decision: %s %v", d.State, err)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), calls.Load())
}
func TestAcceptanceRejectsTamperedHistory(t *testing.T) {
	s, page, i, f := acceptanceFixture(t)
	d, err := s.RecordAcceptance(context.Background(), "demo", "one", i, func(context.Context) (AcceptanceFacts, error) { return f, nil })
	require.NoError(t, err)
	relative, err := acceptancePath(i.DesignID, d.ID)
	require.NoError(t, err)
	path := filepath.Join(filepath.Dir(filepath.Dir(page)), relative)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "test-operator", "tampered", 1)), 0600))
	_, err = s.ReadAcceptance("demo", i.DesignID, d.ID)
	require.Error(t, err)
}
