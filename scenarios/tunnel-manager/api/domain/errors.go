package domain

// ApiErrorResponse is the standard error response envelope.
type ApiErrorResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

const (
	ErrCodeValidation  = "validation_error"
	ErrCodeNotFound    = "not_found"
	ErrCodeConflict    = "conflict"
	ErrCodeInternal    = "internal_error"
	ErrCodeUnavailable = "service_unavailable"
)

// DomainError is a typed error that carries an error code for HTTP status mapping.
type DomainError struct {
	Code    string
	Message string
	Err     error // optional wrapped cause
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Err }

func ErrValidation(msg string) *DomainError {
	return &DomainError{Code: ErrCodeValidation, Message: msg}
}
func ErrNotFound(msg string) *DomainError { return &DomainError{Code: ErrCodeNotFound, Message: msg} }
func ErrConflict(msg string) *DomainError { return &DomainError{Code: ErrCodeConflict, Message: msg} }
func ErrInternal(msg string, err error) *DomainError {
	return &DomainError{Code: ErrCodeInternal, Message: msg, Err: err}
}

func ErrUnavailable(msg string) *DomainError {
	return &DomainError{Code: ErrCodeUnavailable, Message: msg}
}
