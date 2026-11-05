package env

// Config is the default home for environment/config helpers for native-cli
// resources.
type Config struct{}

// Load returns the default resource config placeholder.
func Load() Config { return Config{} }
