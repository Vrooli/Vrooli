package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecretScanner handles resource scanning for secret discovery
type SecretScanner struct {
	db     *sql.DB
	logger Logger
}

// NewSecretScanner creates a new secret scanner
func NewSecretScanner(db *sql.DB) *SecretScanner {
	return &SecretScanner{
		db:     db,
		logger: *NewLogger("scanner"),
	}
}

// ScanConfig represents the configuration for different scan types
type ScanConfig struct {
	MaxResources   int      `json:"max_resources"`
	ScanDepth      string   `json:"scan_depth"`
	FileExtensions []string `json:"file_extensions"`
}

// SecretPattern represents a pattern for detecting secrets
type SecretPattern struct {
	Pattern     string `json:"pattern"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// getScanConfig returns the appropriate scan configuration
func (s *SecretScanner) getScanConfig(scanType string) ScanConfig {
	configs := map[string]ScanConfig{
		"quick": {
			MaxResources:   5,
			ScanDepth:      "shallow",
			FileExtensions: []string{".sh", ".yml", ".yaml", ".env"},
		},
		"full": {
			MaxResources:   50,
			ScanDepth:      "deep",
			FileExtensions: []string{".sh", ".bash", ".yml", ".yaml", ".json", ".env", ".conf", ".config", ".go", ".js", ".py"},
		},
		"targeted": {
			MaxResources:   10,
			ScanDepth:      "medium",
			FileExtensions: []string{".sh", ".yml", ".yaml", ".env", ".json"},
		},
	}

	if config, exists := configs[scanType]; exists {
		return config
	}
	return configs["full"] // Default to full scan
}

// getSecretPatterns returns patterns for detecting secrets
func (s *SecretScanner) getSecretPatterns() []SecretPattern {
	return []SecretPattern{
		// Environment variables
		{Pattern: `\$\{([A-Z_]+[A-Z0-9_]*)\}`, Type: "env_var", Description: "Environment variable reference ${VAR}"},
		{Pattern: `\$([A-Z_]+[A-Z0-9_]*)`, Type: "env_var", Description: "Environment variable reference $VAR"},
		{Pattern: `([A-Z_]+[A-Z0-9_]*)=`, Type: "env_var", Description: "Environment variable assignment"},

		// Function calls
		{Pattern: `getenv\("([A-Z_]+[A-Z0-9_]*)"\)`, Type: "env_var", Description: "getenv() call"},
		{Pattern: `os\.Getenv\("([A-Z_]+[A-Z0-9_]*)"\)`, Type: "env_var", Description: "Go os.Getenv() call"},
		{Pattern: `process\.env\.([A-Z_]+[A-Z0-9_]*)`, Type: "env_var", Description: "Node.js process.env access"},

		// Configuration file patterns
		{Pattern: `env\s*:\s*([A-Z_]+[A-Z0-9_]*)`, Type: "env_var", Description: "YAML env field"},
		{Pattern: `"([A-Z_]+[A-Z0-9_]*)"s*:`, Type: "env_var", Description: "JSON environment key"},
	}
}

// getSecretKeywords returns keywords that indicate secrets
func (s *SecretScanner) getSecretKeywords() []string {
	return []string{
		"PASSWORD", "PASS", "PWD", "SECRET", "KEY", "TOKEN", "AUTH", "API",
		"CREDENTIAL", "CERT", "TLS", "SSL", "PRIVATE", "HOST", "PORT",
		"URL", "ADDR", "DATABASE", "DB", "USER", "NAMESPACE",
	}
}

// ScanResources performs a resource scan for secrets
func (s *SecretScanner) ScanResources(request ScanRequest) (*ScanResponse, error) {
	scanID := uuid.New().String()
	startTime := time.Now()

	s.logger.Info(fmt.Sprintf("Starting %s scan (ID: %s)", request.ScanType, scanID))

	// Get scan configuration
	scanConfig := s.getScanConfig(request.ScanType)
	patterns := s.getSecretPatterns()
	keywords := s.getSecretKeywords()

	// Find files to scan
	files, err := s.findResourceFiles(request.Resources, scanConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find resource files: %w", err)
	}

	// Scan files for secrets
	discoveredSecrets := []ResourceSecret{}
	resourcesScanned := map[string]bool{}

	for _, filePath := range files {
		secrets, resourceName, err := s.scanFile(filePath, patterns, keywords)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to scan file %s", filePath), err)
			continue
		}

		discoveredSecrets = append(discoveredSecrets, secrets...)
		if resourceName != "" {
			resourcesScanned[resourceName] = true
		}
	}

	// Convert map to slice
	var scannedResources []string
	for resource := range resourcesScanned {
		scannedResources = append(scannedResources, resource)
	}

	// Store scan results in database
	scanRecord := SecretScan{
		ID:                uuid.New().String(),
		ScanType:          request.ScanType,
		ResourcesScanned:  scannedResources,
		SecretsDiscovered: len(discoveredSecrets),
		ScanDurationMs:    int(time.Since(startTime).Milliseconds()),
		ScanTimestamp:     startTime,
		ScanStatus:        "completed",
	}

	if err := s.storeScanRecord(scanRecord); err != nil {
		s.logger.Error("Failed to store scan record", err)
		// Continue anyway - don't fail the entire scan
	}

	// Store discovered secrets
	for _, secret := range discoveredSecrets {
		if err := s.storeResourceSecret(secret); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to store secret %s", secret.SecretKey), err)
		}
	}

	response := &ScanResponse{
		ScanID:            scanID,
		DiscoveredSecrets: discoveredSecrets,
		ScanDurationMs:    int(time.Since(startTime).Milliseconds()),
		ResourcesScanned:  scannedResources,
	}

	s.logger.Info(fmt.Sprintf("Scan completed: %d secrets found in %d resources", len(discoveredSecrets), len(scannedResources)))

	return response, nil
}

// findResourceFiles finds files to scan based on configuration
func (s *SecretScanner) findResourceFiles(targetResources []string, config ScanConfig) ([]string, error) {
	var files []string

	// Get VROOLI_ROOT or fallback to HOME/Vrooli
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	resourcesDir := filepath.Join(vrooliRoot, "resources")

	// Check if resources directory exists
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("resources directory not found at %s", resourcesDir)
	}

	// Build file extension filter
	extMap := make(map[string]bool)
	for _, ext := range config.FileExtensions {
		extMap[ext] = true
	}

	// Walk through resources directory
	err := filepath.WalkDir(resourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if d.IsDir() {
			return nil
		}

		// Check file extension
		ext := filepath.Ext(path)
		if !extMap[ext] {
			return nil
		}

		// Extract resource name from path
		relPath, err := filepath.Rel(resourcesDir, path)
		if err != nil {
			return nil
		}

		pathParts := strings.Split(relPath, string(filepath.Separator))
		if len(pathParts) == 0 {
			return nil
		}

		resourceName := pathParts[0]

		// Filter by target resources if specified
		if len(targetResources) > 0 {
			found := false
			for _, target := range targetResources {
				if target == resourceName {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk resources directory: %w", err)
	}

	// Limit number of files based on configuration
	if len(files) > config.MaxResources*10 { // Assume ~10 files per resource on average
		files = files[:config.MaxResources*10]
	}

	return files, nil
}

// scanFile scans a single file for secret patterns
func (s *SecretScanner) scanFile(filePath string, patterns []SecretPattern, keywords []string) ([]ResourceSecret, string, error) {
	// Extract resource name from path
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, _ := os.UserHomeDir()
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	resourcesDir := filepath.Join(vrooliRoot, "resources")
	relPath, err := filepath.Rel(resourcesDir, filePath)
	if err != nil {
		return nil, "", err
	}

	pathParts := strings.Split(relPath, string(filepath.Separator))
	if len(pathParts) == 0 {
		return nil, "", fmt.Errorf("invalid path structure")
	}

	resourceName := pathParts[0]

	// Read file content (with size limit for safety)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, resourceName, err
	}
	defer file.Close()

	// Read file with size limit (100KB)
	const maxFileSize = 100 * 1024
	content := make([]byte, maxFileSize)
	n, _ := file.Read(content)
	contentStr := string(content[:n])

	var discoveredSecrets []ResourceSecret
	foundSecrets := make(map[string]bool) // Prevent duplicates

	// Search for secret patterns
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		matches := regex.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			secretKey := match[1]
			if secretKey == "" {
				continue
			}

			// Check if this looks like a secret based on keywords
			upperKey := strings.ToUpper(secretKey)
			isSecret := false
			for _, keyword := range keywords {
				if strings.Contains(upperKey, keyword) {
					isSecret = true
					break
				}
			}

			if !isSecret {
				continue
			}

			// Prevent duplicates
			secretId := resourceName + ":" + secretKey
			if foundSecrets[secretId] {
				continue
			}
			foundSecrets[secretId] = true

			// Determine secret type
			secretType := s.determineSecretType(upperKey)

			// Determine if likely required
			isRequired := s.isLikelyRequired(upperKey)

			description := fmt.Sprintf("Found in %s - %s", filepath.Base(filePath), pattern.Description)

			secret := ResourceSecret{
				ID:           uuid.New().String(),
				ResourceName: resourceName,
				SecretKey:    secretKey,
				SecretType:   secretType,
				Required:     isRequired,
				Description:  &description,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			discoveredSecrets = append(discoveredSecrets, secret)
		}
	}

	return discoveredSecrets, resourceName, nil
}

// determineSecretType determines the type of secret based on its key
func (s *SecretScanner) determineSecretType(upperKey string) string {
	if strings.Contains(upperKey, "PASSWORD") || strings.Contains(upperKey, "PASS") || strings.Contains(upperKey, "PWD") {
		return "password"
	} else if strings.Contains(upperKey, "TOKEN") {
		return "token"
	} else if strings.Contains(upperKey, "KEY") && (strings.Contains(upperKey, "API") || strings.Contains(upperKey, "ACCESS")) {
		return "api_key"
	} else if strings.Contains(upperKey, "SECRET") || strings.Contains(upperKey, "CREDENTIAL") {
		return "credential"
	} else if strings.Contains(upperKey, "CERT") || strings.Contains(upperKey, "TLS") || strings.Contains(upperKey, "SSL") {
		return "certificate"
	}
	return "env_var"
}

// isLikelyRequired determines if a secret is likely required
func (s *SecretScanner) isLikelyRequired(upperKey string) bool {
	requiredKeywords := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "DATABASE", "DB", "HOST"}
	for _, keyword := range requiredKeywords {
		if strings.Contains(upperKey, keyword) {
			return true
		}
	}
	return false
}

// storeScanRecord stores a scan record in the database
func (s *SecretScanner) storeScanRecord(scan SecretScan) error {
	if s.db == nil {
		return nil // Skip if no database connection
	}

	resourcesJSON, _ := json.Marshal(scan.ResourcesScanned)

	query := `
		INSERT INTO secret_scans (
			id, scan_type, resources_scanned, secrets_discovered, 
			scan_duration_ms, scan_timestamp, scan_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			secrets_discovered = EXCLUDED.secrets_discovered,
			scan_duration_ms = EXCLUDED.scan_duration_ms,
			scan_status = EXCLUDED.scan_status
	`

	_, err := s.db.Exec(query,
		scan.ID, scan.ScanType, resourcesJSON, scan.SecretsDiscovered,
		scan.ScanDurationMs, scan.ScanTimestamp, scan.ScanStatus,
	)

	return err
}

// storeResourceSecret stores a resource secret in the database
func (s *SecretScanner) storeResourceSecret(secret ResourceSecret) error {
	if s.db == nil {
		return nil // Skip if no database connection
	}

	query := `
		INSERT INTO resource_secrets (
			id, resource_name, secret_key, secret_type, required,
			description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (resource_name, secret_key) DO UPDATE SET
			secret_type = EXCLUDED.secret_type,
			required = EXCLUDED.required,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query,
		secret.ID, secret.ResourceName, secret.SecretKey, secret.SecretType,
		secret.Required, secret.Description, secret.CreatedAt, secret.UpdatedAt,
	)

	return err
}

// GetScanHistory returns recent scan history
func (s *SecretScanner) GetScanHistory(limit int) ([]SecretScan, error) {
	if s.db == nil {
		return []SecretScan{}, nil
	}

	query := `
		SELECT id, scan_type, resources_scanned, secrets_discovered,
		       scan_duration_ms, scan_timestamp, scan_status
		FROM secret_scans
		ORDER BY scan_timestamp DESC
		LIMIT $1
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []SecretScan
	for rows.Next() {
		var scan SecretScan
		var resourcesJSON []byte

		err := rows.Scan(
			&scan.ID, &scan.ScanType, &resourcesJSON, &scan.SecretsDiscovered,
			&scan.ScanDurationMs, &scan.ScanTimestamp, &scan.ScanStatus,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(resourcesJSON, &scan.ResourcesScanned)
		scans = append(scans, scan)
	}

	return scans, nil
}
