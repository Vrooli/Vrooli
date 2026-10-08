package desktoplink

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is a deterministic repository for unit and transport tests.
// Production composition uses SQLRepository below; keeping this fake behind the
// same interface prevents protocol tests from depending on PostgreSQL.
type MemoryRepository struct {
	mu    sync.Mutex
	auths map[string]Authorization
	links map[string]Link
	now   func() time.Time
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{auths: make(map[string]Authorization), links: make(map[string]Link), now: time.Now}
}

func (r *MemoryRepository) CreateAuthorization(_ context.Context, authorization Authorization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.auths {
		if existing.CodeHash == authorization.CodeHash {
			return errors.New("desktop authorization code collision")
		}
	}
	r.auths[authorization.ID] = authorization
	return nil
}

func (r *MemoryRepository) RedeemAuthorization(_ context.Context, codeHash, verifier, localPrincipal, installationID, resource string) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, authorization := range r.auths {
		if authorization.CodeHash != codeHash {
			continue
		}
		if !r.now().Before(authorization.ExpiresAt) || !pkceMatches(verifier, authorization.CodeChallenge) || authorization.InstallationID != installationID || authorization.Resource != resource {
			return Link{}, ErrCodeRejected
		}
		delete(r.auths, id)
		linkID, err := randomSecret()
		if err != nil {
			return Link{}, err
		}
		link := Link{ID: linkID, LPBSUserID: authorization.LPBSUserID, BusinessAccountID: authorization.BusinessAccountID, LocalProvider: "scenario-authenticator", LocalPrincipal: localPrincipal, InstallationID: installationID, Resource: resource, Audience: authorization.Audience, Scopes: clone(authorization.Scopes), CreatedAt: r.now().UTC()}
		r.links[link.ID] = link
		return link, nil
	}
	return Link{}, ErrCodeRejected
}

func (r *MemoryRepository) StatusByLocal(_ context.Context, principal, installationID, resource string) (Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, link := range r.links {
		if link.LocalPrincipal == principal && link.InstallationID == installationID && link.Resource == resource {
			return link, nil
		}
	}
	return Link{}, ErrCodeRejected
}

func (r *MemoryRepository) RevokeByLPBS(_ context.Context, userID, installationID, resource, actor string) (int64, error) {
	return r.revoke(func(link Link) bool {
		return link.LPBSUserID == userID && link.InstallationID == installationID && link.Resource == resource
	}, actor)
}

func (r *MemoryRepository) RevokeByLocal(_ context.Context, principal, installationID, resource, actor string) (int64, error) {
	return r.revoke(func(link Link) bool {
		return link.LocalPrincipal == principal && link.InstallationID == installationID && link.Resource == resource
	}, actor)
}

func (r *MemoryRepository) revoke(matches func(Link) bool, actor string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now().UTC()
	var count int64
	for id, link := range r.links {
		if link.RevokedAt == nil && matches(link) {
			link.RevokedAt = &now
			link.RevokedBy = strings.TrimSpace(actor)
			r.links[id] = link
			count++
		}
	}
	return count, nil
}

func pkceMatches(verifier, challenge string) bool {
	digest := sha256.Sum256([]byte(verifier))
	derived := base64.RawURLEncoding.EncodeToString(digest[:])
	return derived == challenge
}
