package config

import "os"

type Config struct {
	Endpoint, Model             string
	Port, StartupTimeoutSeconds int
}

func Defaults() Config {
	return Config{Endpoint: "http://127.0.0.1:11460", Model: os.Getenv("PII_DETECTOR_MODEL"), Port: 11460, StartupTimeoutSeconds: 120}
}
