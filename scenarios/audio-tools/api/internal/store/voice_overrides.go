package store

import (
	"context"

	"github.com/vrooli/api-core/database"
)

// VoiceOverride maps a canonical voice id to an adapter-specific id
// for a single tier+provider combination.
type VoiceOverride struct {
	CanonicalVoice string
	TierProvider   string // "byok:elevenlabs", "local:kokoro", ...
	AdapterVoice   string
}

type VoiceOverrideStore struct{ db *database.RoutedDB }

func NewVoiceOverrideStore(db *database.RoutedDB) *VoiceOverrideStore {
	return &VoiceOverrideStore{db: db}
}

// List returns every override in stable order.
func (s *VoiceOverrideStore) List(ctx context.Context) ([]VoiceOverride, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT canonical_voice, tier_provider, adapter_voice
		FROM voice_overrides ORDER BY canonical_voice, tier_provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VoiceOverride
	for rows.Next() {
		var v VoiceOverride
		if err := rows.Scan(&v.CanonicalVoice, &v.TierProvider, &v.AdapterVoice); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Set upserts the override. If AdapterVoice is empty the row is deleted.
func (s *VoiceOverrideStore) Set(ctx context.Context, v VoiceOverride) error {
	if v.AdapterVoice == "" {
		_, err := s.db.ExecContext(ctx, `DELETE FROM voice_overrides WHERE canonical_voice=? AND tier_provider=?`,
			v.CanonicalVoice, v.TierProvider)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO voice_overrides(canonical_voice, tier_provider, adapter_voice)
		VALUES (?,?,?)
		ON CONFLICT(canonical_voice, tier_provider) DO UPDATE SET adapter_voice=excluded.adapter_voice
	`, v.CanonicalVoice, v.TierProvider, v.AdapterVoice)
	return err
}
