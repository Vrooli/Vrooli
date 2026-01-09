package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] returns error message", func(t *testing.T) {
		err := &APIError{
			Status:  http.StatusBadRequest,
			Code:    "TEST_ERROR",
			Message: "Test error message",
		}

		assert.Equal(t, "Test error message", err.Error())
	})
}

func TestAPIError_WithDetails(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] adds details to error", func(t *testing.T) {
		original := &APIError{
			Status:  http.StatusBadRequest,
			Code:    "TEST_ERROR",
			Message: "Test error message",
		}

		details := map[string]string{"field": "test_field"}
		withDetails := original.WithDetails(details)

		assert.Equal(t, original.Status, withDetails.Status)
		assert.Equal(t, original.Code, withDetails.Code)
		assert.Equal(t, original.Message, withDetails.Message)
		assert.Equal(t, details, withDetails.Details)

		// Verify original is unchanged
		assert.Nil(t, original.Details)
	})
}

func TestAPIError_WithMessage(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] overrides error message", func(t *testing.T) {
		original := &APIError{
			Status:  http.StatusBadRequest,
			Code:    "TEST_ERROR",
			Message: "Original message",
			Details: map[string]string{"key": "value"},
		}

		newMessage := "New message"
		withMessage := original.WithMessage(newMessage)

		assert.Equal(t, original.Status, withMessage.Status)
		assert.Equal(t, original.Code, withMessage.Code)
		assert.Equal(t, newMessage, withMessage.Message)
		assert.Equal(t, original.Details, withMessage.Details)

		// Verify original is unchanged
		assert.Equal(t, "Original message", original.Message)
	})
}

func TestRespondError(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] writes JSON error response", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := &APIError{
			Status:  http.StatusBadRequest,
			Code:    "TEST_ERROR",
			Message: "Test error message",
			Details: map[string]string{"field": "test_field"},
		}

		RespondError(w, err)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "TEST_ERROR", response.Code)
		assert.Equal(t, "Test error message", response.Message)

		detailsMap, ok := response.Details.(map[string]interface{})
		require.True(t, ok, "Details should be a map")
		assert.Equal(t, "test_field", detailsMap["field"])
	})
}

func TestRespondSuccess(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] writes JSON success response", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{
			"status": "success",
			"result": "test_result",
		}

		RespondSuccess(w, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]string
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "test_result", response["result"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] writes 201 Created response", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"id": "123"}

		RespondSuccess(w, http.StatusCreated, data)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestCommonErrors(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] ErrInvalidRequest has correct status", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, ErrInvalidRequest.Status)
		assert.Equal(t, "INVALID_REQUEST", ErrInvalidRequest.Code)
		assert.NotEmpty(t, ErrInvalidRequest.Message)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] ErrMissingRequiredField has correct status", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, ErrMissingRequiredField.Status)
		assert.Equal(t, "MISSING_REQUIRED_FIELD", ErrMissingRequiredField.Code)
		assert.NotEmpty(t, ErrMissingRequiredField.Message)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] ErrInternalServer has correct status", func(t *testing.T) {
		assert.Equal(t, http.StatusInternalServerError, ErrInternalServer.Status)
		assert.Equal(t, "INTERNAL_SERVER_ERROR", ErrInternalServer.Code)
		assert.NotEmpty(t, ErrInternalServer.Message)
	})
}
