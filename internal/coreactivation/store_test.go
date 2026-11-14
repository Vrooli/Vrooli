package coreactivation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRejectsAmbiguousAndUnsafeMetadataWithoutEffects(t *testing.T) {
	for _, mutation := range []string{"duplicate", "case_alias", "nested_duplicate", "nested_alias", "missing_required", "null_required", "wrong_scalar", "unknown", "trailing", "oversized", "native_namespace", "false_verified", "symlink", "empty"} {
		t.Run(mutation, func(t *testing.T) {
			e, r, _, f := fixture(t)
			r = admit(t, e, r)
			data, err := os.ReadFile(e.Store.path)
			if err != nil {
				t.Fatal(err)
			}
			s := string(data)
			switch mutation {
			case "duplicate":
				s = strings.Replace(s, `"id":`, `"id":"`+r.ID+`","id":`, 1)
			case "case_alias":
				s = strings.Replace(s, `"id":`, `"ID":`, 1)
			case "nested_duplicate":
				s = strings.Replace(s, `"owner":`, `"owner":"core-activation","owner":`, 1)
			case "nested_alias":
				s = strings.Replace(s, `"owner":`, `"Owner":`, 1)
			case "missing_required":
				s = strings.Replace(s, `"failure_class":"",`, "", 1)
			case "null_required":
				s = strings.Replace(s, `"failure_class":""`, `"failure_class":null`, 1)
			case "wrong_scalar":
				s = strings.Replace(s, `"dispatch_committed":false`, `"dispatch_committed":"false"`, 1)
			case "unknown":
				s = strings.Replace(s, `"schema_version":1`, `"schema_version":1,"unknown":1`, 1)
			case "trailing":
				s += "{}"
			case "oversized":
				s = strings.Repeat(" ", maxRecordBytes+1)
			case "native_namespace":
				s = strings.Replace(s, `"owner":"core-activation"`, `"owner":"other-owner"`, 1)
			case "false_verified":
				s = strings.ReplaceAll(strings.ReplaceAll(s, `"phase":"prepared"`, `"phase":"verified"`), `"fence":"ready"`, `"fence":"revoked"`)
			case "empty":
				s = ""
			case "symlink":
				other := e.Store.path + ".foreign"
				if err := os.Rename(e.Store.path, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, e.Store.path); err != nil {
					t.Fatal(err)
				}
			}
			if mutation != "symlink" {
				if err := os.WriteFile(e.Store.path, []byte(s), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			f.calls = nil
			if _, _, err := e.Recover(context.Background()); err == nil {
				t.Fatal("unsafe record accepted")
			}
			if len(f.calls) != 0 {
				t.Fatalf("effects from unsafe metadata: %v", f.calls)
			}
		})
	}
}

func TestAdmissionLostCommitCannotBecomeAnotherOperation(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprintf("visible_%t", visible), func(t *testing.T) {
			e, r, _, f := fixture(t)
			original := e.Store.write
			e.Store.write = func(path string, data []byte, mode os.FileMode) error {
				if visible {
					if err := original(path, data, mode); err != nil {
						return err
					}
				}
				return errors.New("acknowledgement lost")
			}
			if _, err := e.Begin(context.Background(), r, false); err == nil {
				t.Fatal("expected commit error")
			}
			e.Store.write = original
			f.calls = nil
			if _, err := e.Begin(context.Background(), r, false); err == nil {
				t.Fatal("ambiguous admission replayed")
			}
			r.ID = strings.Repeat("7", 32)
			r.WorkerIntent.AttemptID = r.ID
			if _, err := e.Begin(context.Background(), r, false); err == nil {
				t.Fatal("unresolved store replaced by different request")
			}
			if len(f.calls) != 0 {
				t.Fatalf("effects after ambiguous admission: %v", f.calls)
			}
		})
	}
}

func TestRecordRoundTripIncludesNativeClaimAndRestorationIntent(t *testing.T) {
	e, r, ref, f := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	f.fail["verify"] = errors.New("not healthy")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("expected failure")
	}
	f.restoreLost = true
	if _, _, err := e.Recover(context.Background()); err == nil {
		t.Fatal("expected lost response")
	}
	got, exists, err := e.Store.Current()
	if err != nil || !exists || !got.RestorationRequired || got.WorkerRef == nil || got.Fence != Revoked {
		t.Fatalf("incomplete restoration record: %+v %v", got, err)
	}
	data, _ := json.Marshal(got)
	if strings.Contains(string(data), "not healthy") {
		t.Fatal("raw child diagnostic retained")
	}
}

func TestFrozenStoreDirectoryIsHomeBoundedAndSourceIndependent(t *testing.T) {
	home := t.TempDir()
	for _, directory := range []string{"relative/core-activation", filepath.Join(home, "..", "outside", "core-activation"), home, filepath.Join(home, "other")} {
		if _, err := NewStoreAt(home, directory); err == nil {
			t.Fatalf("unsafe frozen store accepted: %s", directory)
		}
	}
	directory := filepath.Join(home, "frozen", "core-activation")
	s, err := NewStoreAt(home, directory)
	if err != nil {
		t.Fatal(err)
	}
	if s.Directory() != directory {
		t.Fatal("frozen store location changed")
	}
	if _, exists, err := s.Current(); err != nil || exists {
		t.Fatalf("source-independent empty store: exists=%t err=%v", exists, err)
	}
}

func TestOrdinaryMutationCannotCrossAnUnresolvedRootOperation(t *testing.T) {
	e, r, _, _ := fixture(t)
	admit(t, e, r)
	before, err := os.ReadFile(e.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Store.GuardOrdinaryMutation(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatalf("active root operation did not gate ordinary mutation: %v", err)
	}
	after, err := os.ReadFile(e.Store.path)
	if err != nil || string(after) != string(before) {
		t.Fatal("ordinary mutation guard rewrote operation state")
	}
	current, _, _ := e.Store.Current()
	current.Phase = FailedPreserved
	current.Fence = Revoked
	current.FailureClass = "deadline"
	current.FailedReleases = []string{r.CandidateSHA}
	current.FailedBuilds = []string{r.CandidateBuild}
	if err := e.Store.put(&current); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.GuardOrdinaryMutation(context.Background()); err != nil {
		t.Fatalf("unsubmitted terminal operation blocked mutation: %v", err)
	}
}
