package control

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/googlecast"

	"github.com/google/uuid"
)

type DiscoveredService struct {
	StrategyID       string            `json:"strategy_id"`
	Transport        string            `json:"transport"`
	Service          string            `json:"service"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Model            string            `json:"model,omitempty"`
	Endpoint         string            `json:"endpoint"`
	Address          string            `json:"address,omitempty"`
	Port             int               `json:"port,omitempty"`
	TXT              map[string]string `json:"txt,omitempty"`
	IdentityKey      string            `json:"identity_key,omitempty"`
	Paired           bool              `json:"paired"`
	PairingAvailable bool              `json:"pairing_available"`
}

func (s *Service) DiscoverLAN(ctx context.Context) ([]DiscoveredService, error) {
	services := []DiscoveredService{}
	var failures []string
	for _, id := range []string{"android-tv-remote", "google-cast"} {
		item, ok := s.registry.Get(id)
		if !ok {
			continue
		}
		declaration, _ := item.Describe(ctx)
		pairingAvailable := declaration.Capabilities[strategy.CapPairing].Status == strategy.StatusAvailable
		switch id {
		case "android-tv-remote":
			discoverer, ok := item.(interface {
				DiscoverMDNS(context.Context) ([]androidtvremote.Device, error)
			})
			if ok {
				devices, err := discoverer.DiscoverMDNS(ctx)
				if err != nil {
					failures = append(failures, id+": "+err.Error())
					continue
				}
				for _, device := range devices {
					paired := false
					if checker, checkerOK := item.(interface {
						IsPaired(context.Context, string) bool
					}); checkerOK {
						paired = checker.IsPaired(ctx, device.Serial)
					}
					identity := strings.TrimSpace(device.IdentityKey)
					if identity == "" {
						identity = strings.TrimSpace(device.Serial)
					}
					services = append(services, DiscoveredService{StrategyID: id, Transport: "android-tv-remote", Service: device.Service, ID: "android-tv:" + identity, Name: device.Name, Model: device.Model, Endpoint: device.Endpoint, Address: device.Address, Port: device.Port, TXT: device.TXT, IdentityKey: device.IdentityKey, Paired: paired, PairingAvailable: pairingAvailable})
				}
			}
		case "google-cast":
			discoverer, ok := item.(interface {
				DiscoverCast(context.Context) ([]googlecast.Device, error)
			})
			if ok {
				devices, err := discoverer.DiscoverCast(ctx)
				if err != nil {
					failures = append(failures, id+": "+err.Error())
					continue
				}
				for _, device := range devices {
					services = append(services, DiscoveredService{StrategyID: id, Transport: "google-cast", Service: device.Service, ID: device.ID, Name: device.Name, Model: device.Model, Endpoint: device.Endpoint, Address: device.Address, Port: device.Port, TXT: device.TXT, IdentityKey: device.IdentityKey, PairingAvailable: pairingAvailable})
				}
			}
		}
	}
	if len(failures) > 0 {
		return services, fmt.Errorf("LAN discovery degraded: %s", strings.Join(failures, "; "))
	}
	return services, nil
}

var pairingPIN = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

const pairingSessionTTL = 2 * time.Minute

func (s *Service) PairDevice(ctx context.Context, deviceID, pin string) error {
	secret := []byte(strings.TrimSpace(pin))
	defer zeroSecret(secret)
	_, err := s.PairDeviceSecret(ctx, deviceID, secret)
	return err
}

func (s *Service) PairDeviceSecret(ctx context.Context, deviceID string, secret []byte) (strategy.PairResult, error) {
	defer zeroSecret(secret)
	if !pairingPIN.Match(secret) {
		result := strategy.PairResult{Outcome: "invalid", Transport: "android-tv-remote", Detail: "pairing code must contain exactly six hexadecimal characters"}
		s.recordPairAudit(ctx, deviceID, result.Transport, result.Outcome)
		return result, fmt.Errorf("pairing code must contain exactly six hexadecimal characters")
	}
	adapter, err := s.preparePairingAdapter(ctx, deviceID)
	if err != nil {
		return strategy.PairResult{Outcome: "failed", Transport: "android-tv-remote", Detail: "pairing target was not discovered"}, err
	}
	pairer, ok := adapter.(strategy.Pairer)
	if !ok {
		return strategy.PairResult{Outcome: "unsupported", Transport: "android-tv-remote", Detail: "pairing capability is unavailable"}, fmt.Errorf("android-tv-remote pairing is unavailable")
	}
	result, err := pairer.Pair(ctx, strategy.PairRequest{Secret: secret})
	s.recordPairAudit(ctx, deviceID, result.Transport, result.Outcome)
	if err != nil {
		detail := strings.TrimSpace(result.Detail)
		if detail == "" {
			detail = "pairing exchange failed"
		}
		return result, fmt.Errorf("pairing failed: %s", detail)
	}
	return result, err
}

func (s *Service) preparePairingAdapter(ctx context.Context, deviceID string) (strategy.Strategy, error) {
	record, ok := s.devices.Get(deviceID)
	if !ok {
		var discoverErr error
		record, discoverErr = s.materializePairingDiscovery(ctx, deviceID)
		if discoverErr != nil {
			return nil, discoverErr
		}
	}
	adapter, ok := s.registry.Get("android-tv-remote")
	if !ok {
		return nil, fmt.Errorf("android-tv-remote strategy is unavailable")
	}
	remoteDeviceID := strings.TrimSpace(record.Serial)
	if discoverer, discoverOK := adapter.(interface {
		DiscoverMDNS(context.Context) ([]androidtvremote.Device, error)
	}); discoverOK {
		devices, discoverErr := discoverer.DiscoverMDNS(ctx)
		if discoverErr != nil {
			return nil, fmt.Errorf("discover Android TV Remote pairing target: %w", discoverErr)
		}
		for _, device := range devices {
			if (record.IdentityKey != "" && device.IdentityKey == record.IdentityKey) ||
				(device.Serial != "" && device.Serial == record.Serial) ||
				(device.Endpoint != "" && device.Endpoint == record.Endpoint) {
				remoteDeviceID = device.Serial
				break
			}
		}
	}
	if remoteDeviceID == "" {
		return nil, fmt.Errorf("device %q has no Android TV Remote identity", deviceID)
	}
	if scoped, scopedOK := adapter.(strategy.DeviceScoped); scopedOK {
		adapter = scoped.ForDevice(remoteDeviceID)
	}
	return adapter, nil
}

// BeginPairDevice opens the Android TV Remote handshake through the
// configuration acknowledgement. The television displays its six-character
// hexadecimal pairing code before this method returns; the returned session is
// short-lived and memory-only.
func (s *Service) BeginPairDevice(ctx context.Context, deviceID string) (string, error) {
	adapter, err := s.preparePairingAdapter(ctx, deviceID)
	if err != nil {
		return "", err
	}
	pairer, ok := adapter.(interactivePairer)
	if !ok {
		return "", fmt.Errorf("interactive Android TV Remote pairing is unavailable")
	}
	session, err := pairer.BeginPairing(ctx)
	if err != nil {
		return "", fmt.Errorf("begin Android TV Remote pairing: %w", err)
	}
	pairingID := uuid.NewString()
	now := time.Now().UTC()
	s.mu.Lock()
	if s.pendingPairings == nil {
		s.pendingPairings = map[string]pendingPairing{}
	}
	type closeJob struct {
		id      string
		pending pendingPairing
	}
	var replaced []closeJob
	for id, pending := range s.pendingPairings {
		if pending.deviceID == deviceID && !pending.closing && !pending.uncertain {
			pending.closing = true
			s.pendingPairings[id] = pending
			replaced = append(replaced, closeJob{id: id, pending: pending})
		}
	}
	s.pendingPairings[pairingID] = pendingPairing{deviceID: deviceID, pairer: pairer, session: session, expiresAt: now.Add(pairingSessionTTL)}
	s.mu.Unlock()
	for _, old := range replaced {
		closeErr := old.pending.session.Close()
		s.mu.Lock()
		if current, stillPending := s.pendingPairings[old.id]; stillPending && current.closing {
			if closeErr == nil {
				delete(s.pendingPairings, old.id)
			} else {
				current.closing = false
				current.uncertain = true
				s.pendingPairings[old.id] = current
			}
		}
		s.mu.Unlock()
	}
	time.AfterFunc(pairingSessionTTL, func() { s.expirePairing(pairingID) })
	return pairingID, nil
}

func (s *Service) expirePairing(pairingID string) {
	s.mu.Lock()
	pending, ok := s.pendingPairings[pairingID]
	if !ok || time.Now().UTC().Before(pending.expiresAt) || pending.closing || pending.uncertain {
		s.mu.Unlock()
		return
	}
	pending.closing = true
	pending.uncertain = false
	s.pendingPairings[pairingID] = pending
	s.mu.Unlock()

	closeErr := pending.session.Close()
	s.mu.Lock()
	if current, stillPending := s.pendingPairings[pairingID]; stillPending && current.closing {
		if closeErr == nil {
			delete(s.pendingPairings, pairingID)
		} else {
			current.closing = false
			current.uncertain = true
			s.pendingPairings[pairingID] = current
		}
	}
	s.mu.Unlock()
}

// CompletePairDevice submits the owner-provided pairing code to a previously started
// handshake. Invalid input leaves the session available for one retry.
func (s *Service) CompletePairDevice(ctx context.Context, deviceID, pairingID string, secret []byte) (strategy.PairResult, error) {
	defer zeroSecret(secret)
	s.mu.Lock()
	pending, ok := s.pendingPairings[pairingID]
	if ok && !time.Now().UTC().Before(pending.expiresAt) && !pending.closing {
		s.mu.Unlock()
		if !pending.uncertain {
			s.expirePairing(pairingID)
		}
		return strategy.PairResult{Outcome: "failed", Transport: "android-tv-remote", Detail: "pairing session is unavailable or expired"}, fmt.Errorf("pairing session is unavailable or expired")
	}
	closing := pending.closing
	s.mu.Unlock()
	if !ok || closing || pending.deviceID != deviceID {
		return strategy.PairResult{Outcome: "failed", Transport: "android-tv-remote", Detail: "pairing session is unavailable"}, fmt.Errorf("pairing session is unavailable or expired")
	}
	if !pairingPIN.Match(secret) {
		result := strategy.PairResult{Outcome: "invalid", Transport: "android-tv-remote", Detail: "pairing code must contain exactly six hexadecimal characters"}
		s.recordPairAudit(ctx, pending.deviceID, result.Transport, result.Outcome)
		return result, fmt.Errorf("pairing code must contain exactly six hexadecimal characters")
	}
	s.mu.Lock()
	current, stillPending := s.pendingPairings[pairingID]
	if !stillPending || current.closing || current.deviceID != deviceID || !time.Now().UTC().Before(current.expiresAt) {
		s.mu.Unlock()
		s.expirePairing(pairingID)
		return strategy.PairResult{Outcome: "failed", Transport: "android-tv-remote", Detail: "pairing session is unavailable"}, fmt.Errorf("pairing session is unavailable or expired")
	}
	current.closing = true
	s.pendingPairings[pairingID] = current
	s.mu.Unlock()
	result, err := pending.pairer.CompletePairing(ctx, pending.session, secret)
	s.mu.Lock()
	if current, stillPending := s.pendingPairings[pairingID]; stillPending && current.closing {
		if err == nil {
			delete(s.pendingPairings, pairingID)
		} else {
			current.closing = false
			current.uncertain = true
			s.pendingPairings[pairingID] = current
		}
	}
	s.mu.Unlock()
	s.recordPairAudit(ctx, pending.deviceID, result.Transport, result.Outcome)
	if err != nil {
		detail := strings.TrimSpace(result.Detail)
		if detail == "" {
			detail = "pairing exchange failed"
		}
		return result, fmt.Errorf("pairing failed: %s", detail)
	}
	return result, nil
}

func (s *Service) materializePairingDiscovery(ctx context.Context, deviceID string) (devicedomain.Record, error) {
	services, discoverErr := s.DiscoverLAN(ctx)
	for _, service := range services {
		if service.StrategyID != "android-tv-remote" || service.ID != deviceID {
			continue
		}
		serial := strings.TrimSpace(service.IdentityKey)
		if serial == "" {
			serial = strings.TrimPrefix(strings.TrimSpace(service.ID), "android-tv:")
		}
		if serial == "" || strings.TrimSpace(service.Endpoint) == "" {
			continue
		}
		unlockInventory, lockErr := s.lockInventory(ctx)
		if lockErr != nil {
			return devicedomain.Record{}, lockErr
		}
		defer unlockInventory()
		return s.devices.UpsertIdentity(devicedomain.Record{
			ID:           service.ID,
			IdentityKey:  service.IdentityKey,
			IdentityKind: "bluetooth-mac",
			Name:         service.Name,
			Kind:         "physical",
			Serial:       serial,
			Model:        service.Model,
			Endpoint:     service.Endpoint,
			StrategyID:   "android-tv-remote",
			Status:       strategy.StatusAvailable,
			Health:       strategy.StatusAvailable,
			Transport:    "android-tv-remote",
			ObservedAt:   time.Now().UTC(),
		}), nil
	}
	if discoverErr != nil {
		return devicedomain.Record{}, fmt.Errorf("discover Android TV Remote pairing target: %w", discoverErr)
	}
	return devicedomain.Record{}, fmt.Errorf("unknown device %q", deviceID)
}

func (s *Service) recordPairAudit(ctx context.Context, deviceID, transport, outcome string) {
	record := Audit{ID: uuid.NewString(), Actor: "operator", DeviceID: deviceID, Transport: transport, Verb: "pair", Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true, EvidenceBacked: false}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, transport, verb, outcome, created_at, redaction_verified, redaction_opted_out, interactive, evidence_backed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.Transport, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0, 1, 0)
	}
	s.audits = append(s.audits, record)
}

func zeroSecret(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}
