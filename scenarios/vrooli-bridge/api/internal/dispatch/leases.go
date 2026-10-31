package dispatch

import (
	"fmt"
	"sync"
	"time"
)

type deviceLease struct {
	Token     string
	Actor     string
	ExpiresAt time.Time
}

// MemoryDeviceLeaseStore is intentionally small: bridge owns the authorization
// record, while device-control owns the lease's lifecycle and policy. Expired
// records are removed on access so a stale token cannot authorize a dispatch.
type MemoryDeviceLeaseStore struct {
	mu     sync.Mutex
	leases map[string]deviceLease
}

func NewMemoryDeviceLeaseStore() *MemoryDeviceLeaseStore {
	return &MemoryDeviceLeaseStore{leases: make(map[string]deviceLease)}
}

func (s *MemoryDeviceLeaseStore) Hold(deviceID, token, actor string, expiresAt time.Time) error {
	if deviceID == "" || token == "" {
		return fmt.Errorf("device lease requires device_id and token")
	}
	if !expiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("device lease must expire in the future")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, lease := range s.leases {
		if id == deviceID && lease.ExpiresAt.After(time.Now().UTC()) && lease.Token != token {
			return fmt.Errorf("device %q already has a held bridge lease", deviceID)
		}
	}
	s.leases[deviceID] = deviceLease{Token: token, Actor: actor, ExpiresAt: expiresAt}
	return nil
}

func (s *MemoryDeviceLeaseStore) Release(deviceID, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lease, ok := s.leases[deviceID]
	if ok && (token == "" || lease.Token == token) {
		delete(s.leases, deviceID)
	}
}

func (s *MemoryDeviceLeaseStore) Held(deviceID, token string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	lease, ok := s.leases[deviceID]
	if !ok {
		return false
	}
	if !lease.ExpiresAt.After(now) {
		delete(s.leases, deviceID)
		return false
	}
	return token != "" && lease.Token == token
}

var _ DeviceLeaseStore = (*MemoryDeviceLeaseStore)(nil)
