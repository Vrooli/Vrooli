package handlers

import (
	"context"
	"errors"
	"net/http"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

// errorResponse creates a standardized error response
func errorResponse(message string) gin.H {
	return gin.H{
		"success": false,
		"error":   message,
	}
}

// successResponse creates a standardized success response with optional data
func successResponse(data interface{}) gin.H {
	if data == nil {
		return gin.H{
			"success": true,
		}
	}
	return gin.H{
		"success": true,
		"data":    data,
	}
}

// ServiceCallFunc represents a service method that returns data and an error
type ServiceCallFunc[T any] func(context.Context) (T, error)

// HandleServiceCall is a generic wrapper for service calls that automatically
// handles errors and generates appropriate responses
func HandleServiceCall[T any](
	c *gin.Context,
	fn ServiceCallFunc[T],
	errorMsg string,
) {
	result, err := fn(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError

		// Map known service errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrDatabaseUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrScenarioAuditorUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrIssueTrackerUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, errorResponse(errorMsg))
		return
	}

	c.JSON(http.StatusOK, successResponse(result))
}

// HandleServiceCallRaw is similar to HandleServiceCall but returns the data
// directly without wrapping it in a success response
func HandleServiceCallRaw[T any](
	c *gin.Context,
	fn ServiceCallFunc[T],
	errorMsg string,
) {
	result, err := fn(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError

		// Map known service errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrDatabaseUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrScenarioAuditorUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrIssueTrackerUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, errorResponse(errorMsg))
		return
	}

	c.JSON(http.StatusOK, result)
}

// ActionCallFunc represents a service method that performs an action and returns only an error
type ActionCallFunc func(context.Context) error

// HandleServiceAction is a wrapper for service actions that don't return data
func HandleServiceAction(
	c *gin.Context,
	fn ActionCallFunc,
	successMsg string,
	errorMsg string,
) {
	err := fn(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError

		// Map known service errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, services.ErrAppNotFound):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrAppIdentifierRequired):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrDatabaseUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrScenarioAuditorUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, services.ErrIssueTrackerUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, errorResponse(errorMsg))
		return
	}

	c.JSON(http.StatusOK, successResponse(gin.H{"message": successMsg}))
}
