package support

import (
	"encoding/json"
	"time"
)

// Device mirrors the shape returned by /api/v1/devices.
type Device struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id,omitempty"`
	Name         string    `json:"name"`
	Type         string    `json:"type,omitempty"`
	Platform     string    `json:"platform,omitempty"`
	LastSeen     time.Time `json:"last_seen"`
	Capabilities []string  `json:"capabilities,omitempty"`
	IsOnline     bool      `json:"is_online"`
}

// SyncItem mirrors the shape returned by /api/v1/sync/items.
type SyncItem struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id,omitempty"`
	Type          string                 `json:"type"`
	Content       map[string]interface{} `json:"content"`
	SourceDevice  string                 `json:"source_device,omitempty"`
	TargetDevices []string               `json:"target_devices,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	Status        string                 `json:"status,omitempty"`
}

// UploadResponse is the {success, item_id, expires_at, ...} payload from POST /api/v1/sync/upload.
type UploadResponse struct {
	Success      bool            `json:"success"`
	ItemID       string          `json:"item_id"`
	ExpiresAt    string          `json:"expires_at"`
	Filename     string          `json:"filename,omitempty"`
	FileSize     json.RawMessage `json:"file_size,omitempty"`
	ThumbnailURL string          `json:"thumbnail_url,omitempty"`
}

// DeleteResponse is the payload from DELETE /api/v1/sync/items/{id}.
type DeleteResponse struct {
	Success   bool   `json:"success"`
	DeletedAt string `json:"deleted_at"`
}
