package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"landing-page-business-suite-api/internal/logx"

	"github.com/gorilla/mux"
)

// decodeJSONBody decodes the request body into dst.
// Returns true on success, or writes a JSON error response and returns false on failure.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
		return false
	}
	return true
}

// requireAuth extracts the authenticated user's email from the request context.
// Returns the user email and true on success, or writes a 401 error response and returns empty string, false.
func requireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	user := getUserEmail(r.Context())
	if user == "" {
		writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
		return "", false
	}
	return user, true
}

// getPathParam extracts a string path parameter from mux vars.
// Returns the value and true, or empty string and false if not found.
func getPathParam(r *http.Request, key string) (string, bool) {
	vars := mux.Vars(r)
	val := vars[key]
	return val, val != ""
}

// getPathParamInt64 extracts an int64 path parameter from mux vars.
// Returns the value and true on success, or writes an error response and returns 0, false on failure.
func getPathParamInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	vars := mux.Vars(r)
	idStr := vars[key]

	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing required parameter: "+key, ApiErrorTypeValidation)
		return 0, false
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid "+key+": must be a number", ApiErrorTypeValidation)
		return 0, false
	}

	return id, true
}

// getPathParamInt extracts an int path parameter from mux vars.
// Returns the value and true on success, or writes an error response and returns 0, false on failure.
func getPathParamInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	vars := mux.Vars(r)
	idStr := vars[key]

	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing required parameter: "+key, ApiErrorTypeValidation)
		return 0, false
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid "+key+": must be a number", ApiErrorTypeValidation)
		return 0, false
	}

	return id, true
}

// requireQueryParam extracts a required query parameter.
// Returns the value and true on success, or writes an error response and returns empty string, false.
func requireQueryParam(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	val := r.URL.Query().Get(key)
	if val == "" {
		writeJSONError(w, http.StatusBadRequest, key+" is required", ApiErrorTypeValidation)
		return "", false
	}
	return val, true
}

// getQueryParam extracts an optional query parameter.
// Returns the value (may be empty string) - never fails.
func getQueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// writeJSONSuccess writes a standard success response with the given message.
func writeJSONSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
	}); err != nil {
		logx.Error("write_json_success_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// writeJSONSuccessData writes a success response with custom data.
func writeJSONSuccessData(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logx.Error("write_json_success_data_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// writeJSONSuccessSimple writes a simple {"success": true} response.
func writeJSONSuccessSimple(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		logx.Error("write_json_success_simple_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}
