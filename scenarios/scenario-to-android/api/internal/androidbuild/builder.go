// Package androidbuild renders and builds the provider-neutral Android shell
// used by scenario-to-android. It owns no signing identity: debug artifacts
// are the only outputs this ramp is allowed to produce.
package androidbuild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type RunCommand func(context.Context, string, ...string) ([]byte, error)

type Builder struct {
	BuildRoot string
	// GradleBin is optional so governed hosts can provide the exact Gradle
	// distribution they installed. An empty value resolves GRADLE_BIN,
	// ANDROID_GRADLE_BIN, then PATH.
	GradleBin string
	Run       RunCommand
	Now       func() time.Time
}

var _ deliveryramp.Builder = Builder{}

var packagePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){2,}$`)

const capacitorVersion = "6.1.0"

type projectConfig struct {
	PackageName string
	AppName     string
	VersionName string
	VersionCode int
	TargetSDK   int
}

func (b Builder) Build(ctx context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	if strings.TrimSpace(request.SourceRef) == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("Android build source reference is required")
	}
	if ctx == nil {
		return deliveryramp.Artifact{}, fmt.Errorf("Android build context is required")
	}
	config, err := parseConfig(request)
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	sourceInfo, err := os.Stat(request.SourceRef)
	if err != nil || !sourceInfo.IsDir() {
		return deliveryramp.Artifact{}, fmt.Errorf("Android web bundle directory %q is unavailable", request.SourceRef)
	}
	root := b.BuildRoot
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(os.TempDir(), "vrooli-android-builds")
	}
	bundleDigest, err := digestBundle(request.SourceRef)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("fingerprint Android web bundle: %w", err)
	}
	digest := sha256.Sum256([]byte(request.SourceRef + "\x00" + bundleDigest + "\x00" + config.PackageName + "\x00" + strconv.Itoa(config.TargetSDK)))
	projectRoot := filepath.Join(root, hex.EncodeToString(digest[:8]))
	if err := renderProject(projectRoot, request.SourceRef, config); err != nil {
		return deliveryramp.Artifact{}, err
	}
	run := b.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		}
	}
	gradleBin := strings.TrimSpace(b.GradleBin)
	if gradleBin == "" {
		gradleBin = strings.TrimSpace(os.Getenv("ANDROID_GRADLE_BIN"))
	}
	if gradleBin == "" {
		gradleBin = strings.TrimSpace(os.Getenv("GRADLE_BIN"))
	}
	if gradleBin == "" {
		if home, homeErr := os.UserHomeDir(); homeErr == nil {
			governed := filepath.Join(home, ".vrooli", "bin", "gradle")
			if _, statErr := os.Stat(governed); statErr == nil {
				gradleBin = governed
			}
		}
	}
	if gradleBin == "" {
		gradleBin = "gradle"
	}
	if filepath.Base(gradleBin) == gradleBin {
		if _, lookErr := exec.LookPath(gradleBin); lookErr != nil {
			return deliveryramp.Artifact{}, fmt.Errorf("Android build unavailable: Gradle is not installed; set GRADLE_BIN or install a governed Gradle distribution: %w", lookErr)
		}
	}
	gradleArgs := []string{"--no-daemon", "--project-dir", projectRoot, ":app:assembleDebug", ":app:bundleDebug"}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ANDROID_GRADLE_OFFLINE")), "1") {
		gradleArgs = append([]string{"--no-daemon", "--offline", "--project-dir", projectRoot}, ":app:assembleDebug", ":app:bundleDebug")
	}
	output, err := run(ctx, gradleBin, gradleArgs...)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		return deliveryramp.Artifact{}, fmt.Errorf("build debug Android APK/AAB: %w: %s", err, message)
	}
	apkPath := filepath.Join(projectRoot, "app", "build", "outputs", "apk", "debug", "app-debug.apk")
	aabPath := filepath.Join(projectRoot, "app", "build", "outputs", "bundle", "debug", "app-debug.aab")
	artifact, err := artifactForOutputs(apkPath, aabPath, b.now(), config.PackageName, config.TargetSDK)
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	return artifact, nil
}

func RenderProject(projectRoot, webRoot, packageName, appName string, targetSDK int) error {
	return renderProject(projectRoot, webRoot, projectConfig{PackageName: packageName, AppName: appName, VersionName: "1.0.0", VersionCode: 1, TargetSDK: targetSDK})
}

func parseConfig(request deliveryramp.BuildRequest) (projectConfig, error) {
	parameters := request.Parameters
	packageName := strings.TrimSpace(parameters["package_name"])
	if packageName == "" {
		packageName = "com.vrooli.generated.app"
	}
	if !packagePattern.MatchString(packageName) {
		return projectConfig{}, fmt.Errorf("invalid Android package name %q", packageName)
	}
	appName := strings.TrimSpace(parameters["app_name"])
	if appName == "" {
		appName = filepath.Base(filepath.Clean(request.SourceRef))
	}
	versionName := strings.TrimSpace(parameters["version_name"])
	if versionName == "" {
		versionName = "1.0.0"
	}
	versionCode := 1
	if raw := strings.TrimSpace(parameters["version_code"]); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return projectConfig{}, fmt.Errorf("version_code must be a positive integer")
		}
		versionCode = parsed
	}
	targetSDK := 36
	if raw := strings.TrimSpace(parameters["target_sdk"]); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return projectConfig{}, fmt.Errorf("target_sdk must be an integer")
		}
		targetSDK = parsed
	}
	if targetSDK < 36 {
		return projectConfig{}, fmt.Errorf("target_sdk %d is below the Android API 36 floor", targetSDK)
	}
	return projectConfig{PackageName: packageName, AppName: appName, VersionName: versionName, VersionCode: versionCode, TargetSDK: targetSDK}, nil
}

func renderProject(projectRoot, webRoot string, config projectConfig) error {
	if err := os.MkdirAll(filepath.Join(projectRoot, "app", "src", "main", "java", filepath.FromSlash(strings.ReplaceAll(config.PackageName, ".", "/"))), 0o755); err != nil {
		return fmt.Errorf("create Android project: %w", err)
	}
	// Capacitor serves the copied bundle from its native `public` asset
	// directory through the WebView origin. Keep this separate from the input
	// bundle so the source scenario is never modified.
	assets := filepath.Join(projectRoot, "app", "src", "main", "assets", "public")
	if err := copyTree(webRoot, assets); err != nil {
		return fmt.Errorf("copy web bundle into Android assets: %w", err)
	}
	webDir := filepath.Join(projectRoot, "web")
	if err := copyTree(webRoot, webDir); err != nil {
		return fmt.Errorf("copy web bundle into Capacitor web directory: %w", err)
	}
	capacitorConfig := fmt.Sprintf("{\n  \"appId\": \"%s\",\n  \"appName\": %s,\n  \"webDir\": \"web\",\n  \"server\": {\n    \"hostname\": \"localhost\",\n    \"androidScheme\": \"http\"\n  }\n}\n", config.PackageName, jsonString(config.AppName))
	files := map[string]string{
		"package.json":                              fmt.Sprintf("{\n  \"private\": true,\n  \"name\": \"%s\",\n  \"dependencies\": {\n    \"@capacitor/core\": \"%s\",\n    \"@capacitor/android\": \"%s\"\n  },\n  \"devDependencies\": {\n    \"@capacitor/cli\": \"%s\"\n  }\n}\n", strings.ReplaceAll(strings.ToLower(config.PackageName), ".", "-"), capacitorVersion, capacitorVersion, capacitorVersion),
		"capacitor.config.json":                     capacitorConfig,
		"app/src/main/assets/capacitor.config.json": capacitorConfig,
		"settings.gradle":                           "pluginManagement { repositories { google(); mavenCentral(); gradlePluginPortal() } }\ndependencyResolutionManagement { repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS); repositories { google(); mavenCentral() } }\nrootProject.name = 'vrooli-generated-app'\ninclude ':app'\n",
		"build.gradle":                              "plugins { id 'com.android.application' version '8.7.3' apply false }\n",
		"gradle.properties":                         "android.useAndroidX=true\nandroid.nonTransitiveRClass=true\nandroid.suppressUnsupportedCompileSdk=36\n",
		"app/build.gradle":                          fmt.Sprintf("plugins { id 'com.android.application' }\n\nandroid { namespace '%s'; compileSdk %d; buildToolsVersion '35.0.0'\n defaultConfig { applicationId '%s'; minSdk 26; targetSdk %d; versionCode %d; versionName '%s' }\n compileOptions { sourceCompatibility JavaVersion.VERSION_17; targetCompatibility JavaVersion.VERSION_17 }\n}\n\ndependencies { implementation 'com.capacitorjs:core:%s'; implementation 'androidx.appcompat:appcompat:1.7.0' }\n", config.PackageName, config.TargetSDK, config.PackageName, config.TargetSDK, config.VersionCode, config.VersionName, capacitorVersion),
		"app/src/main/AndroidManifest.xml":          fmt.Sprintf("<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\"><uses-permission android:name=\"android.permission.INTERNET\"/><uses-permission android:name=\"android.permission.POST_NOTIFICATIONS\"/><application android:usesCleartextTraffic=\"true\" android:theme=\"@style/AppTheme\" android:label=\"%s\"><activity android:name=\".MainActivity\" android:exported=\"true\"><intent-filter><action android:name=\"android.intent.action.MAIN\"/><category android:name=\"android.intent.category.LAUNCHER\"/></intent-filter><intent-filter><action android:name=\"android.intent.action.VIEW\"/><category android:name=\"android.intent.category.DEFAULT\"/><category android:name=\"android.intent.category.BROWSABLE\"/><data android:scheme=\"vrooli-hello\" android:host=\"home\"/></intent-filter></activity></application></manifest>\n", xmlEscape(config.AppName)),
		"app/src/main/res/values/styles.xml":        "<resources><style name=\"AppTheme\" parent=\"android:style/Theme.Material.Light.NoActionBar\"><item name=\"android:fontFamily\">sans</item><item name=\"android:colorAccent\">#6750A4</item></style></resources>\n",
		filepath.ToSlash(filepath.Join("app", "src", "main", "java", filepath.FromSlash(strings.ReplaceAll(config.PackageName, ".", "/")), "MainActivity.java")): fmt.Sprintf(`package %s;

import com.getcapacitor.BridgeActivity;

public final class MainActivity extends BridgeActivity {}
`, config.PackageName),
	}
	if sdkRoot := governedSDKRoot(); sdkRoot != "" {
		files["local.properties"] = "sdk.dir=" + strings.ReplaceAll(filepath.ToSlash(sdkRoot), "\\", "\\\\") + "\n"
	}
	for relative, content := range files {
		path := filepath.Join(projectRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write Android project file %s: %w", relative, err)
		}
	}
	return nil
}

func governedSDKRoot() string {
	for _, candidate := range []string{os.Getenv("ANDROID_SDK_ROOT"), os.Getenv("ANDROID_HOME")} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			if _, err := os.Stat(filepath.Join(candidate, "platforms")); err == nil {
				return candidate
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".vrooli", "resources", "android-sdk")
		if _, err := os.Stat(filepath.Join(candidate, "platforms")); err == nil {
			return candidate
		}
	}
	return ""
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func digestBundle(source string) (string, error) {
	hasher := sha256.New()
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(hasher, filepath.ToSlash(relative))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(data)
		_, _ = hasher.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func artifactForOutputs(apkPath, aabPath string, createdAt time.Time, packageName string, targetSDK int) (deliveryramp.Artifact, error) {
	apk, err := os.ReadFile(apkPath)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("debug APK was not produced: %w", err)
	}
	aab, err := os.ReadFile(aabPath)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("debug AAB was not produced: %w", err)
	}
	hasher := sha256.New()
	_, _ = hasher.Write(apk)
	_, _ = hasher.Write([]byte("\x00"))
	_, _ = hasher.Write(aab)
	digest := hasher.Sum(nil)
	return deliveryramp.Artifact{ImmutableRef: "android-debug:" + hex.EncodeToString(digest), LocalPath: apkPath, Kind: "android-debug-apk-aab", Checksum: "sha256:" + hex.EncodeToString(digest), SizeBytes: int64(len(apk) + len(aab)), CreatedAt: createdAt, Metadata: map[string]string{"apk_path": apkPath, "aab_path": aabPath, "package_name": packageName, "signing": "debug-only", "target_sdk": strconv.Itoa(targetSDK), "target_api_assertion": fmt.Sprintf("targetSdk(%d) >= 36", targetSDK)}}, nil
}

func (b Builder) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

func xmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(value)
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return strconv.Quote(value)
	}
	return string(data)
}
