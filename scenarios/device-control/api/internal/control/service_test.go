package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"device-control/internal/androidartifact"
	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:control-test-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), db)
	require.NoError(t, err)
	return svc, db
}

type semanticSnapshotFixture struct {
	*fakes.Strategy
	calls    int
	selected *bool
}

func (f *semanticSnapshotFixture) ObserveSemantics(context.Context) (strategy.SemanticSnapshot, error) {
	f.calls++
	return strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{{Text: "Settings", Clickable: true, Enabled: true, Selected: f.selected}}, ObservedAt: time.Now()}, nil
}

func TestSemanticObservationOwnsOrValidatesLease(t *testing.T) {
	fixture := &semanticSnapshotFixture{Strategy: fakes.New("semantic", strategy.StatusAvailable, strategy.CapSemanticTree)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	snapshot, err := svc.ObserveSemantics(context.Background(), "target", "operator", "", "usb")
	require.NoError(t, err)
	require.Equal(t, "Settings", snapshot.Elements[0].Text)
	require.Empty(t, svc.ListLiveSessions(), "automatically acquired lease must be released")
	_, err = svc.ObserveSemantics(context.Background(), "target", "operator", "invalid", "usb")
	require.Error(t, err)
	require.Equal(t, 1, fixture.calls, "invalid lease must not inspect the device")
}

type blockingSemanticFixture struct {
	*fakes.Strategy
	started chan struct{}
}

func (f *blockingSemanticFixture) ObserveSemantics(ctx context.Context) (strategy.SemanticSnapshot, error) {
	close(f.started)
	<-ctx.Done()
	return strategy.SemanticSnapshot{}, ctx.Err()
}

func TestSemanticObservationIsFencedAndCanceledWithLease(t *testing.T) {
	fixture := &blockingSemanticFixture{Strategy: fakes.New("semantic", strategy.StatusAvailable, strategy.CapSemanticTree), started: make(chan struct{})}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	session, err := svc.AcquireContext(context.Background(), "target", "operator", time.Minute)
	require.NoError(t, err)
	completed := make(chan error, 1)
	go func() {
		_, err := svc.ObserveSemantics(context.Background(), "target", "operator", session.LeaseToken, "usb")
		completed <- err
	}()
	select {
	case <-fixture.started:
	case <-time.After(time.Second):
		t.Fatal("observation did not start")
	}
	_, err = svc.ObserveSemantics(context.Background(), "target", "operator", session.LeaseToken, "usb")
	require.ErrorContains(t, err, "active run")
	_, err = svc.ReleaseContext(context.Background(), session.ID)
	require.NoError(t, err)
	select {
	case err := <-completed:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("release did not cancel observation")
	}
}

type slowInventoryProbe struct {
	*fakes.Strategy
	started chan struct{}
	release chan struct{}
}

type countingInventoryProbe struct {
	*fakes.Strategy
	describes atomic.Int32
}

type coordinatedInventoryProbe struct {
	*fakes.Strategy
	ready chan struct{}
	waits bool
}

func (p *coordinatedInventoryProbe) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	if p.waits {
		select {
		case <-p.ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		close(p.ready)
	}
	return []strategy.Device{{ID: p.ID() + "-device", StrategyID: p.ID(), Transport: p.ID(), Health: strategy.StatusAvailable, ObservedAt: time.Now()}}, nil
}

type earlyScopedInventoryProbe struct {
	*fakes.Strategy
	ready  chan struct{}
	scoped bool
}

func (p *earlyScopedInventoryProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.scoped {
		close(p.ready)
	}
	return p.Strategy.Describe(ctx)
}

func (p *earlyScopedInventoryProbe) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{{ID: "early-device", Serial: "serial", StrategyID: p.ID(), Transport: p.ID(), Health: strategy.StatusAvailable, ObservedAt: time.Now()}}, nil
}

func (p *earlyScopedInventoryProbe) ForDevice(string) strategy.Strategy {
	return &earlyScopedInventoryProbe{Strategy: p.Strategy, ready: p.ready, scoped: true}
}

type coordinatedScopedProbe struct {
	*fakes.Strategy
	ready  chan struct{}
	serial string
}

func (p *coordinatedScopedProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.serial == "slow" {
		select {
		case <-p.ready:
		case <-ctx.Done():
			return strategy.Declaration{}, ctx.Err()
		}
	} else if p.serial == "fast" {
		close(p.ready)
	}
	declaration, err := p.Strategy.Describe(ctx)
	if p.serial != "" {
		declaration.Capabilities = map[string]strategy.Capability{strategy.CapScreenshot: {Name: strategy.CapScreenshot, Status: strategy.StatusAvailable}}
	}
	return declaration, err
}

func (p *coordinatedScopedProbe) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{
		{ID: "slow-device", Serial: "slow", StrategyID: p.ID(), Transport: "usb", Health: strategy.StatusAvailable},
		{ID: "fast-device", Serial: "fast", StrategyID: p.ID(), Transport: "usb", Health: strategy.StatusAvailable},
	}, nil
}

func (p *coordinatedScopedProbe) ForDevice(serial string) strategy.Strategy {
	return &coordinatedScopedProbe{Strategy: p.Strategy, ready: p.ready, serial: serial}
}

type failingScopedInventoryProbe struct {
	*fakes.Strategy
	serial string
}

func (p *failingScopedInventoryProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.serial == "failed" {
		return strategy.Declaration{}, errors.New("target probe unavailable")
	}
	return p.Strategy.Describe(ctx)
}

func (p *failingScopedInventoryProbe) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{
		{ID: "failed-device", Serial: "failed", StrategyID: p.ID(), Transport: "usb", Health: strategy.StatusAvailable},
		{ID: "healthy-device", Serial: "healthy", StrategyID: p.ID(), Transport: "usb", Health: strategy.StatusAvailable},
	}, nil
}

func (p *failingScopedInventoryProbe) ForDevice(serial string) strategy.Strategy {
	return &failingScopedInventoryProbe{Strategy: p.Strategy, serial: serial}
}

type coordinatedPromotedProbe struct {
	*fakes.Strategy
	started  *atomic.Int32
	ready    chan struct{}
	deviceID string
}

func (p *coordinatedPromotedProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.started.Add(1) == 2 {
		close(p.ready)
	}
	select {
	case <-p.ready:
		return p.Strategy.Describe(ctx)
	case <-ctx.Done():
		return strategy.Declaration{}, ctx.Err()
	}
}

func (p *coordinatedPromotedProbe) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{{ID: p.deviceID, Serial: p.deviceID, StrategyID: p.ID(), Transport: "wireless", Health: strategy.StatusAvailable, ObservedAt: time.Now()}}, nil
}

type panickingEnumerator struct{ *fakes.Strategy }

func (*panickingEnumerator) Enumerate(context.Context) ([]strategy.Device, error) {
	panic("enumeration failed")
}

func (p *countingInventoryProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	p.describes.Add(1)
	return p.Strategy.Describe(ctx)
}

func (p *countingInventoryProbe) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{{ID: "counted-device", StrategyID: p.ID(), Transport: "usb", Health: strategy.StatusAvailable, ObservedAt: time.Now()}}, nil
}

type slowObserverProbe struct {
	*countingInventoryProbe
	started chan struct{}
	release chan struct{}
}

func (p *slowObserverProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.describes.Add(1) > 1 {
		select {
		case p.started <- struct{}{}:
		default:
		}
		select {
		case <-p.release:
		case <-ctx.Done():
		}
	}
	return p.Strategy.Describe(ctx)
}

func (p *slowObserverProbe) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{}, nil
}

type cancellationIgnoringDescribeStrategy struct {
	pollingStateStrategy
	started chan struct{}
	release chan struct{}
}

func (p cancellationIgnoringDescribeStrategy) Describe(ctx context.Context) (strategy.Declaration, error) {
	close(p.started)
	<-p.release // Deliberately ignore ctx cancellation until the probe returns.
	return p.pollingStateStrategy.Describe(ctx)
}

type panickingObserverProbe struct{ *fakes.Strategy }

func (*panickingObserverProbe) Describe(context.Context) (strategy.Declaration, error) {
	panic("observer probe failed")
}

func (*panickingObserverProbe) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{}, nil
}

func (p *slowInventoryProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}
	select {
	case <-p.release:
	case <-ctx.Done():
	}
	return p.Strategy.Describe(ctx)
}

type flowAuthResolver struct{ value string }

type pairingDiscoveryFixture struct{}

func (pairingDiscoveryFixture) DiscoverMDNS(context.Context) ([]androidtvremote.Device, error) {
	return []androidtvremote.Device{{
		Serial:      "bt-tv-1",
		IdentityKey: "bt-tv-1",
		Endpoint:    "tv.local:6466",
	}}, nil
}

type pairingClientFixture struct{ pin string }

func (f *pairingClientFixture) Pair(_ context.Context, _ androidtvremote.Device, pin string) ([]byte, error) {
	f.pin = pin
	return []byte("fixture-certificate"), nil
}

type interactivePairingSessionFixture struct {
	pin    string
	closed bool
}

func (f *interactivePairingSessionFixture) Complete(_ context.Context, pin string) ([]byte, error) {
	f.pin = pin
	return []byte("fixture-certificate"), nil
}

func (f *interactivePairingSessionFixture) Close() error {
	f.closed = true
	return nil
}

type interactivePairingClientFixture struct {
	session *interactivePairingSessionFixture
}

func (f *interactivePairingClientFixture) Pair(context.Context, androidtvremote.Device, string) ([]byte, error) {
	return nil, nil
}

func (f *interactivePairingClientFixture) Begin(context.Context, androidtvremote.Device) (androidtvremote.PairingSession, error) {
	f.session = &interactivePairingSessionFixture{}
	return f.session, nil
}

type leakyPairingClientFixture struct{}

func (leakyPairingClientFixture) Pair(_ context.Context, _ androidtvremote.Device, pin string) ([]byte, error) {
	return nil, errors.New("remote rejected PIN " + pin)
}

type pairingStoreFixture struct {
	serial      string
	certificate []byte
}

func (f *pairingStoreFixture) SavePairingCertificate(_ context.Context, serial string, certificate []byte) error {
	f.serial = serial
	f.certificate = append([]byte(nil), certificate...)
	return nil
}

func (f *pairingStoreFixture) LoadPairingCertificate(context.Context, string) ([]byte, error) {
	return append([]byte(nil), f.certificate...), nil
}

func (r *flowAuthResolver) Provision(_ context.Context, _, _, value string) error {
	r.value = value
	return nil
}

func (r *flowAuthResolver) Resolve(context.Context, string, string) (string, error) {
	return r.value, nil
}

func (r *flowAuthResolver) Delete(context.Context, string, string) error {
	r.value = ""
	return nil
}

func (r *flowAuthResolver) Status(context.Context, string, string) authdomain.ProviderStatus {
	return authdomain.ProviderStatus{Provider: "fake", ProviderState: "available", Configured: r.value != ""}
}

type flowAuthStrategy struct {
	*fakes.Strategy
	locked              bool
	failReadAfterUnlock bool
	cancelOnUnlock      context.CancelFunc
	restoredStates      []strategy.DeviceState
}

type screenConditionStrategy struct {
	*fakes.Strategy
	state strategy.DeviceState
}

func (s *screenConditionStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return s.state, nil
}

type semanticScreenFixture struct {
	*fakes.Strategy
	state  strategy.DeviceState
	calls  []string
	ignore bool
}

type dismissScreenFixture struct {
	*semanticScreenFixture
	secure     bool
	dismissals int
}

type homeScreenFixture struct {
	*semanticScreenFixture
	observed string
	calls    int
}

func (s *homeScreenFixture) Inspect(context.Context) (strategy.DeviceInspection, error) {
	return strategy.DeviceInspection{HomeComponent: "com.example.home/.Main"}, nil
}

func (s *homeScreenFixture) GoHome(context.Context) error {
	s.calls++
	s.state.ForegroundPackage = s.observed
	return nil
}

func (s *dismissScreenFixture) DismissKeyguard(context.Context) error {
	s.dismissals++
	if !s.secure {
		s.state.LockState = "unlocked"
	}
	return nil
}

func (s *semanticScreenFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return s.state, nil
}

func (s *semanticScreenFixture) SetScreenState(_ context.Context, state string) error {
	s.calls = append(s.calls, state)
	if !s.ignore {
		s.state.ScreenState = state
	}
	return nil
}

type delayedScreenFixture struct {
	*fakes.Strategy
	calls     int
	reads     int
	offAtRead int
}

func (f *delayedScreenFixture) SetScreenState(context.Context, string) error {
	f.calls++
	return nil
}

func (f *delayedScreenFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	f.reads++
	if f.offAtRead == 0 && f.reads > 1 || f.offAtRead > 0 && f.reads >= f.offAtRead {
		return strategy.DeviceState{ScreenState: "off"}, nil
	}
	return strategy.DeviceState{ScreenState: "on"}, nil
}

func (s *semanticScreenFixture) Unlock(_ context.Context, request strategy.UnlockRequest) (strategy.UnlockResult, error) {
	for i := range request.Secret {
		request.Secret[i] = 0
	}
	s.state.LockState = "unlocked"
	return strategy.UnlockResult{Outcome: authdomain.OutcomeUnlocked, Attempts: 1}, nil
}

func (s *flowAuthStrategy) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	if err := ctx.Err(); err != nil {
		return strategy.DeviceState{}, err
	}
	if !s.locked && s.failReadAfterUnlock {
		return strategy.DeviceState{}, errors.New("verification unavailable")
	}
	lockState := "unlocked"
	if s.locked {
		lockState = "locked"
	}
	return strategy.DeviceState{LockState: lockState, ScreenState: "on", Orientation: "portrait"}, nil
}

func (s *flowAuthStrategy) Unlock(_ context.Context, request strategy.UnlockRequest) (strategy.UnlockResult, error) {
	for i := range request.Secret {
		request.Secret[i] = 0
	}
	s.locked = false
	if s.cancelOnUnlock != nil {
		s.cancelOnUnlock()
	}
	return strategy.UnlockResult{Outcome: authdomain.OutcomeUnlocked, Attempts: 1}, nil
}

func (s *flowAuthStrategy) RestoreState(ctx context.Context, state strategy.DeviceState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.restoredStates = append(s.restoredStates, state)
	if state.LockState == "locked" {
		s.locked = true
	}
	return nil
}

func TestDirectActuationRequiresLeaseAndWritesOneNonEvidenceAudit(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.ActuateDevice(context.Background(), "fake", "operator", "", DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "requires a lease")
	require.Empty(t, svc.Audit())

	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	record, err := svc.ActuateDevice(context.Background(), "fake", "operator", lease.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.NoError(t, err)
	require.True(t, record.Interactive)
	require.False(t, record.EvidenceBacked)
	require.NotEmpty(t, record.CausationID)
	audits := svc.Audit()
	require.Len(t, audits, 1)
	require.Equal(t, record.ID, audits[0].ID)
	require.Equal(t, "direct-actuation", audits[0].Verb)
}

func TestAppLifecycleRequiresConfirmationAndUsesTypedStrategyBoundary(t *testing.T) {
	fake := fakes.New("lifecycle", strategy.StatusAvailable, strategy.CapAppLifecycle)
	svc := New(strategyregistry.New(fake))
	svc.devices.Upsert(devicedomain.Record{ID: "lifecycle", StrategyID: "lifecycle", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	lease, err := svc.Acquire("lifecycle", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.ExecuteAppLifecycle(context.Background(), "lifecycle", AppLifecycleOperation{Actor: "operator", LeaseToken: lease.LeaseToken, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "launch", Package: "com.example.editor"}})
	require.ErrorContains(t, err, "requires explicit confirmation")
	require.Empty(t, fake.LifecycleCalls())

	response, err := svc.ExecuteAppLifecycle(context.Background(), "lifecycle", AppLifecycleOperation{Actor: "operator", LeaseToken: lease.LeaseToken, Confirmed: true, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "launch", Package: "com.example.editor"}})
	require.NoError(t, err)
	require.NotEmpty(t, response.CommandID)
	require.Equal(t, "success", response.Audit.Outcome)
	require.Len(t, fake.LifecycleCalls(), 1)
	require.Equal(t, "launch", fake.LifecycleCalls()[0].Operation)
}

type verifiedArtifactResolverFixture struct {
	descriptor androidartifact.Descriptor
	path       string
	calls      int
}

func (f *verifiedArtifactResolverFixture) Resolve(_ context.Context, ref, packageName, versionName string) (androidartifact.StagedAPK, error) {
	f.calls++
	if ref != f.descriptor.ImmutableRef || packageName != f.descriptor.PackageName || versionName != f.descriptor.VersionName {
		return androidartifact.StagedAPK{}, fmt.Errorf("unexpected artifact identity")
	}
	return androidartifact.StagedAPK{Descriptor: f.descriptor, SigningCertificateSHA256: strings.Repeat("a", 64), Path: f.path}, nil
}

type verifiedArtifactStrategyFixture struct {
	*fakes.Strategy
	installedPath   string
	installedBytes  []byte
	version         string
	versionCode     int64
	signer          string
	existingInstall bool
}

func (f *verifiedArtifactStrategyFixture) Actuate(_ context.Context, event strategy.Actuation) error {
	if event.Action != "install" {
		return fmt.Errorf("unexpected actuation %q", event.Action)
	}
	f.installedPath = event.Value
	data, err := os.ReadFile(event.Value)
	if err != nil {
		return err
	}
	f.installedBytes = data
	return nil
}

func (f *verifiedArtifactStrategyFixture) AppLifecycle(_ context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	if request.Operation == "package-signing-state" {
		signer := f.signer
		if signer == "" {
			signer = strings.Repeat("a", 64)
		}
		return strategy.AppLifecycleResult{Operation: request.Operation, Package: request.Package, Status: strategy.StatusAvailable, Installed: f.existingInstall || len(f.installedBytes) > 0, SigningCertificateSHA256: signer}, nil
	}
	if request.Operation != "package-state" {
		return strategy.AppLifecycleResult{}, fmt.Errorf("unexpected lifecycle operation %q", request.Operation)
	}
	version := f.version
	if version == "" {
		version = "1.4.0"
	}
	versionCode := f.versionCode
	if versionCode == 0 {
		versionCode = 14
	}
	return strategy.AppLifecycleResult{Operation: request.Operation, Package: request.Package, Status: strategy.StatusAvailable, Installed: true, Version: version, VersionCode: versionCode}, nil
}

func TestVerifiedArtifactInstallRefusesDifferentInstalledSignerBeforeInstall(t *testing.T) {
	apkPath := filepath.Join(t.TempDir(), "verified.apk")
	require.NoError(t, os.WriteFile(apkPath, []byte("verified apk"), 0o600))
	adapter := &verifiedArtifactStrategyFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput, strategy.CapAppLifecycle), existingInstall: true, signer: strings.Repeat("b", 64)}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: "android-adb", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	descriptor := androidartifact.Descriptor{ImmutableRef: "android-debug:" + strings.Repeat("f", 64), Checksum: "sha256:" + strings.Repeat("f", 64), APKChecksum: "sha256:" + strings.Repeat("e", 64), PackageName: "com.example.portal", VersionName: "1.4.0", VersionCode: 14}
	svc.SetAndroidArtifactResolver(&verifiedArtifactResolverFixture{descriptor: descriptor, path: apkPath})
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	flow := Flow{Steps: []Step{{ID: "install", Kind: "install-artifact", RequiredCapabilities: []string{strategy.CapAppLifecycle}, Arguments: map[string]any{"artifact_ref": descriptor.ImmutableRef, "package": descriptor.PackageName, "version_name": descriptor.VersionName}}}}

	result, err := svc.RunWithLeaseConfirmed(context.Background(), flow, "phone", "operator", lease.LeaseToken, true)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Empty(t, adapter.installedBytes, "a mismatched signer must stop before adb install")
	require.False(t, result.Chapters[0].ArtifactInstall.InstallAccepted)
	require.Equal(t, strings.Repeat("b", 64), result.Chapters[0].ArtifactInstall.InstalledSigningCertificateSHA256)
	require.Contains(t, result.Chapters[0].Message, "refusing update")
}

func TestVerifiedArtifactInstallFailsWhenInstalledVersionDoesNotMatch(t *testing.T) {
	apkPath := filepath.Join(t.TempDir(), "verified.apk")
	require.NoError(t, os.WriteFile(apkPath, []byte("verified apk"), 0o600))
	adapter := &verifiedArtifactStrategyFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput, strategy.CapAppLifecycle), version: "9.9.9"}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: "android-adb", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	descriptor := androidartifact.Descriptor{ImmutableRef: "android-debug:" + strings.Repeat("f", 64), Checksum: "sha256:" + strings.Repeat("f", 64), APKChecksum: "sha256:" + strings.Repeat("e", 64), PackageName: "com.example.portal", VersionName: "1.4.0", VersionCode: 14}
	svc.SetAndroidArtifactResolver(&verifiedArtifactResolverFixture{descriptor: descriptor, path: apkPath})
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	flow := Flow{Steps: []Step{{ID: "install", Kind: "install-artifact", RequiredCapabilities: []string{strategy.CapAppLifecycle}, Arguments: map[string]any{"artifact_ref": descriptor.ImmutableRef, "package": descriptor.PackageName, "version_name": descriptor.VersionName}}}}

	result, err := svc.RunWithLeaseConfirmed(context.Background(), flow, "phone", "operator", lease.LeaseToken, true)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.NotNil(t, result.Chapters[0].ArtifactInstall)
	require.True(t, result.Chapters[0].ArtifactInstall.InstallAccepted)
	require.True(t, result.Chapters[0].ArtifactInstall.Installed)
	require.False(t, result.Chapters[0].ArtifactInstall.PostInstallVerified)
	require.Contains(t, result.Chapters[0].Message, "does not match artifact version")
}

func TestVerifiedArtifactInstallFailsWhenInstalledVersionCodeDoesNotMatch(t *testing.T) {
	apkPath := filepath.Join(t.TempDir(), "verified.apk")
	require.NoError(t, os.WriteFile(apkPath, []byte("verified apk"), 0o600))
	adapter := &verifiedArtifactStrategyFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput, strategy.CapAppLifecycle), versionCode: 13}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: "android-adb", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	descriptor := androidartifact.Descriptor{ImmutableRef: "android-debug:" + strings.Repeat("f", 64), Checksum: "sha256:" + strings.Repeat("f", 64), APKChecksum: "sha256:" + strings.Repeat("e", 64), PackageName: "com.example.portal", VersionName: "1.4.0", VersionCode: 14}
	svc.SetAndroidArtifactResolver(&verifiedArtifactResolverFixture{descriptor: descriptor, path: apkPath})
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	flow := Flow{Steps: []Step{{ID: "install", Kind: "install-artifact", RequiredCapabilities: []string{strategy.CapAppLifecycle}, Arguments: map[string]any{"artifact_ref": descriptor.ImmutableRef, "package": descriptor.PackageName, "version_name": descriptor.VersionName}}}}

	result, err := svc.RunWithLeaseConfirmed(context.Background(), flow, "phone", "operator", lease.LeaseToken, true)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	proof := result.Chapters[0].ArtifactInstall
	require.NotNil(t, proof)
	require.True(t, proof.InstallAccepted)
	require.True(t, proof.Installed)
	require.EqualValues(t, 13, proof.InstalledVersionCode)
	require.False(t, proof.PostInstallVerified)
	require.Contains(t, result.Chapters[0].Message, "installed package version code 13 does not match artifact version code 14")
}

func TestVerifiedArtifactInstallRequiresConfirmationAndRecordsPostInstallProof(t *testing.T) {
	apkPath := filepath.Join(t.TempDir(), "verified.apk")
	apkBytes := []byte("verified apk")
	require.NoError(t, os.WriteFile(apkPath, apkBytes, 0o600))
	base := fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput, strategy.CapAppLifecycle)
	adapter := &verifiedArtifactStrategyFixture{Strategy: base}
	db, err := sql.Open("sqlite", "file:verified-artifact-install-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc, err := NewWithDB(strategyregistry.New(adapter), db)
	require.NoError(t, err)
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: "android-adb", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	descriptor := androidartifact.Descriptor{ImmutableRef: "android-debug:" + strings.Repeat("c", 64), Checksum: "sha256:" + strings.Repeat("d", 64), APKChecksum: "sha256:" + strings.Repeat("e", 64), PackageName: "com.example.portal", VersionName: "1.4.0", VersionCode: 14}
	resolver := &verifiedArtifactResolverFixture{descriptor: descriptor, path: apkPath}
	svc.SetAndroidArtifactResolver(resolver)
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	flow := Flow{Steps: []Step{{ID: "install", Kind: "install-artifact", RequiredCapabilities: []string{strategy.CapAppLifecycle}, Arguments: map[string]any{"artifact_ref": descriptor.ImmutableRef, "package": descriptor.PackageName, "version_name": descriptor.VersionName}}}}

	_, err = svc.RunWithLeaseConfirmed(context.Background(), flow, "phone", "operator", lease.LeaseToken, false)
	require.ErrorContains(t, err, "requires explicit confirmation")
	require.Zero(t, resolver.calls)
	require.Empty(t, adapter.installedBytes)

	result, err := svc.RunWithLeaseConfirmed(context.Background(), flow, "phone", "operator", lease.LeaseToken, true)
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Chapters, 1)
	proof := result.Chapters[0].ArtifactInstall
	require.NotNil(t, proof)
	require.True(t, proof.InstallAccepted)
	require.True(t, proof.Installed)
	require.True(t, proof.PostInstallVerified)
	require.Equal(t, descriptor.ImmutableRef, proof.ImmutableRef)
	require.Equal(t, descriptor.Checksum, proof.ArtifactChecksum)
	require.Equal(t, descriptor.APKChecksum, proof.APKChecksum)
	require.Equal(t, strings.Repeat("a", 64), proof.SigningCertificateSHA256)
	require.Equal(t, strings.Repeat("a", 64), proof.InstalledSigningCertificateSHA256)
	require.Equal(t, descriptor.PackageName, proof.PackageName)
	require.Equal(t, descriptor.VersionName, proof.InstalledVersion)
	require.Equal(t, descriptor.VersionCode, proof.InstalledVersionCode)
	require.Equal(t, apkBytes, adapter.installedBytes)
	require.NotEmpty(t, adapter.installedPath)
	require.False(t, strings.Contains(result.Chapters[0].Message, adapter.installedPath), "proof must not expose the private staging path")
	audits := svc.Audit()
	require.Len(t, audits, 1)
	require.Equal(t, descriptor.ImmutableRef, audits[0].Details["immutable_ref"])
	require.Equal(t, descriptor.APKChecksum, audits[0].Details["apk_checksum"])
	require.Equal(t, strings.Repeat("a", 64), audits[0].Details["signing_certificate_sha256"])
	require.Equal(t, strings.Repeat("a", 64), audits[0].Details["installed_signing_certificate_sha256"])
	require.Equal(t, "14", audits[0].Details["version_code"])
	require.Equal(t, "14", audits[0].Details["installed_version_code"])
	require.Equal(t, "true", audits[0].Details["post_install_verified"])
}

func TestAppLifecycleRejectsUnallowlistedIdentityAndUndeclaredAdapter(t *testing.T) {
	fake := fakes.New("lifecycle", strategy.StatusAvailable, strategy.CapAppLifecycle)
	svc := New(strategyregistry.New(fake))
	svc.devices.Upsert(devicedomain.Record{ID: "lifecycle", StrategyID: "lifecycle", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	lease, err := svc.Acquire("lifecycle", "operator", time.Minute)
	require.NoError(t, err)
	_, err = svc.ExecuteAppLifecycle(context.Background(), "lifecycle", AppLifecycleOperation{LeaseToken: lease.LeaseToken, Confirmed: true, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "launch", Package: "not-a-package"}})
	require.ErrorContains(t, err, "fully-qualified application identity")

	unsupported := fakes.New("no-lifecycle", strategy.StatusAvailable, strategy.CapInput)
	noLifecycle := New(strategyregistry.New(unsupported))
	noLifecycle.devices.Upsert(devicedomain.Record{ID: "no-lifecycle", StrategyID: "no-lifecycle", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapInput, Status: strategy.StatusAvailable}}})
	noLease, err := noLifecycle.Acquire("no-lifecycle", "operator", time.Minute)
	require.NoError(t, err)
	_, err = noLifecycle.ExecuteAppLifecycle(context.Background(), "no-lifecycle", AppLifecycleOperation{LeaseToken: noLease.LeaseToken, Confirmed: true, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "launch", Package: "com.example.editor"}})
	require.Error(t, err)
}

type blockingLifecycleFake struct {
	*fakes.Strategy
	deadlines chan time.Time
}

func (s *blockingLifecycleFake) AppLifecycle(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	deadline, ok := ctx.Deadline()
	if ok {
		s.deadlines <- deadline
	}
	<-ctx.Done()
	err := fmt.Errorf("command failed (signal: killed) after context ended: %w", ctx.Err())
	return strategy.AppLifecycleResult{Operation: request.Operation, Package: request.Package, Status: strategy.StatusUnavailable, Reason: err.Error()}, err
}

func TestFailedDirectActuationStillWritesInteractiveAudit(t *testing.T) {
	svc, _ := testService(t)
	fake, ok := svc.registry.Get("fake")
	require.True(t, ok)
	fake.(*fakes.Strategy).ActuateErr = errors.New("fixture transport unavailable")
	lease, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.ActuateDevice(context.Background(), "fake", "operator", lease.LeaseToken, DirectActuation{Key: "DPAD_DOWN"})
	require.ErrorContains(t, err, "fixture transport unavailable")
	audits := svc.Audit()
	require.Len(t, audits, 1)
	require.Equal(t, "failed", audits[0].Outcome)
	require.True(t, audits[0].Interactive)
	require.False(t, audits[0].EvidenceBacked)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}

type framelessAgentStrategy struct {
	*fakes.Strategy
	commands []strategy.MediaCommand
}

func (s *framelessAgentStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"application": {Value: "YouTube", Status: strategy.StatusAvailable, Transport: s.ID()},
	}}, nil
}

func (s *framelessAgentStrategy) ControlMedia(_ context.Context, command strategy.MediaCommand) error {
	s.commands = append(s.commands, command)
	return nil
}

type semanticAgentFixture struct{ *semanticSnapshotFixture }

func (*semanticAgentFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{ScreenState: "on", LockState: "unlocked"}, nil
}

func (*semanticAgentFixture) ResolveSemantic(context.Context, string) (strategy.SemanticResult, error) {
	return strategy.SemanticResult{Bounds: []float64{0, 0, 1, 1}, Confidence: 1}, nil
}

type capturingAgentPlanner struct{ worlds []internalflows.AgentWorld }

func (p *capturingAgentPlanner) Plan(_ context.Context, world internalflows.AgentWorld) (internalflows.AgentPlan, error) {
	p.worlds = append(p.worlds, world)
	if len(p.worlds) == 1 {
		return internalflows.AgentPlan{StepKind: "key", Value: "HOME"}, nil
	}
	return internalflows.AgentPlan{GoalMet: true}, nil
}

func (p *capturingAgentPlanner) PlanWithEvidence(ctx context.Context, world internalflows.AgentWorld) (internalflows.AgentPlan, *internalflows.AgentInferenceEvidence, error) {
	plan, err := p.Plan(ctx, world)
	input, output, cost := int64(41), int64(7), int64(23)
	return plan, &internalflows.AgentInferenceEvidence{Provider: "fixture", Model: "fixture-model", ProviderCallsMade: 1, InputTokens: &input, OutputTokens: &output, CostMicros: &cost, CostProvenance: "USAGE_PROVENANCE_GATEWAY_ESTIMATE"}, err
}

type agentLifecycleStateFixture struct {
	*fakes.Strategy
	foreground         string
	foregroundOnLaunch string
	requests           []strategy.AppLifecycleRequest
}

func (s *agentLifecycleStateFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{ScreenState: "on", LockState: "unlocked", ForegroundPackage: s.foreground}, nil
}

func (s *agentLifecycleStateFixture) AppLifecycle(_ context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	s.requests = append(s.requests, request)
	if request.Operation == "launch" {
		s.foreground = s.foregroundOnLaunch
	}
	return strategy.AppLifecycleResult{Operation: request.Operation, Package: request.Package, Status: strategy.StatusAvailable}, nil
}

type wirelessDebugFlowFixture struct {
	*fakes.Strategy
	changes []bool
	enabled bool
}

func (f *wirelessDebugFlowFixture) SetWirelessDebugging(_ context.Context, enabled bool) error {
	f.changes = append(f.changes, enabled)
	f.enabled = enabled
	return nil
}

func (f *wirelessDebugFlowFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	value := f.enabled
	return strategy.DeviceState{WirelessDebugging: &strategy.WirelessDebuggingState{NativeEnabled: &value}}, nil
}

func TestReusableWirelessDebuggingDefinitionsRunAndVerify(t *testing.T) {
	fixture := &wirelessDebugFlowFixture{Strategy: fakes.New("debug-device", strategy.StatusAvailable, strategy.CapWirelessDebugging)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	for _, name := range []string{"native-wireless-debugging-enable.json", "native-wireless-debugging-disable.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", name))
		require.NoError(t, err)
		var flow Flow
		require.NoError(t, json.Unmarshal(data, &flow))
		require.NoError(t, ValidatePromotionCandidate(flow))
		result, err := svc.Run(context.Background(), flow, "phone", "test")
		require.NoError(t, err)
		require.Equal(t, "passed", result.Disposition, name)
	}
	require.Equal(t, []bool{true, false}, fixture.changes)
}

type sequenceAgentPlanner struct {
	plans  []internalflows.AgentPlan
	calls  int
	worlds [][]string
}

func (p *sequenceAgentPlanner) Plan(_ context.Context, world internalflows.AgentWorld) (internalflows.AgentPlan, error) {
	p.worlds = append(p.worlds, append([]string(nil), world.StepKinds...))
	index := p.calls
	p.calls++
	if index >= len(p.plans) {
		return internalflows.AgentPlan{GoalMet: true}, nil
	}
	return p.plans[index], nil
}

type lifecycleAgentStrategy struct{ *fakes.Strategy }

func (s *lifecycleAgentStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{}, nil
}

type pollingStateStrategy struct{}

func (pollingStateStrategy) ID() string { return "polling-state" }

func (pollingStateStrategy) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{
		StrategyID: "polling-state", Status: strategy.StatusAvailable,
		Capabilities: map[string]strategy.Capability{
			strategy.CapProperty: {Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing},
		},
		Properties:       []strategy.PropertyDescriptor{{Name: "enabled", ValueType: "boolean", StateClass: strategy.StateBearing}},
		StateObservation: strategy.StateObservation{Mode: "poll", Interval: time.Millisecond},
	}, nil
}

func (pollingStateStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"enabled": {Value: true, Status: strategy.StatusAvailable, Transport: "polling-state"},
	}}, nil
}

type serialScopedFlowFixture struct{ *fakes.Strategy }

func (s *serialScopedFlowFixture) ForDevice(serial string) strategy.Strategy {
	if serial != "serial-1" {
		return fakes.New("wrong-device", strategy.StatusUnavailable, strategy.CapInput)
	}
	return s.Strategy
}

type serialScopedAgentFixture struct{ *serialScopedFlowFixture }

func (s *serialScopedAgentFixture) ForDevice(serial string) strategy.Strategy {
	if serial != "serial-1" {
		return fakes.New("wrong-device", strategy.StatusUnavailable, strategy.CapInput)
	}
	return s
}

func (*serialScopedAgentFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{ScreenState: "on", LockState: "unlocked"}, nil
}

type recoveringWirelessStateFixture struct {
	*fakes.Strategy
	online     bool
	fail       bool
	reconnects int
}

func (f *recoveringWirelessStateFixture) Describe(ctx context.Context) (strategy.Declaration, error) {
	if !f.online {
		return strategy.UnavailableDeclaration(f.ID(), "disconnected", []strategy.Capability{{Name: strategy.CapInput}}, "reconnect"), nil
	}
	return f.Strategy.Describe(ctx)
}

func (f *recoveringWirelessStateFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	if !f.online {
		return strategy.DeviceState{}, errors.New("disconnected")
	}
	return strategy.DeviceState{ScreenState: "on"}, nil
}

func (f *recoveringWirelessStateFixture) ReconnectWireless(context.Context) error {
	f.reconnects++
	if f.fail {
		return errors.New("unreachable")
	}
	f.online = true
	return nil
}

func (*recoveringWirelessStateFixture) WirelessEndpoint() string { return "192.0.2.1:5555" }

type stateReaderOverride struct {
	*fakes.PropertyOnly
	state strategy.DeviceState
}

func (s stateReaderOverride) ReadState(context.Context) (strategy.DeviceState, error) {
	return s.state, nil
}

type transientPropertyStrategy struct {
	*fakes.PropertyOnly
	failures int
}

func (s *transientPropertyStrategy) GetProperty(ctx context.Context, name string) (any, error) {
	if s.failures > 0 {
		s.failures--
		return nil, errors.New("transient property read failure")
	}
	return s.PropertyOnly.GetProperty(ctx, name)
}

type failingAttachedReader struct{}

func (failingAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	return nil, context.DeadlineExceeded
}

type staticAttachedReader struct{ devices []AttachedDevice }

type persistentWirelessStrategy struct {
	*fakes.Strategy
	endpoint string
}

func newPersistentWirelessStrategy() *persistentWirelessStrategy {
	return &persistentWirelessStrategy{Strategy: fakes.New("persistent-wireless", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)}
}

func (s *persistentWirelessStrategy) Describe(ctx context.Context) (strategy.Declaration, error) {
	if s.endpoint == "" {
		return strategy.UnavailableDeclaration(s.ID(), "wireless endpoint has not been restored", []strategy.Capability{
			{Name: strategy.CapInput},
			{Name: strategy.CapScreenshot},
		}, "restore a verified wireless endpoint"), nil
	}
	return s.Strategy.Describe(ctx)
}

func (s *persistentWirelessStrategy) Enumerate(context.Context) ([]strategy.Device, error) {
	if s.endpoint == "" {
		return nil, nil
	}
	return []strategy.Device{{
		ID: "wireless-device", Serial: "serial-1", Model: "Test phone", StrategyID: s.ID(),
		Transport: "wireless", Health: strategy.StatusAvailable,
	}}, nil
}

func (s *persistentWirelessStrategy) ForDevice(string) strategy.Strategy { return s }

func (s *persistentWirelessStrategy) PromoteWireless(context.Context) error {
	s.endpoint = "192.168.1.42:5555"
	return nil
}

func (s *persistentWirelessStrategy) WirelessEndpoint() string { return s.endpoint }

func (s *persistentWirelessStrategy) ReconnectWireless(context.Context) error {
	s.endpoint = "192.168.1.43:37123"
	return nil
}

func (s *persistentWirelessStrategy) RestoreWireless(endpoint string) strategy.Strategy {
	return &persistentWirelessStrategy{Strategy: s.Strategy, endpoint: endpoint}
}

func (r staticAttachedReader) List(context.Context) ([]AttachedDevice, error) { return r.devices, nil }

type sequenceAttachedReader struct {
	responses [][]AttachedDevice
	index     int
}

func (r *sequenceAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	if r.index >= len(r.responses) {
		return nil, nil
	}
	response := r.responses[r.index]
	r.index++
	return response, nil
}

type enumeratingFake struct {
	*fakes.Strategy
	devices []strategy.Device
}

func (f *enumeratingFake) Enumerate(context.Context) ([]strategy.Device, error) {
	return f.devices, nil
}

type failingEnumeratingFake struct {
	*fakes.Strategy
	err error
}

func (f *failingEnumeratingFake) Enumerate(context.Context) ([]strategy.Device, error) {
	return nil, f.err
}

type semanticReadinessStrategy struct {
	*fakes.Strategy
	calls   int
	failFor int
	locked  bool
}

func (s *semanticReadinessStrategy) Actuate(ctx context.Context, event strategy.Actuation) error {
	s.calls++
	if s.calls <= s.failFor {
		return errors.New("expected control is not ready")
	}
	return s.Strategy.Actuate(ctx, event)
}

func (s *semanticReadinessStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	lock := "unlocked"
	if s.locked {
		lock = "locked"
	}
	return strategy.DeviceState{ScreenState: "on", LockState: lock}, nil
}

func semanticReadinessService(t *testing.T, failFor int) (*Service, *semanticReadinessStrategy) {
	t.Helper()
	adapter := &semanticReadinessStrategy{Strategy: fakes.New("semantic-ready", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot), failFor: failFor}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "semantic-tv", StrategyID: adapter.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	return svc, adapter
}

func TestSemanticReadinessRetriesReadOnlyAssertionsButNotInput(t *testing.T) {
	svc, adapter := semanticReadinessService(t, 1)
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "ready", Kind: "semantic-assert", Target: "Nooch", Arguments: map[string]any{"expected": "Nooch"}, RetryBudget: 2}}}, "semantic-tv", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, 2, adapter.calls)
	require.Equal(t, 2, result.Chapters[0].Attempts)
	svc, adapter = semanticReadinessService(t, 10)
	result, err = svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "press", Kind: "key", Target: "DPAD_CENTER", RetryBudget: 3}}}, "semantic-tv", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Equal(t, 1, adapter.calls, "input effects must never be replayed by readiness retry")
}

func TestSemanticFailureRetainsUnredactedFrameByDefaultBeforeRestoration(t *testing.T) {
	svc, _ := semanticReadinessService(t, 10)
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "ready", Kind: "semantic-assert", Target: "Nooch", Arguments: map[string]any{"expected": "Nooch"}, RetryBudget: 1}}}, "semantic-tv", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Contains(t, result.Chapters[0].Message, "not ready")
	require.Len(t, result.Evidence, 1)
	require.False(t, result.Evidence[0].RedactionVerified)
	require.Contains(t, result.Evidence[0].AppliedRules, "redaction_not_requested")
	require.False(t, result.Evidence[0].OptedOut)
	require.Len(t, result.Chapters[0].EvidenceIDs, 1)
}

func TestSemanticFailureDoesNotCaptureLockedScreen(t *testing.T) {
	svc, adapter := semanticReadinessService(t, 10)
	adapter.locked = true
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "ready", Kind: "semantic-assert", Target: "Nooch", Arguments: map[string]any{"expected": "Nooch"}}}}, "semantic-tv", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Empty(t, result.Evidence)
}

type foregroundReadinessStrategy struct {
	*semanticReadinessStrategy
	foreground             string
	stateReads, readyAfter int
}

func (s *foregroundReadinessStrategy) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	state, err := s.semanticReadinessStrategy.ReadState(ctx)
	s.stateReads++
	state.ForegroundPackage = s.foreground
	if s.readyAfter > 0 && s.stateReads > s.readyAfter {
		state.ForegroundPackage = "com.example.app"
	}
	return state, err
}

func foregroundReadinessService(t *testing.T, foreground string, readyAfter int) (*Service, *foregroundReadinessStrategy) {
	t.Helper()
	adapter := &foregroundReadinessStrategy{semanticReadinessStrategy: &semanticReadinessStrategy{Strategy: fakes.New("foreground-ready", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)}, foreground: foreground, readyAfter: readyAfter}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "foreground-tv", StrategyID: adapter.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	return svc, adapter
}

func TestForegroundAssertionRequiresActualMatchingObservation(t *testing.T) {
	for _, test := range []struct{ name, foreground, disposition string }{{"matching", "com.example.app", "passed"}, {"different", "com.example.other", "failed"}, {"unknown", "", "failed"}} {
		t.Run(test.name, func(t *testing.T) {
			svc, adapter := foregroundReadinessService(t, test.foreground, 0)
			result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "actual-app", Kind: "assert-foreground", Target: "com.example.app"}}}, "foreground-tv", "operator")
			require.NoError(t, err)
			require.Equal(t, test.disposition, result.Disposition)
			require.Zero(t, adapter.calls, "foreground assertion must never launch or actuate the app")
			if test.disposition == "failed" {
				require.Len(t, result.Evidence, 1)
				require.False(t, result.Evidence[0].RedactionVerified)
				require.Contains(t, result.Evidence[0].AppliedRules, "redaction_not_requested")
			}
		})
	}
}

func TestForegroundReadinessRetriesObservationWithoutRepeatingLaunch(t *testing.T) {
	svc, adapter := foregroundReadinessService(t, "com.example.other", 2)
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "actual-app", Kind: "assert-foreground", Target: "com.example.app", RetryBudget: 2}}}, "foreground-tv", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, 2, result.Chapters[0].Attempts)
	require.Zero(t, adapter.calls)
}

type restartFlowStrategy struct {
	*semanticReadinessStrategy
	restarts int
}

func (s *restartFlowStrategy) RestartDevice(context.Context) error { s.restarts++; return nil }

func TestRestartFlowIsSingleExplicitEffectAndDryRunDoesNotRestart(t *testing.T) {
	for _, dry := range []bool{false, true} {
		adapter := &restartFlowStrategy{semanticReadinessStrategy: &semanticReadinessStrategy{Strategy: fakes.New("restart-ready", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot, strategy.CapDeviceRestart)}}
		svc := New(strategyregistry.New(adapter))
		svc.devices.Upsert(devicedomain.Record{ID: "restart-tv", StrategyID: adapter.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
		result, err := svc.Run(context.Background(), Flow{Transport: "rest", RequireUnlocked: true, SuppressActuation: dry, Steps: []Step{{ID: "restart", Kind: "restart-device"}}}, "restart-tv", "operator")
		require.NoError(t, err)
		require.Equal(t, "passed", result.Disposition)
		expected := 1
		if dry {
			expected = 0
		}
		require.Equal(t, expected, adapter.restarts)
		require.Zero(t, adapter.calls, "restart must use its typed seam rather than generic arbitrary actuation")
	}
}

func TestRestartContractRejectsRetriesArgumentsAndFollowingActions(t *testing.T) {
	valid := Flow{RequireUnlocked: true, Steps: []Step{{ID: "restart", Kind: "restart-device"}}}
	require.NoError(t, validateFlowContract(valid))
	for _, mutate := range []func(*Flow){func(f *Flow) { f.RetryBudget = 1 }, func(f *Flow) { f.Steps[0].RetryBudget = 1 }, func(f *Flow) { f.RequireUnlocked = false }, func(f *Flow) { f.Steps[0].Target = "bootloader" }, func(f *Flow) { f.Steps[0].Arguments = map[string]any{"command": "reboot"} }, func(f *Flow) { f.Steps = append(f.Steps, Step{ID: "after", Kind: "key", Target: "HOME"}) }} {
		candidate := valid
		candidate.Steps = append([]Step(nil), valid.Steps...)
		mutate(&candidate)
		require.Error(t, validateFlowContract(candidate))
	}
}

func TestHomeSetFlowRequiresExplicitPriorAndCannotRetry(t *testing.T) {
	valid := Flow{RequireUnlocked: true, Steps: []Step{{ID: "home", Kind: "home-set", Target: "com.example.daily", Arguments: map[string]any{"value": "com.example.stock/.Home"}}}}
	require.NoError(t, validateFlowContract(valid))
	for _, mutate := range []func(*Flow){func(f *Flow) { f.RetryBudget = 1 }, func(f *Flow) { f.Steps[0].RetryBudget = 1 }, func(f *Flow) { f.RequireUnlocked = false }, func(f *Flow) { f.Steps[0].Arguments = nil }, func(f *Flow) { f.Steps[0].Target = "invalid" }} {
		copy := valid
		copy.Steps = append([]Step(nil), valid.Steps...)
		mutate(&copy)
		require.Error(t, validateFlowContract(copy))
	}
}

func TestCheckedPackageConfigurationPreservesLeaseConfirmationAndUnlockGates(t *testing.T) {
	for _, test := range []struct {
		name                     string
		lease, confirmed, locked bool
		wantCalls                int
	}{{"missing-lease", false, true, false, 0}, {"missing-confirmation", true, false, false, 0}, {"locked", true, true, true, 0}, {"authorized", true, true, false, 1}} {
		t.Run(test.name, func(t *testing.T) {
			adapter := &semanticReadinessStrategy{Strategy: fakes.New("config", strategy.StatusAvailable, strategy.CapAppLifecycle), locked: test.locked}
			svc := New(strategyregistry.New(adapter))
			svc.devices.Upsert(devicedomain.Record{ID: "config", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
			request := AppLifecycleOperation{Actor: "operator", Confirmed: test.confirmed, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "package-enabled-set", Package: "com.example.home", Value: "disabled-user", ExpectedValue: "default"}}
			if test.lease {
				lease, err := svc.Acquire("config", "operator", time.Minute)
				require.NoError(t, err)
				request.LeaseToken = lease.LeaseToken
			}
			_, err := svc.ExecuteAppLifecycle(context.Background(), "config", request)
			if test.wantCalls == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			calls := adapter.LifecycleCalls()
			require.Len(t, calls, test.wantCalls)
			if len(calls) > 0 {
				require.Equal(t, "default", calls[0].ExpectedValue)
				require.Equal(t, "disabled-user", calls[0].Value)
			}
		})
	}
}
