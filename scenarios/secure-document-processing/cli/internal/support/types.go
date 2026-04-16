package support

import "time"

// Document mirrors the shape returned by GET /api/documents.
type Document struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Status   string    `json:"status"`
	Created  time.Time `json:"created"`
}

// ProcessingJob mirrors the shape returned by GET /api/jobs.
type ProcessingJob struct {
	ID        string    `json:"id"`
	JobName   string    `json:"jobName"`
	Status    string    `json:"status"`
	Created   time.Time `json:"created"`
	Documents []string  `json:"documents"`
}

// Workflow mirrors the shape returned by GET /api/workflows.
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Created     time.Time `json:"created"`
}
