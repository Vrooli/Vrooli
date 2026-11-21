package gocache

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StateDirName is the protocol's directory inside a store. cmd/go ignores it:
// its trim visits only the hex shard directories and `go clean -cache`
// removes only those shards and log.txt.
const StateDirName = ".vrooli-gocache"

const (
	coordinationLockName = "coordination.lock"
	reclaimLockName      = "reclaim.lock"
	storeTokenName       = "store-token"
	sessionsDirName      = "sessions"
	planFileName         = "plan.json"
	cursorFileName       = "cursor.json"
	receiptFileName      = "last-reclaim.json"
)

// protocolState is one process's handle on a store's coordination state.
type protocolState struct {
	store Store
	dir   string
	coord *os.File
}

func openState(store Store) (*protocolState, error) {
	if !protocolSupported {
		return nil, errors.New("the Go build-cache protocol is not supported on this platform")
	}
	dir := filepath.Join(store.Root, StateDirName)
	if err := os.MkdirAll(filepath.Join(dir, sessionsDirName), 0o755); err != nil {
		return nil, fmt.Errorf("create build-cache protocol state: %w", err)
	}
	coord, err := os.OpenFile(filepath.Join(dir, coordinationLockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open build-cache coordination lock: %w", err)
	}
	return &protocolState{store: store, dir: dir, coord: coord}, nil
}

func (s *protocolState) Close() error {
	if s == nil || s.coord == nil {
		return nil
	}
	return s.coord.Close()
}

func (s *protocolState) path(name string) string { return filepath.Join(s.dir, name) }

func (s *protocolState) sessionsDir() string { return filepath.Join(s.dir, sessionsDirName) }

// shared runs fn while holding the coordination lock shared. Sessions use it
// to make a lookup and its pin one step with respect to reclaim.
func (s *protocolState) shared(fn func() error) error {
	if err := lockShared(s.coord); err != nil {
		return fmt.Errorf("acquire build-cache coordination lock: %w", err)
	}
	err := fn()
	if unlockErr := unlock(s.coord); err == nil && unlockErr != nil {
		err = fmt.Errorf("release build-cache coordination lock: %w", unlockErr)
	}
	return err
}

// StoreID returns the identity a plan binds to: a random token created with
// the protocol state, combined with the store directory's device and inode.
// Replacing or recreating the store changes it.
func (s *protocolState) StoreID() (string, error) {
	token, err := s.storeToken()
	if err != nil {
		return "", err
	}
	info, err := os.Stat(s.store.Root)
	if err != nil {
		return "", err
	}
	device, inode, ok := fileIdentity(info)
	if !ok {
		return "", errors.New("store identity is unavailable on this platform")
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%d", token, device, inode))
	return hex.EncodeToString(sum[:12]), nil
}

func (s *protocolState) storeToken() (string, error) {
	path := s.path(storeTokenName)
	if data, err := os.ReadFile(path); err == nil {
		return validToken(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	// Link a complete temporary file into place so a concurrent reader never
	// observes an empty token; the loser of a creation race reads the winner's.
	temp, err := os.CreateTemp(s.dir, storeTokenName+tempMarker+"*")
	if err != nil {
		return "", err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.WriteString(hex.EncodeToString(raw[:])); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Link(temp.Name(), path); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return validToken(data)
}

func validToken(data []byte) (string, error) {
	token := strings.TrimSpace(string(data))
	if len(token) != 32 {
		return "", fmt.Errorf("build-cache store token is corrupt")
	}
	if _, err := hex.DecodeString(token); err != nil {
		return "", fmt.Errorf("build-cache store token is corrupt")
	}
	return token, nil
}

// writeJSONAtomic replaces a small state file without exposing a partial
// document to concurrent readers.
func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+tempMarker+"*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(append(data, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
