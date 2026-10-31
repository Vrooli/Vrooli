package env

import "os"

type Config struct{ DataDir string }

func Load() Config {
	dataDir := os.Getenv("RESOURCE_DATA_DIR")
	if dataDir == "" {
		dataDir = os.Getenv("VROOLI_RESOURCE_DATA_DIR")
	}
	return Config{DataDir: dataDir}
}
