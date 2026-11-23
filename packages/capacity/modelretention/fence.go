package modelretention

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	platform "github.com/vrooli/platform-go"
)

// ErrModelFenced is returned to a request whose model is being evicted.
var ErrModelFenced = errors.New("model is being evicted from disk")

// ErrModelInUse is returned to an eviction while a request for the model (or
// a request whose model could not be identified) is in flight.
var ErrModelInUse = errors.New("model has a request in flight")

// anyModelKey is held by requests whose model could not be identified. An
// eviction must hold it exclusively too, so an unidentified request is never
// assumed to be about some other model.
const anyModelKey = "_unidentified"

// Fence is the per-model cross-process fence for one store.
type Fence struct {
	Dir string
}

// EnterRequest holds a shared lock on the request's model until release.
// An empty model holds the unidentified-request lock. ErrModelFenced means an
// eviction holds the model; any other error is an infrastructure failure the
// caller decides how to treat (the activity edge fails open).
func (f Fence) EnterRequest(model string) (func(), error) {
	key := CanonicalName(model)
	if key == "" {
		key = anyModelKey
	}
	file, err := f.open(key)
	if err != nil {
		return nil, err
	}
	release, err := platform.ReadLockFile(file, true)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, platform.ErrLockUnavailable) {
			return nil, ErrModelFenced
		}
		return nil, fmt.Errorf("lock model fence: %w", err)
	}
	return closeOnce(release, file), nil
}

// AcquireEviction takes the model's lock and the unidentified-request lock
// exclusively without waiting. It fails with ErrModelInUse while any request
// for the model, or any unidentified request, holds its shared lock. While the
// returned release is unheld, no new request for the model can start.
func (f Fence) AcquireEviction(model string) (func(), error) {
	key := CanonicalName(model)
	if key == "" {
		return nil, errors.New("eviction fence requires a model name")
	}
	var releases []func()
	releaseAll := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	for _, k := range []string{key, anyModelKey} {
		file, err := f.open(k)
		if err != nil {
			releaseAll()
			return nil, err
		}
		release, err := platform.LockFile(file, true)
		if err != nil {
			_ = file.Close()
			releaseAll()
			if errors.Is(err, platform.ErrLockUnavailable) {
				if k == anyModelKey {
					return nil, fmt.Errorf("%w: a request whose model could not be identified is in flight", ErrModelInUse)
				}
				return nil, ErrModelInUse
			}
			return nil, fmt.Errorf("lock model fence: %w", err)
		}
		if err := platform.VerifyLockFile(file, f.path(k)); err != nil {
			release()
			_ = file.Close()
			releaseAll()
			return nil, fmt.Errorf("verify model fence: %w", err)
		}
		releases = append(releases, closeOnce(release, file))
	}
	var once sync.Once
	return func() { once.Do(releaseAll) }, nil
}

func (f Fence) open(key string) (*os.File, error) {
	if strings.TrimSpace(f.Dir) == "" {
		return nil, errors.New("model fence directory is not configured")
	}
	if err := os.MkdirAll(f.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create model fence directory: %w", err)
	}
	file, err := platform.OpenLockFile(f.path(key), 0o600)
	if err != nil {
		return nil, fmt.Errorf("open model fence: %w", err)
	}
	return file, nil
}

// path keeps lock names readable and bounded: the sanitized name plus a short
// hash so distinct names never share a lock.
func (f Fence) path(key string) string {
	var b strings.Builder
	for _, r := range key {
		if b.Len() >= 80 {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(f.Dir, b.String()+"-"+hex.EncodeToString(sum[:4])+".lock")
}

func closeOnce(release func(), file *os.File) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			release()
			_ = file.Close()
		})
	}
}
