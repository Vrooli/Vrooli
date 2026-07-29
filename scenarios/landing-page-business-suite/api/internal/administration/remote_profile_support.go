package administration

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/envx"
)

const (
	// These values are part of the HTTP error contract owned by the API
	// boundary. Keep the domain copies aligned so handlers can serialize
	// RemoteProfileError without translating implementation details.
	apiErrorTypeNetwork      = "network"
	apiErrorTypeTimeout      = "timeout"
	apiErrorTypeValidation   = "validation"
	apiErrorTypeUnauthorized = "unauthorized"
	apiErrorTypeServerError  = "server_error"
)

type remoteProfileLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type remoteProfileLoginResponse struct {
	Authenticated bool   `json:"authenticated"`
	SessionID     string `json:"session_id"`
}

type remoteProfileAPIErrorResponse struct {
	Error string `json:"error"`
}

var errRemoteProfileURLInvalid = errors.New("invalid URL format")

func resolveRemoteProfileSecret(name string) string { return envx.Get(name) }

// validateRemoteProfileEnvironment canonicalizes the only environment values
// that affect remote-profile network policy. An unset value intentionally maps
// to development to retain the scenario's local-safe default.
func validateRemoteProfileEnvironment(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "production":
		return "production"
	case "", "development":
		return "development"
	default:
		return "development"
	}
}

func isRemoteProfileProductionEnvironment() bool {
	return validateRemoteProfileEnvironment(envx.Get("LPBS_ENVIRONMENT")) == "production"
}

func validateRemoteProfileURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errRemoteProfileURLInvalid
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errRemoteProfileURLInvalid
	}
	return parsed.String(), nil
}

func logRemoteProfileEvent(_ string, _ map[string]interface{}) {}

func logRemoteProfileError(_ string, _ map[string]interface{}) {}

func inferRemoteErrorType(status int) string {
	switch status {
	case 400, 409, 422:
		return apiErrorTypeValidation
	case 401, 403:
		return apiErrorTypeUnauthorized
	default:
		return apiErrorTypeServerError
	}
}

func stringToNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func int64ToNullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func timeToNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullStringValue(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func nullTimeValue(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}

func nullInt64Value(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	copy := value.Int64
	return &copy
}
