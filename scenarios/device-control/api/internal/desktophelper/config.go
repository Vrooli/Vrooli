package desktophelper

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

var ErrBootstrap = errors.New("desktop helper bootstrap refused")

type Config struct {
	Version             int                    `json:"version"`
	Surface             targetmodel.SurfaceRef `json:"surface"`
	SessionID           string                 `json:"session_id"`
	Display             uint16                 `json:"display"`
	XAuthorityFile      string                 `json:"xauthority_file"`
	PublicKey           string                 `json:"public_key"`
	GrantStatusFile     string                 `json:"grant_status_file"`
	StateDirectory      string                 `json:"state_directory"`
	AccessibilitySocket string                 `json:"accessibility_socket,omitempty"`
	AccessibilityBusID  string                 `json:"accessibility_bus_id,omitempty"`
}

func decodeStrict(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(v) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrBootstrap
	}
	return nil
}
func LoadConfig(path string) (Config, error) {
	var config Config
	data, err := readPrivate(path, 64*1024)
	if err != nil {
		return config, err
	}
	if decodeStrict(data, &config) != nil {
		return Config{}, ErrBootstrap
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validateConfig(config Config) error {
	if config.Version != 1 || config.Surface.Validate() != nil || config.Surface.OwnerScenario != "device-control" || config.Surface.Target.OwnerScenario != "vrooli-bridge" || config.Surface.Target.HostNodeID == "" || config.Surface.Target.ResourceID != config.Surface.Target.HostNodeID || config.SessionID == "" {
		return ErrBootstrap
	}
	for _, path := range []string{config.XAuthorityFile, config.GrantStatusFile, config.StateDirectory} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return ErrBootstrap
		}
	}
	if config.XAuthorityFile == config.GrantStatusFile || config.XAuthorityFile == config.StateDirectory || config.GrantStatusFile == config.StateDirectory {
		return ErrBootstrap
	}
	if config.AccessibilitySocket != "" || config.AccessibilityBusID != "" {
		path := config.AccessibilitySocket
		id, err := hex.DecodeString(config.AccessibilityBusID)
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 107 || err != nil || len(id) != 16 || path == config.XAuthorityFile || path == config.GrantStatusFile || path == config.StateDirectory || path == filepath.Join(config.StateDirectory, "helper.sock") || path == filepath.Join(config.StateDirectory, "sessions.sqlite") {
			return ErrBootstrap
		}
	}
	key, err := base64.StdEncoding.DecodeString(config.PublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return ErrBootstrap
	}
	return nil
}

// Authority records are binary length-prefixed fields. Never use a display
// substring or an arbitrary first record: multiple sessions may share a file.
func xCookie(data []byte, display uint16, hostname string) (string, error) {
	reader := bytes.NewReader(data)
	var matches [][]byte
	field := func() ([]byte, error) {
		var n uint16
		if err := binary.Read(reader, binary.BigEndian, &n); err != nil {
			return nil, err
		}
		b := make([]byte, n)
		_, err := io.ReadFull(reader, b)
		return b, err
	}
	for reader.Len() > 0 {
		var family uint16
		if binary.Read(reader, binary.BigEndian, &family) != nil {
			return "", ErrBootstrap
		}
		address, err := field()
		if err != nil {
			return "", ErrBootstrap
		}
		number, err := field()
		if err != nil {
			return "", ErrBootstrap
		}
		name, err := field()
		if err != nil {
			return "", ErrBootstrap
		}
		cookie, err := field()
		if err != nil {
			return "", ErrBootstrap
		}
		// GDM emits empty-number records, which libXau treats as a display
		// wildcard. This selects a cookie, never a destination: the configured
		// Unix display and kernel-peer/logind session check still bind the server.
		if (len(number) != 0 && string(number) != strconv.Itoa(int(display))) || string(name) != "MIT-MAGIC-COOKIE-1" {
			continue
		}
		if (family == 256 && string(address) == hostname) || family == 65535 {
			if len(cookie) != 16 {
				return "", ErrBootstrap
			}
			matches = append(matches, cookie)
		}
	}
	if len(matches) == 0 {
		return "", ErrBootstrap
	}
	for _, candidate := range matches[1:] {
		if !bytes.Equal(candidate, matches[0]) {
			return "", ErrBootstrap
		}
	}
	return hex.EncodeToString(matches[0]), nil
}

type GrantStatus struct {
	Active     []string  `json:"active"`
	ObservedAt time.Time `json:"observed_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

const GrantStatusLifetime = time.Second

func validGrantStatus(state GrantStatus, now time.Time) bool {
	if state.ObservedAt.IsZero() || state.ObservedAt.After(now) || !now.Before(state.ExpiresAt) || !state.ObservedAt.Before(state.ExpiresAt) || state.ExpiresAt.Sub(state.ObservedAt) > GrantStatusLifetime || len(state.Active) > 1024 {
		return false
	}
	seen := make(map[string]bool, len(state.Active))
	for _, id := range state.Active {
		if id == "" || len(id) > 128 || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func grantActive(path, id string, now time.Time) (bool, error) {
	data, err := readPrivate(path, 64*1024)
	if err != nil {
		return false, err
	}
	var state GrantStatus
	if decodeStrict(data, &state) != nil || !validGrantStatus(state, now) {
		return false, ErrBootstrap
	}
	for _, active := range state.Active {
		if active == id {
			return true, nil
		}
	}
	return false, nil
}
