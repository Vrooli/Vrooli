package resources

import manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"

type (
	ResourceManifest             = manifestpkg.ResourceManifest
	ResourcePlatforms            = manifestpkg.ResourcePlatforms
	ResourcePort                 = manifestpkg.ResourcePort
	ResourceHealthCheck          = manifestpkg.ResourceHealthCheck
	ResourceInstall              = manifestpkg.ResourceInstall
	ResourceCredentials          = manifestpkg.ResourceCredentials
	ResourceRuntime              = manifestpkg.ResourceRuntime
	ResourceEnvironmentExports   = manifestpkg.ResourceEnvironmentExports
	ResourceOrchestration        = manifestpkg.ResourceOrchestration
	ResourceDerivedTemplate      = manifestpkg.ResourceDerivedTemplate
	ResourceVolume               = manifestpkg.ResourceVolume
	ResourceLifecycle            = manifestpkg.ResourceLifecycle
	ResourceManifestCapabilities = manifestpkg.ResourceManifestCapabilities
)

func (c *Controller) loadResourceManifest(path string) (ResourceManifest, error) {
	return manifestpkg.Load(path)
}

// ResourceManifest loads a resource's manifest by name from the controller's
// root. Exposed as a public seam so callers (e.g. the scenario orchestrator)
// can introspect declared capabilities without reaching into the controller's
// internals.
func (c *Controller) ResourceManifest(name string) (ResourceManifest, error) {
	return c.loadResourceManifest(defaultResourceManifestPath(c.Root, name))
}

func defaultResourceManifestPath(root, name string) string {
	return manifestpkg.DefaultPath(root, name)
}
