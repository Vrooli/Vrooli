package main

import (
	"strings"
	"testing"
	"time"
)

func clearUserSessionConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"LPBS_USER_ACCESS_TTL",
		"LPBS_USER_SESSION_IDLE_TTL",
		"LPBS_USER_SESSION_ABSOLUTE_TTL",
		"LPBS_USER_REAUTH_MAX_AGE",
		"LPBS_USER_REFRESH_GRACE",
	} {
		t.Setenv(key, "")
	}
}

func TestResolveUserSessionConfigDefaultsAndOverrides(t *testing.T) {
	clearUserSessionConfig(t)
	t.Setenv("LPBS_ENVIRONMENT", "development")
	config, err := resolveUserSessionConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.AccessTTL != 15*time.Minute || config.IdleTTL != 30*24*time.Hour || config.AbsoluteTTL != 90*24*time.Hour || config.ReauthMaxAge != 15*time.Minute || config.RefreshGrace != 30*time.Second {
		t.Fatalf("defaults = %+v", config)
	}
	t.Setenv("LPBS_USER_ACCESS_TTL", "20m")
	t.Setenv("LPBS_USER_SESSION_IDLE_TTL", "48h")
	t.Setenv("LPBS_USER_SESSION_ABSOLUTE_TTL", "120h")
	t.Setenv("LPBS_USER_REAUTH_MAX_AGE", "30m")
	t.Setenv("LPBS_USER_REFRESH_GRACE", "45s")
	config, err = resolveUserSessionConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.AccessTTL != 20*time.Minute || config.IdleTTL != 48*time.Hour || config.AbsoluteTTL != 120*time.Hour || config.ReauthMaxAge != 30*time.Minute || config.RefreshGrace != 45*time.Second {
		t.Fatalf("overrides = %+v", config)
	}
}

func TestResolveUserSessionConfigDevelopmentClampsInvalidValues(t *testing.T) {
	clearUserSessionConfig(t)
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("LPBS_USER_ACCESS_TTL", "1m")
	t.Setenv("LPBS_USER_SESSION_IDLE_TTL", "1h")
	t.Setenv("LPBS_USER_SESSION_ABSOLUTE_TTL", "2h")
	t.Setenv("LPBS_USER_REAUTH_MAX_AGE", "2h")
	t.Setenv("LPBS_USER_REFRESH_GRACE", "90s")
	config, err := resolveUserSessionConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.AccessTTL != 5*time.Minute || config.IdleTTL != 24*time.Hour || config.AbsoluteTTL != 24*time.Hour || config.ReauthMaxAge != time.Hour || config.RefreshGrace != time.Minute {
		t.Fatalf("clamped config = %+v", config)
	}
}

func TestResolveUserSessionConfigProductionRejectsInvalidValues(t *testing.T) {
	clearUserSessionConfig(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("LPBS_USER_ACCESS_TTL", "1m")
	_, err := resolveUserSessionConfig()
	if err == nil || !strings.Contains(err.Error(), "LPBS_USER_ACCESS_TTL") {
		t.Fatalf("production invalid config error = %v", err)
	}

	t.Setenv("LPBS_USER_ACCESS_TTL", "15m")
	t.Setenv("LPBS_USER_SESSION_IDLE_TTL", "48h")
	t.Setenv("LPBS_USER_SESSION_ABSOLUTE_TTL", "24h")
	_, err = resolveUserSessionConfig()
	if err == nil || !strings.Contains(err.Error(), "LPBS_USER_SESSION_ABSOLUTE_TTL") {
		t.Fatalf("production absolute-before-idle error = %v", err)
	}
}
