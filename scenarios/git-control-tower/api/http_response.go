package main

import (
	"encoding/json"
	"net/http"
)

// HTTPResponse provides consistent response handling for all handlers.
// This reduces cognitive load by centralizing JSON encoding and status decisions.
type HTTPResponse struct {
	w http.ResponseWriter
}

// NewResponse creates an HTTPResponse for the given ResponseWriter.
func NewResponse(w http.ResponseWriter) *HTTPResponse {
	return &HTTPResponse{w: w}
}

// JSON writes a JSON response with the given status code.
// Use this for all successful responses.
func (r *HTTPResponse) JSON(status int, data interface{}) {
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(status)
	_ = json.NewEncoder(r.w).Encode(data)
}

// OK writes a 200 OK JSON response.
func (r *HTTPResponse) OK(data interface{}) {
	r.JSON(http.StatusOK, data)
}

// Error writes a JSON error response with the given status code and message.
func (r *HTTPResponse) Error(status int, message string) {
	r.JSON(status, errorResponse{Error: message})
}

// BadRequest writes a 400 Bad Request error response.
func (r *HTTPResponse) BadRequest(message string) {
	r.Error(http.StatusBadRequest, message)
}

// InternalError writes a 500 Internal Server Error response.
func (r *HTTPResponse) InternalError(message string) {
	r.Error(http.StatusInternalServerError, message)
}

// ServiceUnavailable writes a 503 Service Unavailable response.
func (r *HTTPResponse) ServiceUnavailable(data interface{}) {
	r.JSON(http.StatusServiceUnavailable, data)
}

// UnprocessableEntity writes a 422 Unprocessable Entity response.
func (r *HTTPResponse) UnprocessableEntity(data interface{}) {
	r.JSON(http.StatusUnprocessableEntity, data)
}
