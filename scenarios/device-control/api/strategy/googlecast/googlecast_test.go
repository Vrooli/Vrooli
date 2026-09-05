package googlecast

import (
	"context"
	"testing"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

type fixtureClient struct {
	status ReceiverStatus
	volume float64
	muted  bool
	media  []string
}

func (f *fixtureClient) Status(context.Context) (ReceiverStatus, error) { return f.status, nil }
func (f *fixtureClient) SetVolume(_ context.Context, value float64) error {
	f.volume = value
	return nil
}

func (f *fixtureClient) SetMuted(_ context.Context, value bool) error {
	f.muted = value
	return nil
}
func (f *fixtureClient) Launch(context.Context, string) error { return nil }
func (f *fixtureClient) Media(_ context.Context, action string) error {
	f.media = append(f.media, action)
	return nil
}
func (f *fixtureClient) Observe(context.Context, Observer) error { return nil }
func (f *fixtureClient) Close() error                            { return nil }

func TestGoogleCastDeclaresStateBearingPropertiesAndReadsThem(t *testing.T) {
	client := &fixtureClient{status: ReceiverStatus{Application: "YouTube", PlayerState: "PLAYING", Volume: 0.4, Muted: false}}
	s := New(WithClient(client), WithDevices(Device{ID: "cast-1", IdentityKey: "cast-1", Endpoint: "192.168.1.42:8009"}))
	declaration, err := s.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapProperty].Status)
	require.Equal(t, strategy.StatusUnavailable, declaration.Capabilities[strategy.CapInput].Status)
	require.Equal(t, "push", declaration.StateObservation.Mode)
	require.NotEmpty(t, declaration.Properties)

	state, err := s.ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "YouTube", state.Properties["application"].Value)
	require.Equal(t, 0.4, state.Properties["volume"].Value)
	require.NotContains(t, state.Properties, "input")
	require.NotContains(t, state.Properties, "power")
	require.Equal(t, "Cast receiver status does not report an input source", state.Unavailable["input"])
	require.Equal(t, "Cast receiver status does not expose physical display power state", state.Unavailable["power"])
	require.NoError(t, s.SetProperty(context.Background(), strategy.PropertySet{Name: "volume", Value: 0.7}))
	require.Equal(t, 0.7, client.volume)
	require.NoError(t, s.ControlMedia(context.Background(), strategy.MediaCommand{Action: "pause"}))
	require.Equal(t, []string{"pause"}, client.media)
}

func TestGoogleCastConformanceCoversFramelessStateAndMediaTransport(t *testing.T) {
	client := &fixtureClient{}
	report := strategy.Verify(context.Background(), New(WithClient(client)))
	require.Empty(t, report.Failed)
	require.Contains(t, report.Passed, "screenless-floor")
	require.Contains(t, report.Passed, "state-reader")
	require.Contains(t, report.Passed, "state-observation")
	require.Contains(t, report.Passed, "media")
	require.Contains(t, report.Passed, "property")
}

func TestStatusDiffPublishesOnlyChangedAttributes(t *testing.T) {
	var events []strategy.StateChangeEvent
	emitStatusDiff(ReceiverStatus{Application: "YouTube", PlayerState: "PLAYING", Volume: 0.4, Muted: false}, ReceiverStatus{Application: "Netflix", PlayerState: "PLAYING", Volume: 0.6, Muted: false}, func(event strategy.StateChangeEvent) {
		events = append(events, event)
	})
	require.Len(t, events, 2)
	require.ElementsMatch(t, []string{"volume", "application"}, []string{events[0].Attribute, events[1].Attribute})
	for _, event := range events {
		require.Equal(t, strategy.StateBearing, event.StateClass)
		require.NotEqual(t, event.OldValue, event.NewValue)
	}
}

func TestGoogleCastEnumerationPreservesHardwareIdentity(t *testing.T) {
	s := New(WithDiscovery(fixtureDiscovery{devices: []Device{{ID: "cast-1", IdentityKey: "bt-1", Name: "Living Room", Endpoint: "192.168.1.42:8009"}}}))
	devices, err := s.Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "bt-1", devices[0].IdentityKey)
	require.Equal(t, "google-cast:bt-1", devices[0].ID)
}

type fixtureDiscovery struct{ devices []Device }

func (f fixtureDiscovery) DiscoverCast(context.Context) ([]Device, error) { return f.devices, nil }
