package resources

import (
	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
)

const resourceConfigPath = catalogpkg.ResourceConfigPath

type ConfigEntry = catalogpkg.ConfigEntry
type Resource = catalogpkg.Resource

func (c *Controller) catalogService() *catalogpkg.Service {
	return catalogpkg.New(c.Root)
}
