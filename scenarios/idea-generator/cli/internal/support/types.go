package support

import "time"

// Campaign mirrors the shape returned by /api/campaigns.
type Campaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Idea mirrors the shape returned by /api/ideas.
type Idea struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Workflow mirrors one capability entry from /api/workflows.
type Workflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	URL         string `json:"url,omitempty"`
}
