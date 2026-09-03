package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	LPBSBaseURL         string
	BridgeBaseURL       string
	ReconcileInterval   time.Duration
	ExpiryInterval      time.Duration
	MinimumBillableUnit time.Duration
}

func Load() (Config, error) {
	c := Config{
		LPBSBaseURL:         os.Getenv("COMPUTE_MANAGER_LPBS_BASE_URL"),
		BridgeBaseURL:       os.Getenv("COMPUTE_MANAGER_BRIDGE_BASE_URL"),
		ReconcileInterval:   15 * time.Minute,
		ExpiryInterval:      time.Minute,
		MinimumBillableUnit: time.Hour,
	}
	var err error
	if c.ReconcileInterval, err = duration("COMPUTE_MANAGER_RECONCILE_INTERVAL", c.ReconcileInterval); err != nil {
		return Config{}, err
	}
	if c.ExpiryInterval, err = duration("COMPUTE_MANAGER_EXPIRY_INTERVAL", c.ExpiryInterval); err != nil {
		return Config{}, err
	}
	if c.MinimumBillableUnit, err = duration("COMPUTE_MANAGER_MINIMUM_BILLABLE_UNIT", c.MinimumBillableUnit); err != nil {
		return Config{}, err
	}
	return c, nil
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration: %q", key, value)
	}
	return parsed, nil
}
