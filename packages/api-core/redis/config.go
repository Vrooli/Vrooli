// Package redis owns resolution of the lifecycle-provided Redis environment.
package redis

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

// Resolve reads REDIS_URL first, then the lifecycle's component exports.
func Resolve(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("redis environment reader is required")
	}
	if raw := strings.TrimSpace(getenv("REDIS_URL")); raw != "" {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return Config{}, fmt.Errorf("parse REDIS_URL: %w", err)
		}
		db, err := databaseIndex(parsed.Path)
		if err != nil {
			return Config{}, err
		}
		password := ""
		if parsed.User != nil {
			password, _ = parsed.User.Password()
		}
		return Config{Addr: parsed.Host, Password: password, DB: db}, nil
	}
	host := strings.TrimSpace(getenv("REDIS_HOST"))
	port := strings.TrimSpace(getenv("REDIS_PORT"))
	if host == "" {
		return Config{}, fmt.Errorf("REDIS_URL or REDIS_HOST must be set by the lifecycle system")
	}
	if port == "" {
		port = "6379"
	}
	db, err := databaseIndex(getenv("REDIS_DB"))
	if err != nil {
		return Config{}, err
	}
	return Config{Addr: host + ":" + port, Password: getenv("REDIS_PASSWORD"), DB: db}, nil
}

func databaseIndex(raw string) (int, error) {
	raw = strings.Trim(raw, "/ ")
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid Redis database index %q", raw)
	}
	return value, nil
}
