package builds

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const maxRetainedArtifactBytes int64 = 1 << 30

var (
	immutableAndroidRef = regexp.MustCompile(`^(android-debug|android-release):([0-9a-f]{64})$`)
	androidPackageName  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

// APKDescriptor is the path-free owner contract consumed by Device Control.
// ImmutableRef binds the retained APK and AAB pair; APKChecksum lets the
// consumer independently verify the exact bytes it will install.
type APKDescriptor struct {
	ImmutableRef string    `json:"immutable_ref"`
	Kind         string    `json:"kind"`
	Checksum     string    `json:"checksum"`
	APKChecksum  string    `json:"apk_checksum"`
	SizeBytes    int64     `json:"size_bytes"`
	APKSizeBytes int64     `json:"apk_size_bytes"`
	PackageName  string    `json:"package_name"`
	VersionName  string    `json:"version_name"`
	VersionCode  int64     `json:"version_code"`
	CreatedAt    time.Time `json:"created_at"`
}

// ResolvedAPK holds the verified owner descriptor and an open handle to the
// retained APK. The caller must close File after the response is written.
type ResolvedAPK struct {
	Descriptor APKDescriptor
	File       *os.File
}

// ResolveAPK resolves only an exact retained immutable reference. It derives
// paths from the validated digest instead of trusting manifest paths, rejects
// symlinks/non-regular files and verifies the full APK/AAB identity before
// returning the APK handle.
func ResolveAPK(dataRoot, immutableRef string) (ResolvedAPK, error) {
	match := immutableAndroidRef.FindStringSubmatch(strings.TrimSpace(immutableRef))
	if match == nil || dataRoot == "" {
		return ResolvedAPK{}, errors.New("Android artifact reference is invalid or unavailable")
	}
	kind, digest := match[1], match[2]
	if immutableRef != strings.TrimSpace(immutableRef) {
		return ResolvedAPK{}, errors.New("Android artifact reference is invalid or unavailable")
	}
	root := filepath.Join(dataRoot, "builds", "artifacts", digest)
	if err := requireRegularDirectory(root); err != nil {
		return ResolvedAPK{}, errors.New("Android artifact is unavailable")
	}
	manifestPath := filepath.Join(root, "artifact.json")
	if err := requireRegularFile(manifestPath); err != nil {
		return ResolvedAPK{}, errors.New("Android artifact is unavailable")
	}
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return ResolvedAPK{}, errors.New("Android artifact is unavailable")
	}
	var artifact deliveryramp.Artifact
	decodeErr := json.NewDecoder(io.LimitReader(manifestFile, 1<<20)).Decode(&artifact)
	closeErr := manifestFile.Close()
	if decodeErr != nil || closeErr != nil {
		return ResolvedAPK{}, errors.New("Android artifact manifest is invalid")
	}
	expectedKind := "android-debug-apk-aab"
	if kind == "android-release" {
		expectedKind = "android-release-apk-aab"
	}
	if artifact.ImmutableRef != immutableRef || artifact.Checksum != "sha256:"+digest || artifact.Kind != expectedKind || artifact.SizeBytes <= 0 || artifact.SizeBytes > maxRetainedArtifactBytes || artifact.CreatedAt.IsZero() {
		return ResolvedAPK{}, errors.New("Android artifact identity is invalid")
	}
	packageName := strings.TrimSpace(artifact.Metadata["package_name"])
	versionName := strings.TrimSpace(artifact.Metadata["version_name"])
	versionCode, versionErr := parsePositiveInt64(artifact.Metadata["version_code"])
	if !androidPackageName.MatchString(packageName) || versionName == "" || len(versionName) > 128 || versionErr != nil {
		return ResolvedAPK{}, errors.New("Android artifact package metadata is invalid")
	}
	apkPath := filepath.Join(root, "app.apk")
	aabPath := filepath.Join(root, "app.aab")
	if err := requireRegularFile(apkPath); err != nil {
		return ResolvedAPK{}, errors.New("Android APK is unavailable")
	}
	if err := requireRegularFile(aabPath); err != nil {
		return ResolvedAPK{}, errors.New("Android AAB is unavailable")
	}
	apk, err := os.Open(apkPath)
	if err != nil {
		return ResolvedAPK{}, errors.New("Android APK is unavailable")
	}
	apkInfo, err := os.Lstat(apkPath)
	apkOpenInfo, statErr := apk.Stat()
	if err != nil || statErr != nil || !apkInfo.Mode().IsRegular() || !os.SameFile(apkInfo, apkOpenInfo) || apkInfo.Size() <= 0 || apkInfo.Size() >= artifact.SizeBytes {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK size is invalid")
	}
	aabInfo, err := os.Lstat(aabPath)
	if err != nil || !aabInfo.Mode().IsRegular() || aabInfo.Size() <= 0 || apkInfo.Size()+aabInfo.Size() != artifact.SizeBytes {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android artifact size is invalid")
	}
	apkHasher := sha256.New()
	fullHasher := sha256.New()
	if _, err := io.Copy(apkHasher, apk); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK verification failed")
	}
	if _, err := apk.Seek(0, io.SeekStart); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK could not be rewound for identity verification")
	}
	if _, err := io.Copy(fullHasher, apk); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK verification failed")
	}
	_, _ = fullHasher.Write([]byte{0})
	aab, err := os.Open(aabPath)
	if err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android AAB is unavailable")
	}
	aabOpenInfo, statErr := aab.Stat()
	if statErr != nil || !os.SameFile(aabInfo, aabOpenInfo) {
		_ = aab.Close()
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android AAB changed during resolution")
	}
	if _, err := io.Copy(fullHasher, aab); err != nil {
		_ = aab.Close()
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android AAB verification failed")
	}
	if err := aab.Close(); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android AAB verification failed")
	}
	if hex.EncodeToString(fullHasher.Sum(nil)) != digest {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android artifact digest mismatch")
	}
	if _, err := apk.Seek(0, io.SeekStart); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK could not be rewound")
	}
	return ResolvedAPK{
		Descriptor: APKDescriptor{
			ImmutableRef: artifact.ImmutableRef,
			Kind:         artifact.Kind,
			Checksum:     artifact.Checksum,
			APKChecksum:  "sha256:" + hex.EncodeToString(apkHasher.Sum(nil)),
			SizeBytes:    artifact.SizeBytes,
			APKSizeBytes: apkInfo.Size(),
			PackageName:  packageName,
			VersionName:  versionName,
			VersionCode:  versionCode,
			CreatedAt:    artifact.CreatedAt,
		},
		File: apk,
	}, nil
}

func requireRegularDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("not a regular directory")
	}
	return nil
}

func requireRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 {
		return fmt.Errorf("not a regular file")
	}
	return nil
}

func parsePositiveInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 || strconv.FormatInt(parsed, 10) != value {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return parsed, nil
}
