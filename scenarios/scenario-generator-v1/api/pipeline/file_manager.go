package pipeline

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileManager handles file operations for generated scenarios
type FileManager struct {
	baseDir string
}

// NewFileManager creates a new file manager
func NewFileManager() *FileManager {
	baseDir := os.Getenv("SCENARIO_OUTPUT_DIR")
	if baseDir == "" {
		baseDir = "/tmp/generated-scenarios"
	}
	
	return &FileManager{
		baseDir: baseDir,
	}
}

// SaveScenarioFiles saves generated files to disk
func (fm *FileManager) SaveScenarioFiles(scenarioID string, files map[string]string) error {
	scenarioDir := filepath.Join(fm.baseDir, scenarioID)
	
	// Create scenario directory
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		return fmt.Errorf("failed to create scenario directory: %w", err)
	}
	
	// Write each file
	for filename, content := range files {
		fullPath := filepath.Join(scenarioDir, filename)
		
		// Create parent directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}
		
		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
		
		// Make shell scripts executable
		if strings.HasSuffix(filename, ".sh") {
			os.Chmod(fullPath, 0755)
		}
	}
	
	// Create metadata file
	metadata := map[string]interface{}{
		"scenario_id": scenarioID,
		"file_count":  len(files),
		"files":       getFileList(files),
		"created_at":  fmt.Sprintf("%v", os.FileInfo.Name),
	}
	
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataPath := filepath.Join(scenarioDir, ".scenario-metadata.json")
	os.WriteFile(metadataPath, metadataJSON, 0644)
	
	return nil
}

// LoadScenarioFiles loads files from a saved scenario
func (fm *FileManager) LoadScenarioFiles(scenarioID string) (map[string]string, error) {
	scenarioDir := filepath.Join(fm.baseDir, scenarioID)
	
	// Check if directory exists
	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario %s not found", scenarioID)
	}
	
	files := make(map[string]string)
	
	// Walk through directory and load files
	err := filepath.Walk(scenarioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and metadata
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(scenarioDir, path)
		if err != nil {
			return err
		}
		
		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", relPath, err)
		}
		
		files[relPath] = string(content)
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to load scenario files: %w", err)
	}
	
	return files, nil
}

// CreateZipArchive creates a ZIP file of the scenario
func (fm *FileManager) CreateZipArchive(scenarioID string, files map[string]string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	
	// Add each file to the ZIP
	for filename, content := range files {
		// Create file in ZIP
		writer, err := zipWriter.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s in ZIP: %w", filename, err)
		}
		
		// Write content
		if _, err := io.WriteString(writer, content); err != nil {
			return nil, fmt.Errorf("failed to write %s to ZIP: %w", filename, err)
		}
	}
	
	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize ZIP: %w", err)
	}
	
	return buf.Bytes(), nil
}

// DeleteScenarioFiles removes a scenario's files
func (fm *FileManager) DeleteScenarioFiles(scenarioID string) error {
	scenarioDir := filepath.Join(fm.baseDir, scenarioID)
	
	if err := os.RemoveAll(scenarioDir); err != nil {
		return fmt.Errorf("failed to delete scenario files: %w", err)
	}
	
	return nil
}

// ListScenarios returns a list of saved scenario IDs
func (fm *FileManager) ListScenarios() ([]string, error) {
	scenarios := []string{}
	
	// Check if base directory exists
	if _, err := os.Stat(fm.baseDir); os.IsNotExist(err) {
		return scenarios, nil
	}
	
	// Read directory
	entries, err := os.ReadDir(fm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			scenarios = append(scenarios, entry.Name())
		}
	}
	
	return scenarios, nil
}

// getFileList returns a list of file paths
func getFileList(files map[string]string) []string {
	list := []string{}
	for filename := range files {
		list = append(list, filename)
	}
	return list
}