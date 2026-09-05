package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

const (
	appID        = "vrooli"
	scenarioID   = "visited-tracker"
	campaignsDir = "campaigns"
	dataDir      = campaignsDir
)

var (
	fileLocks = make(map[string]*sync.RWMutex)
	locksLock = sync.RWMutex{}
)

// initFileStorage ensures the data directory exists
func initFileStorage() error {
	dataPath := storageDataPath()
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logger.Printf("✅ JSON file storage initialized at: %s", dataPath)
	return nil
}

func storageDataPath() string {
	return mustStoragePath(storage.ClassData, campaignsDir)
}

func storageHealthCheck(ctx context.Context) error {
	_ = ctx
	dataPath := storageDataPath()
	if _, err := os.Stat(dataPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("storage directory missing")
		}
		return fmt.Errorf("storage check failed: %w", err)
	}
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
	return mustStoragePath(storage.ClassData, filepath.Join(campaignsDir, campaignID.String()+".json"))
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

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create campaign directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write campaign file: %w", err)
	}

	return nil
}

func mustStoragePath(class storage.Class, rel string) string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   appID,
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		panic(fmt.Sprintf("build storage resolver: %v", err))
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioID}, class, rel)
	if err != nil {
		panic(fmt.Sprintf("resolve storage path: %v", err))
	}
	return path
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
	dataPath := storageDataPath()

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
