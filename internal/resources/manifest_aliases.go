package resources

import manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"

const resourceManifestSchemaPath = manifestpkg.SchemaPath

type ResourceManifest = manifestpkg.ResourceManifest
type ResourceLegacyAdapter = manifestpkg.ResourceLegacyAdapter
type ResourcePlatforms = manifestpkg.ResourcePlatforms
type ResourcePort = manifestpkg.ResourcePort
type ResourceHealthCheck = manifestpkg.ResourceHealthCheck
type ResourceInstall = manifestpkg.ResourceInstall
type ResourceCredentials = manifestpkg.ResourceCredentials
type ResourceRuntime = manifestpkg.ResourceRuntime
type ResourceVolume = manifestpkg.ResourceVolume
type ResourceLifecycle = manifestpkg.ResourceLifecycle
type ResourceManifestCapabilities = manifestpkg.ResourceManifestCapabilities

var allowedResourceDrivers = manifestpkg.AllowedDrivers
var allowedPortabilityTiers = manifestpkg.AllowedPortabilityTiers
var allowedPlatformSupportStates = manifestpkg.AllowedPlatformSupportStates

func (c *Controller) loadResourceManifest(path string) (ResourceManifest, error) {
	return manifestpkg.Load(path)
}

func (c *Controller) LoadManifest(path string) (ResourceManifest, error) {
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
