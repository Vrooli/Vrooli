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
	mdns "github.com/vrooli/mdns-go"
)

type Device struct {
	Serial       string
	Name         string
	Model        string
	Endpoint     string
	IdentityKey  string
	IdentityKind string
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

// PairingSession represents the protocol state after the television has been
// asked to display its pairing code and before the PIN-derived secret is sent.
// The session remains in memory only; it is never serialized or persisted.
type PairingSession interface {
	Complete(context.Context, string) ([]byte, error)
	Close() error
}

// InteractivePairingClient exposes the two-stage pairing exchange needed by
// operator surfaces: begin the handshake first, then submit the displayed PIN.
type InteractivePairingClient interface {
	Begin(context.Context, Device) (PairingSession, error)
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
	endpoint string
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

func (s *Strategy) IsPaired(ctx context.Context, serial string) bool {
	loader, ok := s.pairings.(CertificateLoader)
	if !ok {
		return false
	}
	certificate, err := loader.LoadPairingCertificate(ctx, strings.TrimSpace(serial))
	return err == nil && len(certificate) > 0
}

func (s *Strategy) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{
		StrategyID: s.ID(), Description: "Google TV / Android TV Remote transport", Status: strategy.StatusAvailable,
		SupportedHostOS: []string{"linux", "macos", "windows"}, Promotable: true, EvidenceClass: "transport-fixture",
		Capabilities: map[string]strategy.Capability{
			strategy.CapInput:        {Name: strategy.CapInput, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing, ProbeEvidence: "Android TV Remote key transport"},
			strategy.CapMedia:        {Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing, ProbeEvidence: "Android TV Remote media transport"},
			strategy.CapPairing:      {Name: strategy.CapPairing, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing, ProbeEvidence: "Android TV Remote PIN/certificate exchange"},
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

// ForEndpoint binds a composed identity to the exact Remote service profile,
// avoiding any dependence on which transport supplied the canonical serial.
func (s *Strategy) ForEndpoint(endpoint string) strategy.Strategy {
	clone := *s
	clone.endpoint = strings.TrimSpace(endpoint)
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
		if (s.endpoint != "" && strings.TrimSpace(device.Endpoint) == s.endpoint) || (s.endpoint == "" && (s.serial == "" || strings.TrimSpace(device.Serial) == s.serial)) {
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
func (s *Strategy) Pair(ctx context.Context, request strategy.PairRequest) (strategy.PairResult, error) {
	defer zeroBytes(request.Secret)
	if s.pairing == nil {
		s.pairing = wirePairingClient{}
	}
	if _, interactive := s.pairing.(InteractivePairingClient); interactive {
		pending, err := s.BeginPairing(ctx)
		if err != nil {
			return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing target was not discovered"}, err
		}
		return s.CompletePairing(ctx, pending, request.Secret)
	}
	serial := strings.TrimSpace(s.serial)
	target, err := s.targetForSerial(ctx, serial)
	if err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing target was not discovered"}, err
	}
	certificate, err := s.pairing.Pair(ctx, target, string(request.Secret))
	if err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing exchange failed"}, fmt.Errorf("android-tv remote pairing exchange failed")
	}
	if err := s.PairCertificate(ctx, serial, certificate); err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing certificate could not be stored"}, err
	}
	if bundle, decodeErr := decodeCertificateBundle(certificate); decodeErr == nil {
		if client, clientErr := newWireClient(target.Endpoint, bundle); clientErr == nil {
			s.client = client
		}
	}
	return strategy.PairResult{Outcome: "paired", Transport: s.ID(), Detail: "certificate stored"}, nil
}

type strategyPairingSession struct {
	session PairingSession
	target  Device
}

func (p *strategyPairingSession) Complete(ctx context.Context, pin string) ([]byte, error) {
	return p.session.Complete(ctx, pin)
}

func (p *strategyPairingSession) Close() error { return p.session.Close() }

// BeginPairing opens the protocol through the configuration acknowledgement.
// At that point the television is showing its pairing code and the caller can
// safely ask the owner for the PIN.
func (s *Strategy) BeginPairing(ctx context.Context) (PairingSession, error) {
	if s.pairing == nil {
		s.pairing = wirePairingClient{}
	}
	serial := strings.TrimSpace(s.serial)
	target, err := s.targetForSerial(ctx, serial)
	if err != nil {
		return nil, err
	}
	interactive, ok := s.pairing.(InteractivePairingClient)
	if !ok {
		return nil, fmt.Errorf("interactive Android TV Remote pairing is unavailable")
	}
	session, err := interactive.Begin(ctx, target)
	if err != nil {
		return nil, err
	}
	return &strategyPairingSession{session: session, target: target}, nil
}

// CompletePairing sends the owner-provided PIN, persists the resulting
// certificate, and prepares the paired remote client for reconnects.
func (s *Strategy) CompletePairing(ctx context.Context, session PairingSession, secret []byte) (strategy.PairResult, error) {
	defer zeroBytes(secret)
	pending, ok := session.(*strategyPairingSession)
	if !ok || pending == nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing session is invalid"}, fmt.Errorf("android-tv remote pairing session is invalid")
	}
	defer pending.Close()
	certificate, err := pending.Complete(ctx, string(secret))
	if err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing exchange failed"}, fmt.Errorf("android-tv remote pairing exchange failed")
	}
	serial := strings.TrimSpace(s.serial)
	if err := s.PairCertificate(ctx, serial, certificate); err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: s.ID(), Detail: "pairing certificate could not be stored"}, err
	}
	if bundle, decodeErr := decodeCertificateBundle(certificate); decodeErr == nil {
		if client, clientErr := newWireClient(pending.target.Endpoint, bundle); clientErr == nil {
			s.client = client
		}
	}
	return strategy.PairResult{Outcome: "paired", Transport: s.ID(), Detail: "certificate stored"}, nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

// ReadState is intentionally explicit: Remote provides commands, not a
// readable status channel. The control plane can therefore distinguish an
// unavailable state from a fabricated zero value.
func (s *Strategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Unavailable: map[string]string{"state": "Android TV Remote does not expose readable state"}}, nil
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
	// DNS-SD browse is deliberately performed by the shared multicast browser;
	// DNS-SD enumeration is kept in the shared multicast browser so all
	// transports use the same bounded interface and error semantics.
	var lastErr error
	for _, service := range []string{"_androidtvremote2._tcp", "_androidtvremote._tcp"} {
		records, err := mdns.Browse(ctx, []string{service}, mdns.Options{})
		if err != nil {
			lastErr = err
			continue
		}
		out := make([]Device, 0, len(records))
		seen := map[string]bool{}
		for _, record := range records {
			host := strings.TrimSuffix(strings.TrimSpace(record.Host), ".")
			if host == "" || record.Port == 0 || len(record.Addrs) == 0 {
				continue
			}
			identity := strings.TrimSpace(record.TXT["bt"])
			identityKind := ""
			if identity != "" {
				identityKind = "bluetooth-mac"
			}
			for _, address := range record.Addrs {
				endpoint := formatEndpoint(address, record.Port)
				if seen[endpoint] {
					continue
				}
				seen[endpoint] = true
				serial := identity
				if serial == "" {
					serial = strings.TrimSuffix(record.Instance, ".")
				}
				out = append(out, Device{Serial: serial, IdentityKey: identity, IdentityKind: identityKind, Name: record.TXT["fn"], Model: record.TXT["md"], Endpoint: endpoint})
			}
		}
		if len(out) == 0 {
			lastErr = fmt.Errorf("DNS-SD browse returned no reachable %s instances", service)
			continue
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no Android TV Remote mDNS service was found")
	}
	return nil, fmt.Errorf("discover Android TV Remote mDNS services: %w", lastErr)
}

func formatEndpoint(address net.IP, port int) string {
	return net.JoinHostPort(address.String(), fmt.Sprintf("%d", port))
}

func toDevices(devices []Device) []strategy.Device {
	out := make([]strategy.Device, 0, len(devices))
	for _, device := range devices {
		serial := strings.TrimSpace(device.Serial)
		if serial == "" {
			continue
		}
		identity := device.IdentityKey
		id := identity
		if id == "" {
			id = serial
		}
		out = append(out, strategy.Device{ID: "android-tv:" + id, Serial: serial, IdentityKey: device.IdentityKey, IdentityKind: device.IdentityKind, Name: device.Name, Model: device.Model, Endpoint: device.Endpoint, StrategyID: "android-tv-remote", Transport: "mdns", Health: strategy.StatusAvailable, ObservedAt: time.Now().UTC()})
	}
	return out
}
