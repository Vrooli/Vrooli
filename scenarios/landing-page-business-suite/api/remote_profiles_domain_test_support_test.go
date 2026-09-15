package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"landing-page-business-suite-api/internal/administration"
)

type (
	RemoteProfile                = administration.RemoteProfile
	RemoteProfileCreateRequest   = administration.RemoteProfileCreateRequest
	RemoteProfileUpdateRequest   = administration.RemoteProfileUpdateRequest
	RemoteProfileLoginRequest    = administration.RemoteProfileLoginRequest
	RemoteProfileProxyRequest    = administration.RemoteProfileProxyRequest
	RemoteProxyResponse          = administration.RemoteProxyResponse
	IncomingRemoteProfileSession = administration.IncomingRemoteProfileSession
	RemoteProfileSessionLinks    = administration.RemoteProfileSessionLinks
	RemoteProfileError           = administration.RemoteProfileError
)

var (
	ErrRemoteProfileNotFound       = administration.ErrRemoteProfileNotFound
	ErrRemoteProfileTagExists      = administration.ErrRemoteProfileTagExists
	ErrRemoteProfileInvalid        = administration.ErrRemoteProfileInvalid
	ErrRemoteProfileSessionMissing = administration.ErrRemoteProfileSessionMissing
	ErrRemoteProfileDisallowedPath = administration.ErrRemoteProfileDisallowedPath
	remoteProfileCookieName        = administration.RemoteProfileCookieName
	remoteProfileStatusUnknown     = administration.RemoteProfileStatusUnknown
	remoteProfileStatusActive      = administration.RemoteProfileStatusActive
	remoteProfileStatusExpired     = administration.RemoteProfileStatusExpired
	remoteProfileStatusError       = administration.RemoteProfileStatusError
	normalizeRemoteProfileTag      = administration.NormalizeRemoteProfileTag
	normalizeRemoteProfileLabel    = administration.NormalizeRemoteProfileLabel
	normalizeRemoteProfileAPIBase  = administration.NormalizeRemoteProfileAPIBase
	normalizeRemoteProxyPath       = administration.NormalizeRemoteProxyPath
	readLimitedBody                = administration.ReadLimitedBody
	classifyRemoteError            = administration.ClassifyRemoteError
	extractRemoteErrorMessage      = administration.ExtractRemoteErrorMessage
	mapRemoteStatus                = administration.MapRemoteStatus
	isAllowedRemoteProxyPath       = administration.IsAllowedRemoteProxyPath
)

// RemoteProfileService is test-only compatibility for legacy package-main
// characterization tests. Production composes the administration service directly.
type RemoteProfileService struct {
	db            administration.RemoteProfileStore
	encryptionKey []byte
	httpClient    administration.HTTPDoer
	now           func() time.Time
}

type remoteProfileRecord struct {
	ID               int64
	Tag              string
	APIBase          string
	Status           string
	EncryptedSession sql.NullString
	SessionExpiresAt sql.NullTime
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (r *remoteProfileRecord) toProfile(now time.Time) RemoteProfile {
	status := r.Status
	if r.SessionExpiresAt.Valid && r.SessionExpiresAt.Time.Before(now) {
		status = administration.RemoteProfileStatusExpired
	}
	return RemoteProfile{ID: r.ID, Tag: r.Tag, APIBase: r.APIBase, Status: status, HasSession: r.EncryptedSession.Valid && r.EncryptedSession.String != "", CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *RemoteProfileService) domain() *administration.RemoteProfileService {
	return &administration.RemoteProfileService{DB: s.db, EncryptionKey: s.encryptionKey, HTTPClient: s.httpClient, Now: s.now}
}

func (s *RemoteProfileService) List(ctx context.Context) ([]administration.RemoteProfile, error) {
	return s.domain().List(ctx)
}

func (s *RemoteProfileService) Create(ctx context.Context, req administration.RemoteProfileCreateRequest, email string) (*administration.RemoteProfile, error) {
	return s.domain().Create(ctx, req, email)
}

func (s *RemoteProfileService) Update(ctx context.Context, id int64, req administration.RemoteProfileUpdateRequest) (*administration.RemoteProfile, error) {
	return s.domain().Update(ctx, id, req)
}

func (s *RemoteProfileService) Delete(ctx context.Context, id int64) error {
	return s.domain().Delete(ctx, id)
}

func (s *RemoteProfileService) Login(ctx context.Context, id int64, email, password string) (*administration.RemoteProfile, error) {
	return s.domain().Login(ctx, id, email, password)
}

func (s *RemoteProfileService) Logout(ctx context.Context, id int64) (*administration.RemoteProfile, error) {
	return s.domain().Logout(ctx, id)
}

func (s *RemoteProfileService) Test(ctx context.Context, id int64) (*administration.RemoteProfile, error) {
	return s.domain().Test(ctx, id)
}

func (s *RemoteProfileService) SessionLinks(ctx context.Context, id int64) (*administration.RemoteProfileSessionLinks, error) {
	return s.domain().SessionLinks(ctx, id)
}

func (s *RemoteProfileService) RevokeRemoteSessions(ctx context.Context, id int64) (*administration.RemoteProfileSessionLinks, error) {
	return s.domain().RevokeRemoteSessions(ctx, id)
}

func (s *RemoteProfileService) Proxy(ctx context.Context, id int64, req administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error) {
	return s.domain().Proxy(ctx, id, req)
}

func (s *RemoteProfileService) GetByID(ctx context.Context, id int64) (*administration.RemoteProfile, error) {
	return s.domain().GetByID(ctx, id)
}

func (s *RemoteProfileService) encrypt(value string) (string, error) {
	return s.domain().Encrypt(value)
}

func (s *RemoteProfileService) decrypt(value string) (string, error) {
	return s.domain().Decrypt(value)
}

func (s *RemoteProfileService) ensureConnectorID(ctx context.Context, id int64, current string) (string, error) {
	return s.domain().EnsureConnectorID(ctx, id, current)
}

func (s *RemoteProfileService) setSession(ctx context.Context, id int64, value, sessionID string, expiresAt *time.Time) error {
	return s.domain().SetSession(ctx, id, value, sessionID, expiresAt)
}

func (s *RemoteProfileService) buildRemoteURL(base, path string, query map[string]string) (string, error) {
	return s.domain().BuildRemoteURL(base, path, query)
}

func (s *RemoteProfileService) remoteLogin(ctx context.Context, base, email, password string, metadata administration.RemoteProfileSessionMetadata) (string, string, *time.Time, error) {
	return s.domain().RemoteLogin(ctx, base, email, password, metadata)
}

func (s *RemoteProfileService) remoteSessionCheck(ctx context.Context, base, session string) (bool, error) {
	return s.domain().RemoteSessionCheck(ctx, base, session)
}

func (s *RemoteProfileService) remoteLogout(ctx context.Context, base, session string) error {
	return s.domain().RemoteLogout(ctx, base, session)
}

func (s *RemoteProfileService) doJSONRequest(ctx context.Context, method, url string, body []byte, cookies []*http.Cookie) (*http.Response, []byte, error) {
	return s.domain().DoJSONRequest(ctx, method, url, body, cookies)
}
