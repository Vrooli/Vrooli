package support

import "time"

// Campaign mirrors the shape returned by GET/POST /campaigns.
type Campaign struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Document mirrors the shape returned by GET /campaigns/{id}/documents.
type Document struct {
	ID            string    `json:"id"`
	CampaignID    string    `json:"campaign_id"`
	Filename      string    `json:"filename"`
	FilePath      string    `json:"file_path,omitempty"`
	ContentType   string    `json:"content_type,omitempty"`
	ProcessedText string    `json:"processed_text,omitempty"`
	EmbeddingID   string    `json:"embedding_id,omitempty"`
	UploadDate    time.Time `json:"upload_date"`
}
