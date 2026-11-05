package isolation

import (
	"os"
)

// ApplyEnv sets the provided environment variables on the current process and
// returns a function to restore the previous values. This is used to ensure
// vrooli lifecycle commands inherit the temporary Playbooks isolation settings.
func ApplyEnv(vars map[string]string) func() {
	previous := make(map[string]*string, len(vars))
	for k, v := range vars {
		if existing, ok := os.LookupEnv(k); ok {
			previous[k] = &existing
		} else {
			previous[k] = nil
		}
		_ = os.Setenv(k, v)
	}

	return func() {
		for k, v := range previous {
			if v == nil {
				_ = os.Unsetenv(k)
				continue
			}
			_ = os.Setenv(k, *v)
		}
	}
}
