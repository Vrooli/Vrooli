package sessions

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	_ "modernc.org/sqlite"
)

type desktopAuthority struct{ denied bool }

func (a *desktopAuthority) Authorize(context.Context, DesktopLease, string) error {
	if a.denied {
		return ErrDesktopAdmission
	}
	return nil
}

type desktopNative struct {
	effects                    int
	invalid, fail, releaseFail bool
}

func (n *desktopNative) Validate(context.Context, DesktopCommand) error {
	if n.invalid {
		return ErrDesktopAdmission
	}
	return nil
}
func (n *desktopNative) Apply(context.Context, DesktopCommand) error {
	n.effects++
	if n.fail {
		return errors.New("connection lost after effect")
	}
	return nil
}
func (n *desktopNative) ReleaseHeld(context.Context) error {
	if n.releaseFail {
		return errors.New("input release failed")
	}
	return nil
}

func desktopFixture(t *testing.T) (*DesktopController, *SQLiteDesktopRepository, *desktopNative, *desktopAuthority, DesktopLease) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "sessions.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repo, err := NewSQLiteDesktopRepository(context.Background(), db, "host/os-login")
	require.NoError(t, err)
	native, auth := &desktopNative{}, &desktopAuthority{}
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	controller, err := NewDesktopController(repo, auth, native, surface, "os-login", "helper-generation-1")
	require.NoError(t, err)
	lease := DesktopLease{Ref: targetmodel.SessionRef{Surface: surface, SessionID: "session-1", DesktopSessionID: "os-login"}, Actor: "actor", HelperID: "helper-generation-1", ExpiresAt: time.Now().Add(time.Minute), Control: true}
	lease, err = controller.Open(context.Background(), lease, 0, false)
	require.NoError(t, err)
	return controller, repo, native, auth, lease
}
func desktopCommand(lease DesktopLease) DesktopCommand {
	return DesktopCommand{Lease: lease, ID: "command-1", GeometryRevision: "geometry-1", Payload: []byte(`{"kind":"click","x":1,"y":2}`)}
}

func TestDesktopReceiptsSurviveControllerReconstruction(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-05]
	c, repo, native, auth, lease := desktopFixture(t)
	command := desktopCommand(lease)
	first, err := c.Act(context.Background(), command)
	require.NoError(t, err)
	require.Equal(t, "applied", first.Outcome)
	reconstructed, err := NewDesktopController(repo, auth, native, c.surface, c.desktopID, c.helperID)
	require.NoError(t, err)
	second, err := reconstructed.Act(context.Background(), command)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, native.effects)
	command.Payload = []byte(`{"kind":"click","x":100,"y":200}`)
	_, err = reconstructed.Act(context.Background(), command)
	require.ErrorContains(t, err, "conflicting command identity")
	require.Equal(t, 1, native.effects)
}

func TestDesktopUncertainEffectsAreNotRepeated(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-07]
	c, _, native, _, lease := desktopFixture(t)
	native.fail = true
	receipt, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	native.fail = false
	receipt, err = c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	require.Equal(t, 1, native.effects)
}

func TestDesktopAdmissionRejectsWrongDestinationAndAuthority(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-01] [REQ:DEVICECONTROL-EVERYWHERE-AUTH-08]
	for _, kind := range []string{"target", "os-session", "helper", "actor", "observation-only", "expired", "revoked", "geometry"} {
		t.Run(kind, func(t *testing.T) {
			c, _, native, auth, lease := desktopFixture(t)
			switch kind {
			case "target":
				lease.Ref.Surface.Target.ResourceID = "other"
			case "os-session":
				lease.Ref.DesktopSessionID = "other"
			case "helper":
				lease.HelperID = "other"
			case "actor":
				lease.Actor = "other"
			case "observation-only":
				lease.Control = false
			case "expired":
				c.now = func() time.Time { return lease.ExpiresAt.Add(time.Second) }
			case "revoked":
				auth.denied = true
			case "geometry":
				native.invalid = true
			}
			_, err := c.Act(context.Background(), desktopCommand(lease))
			require.ErrorIs(t, err, ErrDesktopAdmission)
			require.Zero(t, native.effects)
		})
	}
}

func TestDesktopTakeoverFencesPriorEpochAndStop(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-04]
	c, _, native, _, lease := desktopFixture(t)
	next := lease
	next.Ref.SessionID = "session-2"
	next.Actor = "new-actor"
	_, err := c.Open(context.Background(), next, lease.Epoch, false)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	next, err = c.Open(context.Background(), next, lease.Epoch, true)
	require.NoError(t, err)
	require.Greater(t, next.Epoch, lease.Epoch)
	_, err = c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.ErrorIs(t, c.Stop(context.Background(), lease), ErrDesktopAdmission)
	_, err = c.Act(context.Background(), desktopCommand(next))
	require.NoError(t, err)
	require.Equal(t, 1, native.effects)
	native.releaseFail = true
	require.Error(t, c.Stop(context.Background(), next))
	command := desktopCommand(next)
	command.ID = "after-stop"
	_, err = c.Act(context.Background(), command)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Equal(t, 1, native.effects)
}

// A failed completion commit simulates a crash boundary: the admission remains
// durable even though the external effect cannot be rolled back.
type failingCompletion struct {
	DesktopRepository
	calls int
}

func (r *failingCompletion) Update(ctx context.Context, change func(*DesktopState) error) error {
	r.calls++
	if r.calls == 2 {
		return r.DesktopRepository.Update(ctx, func(s *DesktopState) error {
			if err := change(s); err != nil {
				return err
			}
			return errors.New("commit failed")
		})
	}
	return r.DesktopRepository.Update(ctx, change)
}
func TestDesktopFailedReceiptCommitLeavesDurableUnknown(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-07]
	c, repo, native, _, lease := desktopFixture(t)
	c.repo = &failingCompletion{DesktopRepository: repo}
	receipt, err := c.Act(context.Background(), desktopCommand(lease))
	require.Error(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	c.repo = repo
	receipt, err = c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	require.Equal(t, 1, native.effects)
}

func TestDesktopReceiptSurvivesDatabaseReopen(t *testing.T) {
	c, repo, native, auth, lease := desktopFixture(t)
	_, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	var seq int
	var name, path string
	require.NoError(t, repo.db.QueryRow(`PRAGMA database_list`).Scan(&seq, &name, &path))
	require.NoError(t, repo.db.Close())
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	reopened, err := NewSQLiteDesktopRepository(context.Background(), db, repo.destination)
	require.NoError(t, err)
	c, err = NewDesktopController(reopened, auth, native, c.surface, c.desktopID, c.helperID)
	require.NoError(t, err)
	receipt, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Outcome)
	require.Equal(t, 1, native.effects)
}

type afterAdmission struct {
	DesktopRepository
	once bool
	hook func()
}

func (r *afterAdmission) Update(ctx context.Context, change func(*DesktopState) error) error {
	err := r.DesktopRepository.Update(ctx, change)
	if err == nil && !r.once {
		r.once = true
		r.hook()
	}
	return err
}
func TestDesktopTakeoverBetweenAdmissionAndEffect(t *testing.T) {
	c, repo, native, auth, lease := desktopFixture(t)
	successor, err := NewDesktopController(repo, auth, native, c.surface, c.desktopID, c.helperID)
	require.NoError(t, err)
	c.repo = &afterAdmission{DesktopRepository: repo, hook: func() {
		next := lease
		next.Ref.SessionID = "successor"
		_, err := successor.Open(context.Background(), next, lease.Epoch, true)
		require.NoError(t, err)
	}}
	_, err = c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
}

type lateRevokingNative struct {
	desktopNative
	auth  *desktopAuthority
	calls int
}

func (n *lateRevokingNative) Apply(ctx context.Context, _ DesktopCommand) error {
	n.calls++
	n.auth.denied = true
	if err := CheckDesktopMutation(ctx); err != nil {
		return err
	}
	n.effects++
	return nil
}
func TestNativeMutationRechecksLateRevocation(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	native := &lateRevokingNative{auth: auth}
	c.native = native
	receipt, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", receipt.Outcome)
	require.Zero(t, native.effects)
	require.Equal(t, 1, native.calls)
	require.ErrorIs(t, CheckDesktopMutation(context.Background()), ErrDesktopAdmission)
	auth.denied = false
	duplicate, err := c.Act(context.Background(), desktopCommand(lease))
	require.NoError(t, err)
	require.Equal(t, receipt, duplicate)
	require.Equal(t, 1, native.calls, "an uncertain command is not retried after reauthorization")
}
