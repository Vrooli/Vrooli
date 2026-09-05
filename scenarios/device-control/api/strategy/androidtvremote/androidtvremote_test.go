package androidtvremote

import (
	"context"
	"net"
	"testing"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	keys  []string
	texts []string
	media []strategy.MediaCommand
}

func (f *fakeClient) Key(_ context.Context, key string) error {
	f.keys = append(f.keys, key)
	return nil
}

func (f *fakeClient) Text(_ context.Context, text string) error {
	f.texts = append(f.texts, text)
	return nil
}

func (f *fakeClient) Media(_ context.Context, command strategy.MediaCommand) error {
	f.media = append(f.media, command)
	return nil
}

type fakePairingClient struct {
	serial string
	pin    string
}

type fakePairingSession struct {
	pin    string
	closed bool
}

func (f *fakePairingSession) Complete(_ context.Context, pin string) ([]byte, error) {
	f.pin = pin
	return []byte("fixture-certificate"), nil
}

func (f *fakePairingSession) Close() error {
	f.closed = true
	return nil
}

type fakeInteractivePairingClient struct{ session *fakePairingSession }

func (f *fakeInteractivePairingClient) Pair(context.Context, Device, string) ([]byte, error) {
	return nil, nil
}

func (f *fakeInteractivePairingClient) Begin(context.Context, Device) (PairingSession, error) {
	f.session = &fakePairingSession{}
	return f.session, nil
}

func (f *fakePairingClient) Pair(_ context.Context, device Device, pin string) ([]byte, error) {
	f.serial, f.pin = device.Serial, pin
	return []byte("fixture-certificate"), nil
}

type fakeCertificateStore struct {
	serial      string
	certificate []byte
}

type fixtureDiscovery []Device

func (f fixtureDiscovery) DiscoverMDNS(context.Context) ([]Device, error) {
	return append([]Device(nil), f...), nil
}

func (f *fakeCertificateStore) SavePairingCertificate(_ context.Context, serial string, certificate []byte) error {
	f.serial, f.certificate = serial, append([]byte(nil), certificate...)
	return nil
}

func TestFixtureGoogleTVUsesStableSerialAndTypedCommands(t *testing.T) {
	client := &fakeClient{}
	s := New(WithDevices(Device{Serial: "google-tv-serial", Name: "Living Room TV", Endpoint: "tv.local:6466"}), WithClient(client))
	devices, err := s.Enumerate(context.Background())
	require.NoError(t, err)
	require.Equal(t, "android-tv:google-tv-serial", devices[0].ID)
	// A serial-like value without an explicit accepted claim is only a
	// transport selector; it must not become durable identity evidence.
	require.Empty(t, devices[0].IdentityKey)
	require.NoError(t, s.ForDevice("google-tv-serial").(interface {
		Actuate(context.Context, strategy.Actuation) error
	}).Actuate(context.Background(), strategy.Actuation{Key: &strategy.KeyEvent{Key: "DPAD_CENTER"}}))
	require.NoError(t, s.ForDevice("google-tv-serial").(interface {
		Actuate(context.Context, strategy.Actuation) error
	}).Actuate(context.Background(), strategy.Actuation{Text: "hello TV"}))
	require.NoError(t, s.ControlMedia(context.Background(), strategy.MediaCommand{Action: "play", CausationID: "cause-1"}))
	require.Equal(t, []string{"DPAD_CENTER"}, client.keys)
	require.Equal(t, []string{"hello TV"}, client.texts)
	require.Equal(t, "cause-1", client.media[0].CausationID)
	declaration, err := s.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, declaration.Capabilities[strategy.CapScreenshot].Status)
	require.Contains(t, strategy.StepKinds(declaration), "media-play")
}

func TestPairPerformsProtocolExchangeAndPersistsCertificate(t *testing.T) {
	pairing := &fakePairingClient{}
	store := &fakeCertificateStore{}
	s := New(WithDevices(Device{Serial: "tv-serial", Endpoint: "tv.local:6466"}), WithPairingClient(pairing), WithPairingStore(store))
	result, err := s.ForDevice("tv-serial").(strategy.Pairer).Pair(context.Background(), strategy.PairRequest{Secret: []byte("123456")})
	require.NoError(t, err)
	require.Equal(t, "paired", result.Outcome)
	require.Equal(t, "tv-serial", pairing.serial)
	require.Equal(t, "123456", pairing.pin)
	require.Equal(t, "tv-serial", store.serial)
	require.Equal(t, []byte("fixture-certificate"), store.certificate)
}

func TestInteractivePairingBeginsBeforePINAndCompletesSession(t *testing.T) {
	pairing := &fakeInteractivePairingClient{}
	store := &fakeCertificateStore{}
	s := New(WithDevices(Device{Serial: "tv-serial", Endpoint: "tv.local:6466"}), WithPairingClient(pairing), WithPairingStore(store))
	scoped := s.ForDevice("tv-serial").(*Strategy)
	session, err := scoped.BeginPairing(context.Background())
	require.NoError(t, err)
	require.NotNil(t, pairing.session)
	require.Empty(t, pairing.session.pin, "the PIN must not be consumed during handshake startup")

	result, err := scoped.CompletePairing(context.Background(), session, []byte("123456"))
	require.NoError(t, err)
	require.Equal(t, "paired", result.Outcome)
	require.Equal(t, "123456", pairing.session.pin)
	require.True(t, pairing.session.closed)
	require.Equal(t, "tv-serial", store.serial)
}

func TestAndroidTVRemoteConformanceRequiresSecretSafePairer(t *testing.T) {
	report := strategy.Verify(context.Background(), New(WithClient(&fakeClient{})))
	require.Empty(t, report.Failed)
	require.Contains(t, report.Passed, "pairing")
	require.Equal(t, strategy.StatusUnavailable, reportStatus(New(), strategy.CapScreenshot))
}

func TestUnboundAndroidTVRemoteConformanceUsesNonNetworkTarget(t *testing.T) {
	report := strategy.Verify(context.Background(), New())
	require.Empty(t, report.Failed)
	require.Contains(t, report.Passed, "actuate")
}

func TestAndroidTVRemoteDiscoveryFormatsIPv6Endpoints(t *testing.T) {
	endpoint := formatEndpoint(net.ParseIP("fe80::1"), 6466)
	if endpoint != "[fe80::1]:6466" {
		t.Fatalf("net.JoinHostPort() = %q, want bracketed IPv6 endpoint", endpoint)
	}
	require.Equal(t, "[fe80::1%enp0s31f6]:6466", formatEndpointForInterface(net.ParseIP("fe80::1"), 6466, "enp0s31f6"))
}

func TestEndpointScopedRestoredPairingFallsBackToDurableSerial(t *testing.T) {
	s := New(WithDiscovery(fixtureDiscovery{
		{Serial: "tv-serial", Endpoint: "fresh-tv.local:6466"},
	})).ForDevice("tv-serial").(*Strategy)
	s = s.ForEndpoint("stale-tv.local:6466").(*Strategy)

	target, err := s.targetDevice(context.Background())
	require.NoError(t, err)
	require.Equal(t, "tv-serial", target.Serial)
	require.Equal(t, "fresh-tv.local:6466", target.Endpoint)
}

func reportStatus(s *Strategy, capability string) string {
	declaration, _ := s.Describe(context.Background())
	return declaration.Capabilities[capability].Status
}
