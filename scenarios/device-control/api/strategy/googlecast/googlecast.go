// Package googlecast implements the state-bearing Google Cast receiver
// transport. Cast is deliberately complementary to Android TV Remote: it
// reports receiver/media state and absolute volume, while Remote owns keys.
package googlecast

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
	mdns "github.com/vrooli/mdns-go"
)

const defaultPort = 8009

type Device struct {
	ID, Name, Model, Endpoint string
	IdentityKey               string
	IdentityKind              string
}

type ReceiverStatus struct {
	Application string  `json:"application,omitempty"`
	TransportID string  `json:"transport_id,omitempty"`
	PlayerState string  `json:"player_state,omitempty"`
	Volume      float64 `json:"volume"`
	Muted       bool    `json:"muted"`
	MediaTitle  string  `json:"media_title,omitempty"`
	MediaArtist string  `json:"media_artist,omitempty"`
}

type Observer func(strategy.StateChangeEvent)

type Client interface {
	Status(context.Context) (ReceiverStatus, error)
	SetVolume(context.Context, float64) error
	SetMuted(context.Context, bool) error
	Launch(context.Context, string) error
	Media(context.Context, string) error
	Observe(context.Context, Observer) error
	Close() error
}

type Discovery interface {
	DiscoverCast(context.Context) ([]Device, error)
}

type Strategy struct {
	devices  []Device
	serial   string
	endpoint string
	client   Client
	discover Discovery
	interval time.Duration
}

type Option func(*Strategy)

func WithDevices(devices ...Device) Option {
	return func(s *Strategy) { s.devices = append([]Device(nil), devices...) }
}
func WithClient(client Client) Option          { return func(s *Strategy) { s.client = client } }
func WithDiscovery(discovery Discovery) Option { return func(s *Strategy) { s.discover = discovery } }
func WithObservationInterval(interval time.Duration) Option {
	return func(s *Strategy) { s.interval = interval }
}

func New(options ...Option) *Strategy {
	s := &Strategy{interval: 2 * time.Second}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Strategy) ID() string { return "google-cast" }

func (s *Strategy) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{
		StrategyID: s.ID(), Description: "Google Cast receiver transport", Status: strategy.StatusAvailable,
		SupportedHostOS: []string{"linux", "macos", "windows"}, Promotable: false, EvidenceClass: "transport-fixture",
		ObservationMode: "push", ObservationInterval: s.interval,
		StateObservation: strategy.StateObservation{Mode: "push", Interval: s.interval},
		Capabilities: map[string]strategy.Capability{
			strategy.CapMedia:        {Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing, ProbeEvidence: "Cast receiver and media namespaces"},
			strategy.CapProperty:     {Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing, ProbeEvidence: "Cast receiver status properties"},
			strategy.CapSensor:       {Name: strategy.CapSensor, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing, ProbeEvidence: "Cast receiver status observation"},
			strategy.CapInput:        {Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: "Google Cast does not expose directional input", NextAction: "Use the Android TV Remote transport for navigation"},
			strategy.CapScreenshot:   {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "Google Cast does not expose a screen capture modality", NextAction: "Use a screen-bearing transport when a frame is required"},
			strategy.CapSemanticTree: {Name: strategy.CapSemanticTree, Status: strategy.StatusUnavailable, Reason: "Google Cast does not expose a semantic tree"},
		},
		Properties: propertyDescriptors(),
	}, nil
}

func propertyDescriptors() []strategy.PropertyDescriptor {
	zero, one := 0.0, 1.0
	return []strategy.PropertyDescriptor{
		{Name: "volume", ValueType: "number", Writable: true, Minimum: &zero, Maximum: &one, StateClass: strategy.StateBearing},
		{Name: "muted", ValueType: "boolean", Writable: true, StateClass: strategy.StateBearing},
		{Name: "application", ValueType: "string", Writable: true, StateClass: strategy.StateBearing},
		{Name: "player_state", ValueType: "string", Writable: false, StateClass: strategy.StateBearing},
		{Name: "input", ValueType: "string", Writable: false, StateClass: strategy.StateBearing},
		// Cast reports receiver/media state, not the physical display power
		// state. Keep the property in the declared contract and expose it as
		// unavailable rather than inferring power from a live socket.
		{Name: "power", ValueType: "boolean", Writable: false, StateClass: strategy.StateBearing},
	}
}

func (s *Strategy) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	devices, err := s.discoveredDevices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]strategy.Device, 0, len(devices))
	for _, d := range devices {
		key := d.IdentityKey
		if key == "" {
			key = d.ID
		}
		out = append(out, strategy.Device{ID: "google-cast:" + key, Serial: key, IdentityKey: d.IdentityKey, IdentityKind: d.IdentityKind, Name: d.Name, Model: d.Model, Endpoint: d.Endpoint, StrategyID: s.ID(), Transport: "cast", Health: strategy.StatusAvailable, ObservedAt: time.Now().UTC()})
	}
	return out, nil
}

func (s *Strategy) ForDevice(serial string) strategy.Strategy {
	clone := *s
	clone.serial = strings.TrimSpace(serial)
	return &clone
}

// ForEndpoint binds a composed identity to the exact Cast service profile.
// This matters when a canonical device identity was first contributed by a
// different transport with a different serial-like claim.
func (s *Strategy) ForEndpoint(endpoint string) strategy.Strategy {
	clone := *s
	clone.endpoint = strings.TrimSpace(endpoint)
	return &clone
}

func (s *Strategy) DiscoverCast(ctx context.Context) ([]Device, error) {
	return s.discoveredDevices(ctx)
}

func (s *Strategy) discoveredDevices(ctx context.Context) ([]Device, error) {
	if s.discover != nil {
		return s.discover.DiscoverCast(ctx)
	}
	if len(s.devices) > 0 {
		return append([]Device(nil), s.devices...), nil
	}
	records, err := mdns.Browse(ctx, []string{"_googlecast._tcp"}, mdns.Options{})
	if err != nil {
		return nil, fmt.Errorf("discover Google Cast services: %w", err)
	}
	out := make([]Device, 0, len(records))
	for _, record := range records {
		if record.Port == 0 || len(record.Addrs) == 0 {
			continue
		}
		key := strings.TrimSpace(record.TXT["id"])
		for _, address := range record.Addrs {
			port := record.Port
			if port == 0 {
				port = defaultPort
			}
			out = append(out, Device{ID: key, IdentityKey: key, IdentityKind: "cast-id", Name: first(record.TXT["fn"], record.Instance), Model: record.TXT["md"], Endpoint: net.JoinHostPort(address.String(), strconv.Itoa(port))})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("DNS-SD browse returned no reachable _googlecast._tcp instances")
	}
	return out, nil
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "Google Cast receiver"
}

func (s *Strategy) boundClient(ctx context.Context) (Client, error) {
	if s.client != nil {
		return s.client, nil
	}
	devices, err := s.discoveredDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if (s.endpoint != "" && d.Endpoint == s.endpoint) || (s.endpoint == "" && (s.serial == "" || d.IdentityKey == s.serial || d.ID == s.serial)) {
			client, err := newWireClient(d.Endpoint)
			if err != nil {
				return nil, err
			}
			s.client = client
			return client, nil
		}
	}
	return nil, fmt.Errorf("Google Cast target %q was not discovered", s.serial)
}

func (s *Strategy) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	client, err := s.boundClient(ctx)
	if err != nil {
		return strategy.DeviceState{}, err
	}
	status, err := client.Status(ctx)
	if err != nil {
		return strategy.DeviceState{}, err
	}
	state := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{}, Unavailable: map[string]string{}}
	values := map[string]any{"volume": status.Volume, "muted": status.Muted, "application": status.Application, "player_state": status.PlayerState}
	unavailable := map[string]string{
		"input": "Cast receiver status does not report an input source",
		"power": "Cast receiver status does not expose physical display power state",
	}
	for _, descriptor := range propertyDescriptors() {
		if reason, unavailable := unavailable[descriptor.Name]; unavailable {
			state.Unavailable[descriptor.Name] = reason
			continue
		}
		value := values[descriptor.Name]
		if err := strategy.ValidateObservedPropertyValue(descriptor, value); err != nil {
			state.Unavailable[descriptor.Name] = err.Error()
			continue
		}
		state.Properties[descriptor.Name] = strategy.PropertyValue{Value: value, Status: strategy.StatusAvailable, Transport: s.ID()}
	}
	if len(state.Unavailable) == 0 {
		state.Unavailable = nil
	}
	return state, nil
}

func (s *Strategy) ReadSensors(ctx context.Context) ([]strategy.SensorReading, error) {
	state, err := s.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	observedAt := time.Now().UTC()
	readings := make([]strategy.SensorReading, 0, len(state.Properties))
	for name, property := range state.Properties {
		if property.Status != strategy.StatusAvailable {
			continue
		}
		readings = append(readings, strategy.SensorReading{Name: name, Value: property.Value, ObservedAt: observedAt, StateClass: strategy.StateBearing})
	}
	return readings, nil
}

func (s *Strategy) GetProperty(ctx context.Context, name string) (any, error) {
	state, err := s.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	value, ok := state.Properties[name]
	if !ok || value.Status != strategy.StatusAvailable {
		return nil, &strategy.AvailabilityError{Reason: "Cast property " + name + " is unavailable", NextAction: "Probe receiver status again"}
	}
	return value.Value, nil
}

func (s *Strategy) SetProperty(ctx context.Context, set strategy.PropertySet) error {
	client, err := s.boundClient(ctx)
	if err != nil {
		return err
	}
	switch set.Name {
	case "volume":
		value, ok := asFloat(set.Value)
		if !ok || value < 0 || value > 1 {
			return fmt.Errorf("volume must be between 0 and 1")
		}
		return client.SetVolume(ctx, value)
	case "muted":
		value, ok := set.Value.(bool)
		if !ok {
			return fmt.Errorf("muted must be boolean")
		}
		return client.SetMuted(ctx, value)
	case "application":
		value, ok := set.Value.(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("application must be a non-empty Cast application id")
		}
		return client.Launch(ctx, value)
	default:
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "Cast property " + set.Name + " is read-only or unknown"}
	}
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func (s *Strategy) ControlMedia(ctx context.Context, command strategy.MediaCommand) error {
	client, err := s.boundClient(ctx)
	if err != nil {
		return err
	}
	action := strings.ToLower(strings.TrimSpace(command.Action))
	if action == "launch" {
		id, ok := command.Value.(string)
		if !ok || id == "" {
			return fmt.Errorf("Cast launch requires an application id")
		}
		return client.Launch(ctx, id)
	}
	return client.Media(ctx, action)
}

func (s *Strategy) ObserveState(ctx context.Context, sink strategy.StateChangeSink) error {
	client, err := s.boundClient(ctx)
	if err != nil {
		return err
	}
	return client.Observe(ctx, func(event strategy.StateChangeEvent) {
		event.Transport = s.ID()
		event.StateClass = strategy.StateBearing
		sink.Publish(event)
	})
}

func (s *Strategy) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
