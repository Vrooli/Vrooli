package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPageLimit = 100
	maxPageLimit     = 500
)

func parsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	if defaultLimit <= 0 {
		defaultLimit = defaultPageLimit
	}
	if maxLimit <= 0 {
		maxLimit = maxPageLimit
	}

	limit = defaultLimit
	offset = 0

	query := r.URL.Query()
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		if parsed, err := strconv.Atoi(rawOffset); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
