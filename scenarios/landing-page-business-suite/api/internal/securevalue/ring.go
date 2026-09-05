package securevalue

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnknownKeyVersion = errors.New("encryption key version is not present in the key ring")

type VersionedKey struct {
	Version int    `json:"version"`
	Key     string `json:"key"`
}

type Ring struct {
	Active int            `json:"active"`
	Keys   []VersionedKey `json:"keys"`
}

func NewRing() (Ring, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Ring{}, fmt.Errorf("generate encryption ring key: %w", err)
	}
	return Ring{Active: 1, Keys: []VersionedKey{{Version: 1, Key: base64.StdEncoding.EncodeToString(key)}}}, nil
}

func ParseRing(encoded string) (Ring, error) {
	var ring Ring
	if err := json.Unmarshal([]byte(encoded), &ring); err != nil {
		return Ring{}, fmt.Errorf("decode encryption key ring: %w", err)
	}
	if ring.Active <= 0 {
		return Ring{}, errors.New("encryption key ring has no active version")
	}
	if len(ring.Keys) == 0 {
		return Ring{}, errors.New("encryption key ring has no keys")
	}
	seen := make(map[int]struct{}, len(ring.Keys))
	activeFound := false
	for _, versioned := range ring.Keys {
		if versioned.Version <= 0 {
			return Ring{}, errors.New("encryption key ring contains an invalid version")
		}
		if _, ok := seen[versioned.Version]; ok {
			return Ring{}, fmt.Errorf("encryption key ring contains duplicate version %d", versioned.Version)
		}
		seen[versioned.Version] = struct{}{}
		key, err := base64.StdEncoding.DecodeString(versioned.Key)
		if err != nil || len(key) != 32 {
			return Ring{}, fmt.Errorf("encryption key ring version %d must contain a base64-encoded 32-byte key", versioned.Version)
		}
		if versioned.Version == ring.Active {
			activeFound = true
		}
	}
	if !activeFound {
		return Ring{}, fmt.Errorf("encryption key ring active version %d is absent", ring.Active)
	}
	return ring, nil
}

func (r Ring) Marshal() (string, error) {
	if _, err := r.Get(r.Active); err != nil {
		return "", err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r Ring) ActiveKey() ([]byte, error) { return r.Get(r.Active) }

func (r Ring) Get(version int) ([]byte, error) {
	for _, candidate := range r.Keys {
		if candidate.Version != version {
			continue
		}
		key, err := base64.StdEncoding.DecodeString(candidate.Key)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("encryption key ring version %d is invalid", version)
		}
		return key, nil
	}
	return nil, fmt.Errorf("%w: %d", ErrUnknownKeyVersion, version)
}

func (r Ring) Rotate() (Ring, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Ring{}, fmt.Errorf("generate rotated encryption key: %w", err)
	}
	max := r.Active
	for _, candidate := range r.Keys {
		if candidate.Version > max {
			max = candidate.Version
		}
	}
	r.Active = max + 1
	r.Keys = append(r.Keys, VersionedKey{Version: r.Active, Key: base64.StdEncoding.EncodeToString(key)})
	return r, nil
}

func EncryptRing(ring Ring, plaintext string) (string, error) {
	key, err := ring.ActiveKey()
	if err != nil {
		return "", err
	}
	sealed, err := Encrypt(key, plaintext)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d:%s", ring.Active, sealed), nil
}

func DecryptRing(ring Ring, ciphertext string) (string, error) {
	version := 1
	payload := ciphertext
	if len(ciphertext) > 2 && ciphertext[0] == 'v' {
		var parsed int
		if _, err := fmt.Sscanf(ciphertext, "v%d:", &parsed); err != nil || parsed <= 0 {
			return "", fmt.Errorf("invalid encryption ciphertext version")
		}
		version = parsed
		prefix := fmt.Sprintf("v%d:", version)
		payload = ciphertext[len(prefix):]
	}
	key, err := ring.Get(version)
	if err != nil {
		return "", err
	}
	return Decrypt(key, payload)
}
