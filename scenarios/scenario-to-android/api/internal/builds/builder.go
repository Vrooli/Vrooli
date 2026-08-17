// Package builds renders and builds the provider-neutral Android shell
// used by scenario-to-android. Signing values are resolved only in memory
// through the credential authority and are never copied into the generated
// project or evidence.
package builds

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
	Signing   SigningStore
}

// SigningStore is deliberately narrower than the credential-client surface.
// The builder only needs to resolve an existing upload identity; provisioning
// is an explicit operator action handled by ProvisionSigningKey.
type SigningStore interface {
	Resolve(context.Context, string, string) (string, error)
}

const (
	DefaultSigningIdentity = "vrooli/scenario-to-android/android-upload-signing-key"
	SigningKeystoreField   = "keystore-base64"
	SigningPasswordField   = "password" //gitleaks:allow -- public credential field label, not credential material
	SigningAliasField      = "alias"
)

var _ deliveryramp.Builder = Builder{}

var packagePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){2,}$`)

const capacitorVersion = "6.1.0"

type projectConfig struct {
	PackageName string
	AppName     string
	VersionName string
	VersionCode int
	TargetSDK   int
	BrandName   string
	BrandID     string
	BrandSource string
	Primary     string
	Secondary   string
	Accent      string
	Background  string
	TextColor   string
}

// GeneratedProject is the operator-facing result of rendering the Android
// shell. It deliberately stops before Gradle and contains no signing
// material; Build consumes the same project preparation seam to produce APK
// and AAB outputs.
type GeneratedProject struct {
	ProjectPath string    `json:"project_path"`
	SourceRef   string    `json:"source_ref"`
	PackageName string    `json:"package_name"`
	AppName     string    `json:"app_name"`
	TargetSDK   int       `json:"target_sdk"`
	BrandID     string    `json:"brand_id,omitempty"`
	BrandName   string    `json:"brand_name,omitempty"`
	BrandSource string    `json:"branding_source"`
	GeneratedAt time.Time `json:"generated_at"`
}

func (b Builder) Generate(ctx context.Context, request deliveryramp.BuildRequest) (GeneratedProject, error) {
	generated, _, err := b.prepareProject(ctx, request)
	return generated, err
}

func (b Builder) Build(ctx context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	generated, config, err := b.prepareProject(ctx, request)
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	projectRoot := generated.ProjectPath
	run := b.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput() // #nosec G702 -- explicit governed Gradle binary; tests inject Run
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
	buildType := "debug"
	gradleTasks := []string{" :app:assembleDebug", ":app:bundleDebug"}
	signingIdentity := ""
	var signingFiles []string
	if strings.EqualFold(strings.TrimSpace(request.Parameters["signing"]), "required") {
		buildType = "release"
		gradleTasks = []string{" :app:assembleRelease", ":app:bundleRelease"}
		identity := strings.TrimSpace(request.Parameters["signing_identity"])
		if identity == "" {
			identity = DefaultSigningIdentity
		}
		signingIdentity = identity
		files, signingErr := b.prepareSigningFiles(ctx, projectRoot, identity)
		if signingErr != nil {
			return deliveryramp.Artifact{}, signingErr
		}
		signingFiles = files
		defer removeFiles(signingFiles)
	}
	gradleArgs := []string{"--no-daemon"}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ANDROID_GRADLE_OFFLINE")), "1") {
		gradleArgs = append(gradleArgs, "--offline")
	}
	gradleArgs = append(gradleArgs, "--project-dir", projectRoot)
	if len(signingFiles) > 0 {
		gradleArgs = append(gradleArgs, "-Dvrooli.android.signing.properties="+signingFiles[1])
	}
	for _, task := range gradleTasks {
		gradleArgs = append(gradleArgs, strings.TrimSpace(task))
	}
	output, err := run(ctx, gradleBin, gradleArgs...)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		return deliveryramp.Artifact{}, fmt.Errorf("build %s Android APK/AAB: %w: %s", buildType, err, message)
	}
	apkName, aabName := "app-debug.apk", "app-debug.aab"
	if buildType == "release" {
		apkName, aabName = "app-release.apk", "app-release.aab"
	}
	apkPath := filepath.Join(projectRoot, "app", "build", "outputs", "apk", buildType, apkName)
	aabPath := filepath.Join(projectRoot, "app", "build", "outputs", "bundle", buildType, aabName)
	artifact, err := artifactForOutputsWithSigning(apkPath, aabPath, b.now(), config.PackageName, config.TargetSDK, buildType)
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	artifact.Metadata["branding_source"] = config.BrandSource
	artifact.Metadata["brand_name"] = config.BrandName
	if buildType == "release" {
		artifact.Metadata["signing_identity"] = signingIdentity
	}
	return artifact, nil
}

func (b Builder) prepareProject(ctx context.Context, request deliveryramp.BuildRequest) (GeneratedProject, projectConfig, error) {
	if strings.TrimSpace(request.SourceRef) == "" {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("Android build source reference is required")
	}
	if ctx == nil {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("Android build context is required")
	}
	parameters := cloneParameters(request.Parameters)
	loadAssignedBrand(ctx, parameters)
	request.Parameters = parameters
	config, err := parseConfig(request)
	if err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	sourceInfo, err := os.Stat(request.SourceRef)
	if err != nil || !sourceInfo.IsDir() {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("Android web bundle directory %q is unavailable", request.SourceRef)
	}
	root := b.BuildRoot
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(os.TempDir(), "vrooli-android-builds")
	}
	bundleDigest, err := digestBundle(request.SourceRef)
	if err != nil {
		return GeneratedProject{}, projectConfig{}, fmt.Errorf("fingerprint Android web bundle: %w", err)
	}
	digest := sha256.Sum256([]byte(request.SourceRef + "\x00" + bundleDigest + "\x00" + config.PackageName + "\x00" + strconv.Itoa(config.TargetSDK) + "\x00" + config.BrandID + "\x00" + config.Primary))
	projectRoot := filepath.Join(root, hex.EncodeToString(digest[:8]))
	if err := renderProject(projectRoot, request.SourceRef, config); err != nil {
		return GeneratedProject{}, projectConfig{}, err
	}
	return GeneratedProject{ProjectPath: projectRoot, SourceRef: request.SourceRef, PackageName: config.PackageName, AppName: config.AppName, TargetSDK: config.TargetSDK, BrandID: config.BrandID, BrandName: config.BrandName, BrandSource: config.BrandSource, GeneratedAt: b.now()}, config, nil
}

func RenderProject(projectRoot, webRoot, packageName, appName string, targetSDK int) error {
	return renderProject(projectRoot, webRoot, projectConfig{PackageName: packageName, AppName: appName, VersionName: "1.0.0", VersionCode: 1, TargetSDK: targetSDK, BrandSource: "placeholder", Primary: "#2457D6", Secondary: "#173B8F", Accent: "#FFB020", Background: "#F4F7FF", TextColor: "#13203A"})
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
	primary := normalizedColor(parameters["brand_primary"], "#2457D6")
	secondary := normalizedColor(parameters["brand_secondary"], "#173B8F")
	accent := normalizedColor(parameters["brand_accent"], "#FFB020")
	background := normalizedColor(parameters["brand_background"], "#F4F7FF")
	textColor := normalizedColor(parameters["brand_text"], "#13203A")
	brandName := strings.TrimSpace(parameters["brand_name"])
	brandID := strings.TrimSpace(parameters["brand_id"])
	brandSource := strings.TrimSpace(parameters["branding_source"])
	if brandSource == "" {
		brandSource = "placeholder"
	}
	if brandName == "" {
		brandName = "Generic Vrooli App"
	}
	return projectConfig{PackageName: packageName, AppName: appName, VersionName: versionName, VersionCode: versionCode, TargetSDK: targetSDK, BrandName: brandName, BrandID: brandID, BrandSource: brandSource, Primary: primary, Secondary: secondary, Accent: accent, Background: background, TextColor: textColor}, nil
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
		"package.json":                                         fmt.Sprintf("{\n  \"private\": true,\n  \"name\": \"%s\",\n  \"dependencies\": {\n    \"@capacitor/core\": \"%s\",\n    \"@capacitor/android\": \"%s\"\n  },\n  \"devDependencies\": {\n    \"@capacitor/cli\": \"%s\"\n  }\n}\n", strings.ReplaceAll(strings.ToLower(config.PackageName), ".", "-"), capacitorVersion, capacitorVersion, capacitorVersion),
		"capacitor.config.json":                                capacitorConfig,
		"app/src/main/assets/capacitor.config.json":            capacitorConfig,
		"settings.gradle":                                      "pluginManagement { repositories { google(); mavenCentral(); gradlePluginPortal() } }\ndependencyResolutionManagement { repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS); repositories { google(); mavenCentral() } }\nrootProject.name = 'vrooli-generated-app'\ninclude ':app'\n",
		"build.gradle":                                         "plugins { id 'com.android.application' version '8.7.3' apply false }\n",
		"gradle.properties":                                    "android.useAndroidX=true\nandroid.nonTransitiveRClass=true\nandroid.suppressUnsupportedCompileSdk=36\n",
		"app/build.gradle":                                     fmt.Sprintf("plugins { id 'com.android.application' }\n\nandroid { namespace '%s'; compileSdk %d; buildToolsVersion '35.0.0'\n defaultConfig { applicationId '%s'; minSdk 26; targetSdk %d; versionCode %d; versionName '%s' }\n signingConfigs { release { def path = System.getProperty('vrooli.android.signing.properties', '') ; if (path) { def p = new Properties(); file(path).withInputStream { p.load(it) }; storeFile file(p['storeFile']); storePassword p['storePassword']; keyAlias p['keyAlias']; keyPassword p['keyPassword'] } } }\n buildTypes { release { signingConfig signingConfigs.release; minifyEnabled false } }\n compileOptions { sourceCompatibility JavaVersion.VERSION_17; targetCompatibility JavaVersion.VERSION_17 }\n}\n\ndependencies { implementation 'com.capacitorjs:core:%s'; implementation 'androidx.appcompat:appcompat:1.7.0' }\n", config.PackageName, config.TargetSDK, config.PackageName, config.TargetSDK, config.VersionCode, config.VersionName, capacitorVersion),
		"app/src/main/AndroidManifest.xml":                     fmt.Sprintf("<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\"><uses-permission android:name=\"android.permission.INTERNET\"/><uses-permission android:name=\"android.permission.POST_NOTIFICATIONS\"/><application android:usesCleartextTraffic=\"true\" android:theme=\"@style/AppTheme\" android:label=\"%s\" android:icon=\"@drawable/ic_launcher_legacy\"><activity android:name=\".MainActivity\" android:exported=\"true\"><intent-filter><action android:name=\"android.intent.action.MAIN\"/><category android:name=\"android.intent.category.LAUNCHER\"/></intent-filter><intent-filter><action android:name=\"android.intent.action.VIEW\"/><category android:name=\"android.intent.category.DEFAULT\"/><category android:name=\"android.intent.category.BROWSABLE\"/><data android:scheme=\"vrooli-hello\" android:host=\"home\"/></intent-filter><intent-filter><action android:name=\"android.intent.action.SEND\"/><category android:name=\"android.intent.category.DEFAULT\"/><data android:mimeType=\"text/plain\"/></intent-filter><meta-data android:name=\"android.app.shortcuts\" android:resource=\"@xml/shortcuts\"/></activity></application></manifest>\n", xmlEscape(config.AppName)),
		"app/src/main/res/values/styles.xml":                   fmt.Sprintf("<resources><style name=\"AppTheme\" parent=\"android:style/Theme.Material.Light.NoActionBar\"><item name=\"android:fontFamily\">sans</item><item name=\"android:colorAccent\">%s</item><item name=\"android:windowSplashScreenBackground\">%s</item><item name=\"android:windowSplashScreenAnimatedIcon\">@drawable/ic_launcher_legacy</item></style></resources>\n", config.Accent, config.Background),
		"app/src/main/res/values/colors.xml":                   fmt.Sprintf("<resources><color name=\"brand_primary\">%s</color><color name=\"brand_secondary\">%s</color><color name=\"brand_accent\">%s</color><color name=\"brand_background\">%s</color><color name=\"brand_text\">%s</color></resources>\n", config.Primary, config.Secondary, config.Accent, config.Background, config.TextColor),
		"app/src/main/res/values/branding.xml":                 fmt.Sprintf("<resources><string name=\"brand_source\">%s</string><string name=\"brand_name\">%s</string></resources>\n", xmlEscape(config.BrandSource), xmlEscape(config.BrandName)),
		"app/src/main/res/values/strings.xml":                  fmt.Sprintf("<resources><string name=\"shortcut_home_short\">%s</string><string name=\"shortcut_home_long\">Open %s</string></resources>\n", xmlEscape(config.AppName), xmlEscape(config.AppName)),
		"app/src/main/res/drawable/ic_launcher_foreground.xml": fmt.Sprintf("<vector xmlns:android=\"http://schemas.android.com/apk/res/android\" android:width=\"108dp\" android:height=\"108dp\" android:viewportWidth=\"108\" android:viewportHeight=\"108\"><path android:fillColor=\"%s\" android:pathData=\"M54,8A46,46 0,1 0,54 100A46,46 0,1 0,54 8\"/><path android:fillColor=\"%s\" android:pathData=\"M30,54L48,72L80,36L73,29L48,58L37,47Z\"/></vector>\n", config.Primary, config.Accent),
		"app/src/main/res/drawable/ic_launcher_legacy.xml":     fmt.Sprintf("<vector xmlns:android=\"http://schemas.android.com/apk/res/android\" android:width=\"48dp\" android:height=\"48dp\" android:viewportWidth=\"48\" android:viewportHeight=\"48\"><path android:fillColor=\"%s\" android:pathData=\"M24,2A22,22 0,1 0,24 46A22,22 0,1 0,24 2\"/><path android:fillColor=\"%s\" android:pathData=\"M11,24L20,33L37,14L33,10L20,25L15,20Z\"/></vector>\n", config.Primary, config.Accent),
		"app/src/main/res/drawable/ic_notification.xml":        "<vector xmlns:android=\"http://schemas.android.com/apk/res/android\" android:width=\"24dp\" android:height=\"24dp\" android:viewportWidth=\"24\" android:viewportHeight=\"24\"><path android:fillColor=\"#FFFFFFFF\" android:pathData=\"M12,2A10,10 0,1 0,12 22A10,10 0,1 0,12 2\"/></vector>\n",
		"app/src/main/res/drawable/splash_background.xml":      "<layer-list xmlns:android=\"http://schemas.android.com/apk/res/android\"><item android:drawable=\"@color/brand_background\"/><item android:gravity=\"center\" android:drawable=\"@drawable/ic_launcher_legacy\"/></layer-list>\n",
		"app/src/main/res/xml/shortcuts.xml":                   "<shortcuts xmlns:android=\"http://schemas.android.com/apk/res/android\"><shortcut android:shortcutId=\"home\" android:enabled=\"true\" android:icon=\"@drawable/ic_launcher_legacy\" android:shortcutShortLabel=\"@string/shortcut_home_short\" android:shortcutLongLabel=\"@string/shortcut_home_long\"><intent android:action=\"android.intent.action.VIEW\" android:data=\"vrooli-hello://home\"/></shortcut></shortcuts>\n",
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

func cloneParameters(input map[string]string) map[string]string {
	output := make(map[string]string, len(input)+8)
	for key, value := range input {
		output[key] = value
	}
	return output
}

func normalizedColor(value, fallback string) string {
	value = strings.TrimSpace(value)
	if matched, _ := regexp.MatchString(`^#[0-9a-fA-F]{6,8}$`, value); matched {
		return value
	}
	return fallback
}

func loadAssignedBrand(ctx context.Context, parameters map[string]string) {
	scenario := strings.TrimSpace(parameters["scenario_name"])
	if scenario == "" || strings.TrimSpace(parameters["brand_id"]) != "" || os.Getenv("ANDROID_SKIP_BRAND_LOOKUP") == "1" {
		return
	}
	statusOutput, err := exec.CommandContext(ctx, "brand-manager", "assignments", "status", scenario, "--json").Output()
	if err != nil {
		parameters["branding_source"] = "placeholder (brand-manager unavailable)"
		return
	}
	var status struct {
		Status struct {
			HasBrand bool   `json:"has_brand"`
			BrandID  string `json:"brand_id"`
		} `json:"status"`
	}
	if json.Unmarshal(statusOutput, &status) != nil || !status.Status.HasBrand || status.Status.BrandID == "" {
		parameters["branding_source"] = "placeholder (no assigned brand)"
		return
	}
	brandOutput, err := exec.CommandContext(ctx, "brand-manager", "brands", "get", status.Status.BrandID, "--json").Output()
	if err != nil {
		parameters["branding_source"] = "placeholder (brand asset unavailable)"
		return
	}
	var response struct {
		Brand struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Colors struct {
				Primary, Secondary, Accent, Background, Text string
			} `json:"colors"`
		} `json:"brand"`
	}
	if json.Unmarshal(brandOutput, &response) != nil || response.Brand.ID == "" {
		parameters["branding_source"] = "placeholder (invalid brand output)"
		return
	}
	parameters["brand_id"] = response.Brand.ID
	parameters["brand_name"] = response.Brand.Name
	parameters["brand_primary"] = response.Brand.Colors.Primary
	parameters["brand_secondary"] = response.Brand.Colors.Secondary
	parameters["brand_accent"] = response.Brand.Colors.Accent
	parameters["brand_background"] = response.Brand.Colors.Background
	parameters["brand_text"] = response.Brand.Colors.Text
	parameters["branding_source"] = "brand-manager:" + response.Brand.ID
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
	return artifactForOutputsWithSigning(apkPath, aabPath, createdAt, packageName, targetSDK, "debug")
}

func artifactForOutputsWithSigning(apkPath, aabPath string, createdAt time.Time, packageName string, targetSDK int, buildType string) (deliveryramp.Artifact, error) {
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
	signing := "debug-only"
	kind := "android-debug-apk-aab"
	ref := "android-debug:"
	if buildType == "release" {
		signing = "secrets-manager"
		kind = "android-release-apk-aab"
		ref = "android-release:"
	}
	return deliveryramp.Artifact{ImmutableRef: ref + hex.EncodeToString(digest), LocalPath: apkPath, Kind: kind, Checksum: "sha256:" + hex.EncodeToString(digest), SizeBytes: int64(len(apk) + len(aab)), CreatedAt: createdAt, Metadata: map[string]string{"apk_path": apkPath, "aab_path": aabPath, "package_name": packageName, "signing": signing, "target_sdk": strconv.Itoa(targetSDK), "target_api_assertion": fmt.Sprintf("targetSdk(%d) >= 36", targetSDK)}}, nil
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
