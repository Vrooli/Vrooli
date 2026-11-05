package config

import (
	"fmt"
	"time"

	"compute-manager/internal/envreader"
)

type Config struct {
	LPBSBaseURL          string
	BridgeBaseURL        string
	ReconcileInterval    time.Duration
	ExpiryInterval       time.Duration
	MinimumBillableUnit  time.Duration
	TenantCeilingMinutes int64
}

func Load() (Config, error) {
	return LoadWith(envreader.System{})
}

func LoadWith(env envreader.Reader) (Config, error) {
	if env == nil {
		env = envreader.System{}
	}
	c := Config{
		LPBSBaseURL:          env.Getenv("COMPUTE_MANAGER_LPBS_BASE_URL"),
		BridgeBaseURL:        env.Getenv("COMPUTE_MANAGER_BRIDGE_BASE_URL"),
		ReconcileInterval:    15 * time.Minute,
		ExpiryInterval:       time.Minute,
		MinimumBillableUnit:  time.Hour,
		TenantCeilingMinutes: 1440,
	}
	var err error
	if c.ReconcileInterval, err = durationWith(env, "COMPUTE_MANAGER_RECONCILE_INTERVAL", c.ReconcileInterval); err != nil {
		return Config{}, err
	}
	if c.ExpiryInterval, err = durationWith(env, "COMPUTE_MANAGER_EXPIRY_INTERVAL", c.ExpiryInterval); err != nil {
		return Config{}, err
	}
	if c.MinimumBillableUnit, err = durationWith(env, "COMPUTE_MANAGER_MINIMUM_BILLABLE_UNIT", c.MinimumBillableUnit); err != nil {
		return Config{}, err
	}
	if value := env.Getenv("COMPUTE_MANAGER_TENANT_CEILING_MINUTES"); value != "" {
		var parsed int64
		if _, scanErr := fmt.Sscan(value, &parsed); scanErr != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("COMPUTE_MANAGER_TENANT_CEILING_MINUTES must be a positive integer: %q", value)
		}
		c.TenantCeilingMinutes = parsed
	}
	return c, nil
}

func durationWith(env envreader.Reader, key string, fallback time.Duration) (time.Duration, error) {
	value := env.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration: %q", key, value)
	}
	return parsed, nil
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	return durationWith(envreader.System{}, key, fallback)
}
