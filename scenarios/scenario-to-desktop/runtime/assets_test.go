package bundleruntime

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

func TestEnsureAssets(t *testing.T) {
	t.Run("verifies all assets exist", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		// Create asset files
		asset1Content := []byte("asset1 content")
		asset2Content := []byte("asset2 content")
		asset1Path := filepath.Join(bundleDir, "assets", "file1.txt")
		asset2Path := filepath.Join(bundleDir, "assets", "file2.txt")

		if err := os.MkdirAll(filepath.Dir(asset1Path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(asset1Path, asset1Content, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(asset2Path, asset2Content, 0644); err != nil {
			t.Fatal(err)
		}

		svc := manifest.Service{
			ID: "test-svc",
			Assets: []manifest.Asset{
				{Path: "assets/file1.txt"},
				{Path: "assets/file2.txt"},
			},
		}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.ensureAssets(svc)
		if err != nil {
			t.Errorf("ensureAssets() error = %v, want nil", err)
		}
	})

	t.Run("returns error for missing asset", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		svc := manifest.Service{
			ID: "test-svc",
			Assets: []manifest.Asset{
				{Path: "nonexistent/file.txt"},
			},
		}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.ensureAssets(svc)
		if err == nil {
			t.Error("ensureAssets() should return error for missing asset")
		}
	})
}

func TestVerifyAsset(t *testing.T) {
	t.Run("accepts valid asset", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		content := []byte("test content")
		assetPath := filepath.Join(bundleDir, "test.txt")
		if err := os.WriteFile(assetPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		svc := manifest.Service{ID: "test-svc"}
		asset := manifest.Asset{Path: "test.txt"}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.verifyAsset(svc, asset)
		if err != nil {
			t.Errorf("verifyAsset() error = %v, want nil", err)
		}
	})

	t.Run("rejects directory as asset", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		dirPath := filepath.Join(bundleDir, "somedir")
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		svc := manifest.Service{ID: "test-svc"}
		asset := manifest.Asset{Path: "somedir"}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.verifyAsset(svc, asset)
		if err == nil {
			t.Error("verifyAsset() should reject directory")
		}
	})
}

func TestVerifyAssetChecksum(t *testing.T) {
	t.Run("accepts matching checksum", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		content := []byte("checksummed content")
		assetPath := filepath.Join(bundleDir, "checksummed.txt")
		if err := os.WriteFile(assetPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))

		svc := manifest.Service{ID: "test-svc"}
		asset := manifest.Asset{Path: "checksummed.txt", SHA256: expectedHash}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.verifyAssetChecksum(svc, asset, assetPath)
		if err != nil {
			t.Errorf("verifyAssetChecksum() error = %v, want nil", err)
		}
	})

	t.Run("rejects mismatched checksum", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		content := []byte("actual content")
		assetPath := filepath.Join(bundleDir, "checksummed.txt")
		if err := os.WriteFile(assetPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		wrongHash := fmt.Sprintf("%x", sha256.Sum256([]byte("different content")))

		svc := manifest.Service{ID: "test-svc"}
		asset := manifest.Asset{Path: "checksummed.txt", SHA256: wrongHash}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.verifyAssetChecksum(svc, asset, assetPath)
		if err == nil {
			t.Error("verifyAssetChecksum() should reject mismatched checksum")
		}
	})

	t.Run("accepts case-insensitive hash comparison", func(t *testing.T) {
		bundleDir := t.TempDir()
		appData := t.TempDir()

		content := []byte("test")
		assetPath := filepath.Join(bundleDir, "test.txt")
		if err := os.WriteFile(assetPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256(content)
		upperHash := fmt.Sprintf("%X", hash) // uppercase

		svc := manifest.Service{ID: "test-svc"}
		asset := manifest.Asset{Path: "test.txt", SHA256: upperHash}

		s := &Supervisor{
			opts:          Options{BundlePath: bundleDir},
			appData:       appData,
			fs:            RealFileSystem{},
			clock:         &fakeClock{now: time.Now()},
			telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
		}

		err := s.verifyAssetChecksum(svc, asset, assetPath)
		if err != nil {
			t.Errorf("verifyAssetChecksum() should accept case-insensitive hash, error = %v", err)
		}
	})
}

func TestCheckAssetSizeBudget(t *testing.T) {
	bundleDir := t.TempDir()
	appData := t.TempDir()

	s := &Supervisor{
		opts:          Options{BundlePath: bundleDir},
		appData:       appData,
		fs:            RealFileSystem{},
		clock:         &fakeClock{now: time.Now()},
		telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
	}

	svc := manifest.Service{ID: "test-svc"}

	t.Run("accepts exact size", func(t *testing.T) {
		asset := manifest.Asset{Path: "test.bin", SizeBytes: 1000}
		err := s.checkAssetSizeBudget(svc, asset, 1000)
		if err != nil {
			t.Errorf("checkAssetSizeBudget() error = %v, want nil for exact size", err)
		}
	})

	t.Run("accepts slightly larger with slack", func(t *testing.T) {
		// 10MB expected, 10.5MB actual (5% overage)
		asset := manifest.Asset{Path: "test.bin", SizeBytes: 10 * 1024 * 1024}
		actual := int64(10.5 * 1024 * 1024)
		err := s.checkAssetSizeBudget(svc, asset, actual)
		if err != nil {
			t.Errorf("checkAssetSizeBudget() error = %v, want nil for within slack", err)
		}
	})

	t.Run("rejects significantly oversized", func(t *testing.T) {
		// 10MB expected, 15MB actual (50% overage)
		asset := manifest.Asset{Path: "test.bin", SizeBytes: 10 * 1024 * 1024}
		actual := int64(15 * 1024 * 1024)
		err := s.checkAssetSizeBudget(svc, asset, actual)
		if err == nil {
			t.Error("checkAssetSizeBudget() should reject significantly oversized asset")
		}
	})

	t.Run("rejects suspiciously small", func(t *testing.T) {
		// 10MB expected, 2MB actual (80% smaller)
		asset := manifest.Asset{Path: "test.bin", SizeBytes: 10 * 1024 * 1024}
		actual := int64(2 * 1024 * 1024)
		err := s.checkAssetSizeBudget(svc, asset, actual)
		if err == nil {
			t.Error("checkAssetSizeBudget() should reject suspiciously small asset")
		}
	})

	t.Run("accepts at half size boundary", func(t *testing.T) {
		// 10MB expected, 5MB actual (exactly half)
		asset := manifest.Asset{Path: "test.bin", SizeBytes: 10 * 1024 * 1024}
		actual := int64(5 * 1024 * 1024)
		err := s.checkAssetSizeBudget(svc, asset, actual)
		if err != nil {
			t.Errorf("checkAssetSizeBudget() error = %v, want nil for exactly half", err)
		}
	})
}

func TestSizeBudgetSlack(t *testing.T) {
	t.Run("returns minimum 1MB for small files", func(t *testing.T) {
		// 100KB expected -> should get 1MB slack (minimum)
		expected := int64(100 * 1024)
		slack := sizeBudgetSlack(expected)
		minSlack := int64(1 * 1024 * 1024)
		if slack != minSlack {
			t.Errorf("sizeBudgetSlack(%d) = %d, want %d (1MB minimum)", expected, slack, minSlack)
		}
	})

	t.Run("returns 5% for large files", func(t *testing.T) {
		// 100MB expected -> should get 5MB slack (5%)
		expected := int64(100 * 1024 * 1024)
		slack := sizeBudgetSlack(expected)
		expectedSlack := expected / 20
		if slack != expectedSlack {
			t.Errorf("sizeBudgetSlack(%d) = %d, want %d (5%%)", expected, slack, expectedSlack)
		}
	})

	t.Run("returns 0 for zero expected", func(t *testing.T) {
		slack := sizeBudgetSlack(0)
		if slack != 0 {
			t.Errorf("sizeBudgetSlack(0) = %d, want 0", slack)
		}
	})

	t.Run("returns 0 for negative expected", func(t *testing.T) {
		slack := sizeBudgetSlack(-100)
		if slack != 0 {
			t.Errorf("sizeBudgetSlack(-100) = %d, want 0", slack)
		}
	})

	t.Run("threshold between minimum and percentage", func(t *testing.T) {
		// 20MB expected -> 5% is 1MB, which equals the minimum
		// Anything above 20MB should use percentage
		expected := int64(20 * 1024 * 1024)
		slack := sizeBudgetSlack(expected)
		minSlack := int64(1 * 1024 * 1024)
		if slack != minSlack {
			t.Errorf("sizeBudgetSlack(%d) = %d, want %d (at threshold)", expected, slack, minSlack)
		}

		// 25MB expected -> 5% is 1.25MB, should use percentage
		expected = int64(25 * 1024 * 1024)
		slack = sizeBudgetSlack(expected)
		expectedSlack := expected / 20
		if slack != expectedSlack {
			t.Errorf("sizeBudgetSlack(%d) = %d, want %d (above threshold)", expected, slack, expectedSlack)
		}
	})
}

func TestVerifyAssetWithSizeAndChecksum(t *testing.T) {
	bundleDir := t.TempDir()
	appData := t.TempDir()

	content := []byte("complete asset verification test content")
	assetPath := filepath.Join(bundleDir, "verified.bin")
	if err := os.WriteFile(assetPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))

	svc := manifest.Service{ID: "test-svc"}
	asset := manifest.Asset{
		Path:      "verified.bin",
		SizeBytes: int64(len(content)),
		SHA256:    expectedHash,
	}

	s := &Supervisor{
		opts:          Options{BundlePath: bundleDir},
		appData:       appData,
		fs:            RealFileSystem{},
		clock:         &fakeClock{now: time.Now()},
		telemetryPath: filepath.Join(appData, "telemetry.jsonl"),
	}

	err := s.verifyAsset(svc, asset)
	if err != nil {
		t.Errorf("verifyAsset() with size and checksum error = %v, want nil", err)
	}
}
