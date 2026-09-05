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
	lease, err := c.Open(context.Background(), lease, lease.Epoch, true)
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
	for _, scenario := range []string{"valid", "denied", "revoked-during", "expired-during", "future", "stale", "cancelled", "wrong-session", "missing-window"} {
		t.Run(scenario, func(t *testing.T) {
			c, _, _, auth, lease := desktopFixture(t)
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			native := &activationNative{capture: func(context.Context) (DesktopActivationContext, error) {
				calls++
				result := DesktopActivationContext{ActiveWindow: 42, ProcessID: 7, PointerX: -100, PointerY: 20, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}
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
		return DesktopActivationContext{ActiveWindow: 42, ProcessID: 7, DisplayID: "display", GeometryRevision: "geometry", CapturedAt: now}, nil
	}}
	first, err := c.CaptureActivationReference(context.Background(), lease)
	require.NoError(t, err)
	require.NotEqual(t, "42", first.ID)
	require.Equal(t, now.Add(30*time.Second), first.ExpiresAt)
	read, err := c.ReadActivation(context.Background(), lease, first.ID)
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
				return DesktopActivationContext{ActiveWindow: 5, ProcessID: 8, DisplayID: "d", GeometryRevision: "g", CapturedAt: time.Now()}, nil
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
