package usageforecast

import (
	"os"
	"strings"
	"time"
)

// TimezoneEnv names the IANA zone the digest's 08:00 is local to.
const TimezoneEnv = "AGENT_MANAGER_TIMEZONE"

// DigestLocation is the zone the daily digest is due in: AGENT_MANAGER_TIMEZONE
// when set, otherwise the host's zone as /etc/localtime names it (the service
// runs with TZ unset). An unreadable setting falls back to the process's local
// zone, never UTC.
func DigestLocation() *time.Location { return digestLocation(os.Getenv, "/etc/localtime") }

func digestLocation(getenv func(string) string, localtime string) *time.Location {
	for _, name := range []string{strings.TrimSpace(getenv(TimezoneEnv)), strings.TrimSpace(getenv("TZ")), zoneOfLink(localtime)} {
		if name == "" {
			continue
		}
		if loc, err := time.LoadLocation(strings.TrimPrefix(name, ":")); err == nil {
			return loc
		}
	}
	return time.Local
}

// zoneOfLink is the zone name a /etc/localtime symlink points at
// ("…/zoneinfo/America/New_York" is "America/New_York").
func zoneOfLink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	_, name, found := strings.Cut(target, "zoneinfo/")
	if !found {
		return ""
	}
	return name
}
