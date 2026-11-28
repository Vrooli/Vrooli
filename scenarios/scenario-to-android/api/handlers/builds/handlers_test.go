package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"

	buildsdomain "scenario-to-android/internal/builds"
)

func TestNativeProfileReachesBuildOwnerThroughHTTP(t *testing.T) {
	var received deliveryramp.BuildRequest
	handler := GenerateHandler(Surface{Generate: func(_ context.Context, request deliveryramp.BuildRequest) (buildsdomain.GeneratedProject, error) {
		received = request
		return buildsdomain.GeneratedProject{PackageName: "com.example.daily"}, nil
	}})
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest("POST", "/android/generate", strings.NewReader(`{"source_ref":"/source","project_kind":"native","native_profile":"daily"}`)))
	require.Equal(t, 200, response.Code)
	require.Equal(t, "daily", received.Parameters["native_profile"], "profile selection must survive the HTTP boundary instead of silently building the default preview")
	require.Equal(t, "native", received.Parameters["project_kind"])
}

func TestArtifactMetadataAndAPKHandlersUseImmutableOwnerReference(t *testing.T) {
	root, ref, apk := retainedHandlerArtifact(t)
	surface := Surface{ArtifactRoot: func(context.Context) (string, error) { return root, nil }}

	metadataResponse := httptest.NewRecorder()
	metadataRequest := httptest.NewRequest("GET", "/api/v1/android/artifacts/metadata?immutable_ref="+ref, nil)
	ArtifactMetadataHandler(surface)(metadataResponse, metadataRequest)
	require.Equal(t, 200, metadataResponse.Code)
	require.Equal(t, "private, no-store", metadataResponse.Header().Get("Cache-Control"))
	var descriptor map[string]any
	require.NoError(t, json.Unmarshal(metadataResponse.Body.Bytes(), &descriptor))
	require.Equal(t, ref, descriptor["immutable_ref"])
	require.Equal(t, "sha256:"+digestHandlerBytes(apk), descriptor["apk_checksum"])
	require.Equal(t, "com.example.portal", descriptor["package_name"])
	require.Equal(t, "debug-only", descriptor["signing"])
	require.NotContains(t, metadataResponse.Body.String(), "local_path")
	require.NotContains(t, metadataResponse.Body.String(), "apk_path")

	apkResponse := httptest.NewRecorder()
	apkRequest := httptest.NewRequest("GET", "/api/v1/android/artifacts/apk?immutable_ref="+ref, nil)
	APKArtifactHandler(surface)(apkResponse, apkRequest)
	require.Equal(t, 200, apkResponse.Code)
	require.Equal(t, apk, apkResponse.Body.Bytes())
	require.Equal(t, "application/vnd.android.package-archive", apkResponse.Header().Get("Content-Type"))
	require.Equal(t, ref, apkResponse.Header().Get("X-Vrooli-Immutable-Ref"))
	require.Equal(t, "sha256:"+digestHandlerBytes(apk), apkResponse.Header().Get("X-Vrooli-APK-Checksum"))
	require.Equal(t, "com.example.portal", apkResponse.Header().Get("X-Vrooli-Android-Package"))
	require.Equal(t, "43", apkResponse.Header().Get("X-Vrooli-Android-Version-Code"))
}

func TestArtifactListHandlerReturnsPathFreeCandidates(t *testing.T) {
	root, ref, _ := retainedHandlerArtifact(t)
	handler := ArtifactListHandler(Surface{ArtifactRoot: func(context.Context) (string, error) { return root, nil }})
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest("GET", "/artifacts?limit=10", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Body.String(), ref)
	require.NotContains(t, response.Body.String(), root)
	require.NotContains(t, response.Body.String(), "local_path")

	invalid := httptest.NewRecorder()
	handler(invalid, httptest.NewRequest("GET", "/artifacts?limit=101", nil))
	require.Equal(t, http.StatusBadRequest, invalid.Code)
}

func TestArtifactHandlersFailClosedForUnknownOrUnavailableReference(t *testing.T) {
	root, ref, _ := retainedHandlerArtifact(t)
	handler := ArtifactMetadataHandler(Surface{ArtifactRoot: func(context.Context) (string, error) { return root, nil }})
	for _, query := range []string{"", "immutable_ref=../../app.apk", "immutable_ref=android-debug:" + strings.Repeat("0", 64)} {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest("GET", "/metadata?"+query, nil))
		require.NotEqual(t, 200, response.Code)
	}
	unavailable := httptest.NewRecorder()
	APKArtifactHandler(Surface{})(unavailable, httptest.NewRequest("GET", "/apk?immutable_ref="+ref, nil))
	require.Equal(t, 503, unavailable.Code)
}

func TestArtifactHandlersRejectScenarioUIProxyRequests(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"metadata": ArtifactMetadataHandler(Surface{}),
		"apk":      APKArtifactHandler(Surface{}),
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/artifact", nil)
			request.Header.Set(scenarioUIProxyHeader, scenarioUIProxyValue)
			response := httptest.NewRecorder()
			handler(response, request)
			require.Equal(t, 403, response.Code)
			require.Contains(t, response.Body.String(), "only on the local producer API")
		})
	}
}

func retainedHandlerArtifact(t *testing.T) (string, string, []byte) {
	t.Helper()
	sdkRoot := t.TempDir()
	const signerDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	toolPath := filepath.Join(sdkRoot, "build-tools", "35.0.0", "apksigner")
	require.NoError(t, os.MkdirAll(filepath.Join(sdkRoot, "platforms"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(toolPath), 0o755))
	require.NoError(t, os.WriteFile(toolPath, []byte("#!/bin/sh\nprintf '%s\\n' 'Signer #1 certificate SHA-256 digest: "+signerDigest+"'\n"), 0o755))
	t.Setenv("ANDROID_SDK_ROOT", sdkRoot)
	t.Setenv("ANDROID_HOME", "")
	root, source := t.TempDir(), t.TempDir()
	apk, aab := []byte("verified APK bytes"), []byte("verified AAB bytes")
	apkPath, aabPath := filepath.Join(source, "app.apk"), filepath.Join(source, "app.aab")
	require.NoError(t, os.WriteFile(apkPath, apk, 0o600))
	require.NoError(t, os.WriteFile(aabPath, aab, 0o600))
	combined := sha256.New()
	_, _ = combined.Write(apk)
	_, _ = combined.Write([]byte{0})
	_, _ = combined.Write(aab)
	digest := hex.EncodeToString(combined.Sum(nil))
	artifact := deliveryramp.Artifact{
		ImmutableRef: "android-debug:" + digest, Kind: "android-debug-apk-aab", Checksum: "sha256:" + digest,
		SizeBytes: int64(len(apk) + len(aab)), CreatedAt: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
		Metadata: map[string]string{"apk_path": apkPath, "aab_path": aabPath, "package_name": "com.example.portal", "version_name": "0.4.3", "version_code": "43", "signing": "debug-only", "signing_certificate_sha256": signerDigest},
	}
	retained, err := buildsdomain.RetainArtifact(context.Background(), root, artifact)
	require.NoError(t, err)
	return root, retained.ImmutableRef, apk
}

func digestHandlerBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
