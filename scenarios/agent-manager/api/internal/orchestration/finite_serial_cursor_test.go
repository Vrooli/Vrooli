// finite_serial_cursor_test.go uses synthetic repository reads and disposable authority.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"context"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	"sync"
	"testing"
	"time"
)

type cursorReadSnapshot struct {
	offset, limit int
	from, to      time.Time
	status        domain.RunStatus
}
type cursorFixtureRuns struct {
	repository.RunRepository
	rows             []*domain.Run
	calls            []cursorReadSnapshot
	fail             bool
	entered, release chan struct{}
}

func (r *cursorFixtureRuns) List(ctx context.Context, f repository.RunListFilter) ([]*domain.Run, error) {
	if f.EndedFrom == nil || f.EndedTo == nil || f.Status == nil {
		panic("missing bounded terminal cursor")
	}
	r.calls = append(r.calls, cursorReadSnapshot{f.Offset, f.Limit, *f.EndedFrom, *f.EndedTo, *f.Status})
	if r.entered != nil {
		r.entered <- struct{}{}
		select {
		case <-r.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if r.fail {
		r.fail = false
		return nil, errors.New("disposable read refused")
	}
	if f.Offset >= len(r.rows) {
		return nil, nil
	}
	end := f.Offset + f.Limit
	if end > len(r.rows) {
		end = len(r.rows)
	}
	return r.rows[f.Offset:end], nil
}

type cursorFixtureEngine struct {
	effortauthority.Engine
	bindingReads int
}

func (e *cursorFixtureEngine) CheckBinding(ctx context.Context, b effortauthority.Binding) (effortauthority.Policy, error) {
	e.bindingReads++
	return e.Engine.CheckBinding(ctx, b)
}
func cursorFixture(t *testing.T) (*Orchestrator, *cursorFixtureRuns, *cursorFixtureEngine, *time.Time) {
	t.Helper()
	o, a, p, _, _, _, now := nativeEffortFixture(t)
	r := &cursorFixtureRuns{}
	engine := &cursorFixtureEngine{Engine: a}
	o.effortAuthority = engine
	o.runs = r
	for i := 0; i < 256; i++ {
		r.rows = append(r.rows, &domain.Run{})
	}
	b := effortauthority.Binding{PolicyID: p.ID, PolicyDigest: effortauthority.Digest(p), Epoch: p.Epoch, Owner: p.Owner, Client: p.Client, Deadline: p.Deadline}
	r.rows = append(r.rows, &domain.Run{ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{Effort: &b}}})
	return o, r, engine, now
}
func TestFiniteSerialCursorReachesSourceBeyond256AndKeepsOriginalWindow(t *testing.T) {
	o, r, engine, now := cursorFixture(t)
	initial := *now
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if engine.bindingReads != 0 || len(r.calls) != 1 || r.calls[0].offset != 0 || o.serialScanOffset != 256 {
		t.Fatal("first bounded page")
	}
	*now = now.Add(time.Minute)
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if engine.bindingReads != 1 || len(r.calls) != 2 || r.calls[1].offset != 256 {
		t.Fatal("older finite source starved")
	}
	for _, c := range r.calls {
		if c.limit != 256 || !c.to.Equal(initial) || !c.from.Equal(initial.Add(-8*time.Hour)) || c.status != domain.RunStatusComplete {
			t.Fatal("sweep window changed", c)
		}
	}
	if !o.serialScanThrough.IsZero() || o.serialScanOffset != 0 {
		t.Fatal("completed sweep cursor not released")
	}
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.calls[2].to.Equal(*now) || r.calls[2].offset != 0 {
		t.Fatal("new sweep does not capture current window")
	}
}
func TestFiniteSerialCursorReadFailureRetriesSameWindowAndRestartIsNotAuthority(t *testing.T) {
	o, r, _, now := cursorFixture(t)
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	original := r.calls[0].to
	r.fail = true
	*now = now.Add(time.Minute)
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err == nil {
		t.Fatal("read failure hidden")
	}
	if o.serialScanOffset != 256 || !o.serialScanThrough.Equal(original) {
		t.Fatal("failed read advanced cursor")
	}
	if err := o.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.calls[1] != r.calls[2] {
		t.Fatal("retry changed exact read bounds")
	}
	restarted := &Orchestrator{runs: r, effortAuthority: o.effortAuthority, finiteNativeFactory: o.finiteNativeFactory, clock: o.clock}
	if err := restarted.ReconcileFiniteSerialEpisodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.calls[3].offset != 0 || !r.calls[3].to.Equal(*now) {
		t.Fatal("restart inherited nonauthority cursor")
	}
}
func TestFiniteSerialCursorConcurrentOwnerCyclesSerializeBeforeRepositoryRead(t *testing.T) {
	o, r, _, _ := cursorFixture(t)
	r.rows = nil
	r.entered = make(chan struct{}, 2)
	r.release = make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(1)
	go func() { defer wg.Done(); errs <- o.ReconcileFiniteSerialEpisodes(context.Background()) }()
	<-r.entered
	if o.serialScanMu.TryLock() {
		o.serialScanMu.Unlock()
		t.Fatal("owner read not serialized")
	}
	wg.Add(1)
	go func() { defer wg.Done(); errs <- o.ReconcileFiniteSerialEpisodes(context.Background()) }()
	close(r.release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(r.calls) != 2 || r.calls[0].offset != 0 || r.calls[1].offset != 0 {
		t.Fatal("serialized sweeps changed cursor")
	}
}
