package evidence

import (
	"context"
	"errors"
	"github.com/vrooli/freshness-go/treedigest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Only this disposable controlled-writer fixture initializes the protocol.
// Temp-directory permissions do NOT prove custody against ordinary host writers.
func custodyFixture(t *testing.T) (*productCustody, string, string, *treedigest.ManifestRequest) {
	t.Helper()
	primary, dependency := t.TempDir(), t.TempDir()
	source, shared := filepath.Join(primary, "main.go"), filepath.Join(dependency, "types.go")
	for _, p := range []string{source, shared} {
		if err := os.WriteFile(p, []byte("original bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := &treedigest.ManifestRequest{Primary: treedigest.RootSpec{Name: "scenario", Path: primary, Files: []string{"main.go"}}, Dependencies: []treedigest.RootSpec{{Name: "shared", Path: dependency, Files: []string{"types.go"}}}, Configuration: map[string]string{"policy": "strict"}, Toolchain: map[string]string{"go": "fixture-1"}}
	capture := func(ctx context.Context) (treedigest.InputManifest, error) {
		return treedigest.BuildInputManifestContext(ctx, *request)
	}
	m, err := capture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return &productCustody{capture: capture, epoch: 1, identity: m.Identity, deadline: time.Now().Add(time.Minute)}, source, shared, request
}
func TestDevelopmentProductCustodyHoldsWriterThroughPublication(t *testing.T) {
	o, source, _, _ := custodyFixture(t)
	epoch, id := o.epoch, o.identity
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- o.publish(context.Background(), epoch, id, func(check func() error) error { close(entered); <-release; return check() })
	}()
	<-entered
	before, _ := os.ReadFile(source)
	called := false
	if o.mutate(context.Background(), epoch, func() error { called = true; return os.WriteFile(source, []byte("forbidden concurrent write"), 0600) }) == nil || called {
		t.Fatal("concurrent managed writer crossed commit fence")
	}
	after, _ := os.ReadFile(source)
	if string(before) != string(after) {
		t.Fatal("refused mutation changed source")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := o.mutate(context.Background(), epoch, func() error { return os.WriteFile(source, []byte("permitted later bytes"), 0600) }); err != nil {
		t.Fatal(err)
	}
	committed := false
	if o.publish(context.Background(), epoch, id, func(check func() error) error { committed = true; return check() }) == nil || committed {
		t.Fatal("stale epoch/content published")
	}
}
func TestDevelopmentProductCustodyFullClosureRefusal(t *testing.T) {
	for _, kind := range []string{"primary", "dependency", "configuration", "toolchain", "required-missing", "symlink", "expired", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			o, source, shared, r := custodyFixture(t)
			ctx := context.Background()
			switch kind {
			case "primary":
				os.WriteFile(source, []byte("changed"), 0600)
			case "dependency":
				os.WriteFile(shared, []byte("changed"), 0600)
			case "configuration":
				r.Configuration["policy"] = "weaker"
			case "toolchain":
				r.Toolchain["go"] = "other"
			case "required-missing":
				os.Remove(shared)
			case "symlink":
				os.Remove(shared)
				os.Symlink(source, shared)
			case "expired":
				o.deadline = time.Now().Add(-time.Second)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			called := false
			if o.publish(ctx, o.epoch, o.identity, func(check func() error) error { called = true; return check() }) == nil || called {
				t.Fatal("changed/unavailable complete inputs reached disposition")
			}
		})
	}
}
func TestDevelopmentProductCustodyDeadlineGuardReleaseAndUnknownCommit(t *testing.T) {
	o, _, _, _ := custodyFixture(t)
	originalDeadline := o.deadline
	var escaped func() error
	renamed := false
	err := o.publish(context.Background(), o.epoch, o.identity, func(check func() error) error {
		escaped = check
		o.deadline = time.Now().Add(-time.Second)
		if e := check(); e != nil {
			return e
		}
		renamed = true
		return nil
	})
	if err == nil || renamed {
		t.Fatal("expired lease crossed pre-publication guard")
	}
	if escaped() == nil {
		t.Fatal("released lease remained valid")
	}
	o.deadline = originalDeadline
	flushError := errors.New("fixture post-rename directory flush failure")
	if e := o.publish(context.Background(), o.epoch, o.identity, func(check func() error) error {
		if e := check(); e != nil {
			return e
		}
		renamed = true
		return flushError
	}); !errors.Is(e, flushError) || !renamed {
		t.Fatal("unknown post-publication failure mislabeled unchanged")
	}
	if e := o.publish(context.Background(), o.epoch, o.identity, func(check func() error) error { return check() }); e != nil {
		t.Fatal("error leaked owner lock", e)
	}
}
func TestDevelopmentProductCustodyFailedMutationPoisonsOwner(t *testing.T) {
	o, source, _, _ := custodyFixture(t)
	failure := errors.New("fixture mutation failed after write")
	if err := o.mutate(context.Background(), o.epoch, func() error { os.WriteFile(source, []byte("partial"), 0600); return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	called := false
	if o.publish(context.Background(), o.epoch, o.identity, func(check func() error) error { called = true; return check() }) == nil || called {
		t.Fatal("partial mutation restored old authority")
	}
	var absent *productCustody
	if absent.publish(context.Background(), 1, "identity", func(func() error) error { return nil }) != errProductCustodyUnavailable {
		t.Fatal("absent custody accepted")
	}
}

func TestDevelopmentProductCustodyEscapedGuardRacesReleaseAndMutation(t *testing.T) {
	o, source, _, _ := custodyFixture(t)
	entered, release, result := make(chan func() error, 1), make(chan struct{}), make(chan error, 1)
	epoch, id := o.epoch, o.identity
	go func() {
		result <- o.publish(context.Background(), epoch, id, func(check func() error) error {
			if err := check(); err != nil {
				return err
			}
			entered <- check
			<-release
			return nil
		})
	}()
	guard := <-entered
	checked := make(chan struct{})
	go func() {
		for range 1000 {
			_ = guard()
		}
		close(checked)
	}()
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if err := o.mutate(context.Background(), epoch, func() error { return os.WriteFile(source, []byte("new managed epoch"), 0600) }); err != nil {
		t.Fatal(err)
	}
	<-checked
	if guard() == nil {
		t.Fatal("released guard retained authority")
	}
}
