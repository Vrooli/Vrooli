package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestResolveAPKReturnsPathFreeVerifiedDescriptorAndBytes(t *testing.T) {
	dataRoot, ref, wantAPK := retainResolverFixture(t)
	resolved, err := ResolveAPK(dataRoot, ref)
	require.NoError(t, err)
	defer resolved.File.Close()
	require.Equal(t, ref, resolved.Descriptor.ImmutableRef)
	require.Equal(t, "sha256:"+digestBytes(wantAPK), resolved.Descriptor.APKChecksum)
	require.Equal(t, "com.example.portal", resolved.Descriptor.PackageName)
	require.Equal(t, "0.4.3", resolved.Descriptor.VersionName)
	require.EqualValues(t, 43, resolved.Descriptor.VersionCode)
	require.Equal(t, "debug-only", resolved.Descriptor.Signing)
	require.NotContains(t, resolved.Descriptor.PackageName, filepath.Base(dataRoot))
	got, err := os.ReadFile(resolved.File.Name())
	require.NoError(t, err)
	require.Equal(t, wantAPK, got)
}

func TestResolveAPKRejectsInvalidReferenceTamperingAndSymlinks(t *testing.T) {
	t.Run("reference cannot select arbitrary paths", func(t *testing.T) {
		if _, err := ResolveAPK(t.TempDir(), "../../app.apk"); err == nil {
			t.Fatal("accepted non-immutable reference")
		}
	})
	t.Run("signer metadata must match the retained APK", func(t *testing.T) {
		dataRoot, ref, _ := retainResolverFixture(t)
		manifestPath := filepath.Join(dataRoot, "builds", "artifacts", ref[len("android-debug:"):], "artifact.json")
		data, err := os.ReadFile(manifestPath)
		require.NoError(t, err)
		data = []byte(strings.Replace(string(data), strings.Repeat("b", 64), strings.Repeat("c", 64), 1))
		require.NoError(t, os.WriteFile(manifestPath, data, 0o600))
		if _, err := ResolveAPK(dataRoot, ref); err == nil {
			t.Fatal("accepted signer metadata that differs from the retained APK")
		}
	})
	t.Run("manifest reference must match request", func(t *testing.T) {
		dataRoot, ref, _ := retainResolverFixture(t)
		manifestPath := filepath.Join(dataRoot, "builds", "artifacts", ref[len("android-debug:"):], "artifact.json")
		var artifact deliveryramp.Artifact
		data, err := os.ReadFile(manifestPath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &artifact))
		artifact.ImmutableRef = "android-release:" + ref[len("android-debug:"):]
		data, err = json.Marshal(artifact)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(manifestPath, data, 0o600))
		if _, err := ResolveAPK(dataRoot, ref); err == nil {
			t.Fatal("accepted mismatched retained identity")
		}
	})
	t.Run("tampered bytes fail digest verification", func(t *testing.T) {
		dataRoot, ref, _ := retainResolverFixture(t)
		apkPath := filepath.Join(dataRoot, "builds", "artifacts", ref[len("android-debug:"):], "app.apk")
		require.NoError(t, os.WriteFile(apkPath, []byte("changed APK"), 0o600))
		if _, err := ResolveAPK(dataRoot, ref); err == nil {
			t.Fatal("accepted tampered APK")
		}
	})
	t.Run("symlinked AAB is refused", func(t *testing.T) {
		dataRoot, ref, _ := retainResolverFixture(t)
		dir := filepath.Join(dataRoot, "builds", "artifacts", ref[len("android-debug:"):])
		aabPath := filepath.Join(dir, "app.aab")
		require.NoError(t, os.Remove(aabPath))
		target := filepath.Join(t.TempDir(), "outside.aab")
		require.NoError(t, os.WriteFile(target, []byte("outside"), 0o600))
		require.NoError(t, os.Symlink(target, aabPath))
		if _, err := ResolveAPK(dataRoot, ref); err == nil {
			t.Fatal("accepted symlinked AAB")
		}
	})
}

func retainResolverFixture(t *testing.T) (string, string, []byte) {
	t.Helper()
	const signerDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	useFakeAndroidSDK(t, signerDigest)
	dataRoot, source := t.TempDir(), t.TempDir()
	apkBytes, aabBytes := []byte("portal APK payload"), []byte("portal AAB payload")
	apkPath, aabPath := filepath.Join(source, "app.apk"), filepath.Join(source, "app.aab")
	require.NoError(t, os.WriteFile(apkPath, apkBytes, 0o600))
	require.NoError(t, os.WriteFile(aabPath, aabBytes, 0o600))
	hash := sha256.New()
	_, _ = hash.Write(apkBytes)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(aabBytes)
	digest := hex.EncodeToString(hash.Sum(nil))
	artifact := deliveryramp.Artifact{
		ImmutableRef: "android-debug:" + digest,
		Kind:         "android-debug-apk-aab",
		Checksum:     "sha256:" + digest,
		SizeBytes:    int64(len(apkBytes) + len(aabBytes)),
		CreatedAt:    time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC),
		Metadata: map[string]string{
			"apk_path": apkPath, "aab_path": aabPath,
			"package_name": "com.example.portal", "version_name": "0.4.3", "version_code": "43", "signing": "debug-only",
			"signing_certificate_sha256": signerDigest,
		},
	}
	retained, err := RetainArtifact(context.Background(), dataRoot, artifact)
	require.NoError(t, err)
	return dataRoot, retained.ImmutableRef, apkBytes
}

func digestBytes(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
