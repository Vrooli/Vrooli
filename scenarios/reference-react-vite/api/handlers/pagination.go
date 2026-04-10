// Package handlers provides HTTP request handlers for the reference scenario.
// DOC: docs/reference/api-endpoints.md#pagination
package handlers

import (
	"net/url"
	"strconv"
)

// PaginationConfig holds the bounds for pagination parsing.
type PaginationConfig struct {
	DefaultLimit int
	MaxLimit     int
}

// ParsedPagination contains the extracted limit and offset values.
type ParsedPagination struct {
	Limit  int
	Offset int
}

// ParsePagination extracts limit and offset from query parameters.
// It applies bounds from the provided config:
//   - If limit is not provided or invalid, uses DefaultLimit
//   - If limit exceeds MaxLimit, uses MaxLimit
//   - If offset is not provided or invalid, uses 0
func ParsePagination(query url.Values, cfg PaginationConfig) ParsedPagination {
	p := ParsedPagination{
		Limit:  cfg.DefaultLimit,
		Offset: 0,
	}

	if v := query.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > cfg.MaxLimit {
				p.Limit = cfg.MaxLimit
			} else {
				p.Limit = n
			}
		}
	}

	if v := query.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}

	return p
}
