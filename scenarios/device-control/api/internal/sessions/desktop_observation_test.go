package sessions

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type observingNative struct {
	desktopNative
	captured func()
}

type interruptibleObservation struct {
	desktopNative
	entered chan struct{}
}

func (n *interruptibleObservation) Observe(ctx context.Context) (DesktopObservation, error) {
	close(n.entered)
	<-ctx.Done()
	return DesktopObservation{}, ctx.Err()
}

type interruptibleMutation struct {
	desktopNative
	entered chan struct{}
}

func (n *interruptibleMutation) Apply(ctx context.Context, _ DesktopCommand) error {
	n.effects++
	close(n.entered)
	<-ctx.Done()
	return ctx.Err()
}

func TestDesktopStopDuringMutationPreservesUncertainReceipt(t *testing.T) {
	c, repo, _, _, lease := desktopFixture(t)
	native := &interruptibleMutation{entered: make(chan struct{})}
	c.native = native
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	command := desktopCommand(lease)
	done := make(chan DesktopReceipt, 1)
	go func() {
		receipt, _ := c.Act(ctx, command)
		done <- receipt
	}()
	select {
	case <-native.entered:
	case <-ctx.Done():
		t.Fatal("mutation did not start")
	}
	stopCtx, stopCancel := context.WithTimeout(ctx, time.Second)
	defer stopCancel()
	require.NoError(t, c.Stop(stopCtx, lease))
	select {
	case receipt := <-done:
		require.Equal(t, "outcome_unknown", receipt.Outcome)
	case <-ctx.Done():
		t.Fatal("mutation did not exit")
	}
	require.NoError(t, repo.Update(ctx, func(s *DesktopState) error {
		require.Nil(t, s.Lease)
		require.Equal(t, "outcome_unknown", s.Receipts[command.ID].Outcome)
		return nil
	}))
	_, err := c.Act(ctx, command)
	require.Error(t, err)
	require.Equal(t, 1, native.effects)
}

func TestDesktopStopInterruptsOnlyMatchingLease(t *testing.T) {
	c, _, _, _, prior := desktopFixture(t)
	next := prior
	next.Ref.SessionID = "successor"
	next, err := c.Open(context.Background(), next, prior.Epoch, true)
	require.NoError(t, err)
	native := &interruptibleObservation{entered: make(chan struct{})}
	c.native = native
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := c.Observe(ctx, next)
		done <- err
	}()
	select {
	case <-native.entered:
	case <-ctx.Done():
		t.Fatal("observation did not start")
	}
	staleCtx, staleCancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer staleCancel()
	require.Error(t, c.Stop(staleCtx, prior))
	select {
	case err := <-done:
		t.Fatalf("stale Stop interrupted successor: %v", err)
	default:
	}
	stopCtx, stopCancel := context.WithTimeout(ctx, time.Second)
	defer stopCancel()
	require.NoError(t, c.Stop(stopCtx, next))
	select {
	case err := <-done:
		require.Error(t, err)
	case <-ctx.Done():
		t.Fatal("matching Stop failed to interrupt observation")
	}
	_, err = c.Act(ctx, desktopCommand(next))
	require.Error(t, err)
	require.Zero(t, native.effects)
}

func (n *observingNative) Observe(context.Context) (DesktopObservation, error) {
	if n.captured != nil {
		n.captured()
	}
	return DesktopObservation{Image: image.NewRGBA(image.Rect(0, 0, 2, 2)), DisplayID: "display", GeometryRevision: "revision", CapturedAt: time.Now()}, nil
}

func TestObservationOnlyLeaseCannotActAndRevocationDiscardsPixels(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	lease.Control = false
	lease.Ref.SessionID = "observation-only"
	lease, err := c.Open(context.Background(), lease, lease.Epoch, false)
	require.NoError(t, err)
	native := &observingNative{}
	c.native = native
	snapshot, err := c.Observe(context.Background(), lease)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Image)
	_, err = c.Act(context.Background(), desktopCommand(lease))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Zero(t, native.effects)
	native.captured = func() { auth.denied = true }
	snapshot, err = c.Observe(context.Background(), lease)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Nil(t, snapshot.Image)
}

func TestDesktopRepositoryContentionNeverRetriesEffects(t *testing.T) {
	_, repo, _, _, _ := desktopFixture(t)
	tx, err := repo.db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE device_control_desktop_sessions SET state=state`)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	calls := 0
	err = repo.Update(ctx, func(*DesktopState) error { calls++; return nil })
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, calls)
	require.NoError(t, tx.Rollback())
	err = repo.Update(context.Background(), func(*DesktopState) error { calls++; return simulatedDesktopBusy{} })
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

type simulatedDesktopBusy struct{}

func (simulatedDesktopBusy) Error() string { return "effect returned busy" }
func (simulatedDesktopBusy) Code() int     { return 5 }

type semanticNative struct {
	observingNative
	snapshot DesktopSemanticObservation
	leaseID  string
}

func (n *semanticNative) ObserveProcess(ctx context.Context, _ uint32, leaseID string) (DesktopObservation, error) {
	n.leaseID = leaseID
	image, err := n.Observe(ctx)
	image.Semantic = &n.snapshot
	return image, err
}

func TestSemanticObservationLeaseScopeAndFreshness(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	c.native = &observingNative{}
	_, err := c.ObserveProcess(context.Background(), lease, 42)
	require.ErrorIs(t, err, ErrDesktopAdmission, "unsupported semantic request cannot silently return only pixels")
	native := &semanticNative{snapshot: DesktopSemanticObservation{Revision: "semantic-1", ProcessID: 42, ExpiresAt: time.Now().Add(time.Second), Elements: []DesktopSemanticElement{{ID: "element-1", Name: "entry", Editable: true}}}}
	c.native = native
	got, err := c.ObserveProcess(context.Background(), lease, 42)
	require.NoError(t, err)
	require.Equal(t, lease.Ref.SessionID, native.leaseID)
	require.Equal(t, "element-1", got.Semantic.Elements[0].ID)
	native.snapshot.Elements = append(native.snapshot.Elements, native.snapshot.Elements[0])
	_, err = c.ObserveProcess(context.Background(), lease, 42)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	native.snapshot.Elements = native.snapshot.Elements[:1]
	native.snapshot.ExpiresAt = time.Now().Add(-time.Second)
	_, err = c.ObserveProcess(context.Background(), lease, 42)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	native.snapshot.ExpiresAt = time.Now().Add(time.Second)
	native.captured = func() { auth.denied = true }
	got, err = c.ObserveProcess(context.Background(), lease, 42)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Nil(t, got.Semantic)
	require.Nil(t, got.Image)
}

type applicationNative struct {
	semanticNative
	catalog                 DesktopApplications
	discovered              func()
	applicationID, revision string
}

func (n *applicationNative) Applications(_ context.Context, leaseID string) (DesktopApplications, error) {
	n.leaseID = leaseID
	if n.discovered != nil {
		n.discovered()
	}
	return n.catalog, nil
}

func (n *applicationNative) ObserveApplication(ctx context.Context, id, revision, leaseID string) (DesktopObservation, error) {
	n.applicationID, n.revision = id, revision
	return n.ObserveProcess(ctx, 42, leaseID)
}

func TestApplicationCatalogAuthorizationAndSelection(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	n := &applicationNative{catalog: DesktopApplications{Revision: "catalog", ExpiresAt: time.Now().Add(20 * time.Second), Applications: []DesktopApplication{{ID: "opaque", Name: "Editor", ProcessID: 42}}}}
	n.snapshot = DesktopSemanticObservation{Revision: "fields", ProcessID: 42, ExpiresAt: time.Now().Add(time.Second)}
	c.native = n
	got, err := c.Applications(context.Background(), lease)
	require.NoError(t, err)
	require.Equal(t, lease.Ref.SessionID, n.leaseID)
	snapshot, err := c.ObserveApplication(context.Background(), lease, got.Applications[0].ID, got.Revision)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Semantic)
	require.Equal(t, "opaque", n.applicationID)
	require.Equal(t, "catalog", n.revision)
	n.catalog.Applications = append(n.catalog.Applications, n.catalog.Applications[0])
	_, err = c.Applications(context.Background(), lease)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	n.catalog.Applications = n.catalog.Applications[:1]
	n.catalog.ExpiresAt = time.Now().Add(-time.Second)
	_, err = c.Applications(context.Background(), lease)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	n.catalog.ExpiresAt = time.Now().Add(20 * time.Second)
	n.discovered = func() { auth.denied = true }
	got, err = c.Applications(context.Background(), lease)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Empty(t, got.Applications, "revocation during discovery discards names")
	auth.denied = false
	n.captured = func() { auth.denied = true }
	snapshot, err = c.ObserveApplication(context.Background(), lease, "opaque", "catalog")
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Nil(t, snapshot.Semantic)
	require.Nil(t, snapshot.Image)
}

type resolvingNative struct {
	observingNative
	result  DesktopResolution
	resolve func()
	lease   string
}

func (n *resolvingNative) Resolve(_ context.Context, _ DesktopSelector, lease string) (DesktopResolution, error) {
	n.lease = lease
	if n.resolve != nil {
		n.resolve()
	}
	return n.result, nil
}

func TestResolutionRejectsInvalidCardinalityAndMidReadRevocation(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	n := &resolvingNative{result: DesktopResolution{Disposition: "unique", Revision: "r", GeometryRevision: "g", ExpiresAt: time.Now().Add(time.Second), ElementIDs: []string{"field"}}}
	c.native = n
	selector := DesktopSelector{Revision: "r", WindowID: "w", Name: "Entry"}
	got, err := c.Resolve(context.Background(), lease, selector)
	require.NoError(t, err)
	require.Equal(t, []string{"field"}, got.ElementIDs)
	require.Equal(t, lease.Ref.SessionID, n.lease)
	for _, kind := range []string{"absent", "ambiguous", "unknown"} {
		n.result.Disposition = kind
		got, err = c.Resolve(context.Background(), lease, selector)
		require.ErrorIs(t, err, ErrDesktopAdmission)
		require.Empty(t, got.ElementIDs)
	}
	n.result.Disposition = "unique"
	n.resolve = func() { auth.denied = true }
	got, err = c.Resolve(context.Background(), lease, selector)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.Empty(t, got.ElementIDs)
}

type activationNative struct {
	desktopNative
	capture func(context.Context) (DesktopActivationContext, error)
}

func (n *activationNative) CaptureActivation(ctx context.Context) (DesktopActivationContext, error) {
	return n.capture(ctx)
}

func TestDesktopActivationRequiresCurrentObservationAuthority(t *testing.T) {
	for _, scenario := range []string{"valid", "denied", "revoked-during", "expired-during", "future", "stale", "cancelled", "wrong-session", "missing-window", "missing-bounds"} {
		t.Run(scenario, func(t *testing.T) {
			c, _, _, auth, lease := desktopFixture(t)
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			native := &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
				calls++
				result := DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 42, ProcessID: 7, PointerX: -100, PointerY: 20, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}
				switch scenario {
				case "revoked-during":
					auth.denied = true
				case "expired-during":
					c.now = func() time.Time { return lease.ExpiresAt.Add(time.Second) }
				case "future":
					result.CapturedAt = time.Now().Add(time.Hour)
				case "stale":
					result.CapturedAt = time.Now().Add(-time.Minute)
				case "cancelled":
					cancel()
				case "missing-bounds":
					result.SourceBounds = DesktopBounds{}
				case "missing-window":
					result.ActiveWindow = 0
				}
				return result, nil
			}}
			c.native = native
			if scenario == "denied" {
				auth.denied = true
			}
			if scenario == "wrong-session" {
				lease.Ref.SessionID = "another-session"
			}
			result, err := c.CaptureActivation(ctx, lease)
			if scenario == "valid" {
				require.NoError(t, err)
				require.EqualValues(t, -100, result.PointerX)
			} else {
				require.Error(t, err)
				require.Equal(t, DesktopActivationContext{}, result)
			}
			if scenario == "denied" || scenario == "wrong-session" {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.Zero(t, native.effects)
		})
	}
}

func TestDesktopStopInterruptsActivationCapture(t *testing.T) {
	c, _, _, _, lease := desktopFixture(t)
	entered := make(chan struct{})
	c.native = &activationNative{capture: func(ctx context.Context) (DesktopActivationContext, error) {
		close(entered)
		<-ctx.Done()
		return DesktopActivationContext{}, ctx.Err()
	}}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _, err := c.CaptureActivation(ctx, lease); done <- err }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("capture did not start")
	}
	require.NoError(t, c.Stop(ctx, lease))
	select {
	case err := <-done:
		require.Error(t, err)
	case <-ctx.Done():
		t.Fatal("capture did not stop")
	}
}

func TestDesktopActivationReferencesAreEphemeralAndLeaseBound(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	now := time.Now()
	c.now = func() time.Time { return now }
	calls := 0
	c.native = &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
		calls++
		return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 42, ProcessID: 7, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: now}, nil
	}}
	first, err := c.CaptureActivationReference(context.Background(), lease)
	require.NoError(t, err)
	require.NotEqual(t, "42", first.ID)
	require.Equal(t, now.Add(30*time.Second), first.ExpiresAt)
	read, err := c.ReadActivation(context.Background(), lease, first.ID)
	require.NoError(t, err)
	require.Equal(t, first, read)
	wireLease := lease
	wireLease.ExpiresAt = lease.ExpiresAt.Round(0).UTC()
	read, err = c.ReadActivation(context.Background(), wireLease, first.ID)
	require.NoError(t, err)
	require.Equal(t, first, read)
	require.Equal(t, 1, calls)
	changed := lease
	changed.Actor = "other"
	_, err = c.ReadActivation(context.Background(), changed, first.ID)
	require.Error(t, err)
	auth.denied = true
	_, err = c.ReadActivation(context.Background(), lease, first.ID)
	require.Error(t, err)
	auth.denied = false
	now = now.Add(time.Second)
	second, err := c.CaptureActivationReference(context.Background(), lease)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	_, err = c.ReadActivation(context.Background(), lease, first.ID)
	require.Error(t, err)
	// Reconstructed helpers cannot resume ephemeral native identity.
	rebuilt, err := NewDesktopController(c.repo, c.authority, c.native, c.surface, c.desktopID, c.helperID)
	require.NoError(t, err)
	_, err = rebuilt.ReadActivation(context.Background(), lease, second.ID)
	require.Error(t, err)
	now = second.ExpiresAt
	_, err = c.ReadActivation(context.Background(), lease, second.ID)
	require.Error(t, err)
	require.Nil(t, c.activation)
	now = now.Add(time.Second)
	third, err := c.CaptureActivationReference(context.Background(), lease)
	require.NoError(t, err)
	require.Equal(t, lease.ExpiresAt, third.ExpiresAt)
	require.NoError(t, c.Stop(context.Background(), lease))
	_, err = c.ReadActivation(context.Background(), lease, third.ID)
	require.Error(t, err)
}

type verifiedActivationNative struct {
	activationNative
	verify func(context.Context, uint64, uint32) error
}

func (n *verifiedActivationNative) VerifyWindowProcess(ctx context.Context, w uint64, p uint32) error {
	return n.verify(ctx, w, p)
}

func TestCompanionCaptureRequiresOwnershipBeforeAndAfterObservation(t *testing.T) {
	for _, failAt := range []int{0, 1, 2} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			c, _, _, _, lease := desktopFixture(t)
			checks, captures := 0, 0
			c.native = &verifiedActivationNative{activationNative: activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
				captures++
				return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 5, ProcessID: 8, DisplayID: "d", GeometryRevision: "g", CapturedAt: time.Now()}, nil
			}}, verify: func(_ context.Context, w uint64, p uint32) error {
				checks++
				require.EqualValues(t, 42, w)
				require.EqualValues(t, 100, p)
				if checks == failAt {
					return ErrDesktopAdmission
				}
				return nil
			}}
			ref, err := c.CaptureCompanionActivation(context.Background(), lease, 42, 100)
			if failAt == 0 {
				require.NoError(t, err)
				require.NotEmpty(t, ref.ID)
			} else {
				require.Error(t, err)
				require.Empty(t, ref.ID)
				require.Nil(t, c.activation)
			}
			if failAt == 1 {
				require.Zero(t, captures)
			} else {
				require.Equal(t, 1, captures)
			}
		})
	}
}

func TestDesktopActivationErasedByLifecycleWithoutRead(t *testing.T) {
	for _, operation := range []string{"expiry", "stop", "failed-stop", "shutdown", "takeover", "failed-takeover", "successor-helper"} {
		t.Run(operation, func(t *testing.T) {
			c, repo, _, auth, lease := desktopFixture(t)
			now := time.Now()
			c.now = func() time.Time { return now }
			native := &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
				return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 42, ProcessID: 7, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: now}, nil
			}}
			c.native = native
			ref, err := c.CaptureActivationReference(context.Background(), lease)
			require.NoError(t, err)
			require.NoError(t, c.ReapExpired(context.Background()))
			require.NotNil(t, c.activation, "live context must survive unrelated lifecycle checks")
			switch operation {
			case "expiry":
				now = ref.ExpiresAt
				require.NoError(t, c.ReapExpired(context.Background()))
				require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error { require.NotNil(t, s.Lease); return nil }))
			case "stop", "failed-stop":
				native.releaseFail = operation == "failed-stop"
				err = c.Stop(context.Background(), lease)
				if native.releaseFail {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "shutdown":
				require.NoError(t, c.Shutdown(context.Background()))
			case "takeover", "failed-takeover":
				next := lease
				next.Ref.SessionID = "successor"
				native.releaseFail = operation == "failed-takeover"
				_, err = c.Open(context.Background(), next, lease.Epoch, true)
				if native.releaseFail {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "successor-helper":
				next, err := NewDesktopController(repo, auth, native, c.surface, c.desktopID, "helper-generation-2")
				require.NoError(t, err)
				require.NoError(t, next.ActivateHelper(context.Background(), c.helperID, lease.Epoch))
				require.ErrorIs(t, c.ReapExpired(context.Background()), ErrDesktopAdmission)
			}
			require.Nil(t, c.activation, "native identity must be erased without a context read")
			require.Zero(t, native.effects)
		})
	}
}

func TestDesktopDeniedStopPreservesActiveContext(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	c.native = &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
		return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 42, ProcessID: 7, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}, nil
	}}
	ref, err := c.CaptureActivationReference(context.Background(), lease)
	require.NoError(t, err)
	auth.denied = true
	require.ErrorIs(t, c.Stop(context.Background(), lease), ErrDesktopAdmission)
	auth.denied = false
	read, err := c.ReadActivation(context.Background(), lease, ref.ID)
	require.NoError(t, err)
	require.Equal(t, ref, read)
}

func TestDesktopDeleteActivationIsExactAndAuthorized(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	c.native = &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
		return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 42, ProcessID: 7, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}, nil
	}}
	ctx := context.Background()
	old, err := c.CaptureActivationReference(ctx, lease)
	require.NoError(t, err)
	current, err := c.CaptureActivationReference(ctx, lease)
	require.NoError(t, err)
	require.NoError(t, c.DeleteActivation(ctx, lease, old.ID))
	read, err := c.ReadActivation(ctx, lease, current.ID)
	require.NoError(t, err)
	require.Equal(t, current, read)
	wrong := lease
	wrong.Actor = "other"
	require.Error(t, c.DeleteActivation(ctx, wrong, current.ID))
	auth.denied = true
	require.Error(t, c.DeleteActivation(ctx, lease, current.ID))
	auth.denied = false
	require.Error(t, c.DeleteActivation(ctx, lease, "not-a-reference"))
	require.NotNil(t, c.activation)
	require.NoError(t, c.DeleteActivation(ctx, lease, current.ID))
	require.Nil(t, c.activation)
	_, err = c.ReadActivation(ctx, lease, current.ID)
	require.Error(t, err)
	require.NoError(t, c.DeleteActivation(ctx, lease, current.ID))
	require.NoError(t, c.Stop(ctx, lease))
	require.Error(t, c.DeleteActivation(ctx, lease, current.ID))
}

type observerSafetyNative struct {
	observingNative
	releases int
}

func (n *observerSafetyNative) ReleaseHeld(context.Context) error { n.releases++; return nil }

func TestDesktopObserversCannotReleaseControllerInput(t *testing.T) {
	for _, finish := range []string{"stop", "expiry", "revoked"} {
		t.Run(finish, func(t *testing.T) {
			c, repo, _, _, control := desktopFixture(t)
			native := &observerSafetyNative{}
			c.native = native
			now := time.Now()
			c.now = func() time.Time {
				if now.After(time.Now()) {
					return now
				}
				return time.Now()
			}
			observer := control
			observer.Ref.SessionID = "observer"
			observer.Control = false
			observer.ExpiresAt = now.Add(10 * time.Second)
			observer, err := c.Open(context.Background(), observer, control.Epoch, false)
			require.NoError(t, err)
			require.Zero(t, native.releases)
			_, err = c.Act(context.Background(), desktopCommand(control))
			require.NoError(t, err)
			_, err = c.Observe(context.Background(), observer)
			require.NoError(t, err)
			require.NoError(t, c.ReapExpired(context.Background()))
			require.Zero(t, native.releases)
			_, err = c.Act(context.Background(), desktopCommand(observer))
			require.Error(t, err)
			switch finish {
			case "stop":
				require.NoError(t, c.Stop(context.Background(), observer))
			case "expiry":
				now = observer.ExpiresAt
				require.NoError(t, c.ReapExpired(context.Background()))
			case "revoked":
				require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
					v := s.Observers[observer.Epoch]
					v.GrantID = "revoked-observer"
					s.Observers[observer.Epoch] = v
					return nil
				}))
				pending, err := repo.ReadRevokedCleanup(context.Background(), observer, "revoked-observer")
				require.NoError(t, err)
				require.False(t, pending.Released)
				_, err = c.Observe(context.Background(), observer)
				require.Error(t, err)
				require.NoError(t, c.ReapExpired(context.Background()))
			}
			require.Zero(t, native.releases, "observer lifecycle must not release controller input")
			receipt, err := repo.ReadCleanup(context.Background(), observer)
			require.NoError(t, err)
			require.True(t, receipt.Released)
			_, err = c.Observe(context.Background(), observer)
			require.Error(t, err)
			command := desktopCommand(control)
			command.ID = "after-observer"
			_, err = c.Act(context.Background(), command)
			require.NoError(t, err)
			require.Equal(t, 2, native.effects)
			require.NoError(t, c.Stop(context.Background(), control))
			require.Equal(t, 1, native.releases)
		})
	}
}

type imageActivationNative struct {
	verifiedActivationNative
	captureImage func(context.Context) (DesktopActivationImage, error)
}

func (n *imageActivationNative) CaptureActivationImage(ctx context.Context) (DesktopActivationImage, error) {
	return n.captureImage(ctx)
}

func TestDesktopActivationImageIsExplicitFrozenAndLeaseBound(t *testing.T) {
	c, _, _, auth, lease := desktopFixture(t)
	ctx := context.Background()
	now := time.Now()
	c.now = func() time.Time { return now }
	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.Pix[0] = 111
	captures, verifies := 0, 0
	metadata := func(context.Context) (DesktopActivationContext, error) {
		return DesktopActivationContext{SourceBounds: DesktopBounds{Width: 2, Height: 1}, ActiveWindow: 5, ProcessID: 8, DisplayID: "d", GeometryRevision: "g", CapturedAt: now}, nil
	}
	c.native = &imageActivationNative{verifiedActivationNative: verifiedActivationNative{
		activationNative: activationNative{capture: metadata}, verify: func(context.Context, uint64, uint32) error { verifies++; return nil },
	}, captureImage: func(ctx context.Context) (DesktopActivationImage, error) {
		captures++
		m, _ := metadata(ctx)
		return DesktopActivationImage{Context: m, Image: source}, nil
	}}
	plain, err := c.CaptureCompanionActivation(ctx, lease, 42, 100)
	require.NoError(t, err)
	require.False(t, plain.HasImage)
	require.Zero(t, captures)
	_, pixels, err := c.ReadActivationImage(ctx, lease, plain.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
	ref, err := c.CaptureCompanionActivationImage(ctx, lease, 42, 100)
	require.NoError(t, err)
	require.True(t, ref.HasImage)
	require.Equal(t, 1, captures)
	require.Equal(t, 4, verifies)
	source.Pix[0] = 222
	read, pixels, err := c.ReadActivationImage(ctx, lease, ref.ID)
	require.NoError(t, err)
	require.Equal(t, ref, read)
	require.EqualValues(t, 111, pixels.Pix[0])
	pixels.Pix[0] = 33
	_, pixels, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.NoError(t, err)
	require.EqualValues(t, 111, pixels.Pix[0])
	require.Equal(t, 1, captures)
	wrong := lease
	wrong.Actor = "other"
	_, pixels, err = c.ReadActivationImage(ctx, wrong, ref.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
	auth.denied = true
	_, pixels, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
	auth.denied = false
	require.NoError(t, c.DeleteActivation(ctx, lease, plain.ID))
	_, _, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.NoError(t, err)
	require.NoError(t, c.DeleteActivation(ctx, lease, ref.ID))
	require.Nil(t, c.activation)
	_, pixels, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
	ref, err = c.CaptureCompanionActivationImage(ctx, lease, 42, 100)
	require.NoError(t, err)
	now = ref.ExpiresAt
	require.NoError(t, c.ReapExpired(ctx))
	require.Nil(t, c.activation)
	_, pixels, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
	now = now.Add(time.Second)
	ref, err = c.CaptureCompanionActivationImage(ctx, lease, 42, 100)
	require.NoError(t, err)
	require.NoError(t, c.Stop(ctx, lease))
	require.Nil(t, c.activation)
	_, pixels, err = c.ReadActivationImage(ctx, lease, ref.ID)
	require.Error(t, err)
	require.Nil(t, pixels)
}

func TestDesktopActivationImageRejectsInvalidEvidence(t *testing.T) {
	for _, failure := range []string{"nil", "dimensions", "stride", "short-buffer", "oversize", "revoked", "ownership-before", "ownership-after"} {
		t.Run(failure, func(t *testing.T) {
			c, _, _, auth, lease := desktopFixture(t)
			calls, checks := 0, 0
			c.native = &imageActivationNative{verifiedActivationNative: verifiedActivationNative{verify: func(context.Context, uint64, uint32) error {
				checks++
				if (failure == "ownership-before" && checks == 1) || (failure == "ownership-after" && checks == 2) {
					return ErrDesktopAdmission
				}
				return nil
			}}, captureImage: func(context.Context) (DesktopActivationImage, error) {
				calls++
				m := DesktopActivationContext{SourceBounds: DesktopBounds{Width: 2, Height: 1}, ActiveWindow: 5, ProcessID: 8, DisplayID: "d", GeometryRevision: "g", CapturedAt: time.Now()}
				pixels := image.NewRGBA(image.Rect(0, 0, 2, 1))
				switch failure {
				case "nil":
					pixels = nil
				case "dimensions":
					pixels.Rect.Max.X = 3
				case "stride":
					pixels.Stride = 1
				case "short-buffer":
					pixels.Pix = pixels.Pix[:1]
				case "oversize":
					m.SourceBounds = DesktopBounds{Width: 65535, Height: 65535}
					pixels.Rect = image.Rect(0, 0, 65535, 65535)
				case "revoked":
					auth.denied = true
				}
				return DesktopActivationImage{Context: m, Image: pixels}, nil
			}}
			ref, err := c.CaptureCompanionActivationImage(context.Background(), lease, 42, 100)
			require.Error(t, err)
			require.Empty(t, ref.ID)
			require.Nil(t, c.activation)
			if failure == "ownership-before" {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
		})
	}
}
