package support

import "time"

// Profile mirrors the ProfileResponse shape returned by /api/v1/profiles.
type Profile struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
}

// ProfileStats mirrors the StatsResponse returned by /api/v1/profiles/{id}/stats.
type ProfileStats struct {
	TotalBookmarks  int        `json:"total_bookmarks"`
	CategoriesCount int        `json:"categories_count"`
	PendingActions  int        `json:"pending_actions"`
	AccuracyRate    float64    `json:"accuracy_rate"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
}

// Category is one entry from /api/v1/categories.
type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Platform is one entry from /api/v1/platforms.
type Platform struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Supported   bool   `json:"supported"`
}

// PlatformStatus is one entry from /api/v1/platforms/status.
type PlatformStatus struct {
	Name      string     `json:"name"`
	Connected bool       `json:"connected"`
	LastSync  *time.Time `json:"last_sync,omitempty"`
}
