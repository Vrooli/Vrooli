package main

import (
	"net/http"
	"testing"
	"time"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *http.Response, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName     string
	Handler         http.HandlerFunc
	BaseURL         string
	RequiresDB      bool
	RequiredHeaders []string
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

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`,
			}
		},
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, urlPath string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequired_" + fieldName,
		Description:    "Test handler with missing required field: " + fieldName,
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddMethodNotAllowed adds method not allowed test pattern
func (b *TestScenarioBuilder) AddMethodNotAllowed(invalidMethod, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MethodNotAllowed_" + invalidMethod,
		Description:    "Test handler with invalid HTTP method",
		ExpectedStatus: http.StatusMethodNotAllowed,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: invalidMethod,
				Path:   urlPath,
			}
		},
	})
	return b
}

// AddEmptyQueryParameter adds empty query parameter test pattern
func (b *TestScenarioBuilder) AddEmptyQueryParameter(method, urlPath, paramName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyQueryParam_" + paramName,
		Description:    "Test handler with empty query parameter",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:      method,
				Path:        urlPath,
				QueryParams: map[string]string{paramName: ""},
			}
		},
	})
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name        string
	Description string
	MaxDuration time.Duration
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}) time.Duration
	Cleanup     func(setupData interface{})
}

// RunPerformanceTest executes a performance test and validates duration
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute and measure
		duration := pattern.Execute(t, setupData)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test '%s' exceeded max duration: %v > %v",
				pattern.Name, duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test '%s' completed in %v (max: %v)",
				pattern.Name, duration, pattern.MaxDuration)
		}
	})
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name        string
	Description string
	Concurrency int
	Iterations  int
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}, iteration int) error
	Validate    func(t *testing.T, setupData interface{}, results []error)
	Cleanup     func(setupData interface{})
}

// RunConcurrencyTest executes concurrent operations and validates results
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute concurrently
		results := make([]error, pattern.Iterations)
		done := make(chan bool, pattern.Concurrency)

		for i := 0; i < pattern.Iterations; i++ {
			go func(iteration int) {
				results[iteration] = pattern.Execute(t, setupData, iteration)
				done <- true
			}(i)

			// Limit concurrency
			if (i+1)%pattern.Concurrency == 0 {
				for j := 0; j < pattern.Concurrency; j++ {
					<-done
				}
			}
		}

		// Wait for remaining goroutines
		remaining := pattern.Iterations % pattern.Concurrency
		if remaining > 0 {
			for j := 0; j < remaining; j++ {
				<-done
			}
		}

		// Validate results
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}

// Common test data builders

// BuildTestTravel creates a test Travel struct
func BuildTestTravel(userID string) *Travel {
	return &Travel{
		ID:           time.Now().UnixNano(),
		UserID:       userID,
		Location:     "Test Location",
		Lat:          40.7128,
		Lng:          -74.0060,
		Date:         "2024-01-01",
		Type:         "vacation",
		Notes:        "Test notes",
		Country:      "USA",
		City:         "New York",
		Continent:    "North America",
		DurationDays: 3,
		Rating:       4,
		Photos:       []string{},
		Tags:         []string{"test"},
		CreatedAt:    time.Now(),
	}
}

// BuildTestStats creates a test Stats struct
func BuildTestStats() *Stats {
	return &Stats{
		TotalCountries:       5,
		TotalCities:          10,
		TotalContinents:      3,
		TotalDistanceKm:      10000.0,
		TotalDaysTraveled:    30,
		WorldCoveragePercent: 2.5,
	}
}

// BuildTestAchievement creates a test Achievement struct
func BuildTestAchievement(achievementType string) *Achievement {
	return &Achievement{
		Type:        achievementType,
		Name:        "Test Achievement",
		Description: "Test description",
		Icon:        "🏆",
		UnlockedAt:  time.Now(),
	}
}

// BuildTestBucketItem creates a test BucketListItem struct
func BuildTestBucketItem() *BucketListItem {
	return &BucketListItem{
		ID:             1,
		Location:       "Tokyo, Japan",
		Country:        "Japan",
		City:           "Tokyo",
		Priority:       1,
		Notes:          "Visit during cherry blossom",
		EstimatedDate:  "2025-04-01",
		BudgetEstimate: 3000.00,
		Tags:           []string{"asia"},
		Completed:      false,
	}
}
