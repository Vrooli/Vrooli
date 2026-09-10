package authz

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
)

// Environment variables read by FromEnvironment. They are documented in
// docs/reference/configuration.md.
const (
	EnvAuthMode       = "VROOLI_AUTH_MODE"
	EnvAllowedOrigins = "VROOLI_ALLOWED_ORIGINS"
	EnvAllowedHosts   = "VROOLI_ALLOWED_HOSTS"
	EnvBindAddress    = "API_BIND_ADDRESS"
	EnvPolicyPath     = "SCENARIO_TO_CLOUD_AUTHZ_POLICY"
	EnvMaxBodyBytes   = "SCENARIO_TO_CLOUD_MAX_BODY_BYTES"
)

// Modes.
const (
	ModePersonalLocal = "personal_local"
	ModeShared        = "shared"
)

const (
	// DefaultMaxBodyBytes bounds JSON request bodies (2 MiB).
	DefaultMaxBodyBytes int64 = 2 << 20
	// DefaultEffectConcurrency is the per-principal ceiling on in-flight
	// effectful requests.
	DefaultEffectConcurrency = 4
)

// Config is the fully resolved boundary configuration.
type Config struct {
	// Authn is the provider chain; it must have at least one provider.
	Authn authn.Config
	// Mode records how the chain was built (personal_local or shared).
	Mode string
	// Policy binds principals to targets.
	Policy *Policy
	// AllowedOrigins are extra browser origins (scheme://host[:port]) that may
	// perform state-changing requests besides same-origin and, on a loopback
	// bind, loopback origins.
	AllowedOrigins []string
	// AllowedHosts are the Host header values the server answers for. Empty
	// means loopback hosts only, which is only valid on a loopback bind.
	AllowedHosts []string
	// BindLoopback is true when the listener is bound to a loopback address.
	BindLoopback bool
	// MaxBodyBytes bounds request bodies; <= 0 uses DefaultMaxBodyBytes.
	MaxBodyBytes int64
	// EffectConcurrency bounds per-principal in-flight effectful requests;
	// <= 0 uses DefaultEffectConcurrency.
	EffectConcurrency int
	// Logger receives structured audit lines. Nil discards them.
	Logger func(msg string, fields map[string]any)
	// Now is injectable for expiry tests.
	Now func() time.Time
}

// FromEnvironment resolves the boundary from the runtime environment.
//
//   - VROOLI_AUTH_MODE empty or personal_local: the loopback personal-local
//     provider with this scenario's three scopes, followed by any shared
//     providers from VROOLI_AUTH_PROVIDERS.
//   - VROOLI_AUTH_MODE shared: only the shared providers; an empty chain is a
//     startup error because the boundary never falls back to anonymous access.
//   - API_BIND_ADDRESS non-loopback requires VROOLI_ALLOWED_HOSTS.
func FromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	shared, err := authn.FromEnvironment(getenv)
	if err != nil {
		return Config{}, fmt.Errorf("shared authentication providers: %w", err)
	}
	cfg := Config{
		Policy:            NewPolicy(firstNonEmpty(getenv(EnvPolicyPath), DefaultPolicyPath())),
		AllowedOrigins:    splitList(getenv(EnvAllowedOrigins)),
		AllowedHosts:      splitList(getenv(EnvAllowedHosts)),
		MaxBodyBytes:      DefaultMaxBodyBytes,
		EffectConcurrency: DefaultEffectConcurrency,
	}
	mode := strings.ToLower(strings.TrimSpace(getenv(EnvAuthMode)))
	switch mode {
	case "", ModePersonalLocal:
		cfg.Mode = ModePersonalLocal
		providers := []authn.Provider{authn.NewPersonalLocalProvider(Scopes()...)}
		cfg.Authn = authn.Config{Providers: append(providers, shared.Providers...), RecoveryURL: shared.RecoveryURL}
	case ModeShared:
		cfg.Mode = ModeShared
		if !shared.Enabled() {
			return Config{}, errors.New(EnvAuthMode + "=shared requires VROOLI_AUTH_PROVIDERS; the management boundary never admits anonymous callers")
		}
		cfg.Authn = shared
	default:
		return Config{}, fmt.Errorf("unsupported %s value %q (personal_local or shared)", EnvAuthMode, mode)
	}
	if raw := strings.TrimSpace(getenv(EnvMaxBodyBytes)); raw != "" {
		var limit int64
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil || limit <= 0 {
			return Config{}, fmt.Errorf("invalid %s %q", EnvMaxBodyBytes, raw)
		}
		cfg.MaxBodyBytes = limit
	}
	bind := strings.TrimSpace(getenv(EnvBindAddress))
	cfg.BindLoopback = bind == "" || isLoopbackHost(bind)
	if !cfg.BindLoopback && len(cfg.AllowedHosts) == 0 {
		return Config{}, fmt.Errorf("%s=%s is not loopback; set %s to the host names this API answers for", EnvBindAddress, bind, EnvAllowedHosts)
	}
	return cfg, nil
}

func (c Config) maxBodyBytes() int64 {
	if c.MaxBodyBytes <= 0 {
		return DefaultMaxBodyBytes
	}
	return c.MaxBodyBytes
}

func (c Config) effectConcurrency() int {
	if c.EffectConcurrency <= 0 {
		return DefaultEffectConcurrency
	}
	return c.EffectConcurrency
}

func (c Config) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// isLoopbackHost accepts localhost, loopback IPs, and either with a port.
func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
