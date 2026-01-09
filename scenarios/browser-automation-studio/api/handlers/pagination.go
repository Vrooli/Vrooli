package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/vrooli/browser-automation-studio/config"
)

// getDefaultPageLimit returns the configured default page limit.
// Configurable via BAS_HTTP_DEFAULT_PAGE_LIMIT (default: 100)
func getDefaultPageLimit() int {
	return config.Load().HTTP.DefaultPageLimit
}

// getMaxPageLimit returns the configured maximum page limit.
// Configurable via BAS_HTTP_MAX_PAGE_LIMIT (default: 500)
func getMaxPageLimit() int {
	return config.Load().HTTP.MaxPageLimit
}

// getMaxOffset returns the configured maximum offset for pagination.
// Configurable via BAS_HTTP_MAX_OFFSET (default: 100000)
func getMaxOffset() int {
	return config.Load().HTTP.MaxOffset
}

func parsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	if defaultLimit <= 0 {
		defaultLimit = getDefaultPageLimit()
	}
	if maxLimit <= 0 {
		maxLimit = getMaxPageLimit()
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

	// Cap offset to prevent expensive queries
	maxOffset := getMaxOffset()
	if offset > maxOffset {
		offset = maxOffset
	}

	return limit, offset
}
