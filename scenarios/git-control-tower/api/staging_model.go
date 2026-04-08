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
	Warnings  []string  `json:"warnings,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// IsSuccess returns whether the staging operation succeeded.
func (r *StageResponse) IsSuccess() bool { return r.Success }

// ErrorMessages returns the error messages from the staging operation.
func (r *StageResponse) ErrorMessages() []string { return r.Errors }

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

// IsSuccess returns whether the unstaging operation succeeded.
func (r *UnstageResponse) IsSuccess() bool { return r.Success }

// ErrorMessages returns the error messages from the unstaging operation.
func (r *UnstageResponse) ErrorMessages() []string { return r.Errors }
