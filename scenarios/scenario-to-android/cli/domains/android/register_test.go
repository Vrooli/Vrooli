package android

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRequestCarriesNativeProfileAndSigningIdentity(t *testing.T) {
	t.Setenv("ANDROID_SOURCE_REF", "/native/source")
	t.Setenv("ANDROID_PROJECT_KIND", "native")
	t.Setenv("ANDROID_NATIVE_PROFILE", "daily")
	t.Setenv("ANDROID_SIGNING_IDENTITY", "vrooli/portal/tv-home-signing-key")
	request, err := buildRequest()
	require.NoError(t, err)
	require.Equal(t, "daily", request["native_profile"])
	require.Equal(t, "native", request["project_kind"])
	require.Equal(t, "vrooli/portal/tv-home-signing-key", request["signing_identity"])
}

func TestSelectPhysicalTargetPrefersAvailablePhysicalTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": true}, Kind: "local"},
		{Descriptor: map[string]any{"target_id": "android-phone", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	target, err := selectPhysicalTarget(body, "")
	require.NoError(t, err)
	require.Equal(t, "android-phone", target["descriptor"].(map[string]any)["target_id"])
}

func TestSelectPhysicalTargetPreservesExplicitUnavailableTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android-phone", "available": false, "reason": "locked"}, Kind: "local"},
	}})
	require.NoError(t, err)

	target, err := selectPhysicalTarget(body, "android-phone")
	require.NoError(t, err)
	require.Equal(t, false, target["descriptor"].(map[string]any)["available"])
}

func TestSelectPhysicalTargetRejectsExplicitEmulator(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "android:emulator:local")
	require.ErrorContains(t, err, "requires a physical target")
}

func TestSelectPhysicalTargetFailsWithoutPhysicalTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": false}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "")
	require.ErrorContains(t, err, "no available physical Android target")
}

func TestSelectPhysicalTargetRequiresExplicitSelectionWhenAmbiguous(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android-phone-1", "available": true}, Kind: "local"},
		{Descriptor: map[string]any{"target_id": "android-phone-2", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "")
	require.ErrorContains(t, err, "multiple physical Android targets")
}

func TestMatrixSelectionCarriesBuiltPackageName(t *testing.T) {
	request := matrixSelectionRequest("sha256:artifact", "/tmp/app-debug.apk", "com.example.custom")
	metadata, ok := request["metadata"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "com.example.custom", metadata["package_name"])
}

func TestMatrixSelectionUsesBuilderPackageDefault(t *testing.T) {
	request := matrixSelectionRequest("sha256:artifact", "/tmp/app-debug.apk", "")
	metadata, ok := request["metadata"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, defaultAndroidPackageName, metadata["package_name"])
}
