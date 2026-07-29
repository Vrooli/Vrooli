package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/database"
	"landing-page-business-suite-api/internal/delivery"
)

// routedDownloadStore adapts RoutedDB to the delivery catalog persistence
// boundary. Context-free compatibility operations intentionally use primary;
// request-aware reads and writes retain the test-mode routing marker.
type routedDownloadStore struct {
	db *database.RoutedDB
}

func newRoutedDownloadStore(db *database.RoutedDB) delivery.CatalogStore {
	return routedDownloadStore{db: db}
}

func (s routedDownloadStore) Query(query string, args ...any) (*sql.Rows, error) {
	return s.db.Primary().Query(query, args...)
}

func (s routedDownloadStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s routedDownloadStore) QueryRow(query string, args ...any) *sql.Row {
	// #nosec G701 -- DownloadService owns all call sites and supplies package-constant SQL.
	// Dynamic SQL is not accepted at this persistence boundary.
	return s.db.Primary().QueryRow(query, args...)
}

func (s routedDownloadStore) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Primary().Exec(query, args...)
}

func (s routedDownloadStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s routedDownloadStore) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

var (
	// ErrDownloadRequiresActiveSubscription indicates a gated download without active access.
	ErrDownloadRequiresActiveSubscription = errors.New("active subscription required for downloads")
	// ErrDownloadIdentityRequired indicates the caller must provide identity details before accessing gated assets.
	ErrDownloadIdentityRequired = errors.New("user identity required for gated downloads")
	// ErrDownloadPlatformRequired indicates the platform input was blank.
	ErrDownloadPlatformRequired = errors.New("platform is required")
	// ErrDownloadEntitlementsUnavailable indicates the entitlement provider returned an unusable response.
	ErrDownloadEntitlementsUnavailable = errors.New("entitlements unavailable")
)

// Delivery catalog models and persistence are owned by internal/delivery.
type (
	DownloadStore      = delivery.CatalogStore
	DownloadService    = delivery.CatalogService
	DownloadAsset      = delivery.Asset
	DownloadStorefront = delivery.Storefront
	DownloadApp        = delivery.App
)

var (
	ErrDownloadNotFound    = delivery.ErrAssetNotFound
	ErrDownloadAppNotFound = delivery.ErrAppNotFound
)

func NewDownloadService(db delivery.CatalogStore) *delivery.CatalogService {
	return delivery.NewCatalogService(db)
}

type downloadAssetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error)
}

type entitlementProvider interface {
	GetEntitlementsContext(context.Context, string) (*EntitlementPayload, error)
}

type entitlementStatusLookup struct {
	ctx      context.Context
	provider entitlementProvider
}

func (l entitlementStatusLookup) GetStatus(userIdentity string) (string, error) {
	entitlements, err := l.provider.GetEntitlementsContext(l.ctx, userIdentity)
	if err != nil {
		return "", err
	}
	if entitlements == nil {
		return "", delivery.ErrEntitlementsUnavailable
	}
	return entitlements.Status, nil
}

// DownloadAuthorizer coordinates entitlement checks before returning assets.
type DownloadAuthorizer struct {
	downloads    downloadAssetLookup
	entitlements entitlementProvider
	bundleKey    string
}

// NewDownloadAuthorizer wires the dependencies required for download gating.
func NewDownloadAuthorizer(downloads downloadAssetLookup, entitlements entitlementProvider, bundleKey string) *DownloadAuthorizer {
	return &DownloadAuthorizer{
		downloads:    downloads,
		entitlements: entitlements,
		bundleKey:    bundleKey,
	}
}

// Authorize ensures the caller can access the requested download asset.
func (a *DownloadAuthorizer) Authorize(ctx context.Context, appKey string, platform string, userIdentity string) (*DownloadAsset, error) {
	trimmedApp := strings.TrimSpace(appKey)
	if trimmedApp == "" {
		return nil, ErrDownloadAppNotFound
	}
	trimmedPlatform := strings.TrimSpace(platform)
	if trimmedPlatform == "" {
		return nil, ErrDownloadPlatformRequired
	}

	asset, err := a.downloads.GetAsset(a.bundleKey, trimmedApp, trimmedPlatform)
	if err != nil {
		return nil, err
	}

	err = delivery.Authorize(delivery.Request{
		AppKey:              trimmedApp,
		Platform:            trimmedPlatform,
		UserIdentity:        userIdentity,
		RequiresEntitlement: asset.RequiresEntitlement,
	}, entitlementStatusLookup{ctx: ctx, provider: a.entitlements})
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrIdentityRequired):
			return nil, ErrDownloadIdentityRequired
		case errors.Is(err, delivery.ErrRequiresActiveSubscription):
			return nil, ErrDownloadRequiresActiveSubscription
		case errors.Is(err, delivery.ErrEntitlementsUnavailable):
			return nil, fmt.Errorf("retrieve entitlements: %w", ErrDownloadEntitlementsUnavailable)
		default:
			return nil, err
		}
	}

	return asset, nil
}
