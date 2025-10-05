// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Request        HTTPTestRequest
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds a test pattern for invalid UUID
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    urlPath,
			URLVars: map[string]string{"id": "invalid-uuid-format"},
		},
	})
	return b
}

// AddNonExistentResource adds a test pattern for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(urlPath, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    urlPath,
			URLVars: map[string]string{"id": nonExistentID},
		},
	})
	return b
}

// AddInvalidJSON adds a test pattern for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   "invalid-json-{not-parseable", // This will fail to marshal
		},
	})
	return b
}

// AddMissingAuth adds a test pattern for missing authentication
func (b *TestScenarioBuilder) AddMissingAuth(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test handler without authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    urlPath,
			Headers: map[string]string{},
		},
	})
	return b
}

// AddInvalidAuth adds a test pattern for invalid authentication
func (b *TestScenarioBuilder) AddInvalidAuth(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidAuth",
		Description:    "Test handler with invalid authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Request: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Headers: map[string]string{
				"Authorization": "Bearer invalid-token-12345",
			},
		},
	})
	return b
}

// AddEmptyBody adds a test pattern for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   map[string]interface{}{},
		},
	})
	return b
}

// AddMissingRequiredField adds a test pattern for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, method string, partialBody map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   partialBody,
		},
	})
	return b
}

// AddInvalidAlgorithm adds a test pattern for unsupported algorithm
func (b *TestScenarioBuilder) AddInvalidAlgorithm(urlPath, method string, invalidAlgo string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidAlgorithm_%s", invalidAlgo),
		Description:    fmt.Sprintf("Test handler with unsupported algorithm: %s", invalidAlgo),
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body: map[string]interface{}{
				"data":      "test data",
				"algorithm": invalidAlgo,
			},
		},
	})
	return b
}

// Build returns the constructed error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
	Server      *Server
	AuthToken   string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, server *Server, authToken string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		Server:      server,
		AuthToken:   authToken,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Add auth header if not testing auth
			if pattern.Request.Headers == nil {
				pattern.Request.Headers = make(map[string]string)
			}
			if _, hasAuth := pattern.Request.Headers["Authorization"]; !hasAuth && pattern.Name != "MissingAuth" && pattern.Name != "InvalidAuth" {
				pattern.Request.Headers["Authorization"] = "Bearer " + suite.AuthToken
			}

			// Execute request
			w, err := makeHTTPRequest(suite.Server, pattern.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate error response structure
			if pattern.ExpectedStatus >= 400 {
				assertErrorResponse(t, w, pattern.ExpectedStatus, "")
			}
		})
	}
}

// Common crypto-specific test patterns

// CryptoHashErrorPatterns returns common hash operation error patterns
func CryptoHashErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/hash", "POST").
		AddInvalidAuth("/api/v1/crypto/hash", "POST").
		AddEmptyBody("/api/v1/crypto/hash", "POST").
		AddInvalidAlgorithm("/api/v1/crypto/hash", "POST", "invalid-algo").
		AddInvalidAlgorithm("/api/v1/crypto/hash", "POST", "md4").
		Build()
}

// CryptoEncryptErrorPatterns returns common encryption operation error patterns
func CryptoEncryptErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/encrypt", "POST").
		AddInvalidAuth("/api/v1/crypto/encrypt", "POST").
		AddEmptyBody("/api/v1/crypto/encrypt", "POST").
		AddInvalidAlgorithm("/api/v1/crypto/encrypt", "POST", "des").
		AddMissingRequiredField("/api/v1/crypto/encrypt", "POST", map[string]interface{}{
			"algorithm": "aes256",
			// Missing "data" field
		}).
		Build()
}

// CryptoDecryptErrorPatterns returns common decryption operation error patterns
func CryptoDecryptErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/decrypt", "POST").
		AddInvalidAuth("/api/v1/crypto/decrypt", "POST").
		AddEmptyBody("/api/v1/crypto/decrypt", "POST").
		AddMissingRequiredField("/api/v1/crypto/decrypt", "POST", map[string]interface{}{
			"algorithm": "aes256",
			// Missing "encrypted_data" and "key"
		}).
		Build()
}

// CryptoKeyGenErrorPatterns returns common key generation error patterns
func CryptoKeyGenErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/keys/generate", "POST").
		AddInvalidAuth("/api/v1/crypto/keys/generate", "POST").
		AddEmptyBody("/api/v1/crypto/keys/generate", "POST").
		AddInvalidAlgorithm("/api/v1/crypto/keys/generate", "POST", "dsa"). // Using algorithm field as key_type analogy
		Build()
}

// CryptoSignErrorPatterns returns common signing operation error patterns
func CryptoSignErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/sign", "POST").
		AddInvalidAuth("/api/v1/crypto/sign", "POST").
		AddEmptyBody("/api/v1/crypto/sign", "POST").
		AddMissingRequiredField("/api/v1/crypto/sign", "POST", map[string]interface{}{
			"data": "test data",
			// Missing "key_id"
		}).
		AddMissingRequiredField("/api/v1/crypto/sign", "POST", map[string]interface{}{
			"key_id": uuid.New().String(),
			// Missing "data"
		}).
		Build()
}

// CryptoVerifyErrorPatterns returns common verification operation error patterns
func CryptoVerifyErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth("/api/v1/crypto/verify", "POST").
		AddInvalidAuth("/api/v1/crypto/verify", "POST").
		AddEmptyBody("/api/v1/crypto/verify", "POST").
		AddMissingRequiredField("/api/v1/crypto/verify", "POST", map[string]interface{}{
			"data": "test data",
			// Missing "signature" and "public_key"/"key_id"
		}).
		Build()
}

// EdgeCaseTestPattern defines edge case testing scenarios
type EdgeCaseTestPattern struct {
	Name        string
	Description string
	Request     HTTPTestRequest
	Validate    func(t *testing.T, w *testRecorder)
}

// testRecorder is an alias for httptest.ResponseRecorder for pattern compatibility
type testRecorder = interface {
	Code() int
	Body() string
}

// CryptoEdgeCasePatterns returns edge case test patterns for crypto operations
func CryptoEdgeCasePatterns() []struct {
	Name    string
	Request HTTPTestRequest
} {
	return []struct {
		Name    string
		Request HTTPTestRequest
	}{
		{
			Name: "VeryLongData",
			Request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/hash",
				Body: map[string]interface{}{
					"data":      string(make([]byte, 1024*1024)), // 1MB of data
					"algorithm": "sha256",
				},
			},
		},
		{
			Name: "EmptyStringData",
			Request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/hash",
				Body: map[string]interface{}{
					"data":      "",
					"algorithm": "sha256",
				},
			},
		},
		{
			Name: "UnicodeData",
			Request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/hash",
				Body: map[string]interface{}{
					"data":      "Hello 世界 🌍 Привет مرحبا",
					"algorithm": "sha256",
				},
			},
		},
	}
}
