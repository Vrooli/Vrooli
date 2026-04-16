package support

import (
	"encoding/json"
	"time"
)

// User mirrors the public user shape returned by /api/v1/auth/me and the auth
// envelopes.
type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	FirstName        string    `json:"first_name,omitempty"`
	LastName         string    `json:"last_name,omitempty"`
	SubscriptionTier string    `json:"subscription_tier,omitempty"`
	Timezone         string    `json:"timezone,omitempty"`
	Preferences      string    `json:"preferences,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// AuthEnvelope is the shape wrapped inside {success, data} for login/register.
type AuthEnvelope struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

// Platform mirrors one entry from /api/v1/auth/platforms.
type Platform struct {
	Platform     string   `json:"platform"`
	DisplayName  string   `json:"display_name"`
	Color        string   `json:"color,omitempty"`
	MaxChars     int      `json:"max_chars"`
	MediaLimit   int      `json:"media_limit,omitempty"`
	Features     []string `json:"features,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

// SocialAccount mirrors the shape returned by /api/v1/user/accounts.
type SocialAccount struct {
	ID          string                 `json:"id"`
	Platform    string                 `json:"platform"`
	Username    string                 `json:"username"`
	DisplayName string                 `json:"display_name"`
	AvatarURL   string                 `json:"avatar_url,omitempty"`
	AccountData map[string]interface{} `json:"account_data,omitempty"`
	IsActive    bool                   `json:"is_active"`
	LastUsedAt  *time.Time             `json:"last_used_at,omitempty"`
	ConnectedAt time.Time              `json:"connected_at"`
}

// ScheduledPost mirrors the API response for a scheduled post.
type ScheduledPost struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id"`
	CampaignID       *string                `json:"campaign_id,omitempty"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	PlatformVariants map[string]string      `json:"platform_variants,omitempty"`
	MediaURLs        []string               `json:"media_urls,omitempty"`
	Platforms        []string               `json:"platforms,omitempty"`
	ScheduledAt      time.Time              `json:"scheduled_at"`
	Timezone         string                 `json:"timezone,omitempty"`
	Status           string                 `json:"status"`
	PostedAt         *time.Time             `json:"posted_at,omitempty"`
	AnalyticsData    map[string]interface{} `json:"analytics_data,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Campaign mirrors the API response for a campaign.
type Campaign struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	ExternalCampaignID *string                `json:"external_campaign_id,omitempty"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	BrandGuidelines    map[string]interface{} `json:"brand_guidelines,omitempty"`
	Status             string                 `json:"status"`
	StartDate          *string                `json:"start_date,omitempty"`
	EndDate            *string                `json:"end_date,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// QueueHealth mirrors the response shape from /health/queue.
type QueueHealth struct {
	Status    string                 `json:"status"`
	Queues    map[string]json.Number `json:"queues,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	WorkerPID json.Number            `json:"worker_pid,omitempty"`
}
