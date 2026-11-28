package coreactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

type fakeEffects struct {
	calls       []string
	fail        map[string]error
	prior       bool
	restoreLost bool
	onCall      func(string)
}

func (f *fakeEffects) call(name string) error {
	f.calls = append(f.calls, name)
	if f.onCall != nil {
		f.onCall(name)
	}
	return f.fail[name]
}
func (f *fakeEffects) QualifyAdmission(context.Context, Record) error { return f.call("qualify") }
func (f *fakeEffects) AuthorizeWorker(context.Context, Record, platform.OperationRef) error {
	return f.call("authorize")
}

func (f *fakeEffects) PublishCandidate(context.Context, Record) error {
	f.prior = false
	return f.call("publish")
}
func (f *fakeEffects) ActivateCandidate(context.Context, Record) error { return f.call("activate") }
func (f *fakeEffects) VerifyCandidate(context.Context, Record) error   { return f.call("verify") }
func (f *fakeEffects) ExcludeWorker(context.Context, Record) error     { return f.call("exclude") }
func (f *fakeEffects) VerifyPrior(context.Context, Record) error {
	if err := f.call("prior"); err != nil {
		return err
	}
	if !f.prior {
		return errors.New("prior is not serving")
	}
	return nil
}

func (f *fakeEffects) RestorePrior(context.Context, Record) error {
	if err := f.call("restore"); err != nil {
		return err
	}
	f.prior = true
	if f.restoreLost {
		f.restoreLost = false
		return errors.New("restoration response lost")
	}
	return nil
}
func (f *fakeEffects) RetireInputs(context.Context, Record) error { return f.call("retire") }

func fixture(t *testing.T) (*Engine, Record, platform.OperationRef, *fakeEffects) {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	id := strings.Repeat("1", 32)
	intent, err := platform.NewOperationIntent("core-activation", id, fixtureSpec(Record{ID: id, Home: s.home}))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(intent)
	sum := sha256.Sum256(data)
	unit := "vrooli-op-core-activation-" + hex.EncodeToString(sum[:12]) + ".service"
	ref := platform.OperationRef{Intent: intent, Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/fixture/" + platform.OperationsSlice + "/" + unit}, RootPID: 123, RootStartTime: "12345", NativeInvocationID: strings.Repeat("4", 32)}
	r := Record{ID: id, Home: s.home, SourceRoot: filepath.Join(s.home, "source"), CandidateSHA: strings.Repeat("a", 64), CandidateBuild: strings.Repeat("b", 64), CandidateStateContract: strings.Repeat("c", 64), CandidateUnitSHA: strings.Repeat("d", 64), PriorSHA: strings.Repeat("e", 64), PriorBuild: strings.Repeat("f", 64), PriorStateContract: strings.Repeat("c", 64), PriorUnitSHA: strings.Repeat("0", 64), PriorPID: 456, PriorIncarnation: "01234567-89ab-cdef-0123-456789abcdef:12345", PriorNativeInvocation: strings.Repeat("5", 32), CreatedUnixNano: now.UnixNano(), DeadlineUnixNano: now.Add(time.Minute).UnixNano(), WorkerIntent: intent}
	f := &fakeEffects{fail: map[string]error{}, prior: true}
	r.LayoutSHA = strings.Repeat("7", 64)
	r.WorkerSHA = strings.Repeat("8", 64)
	r.NativeOverridesDigest, _ = NativeOverridesDigest(context.Background(), "")
	return &Engine{Store: s, Effects: f, qualifyFrozen: func(context.Context, Record) error { return nil }, Now: func() time.Time { return now }, start: func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
		return &platform.OwnedOperation{Ref: ref}, nil
	}}, r, ref, f
}

func fixtureSpec(r Record) platform.OperationSpec {
	return platform.OperationSpec{Path: filepath.Join(r.Home, "watchdog"), Dir: r.Home, Args: []string{"--core-update-execute", r.ID}, Env: []string{}, Limits: platform.OperationLimits{MemoryMaxBytes: 128 << 20, TasksMax: 16, CPUWeight: 10, WallTime: time.Minute}}
}

func dispatch(t *testing.T, e *Engine, r Record) {
	t.Helper()
	if _, err := e.Dispatch(context.Background(), r.ID, fixtureSpec(r)); err != nil {
		t.Fatal(err)
	}
}

func admit(t *testing.T, e *Engine, r Record) Record {
	t.Helper()
	result, err := e.Begin(context.Background(), r, false)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestRecoveryRetainsRestorationIntentAfterLostResponse(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	f.fail["activate"] = errors.New("candidate failed")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("expected activation failure")
	}
	f.restoreLost = true
	if _, _, err := e.Recover(context.Background()); err == nil {
		t.Fatal("expected lost restoration acknowledgement")
	}
	got, _, err := e.Recover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != FailedRestored {
		t.Fatalf("lost response changed restoration outcome: %s", got.Phase)
	}
	if !contains(got.FailedReleases, r.CandidateSHA) {
		t.Fatal("failed candidate is not suppressed")
	}
}

func TestDispatchAmbiguityNeverPermitsResubmission(t *testing.T) {
	for _, mode := range []string{"commit_before_write", "commit_after_write", "native_response_lost"} {
		t.Run(mode, func(t *testing.T) {
			e, r, _, _ := fixture(t)
			admit(t, e, r)
			calls := 0
			e.start = func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error) {
				calls++
				return nil, errors.New("lost response")
			}
			write := e.Store.write
			if mode != "native_response_lost" {
				e.Store.write = func(path string, data []byte, modeBits os.FileMode) error {
					if mode == "commit_after_write" {
						if err := write(path, data, modeBits); err != nil {
							return err
						}
					}
					return errors.New("commit uncertain")
				}
			}
			if _, err := e.Dispatch(context.Background(), r.ID, fixtureSpec(r)); err == nil {
				t.Fatal("expected ambiguous result")
			}
			e.Store.write = write
			if mode == "commit_before_write" {
				if calls != 0 {
					t.Fatal("native effect after failed commit")
				}
				// The durable record proves no submission was admitted. This
				// retry is safe; the first attempt never reached native dispatch.
				if _, err := e.Dispatch(context.Background(), r.ID, fixtureSpec(r)); err == nil {
					t.Fatal("expected injected native error")
				}
			} else if _, err := e.Dispatch(context.Background(), r.ID, fixtureSpec(r)); !errors.Is(err, ErrClaimed) {
				t.Fatalf("ambiguous submission replayed: %v", err)
			}
			want := 0
			if mode != "commit_after_write" {
				want = 1
			}
			if calls != want {
				t.Fatalf("native submissions=%d want %d", calls, want)
			}
		})
	}
}

func TestWorkerRequiresSubmissionAndCannotReplay(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	f.calls = nil
	if _, err := e.RunWorker(context.Background(), r.ID, ref); !errors.Is(err, ErrClaimed) {
		t.Fatalf("unsubmitted worker allowed: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("unsubmitted effects: %v", f.calls)
	}
	dispatch(t, e, r)
	got, err := e.RunWorker(context.Background(), r.ID, ref)
	if err != nil || got.Phase != Verified {
		t.Fatalf("worker failed: %s %v", got.Phase, err)
	}
	f.calls = nil
	if _, err := e.RunWorker(context.Background(), r.ID, ref); !errors.Is(err, ErrClaimed) {
		t.Fatalf("completed worker replayed: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("replayed effects: %v", f.calls)
	}
}

func TestWorkerCommitLossRecoversWithoutUnrecordedEffects(t *testing.T) {
	for _, afterWrite := range []bool{false, true} {
		for edge := 1; edge <= 5; edge++ {
			t.Run(fmt.Sprintf("edge_%d_visible_%t", edge, afterWrite), func(t *testing.T) {
				e, r, ref, f := fixture(t)
				admit(t, e, r)
				dispatch(t, e, r)
				original := e.Store.write
				n := 0
				f.onCall = func(name string) {
					phase := map[string]Phase{"publish": Publishing, "activate": Activating, "verify": Verifying}[name]
					if phase == "" {
						return
					}
					durable, _, err := e.Store.Current()
					if err != nil || durable.Phase != phase || durable.Fence != Claimed {
						t.Fatalf("%s before durable effect permission: %s %v", name, durable.Phase, err)
					}
				}
				e.Store.write = func(path string, data []byte, mode os.FileMode) error {
					n++
					if n != edge {
						return original(path, data, mode)
					}
					if afterWrite {
						if err := original(path, data, mode); err != nil {
							return err
						}
					}
					return errors.New("injected commit loss")
				}
				if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
					t.Fatal("expected commit failure")
				}
				e.Store.write = original
				f.onCall = nil
				// An unclaimed dispatch may still arrive, so only its admitted
				// deadline permits recovery to revoke it.
				e.Now = func() time.Time { return time.Unix(0, r.DeadlineUnixNano).Add(time.Second) }
				got, _, err := e.Recover(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				want := FailedRestored
				if edge <= 2 {
					want = FailedPreserved
				}
				if edge == 5 && afterWrite {
					want = Verified
				}
				if got.Phase != want {
					t.Fatalf("phase=%s want %s", got.Phase, want)
				}
				f.calls = nil
				if _, err := e.RunWorker(context.Background(), r.ID, ref); !errors.Is(err, ErrClaimed) {
					t.Fatalf("late worker allowed: %v", err)
				}
				if len(f.calls) != 0 {
					t.Fatalf("late effects: %v", f.calls)
				}
			})
		}
	}
}

func TestUnknownWorkerQuiescencePreventsRestorationAndCleanup(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	f.fail["activate"] = errors.New("bad candidate")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("expected failure")
	}
	f.calls = nil
	f.fail["exclude"] = errors.New("native owner unavailable")
	if _, _, err := e.Recover(context.Background()); err == nil {
		t.Fatal("unknown tree accepted")
	}
	if !reflect.DeepEqual(f.calls, []string{"exclude"}) {
		t.Fatalf("unsafe recovery effects: %v", f.calls)
	}
}

func TestFailedReleaseRequiresExplicitRetry(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	f.fail["activate"] = errors.New("bad candidate")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("expected failure")
	}
	if _, _, err := e.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	delete(f.fail, "activate")
	next := r
	next.ID = strings.Repeat("6", 32)
	next.WorkerIntent.AttemptID = next.ID
	if retained, err := e.Begin(context.Background(), next, false); !errors.Is(err, ErrFailedRelease) {
		t.Fatalf("failed release readmitted: %v", err)
	} else if retained.ID != r.ID || !retained.Terminal() {
		t.Fatalf("rejected retry advertised an unadmitted identity: %+v", retained)
	}
	if _, err := e.Begin(context.Background(), next, true); err != nil {
		t.Fatalf("explicit retry rejected: %v", err)
	}
}

func TestRebuildingFailedSourceWithDifferentBytesCannotEvadeSuppression(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	f.fail["activate"] = errors.New("bad source build")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("failed trial verified")
	}
	if _, _, err := e.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	next := r
	next.ID = strings.Repeat("6", 32)
	next.WorkerIntent.AttemptID = next.ID
	next.CandidateSHA = strings.Repeat("7", 64)
	if _, err := e.Begin(context.Background(), next, false); !errors.Is(err, ErrFailedRelease) {
		t.Fatalf("rebuild bytes evaded failed source suppression: %v", err)
	}
}

func TestMissingPreviouslyAdmittedRecordIsNotFreshStore(t *testing.T) {
	e, r, _, f := fixture(t)
	admit(t, e, r)
	if err := os.Remove(e.Store.path); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := e.Store.Current(); err == nil || exists {
		t.Fatal("missing admitted state was accepted as a fresh store")
	}
	f.calls = nil
	if _, _, err := e.Recover(context.Background()); err == nil {
		t.Fatal("recovery ignored lost state")
	}
	if len(f.calls) != 0 {
		t.Fatalf("effects after lost state: %v", f.calls)
	}
}
