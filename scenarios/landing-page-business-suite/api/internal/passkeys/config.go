package passkeys

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

// LoadConfig derives the relying-party identity from the public origin. An
// explicit RP ID is accepted only when it is the origin host or a parent
// domain, preventing a passkey from being silently registered for another
// relying party.
func LoadConfig(publicBaseURL, displayName string, production bool) (Config, error) {
	base, err := url.Parse(strings.TrimSpace(publicBaseURL))
	if err != nil || base.Hostname() == "" {
		return Config{}, fmt.Errorf("public base URL must include a hostname")
	}
	host := strings.ToLower(base.Hostname())
	rpID := strings.ToLower(strings.TrimSpace(os.Getenv("LPBS_WEBAUTHN_RP_ID")))
	if rpID == "" {
		if !production && (host == "localhost" || host == "127.0.0.1") {
			rpID = "localhost"
		} else {
			rpID = host
		}
	}
	if rpID != host && !strings.HasSuffix(host, "."+rpID) {
		return Config{}, fmt.Errorf("webauthn RP ID %q is not valid for origin host %q", rpID, host)
	}
	origins := []string{strings.TrimRight(base.Scheme+"://"+base.Host, "/")}
	for _, extra := range strings.Split(os.Getenv("LPBS_WEBAUTHN_EXTRA_ORIGINS"), ",") {
		extra = strings.TrimRight(strings.TrimSpace(extra), "/")
		if extra == "" {
			continue
		}
		u, parseErr := url.Parse(extra)
		if parseErr != nil || u.Hostname() == "" || (production && u.Scheme != "https") {
			return Config{}, fmt.Errorf("invalid extra WebAuthn origin %q", extra)
		}
		origins = append(origins, extra)
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = "Landing Page Business Suite"
	}
	return Config{RPID: rpID, RPDisplayName: strings.TrimSpace(displayName), RPOrigins: origins}, nil
}
