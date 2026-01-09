package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	dataDir = "data/campaigns"
)

var (
	fileLocks = make(map[string]*sync.RWMutex)
	locksLock = sync.RWMutex{}
)

// initFileStorage ensures the data directory exists
func initFileStorage() error {
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logger.Printf("✅ JSON file storage initialized at: %s", dataPath)
	return nil
}

// getFileLock returns a mutex for the given file path
func getFileLock(filename string) *sync.RWMutex {
	locksLock.Lock()
	defer locksLock.Unlock()

	if lock, exists := fileLocks[filename]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	fileLocks[filename] = lock
	return lock
}

// getCampaignPath returns the file path for a campaign
func getCampaignPath(campaignID uuid.UUID) string {
	return filepath.Join("scenarios", "visited-tracker", dataDir, campaignID.String()+".json")
}

// saveCampaign persists a campaign to disk
func saveCampaign(campaign *Campaign) error {
	campaign.UpdatedAt = time.Now().UTC()
	filePath := getCampaignPath(campaign.ID)

	lock := getFileLock(filePath)
	lock.Lock()
	defer lock.Unlock()

	data, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write campaign file: %w", err)
	}

	return nil
}

// loadCampaign loads a campaign from disk
func loadCampaign(campaignID uuid.UUID) (*Campaign, error) {
	filePath := getCampaignPath(campaignID)

	lock := getFileLock(filePath)
	lock.RLock()
	defer lock.RUnlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("campaign not found")
		}
		return nil, fmt.Errorf("failed to read campaign file: %w", err)
	}

	var campaign Campaign
	if err := json.Unmarshal(data, &campaign); err != nil {
		return nil, fmt.Errorf("failed to unmarshal campaign: %w", err)
	}

	return &campaign, nil
}

// loadAllCampaigns loads all campaigns from disk
func loadAllCampaigns() ([]Campaign, error) {
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)

	var campaigns []Campaign

	// Check if the directory exists first
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// Directory doesn't exist, return empty slice without error
		return campaigns, nil
	}

	err := filepath.WalkDir(dataPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			logger.Printf("⚠️ Failed to read campaign file %s: %v", path, err)
			return nil // Continue with other files
		}

		var campaign Campaign
		if err := json.Unmarshal(data, &campaign); err != nil {
			logger.Printf("⚠️ Failed to unmarshal campaign file %s: %v", path, err)
			return nil // Continue with other files
		}

		campaigns = append(campaigns, campaign)
		return nil
	})

	return campaigns, err
}

// deleteCampaignFile removes a campaign file from disk
func deleteCampaignFile(campaignID uuid.UUID) error {
	filePath := getCampaignPath(campaignID)

	lock := getFileLock(filePath)
	lock.Lock()
	defer lock.Unlock()

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete campaign file: %w", err)
	}

	return nil
}
