package config

import "os"

type Config struct {
	Image, ContainerPrefix, Network, DataRoot, StateRoot string
	DefaultPort, MaxInstances, HealthTimeoutSeconds      int
}

func Defaults() Config {
	return Config{Image: "postgres@sha256:57c72fd2a128e416c7fcc499958864df5301e940bca0a56f58fddf30ffc07777", ContainerPrefix: "vrooli-postgres", Network: "vrooli-network", DataRoot: os.Getenv("RESOURCE_DATA_DIR"), StateRoot: os.Getenv("RESOURCE_STATE_DIR"), DefaultPort: 5433, MaxInstances: 67, HealthTimeoutSeconds: 5}
}
