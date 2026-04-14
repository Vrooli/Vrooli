package resources

import manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"

const resourceManifestSchemaPath = manifestpkg.SchemaPath

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

var (
	allowedResourceDrivers       = manifestpkg.AllowedDrivers
	allowedPortabilityTiers      = manifestpkg.AllowedPortabilityTiers
	allowedPlatformSupportStates = manifestpkg.AllowedPlatformSupportStates
)

func (c *Controller) loadResourceManifest(path string) (ResourceManifest, error) {
	return manifestpkg.Load(path)
}

func validateResourceManifest(manifest ResourceManifest) error {
	return manifestpkg.Validate(manifest)
}

func defaultResourceManifestPath(root, name string) string {
	return manifestpkg.DefaultPath(root, name)
}

func currentResourcePlatform() string {
	return manifestpkg.CurrentPlatform()
}
