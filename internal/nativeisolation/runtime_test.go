package nativeisolation

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

type fixtureChild struct{ wait error }

func (f *fixtureChild) PID() int          { return 99999 }
func (f *fixtureChild) Stdout() io.Reader { return strings.NewReader("disposable-native-output") }
func (f *fixtureChild) Stderr() io.Reader { return strings.NewReader("") }
func (f *fixtureChild) Wait() error       { return f.wait }

type fixtureDriver struct {
	started   int
	shows     int
	terminal  bool
	failStart bool
	signals   []string
}

func (d *fixtureDriver) Start(context.Context, Plan, io.Reader) (child, error) {
	d.started++
	if d.failStart {
		return nil, errors.New("unknown manager reply")
	}
	return &fixtureChild{}, nil
}
func (d *fixtureDriver) Show(context.Context, string) (map[string]string, error) {
	d.shows++
	if d.shows <= 2 {
		return map[string]string{"LoadState": "not-found"}, nil
	}
	if d.terminal {
		return map[string]string{"LoadState": "loaded", "SubState": "exited", "TasksCurrent": "0", "MainPID": "0", "ExecMainCode": "1", "Result": "success"}, nil
	}
	return map[string]string{"LoadState": "loaded", "SubState": "running", "TasksCurrent": "2", "MainPID": "0"}, nil
}
func (d *fixtureDriver) Signal(_ context.Context, _ string, sig string) error {
	d.signals = append(d.signals, sig)
	return nil
}
func runtimeFixture(t *testing.T, d *fixtureDriver) (*Runtime, Request) {
	t.Helper()
	_, rt, r, _ := unitOwnerFixture(t, d)
	return rt, r
}
func TestRefusalBeforeManagerForDisabledAndUnqualifiedHost(t *testing.T) {
	d := &fixtureDriver{}
	rt, r := runtimeFixture(t, d)
	rt.qualify = func(context.Context) error { return ErrRefused }
	if _, e := rt.Start(context.Background(), r, nil); e == nil || d.started != 0 || d.shows != 0 {
		t.Fatal("unqualified host caused manager effects")
	}
	rt.manifest.Enabled = false
	if _, e := rt.Start(context.Background(), r, nil); e == nil || d.started != 0 {
		t.Fatal("disabled route executed")
	}
}
func TestOwnedUnitTerminalAndUnknownDescendants(t *testing.T) {
	for _, terminal := range []bool{true, false} {
		t.Run(map[bool]string{true: "terminal", false: "descendants-live"}[terminal], func(t *testing.T) {
			d := &fixtureDriver{terminal: terminal}
			rt, r := runtimeFixture(t, d)
			p, e := rt.Start(context.Background(), r, nil)
			if e != nil {
				t.Fatal(e)
			}
			if p.PID() == 99999 {
				t.Fatal("wrapper PID reported as native")
			}
			e = p.Wait()
			if terminal && e != nil {
				t.Fatal(e)
			}
			if !terminal && !errors.Is(e, ErrUnknown) {
				t.Fatal("live descendants treated as terminal")
			}
			rt.mu.Lock()
			busy := rt.busy
			rt.mu.Unlock()
			if busy == terminal {
				t.Fatal("occupancy was released without exact terminal evidence")
			}
		})
	}
}
func TestManagerLostReplyKeepsOccupancyAndUnitIdentity(t *testing.T) {
	d := &fixtureDriver{failStart: true}
	rt, r := runtimeFixture(t, d)
	p, e := rt.Start(context.Background(), r, nil)
	if e != nil || !errors.Is(p.Wait(), ErrUnknown) {
		t.Fatal("lost reply incorrectly proves no execution")
	}
	if p.PID() != 0 || !rt.busy {
		t.Fatal("unknown launch had PID/freed slot")
	}
	p.Kill()
	if len(d.signals) != 1 || d.signals[0] != "SIGKILL" {
		t.Fatal("kill did not address exact owned unit")
	}
}

type fencedDriver struct {
	mu    sync.Mutex
	units map[string]chan struct{}
}
type blockedChild struct{ done chan struct{} }

func (c *blockedChild) PID() int          { return 99999 }
func (c *blockedChild) Stdout() io.Reader { return strings.NewReader("") }
func (c *blockedChild) Stderr() io.Reader { return strings.NewReader("") }
func (c *blockedChild) Wait() error       { <-c.done; return nil }
func (d *fencedDriver) Start(_ context.Context, p Plan, _ io.Reader) (child, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan struct{})
	d.units[p.Unit] = ch
	return &blockedChild{ch}, nil
}
func (d *fencedDriver) Show(_ context.Context, u string) (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.units[u] == nil {
		return map[string]string{"LoadState": "not-found"}, nil
	}
	return map[string]string{"LoadState": "loaded", "SubState": "exited", "MainPID": "0", "TasksCurrent": "0", "ExecMainCode": "1", "Result": "success"}, nil
}
func (d *fencedDriver) Signal(context.Context, string, string) error { return nil }
func TestLateWrapperCannotReleaseSuccessorOccupancy(t *testing.T) {
	base, r := runtimeFixture(t, &fixtureDriver{})
	d := &fencedDriver{units: map[string]chan struct{}{}}
	base.unitOwner.driver = d
	a, e := base.Start(context.Background(), r, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = base.Terminal(context.Background(), r.RunID); e != nil {
		t.Fatal(e)
	}
	r.RunID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	r.ReservationKey = "second-receipt"
	addUnitOwnerReceipt(base.unitOwner.authority.(*unitOwnerFixtureEngine), r)
	b, e := base.Start(context.Background(), r, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { d.mu.Lock(); ch := d.units[b.unit]; d.mu.Unlock(); close(ch); b.Wait() }()
	d.mu.Lock()
	ch := d.units[a.unit]
	d.mu.Unlock()
	close(ch)
	if e = a.Wait(); e != nil {
		t.Fatal(e)
	}
	base.mu.Lock()
	busy, unit := base.busy, base.occupiedUnit
	base.mu.Unlock()
	if !busy || unit != b.unit {
		t.Fatal("old wrapper cleared successor occupancy")
	}
	r.RunID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	r.ReservationKey = "third-receipt"
	addUnitOwnerReceipt(base.unitOwner.authority.(*unitOwnerFixtureEngine), r)
	if _, e = base.Start(context.Background(), r, nil); e == nil {
		t.Fatal("third concurrent launch accepted")
	}
}

func TestShowParserRejectsDuplicateEmptyAndUnknownProperties(t *testing.T) {
	valid := "LoadState=loaded\nMainPID=0\nSubState=exited\nTasksCurrent=0\nExecMainCode=1\nResult=success\n"
	if state, e := parseShow([]byte(valid)); e != nil || !terminalState(state) {
		t.Fatal("valid terminal refused", e)
	}
	for _, invalid := range []string{"LoadState=\n" + valid, valid + "Unexpected=0\n", strings.Replace(valid, "TasksCurrent=0\n", "", 1), strings.Replace(valid, "Result=success", "Result=\nResult=success", 1)} {
		if _, e := parseShow([]byte(invalid)); !errors.Is(e, ErrUnknown) {
			t.Fatal("ambiguous manager metadata accepted", e)
		}
	}
}
