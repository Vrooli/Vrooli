package main

// Example: File-tools integration for Go-based scenarios
// Shows how to replace custom file operations with file-tools API calls
// Target scenarios: data-backup-manager, document-manager, crypto-tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Configuration
var (
	FileToolsAPIURL = getEnv("FILE_TOOLS_API_URL", "http://localhost:15458")
)

// =============================================================================
// Data structures matching file-tools API contracts
// =============================================================================

type CompressRequest struct {
	Files            []string         `json:"files"`
	ArchiveFormat    string           `json:"archive_format"`
	OutputPath       string           `json:"output_path"`
	CompressionLevel int              `json:"compression_level,omitempty"`
	Options          *CompressOptions `json:"options,omitempty"`
}

type CompressOptions struct {
	PreservePermissions bool     `json:"preserve_permissions"`
	ExcludePatterns     []string `json:"exclude_patterns,omitempty"`
	Encrypt             bool     `json:"encrypt,omitempty"`
	Password            string   `json:"password,omitempty"`
}

type CompressResponse struct {
	OperationID         string  `json:"operation_id"`
	ArchivePath         string  `json:"archive_path"`
	OriginalSizeBytes   int64   `json:"original_size_bytes"`
	CompressedSizeBytes int64   `json:"compressed_size_bytes"`
	CompressionRatio    float64 `json:"compression_ratio"`
	FilesIncluded       int     `json:"files_included"`
	Checksum            string  `json:"checksum"`
}

type ChecksumRequest struct {
	Files     []string `json:"files"`
	Algorithm string   `json:"algorithm"`
}

type ChecksumResponse struct {
	Results []struct {
		FilePath  string            `json:"file_path"`
		Checksums map[string]string `json:"checksums"`
	} `json:"results"`
}

type DuplicateDetectRequest struct {
	ScanPaths       []string                `json:"scan_paths"`
	DetectionMethod string                  `json:"detection_method"`
	Options         *DuplicateDetectOptions `json:"options,omitempty"`
}

type DuplicateDetectOptions struct {
	SimilarityThreshold float64  `json:"similarity_threshold"`
	IncludeHidden       bool     `json:"include_hidden"`
	FileSizeMin         int64    `json:"file_size_min,omitempty"`
	FileSizeMax         int64    `json:"file_size_max,omitempty"`
	FileExtensions      []string `json:"file_extensions,omitempty"`
}

type DuplicateDetectResponse struct {
	ScanID            string           `json:"scan_id"`
	DuplicateGroups   []DuplicateGroup `json:"duplicate_groups"`
	TotalDuplicates   int              `json:"total_duplicates"`
	TotalSavingsBytes int64            `json:"total_savings_bytes"`
}

type DuplicateGroup struct {
	GroupID               string     `json:"group_id"`
	SimilarityScore       float64    `json:"similarity_score"`
	Files                 []FileInfo `json:"files"`
	PotentialSavingsBytes int64      `json:"potential_savings_bytes"`
}

type FileInfo struct {
	Path         string    `json:"path"`
	SizeBytes    int64     `json:"size_bytes"`
	Checksum     string    `json:"checksum"`
	LastModified time.Time `json:"last_modified"`
}

// =============================================================================
// BEFORE: Custom tar-based compression
// =============================================================================

func CompressWithTarOld(sourcePath, outputPath string) error {
	// Old approach: Shell out to tar command
	// Problems:
	// - No error recovery
	// - No progress tracking
	// - No integrity verification
	// - Manual checksum calculation required
	// - Platform-specific behavior

	fmt.Println("❌ OLD: Using tar command directly")
	// exec.Command("tar", "-czf", outputPath, sourcePath).Run()
	return fmt.Errorf("deprecated: use CompressWithFileTools instead")
}

// =============================================================================
// AFTER: Using file-tools API
// =============================================================================

// CompressWithFileTools compresses files using file-tools API
func CompressWithFileTools(files []string, outputPath string, format string) (*CompressResponse, error) {
	fmt.Println("✅ NEW: Using file-tools API")

	req := CompressRequest{
		Files:            files,
		ArchiveFormat:    format,
		OutputPath:       outputPath,
		CompressionLevel: 6,
		Options: &CompressOptions{
			PreservePermissions: true,
			ExcludePatterns:     []string{".git", "node_modules", "*.tmp"},
		},
	}

	var resp CompressResponse
	if err := callFileToolsAPI("/api/v1/files/compress", req, &resp); err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	fmt.Printf("📦 Compression complete:\n")
	fmt.Printf("   Operation ID: %s\n", resp.OperationID)
	fmt.Printf("   Original: %d bytes\n", resp.OriginalSizeBytes)
	fmt.Printf("   Compressed: %d bytes\n", resp.CompressedSizeBytes)
	fmt.Printf("   Ratio: %.2fx\n", resp.CompressionRatio)
	fmt.Printf("   Checksum: %s\n", resp.Checksum)

	return &resp, nil
}

// VerifyChecksum verifies file integrity using file-tools
func VerifyChecksum(filePath string, expectedChecksum string, algorithm string) (bool, error) {
	fmt.Println("🔐 Verifying archive integrity...")

	req := ChecksumRequest{
		Files:     []string{filePath},
		Algorithm: algorithm,
	}

	var resp ChecksumResponse
	if err := callFileToolsAPI("/api/v1/files/checksum", req, &resp); err != nil {
		return false, fmt.Errorf("checksum calculation failed: %w", err)
	}

	if len(resp.Results) == 0 {
		return false, fmt.Errorf("no checksum results returned")
	}

	actualChecksum := resp.Results[0].Checksums[algorithm]
	match := actualChecksum == expectedChecksum

	if match {
		fmt.Println("✅ Checksum verified: Archive integrity confirmed")
	} else {
		fmt.Println("❌ Checksum mismatch: Archive may be corrupted")
	}

	return match, nil
}

// DetectDuplicates finds duplicate files using content hashing
func DetectDuplicates(scanPaths []string, extensions []string) (*DuplicateDetectResponse, error) {
	fmt.Println("🔍 Scanning for duplicate files...")

	req := DuplicateDetectRequest{
		ScanPaths:       scanPaths,
		DetectionMethod: "hash",
		Options: &DuplicateDetectOptions{
			SimilarityThreshold: 1.0,
			IncludeHidden:       false,
			FileExtensions:      extensions,
			FileSizeMin:         1024, // Ignore files < 1KB
		},
	}

	var resp DuplicateDetectResponse
	if err := callFileToolsAPI("/api/v1/files/duplicates/detect", req, &resp); err != nil {
		return nil, fmt.Errorf("duplicate detection failed: %w", err)
	}

	fmt.Printf("📊 Duplicate Detection Results:\n")
	fmt.Printf("   Scan ID: %s\n", resp.ScanID)
	fmt.Printf("   Duplicate files: %d\n", resp.TotalDuplicates)
	fmt.Printf("   Duplicate groups: %d\n", len(resp.DuplicateGroups))
	fmt.Printf("   Potential savings: %d bytes\n", resp.TotalSavingsBytes)

	return &resp, nil
}

// =============================================================================
// HTTP client helper
// =============================================================================

func callFileToolsAPI(endpoint string, request interface{}, response interface{}) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := FileToolsAPIURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// =============================================================================
// Example usage patterns
// =============================================================================

// ExampleBackupScenario demonstrates file-tools integration for backup scenarios
func ExampleBackupScenario() {
	// Compress data directory
	compressResp, err := CompressWithFileTools(
		[]string{"/data"},
		"/backups/backup.tar.gz",
		"gzip",
	)
	if err != nil {
		fmt.Printf("Compression failed: %v\n", err)
		return
	}

	// Verify integrity
	verified, err := VerifyChecksum(
		compressResp.ArchivePath,
		compressResp.Checksum,
		"sha256",
	)
	if err != nil || !verified {
		fmt.Printf("Integrity verification failed: %v\n", err)
		return
	}

	fmt.Println("✅ Backup created and verified successfully")
}

// ExamplePhotoManager demonstrates file-tools integration for photo management
func ExamplePhotoManager() {
	// Find duplicate photos
	duplicates, err := DetectDuplicates(
		[]string{"/photos"},
		[]string{"jpg", "jpeg", "png", "heic"},
	)
	if err != nil {
		fmt.Printf("Duplicate detection failed: %v\n", err)
		return
	}

	// Process duplicate groups
	for _, group := range duplicates.DuplicateGroups {
		fmt.Printf("\n📷 Duplicate Group %s (%.2f%% similar):\n",
			group.GroupID, group.SimilarityScore*100)
		for _, file := range group.Files {
			fmt.Printf("   - %s (%d bytes)\n", file.Path, file.SizeBytes)
		}
		fmt.Printf("   Potential savings: %d bytes\n", group.PotentialSavingsBytes)
	}
}

// =============================================================================
// Utility functions
// =============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// =============================================================================
// Main (for demonstration)
// =============================================================================

func main() {
	fmt.Println("🎯 File-Tools Go Integration Examples")
	fmt.Println("=====================================\n")

	// Check if file-tools API is available
	resp, err := http.Get(FileToolsAPIURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️  File-tools API not available at %s\n", FileToolsAPIURL)
		fmt.Println("   Make sure file-tools scenario is running:")
		fmt.Println("   $ vrooli scenario start file-tools")
		return
	}

	fmt.Printf("✅ File-tools API available at %s\n\n", FileToolsAPIURL)

	// Run examples
	fmt.Println("Example 1: Backup Scenario")
	fmt.Println("--------------------------")
	ExampleBackupScenario()

	fmt.Println("\n\nExample 2: Photo Manager Scenario")
	fmt.Println("----------------------------------")
	ExamplePhotoManager()
}
