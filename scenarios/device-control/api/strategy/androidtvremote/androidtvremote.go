// Package androidtvremote provides the Google TV / Android TV Remote
// transport. The protocol client is injected so discovery, pairing, and
// command handling can be tested from recorded fixtures without a television.
package androidtvremote

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"device-control/strategy"
)

type Device struct {
	Serial   string
	Name     string
	Model    string
	Endpoint string
}

type Client interface {
	Key(context.Context, string) error
	Media(context.Context, strategy.MediaCommand) error
}

// PairingClient owns the Android TV Remote protocol exchange. Keeping it
// optional makes the strategy deterministic in fixture tests while allowing a
// production transport to perform the PIN/certificate handshake.
type PairingClient interface {
	Pair(context.Context, Device, string) ([]byte, error)
}

type TextClient interface {
	Text(context.Context, string) error
}

type CertificateStore interface {
	SavePairingCertificate(context.Context, string, []byte) error
}

type CertificateLoader interface {
	LoadPairingCertificate(context.Context, string) ([]byte, error)
}

type Discovery interface {
	DiscoverMDNS(context.Context) ([]Device, error)
}

type Strategy struct {
	devices  []Device
	serial   string
	client   Client
	pairing  PairingClient
	pairings CertificateStore
	discover Discovery
}

type Option func(*Strategy)

func WithDevices(devices ...Device) Option {
	return func(s *Strategy) { s.devices = append([]Device(nil), devices...) }
}
func WithClient(client Client) Option { return func(s *Strategy) { s.client = client } }
func WithPairingClient(client PairingClient) Option {
	return func(s *Strategy) { s.pairing = client }
}

func WithPairingStore(store CertificateStore) Option {
	return func(s *Strategy) { s.pairings = store }
}

func WithDiscovery(discovery Discovery) Option {
	return func(s *Strategy) { s.discover = discovery }
}

func New(options ...Option) *Strategy {
	s := &Strategy{}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Strategy) ID() string { return "android-tv-remote" }

// SetCertificateStore connects the strategy to the credential authority owned
// by device-control. The strategy stores a certificate bundle as opaque bytes;
// it never persists private material in its own database or declarations.
func (s *Strategy) SetCertificateStore(store CertificateStore) { s.pairings = store }

func (s *Strategy) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{
		StrategyID: s.ID(), Description: "Google TV / Android TV Remote transport", Status: strategy.StatusAvailable,
		SupportedHostOS: []string{"linux", "macos", "windows"}, Promotable: true, EvidenceClass: "transport-fixture",
		Capabilities: map[string]strategy.Capability{
			strategy.CapInput:        {Name: strategy.CapInput, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing, ProbeEvidence: "Android TV Remote key transport"},
			strategy.CapMedia:        {Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing, ProbeEvidence: "Android TV Remote media transport"},
			strategy.CapScreenshot:   {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "Android TV Remote does not expose a screen capture modality"},
			strategy.CapSemanticTree: {Name: strategy.CapSemanticTree, Status: strategy.StatusUnavailable, Reason: "Android TV Remote does not expose a view hierarchy"},
		},
	}, nil
}

func (s *Strategy) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	devices, err := s.discoveredDevices(ctx)
	if err != nil {
		return nil, err
	}
	return toDevices(devices), nil
}

func (s *Strategy) DiscoverMDNS(ctx context.Context) ([]Device, error) {
	return s.discoveredDevices(ctx)
}

func (s *Strategy) discoveredDevices(ctx context.Context) ([]Device, error) {
	if s.discover != nil {
		return s.discover.DiscoverMDNS(ctx)
	}
	if len(s.devices) > 0 {
		return append([]Device(nil), s.devices...), nil
	}
	return discoverMDNS(ctx)
}

func (s *Strategy) ForDevice(serial string) strategy.Strategy {
	clone := *s
	clone.serial = strings.TrimSpace(serial)
	return &clone
}

func (s *Strategy) Actuate(ctx context.Context, event strategy.Actuation) error {
	client, err := s.ensureClient(ctx)
	if err != nil {
		return err
	}
	if event.Text != "" {
		textClient, ok := client.(TextClient)
		if !ok {
			return &strategy.UnsupportedCapabilityError{Capability: strategy.CapInput, Operation: "android-tv remote text input is unavailable"}
		}
		return textClient.Text(ctx, event.Text)
	}
	if event.Key == nil {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapInput, Operation: "android-tv remote accepts key events only"}
	}
	return client.Key(ctx, event.Key.Key)
}

func (s *Strategy) ControlMedia(ctx context.Context, command strategy.MediaCommand) error {
	client, err := s.ensureClient(ctx)
	if err != nil {
		return err
	}
	return client.Media(ctx, command)
}

func (s *Strategy) ensureClient(ctx context.Context) (Client, error) {
	if s.client != nil {
		return s.client, nil
	}
	if s.pairings == nil {
		return nil, &strategy.AvailabilityError{Reason: "Android TV Remote is not paired", NextAction: "Pair the Google TV transport and retain its certificate."}
	}
	loader, ok := s.pairings.(CertificateLoader)
	if !ok {
		return nil, &strategy.AvailabilityError{Reason: "Android TV Remote certificate loader is unavailable", NextAction: "Configure the device-control credential authority for Android TV Remote certificates."}
	}
	target, err := s.targetDevice(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := loader.LoadPairingCertificate(ctx, target.Serial)
	if err != nil {
		return nil, &strategy.AvailabilityError{Reason: "Android TV Remote is not paired", NextAction: "Pair the Google TV transport and retain its certificate."}
	}
	bundle, err := decodeCertificateBundle(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode Android TV Remote certificate: %w", err)
	}
	client, err := newWireClient(target.Endpoint, bundle)
	if err != nil {
		return nil, fmt.Errorf("prepare Android TV Remote transport: %w", err)
	}
	s.client = client
	return client, nil
}

func (s *Strategy) targetDevice(ctx context.Context) (Device, error) {
	devices, err := s.discoveredDevices(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("discover Android TV Remote target: %w", err)
	}
	for _, device := range devices {
		if s.serial == "" || strings.TrimSpace(device.Serial) == s.serial {
			if strings.TrimSpace(device.Endpoint) == "" {
				continue
			}
			return device, nil
		}
	}
	return Device{}, fmt.Errorf("android-tv remote target %q was not discovered", s.serial)
}

// Pair performs the PIN/certificate exchange through the injected protocol
// client and persists only the resulting certificate through the credential
// store. Private material never enters declarations or audit records.
func (s *Strategy) Pair(ctx context.Context, serial, pin string) error {
	if s.pairing == nil {
		s.pairing = wirePairingClient{}
	}
	target, err := s.targetForSerial(ctx, serial)
	if err != nil {
		return err
	}
	certificate, err := s.pairing.Pair(ctx, target, pin)
	if err != nil {
		return fmt.Errorf("android-tv remote pairing exchange: %w", err)
	}
	if err := s.PairCertificate(ctx, serial, certificate); err != nil {
		return err
	}
	if bundle, decodeErr := decodeCertificateBundle(certificate); decodeErr == nil {
		if client, clientErr := newWireClient(target.Endpoint, bundle); clientErr == nil {
			s.client = client
		}
	}
	return nil
}

func (s *Strategy) targetForSerial(ctx context.Context, serial string) (Device, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return Device{}, fmt.Errorf("android-tv remote pairing requires a serial")
	}
	devices, err := s.discoveredDevices(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("discover Android TV Remote pairing target: %w", err)
	}
	for _, device := range devices {
		if strings.TrimSpace(device.Serial) == serial && strings.TrimSpace(device.Endpoint) != "" {
			return device, nil
		}
	}
	return Device{}, fmt.Errorf("android-tv remote pairing target %q was not discovered", serial)
}

// PairCertificate is the persistence seam for an already-completed protocol
// exchange, useful when a credential authority performs the handshake.
func (s *Strategy) PairCertificate(ctx context.Context, serial string, certificate []byte) error {
	if s.pairings == nil {
		return fmt.Errorf("android-tv remote pairing store is unavailable")
	}
	if strings.TrimSpace(serial) == "" || len(certificate) == 0 {
		return fmt.Errorf("android-tv remote pairing requires serial and certificate")
	}
	return s.pairings.SavePairingCertificate(ctx, serial, append([]byte(nil), certificate...))
}

type certificateBundle struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

func encodeCertificateBundle(bundle certificateBundle) ([]byte, error) {
	return json.Marshal(bundle)
}

func decodeCertificateBundle(encoded []byte) (certificateBundle, error) {
	var bundle certificateBundle
	if err := json.Unmarshal(encoded, &bundle); err != nil {
		return certificateBundle{}, err
	}
	if strings.TrimSpace(bundle.Certificate) == "" || strings.TrimSpace(bundle.PrivateKey) == "" {
		return certificateBundle{}, fmt.Errorf("certificate bundle is missing certificate or private key")
	}
	return bundle, nil
}

func discoverMDNS(ctx context.Context) ([]Device, error) {
	// Android TV Remote advertises both service names across protocol versions.
	// net.Resolver delegates .local resolution to the host's mDNS resolver when
	// one is configured, which keeps this strategy free of OS-specific tools.
	var lastErr error
	for _, service := range []string{"androidtvremote2", "androidtvremote"} {
		_, records, err := net.DefaultResolver.LookupSRV(ctx, service, "tcp", "local")
		if err != nil {
			lastErr = err
			continue
		}
		out := make([]Device, 0, len(records))
		seen := map[string]bool{}
		for _, record := range records {
			host := strings.TrimSuffix(strings.TrimSpace(record.Target), ".")
			if host == "" || record.Port == 0 {
				continue
			}
			addresses, lookupErr := net.DefaultResolver.LookupHost(ctx, host)
			if lookupErr != nil {
				addresses = []string{host}
			}
			for _, address := range addresses {
				endpoint := net.JoinHostPort(address, fmt.Sprint(record.Port))
				if seen[endpoint] {
					continue
				}
				seen[endpoint] = true
				out = append(out, Device{Serial: host, Name: host, Endpoint: endpoint})
			}
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no Android TV Remote mDNS service was found")
	}
	return nil, fmt.Errorf("discover Android TV Remote mDNS services: %w", lastErr)
}

func toDevices(devices []Device) []strategy.Device {
	out := make([]strategy.Device, 0, len(devices))
	for _, device := range devices {
		serial := strings.TrimSpace(device.Serial)
		if serial == "" {
			continue
		}
		out = append(out, strategy.Device{ID: "android-tv:" + serial, Serial: serial, IdentityKey: serial, Name: device.Name, Model: device.Model, Endpoint: device.Endpoint, StrategyID: "android-tv-remote", Transport: "mdns", Health: strategy.StatusAvailable, ObservedAt: time.Now().UTC()})
	}
	return out
}
