package builds

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const maxRetainedArtifactBytes int64 = 1 << 30

var (
	immutableAndroidRef    = regexp.MustCompile(`^(android-debug|android-release):([0-9a-f]{64})$`)
	immutableAndroidDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
	androidPackageName     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
)

// APKDescriptor is the path-free owner contract consumed by Device Control.
// ImmutableRef binds the retained APK and AAB pair; APKChecksum lets the
// consumer independently verify the exact bytes it will install.
type APKDescriptor struct {
	ImmutableRef    string    `json:"immutable_ref"`
	Kind            string    `json:"kind"`
	Checksum        string    `json:"checksum"`
	APKChecksum     string    `json:"apk_checksum"`
	SizeBytes       int64     `json:"size_bytes"`
	APKSizeBytes    int64     `json:"apk_size_bytes"`
	PackageName     string    `json:"package_name"`
	VersionName     string    `json:"version_name"`
	VersionCode     int64     `json:"version_code"`
	Signing         string    `json:"signing"`
	SigningIdentity string    `json:"signing_identity,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ResolvedAPK holds the verified owner descriptor and an open handle to the
// retained APK. The caller must close File after the response is written.
type ResolvedAPK struct {
	Descriptor APKDescriptor
	File       *os.File
	Path       string
}

// ArtifactCandidate is a path-free summary of one retained build. Listing
// reads metadata and file sizes only; ResolveAPK performs the full content
// integrity check before a matrix can use the artifact.
type ArtifactCandidate struct {
	ImmutableRef    string    `json:"immutable_ref"`
	Kind            string    `json:"kind"`
	Checksum        string    `json:"checksum"`
	SizeBytes       int64     `json:"size_bytes"`
	APKSizeBytes    int64     `json:"apk_size_bytes"`
	PackageName     string    `json:"package_name"`
	VersionName     string    `json:"version_name"`
	VersionCode     int64     `json:"version_code"`
	Signing         string    `json:"signing"`
	SigningIdentity string    `json:"signing_identity,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ListArtifactCandidates returns recent retained Android builds without
// exposing paths. It validates manifest identity and file shape, while the
// selected artifact is content-verified by ResolveAPK before use.
func ListArtifactCandidates(dataRoot string, limit int) ([]ArtifactCandidate, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return nil, errors.New("Android artifact storage is unavailable")
	}
	if limit < 1 || limit > 100 {
		return nil, errors.New("Android artifact list limit must be between 1 and 100")
	}
	parent := filepath.Join(dataRoot, "builds", "artifacts")
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return []ArtifactCandidate{}, nil
	}
	if err != nil {
		return nil, err
	}
	candidates := make([]ArtifactCandidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !immutableAndroidDigest.MatchString(entry.Name()) {
			continue
		}
		digest := entry.Name()
		root := filepath.Join(parent, digest)
		manifestPath := filepath.Join(root, "artifact.json")
		manifestInfo, err := os.Lstat(manifestPath)
		if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Size() <= 0 || manifestInfo.Size() > 1<<20 {
			continue
		}
		manifestFile, err := os.Open(manifestPath)
		if err != nil {
			continue
		}
		var artifact deliveryramp.Artifact
		decodeErr := json.NewDecoder(io.LimitReader(manifestFile, 1<<20)).Decode(&artifact)
		closeErr := manifestFile.Close()
		if decodeErr != nil || closeErr != nil {
			continue
		}
		match := immutableAndroidRef.FindStringSubmatch(artifact.ImmutableRef)
		if match == nil || match[2] != digest || artifact.Checksum != "sha256:"+digest || artifact.SizeBytes <= 0 || artifact.SizeBytes > maxRetainedArtifactBytes || artifact.CreatedAt.IsZero() {
			continue
		}
		expectedKind := "android-debug-apk-aab"
		if match[1] == "android-release" {
			expectedKind = "android-release-apk-aab"
		}
		if artifact.Kind != expectedKind {
			continue
		}
		apkInfo, apkErr := os.Lstat(filepath.Join(root, "app.apk"))
		aabInfo, aabErr := os.Lstat(filepath.Join(root, "app.aab"))
		if apkErr != nil || aabErr != nil || !apkInfo.Mode().IsRegular() || !aabInfo.Mode().IsRegular() || apkInfo.Size() <= 0 || aabInfo.Size() <= 0 || apkInfo.Size()+aabInfo.Size() != artifact.SizeBytes {
			continue
		}
		packageName := strings.TrimSpace(artifact.Metadata["package_name"])
		versionName := strings.TrimSpace(artifact.Metadata["version_name"])
		versionCode, versionErr := parsePositiveInt64(artifact.Metadata["version_code"])
		signing := strings.TrimSpace(artifact.Metadata["signing"])
		signingIdentity := strings.TrimSpace(artifact.Metadata["signing_identity"])
		if !androidPackageName.MatchString(packageName) || versionName == "" || versionErr != nil || signing == "" || (match[1] == "android-release" && signingIdentity == "") {
			continue
		}
		candidates = append(candidates, ArtifactCandidate{
			ImmutableRef: artifact.ImmutableRef, Kind: artifact.Kind, Checksum: artifact.Checksum,
			SizeBytes: artifact.SizeBytes, APKSizeBytes: apkInfo.Size(), PackageName: packageName,
			VersionName: versionName, VersionCode: versionCode, Signing: signing,
			SigningIdentity: signingIdentity, CreatedAt: artifact.CreatedAt,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.After(candidates[j].CreatedAt) })
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
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
	signing := strings.TrimSpace(artifact.Metadata["signing"])
	signingIdentity := strings.TrimSpace(artifact.Metadata["signing_identity"])
	if !androidPackageName.MatchString(packageName) || versionName == "" || len(versionName) > 128 || versionErr != nil {
		return ResolvedAPK{}, errors.New("Android artifact package metadata is invalid")
	}
	if signing == "" || (kind == "android-release" && signingIdentity == "") {
		return ResolvedAPK{}, errors.New("Android artifact signing metadata is invalid")
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
	artifactSigner, signerErr := inspectVerifiedAPKSigningCertificate(apk, apkHasher.Sum(nil))
	if signerErr != nil || !validSHA256Digest(artifact.Metadata["signing_certificate_sha256"]) || !strings.EqualFold(artifactSigner, artifact.Metadata["signing_certificate_sha256"]) {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android artifact signing metadata does not match the retained APK")
	}
	if _, err := apk.Seek(0, io.SeekStart); err != nil {
		_ = apk.Close()
		return ResolvedAPK{}, errors.New("Android APK could not be rewound")
	}
	return ResolvedAPK{
		Descriptor: APKDescriptor{
			ImmutableRef:    artifact.ImmutableRef,
			Kind:            artifact.Kind,
			Checksum:        artifact.Checksum,
			APKChecksum:     "sha256:" + hex.EncodeToString(apkHasher.Sum(nil)),
			SizeBytes:       artifact.SizeBytes,
			APKSizeBytes:    apkInfo.Size(),
			PackageName:     packageName,
			VersionName:     versionName,
			VersionCode:     versionCode,
			Signing:         signing,
			SigningIdentity: signingIdentity,
			CreatedAt:       artifact.CreatedAt,
		},
		File: apk,
		Path: apkPath,
	}, nil
}

// inspectVerifiedAPKSigningCertificate binds signer inspection to the exact
// bytes already verified by ResolveAPK. The private copy is hashed again
// before apksigner reads it, so pathname replacement cannot swap the signer
// input independently of the open verified artifact.
func inspectVerifiedAPKSigningCertificate(apk *os.File, expectedSHA256 []byte) (string, error) {
	if _, err := apk.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind verified APK for signer inspection: %w", err)
	}
	temporary, err := os.CreateTemp("", "vrooli-verified-apk-*.apk")
	if err != nil {
		return "", fmt.Errorf("create private APK signer-inspection copy: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("protect APK signer-inspection copy: %w", err)
	}
	copyHasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temporary, copyHasher), apk); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("copy verified APK for signer inspection: %w", err)
	}
	if !bytes.Equal(copyHasher.Sum(nil), expectedSHA256) {
		_ = temporary.Close()
		return "", errors.New("verified APK changed before signer inspection")
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close APK signer-inspection copy: %w", err)
	}
	return inspectAPKSigningCertificate(context.Background(), temporaryPath)
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
