package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
)

var ErrNotFound = errors.New("lighting policy state not found")

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Store persists configuration and the replay state together so a restart
// cannot pair new policy thresholds with an old evaluator checkpoint.
type Store interface {
	Load(context.Context) (*policyv1.PolicyConfig, State, error)
	Save(context.Context, *policyv1.PolicyConfig, State) error
}

// LoadForRead returns detached normalized configuration and replay state without
// initializing or migrating persisted state. Read previews cannot commit policy.
func LoadForRead(ctx context.Context, store Store) (*policyv1.PolicyConfig, State, error) {
	config, state, _, err := loadNormalized(ctx, store)
	return config, state, err
}

func LoadOrInitialize(ctx context.Context, store Store) (*policyv1.PolicyConfig, State, error) {
	config, state, changed, err := loadNormalized(ctx, store)
	if err != nil {
		return nil, State{}, err
	}
	if changed {
		if err := store.Save(ctx, config, state); err != nil {
			return nil, State{}, err
		}
	}
	return config, state, nil
}

func loadNormalized(ctx context.Context, store Store) (*policyv1.PolicyConfig, State, bool, error) {
	config, state, err := store.Load(ctx)
	if errors.Is(err, ErrNotFound) {
		return DefaultConfig(), State{}, true, nil
	}
	if err != nil {
		return nil, State{}, false, err
	}
	if config == nil {
		return nil, State{}, false, errors.New("stored lighting policy config is required")
	}
	config = proto.Clone(config).(*policyv1.PolicyConfig)
	defaults := DefaultConfig()
	changed := false
	if config.HealthPalette == nil {
		config.HealthPalette = defaults.HealthPalette
		changed = true
	}
	if config.ActivityEffects == nil || config.ActivityEffects.BusyAgentThreshold == 0 {
		config.ActivityEffects = defaults.ActivityEffects
		changed = true
	}
	return config, state, changed, nil
}

type sqliteStore struct {
	db  SQLExecutor
	now func() time.Time
}

func NewSQLiteStore(db SQLExecutor) Store {
	return &sqliteStore{db: db, now: time.Now}
}

func (s *sqliteStore) Load(ctx context.Context) (*policyv1.PolicyConfig, State, error) {
	var configBytes []byte
	var stateBytes string
	err := s.db.QueryRowContext(ctx, `
SELECT config_proto, evaluation_state
FROM lighting_policy_state WHERE id = 'default'
`).Scan(&configBytes, &stateBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, State{}, ErrNotFound
	}
	if err != nil {
		return nil, State{}, fmt.Errorf("load lighting policy: %w", err)
	}
	config := &policyv1.PolicyConfig{}
	if err := proto.Unmarshal(configBytes, config); err != nil {
		return nil, State{}, fmt.Errorf("decode lighting policy config: %w", err)
	}
	var state State
	if err := json.Unmarshal([]byte(stateBytes), &state); err != nil {
		return nil, State{}, fmt.Errorf("decode lighting policy evaluator state: %w", err)
	}
	return config, state, nil
}

func (s *sqliteStore) Save(ctx context.Context, config *policyv1.PolicyConfig, state State) error {
	if config == nil {
		return errors.New("lighting policy config is required")
	}
	if err := (Engine{Config: config}).Validate(); err != nil {
		return fmt.Errorf("validate lighting policy: %w", err)
	}
	configBytes, err := proto.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode lighting policy config: %w", err)
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode lighting policy evaluator state: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO lighting_policy_state (id, config_proto, evaluation_state, updated_at)
VALUES ('default', ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  config_proto = excluded.config_proto,
  evaluation_state = excluded.evaluation_state,
  updated_at = excluded.updated_at
`, configBytes, string(stateBytes), s.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save lighting policy: %w", err)
	}
	return nil
}

// DefaultConfig starts safely with lights Off. Telemetry-driven policies are
// available as editable presets and only become active by operator choice.
func DefaultConfig() *policyv1.PolicyConfig {
	return &policyv1.PolicyConfig{
		DefaultPreset:                  policyv1.PresetKind_PRESET_KIND_OFF,
		SourceMaxAgeSeconds:            60,
		LoadEnterPercent:               80,
		LoadExitPercent:                60,
		DebounceSeconds:                5,
		BrightnessCapPercent:           35,
		StaleFallbackColor:             &controlv1.Color{Red: 32, Green: 48, Blue: 64},
		StaleFallbackBrightnessPercent: 20,
		Presets: []*policyv1.Preset{
			{Kind: policyv1.PresetKind_PRESET_KIND_OFF, Label: "Off", Enabled: true, BrightnessCapPercent: 0},
			{Kind: policyv1.PresetKind_PRESET_KIND_LOAD, Label: "Load", Color: &controlv1.Color{Red: 255, Green: 112, Blue: 48}, Enabled: true, BrightnessCapPercent: 30},
			{Kind: policyv1.PresetKind_PRESET_KIND_WORK, Label: "Work", Color: &controlv1.Color{Red: 48, Green: 128, Blue: 224}, Enabled: true, BrightnessCapPercent: 25},
			{Kind: policyv1.PresetKind_PRESET_KIND_HEALTH, Label: "Health", Color: &controlv1.Color{Red: 255, Green: 64, Blue: 32}, Enabled: true, BrightnessCapPercent: 35},
			{Kind: policyv1.PresetKind_PRESET_KIND_COMBINED, Label: "Combined", Color: &controlv1.Color{Red: 40, Green: 176, Blue: 120}, EffectId: "Breathing", Enabled: true, BrightnessCapPercent: 25},
			{Kind: policyv1.PresetKind_PRESET_KIND_STATIC, Label: "Static", Color: &controlv1.Color{Red: 32, Green: 48, Blue: 64}, Enabled: true, BrightnessCapPercent: 20},
		},
		HealthPalette: &policyv1.HealthPalette{
			Healthy:   &controlv1.Color{Red: 87, Green: 190, Blue: 138},
			Attention: &controlv1.Color{Red: 217, Green: 154, Blue: 36},
			Failure:   &controlv1.Color{Red: 217, Green: 77, Blue: 77},
		},
		ActivityEffects: &policyv1.ActivityEffects{ActiveEffectId: "Breathing", BusyEffectId: "Breathing", BusyAgentThreshold: 5},
	}
}
