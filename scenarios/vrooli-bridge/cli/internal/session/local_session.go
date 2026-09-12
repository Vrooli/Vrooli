package session

import (
	"crypto/ed25519"
	"runtime"
	"time"

	shared "github.com/vrooli/api-core/operatorsession"
)

// These small adapters preserve the CLI package's test seam while keeping all
// key/metadata persistence and local minting in the shared operator-session
// contract. They intentionally contain no storage or token-persistence logic.
func loadLocalEnrollment() (ed25519.PrivateKey, shared.Enrollment, error) {
	store, err := shared.DefaultFileStore()
	if err != nil {
		return nil, shared.Enrollment{}, err
	}
	return store.Load()
}

func saveLocalEnrollment(private ed25519.PrivateKey, enrollment shared.Enrollment) error {
	store, err := shared.DefaultFileStore()
	if err != nil {
		return err
	}
	return store.Save(private, enrollment)
}

func mintLocalSession(now time.Time) (string, shared.Enrollment, error) {
	store, err := shared.DefaultFileStore()
	if err != nil {
		return "", shared.Enrollment{}, err
	}
	resolution, err := (shared.LocalResolver{Store: store, Now: func() time.Time { return now }}).Resolve()
	if err != nil {
		return "", shared.Enrollment{}, err
	}
	return resolution.Token, resolution.Enrollment, nil
}

// ResolveLocal is the auth command's explicit local refresh seam. It only
// reads the existing enrollment and signs a new short-lived session.
func ResolveLocal(now time.Time) (string, shared.Enrollment, error) {
	return mintLocalSession(now)
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
	runtime.KeepAlive(value)
}
