package appctx

import (
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// EnvReader is the ambient CLI environment seam used to resolve optional
// bearer tokens. Tests inject a map-backed reader; production uses OSEnv.
// seam: EnvReader
type EnvReader interface {
	LookupEnv(string) (string, bool)
}

type OSEnv struct{}

func (OSEnv) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

type Context struct {
	Name         string
	Core         *cliapp.ScenarioApp
	ScenarioRoot string
	TokenEnvVars []string
	Environment  EnvReader
}

func (c *Context) Token() string {
	environment := c.Environment
	if environment == nil {
		environment = OSEnv{}
	}
	for _, env := range c.TokenEnvVars {
		if raw, ok := environment.LookupEnv(env); ok {
			if value := strings.TrimSpace(raw); value != "" {
				return value
			}
		}
	}
	if c.Core == nil {
		return ""
	}
	return c.Core.Config.Token
}
