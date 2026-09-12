package appctx

import (
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type Context struct {
	Name         string
	Core         *cliapp.ScenarioApp
	ScenarioRoot string
	TokenEnvVars []string
}

func (c *Context) Token() string {
	for _, env := range c.TokenEnvVars {
		if value := strings.TrimSpace(os.Getenv(env)); value != "" {
			return value
		}
	}
	if c.Core == nil {
		return ""
	}
	return c.Core.Config.Token
}
