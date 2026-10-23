package builds

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// Metadata is checked in beside the source and is also read by Gradle.
type nativeMetadata struct {
	PackageName    string                   `json:"package_name"`
	AppName        string                   `json:"app_name"`
	TargetSDK      int                      `json:"target_sdk"`
	VersionCode    int                      `json:"version_code"`
	VersionName    string                   `json:"version_name"`
	Namespace      string                   `json:"namespace"`
	DefaultProfile string                   `json:"default_profile"`
	Profiles       map[string]nativeProfile `json:"profiles"`
}

type nativeProfile struct {
	PackageName string `json:"package_name"`
	AppName     string `json:"app_name"`
	HomeEnabled bool   `json:"home_enabled"`
}

// Native projects share the existing build, signing, and artifact path.
func (b Builder) prepareNativeProject(ctx context.Context, request deliveryramp.BuildRequest) (GeneratedProject, projectConfig, error) {
	data, err := os.ReadFile(filepath.Join(request.SourceRef, "vrooli-native.json"))
	if err != nil || len(data) > 16*1024 {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("native project requires bounded vrooli-native.json metadata")
	}
	var metadata nativeMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("invalid native project metadata: %w", err)
	}
	parameters := cloneParameters(request.Parameters)
	profileName := parameters["native_profile"]
	if profileName == "" {
		profileName = metadata.DefaultProfile
	}
	var stagedMetadata []byte
	if len(metadata.Profiles) > 8 || (metadata.Namespace != "" && !packagePattern.MatchString(metadata.Namespace)) {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("invalid native profile metadata")
	}
	if profileName != "" {
		profile, exists := metadata.Profiles[profileName]
		if !exists {
			return GeneratedProject{}, projectConfig{}, fmt.Errorf("unknown native profile %q", profileName)
		}
		if profile.PackageName == "" || profile.AppName == "" {
			return GeneratedProject{}, projectConfig{}, fmt.Errorf("native profile requires package_name and app_name")
		}
		metadata.PackageName, metadata.AppName = profile.PackageName, profile.AppName
		// Preserve custom Gradle metadata while projecting only the selected
		// profile into the isolated snapshot. Never mutate the source project.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
		for key, value := range map[string]any{"package_name": profile.PackageName, "app_name": profile.AppName, "home_enabled": profile.HomeEnabled, "selected_profile": profileName} {
			raw[key], _ = json.Marshal(value)
		}
		stagedMetadata, err = json.Marshal(raw)
		if err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
	}
	for key, value := range map[string]string{"package_name": metadata.PackageName, "app_name": metadata.AppName, "target_sdk": strconv.Itoa(metadata.TargetSDK), "version_code": strconv.Itoa(metadata.VersionCode), "version_name": metadata.VersionName} {
		if supplied := parameters[key]; supplied != "" && supplied != value {
			return GeneratedProject{}, projectConfig{}, fmt.Errorf("native project metadata conflicts with %s", key)
		}
		parameters[key] = value
	}
	config, err := parseConfig(deliveryramp.BuildRequest{Parameters: parameters})
	if err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	config.BrandSource = "native-project"
	config.BrandName = metadata.AppName
	config.NativeProfile = profileName
	for _, required := range []string{"settings.gradle", "build.gradle", "app/build.gradle", "app/src/main/AndroidManifest.xml"} {
		info, err := os.Lstat(filepath.Join(request.SourceRef, required))
		if err != nil || !info.Mode().IsRegular() {
			return GeneratedProject{}, projectConfig{}, fmt.Errorf("native project requires regular %s", required)
		}
	}
	root := b.BuildRoot
	if root == "" {
		root = filepath.Join(os.TempDir(), "vrooli-android-builds")
	}
	// Never build source in place or copy mutable outputs and symlinks.
	files := map[string][]byte{}
	hasher := sha256.New()
	totalBytes := 0
	err = filepath.WalkDir(request.SourceRef, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("native project symlinks are not supported")
		}
		if entry.IsDir() {
			if path != request.SourceRef && (entry.Name() == "build" || entry.Name() == ".gradle" || entry.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(request.SourceRef, path)
		if err != nil {
			return err
		}
		if rel == "local.properties" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 16<<20 || len(files) >= 4096 || int64(totalBytes)+info.Size() > 64<<20 {
			return fmt.Errorf("native source exceeds bounded regular-file project limits")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		totalBytes += len(content)
		files[rel] = content
		hasher.Write([]byte(filepath.ToSlash(rel) + "\x00"))
		hasher.Write(content)
		hasher.Write([]byte{0})
		return nil
	})
	if err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	if !bytes.Equal(data, files["vrooli-native.json"]) {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("native project metadata changed while preparing the source snapshot")
	}
	if stagedMetadata != nil {
		files["vrooli-native.json"] = stagedMetadata
		hasher.Write([]byte("native_profile\x00" + profileName + "\x00"))
		hasher.Write(stagedMetadata)
	}
	if host := parameters["preview_cleartext_host"]; host != "" {
		hasher.Write([]byte("\x00preview_cleartext_host\x00" + host + "\x00"))
	}
	sourceDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	projectRoot, err := os.MkdirTemp(root, "native-"+hex.EncodeToString(hasher.Sum(nil))[:16]+"-")
	if err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	prepared := false
	defer func() {
		if !prepared {
			_ = os.RemoveAll(projectRoot)
		}
	}()
	for rel, content := range files {
		path := filepath.Join(projectRoot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
	}
	if sdk := governedSDKRoot(); sdk != "" {
		if err := os.WriteFile(filepath.Join(projectRoot, "local.properties"), []byte("sdk.dir="+sdk+"\n"), 0o600); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
	}
	if host := parameters["preview_cleartext_host"]; host != "" {
		ip := net.ParseIP(host)
		if (host != "localhost" && (ip == nil || (!ip.IsPrivate() && !ip.IsLoopback()))) || parameters["signing"] == "required" {
			return GeneratedProject{}, projectConfig{}, fmt.Errorf("preview cleartext requires a private/loopback host and debug signing")
		}
		path := filepath.Join(projectRoot, "app/src/debug/res/xml/network_security_config.xml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
		domain := host
		if ip != nil {
			domain = ip.String()
		}
		policy := "<network-security-config><base-config cleartextTrafficPermitted=\"false\"/><domain-config cleartextTrafficPermitted=\"true\"><domain includeSubdomains=\"false\">" + domain + "</domain></domain-config></network-security-config>\n"
		if err := os.WriteFile(path, []byte(policy), 0o600); err != nil {
			return GeneratedProject{}, projectConfig{}, err
		}
	}
	prepared = true
	return GeneratedProject{ProjectPath: projectRoot, SourceRef: request.SourceRef, SourceDigest: sourceDigest, PackageName: config.PackageName, AppName: config.AppName, TargetSDK: config.TargetSDK, BrandName: config.BrandName, BrandSource: config.BrandSource, GeneratedAt: b.now()}, config, nil
}
