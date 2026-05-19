package tools

import "errors"

var (
	errMissingToolName = errors.New("tool name is required")
	errToolNotFound    = errors.New("tool not found")
)
