package main

import (
	"context"
	"net/http"

	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/delivery"
)

func manifestFilenameToPlatform(v string) string { return downloadhttp.ManifestFilenameToPlatform(v) }
func channelToVariantKey(v string) string        { return downloadhttp.ChannelToVariantKey(v) }
func buildElectronManifest(a *delivery.Artifact, notes string) []byte {
	return downloadhttp.BuildElectronManifest(a, notes)
}

type updateAppLookup interface {
	GetApp(bundleKey, appKey string) (*delivery.App, error)
}
type updateAssetLookup interface {
	GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*delivery.Asset, error)
}
type updateArtifactResolver interface {
	GetArtifact(context.Context, string, int64) (*delivery.Artifact, error)
	GetCurrentArtifactByFilename(context.Context, string, string, string, string) (*delivery.Artifact, error)
	PresignGetArtifact(context.Context, string, delivery.Artifact) (string, error)
}
type updateVerifyArtifactResolver interface {
	GetArtifact(context.Context, string, int64) (*delivery.Artifact, error)
	PresignGetArtifact(context.Context, string, delivery.Artifact) (string, error)
	HeadArtifact(context.Context, string, delivery.Artifact) error
}
type channelDiscoveryLookup interface {
	ListChannels(bundleKey, appKey string) ([]delivery.ChannelInfo, error)
}
type updatePolicyLookup interface {
	GetApp(bundleKey, appKey string) (*delivery.App, error)
	UpdateAppPolicy(bundleKey, appKey string, policy delivery.UpdatePolicy) error
}
type updateBundleKeyProvider interface{ BundleKey() string }

func testUpdateDependencies(bundles updateBundleKeyProvider) downloadhttp.UpdateDependencies {
	return updateDependencies(bundles)
}

func requireUpdateAPIKey(apps updateAppLookup, bundles updateBundleKeyProvider) func(http.HandlerFunc) http.HandlerFunc {
	return downloadhttp.RequireUpdateAPIKey(testUpdateDependencies(bundles), apps)
}

func handleUpdateFile(assets updateAssetLookup, artifacts updateArtifactResolver, bundles updateBundleKeyProvider) http.HandlerFunc {
	return downloadhttp.UpdateFile(testUpdateDependencies(bundles), assets, artifacts)
}

func handleChannelDiscovery(channels channelDiscoveryLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return downloadhttp.ChannelDiscovery(testUpdateDependencies(bundles), channels)
}

func handleUpdateVerify(assets updateAssetLookup, artifacts updateVerifyArtifactResolver, bundles updateBundleKeyProvider) http.HandlerFunc {
	return downloadhttp.VerifyUpdate(testUpdateDependencies(bundles), assets, artifacts)
}

func handleGetUpdatePolicy(apps updatePolicyLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return downloadhttp.GetUpdatePolicy(testUpdateDependencies(bundles), apps)
}

func handlePutUpdatePolicy(apps updatePolicyLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return downloadhttp.PutUpdatePolicy(testUpdateDependencies(bundles), apps)
}
