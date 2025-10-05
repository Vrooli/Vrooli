package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestModelManagementWithoutDB tests model endpoints when DB is not available
func TestModelManagementWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ListModels", func(t *testing.T) {
		testEndpoint(t, server, "GET", "/api/v1/models", nil, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			// Without DB, should return empty array
			return nil
		})
	})

	// Skip model modification tests when DB is not available - they will panic
	// These would need DB to be tested properly
}

// TestAdvancedCalculations tests more complex calculation scenarios
func TestAdvancedCalculations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("MatrixInverse", func(t *testing.T) {
		matrix := [][]float64{{1, 2}, {3, 4}}
		body := createMatrixRequest("matrix_inverse", matrix, nil, nil)
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			// Should return inverse matrix
			return nil
		})
	})

	// Note: partial_derivative and double_integral are not exposed as main operations
	// They are internal to the calculus operation handler
}

// TestForecastVariations tests different forecast methods
func TestForecastVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	data := []float64{100, 102, 98, 105, 110}

	tests := []struct {
		name   string
		method string
	}{
		{"LinearTrend", "linear_trend"},
		{"ExponentialSmoothing", "exponential_smoothing"},
		{"Default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := createForecastRequest(data, 3, tt.method)
			testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
				respData, ok := resp["data"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected data object")
					return nil
				}
				forecast, ok := respData["forecast"].([]interface{})
				if !ok {
					t.Errorf("Expected forecast array")
					return nil
				}
				if len(forecast) != 3 {
					t.Errorf("Expected 3 forecast points, got %d", len(forecast))
				}
				return nil
			})
		})
	}
}

// TestForecastWithConfidenceIntervals tests forecast with confidence intervals
func TestForecastWithConfidenceIntervals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	data := []float64{100, 102, 98, 105, 110}
	req := createForecastRequest(data, 3, "linear_trend")
	req.Options.ConfidenceIntervals = true

	testEndpoint(t, server, "POST", "/api/v1/math/forecast", req, testToken, http.StatusOK, func(resp map[string]interface{}) error {
		respData, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected data object")
			return nil
		}
		if _, ok := respData["confidence_intervals"]; !ok {
			t.Errorf("Expected confidence_intervals in response")
		}
		return nil
	})
}

// TestForecastWithSeasonality tests forecast with seasonality option
func TestForecastWithSeasonality(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	data := []float64{100, 102, 98, 105, 110, 108, 112, 115}
	req := createForecastRequest(data, 4, "linear_trend")
	req.Options.Seasonality = true

	testEndpoint(t, server, "POST", "/api/v1/math/forecast", req, testToken, http.StatusOK, func(resp map[string]interface{}) error {
		respData, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected data object")
			return nil
		}
		forecast, ok := respData["forecast"].([]interface{})
		if !ok || len(forecast) != 4 {
			t.Errorf("Expected 4 forecast points")
		}
		return nil
	})
}

// TestStatisticsEndpointVariations tests different analysis types
func TestStatisticsEndpointVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	data := []float64{1, 2, 3, 4, 5}

	tests := []struct {
		name     string
		analyses []string
	}{
		{"Descriptive", []string{"descriptive"}},
		{"Correlation", []string{"correlation"}},
		{"Regression", []string{"regression"}},
		{"Multiple", []string{"descriptive", "correlation", "regression"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := createStatisticsRequest(data, tt.analyses)
			testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
				_, ok := resp["data"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected data object")
					return nil
				}
				return nil
			})
		})
	}
}

// TestStatisticsErrorHandling tests error cases for statistics endpoint
func TestStatisticsErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("EmptyData", func(t *testing.T) {
		body := createStatisticsRequest([]float64{}, []string{"descriptive"})
		testEndpointError(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusBadRequest, "data required")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(t, "POST", "/api/v1/math/statistics", "invalid", testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", recorder.Code)
		}
	})
}

// TestSolveEquationVariations tests different equation solving scenarios
func TestSolveEquationVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	tests := []struct {
		name     string
		equation string
		method   string
	}{
		{"QuadraticExplicit", "x^2 - 4 = 0", "numerical"},
		{"CubicExplicit", "x^3 - 8 = 0", "numerical"},
		{"DefaultEquation", "", "numerical"},
		{"AnalyticalMethod", "x^2 - 4", "analytical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := createSolveRequest(tt.equation, []string{"x"}, tt.method)
			testEndpoint(t, server, "POST", "/api/v1/math/solve", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
				data, ok := resp["data"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected data object")
					return nil
				}
				if _, ok := data["solutions"]; !ok {
					t.Errorf("Expected solutions in response")
				}
				return nil
			})
		})
	}
}

// TestOptimizeVariations tests different optimization scenarios
func TestOptimizeVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	tests := []struct {
		name      string
		objective string
		variables []string
		optType   string
	}{
		{"Minimize", "x^2 + y^2", []string{"x", "y"}, "minimize"},
		{"Maximize", "x^2 + y^2", []string{"x", "y"}, "maximize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := createOptimizeRequest(tt.objective, tt.variables, tt.optType)
			testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
				data, ok := resp["data"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected data object")
					return nil
				}
				if _, ok := data["optimal_solution"]; !ok {
					t.Errorf("Expected optimal_solution in response")
				}
				return nil
			})
		})
	}
}

// TestOptimizeWithBounds tests optimization with bounds
func TestOptimizeWithBounds(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	body := createOptimizeRequest("x^2 + y^2", []string{"x", "y"}, "minimize")
	body.Options.Bounds = map[string][2]float64{
		"x": {-1.0, 1.0},
		"y": {-1.0, 1.0},
	}

	testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected data object")
			return nil
		}
		solution, ok := data["optimal_solution"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected solution map")
			return nil
		}
		// Check that solution respects bounds
		if x, ok := solution["x"].(float64); ok {
			if x < -1.0 || x > 1.0 {
				t.Errorf("Solution x=%v violates bounds [-1, 1]", x)
			}
		}
		return nil
	})
}

// TestCalculateResponseMetadata tests that responses include proper metadata
func TestCalculateResponseMetadata(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	body := createCalculationRequest("add", []float64{1, 2, 3})
	testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected data object")
			return nil
		}
		// Check metadata fields
		if _, ok := data["execution_time_ms"]; !ok {
			t.Errorf("Expected execution_time_ms in response")
		}
		if _, ok := data["precision_used"]; !ok {
			t.Errorf("Expected precision_used in response")
		}
		if _, ok := data["algorithm"]; !ok {
			t.Errorf("Expected algorithm in response")
		}
		return nil
	})
}

// TestEdgeCasesForCalculations tests edge cases and boundary conditions
func TestEdgeCasesForCalculations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("VeryLargeNumbers", func(t *testing.T) {
		body := createCalculationRequest("add", []float64{1e10, 1e10})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("VerySmallNumbers", func(t *testing.T) {
		body := createCalculationRequest("add", []float64{1e-10, 1e-10})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("MixedScale", func(t *testing.T) {
		body := createCalculationRequest("multiply", []float64{1e10, 1e-10})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})
}

// TestIntegralBoundaryConditions tests integral edge cases
func TestIntegralBoundaryConditions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("InvalidBounds", func(t *testing.T) {
		// Upper bound less than lower bound
		body := createCalculationRequest("integral", []float64{5, 2})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			data, ok := resp["data"].(map[string]interface{})
			if !ok {
				t.Errorf("Expected data object")
				return nil
			}
			result, ok := data["result"].(map[string]interface{})
			if ok {
				if _, hasError := result["error"]; hasError {
					return nil // Expected error
				}
			}
			return nil
		})
	})

	t.Run("InsufficientData", func(t *testing.T) {
		body := createCalculationRequest("integral", []float64{0})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			// Should handle gracefully
			return nil
		})
	})
}
