package healthutil

// NewResult returns a base health check result with connected=false and no error.
func NewResult() map[string]interface{} {
	return map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
}

// WithError sets a structured error on the health check result and returns it.
func WithError(result map[string]interface{}, code, message, category string, retryable bool) map[string]interface{} {
	result["error"] = map[string]interface{}{
		"code":      code,
		"message":   message,
		"category":  category,
		"retryable": retryable,
	}
	return result
}

// MarkConnected sets the result as successfully connected and returns it.
func MarkConnected(result map[string]interface{}) map[string]interface{} {
	result["connected"] = true
	return result
}
