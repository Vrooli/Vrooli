package main

// errorResponse is the standard error response format for the API.
// Used by HTTPResponse.Error() for consistent error formatting.
type errorResponse struct {
	Error string `json:"error"`
}
