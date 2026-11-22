package modelstorage

import (
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/retention"
)

// Environment switches. Every value is optional; unset keeps the default.
const (
	EnvAutoAcquire = "IMAGE_TOOLS_MODEL_AUTO_ACQUIRE"
	EnvAutoOffload = "IMAGE_TOOLS_MODEL_AUTO_OFFLOAD"
	EnvReserve     = "IMAGE_TOOLS_MODEL_OFFLOAD_RESERVE"
	EnvAmple       = "IMAGE_TOOLS_MODEL_OFFLOAD_AMPLE"
	EnvMinIdle     = "IMAGE_TOOLS_MODEL_OFFLOAD_MIN_IDLE"
	EnvMaxIdle     = "IMAGE_TOOLS_MODEL_OFFLOAD_MAX_IDLE"
	EnvInterval    = "IMAGE_TOOLS_MODEL_OFFLOAD_INTERVAL"
	// EnvStagingGrace is how long an unowned acquisition staging directory must
	// go without a write before it is reclaimed as abandoned.
	EnvStagingGrace = "IMAGE_TOOLS_MODEL_STAGING_GRACE"
	defaultInterval = 30 * time.Minute
	// DefaultStartupDelay leaves time to review `image-tools models storage`
	// after a restart before the first automatic pass can remove anything.
	DefaultStartupDelay = 20 * time.Minute
)

// ConfigFromEnv builds the service configuration. floorBytes is the shared
// disk floor every Image Tools disk admission requests.
func ConfigFromEnv(getenv func(string) string, floorBytes int64) (Config, error) {
	cfg := Config{Policy: DefaultPolicy(), AutoAcquire: true, AutoOffload: true, Interval: defaultInterval, StartupDelay: DefaultStartupDelay, StagingGrace: DefaultStagingGrace, StagingBootDelay: DefaultStagingBootDelay}
	cfg.Policy.FloorBytes = floorBytes
	var err error
	if cfg.AutoAcquire, err = envBool(getenv, EnvAutoAcquire, true); err != nil {
		return cfg, err
	}
	if cfg.AutoOffload, err = envBool(getenv, EnvAutoOffload, true); err != nil {
		return cfg, err
	}
	for _, item := range []struct {
		name string
		dst  *int64
	}{{EnvReserve, &cfg.Policy.ReserveBytes}, {EnvAmple, &cfg.Policy.AmpleBytes}} {
		if raw := strings.TrimSpace(getenv(item.name)); raw != "" {
			v, err := retention.ParseBytes(raw)
			if err != nil || v < 0 {
				return cfg, fmt.Errorf("%s: %q is not a size such as 24GiB", item.name, raw)
			}
			*item.dst = v
		}
	}
	for _, item := range []struct {
		name string
		dst  *time.Duration
	}{{EnvMinIdle, &cfg.Policy.MinIdle}, {EnvMaxIdle, &cfg.Policy.MaxIdle}, {EnvInterval, &cfg.Interval}, {EnvStagingGrace, &cfg.StagingGrace}} {
		if raw := strings.TrimSpace(getenv(item.name)); raw != "" {
			v, err := time.ParseDuration(raw)
			if err != nil || v <= 0 {
				return cfg, fmt.Errorf("%s: %q is not a positive duration such as 24h", item.name, raw)
			}
			*item.dst = v
		}
	}
	if cfg.Policy.AmpleBytes <= cfg.Policy.ReserveBytes {
		return cfg, fmt.Errorf("%s (%d) must exceed %s (%d)", EnvAmple, cfg.Policy.AmpleBytes, EnvReserve, cfg.Policy.ReserveBytes)
	}
	if cfg.Policy.MaxIdle < cfg.Policy.MinIdle {
		return cfg, fmt.Errorf("%s must not be shorter than %s", EnvMaxIdle, EnvMinIdle)
	}
	return cfg, nil
}

func envBool(getenv func(string) string, name string, def bool) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(getenv(name))) {
	case "":
		return def, nil
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return def, fmt.Errorf("%s: want on/off, true/false, 1/0 or yes/no", name)
	}
}
