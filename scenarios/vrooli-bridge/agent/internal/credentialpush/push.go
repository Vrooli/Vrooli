// Package credentialpush applies the node-side policy for a signed credential
// push. It knows grant metadata and decryption, but it never logs or persists
// a plaintext value itself.
package credentialpush

import (
	"crypto/ecdh"
	"errors"
	"fmt"
	"strings"
	"sync"

	"vrooli-bridge/agent/internal/credentialgrant"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

type Sink interface {
	Put(logicalID, field, value string) error
	Delete(logicalID, field string) error
}

type Result struct {
	Receipt      *channelv1.CredentialReceipt
	Ephemeral    []byte
	Grant        credentialgrant.Grant
	Rejected     bool
	RejectReason string
}

// EphemeralStore holds decrypted values only in process memory until the one
// typed job that names them consumes them. It has no persistence or List API.
type EphemeralStore struct {
	mu     sync.Mutex
	values map[string][]byte
}

func NewEphemeralStore() *EphemeralStore { return &EphemeralStore{values: make(map[string][]byte)} }

func (s *EphemeralStore) Put(logicalID, field string, value []byte) error {
	if s == nil || len(value) == 0 {
		return errors.New("ephemeral credential value is empty")
	}
	copyValue := append([]byte(nil), value...)
	s.mu.Lock()
	if previous := s.values[ephemeralKey(logicalID, field)]; len(previous) > 0 {
		zero(previous)
	}
	s.values[ephemeralKey(logicalID, field)] = copyValue
	s.mu.Unlock()
	return nil
}

func (s *EphemeralStore) Take(logicalID, field string) ([]byte, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	value, ok := s.values[ephemeralKey(logicalID, field)]
	delete(s.values, ephemeralKey(logicalID, field))
	s.mu.Unlock()
	return value, ok
}

func (s *EphemeralStore) Clear(logicalID, field string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if value := s.values[ephemeralKey(logicalID, field)]; len(value) > 0 {
		zero(value)
	}
	delete(s.values, ephemeralKey(logicalID, field))
	s.mu.Unlock()
}

func ephemeralKey(logicalID, field string) string { return logicalID + "\x00" + field }

// Apply verifies local consent, decrypts the value using the independent node
// encryption key, and either sends it to the caller's authority sink or returns
// an ephemeral buffer that the caller must zero after its job. The sealed bytes
// and plaintext are never included in the receipt.
func Apply(push *channelv1.CredentialPush, nodeID string, private *ecdh.PrivateKey, grants credentialgrant.Store, sink Sink) (Result, error) {
	if push == nil {
		return Result{}, errors.New("credential push: nil push")
	}
	receipt := &channelv1.CredentialReceipt{GrantId: push.GetGrantId(), NodeId: nodeID, LogicalId: push.GetLogicalId(), Field: push.GetField(), Generation: push.GetGeneration()}
	reject := func(reason string) (Result, error) {
		receipt.Accepted = false
		receipt.Reason = reason
		return Result{Receipt: receipt, Rejected: true, RejectReason: reason}, nil
	}
	if strings.TrimSpace(push.GetNodeId()) != nodeID {
		return reject("node identity mismatch")
	}
	if grants == nil {
		return reject("no local grant")
	}
	grant, ok := grants.Lookup(push.GetLogicalId(), push.GetField())
	if !ok {
		return reject("address is not granted to this node")
	}
	if push.GetGeneration() < grant.Generation {
		return reject("credential generation is older than the local grant")
	}
	if push.GetRetention() != grant.Retention {
		return reject("credential retention does not match the local grant")
	}
	if private == nil {
		return reject("node encryption key is unavailable")
	}
	plaintext, err := sealing.Open(private, push.GetSealedValue(), push.GetAad())
	if err != nil {
		return Result{}, fmt.Errorf("credential push: decrypt: %w", err)
	}
	if grant.Retention == credentialgrant.RetentionEphemeral {
		receipt.Accepted = true
		return Result{Receipt: receipt, Ephemeral: plaintext, Grant: grant}, nil
	}
	if sink == nil {
		zero(plaintext)
		return Result{}, errors.New("credential push: durable authority sink is unavailable")
	}
	if err := sink.Put(push.GetLogicalId(), push.GetField(), string(plaintext)); err != nil {
		zero(plaintext)
		return Result{}, fmt.Errorf("credential push: store durable value: %w", err)
	}
	zero(plaintext)
	receipt.Accepted = true
	return Result{Receipt: receipt, Grant: grant}, nil
}

func Zero(value []byte) { zero(value) }

func zero(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
