package androidtvremote

import (
	"context"
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

func (f *fakePairingClient) Pair(_ context.Context, device Device, pin string) ([]byte, error) {
	f.serial, f.pin = device.Serial, pin
	return []byte("fixture-certificate"), nil
}

type fakeCertificateStore struct {
	serial      string
	certificate []byte
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
	require.Equal(t, "google-tv-serial", devices[0].IdentityKey)
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
	require.NoError(t, s.Pair(context.Background(), "tv-serial", "123456"))
	require.Equal(t, "tv-serial", pairing.serial)
	require.Equal(t, "123456", pairing.pin)
	require.Equal(t, "tv-serial", store.serial)
	require.Equal(t, []byte("fixture-certificate"), store.certificate)
}
