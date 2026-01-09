// Package appctx provides shared context for CLI commands.
package appctx

import "net/url"

// Context provides methods that commands can use to interact with the API.
type Context interface {
	// Get performs a GET request and unmarshals the response into result.
	Get(path string, result interface{}) error

	// GetWithQuery performs a GET request with query parameters.
	GetWithQuery(path string, query url.Values, result interface{}) error

	// Post performs a POST request with the given payload.
	Post(path string, payload interface{}, result interface{}) error

	// Put performs a PUT request with the given payload.
	Put(path string, payload interface{}, result interface{}) error

	// Delete performs a DELETE request.
	Delete(path string) error
}
