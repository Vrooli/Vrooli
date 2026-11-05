package config

import "context"

// ConfigRepository is the persistence seam the config service depends on.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire a fake. The config table is a single logical
// row (a singleton keyed on a fixed id), so the surface is narrow:
// read-the-row, upsert-the-row.
type ConfigRepository interface {
	// Get returns the persisted config. When nothing has been persisted
	// yet it returns the domain defaults (DefaultMode / DefaultPromEndpoint)
	// rather than a not-found error — there is always exactly one logical
	// config, even before it is first written.
	Get(ctx context.Context) (TunnelConfig, error)

	// Upsert persists cfg into the single config row, inserting it on first
	// write and overwriting it thereafter. Returns the stored config.
	Upsert(ctx context.Context, cfg TunnelConfig) (TunnelConfig, error)
}
