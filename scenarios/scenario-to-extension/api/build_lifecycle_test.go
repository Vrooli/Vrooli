package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtensionBuildLifecycleTransitionMatrix(t *testing.T) {
	states := []ExtensionBuildStatus{
		ExtensionBuildBuilding, ExtensionBuildReady, ExtensionBuildFailed, ExtensionBuildCanceled,
	}
	want := map[ExtensionBuildStatus]map[ExtensionBuildStatus]bool{
		ExtensionBuildBuilding: {
			ExtensionBuildBuilding: true, ExtensionBuildReady: true,
			ExtensionBuildFailed: true, ExtensionBuildCanceled: true,
		},
		ExtensionBuildReady:    {ExtensionBuildReady: true},
		ExtensionBuildFailed:   {ExtensionBuildFailed: true},
		ExtensionBuildCanceled: {ExtensionBuildCanceled: true},
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				build := &ExtensionBuild{Status: from}
				err := build.TransitionStatus(to)
				if want[from][to] {
					if err != nil || build.Status != to {
						t.Fatalf("transition status=%s want=%s err=%v", build.Status, to, err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidBuildTransition) {
					t.Fatalf("transition %s -> %s error=%v, want invalid transition", from, to, err)
				}
				if build.Status != from {
					t.Fatalf("rejected transition mutated status to %s", build.Status)
				}
			})
		}
	}

	for _, status := range []ExtensionBuildStatus{"", "unknown"} {
		if err := (&ExtensionBuild{Status: ExtensionBuildBuilding}).TransitionStatus(status); !errors.Is(err, ErrInvalidBuildTransition) {
			t.Fatalf("invalid target %q error=%v", status, err)
		}
	}
	var nilBuild *ExtensionBuild
	if err := nilBuild.TransitionStatus(ExtensionBuildBuilding); !errors.Is(err, ErrInvalidBuildTransition) {
		t.Fatalf("nil build transition error=%v", err)
	}
}

func TestCanceledBuildRejectsStaleWorkerFinalization(t *testing.T) {
	manager := NewBuildManager()
	build := &ExtensionBuild{BuildID: "cancel-race", Status: ExtensionBuildBuilding}
	if err := manager.Add(build); err != nil {
		t.Fatal(err)
	}
	worker := cloneExtensionBuild(build)

	if !manager.Cancel(build.BuildID) {
		t.Fatal("manager did not accept cancellation")
	}
	if err := worker.Finish(ExtensionBuildReady, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := manager.Replace(build.BuildID, worker); !errors.Is(err, ErrInvalidBuildTransition) {
		t.Fatalf("stale worker replacement error=%v, want transition rejection", err)
	}
	stored, ok := manager.Get(build.BuildID)
	if !ok || stored.Status != ExtensionBuildCanceled || stored.CompletedAt == nil {
		t.Fatalf("cancellation lost to stale worker result: build=%+v", stored)
	}
}

func TestBuildManagerRejectsUnknownStatusAndPreservesInvalidStateFile(t *testing.T) {
	manager := NewBuildManager()
	if err := manager.Add(&ExtensionBuild{BuildID: "invalid-add", Status: ExtensionBuildStatus("unknown")}); !errors.Is(err, ErrInvalidBuildTransition) {
		t.Fatalf("unknown initial status error=%v", err)
	}

	path := filepath.Join(t.TempDir(), "builds.json")
	payload, err := json.Marshal([]*ExtensionBuild{{BuildID: "invalid-load", Status: ExtensionBuildStatus("unknown")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := NewBuildManager(path)
	if _, ok := loaded.Get("invalid-load"); ok {
		t.Fatal("manager accepted unknown persisted status")
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(persisted) != string(payload) {
		t.Fatal("load-time validation modified the persisted build data")
	}
}
