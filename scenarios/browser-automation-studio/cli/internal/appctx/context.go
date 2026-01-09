package appctx

import (
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Context struct {
	Name         string
	Version      string
	Core         *cliapp.ScenarioApp
	ScenarioRoot string
	TokenEnvVars []string
}

func (c *Context) APIPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(c.Core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

func (c *Context) APIV1Base() string {
	base := strings.TrimRight(strings.TrimSpace(c.Core.HTTPClient.BaseURL()), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/api/v1") {
		return base
	}
	return base + "/api/v1"
}

func (c *Context) APIRoot() string {
	base := strings.TrimRight(strings.TrimSpace(c.Core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return strings.TrimSuffix(base, "/api/v1")
	}
	return base
}

func (c *Context) ResolvedAPIV1Base() string {
	if c == nil || c.Core == nil {
		return ""
	}
	if base := strings.TrimSpace(c.APIV1Base()); base != "" {
		return base
	}
	base := cliutil.DetermineAPIBase(c.Core.APIBaseOptions())
	if strings.TrimSpace(base) == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/api/v1"
}

func (c *Context) ResolvedAPIRoot() string {
	base := strings.TrimRight(strings.TrimSpace(c.ResolvedAPIV1Base()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return strings.TrimSuffix(base, "/api/v1")
	}
	return base
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
