package androidartifact

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

	"github.com/stretchr/testify/require"
)

func TestClientResolveStagesOnlyMatchingVerifiedAPK(t *testing.T) {
	apk := []byte("verified APK fixture")
	descriptor := fixtureDescriptor(apk)
	server := artifactServer(t, descriptor, apk)
	client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }}
	client.Inspector = fixtureInspector{identity: identityFor(descriptor)}

	staged, err := client.Resolve(context.Background(), descriptor.ImmutableRef, descriptor.PackageName, descriptor.VersionName)
	require.NoError(t, err)
	defer staged.Close()
	require.Equal(t, descriptor, staged.Descriptor)
	got, err := os.ReadFile(staged.Path)
	require.NoError(t, err)
	require.Equal(t, apk, got)
}

func TestClientResolveRejectsMismatchedIdentityAndTamperedBytes(t *testing.T) {
	apk := []byte("verified APK fixture")
	descriptor := fixtureDescriptor(apk)
	server := artifactServer(t, descriptor, []byte("tampered APK fixture"))
	client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }}
	client.Inspector = fixtureInspector{identity: identityFor(descriptor)}

	_, err := client.Resolve(context.Background(), descriptor.ImmutableRef, "com.example.other", descriptor.VersionName)
	require.ErrorContains(t, err, "descriptor does not match")

	_, err = client.Resolve(context.Background(), descriptor.ImmutableRef, descriptor.PackageName, descriptor.VersionName)
	require.ErrorContains(t, err, "failed checksum or size")
}

func TestClientResolveRejectsInvalidReferenceAndMetadata(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }}

	_, err := client.Resolve(context.Background(), "../../outside.apk", "com.example.portal", "1.0")
	require.Error(t, err)
	require.Zero(t, calls, "invalid refs must fail before contacting the producer")

	validRef := "android-debug:" + strings.Repeat("a", 64)
	_, err = client.Resolve(context.Background(), validRef, "com.example.portal", "1.0")
	require.ErrorContains(t, err, "descriptor does not match")
}

func TestClientResolveRequiresCombinedChecksumToMatchImmutableRef(t *testing.T) {
	apk := []byte("verified APK fixture")
	descriptor := fixtureDescriptor(apk)
	descriptor.Checksum = "sha256:" + strings.Repeat("b", 64)
	var apkRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/android/artifacts/apk" {
			apkRequests++
			_, _ = w.Write(apk)
			return
		}
		_ = json.NewEncoder(w).Encode(descriptor)
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }}

	_, err := client.Resolve(context.Background(), descriptor.ImmutableRef, descriptor.PackageName, descriptor.VersionName)
	require.ErrorContains(t, err, "descriptor does not match")
	require.Zero(t, apkRequests, "do not download bytes when the descriptor breaks immutable identity")
}

func TestClientResolveCleansStagedFileOnClose(t *testing.T) {
	apk := []byte("verified APK fixture")
	descriptor := fixtureDescriptor(apk)
	server := artifactServer(t, descriptor, apk)
	client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }}
	client.Inspector = fixtureInspector{identity: identityFor(descriptor)}

	staged, err := client.Resolve(context.Background(), descriptor.ImmutableRef, descriptor.PackageName, "")
	require.NoError(t, err)
	require.FileExists(t, staged.Path)
	require.NoError(t, staged.Close())
	require.NoFileExists(t, staged.Path)
}

func TestClientResolveRejectsAPKManifestIdentityBeforeReturningInstallableFile(t *testing.T) {
	apk := []byte("APK bytes whose manifest is inspected")
	descriptor := fixtureDescriptor(apk)
	server := artifactServer(t, descriptor, apk)
	for _, mutate := range []func(*APKIdentity){
		func(identity *APKIdentity) { identity.PackageName = "com.example.unexpected" },
		func(identity *APKIdentity) { identity.VersionName = "9.9" },
		func(identity *APKIdentity) { identity.VersionCode = 99 },
	} {
		identity := identityFor(descriptor)
		mutate(&identity)
		client := Client{HTTP: server.Client(), BaseURL: func(context.Context) (string, error) { return server.URL, nil }, Inspector: fixtureInspector{identity: identity}}
		_, err := client.Resolve(context.Background(), descriptor.ImmutableRef, descriptor.PackageName, descriptor.VersionName)
		require.ErrorContains(t, err, "manifest identity does not match")
	}
}

type fixtureInspector struct{ identity APKIdentity }

func (f fixtureInspector) Inspect(context.Context, string) (APKIdentity, error) {
	return f.identity, nil
}

func identityFor(descriptor Descriptor) APKIdentity {
	return APKIdentity{PackageName: descriptor.PackageName, VersionName: descriptor.VersionName, VersionCode: descriptor.VersionCode}
}

func TestAAPTInspectorReadsCompiledManifestIdentity(t *testing.T) {
	sdkRoot := t.TempDir()
	adbPath := filepath.Join(sdkRoot, "platform-tools", "adb")
	require.NoError(t, os.MkdirAll(filepath.Dir(adbPath), 0o700))
	require.NoError(t, os.WriteFile(adbPath, []byte("stub"), 0o700))
	buildTools := filepath.Join(sdkRoot, "build-tools", "35.0.0")
	require.NoError(t, os.MkdirAll(buildTools, 0o700))
	aaptPath := filepath.Join(buildTools, "aapt")
	require.NoError(t, os.WriteFile(aaptPath, []byte("#!/bin/sh\nprintf \"package: name='com.example.capture' versionCode='7' versionName='2.1.0'\\n\"\n"), 0o700))
	require.NoError(t, os.Chmod(aaptPath, 0o700))

	identity, err := (AAPTInspector{ADBPath: adbPath}).Inspect(context.Background(), "fixture.apk")
	require.NoError(t, err)
	require.Equal(t, APKIdentity{PackageName: "com.example.capture", VersionName: "2.1.0", VersionCode: 7}, identity)
}

func fixtureDescriptor(apk []byte) Descriptor {
	digest := sha256.Sum256(apk)
	return Descriptor{
		ImmutableRef: "android-debug:" + strings.Repeat("a", 64),
		Kind:         "android-debug-apk-aab",
		Checksum:     "sha256:" + strings.Repeat("a", 64),
		APKChecksum:  "sha256:" + hex.EncodeToString(digest[:]),
		SizeBytes:    int64(len(apk) + 3), APKSizeBytes: int64(len(apk)),
		PackageName: "com.example.portal", VersionName: "1.0", VersionCode: 1,
	}
}

func artifactServer(t *testing.T, descriptor Descriptor, apk []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/android/artifacts/metadata":
			_ = json.NewEncoder(w).Encode(descriptor)
		case "/api/v1/android/artifacts/apk":
			_, _ = w.Write(apk)
		default:
			http.NotFound(w, r)
		}
	}))
}
