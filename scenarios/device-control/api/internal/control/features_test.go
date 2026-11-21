package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"device-control/internal/androidartifact"
	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
	"device-control/internal/testutil/gherkin"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
)

// The scenarios in features/*.feature are the executable specification of
// transport unification, app launch, the close protocol and reconnect gating.

type featureWorld struct {
	svc *Service
	// transport unification
	castID, adbID string
	confirmation  RelationConfirmation
	confirmErr    error
	capture       RunResult
	failureNext   []string
	ownerAction   string
	// app launch
	candidates      []string
	resolveErr      error
	fixture         *lifecycleFixture
	launch          LaunchAppResult
	observedPackage string
	// close protocol
	state     closeState
	admitted  bool
	removed   bool
	closeDone bool
	// reconnect gate
	gate    *reconnectGate
	now     time.Time
	allowed []bool
	// quick name resolution
	quickMatches []Device
	// pairing close
	install    InstallAppResult
	apkPath    string
	phone      *semanticScreenFixture
	profileIDs []string
	readyErr   error
	readyRes   ScreenResult
	chosen     string
	pairing    *lifecyclePairingSession
	sweep      chan struct{}
}

func newFeatureWorld(t *testing.T) *featureWorld {
	return &featureWorld{now: time.Now().UTC(), gate: &reconnectGate{}}
}

func (w *featureWorld) recordMatching(id, name, strategyID, transport, endpoint string, mutate func(*devicedomain.Record)) devicedomain.Record {
	record := devicedomain.Record{ID: id, Name: name, Kind: "physical", StrategyID: strategyID, Transport: transport, Endpoint: endpoint, Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, LastSeenAt: time.Now().UTC(), ObservedAt: time.Now().UTC()}
	if mutate != nil {
		mutate(&record)
	}
	return record
}

func (w *featureWorld) candidateID(t *testing.T) string {
	for _, candidate := range w.svc.CorrelationCandidates(context.Background()) {
		if (candidate.LeftID == w.castID && candidate.RightID == w.adbID) || (candidate.LeftID == w.adbID && candidate.RightID == w.castID) {
			return candidate.ID
		}
	}
	t.Fatalf("no relation candidate between %s and %s", w.castID, w.adbID)
	return ""
}

func parseDuration(t *testing.T, text string) time.Duration {
	d, err := time.ParseDuration(text)
	require.NoError(t, err)
	return d
}

func TestFeatures(t *testing.T) {
	gherkin.RunFeatures(t, "features", newFeatureWorld, func(s *gherkin.Suite[featureWorld]) {
		registerTransportSteps(s)
		registerLaunchSteps(s)
		registerCloseSteps(s)
		registerReconnectSteps(s)
		registerPairingSteps(s)
		registerUnlockSteps(s)
		registerInstallSteps(s)
	})
}

func registerTransportSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`a Cast TV named "(.+)" at (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc = New(strategyregistry.New(fakes.New("google-cast", strategy.StatusAvailable, strategy.CapMedia)))
		w.castID = "google-cast:tv"
		w.svc.devices.Upsert(w.recordMatching(w.castID, a[0], "google-cast", "cast", a[1]+":8009", nil))
	})
	s.Step(`a phone named "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc = New(strategyregistry.New(fakes.New("x", strategy.StatusAvailable)))
		w.castID = "phone-1"
		w.svc.devices.Upsert(w.recordMatching(w.castID, a[0], "android-adb", "usb", "", nil))
	})
	s.Step(`an ADB endpoint on the same host reporting model "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		claim, _ := identitydomain.NewClaim("adb-serial", "AE70A4D38B", "android-adb", "observed")
		w.adbID = "android-0df"
		w.svc.devices.Upsert(w.recordMatching(w.adbID, a[0], "android-adb", "wireless", "192.168.1.158:5555", func(r *devicedomain.Record) {
			r.Model, r.Serial, r.Claims = a[0], "AE70A4D38B", []identitydomain.IdentityClaim{claim}
		}))
	})
	s.Step(`the owner confirms their same-host relation`, func(t *testing.T, w *featureWorld, _ []string) {
		w.confirmation, w.confirmErr = w.svc.ConfirmRelation(context.Background(), w.candidateID(t), false)
	})
	s.Step(`the owner forces the confirmation`, func(t *testing.T, w *featureWorld, _ []string) {
		w.confirmation, w.confirmErr = w.svc.ConfirmRelation(context.Background(), w.candidateID(t), true)
	})
	s.Step(`the owner confirms the candidate "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		w.confirmation, w.confirmErr = w.svc.ConfirmRelation(context.Background(), a[0], false)
	})
	s.Step(`the Cast TV is the canonical device`, func(t *testing.T, w *featureWorld, _ []string) {
		require.NoError(t, w.confirmErr)
		require.Equal(t, w.castID, w.confirmation.CanonicalID)
	})
	s.Step(`the ADB identity no longer exists on its own`, func(t *testing.T, w *featureWorld, _ []string) {
		_, found := w.svc.devices.Get(w.adbID)
		require.False(t, found)
	})
	s.Step(`the canonical device has an "(.+)" transport`, func(t *testing.T, w *featureWorld, a []string) {
		var got []string
		for _, profile := range w.confirmation.Device.Transports {
			got = append(got, profile.StrategyID)
		}
		require.Contains(t, got, a[0])
	})
	s.Step(`the asserted claim is "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0], w.confirmation.Claim)
	})
	s.Step(`the confirmation is refused with "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.ErrorContains(t, w.confirmErr, a[0])
	})
	s.Step(`the relation is confirmed`, func(t *testing.T, w *featureWorld, _ []string) {
		require.NoError(t, w.confirmErr)
		require.True(t, w.confirmation.Confirmed)
	})
	s.Step(`a screenshot of the Cast TV is requested`, func(t *testing.T, w *featureWorld, _ []string) {
		var err error
		w.capture, err = w.svc.Capture(context.Background(), CaptureRequest{Device: w.castID, Kind: "screenshot", Actor: "test"})
		require.NoError(t, err)
	})
	s.Step(`the capture is a capability gap`, func(t *testing.T, w *featureWorld, _ []string) {
		require.Equal(t, "capability_gap", w.capture.Disposition)
	})
	s.Step(`the first next action starts with "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.NotEmpty(t, w.capture.NextActions)
		require.True(t, strings.HasPrefix(w.capture.NextActions[0], a[0]), w.capture.NextActions[0])
	})
	s.Step(`the first next action contains "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.NotEmpty(t, w.capture.NextActions)
		require.Contains(t, w.capture.NextActions[0], a[0])
	})
	s.Step(`a capture fails with the reason (.+) and failure class (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		class := a[1]
		if class == "none" {
			class = ""
		}
		w.failureNext = captureFailureNextActions(Device{ID: w.castID, Name: "moto"}, []Chapter{{Message: a[0], FailureClass: class}}, "UNLOCK-HINT")
	})
	s.Step(`the next action contains (.+)`, func(t *testing.T, w *featureWorld, a []string) {
		require.NotEmpty(t, w.failureNext)
		require.Contains(t, w.failureNext[0], a[0])
	})
	s.Step(`the ADB transport on "(.+)" over (\S+) is (\S+) because "(.*)"`, func(t *testing.T, w *featureWorld, a []string) {
		reason := a[3]
		if reason == "none" {
			reason = ""
		}
		w.ownerAction = adbOwnerAction(a[1], a[2], reason, a[0], "tv")
	})
	s.Step(`the owner action contains (.+)`, func(t *testing.T, w *featureWorld, a []string) {
		if a[0] == "nothing" {
			require.Empty(t, w.ownerAction)
			return
		}
		require.Contains(t, w.ownerAction, a[0])
	})
}

func registerLaunchSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`the app name "?([^"]+)"? is resolved`, func(t *testing.T, w *featureWorld, a []string) {
		w.candidates, w.resolveErr = appCandidates(a[0])
	})
	s.Step(`the candidate packages are (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		require.NoError(t, w.resolveErr)
		require.Equal(t, strings.Split(a[0], ","), w.candidates)
	})
	s.Step(`resolution fails with "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.ErrorContains(t, w.resolveErr, a[0])
	})
	s.Step(`a device with "(.+)" installed`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc, w.fixture = launchService(t, a[0])
	})
	s.Step(`a TV whose default Cast transport cannot launch apps but whose ADB transport can, with "(.+)" installed`, func(t *testing.T, w *featureWorld, a []string) {
		w.fixture = &lifecycleFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapAppLifecycle), installed: a[0]}
		cast := fakes.New("google-cast", strategy.StatusAvailable, strategy.CapMedia)
		w.svc = New(strategyregistry.New(cast, w.fixture))
		now := time.Now().UTC()
		profiles := []strategy.DeviceTransport{
			{StrategyID: "google-cast", Name: "cast", Health: strategy.StatusAvailable, ObservedAt: now},
			{StrategyID: "android-adb", Name: "adb", Health: strategy.StatusAvailable, ObservedAt: now},
		}
		w.svc.devices.Upsert(devicedomain.Record{ID: "target", Name: "TV", Kind: "physical", StrategyID: "google-cast", Transport: "cast", Transports: profiles, Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, ObservedAt: now})
	})
	s.Step(`a device with nothing installed`, func(t *testing.T, w *featureWorld, _ []string) {
		w.svc, w.fixture = launchService(t, "")
	})
	s.Step(`"(.+)" is opened`, func(t *testing.T, w *featureWorld, a []string) {
		var err error
		w.launch, err = w.svc.LaunchApp(context.Background(), LaunchAppRequest{Device: "target", App: a[0], Actor: "test"})
		require.NoError(t, err)
	})
	s.Step(`the launch status is "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0], w.launch.Status)
	})
	s.Step(`the device reports resumed activity (\S+) and focused window (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		none := func(v string) string {
			if v == "none" {
				return ""
			}
			return v
		}
		w.observedPackage = observedForegroundPackage(strategy.AppLifecycleResult{ResumedActivity: none(a[0]), FocusedWindowActivity: none(a[1])})
	})
	s.Step(`the observed foreground package is (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		want := a[0]
		if want == "nothing" {
			want = ""
		}
		require.Equal(t, want, w.observedPackage)
	})
	s.Step(`the launched package is "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0], w.launch.Package)
	})
	s.Step(`no lease remains held`, func(t *testing.T, w *featureWorld, _ []string) {
		require.Empty(t, w.svc.ListLiveSessions())
	})
	s.Step(`no launch was issued`, func(t *testing.T, w *featureWorld, _ []string) {
		require.Empty(t, w.fixture.foreground)
	})
}

func registerCloseSteps(s *gherkin.Suite[featureWorld]) {
	states := map[string]closeState{"open": stateOpen, "closing": stateClosing, "uncertain": stateUncertain}
	s.Step(`a resource that is (open|closing|uncertain)`, func(t *testing.T, w *featureWorld, a []string) { w.state = states[a[0]] })
	s.Step(`a close begins`, func(t *testing.T, w *featureWorld, _ []string) { w.state, w.admitted = w.state.beginClose() })
	s.Step(`the close is (admitted|refused)`, func(t *testing.T, w *featureWorld, a []string) { require.Equal(t, a[0] == "admitted", w.admitted) })
	s.Step(`the close finishes with (success|failure)`, func(t *testing.T, w *featureWorld, a []string) {
		var err error
		if a[0] == "failure" {
			err = errors.New("close failed")
		}
		w.state, w.removed = w.state.closeResult(err)
	})
	s.Step(`the resource is (open|closing|uncertain|removed)`, func(t *testing.T, w *featureWorld, a []string) {
		if a[0] == "removed" {
			require.True(t, w.removed)
			return
		}
		require.False(t, w.removed)
		require.Equal(t, states[a[0]], w.state)
	})
}

func registerReconnectSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`a reconnect is requested twice at once`, func(t *testing.T, w *featureWorld, _ []string) {
		w.allowed = []bool{w.gate.allow("tv", w.now), w.gate.allow("tv", w.now)}
	})
	s.Step(`only the first is admitted`, func(t *testing.T, w *featureWorld, _ []string) { require.Equal(t, []bool{true, false}, w.allowed) })
	s.Step(`a failed reconnect just now`, func(t *testing.T, w *featureWorld, _ []string) {
		require.True(t, w.gate.allow("tv", w.now))
		w.gate.record("tv", w.now, false)
	})
	s.Step(`a successful reconnect just now`, func(t *testing.T, w *featureWorld, _ []string) {
		require.True(t, w.gate.allow("tv", w.now))
		w.gate.record("tv", w.now, true)
	})
	s.Step(`no reconnect gate`, func(t *testing.T, w *featureWorld, _ []string) { w.gate = nil })
	s.Step(`a reconnect is requested before the backoff ends`, func(t *testing.T, w *featureWorld, _ []string) {
		w.allowed = append(w.allowed, w.gate.allow("tv", w.now.Add(reconnectBackoff-time.Second)))
	})
	s.Step(`a reconnect is requested after the backoff ends`, func(t *testing.T, w *featureWorld, _ []string) {
		w.allowed = append(w.allowed, w.gate.allow("tv", w.now.Add(reconnectBackoff+time.Second)))
	})
	s.Step(`(?:it|the request) is (admitted|refused)`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0] == "admitted", w.allowed[len(w.allowed)-1])
	})
	s.Step(`a device "(.+)" observed (\S+) ago and (available|unreachable)`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc = New(strategyregistry.New(fakes.New("x", strategy.StatusAvailable)))
		w.svc.devices.Upsert(devicedomain.Record{ID: "tv", Name: a[0], Kind: "physical", Health: a[2], Status: a[2], ObservedAt: w.now.Add(-parseDuration(t, a[1]))})
	})
	s.Step(`the name "(.+)" is resolved without inventory`, func(t *testing.T, w *featureWorld, a []string) {
		w.quickMatches = w.svc.freshDevicesNamed(strings.ToLower(a[0]), w.now)
	})
	s.Step(`the quick resolution (succeeds|declines)`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0] == "succeeds", len(w.quickMatches) == 1)
	})
}

func registerPairingSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`an expired pairing session whose close will (succeed|fail)`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc = newService(nil, nil)
		w.pairing = &lifecyclePairingSession{started: make(chan struct{}), release: make(chan struct{})}
		if a[0] == "fail" {
			w.pairing.closeErr = errors.New("close uncertain")
		}
		w.svc.mu.Lock()
		w.svc.pendingPairings["pairing-expired"] = pendingPairing{deviceID: "tv", session: w.pairing, expiresAt: time.Now().Add(-time.Second)}
		w.svc.mu.Unlock()
	})
	s.Step(`the expiry sweep starts closing it`, func(t *testing.T, w *featureWorld, _ []string) {
		w.sweep = make(chan struct{})
		go func() {
			w.svc.expirePairing("pairing-expired")
			close(w.sweep)
		}()
		<-w.pairing.started
	})
	s.Step(`the close returns`, func(t *testing.T, w *featureWorld, _ []string) {
		close(w.pairing.release)
		<-w.sweep
	})
	s.Step(`the lifecycle inventory shows the pairing as "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		work, blockers, complete := w.svc.LifecycleInventory(context.Background())
		require.True(t, complete)
		require.NotEmpty(t, blockers)
		require.Equal(t, a[0], lifecyclePairingStatus(work, "pairing/pairing-expired"))
	})
	s.Step(`the lifecycle inventory shows no pairing owner`, func(t *testing.T, w *featureWorld, _ []string) {
		work, blockers, complete := w.svc.LifecycleInventory(context.Background())
		require.True(t, complete)
		require.Empty(t, blockers)
		require.Empty(t, lifecyclePairingStatus(work, "pairing/pairing-expired"))
	})
	s.Step(`completing the pairing is refused`, func(t *testing.T, w *featureWorld, _ []string) {
		_, err := w.svc.CompletePairDevice(context.Background(), "tv", "pairing-expired", []byte("835B64"))
		require.Error(t, err)
	})
	s.Step(`the session has been closed exactly once`, func(t *testing.T, w *featureWorld, _ []string) {
		require.EqualValues(t, 1, w.pairing.closes.Load())
	})
}

func registerUnlockSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`a locked phone "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		w.phone = &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "off", LockState: "locked"}}
		w.svc = New(strategyregistry.New(w.phone))
		w.svc.devices.Upsert(devicedomain.Record{ID: a[0], Name: "moto", Kind: "physical", StrategyID: w.phone.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
		store, err := authdomain.NewStore(nil, &flowAuthResolver{value: "1234"})
		require.NoError(t, err)
		w.svc.auth = store
		w.castID = a[0]
	})
	create := func(t *testing.T, w *featureWorld, id, outcome string) {
		p, err := w.svc.auth.Create(context.Background(), authdomain.Profile{ID: id, DeviceID: w.castID, Method: authdomain.MethodPIN, CredentialIdentity: authdomain.CredentialNamespace + w.castID + "/" + id, CredentialField: "unlock", Verification: "fresh_lock_state_unlocked"})
		require.NoError(t, err)
		if outcome != "none" {
			w.svc.auth.RecordOutcome(context.Background(), p.ID, outcome)
		}
	}
	s.Step(`an active unlock profile whose last outcome was "(.+)"`, func(t *testing.T, w *featureWorld, a []string) { create(t, w, "profile", a[0]) })
	s.Step(`the phone is made ready`, func(t *testing.T, w *featureWorld, _ []string) {
		w.readyRes, w.readyErr = w.svc.Screen(context.Background(), ScreenRequest{Device: w.castID, Operation: "ready"})
	})
	s.Step(`the phone is unlocked`, func(t *testing.T, w *featureWorld, _ []string) {
		require.NoError(t, w.readyErr)
		require.Equal(t, "unlocked", w.readyRes.After.LockState)
	})
	s.Step(`the result never contains the PIN`, func(t *testing.T, w *featureWorld, _ []string) {
		require.NotContains(t, string(mustJSON(t, w.readyRes)), "1234")
		require.NotContains(t, string(mustJSON(t, w.svc.Audit())), "1234")
	})
	s.Step(`making it ready is refused as still locked`, func(t *testing.T, w *featureWorld, _ []string) {
		require.ErrorContains(t, w.readyErr, "unlocked device")
	})
	s.Step(`the unlock guidance contains "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.Contains(t, w.svc.unlockSetupAction(context.Background(), Device{ID: w.castID, Name: "moto"}), a[0])
	})
	s.Step(`profiles for "(.+)" with last outcomes (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		w.svc = New(strategyregistry.New(fakes.New("x", strategy.StatusAvailable)))
		store, err := authdomain.NewStore(nil, &flowAuthResolver{value: "1234"})
		require.NoError(t, err)
		w.svc.auth, w.castID = store, a[0]
		for i, outcome := range strings.Split(a[1], ",") {
			create(t, w, fmt.Sprintf("p%d", i), outcome)
			time.Sleep(2 * time.Millisecond)
		}
	})
	s.Step(`the usable profile outcome is (\S+)`, func(t *testing.T, w *featureWorld, a []string) {
		profile, reason := authdomain.UnlockCandidate(w.svc.auth.List(context.Background()), w.castID)
		got := profile.LastOutcome
		if reason != "" {
			got = "none"
		} else if got == "" {
			got = "untried"
		}
		require.Equal(t, a[0], got)
	})
}

type fixedAPK struct{ identity androidartifact.APKIdentity }

func (f fixedAPK) Inspect(context.Context, string) (androidartifact.APKIdentity, error) {
	return f.identity, nil
}

func registerInstallSteps(s *gherkin.Suite[featureWorld]) {
	s.Step(`a phone that can install and screenshot, reporting version code (\d+) after an install`, func(t *testing.T, w *featureWorld, a []string) {
		code, _ := strconv.ParseInt(a[0], 10, 64)
		w.fixture = &lifecycleFixture{Strategy: fakes.New("launch-fixture", strategy.StatusAvailable, strategy.CapAppLifecycle, strategy.CapScreenshot), installVersion: code}
		w.svc = New(strategyregistry.New(w.fixture))
		w.svc.devices.Upsert(devicedomain.Record{ID: "target", Name: "moto", Kind: "physical", StrategyID: w.fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	})
	s.Step(`the device accepts an install but installs nothing`, func(t *testing.T, w *featureWorld, _ []string) { w.fixture.installRefused = true })
	s.Step(`an APK for "(.+)" version code (\d+)`, func(t *testing.T, w *featureWorld, a []string) {
		code, _ := strconv.ParseInt(a[1], 10, 64)
		w.svc.apkInspector = fixedAPK{androidartifact.APKIdentity{PackageName: a[0], VersionName: "1.0", VersionCode: code}}
		w.apkPath = filepath.Join(t.TempDir(), "app.apk")
		require.NoError(t, os.WriteFile(w.apkPath, []byte("apk"), 0o600))
	})
	s.Step(`a file at "(.+)" that is not an APK path`, func(t *testing.T, w *featureWorld, a []string) { w.apkPath = a[0] })
	s.Step(`the APK is installed(?: (and opened and screenshotted))?`, func(t *testing.T, w *featureWorld, a []string) {
		var err error
		w.install, err = w.svc.InstallApp(context.Background(), InstallAppRequest{Device: "target", Actor: "test", APK: w.apkPath, Open: a[0] != "", Screenshot: a[0] != ""})
		require.NoError(t, err)
	})
	s.Step(`installing it is refused with "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		_, err := w.svc.InstallApp(context.Background(), InstallAppRequest{Device: "target", Actor: "test", APK: w.apkPath})
		require.ErrorContains(t, err, a[0])
		require.Zero(t, w.fixture.installs, "nothing was sent to the device")
	})
	s.Step(`the install status is "(.+)"`, func(t *testing.T, w *featureWorld, a []string) {
		require.Equal(t, a[0], w.install.Status, w.install.Detail)
	})
	s.Step(`the install detail contains "(.+)"`, func(t *testing.T, w *featureWorld, a []string) { require.Contains(t, w.install.Detail, a[0]) })
	s.Step(`the installed app is in the foreground`, func(t *testing.T, w *featureWorld, _ []string) {
		require.Equal(t, "foreground_verified", w.install.LaunchStatus)
		require.Equal(t, w.install.Package, w.install.ForegroundPackage)
	})
	s.Step(`a screenshot was retained`, func(t *testing.T, w *featureWorld, _ []string) { require.Len(t, w.install.EvidenceIDs, 1) })
	s.Step(`no screenshot was taken`, func(t *testing.T, w *featureWorld, _ []string) { require.Empty(t, w.install.EvidenceIDs) })
}
