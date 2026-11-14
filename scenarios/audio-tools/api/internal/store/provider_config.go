package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vrooli/api-core/database"
)

// ProviderConfig mirrors the persisted provider-routing row.
type ProviderConfig struct {
	BYOKEnabled         bool
	VrooliEnabled       bool
	LocalEnabled        bool
	WhisperURL          string
	KokoroURL           string
	OllamaURL           string
	LPBSBaseURL         string
	LPBSAppBundleKey    string
	AvailTTLBYOKSeconds int32
	AvailTTLVrooliSecs  int32
	UpdatedAt           time.Time
}

// ProviderConfigPatch carries optional fields for partial updates.
// Nil means "do not change."
type ProviderConfigPatch struct {
	BYOKEnabled         *bool
	VrooliEnabled       *bool
	LocalEnabled        *bool
	WhisperURL          *string
	KokoroURL           *string
	OllamaURL           *string
	LPBSBaseURL         *string
	LPBSAppBundleKey    *string
	AvailTTLBYOKSeconds *int32
	AvailTTLVrooliSecs  *int32
}

// ProviderConfigStore reads/writes the singleton provider_config row.
type ProviderConfigStore struct {
	db       *database.RoutedDB
	defaults ProviderConfig
}

// NewProviderConfigStore returns a store seeded with the given defaults.
// The defaults apply only when no row exists yet (first boot).
func NewProviderConfigStore(db *database.RoutedDB, defaults ProviderConfig) *ProviderConfigStore {
	return &ProviderConfigStore{db: db, defaults: defaults}
}

// Get returns the current provider config, materialising defaults on
// first read.
func (s *ProviderConfigStore) Get(ctx context.Context) (ProviderConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT byok_enabled, vrooli_enabled, local_enabled,
			whisper_url, kokoro_url, ollama_url, lpbs_base_url, lpbs_app_bundle_key,
			avail_ttl_byok_seconds, avail_ttl_vrooli_seconds, updated_at
		FROM provider_config WHERE id = 1
	`)
	var (
		c            ProviderConfig
		byok, vr, lo int
		updatedAt    string
	)
	err := row.Scan(&byok, &vr, &lo,
		&c.WhisperURL, &c.KokoroURL, &c.OllamaURL, &c.LPBSBaseURL, &c.LPBSAppBundleKey,
		&c.AvailTTLBYOKSeconds, &c.AvailTTLVrooliSecs, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		if err := s.seed(ctx); err != nil {
			return ProviderConfig{}, err
		}
		return s.defaults, nil
	}
	if err != nil {
		return ProviderConfig{}, err
	}
	c.BYOKEnabled = byok != 0
	c.VrooliEnabled = vr != 0
	c.LocalEnabled = lo != 0
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

// Update applies the patch and returns the merged config.
func (s *ProviderConfigStore) Update(ctx context.Context, p ProviderConfigPatch) (ProviderConfig, error) {
	cur, err := s.Get(ctx)
	if err != nil {
		return ProviderConfig{}, err
	}
	if p.BYOKEnabled != nil {
		cur.BYOKEnabled = *p.BYOKEnabled
	}
	if p.VrooliEnabled != nil {
		cur.VrooliEnabled = *p.VrooliEnabled
	}
	if p.LocalEnabled != nil {
		cur.LocalEnabled = *p.LocalEnabled
	}
	if p.WhisperURL != nil {
		cur.WhisperURL = *p.WhisperURL
	}
	if p.KokoroURL != nil {
		cur.KokoroURL = *p.KokoroURL
	}
	if p.OllamaURL != nil {
		cur.OllamaURL = *p.OllamaURL
	}
	if p.LPBSBaseURL != nil {
		cur.LPBSBaseURL = *p.LPBSBaseURL
	}
	if p.LPBSAppBundleKey != nil {
		cur.LPBSAppBundleKey = *p.LPBSAppBundleKey
	}
	if p.AvailTTLBYOKSeconds != nil {
		cur.AvailTTLBYOKSeconds = *p.AvailTTLBYOKSeconds
	}
	if p.AvailTTLVrooliSecs != nil {
		cur.AvailTTLVrooliSecs = *p.AvailTTLVrooliSecs
	}
	cur.UpdatedAt = now()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO provider_config(
			id, byok_enabled, vrooli_enabled, local_enabled,
			whisper_url, kokoro_url, ollama_url, lpbs_base_url, lpbs_app_bundle_key,
			avail_ttl_byok_seconds, avail_ttl_vrooli_seconds, updated_at
		) VALUES (1, ?,?,?, ?,?,?,?,?, ?,?, ?)
		ON CONFLICT(id) DO UPDATE SET
			byok_enabled=excluded.byok_enabled,
			vrooli_enabled=excluded.vrooli_enabled,
			local_enabled=excluded.local_enabled,
			whisper_url=excluded.whisper_url,
			kokoro_url=excluded.kokoro_url,
			ollama_url=excluded.ollama_url,
			lpbs_base_url=excluded.lpbs_base_url,
			lpbs_app_bundle_key=excluded.lpbs_app_bundle_key,
			avail_ttl_byok_seconds=excluded.avail_ttl_byok_seconds,
			avail_ttl_vrooli_seconds=excluded.avail_ttl_vrooli_seconds,
			updated_at=excluded.updated_at
	`,
		boolInt(cur.BYOKEnabled), boolInt(cur.VrooliEnabled), boolInt(cur.LocalEnabled),
		cur.WhisperURL, cur.KokoroURL, cur.OllamaURL, cur.LPBSBaseURL, cur.LPBSAppBundleKey,
		cur.AvailTTLBYOKSeconds, cur.AvailTTLVrooliSecs, cur.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return ProviderConfig{}, err
	}
	return cur, nil
}

func (s *ProviderConfigStore) seed(ctx context.Context) error {
	d := s.defaults
	d.UpdatedAt = now()
	s.defaults = d
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO provider_config(
			id, byok_enabled, vrooli_enabled, local_enabled,
			whisper_url, kokoro_url, ollama_url, lpbs_base_url, lpbs_app_bundle_key,
			avail_ttl_byok_seconds, avail_ttl_vrooli_seconds, updated_at
		) VALUES (1, ?,?,?, ?,?,?,?,?, ?,?, ?)
	`,
		boolInt(d.BYOKEnabled), boolInt(d.VrooliEnabled), boolInt(d.LocalEnabled),
		d.WhisperURL, d.KokoroURL, d.OllamaURL, d.LPBSBaseURL, d.LPBSAppBundleKey,
		d.AvailTTLBYOKSeconds, d.AvailTTLVrooliSecs, d.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
