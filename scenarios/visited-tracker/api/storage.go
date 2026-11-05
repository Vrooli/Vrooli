package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/platform-go"

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
	ErrCampaignConflict = errors.New("campaign revision changed; reload before retrying")
	ErrCampaignNotFound = errors.New("campaign not found")
)

// mutateCampaign owns the complete read/modify/replace transaction. All durable
// mutations, including legacy saves, use the same OS lock across API instances.
func mutateCampaign(ctx context.Context, id uuid.UUID, mutate func(*Campaign) error) (*Campaign, error) {
	release, err := lockCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	defer release()
	c, err := loadCampaign(id)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := mutate(c); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := writeCampaign(c); err != nil {
		return nil, err
	}
	return c, nil
}

func lockCampaign(ctx context.Context, id uuid.UUID) (func(), error) {
	path := getCampaignPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return platform.AcquireFileLockContext(ctx, path+".lock")
}

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

// getCampaignPath returns the file path for a campaign
func getCampaignPath(campaignID uuid.UUID) string {
	return mustStoragePath(storage.ClassData, filepath.Join(campaignsDir, campaignID.String()+".json"))
}

// saveCampaign is compare-and-swap for callers that prepared a detached value.
// New attention operations use mutateCampaign instead of retrying stale values.
func saveCampaign(ctx context.Context, campaign *Campaign) error {
	release, err := lockCampaign(ctx, campaign.ID)
	if err != nil {
		return err
	}
	defer release()
	current, err := loadCampaign(campaign.ID)
	if err != nil && !errors.Is(err, ErrCampaignNotFound) {
		return err
	}
	if current != nil && current.Revision != campaign.Revision {
		return ErrCampaignConflict
	}
	if current == nil && campaign.Revision != 0 {
		return ErrCampaignConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeCampaign(campaign)
}

// writeCampaign runs under the campaign lock. A crash before rename leaves the
// previous JSON intact. Readers see a complete old or new revision.
func writeCampaign(campaign *Campaign) error {
	err := replaceCampaign(campaign)
	campaignWriteObserver.record(time.Now(), err)
	return err
}

func replaceCampaign(campaign *Campaign) error {
	next := *campaign
	next.Revision++
	next.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(&next, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal campaign: %w", err)
	}
	path := getCampaignPath(campaign.ID)
	temp, err := os.CreateTemp(filepath.Dir(path), ".campaign-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("replace campaign: %w", err)
	}
	campaign.Revision = next.Revision
	campaign.UpdatedAt = next.UpdatedAt
	// Persist the directory entry as well as the file on systems that support it.
	if runtime.GOOS != "windows" {
		dir, err := os.Open(filepath.Dir(path))
		if err != nil {
			return err
		}
		syncErr := dir.Sync()
		closeErr := dir.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCampaignNotFound
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
			return fmt.Errorf("read campaign %s: %w", path, err)
		}

		var campaign Campaign
		if err := json.Unmarshal(data, &campaign); err != nil {
			logger.Printf("⚠️ Failed to unmarshal campaign file %s: %v", path, err)
			return fmt.Errorf("read campaign %s: %w", path, err)
		}

		campaigns = append(campaigns, campaign)
		return nil
	})

	return campaigns, err
}

// deleteCampaignFile removes a campaign file from disk
func deleteCampaignFile(ctx context.Context, campaignID uuid.UUID) error {
	filePath := getCampaignPath(campaignID)

	release, err := lockCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	defer release()

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete campaign file: %w", err)
	}

	return nil
}

func campaignWriteStatus(err error) int {
	if errors.Is(err, ErrCampaignConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func lockCampaignCatalog(ctx context.Context) (func(), error) {
	if err := os.MkdirAll(storageDataPath(), 0o755); err != nil {
		return nil, err
	}
	return platform.AcquireFileLockContext(ctx, filepath.Join(storageDataPath(), ".catalog.lock"))
}
