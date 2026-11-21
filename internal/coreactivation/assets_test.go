package coreactivation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

func frozenFixture(t *testing.T, e *Engine, r Record) (Record, Layout, Assets) {
	return frozenFixtureWithWorker(t, e, r, "")
}

func frozenFixtureWithWorker(t *testing.T, e *Engine, r Record, worker string) (Record, Layout, Assets) {
	t.Helper()
	ctx := context.Background()
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	l := Layout{Home: r.Home, Executable: filepath.Join(r.Home, "bin", "vrooli"), UnitName: unit.NativeName("linux"), UnitPath: platform.RuntimeSupervisorUnitPath("linux", r.Home), LogPath: filepath.Join(r.Home, "logs", "supervisor.log")}
	if err := os.MkdirAll(r.SourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"candidate": "candidate-image", "prior": "prior-image", "worker": "worker-image"} {
		if err := os.WriteFile(filepath.Join(r.SourceRoot, name), []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	inputs := InputSources{Candidate: filepath.Join(r.SourceRoot, "candidate"), Prior: filepath.Join(r.SourceRoot, "prior"), Worker: filepath.Join(r.SourceRoot, "worker"), CandidateUnit: []byte("candidate-unit\n"), PriorUnit: []byte("prior-unit\n")}
	if worker != "" {
		inputs.Worker = worker
	}
	var d InputDigests
	if err := e.Store.Prepare(ctx, func(p *Preparer) error { var err error; d, err = p.Freeze(ctx, r.ID, l, inputs); return err }); err != nil {
		t.Fatal(err)
	}
	r.LayoutSHA, r.WorkerSHA = d.Layout, d.Worker
	r.CandidateSHA, r.PriorSHA = d.Candidate, d.Prior
	r.CandidateUnitSHA, r.PriorUnitSHA = d.CandidateUnit, d.PriorUnit
	if err := config.WriteOwnedFileDurable(l.Executable, []byte("prior-image"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(l.UnitPath, inputs.PriorUnit, 0o644); err != nil {
		t.Fatal(err)
	}
	return r, l, Assets{Store: e.Store}
}

func TestFrozenInputsSurviveSourceLossAndRefuseChangedMetadata(t *testing.T) {
	e, r, _, _ := fixture(t)
	r, l, a := frozenFixture(t, e, r)
	if err := os.RemoveAll(r.SourceRoot); err != nil {
		t.Fatal(err)
	}
	got, err := a.Layout(r)
	if err != nil || got != l {
		t.Fatalf("frozen layout required source: %+v %v", got, err)
	}
	for _, name := range []string{"candidate", "prior", "worker"} {
		if _, err := a.Image(context.Background(), r, name); err != nil {
			t.Fatalf("frozen %s required source: %v", name, err)
		}
	}
	for _, candidate := range []bool{false, true} {
		if _, err := a.Definition(r, candidate); err != nil {
			t.Fatal(err)
		}
	}
	path, _ := a.path(r, "layout.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), l.Executable, filepath.Join(r.Home, "other", "vrooli"), 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Layout(r); err == nil {
		t.Fatal("changed layout was trusted for recovery")
	}
}

func TestFrozenImagesRejectAliasesCorruptionAndOversize(t *testing.T) {
	for _, failure := range []string{"symlink", "empty", "changed", "oversized"} {
		t.Run(failure, func(t *testing.T) {
			e, r, _, _ := fixture(t)
			r, _, a := frozenFixture(t, e, r)
			path, _ := a.path(r, "candidate")
			switch failure {
			case "symlink":
				other := path + ".foreign"
				if err := os.Rename(path, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			case "empty":
				if err := os.Truncate(path, 0); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(path, []byte("foreign"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.Truncate(path, imageLimit+1); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := a.Image(context.Background(), r, "candidate"); err == nil {
				t.Fatal("invalid frozen image accepted")
			}
		})
	}
}

type assetFixtureEffects struct {
	*fakeEffects
	assets Assets
}

func (f assetFixtureEffects) PublishCandidate(ctx context.Context, r Record) error {
	if err := f.assets.PublishBinary(ctx, r, false); err != nil {
		return err
	}
	return f.fakeEffects.PublishCandidate(ctx, r)
}

func (f assetFixtureEffects) ActivateCandidate(ctx context.Context, r Record) error {
	if err := f.assets.PublishDefinition(ctx, r, false); err != nil {
		return err
	}
	return f.fakeEffects.ActivateCandidate(ctx, r)
}

func (f assetFixtureEffects) VerifyPrior(ctx context.Context, r Record) error {
	l, err := f.assets.Layout(r)
	if err != nil {
		return err
	}
	sha, err := imageDigest(ctx, l.Executable)
	if err != nil || sha != r.PriorSHA {
		return ErrConflict
	}
	data, err := readFrozen(l.UnitPath, definitionLimit)
	if err != nil || bytesDigest(data) != r.PriorUnitSHA {
		return ErrConflict
	}
	return nil
}

func (f assetFixtureEffects) RestorePrior(ctx context.Context, r Record) error {
	if err := f.assets.PublishBinary(ctx, r, true); err != nil {
		return err
	}
	if err := f.assets.PublishDefinition(ctx, r, true); err != nil {
		return err
	}
	return f.fakeEffects.RestorePrior(ctx, r)
}

func (f assetFixtureEffects) RetireInputs(_ context.Context, r Record) error {
	return f.assets.Retire(r)
}

func TestFrozenPublicationAndRestorationUseExactBinaryAndDefinition(t *testing.T) {
	e, r, ref, f := fixture(t)
	r, l, a := frozenFixture(t, e, r)
	effects := assetFixtureEffects{fakeEffects: f, assets: a}
	e.Effects = effects
	admit(t, e, r)
	dispatch(t, e, r)
	if err := a.PublishBinary(context.Background(), r, false); !errors.Is(err, ErrClaimed) {
		t.Fatalf("publication before native claim: %v", err)
	}
	f.fail["verify"] = errors.New("candidate health rejected")
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
		t.Fatal("expected failed health")
	}
	if data, err := os.ReadFile(l.Executable); err != nil || string(data) != "candidate-image" {
		t.Fatal("candidate image not published")
	}
	if data, err := os.ReadFile(l.UnitPath); err != nil || string(data) != "candidate-unit\n" {
		t.Fatal("candidate definition not published")
	}
	if err := os.RemoveAll(r.SourceRoot); err != nil {
		t.Fatal(err)
	}
	got, _, err := e.Recover(context.Background())
	if err != nil || got.Phase != FailedRestored {
		t.Fatalf("frozen restoration failed: %s %v", got.Phase, err)
	}
	if err := effects.VerifyPrior(context.Background(), got); err == nil {
		t.Fatal("retired inputs unexpectedly still readable")
	}
	if data, err := os.ReadFile(l.Executable); err != nil || string(data) != "prior-image" {
		t.Fatal("wrong restored image")
	}
	if data, err := os.ReadFile(l.UnitPath); err != nil || string(data) != "prior-unit\n" {
		t.Fatal("wrong restored definition")
	}
}

func TestPublicationRefusesUnexpectedServingBytes(t *testing.T) {
	for _, target := range []string{"binary", "definition"} {
		t.Run(target, func(t *testing.T) {
			e, r, ref, f := fixture(t)
			r, l, a := frozenFixture(t, e, r)
			e.Effects = assetFixtureEffects{fakeEffects: f, assets: a}
			admit(t, e, r)
			dispatch(t, e, r)
			path := l.Executable
			if target == "definition" {
				path = l.UnitPath
			}
			if err := os.WriteFile(path, []byte("foreign-owner"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := e.RunWorker(context.Background(), r.ID, ref); err == nil {
				t.Fatal("unexpected serving bytes replaced")
			}
			if target == "definition" {
				if data, err := os.ReadFile(l.Executable); err != nil || string(data) != "prior-image" {
					t.Fatal("foreign definition did not prevent binary publication")
				}
			}
			if _, _, err := e.Recover(context.Background()); err == nil {
				t.Fatal("restoration replaced unexpected owner bytes")
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "foreign-owner" {
				t.Fatal("foreign bytes changed")
			}
		})
	}
}

func TestPreparationCapabilityExpiresAndPrivateInputRetentionIsBounded(t *testing.T) {
	e, r, _, _ := fixture(t)
	_, _, _ = frozenFixture(t, e, r)
	var retained *Preparer
	if err := e.Store.Prepare(context.Background(), func(p *Preparer) error { retained = p; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := retained.Freeze(context.Background(), r.ID, Layout{}, InputSources{}); err == nil {
		t.Fatal("expired preparation capability retained write authority")
	}
	// The prior unadmitted set is owned input, not a serving or admitted root.
	next := strings.Repeat("9", 32)
	if err := e.Store.pruneUnadmittedInputs(next, ""); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.Store.inputDirectory(r.ID)
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("orphaned input set not retired")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "foreign-file")
	if err := os.WriteFile(foreign, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.pruneUnadmittedInputs(next, ""); err == nil {
		t.Fatal("unexpected private content deleted")
	}
	if data, err := os.ReadFile(foreign); err != nil || string(data) != "keep" {
		t.Fatal("foreign private content changed")
	}
}

func TestInputRetirementReconfirmsDurabilityAfterVisibleRemoval(t *testing.T) {
	e, r, ref, f := fixture(t)
	r, _, a := frozenFixture(t, e, r)
	calls := 0
	a.confirmRemoval = func(parent string) error {
		calls++
		if calls == 1 {
			return errors.New("cleanup durability acknowledgement lost")
		}
		return confirmRemovedInputs(parent)
	}
	e.Effects = assetFixtureEffects{fakeEffects: f, assets: a}
	admit(t, e, r)
	dispatch(t, e, r)
	if _, err := e.RunWorker(context.Background(), r.ID, ref); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Recover(context.Background()); err == nil {
		t.Fatal("cleanup durability failure ignored")
	}
	dir, _ := e.Store.inputDirectory(r.ID)
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("fixture did not lose acknowledgement after visible removal")
	}
	if _, _, err := e.Recover(context.Background()); err != nil {
		t.Fatalf("repeat cleanup did not reconcile visible removal: %v", err)
	}
	if calls != 2 {
		t.Fatalf("visible removal accepted without fresh durability confirmation: calls=%d", calls)
	}
}
