package androidartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"device-control/internal/httpc"
)

const maxAPKBytes int64 = 1 << 30

var (
	immutableRefPattern = regexp.MustCompile(`^android-(debug|release):[0-9a-f]{64}$`)
	checksumPattern     = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	certificatePattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	packagePattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

type Descriptor struct {
	ImmutableRef string `json:"immutable_ref"`
	Kind         string `json:"kind"`
	Checksum     string `json:"checksum"`
	APKChecksum  string `json:"apk_checksum"`
	SizeBytes    int64  `json:"size_bytes"`
	APKSizeBytes int64  `json:"apk_size_bytes"`
	PackageName  string `json:"package_name"`
	VersionName  string `json:"version_name"`
	VersionCode  int64  `json:"version_code"`
}

type BaseURLResolver func(context.Context) (string, error)

type APKIdentity struct {
	PackageName string
	VersionName string
	VersionCode int64
}

type APKInspector interface {
	Inspect(context.Context, string) (APKIdentity, error)
}

type APKSigningInspector interface {
	InspectSigningCertificate(context.Context, string) (string, error)
}

// AndroidSDKSigningInspector reads the signing certificate digest from the
// staged APK. The result is computed from verified immutable bytes; it is not
// trusted from producer-supplied descriptive metadata.
type AndroidSDKSigningInspector struct{ ADBPath string }

func (i AndroidSDKSigningInspector) InspectSigningCertificate(ctx context.Context, apkPath string) (string, error) {
	apkSignerPath, err := resolveBuildToolsTool(i.ADBPath, "apksigner")
	if err != nil {
		return "", err
	}
	output, err := exec.CommandContext(ctx, apkSignerPath, "verify", "--print-certs", apkPath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect APK signing certificate: %w", err)
	}
	match := certificateDigestPattern.FindSubmatch(output)
	if len(match) != 2 {
		return "", fmt.Errorf("APK signing certificate SHA-256 digest is missing or malformed")
	}
	digest := strings.ToLower(string(match[1]))
	if !certificatePattern.MatchString(digest) {
		return "", fmt.Errorf("APK signing certificate SHA-256 digest is invalid")
	}
	return digest, nil
}

var certificateDigestPattern = regexp.MustCompile(`(?mi)^Signer #1 certificate SHA-256 digest:\s*([0-9a-f]{64})\s*$`)

// ValidSigningCertificateSHA256 reports whether value is a normalized
// lowercase SHA-256 certificate digest returned by artifact verification.
func ValidSigningCertificateSHA256(value string) bool {
	return certificatePattern.MatchString(value)
}

// AAPTInspector reads the identity embedded in a compiled APK manifest using
// the Android SDK that owns the adb executable on PATH.
type AAPTInspector struct{ ADBPath string }

func (i AAPTInspector) Inspect(ctx context.Context, apkPath string) (APKIdentity, error) {
	aaptPath, err := resolveAAPT(i.ADBPath)
	if err != nil {
		return APKIdentity{}, err
	}
	output, err := exec.CommandContext(ctx, aaptPath, "dump", "badging", apkPath).Output()
	if err != nil {
		return APKIdentity{}, fmt.Errorf("inspect APK manifest: %w", err)
	}
	match := packageBadgingPattern.FindSubmatch(output)
	if len(match) != 4 {
		return APKIdentity{}, fmt.Errorf("APK manifest package identity is missing or malformed")
	}
	versionCode, err := strconv.ParseInt(string(match[2]), 10, 64)
	if err != nil || versionCode <= 0 {
		return APKIdentity{}, fmt.Errorf("APK manifest version code is missing or invalid")
	}
	return APKIdentity{PackageName: string(match[1]), VersionCode: versionCode, VersionName: string(match[3])}, nil
}

var packageBadgingPattern = regexp.MustCompile(`(?m)^package: name='([^']+)' versionCode='([0-9]+)' versionName='([^']*)'`)

func resolveAAPT(adbPath string) (string, error) {
	if strings.TrimSpace(adbPath) == "" {
		adbPath = "adb"
	}
	resolved, err := exec.LookPath(adbPath)
	if err != nil {
		return "", fmt.Errorf("locate adb for APK manifest inspection: %w", err)
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve adb for APK manifest inspection: %w", err)
	}
	sdkRoot := filepath.Dir(filepath.Dir(resolved))
	buildTools := filepath.Join(sdkRoot, "build-tools")
	versions, err := os.ReadDir(buildTools)
	if err != nil {
		return "", fmt.Errorf("locate Android SDK build tools: %w", err)
	}
	sort.Slice(versions, func(a, b int) bool { return versions[a].Name() > versions[b].Name() })
	for _, version := range versions {
		if !version.IsDir() {
			continue
		}
		candidate := filepath.Join(buildTools, version.Name(), "aapt")
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Android SDK build tools do not provide aapt")
}

type Client struct {
	HTTP             httpc.Doer
	BaseURL          BaseURLResolver
	Inspector        APKInspector
	SigningInspector APKSigningInspector
}

// StagedAPK is a bounded, independently checksum-verified temporary APK.
// Close removes the staged bytes after the device operation finishes.
type StagedAPK struct {
	Descriptor               Descriptor
	SigningCertificateSHA256 string
	Path                     string
}

func (a StagedAPK) Close() error {
	if a.Path == "" {
		return nil
	}
	return os.Remove(a.Path)
}

// Resolve verifies producer identity and expected package before downloading
// the APK. It then verifies the returned bytes against both the producer's
// APK checksum and exact declared byte count before publishing a private temp
// file for the caller's lease-owned install operation.
func (c Client) Resolve(ctx context.Context, immutableRef, expectedPackage, expectedVersion string) (StagedAPK, error) {
	if !immutableRefPattern.MatchString(immutableRef) {
		return StagedAPK{}, fmt.Errorf("immutable Android artifact reference is invalid")
	}
	if !packagePattern.MatchString(expectedPackage) {
		return StagedAPK{}, fmt.Errorf("expected package must be a fully-qualified application identity")
	}
	if strings.TrimSpace(expectedVersion) != expectedVersion || len(expectedVersion) > 128 {
		return StagedAPK{}, fmt.Errorf("expected version name is invalid")
	}
	if c.HTTP == nil || c.BaseURL == nil {
		return StagedAPK{}, fmt.Errorf("Android artifact producer is unavailable")
	}
	base, err := c.BaseURL(ctx)
	if err != nil {
		return StagedAPK{}, fmt.Errorf("resolve Android artifact producer: %w", err)
	}
	if strings.TrimSpace(base) == "" {
		return StagedAPK{}, fmt.Errorf("resolve Android artifact producer: URL is empty")
	}
	base = strings.TrimRight(base, "/")
	descriptorURL, err := artifactURL(base, "/api/v1/android/artifacts/metadata", immutableRef)
	if err != nil {
		return StagedAPK{}, err
	}
	var descriptor Descriptor
	if err := c.getJSON(ctx, descriptorURL, &descriptor); err != nil {
		return StagedAPK{}, fmt.Errorf("resolve Android artifact metadata: %w", err)
	}
	refParts := strings.SplitN(immutableRef, ":", 2)
	expectedKind := "android-" + refParts[0][len("android-"):] + "-apk-aab"
	if descriptor.ImmutableRef != immutableRef || descriptor.Kind != expectedKind || !checksumPattern.MatchString(descriptor.Checksum) || descriptor.Checksum != "sha256:"+refParts[1] || !checksumPattern.MatchString(descriptor.APKChecksum) || descriptor.SizeBytes <= 0 || descriptor.APKSizeBytes <= 0 || descriptor.APKSizeBytes > maxAPKBytes || descriptor.APKSizeBytes >= descriptor.SizeBytes || descriptor.VersionCode <= 0 || descriptor.PackageName != expectedPackage || (expectedVersion != "" && descriptor.VersionName != expectedVersion) || !packagePattern.MatchString(descriptor.PackageName) || descriptor.VersionName == "" || len(descriptor.VersionName) > 128 {
		return StagedAPK{}, fmt.Errorf("Android artifact descriptor does not match the requested identity")
	}
	apkURL, err := artifactURL(base, "/api/v1/android/artifacts/apk", immutableRef)
	if err != nil {
		return StagedAPK{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apkURL, nil)
	if err != nil {
		return StagedAPK{}, fmt.Errorf("create APK request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.android.package-archive")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return StagedAPK{}, fmt.Errorf("fetch Android APK: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return StagedAPK{}, fmt.Errorf("fetch Android APK: producer returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength >= 0 && resp.ContentLength != descriptor.APKSizeBytes {
		return StagedAPK{}, fmt.Errorf("Android APK length does not match its descriptor")
	}
	file, err := os.CreateTemp(os.TempDir(), "device-control-verified-apk-*.apk")
	if err != nil {
		return StagedAPK{}, fmt.Errorf("stage Android APK: %w", err)
	}
	path := filepath.Clean(file.Name())
	removeOnError := true
	defer func() {
		_ = file.Close()
		if removeOnError {
			_ = os.Remove(path)
		}
	}()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, descriptor.APKSizeBytes+1))
	if err != nil {
		return StagedAPK{}, fmt.Errorf("stage Android APK bytes: %w", err)
	}
	if size != descriptor.APKSizeBytes || size > maxAPKBytes || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != descriptor.APKChecksum {
		return StagedAPK{}, fmt.Errorf("Android APK bytes failed checksum or size verification")
	}
	if err := file.Sync(); err != nil {
		return StagedAPK{}, fmt.Errorf("flush verified Android APK: %w", err)
	}
	if err := file.Close(); err != nil {
		return StagedAPK{}, fmt.Errorf("close verified Android APK: %w", err)
	}
	if c.Inspector == nil {
		return StagedAPK{}, fmt.Errorf("Android APK manifest inspector is unavailable")
	}
	identity, err := c.Inspector.Inspect(ctx, path)
	if err != nil {
		return StagedAPK{}, fmt.Errorf("verify APK manifest identity: %w", err)
	}
	if identity.PackageName != descriptor.PackageName || identity.VersionName != descriptor.VersionName || identity.VersionCode != descriptor.VersionCode {
		return StagedAPK{}, fmt.Errorf("APK manifest identity does not match its immutable descriptor")
	}
	if c.SigningInspector == nil {
		return StagedAPK{}, fmt.Errorf("Android APK signing inspector is unavailable")
	}
	signingCertificate, err := c.SigningInspector.InspectSigningCertificate(ctx, path)
	if err != nil {
		return StagedAPK{}, fmt.Errorf("verify APK signing certificate: %w", err)
	}
	signingCertificate = strings.ToLower(strings.TrimSpace(signingCertificate))
	if !certificatePattern.MatchString(signingCertificate) {
		return StagedAPK{}, fmt.Errorf("APK signing certificate SHA-256 digest is invalid")
	}
	removeOnError = false
	return StagedAPK{Descriptor: descriptor, SigningCertificateSHA256: signingCertificate, Path: path}, nil
}

func resolveBuildToolsTool(adbPath, tool string) (string, error) {
	if strings.TrimSpace(adbPath) == "" {
		adbPath = "adb"
	}
	resolved, err := exec.LookPath(adbPath)
	if err != nil {
		return "", fmt.Errorf("locate adb for Android SDK build-tools inspection: %w", err)
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve adb for Android SDK build-tools inspection: %w", err)
	}
	buildTools := filepath.Join(filepath.Dir(filepath.Dir(resolved)), "build-tools")
	versions, err := os.ReadDir(buildTools)
	if err != nil {
		return "", fmt.Errorf("locate Android SDK build tools: %w", err)
	}
	sort.Slice(versions, func(a, b int) bool { return versions[a].Name() > versions[b].Name() })
	for _, version := range versions {
		if !version.IsDir() {
			continue
		}
		candidate := filepath.Join(buildTools, version.Name(), tool)
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Android SDK build tools do not provide %s", tool)
}

func (c Client) getJSON(ctx context.Context, target string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("producer returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(result); err != nil {
		return fmt.Errorf("decode producer metadata: %w", err)
	}
	return nil
}

func artifactURL(base, path, immutableRef string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("Android artifact producer URL is invalid")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	query := parsed.Query()
	query.Set("immutable_ref", immutableRef)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
