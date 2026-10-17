package config

import "os"

type Config struct {
	Host, DataDir                             string
	HTTPPort, GRPCPort, StartupTimeoutSeconds int
}

func Defaults() Config {
	return Config{Host: "127.0.0.1", DataDir: os.Getenv("RESOURCE_DATA_DIR"), HTTPPort: 6333, GRPCPort: 6334, StartupTimeoutSeconds: 60}
}
