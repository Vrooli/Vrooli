// Package builds owns deterministic Linux-side iOS shell generation and the
// Apple-hosted xcodebuild boundary.
package builds

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const MinimumSDK = 26

// RunCommand is injectable for build tests.
type RunCommand func(context.Context, string, ...string) ([]byte, error)

// Builder generates a complete Capacitor-compatible iOS project before
// delegating Apple-only compilation to xcodebuild on a qualifying host.
type Builder struct {
	BuildRoot string
	GOOS      string
	Run       RunCommand
	Now       func() time.Time
}

// GeneratedProject describes deterministic source output and contains no
// credentials or signing material.
type GeneratedProject struct {
	ProjectPath string    `json:"project_path"`
	SourceRef   string    `json:"source_ref"`
	BundleID    string    `json:"bundle_id"`
	SDK         int       `json:"sdk"`
	GeneratedAt time.Time `json:"generated_at"`
}

// UnavailableError identifies a missing Apple capability and its recovery.
type UnavailableError struct {
	MissingCapability string
	NextAction        string
}

func (e UnavailableError) Error() string {
	return "iOS build unavailable: " + e.MissingCapability + "; next action: " + e.NextAction
}

var _ deliveryramp.Builder = Builder{}

// Generate renders a deterministic Capacitor iOS project on any host.
func (b Builder) Generate(ctx context.Context, request deliveryramp.BuildRequest) (GeneratedProject, error) {
	if ctx == nil {
		return GeneratedProject{}, fmt.Errorf("iOS generation context is required")
	}
	if strings.TrimSpace(request.SourceRef) == "" {
		return GeneratedProject{}, fmt.Errorf("iOS source reference is required")
	}
	info, err := os.Stat(request.SourceRef)
	if err != nil || !info.IsDir() {
		return GeneratedProject{}, fmt.Errorf("iOS web bundle %q is unavailable", request.SourceRef)
	}
	bundleDigest, err := digest(request.SourceRef)
	if err != nil {
		return GeneratedProject{}, fmt.Errorf("fingerprint iOS web bundle: %w", err)
	}
	bundleID := strings.TrimSpace(request.Parameters["bundle_id"])
	if bundleID == "" {
		bundleID = "com.vrooli.generated.app"
	}
	if !validBundleID(bundleID) {
		return GeneratedProject{}, fmt.Errorf("invalid iOS bundle id %q", bundleID)
	}
	sdk := MinimumSDK
	if raw := strings.TrimSpace(request.Parameters["ios_sdk"]); raw != "" {
		sdk, err = strconv.Atoi(raw)
		if err != nil || sdk < MinimumSDK {
			return GeneratedProject{}, fmt.Errorf("iOS SDK must be at least %d", MinimumSDK)
		}
	}
	root := b.BuildRoot
	if root == "" {
		root = filepath.Join(os.TempDir(), "vrooli-ios-builds")
	}
	hash := sha256.Sum256([]byte(bundleDigest + "\x00" + bundleID + "\x00" + strconv.Itoa(sdk)))
	projectRoot := filepath.Join(root, hex.EncodeToString(hash[:8]))
	if err := render(projectRoot, request.SourceRef, bundleID, sdk); err != nil {
		return GeneratedProject{}, err
	}
	now := time.Now
	if b.Now != nil {
		now = b.Now
	}
	return GeneratedProject{ProjectPath: projectRoot, SourceRef: request.SourceRef, BundleID: bundleID, SDK: sdk, GeneratedAt: now().UTC()}, nil
}

// Build generates the project and reports an explicit unavailable error until
// a macOS node provides xcodebuild. It never fabricates an Apple artifact.
func (b Builder) Build(ctx context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	generated, err := b.Generate(ctx, request)
	if err != nil {
		return deliveryramp.Artifact{}, err
	}
	goos := b.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos != "darwin" {
		return deliveryramp.Artifact{}, UnavailableError{MissingCapability: "macOS xcodebuild host", NextAction: "register a macOS bridge node and run the generated project there"}
	}
	run := b.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		}
	}
	if _, err := run(ctx, "xcodebuild", "-project", filepath.Join(generated.ProjectPath, "ios", "App.xcodeproj"), "-scheme", "App", "-sdk", "iphoneos", "build"); err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("xcodebuild iOS project: %w", err)
	}
	return deliveryramp.Artifact{ImmutableRef: "ios-build:" + generated.BundleID, Kind: "ipa", LocalPath: filepath.Join(generated.ProjectPath, "ios", "build", "App.ipa"), Metadata: map[string]string{"sdk": strconv.Itoa(generated.SDK), "bundle_id": generated.BundleID}, CreatedAt: time.Now().UTC()}, nil
}

func render(root, webRoot, bundleID string, sdk int) error {
	if err := os.MkdirAll(filepath.Join(root, "ios", "App.xcodeproj"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "ios", "App"), 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"capacitor.config.json":             fmt.Sprintf("{\n  \"appId\": %q,\n  \"appName\": \"Vrooli App\",\n  \"webDir\": \"public\"\n}\n", bundleID),
		"ios/Podfile":                       "platform :ios, '26.0'\nproject 'App/App.xcodeproj'\n\ntarget 'App' do\n  # Capacitor dependencies are resolved on the macOS build host.\nend\n",
		"ios/App/Info.plist":                fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\"><dict><key>CFBundleIdentifier</key><string>%s</string><key>MinimumOSVersion</key><string>%d.0</string></dict></plist>\n", bundleID, sdk),
		"ios/App/App.entitlements":          "<?xml version=\"1.0\" encoding=\"UTF-8\"?><plist version=\"1.0\"><dict></dict></plist>\n",
		"ios/App.xcodeproj/project.pbxproj": fmt.Sprintf("// Deterministic Vrooli Capacitor iOS project\n// IPHONEOS_DEPLOYMENT_TARGET = %d.0\n// PRODUCT_BUNDLE_IDENTIFIER = %s\n", sdk, bundleID),
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(files[path]), 0o644); err != nil {
			return err
		}
	}
	public := filepath.Join(root, "public")
	if err := copyDir(public, webRoot); err != nil {
		return fmt.Errorf("copy iOS web bundle: %w", err)
	}
	return nil
}

func copyDir(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func digest(root string) (string, error) {
	h := sha256.New()
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			paths = append(paths, rel)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(rel))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validBundleID(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}
