package main

import "time"

// IgnoreRequest specifies a file to ignore.
type IgnoreRequest struct {
	Path string `json:"path"`
}

// IgnoreResponse contains the result of an ignore operation.
type IgnoreResponse struct {
	Success       bool      `json:"success"`
	Ignored       []string  `json:"ignored"`
	Failed        []string  `json:"failed,omitempty"`
	Errors        []string  `json:"errors,omitempty"`
	GitignorePath string    `json:"gitignore_path,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}
