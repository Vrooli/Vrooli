package builds

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestNativeProfilesIsolateDailyAndPreviewWithoutChangingSource(t *testing.T) {
	source := nativeFixture(t)
	original := []byte(`{"package_name":"com.example.daily","namespace":"com.example.code","app_name":"Daily","target_sdk":36,"version_code":8,"version_name":"0.3","custom_gradle_setting":"preserve","default_profile":"preview","profiles":{"preview":{"package_name":"com.example.preview","app_name":"Preview","home_enabled":false},"daily":{"package_name":"com.example.daily","app_name":"Daily","home_enabled":true}}}`)
	require.NoError(t, os.WriteFile(filepath.Join(source, "vrooli-native.json"), original, 0o644))
	builder := Builder{BuildRoot: t.TempDir()}
	digests := map[string]bool{}
	for _, candidate := range []struct {
		profile, pkg string
		home         bool
	}{{"", "com.example.preview", false}, {"daily", "com.example.daily", true}} {
		project, err := builder.Generate(context.Background(), deliveryramp.BuildRequest{SourceRef: source, Parameters: map[string]string{"project_kind": "native", "native_profile": candidate.profile}})
		require.NoError(t, err)
		require.Equal(t, candidate.pkg, project.PackageName)
		require.Len(t, project.SourceDigest, len("sha256:")+64)
		require.False(t, digests[project.SourceDigest], "distinct native profiles must retain distinct source/config identities")
		digests[project.SourceDigest] = true
		data, err := os.ReadFile(filepath.Join(project.ProjectPath, "vrooli-native.json"))
		require.NoError(t, err)
		var effective map[string]any
		require.NoError(t, json.Unmarshal(data, &effective))
		require.Equal(t, candidate.pkg, effective["package_name"])
		require.Equal(t, candidate.home, effective["home_enabled"])
		require.Equal(t, "com.example.code", effective["namespace"])
		require.Equal(t, "preserve", effective["custom_gradle_setting"])
	}
	unchanged, err := os.ReadFile(filepath.Join(source, "vrooli-native.json"))
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
	_, err = builder.Generate(context.Background(), deliveryramp.BuildRequest{SourceRef: source, Parameters: map[string]string{"project_kind": "native", "native_profile": "missing"}})
	require.ErrorContains(t, err, "unknown native profile")
	_, err = builder.Generate(context.Background(), deliveryramp.BuildRequest{SourceRef: source, Parameters: map[string]string{"project_kind": "native", "native_profile": "daily", "package_name": "com.example.preview"}})
	require.ErrorContains(t, err, "conflicts")
}

func nativeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range map[string]string{
		"vrooli-native.json": `{"package_name":"com.example.native","app_name":"Native","target_sdk":36,"version_code":1,"version_name":"1.0"}`,
		"settings.gradle":    "native settings", "build.gradle": "native build", "app/build.gradle": "native app", "app/src/main/AndroidManifest.xml": "<manifest/>",
	} {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return root
}

func TestNativePreparationPreservesSourceAndUsesIsolatedProject(t *testing.T) {
	source := nativeFixture(t)
	request := deliveryramp.BuildRequest{SourceRef: source, Parameters: map[string]string{"project_kind": "native", "preview_cleartext_host": "192.168.1.173"}}
	builder := Builder{BuildRoot: t.TempDir()}
	first, err := builder.Generate(context.Background(), request)
	require.NoError(t, err)
	second, err := builder.Generate(context.Background(), request)
	require.NoError(t, err)
	require.NotEqual(t, first.ProjectPath, second.ProjectPath)
	require.Equal(t, "com.example.native", first.PackageName)
	data, err := os.ReadFile(filepath.Join(first.ProjectPath, "app/build.gradle"))
	require.NoError(t, err)
	require.Equal(t, "native app", string(data), "native code must not be replaced by the web wrapper")
	policy, err := os.ReadFile(filepath.Join(first.ProjectPath, "app/src/debug/res/xml/network_security_config.xml"))
	require.NoError(t, err)
	require.Contains(t, string(policy), `<base-config cleartextTrafficPermitted="false"/>`)
	require.Contains(t, string(policy), `<domain includeSubdomains="false">192.168.1.173</domain>`)
	_, err = os.Stat(filepath.Join(source, "app/src/debug"))
	require.True(t, os.IsNotExist(err))
}

func TestNativePreparationRejectsConflictsAndUnscopedCleartext(t *testing.T) {
	for _, parameters := range []map[string]string{
		{"project_kind": "native", "package_name": "com.example.other"},
		{"project_kind": "native", "preview_cleartext_host": "example.com"},
		{"project_kind": "native", "preview_cleartext_host": "8.8.8.8"},
		{"project_kind": "native", "preview_cleartext_host": "192.168.1.173", "signing": "required"},
	} {
		_, err := (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), deliveryramp.BuildRequest{SourceRef: nativeFixture(t), Parameters: parameters})
		require.Error(t, err)
	}
}

func TestNativePreparationRejectsSymlinkInputs(t *testing.T) {
	source := nativeFixture(t)
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(source, "outside")))
	_, err := (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), deliveryramp.BuildRequest{SourceRef: source, Parameters: map[string]string{"project_kind": "native"}})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "symlink"))
}

func TestNativeLoopbackPolicyIsDebugOnlyAndDomainScoped(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		request := deliveryramp.BuildRequest{SourceRef: nativeFixture(t), Parameters: map[string]string{"project_kind": "native", "preview_cleartext_host": host}}
		project, err := (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), request)
		require.NoError(t, err)
		policy, err := os.ReadFile(filepath.Join(project.ProjectPath, "app/src/debug/res/xml/network_security_config.xml"))
		require.NoError(t, err)
		require.Contains(t, string(policy), `<base-config cleartextTrafficPermitted="false"/>`)
		require.Contains(t, string(policy), `<domain includeSubdomains="false">`+host+`</domain>`)
		request.Parameters["signing"] = "required"
		_, err = (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), request)
		require.Error(t, err)
	}
}
