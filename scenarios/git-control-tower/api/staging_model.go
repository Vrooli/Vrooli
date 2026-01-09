package main

import "time"

// StageRequest specifies files to stage
type StageRequest struct {
	Paths []string `json:"paths"`
	Scope string   `json:"scope,omitempty"`
}

// StageResponse contains the result of a staging operation
type StageResponse struct {
	Success   bool      `json:"success"`
	Staged    []string  `json:"staged"`
	Failed    []string  `json:"failed,omitempty"`
	Errors    []string  `json:"errors,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// UnstageRequest specifies files to unstage
type UnstageRequest struct {
	Paths []string `json:"paths"`
	Scope string   `json:"scope,omitempty"`
}

// UnstageResponse contains the result of an unstaging operation
type UnstageResponse struct {
	Success   bool      `json:"success"`
	Unstaged  []string  `json:"unstaged"`
	Failed    []string  `json:"failed,omitempty"`
	Errors    []string  `json:"errors,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
