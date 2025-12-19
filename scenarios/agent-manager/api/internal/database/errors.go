// Package database provides PostgreSQL persistence for agent-manager entities.
package database

import "errors"

// Common database errors
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned when attempting to create a duplicate entity.
	ErrAlreadyExists = errors.New("already exists")

	// ErrConflict is returned when there's a concurrency conflict.
	ErrConflict = errors.New("conflict")
)
