package config

import (
	"os"
	"strconv"
)

type Config struct {
	APIHost, DataDir, ConfigDir, Region, RootUser, RootPassword string
	APIPort, ConsolePort, StartupTimeoutSeconds                 int
}

func Defaults() Config {
	return Config{APIHost: "127.0.0.1", DataDir: os.Getenv("RESOURCE_DATA_DIR"), ConfigDir: os.Getenv("RESOURCE_CONFIG_DIR"), Region: "us-east-1", RootUser: "minioadmin", RootPassword: "minioadmin", APIPort: 9000, ConsolePort: 9001, StartupTimeoutSeconds: 60}
}

// Load preserves operator-provided paths and credentials during migration
// from the retired shell defaults. Empty values fall back to typed defaults;
// existing non-empty values are never overwritten.
func Load(getenv func(string) string) Config {
	cfg := Defaults()
	if value := getenv("MINIO_HOST"); value != "" {
		cfg.APIHost = value
	}
	if value := getenv("RESOURCE_DATA_DIR"); value != "" {
		cfg.DataDir = value
	}
	if value := getenv("RESOURCE_CONFIG_DIR"); value != "" {
		cfg.ConfigDir = value
	}
	if value := getenv("MINIO_ROOT_USER"); value != "" {
		cfg.RootUser = value
	}
	if value := getenv("MINIO_ROOT_PASSWORD"); value != "" {
		cfg.RootPassword = value
	}
	if value, err := strconv.Atoi(getenv("RESOURCE_PORT_API")); err == nil && value > 0 {
		cfg.APIPort = value
	}
	if value, err := strconv.Atoi(getenv("RESOURCE_PORT_CONSOLE")); err == nil && value > 0 {
		cfg.ConsolePort = value
	}
	return cfg
}
