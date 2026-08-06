package config

import "os"

type Config struct {
	ContainerName, BaseURL, TimeZone                  string
	Port, InstallTimeoutSeconds, HealthTimeoutSeconds int
}

func Defaults() Config {
	return Config{ContainerName: "home-assistant", BaseURL: "http://localhost:8123", TimeZone: "America/New_York", Port: 8123, InstallTimeoutSeconds: 300, HealthTimeoutSeconds: 60}
}

func (c Config) WithEnvironment() Config {
	if v := os.Getenv("HOME_ASSISTANT_CONTAINER_NAME"); v != "" {
		c.ContainerName = v
	}
	if v := os.Getenv("HOME_ASSISTANT_TIME_ZONE"); v != "" {
		c.TimeZone = v
	}
	return c
}
