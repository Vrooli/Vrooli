package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// executeRequest executes an HTTP request against the server
func executeRequest(server *Server, req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	return recorder
}

// parseResponse parses a JSON response
func parseResponse(t *testing.T, recorder *httptest.ResponseRecorder) Response {
	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	return resp
}

// TestScenarioBuilderPatterns tests the TestScenarioBuilder systematically
func TestScenarioBuilderPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("TestScenarioBuilderUsage", func(t *testing.T) {
		// Build comprehensive test scenarios using the builder
		scenarios := NewTestScenarioBuilder().
			AddUnauthorized("/api/v1/math/calculate").
			AddInvalidToken("/api/v1/math/statistics").
			AddInvalidJSON("/api/v1/math/solve", "POST", testToken).
			AddMissingRequiredField("/api/v1/math/calculate", "POST", testToken, map[string]interface{}{}).
			AddScenario(TestScenario{
				Name:           "Custom scenario for optimization",
				Method:         "POST",
				Path:           "/api/v1/math/optimize",
				Body:           map[string]interface{}{"invalid": "data"},
				Token:          testToken,
				ExpectedStatus: http.StatusBadRequest,
			}).
			Build()

		// Execute all scenarios
		for _, scenario := range scenarios {
			t.Run(scenario.Name, func(t *testing.T) {
				req := makeHTTPRequest(t, scenario.Method, scenario.Path, scenario.Body, scenario.Token)
				recorder := executeRequest(server, req)

				if recorder.Code != scenario.ExpectedStatus {
					t.Errorf("Expected status %d, got %d for scenario: %s",
						scenario.ExpectedStatus, recorder.Code, scenario.Name)
				}

				if scenario.ErrorSubstring != "" {
					resp := parseResponse(t, recorder)
					if resp.Error == "" || !contains(resp.Error, scenario.ErrorSubstring) {
						t.Errorf("Expected error containing '%s', got: %s",
							scenario.ErrorSubstring, resp.Error)
					}
				}
			})
		}
	})
}

// TestErrorTestPatterns tests systematic error patterns
func TestErrorTestPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	pattern := NewErrorTestPattern(t, server, testToken)

	t.Run("CalculateEndpoint", func(t *testing.T) {
		pattern.TestAuthErrors("/api/v1/math/calculate", "POST")
		pattern.TestInvalidJSON("/api/v1/math/calculate", "POST")
		pattern.TestEmptyBody("/api/v1/math/calculate", "POST")
		pattern.TestInvalidOperation("/api/v1/math/calculate")
	})

	t.Run("StatisticsEndpoint", func(t *testing.T) {
		pattern.TestAuthErrors("/api/v1/math/statistics", "POST")
		pattern.TestEmptyData("/api/v1/math/statistics", "POST")
		pattern.TestInsufficientData("/api/v1/math/statistics", "POST")
	})

	t.Run("CalculusOperations", func(t *testing.T) {
		pattern.TestEmptyData("/api/v1/math/calculate", "POST")
		pattern.TestInsufficientData("/api/v1/math/calculate", "POST")
		pattern.TestDivisionByZero("/api/v1/math/calculate")
		pattern.TestNegativeLogarithm("/api/v1/math/calculate")
	})
}

// TestModelManagementEndpoints tests CRUD operations for models
func TestModelManagementEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ListModels", func(t *testing.T) {
		testEndpoint(t, server, "GET", "/api/v1/models", nil, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			// Should return empty array when no DB
			return nil
		})
	})

	// Note: Model CRUD operations require DB connection
	// Skipped in tests since we don't have a test database
	// The ListModels test above confirms the endpoint exists and auth works
}

// TestAdvancedCalculusOperations tests all calculus operations thoroughly
func TestAdvancedCalculusOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("PartialDerivative", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "partial_derivative",
			Data:      []float64{3.0, 4.0},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map for partial derivatives")
			}

			partials, ok := result["partial_derivatives"].(map[string]interface{})
			if !ok {
				t.Error("Should have partial_derivatives field")
			}

			if partials["df_dx"] == nil || partials["df_dy"] == nil {
				t.Error("Should have both df_dx and df_dy")
			}
			return nil
		})
	})

	t.Run("DoubleIntegral", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "double_integral",
			Data:      []float64{0, 1, 0, 1}, // x_min, x_max, y_min, y_max
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map for double integral")
			}

			if result["method"] != "simpson_2d" {
				t.Error("Should use Simpson's 2D method")
			}

			if result["integral"] == nil {
				t.Error("Should have integral value")
			}
			return nil
		})
	})

	t.Run("DoubleIntegralInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "double_integral",
			Data:      []float64{0, 1}, // Not enough bounds
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map")
			}

			if result["error"] == nil {
				t.Error("Should have error for insufficient data")
			}
			return nil
		})
	})
}

// TestOptimizationEdgeCases tests optimization with various scenarios
func TestOptimizationEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("OptimizationWithBoundsEnforcement", func(t *testing.T) {
		body := OptimizeRequest{
			ObjectiveFunction: "x^2 + y^2",
			Variables:         []string{"x", "y"},
			OptimizationType:  "minimize",
			Algorithm:         "gradient_descent",
		}
		body.Options.Tolerance = 1e-6
		body.Options.MaxIterations = 100
		body.Options.Bounds = map[string][2]float64{
			"x": {-10, 10},
			"y": {-10, 10},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["optimal_solution"] == nil {
				t.Error("Should have optimal_solution")
			}

			if resp["sensitivity_analysis"] == nil {
				t.Error("Should have sensitivity_analysis")
			}

			sensitivity, ok := resp["sensitivity_analysis"].(map[string]interface{})
			if !ok {
				t.Error("sensitivity_analysis should be a map")
			}

			if sensitivity["gradient"] == nil {
				t.Error("Should have gradient in sensitivity analysis")
			}
			return nil
		})
	})

	t.Run("OptimizationMaximize", func(t *testing.T) {
		body := OptimizeRequest{
			ObjectiveFunction: "-x^2 - y^2",
			Variables:         []string{"x", "y"},
			OptimizationType:  "maximize",
		}

		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["status"] == nil {
				t.Error("Should have status field")
			}
			return nil
		})
	})
}

// TestForecastingMethods tests all forecasting methods
func TestForecastingMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	timeSeries := []interface{}{100.0, 102.0, 98.0, 105.0, 110.0, 108.0, 112.0, 115.0, 118.0, 120.0}

	t.Run("LinearTrendForecast", func(t *testing.T) {
		body := ForecastRequest{
			TimeSeries:      timeSeries,
			ForecastHorizon: 5,
			Method:          "linear_trend",
		}
		body.Options.ConfidenceIntervals = true

		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["forecast"] == nil {
				t.Error("Should have forecast")
			}

			if resp["confidence_intervals"] == nil {
				t.Error("Should have confidence intervals when requested")
			}

			ci, ok := resp["confidence_intervals"].(map[string]interface{})
			if !ok {
				t.Error("confidence_intervals should be a map")
			}

			if ci["lower"] == nil || ci["upper"] == nil {
				t.Error("Should have both lower and upper bounds")
			}
			return nil
		})
	})

	t.Run("ExponentialSmoothingForecast", func(t *testing.T) {
		body := ForecastRequest{
			TimeSeries:      timeSeries,
			ForecastHorizon: 3,
			Method:          "exponential_smoothing",
		}
		body.Options.ConfidenceIntervals = true

		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			forecast, ok := resp["forecast"].([]interface{})
			if !ok {
				t.Error("forecast should be an array")
			}

			if len(forecast) != 3 {
				t.Errorf("Expected 3 forecast points, got %d", len(forecast))
			}
			return nil
		})
	})

	t.Run("MovingAverageForecast", func(t *testing.T) {
		body := ForecastRequest{
			TimeSeries:      timeSeries,
			ForecastHorizon: 5,
			Method:          "moving_average",
		}

		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["model_metrics"] == nil {
				t.Error("Should have model_metrics")
			}

			metrics, ok := resp["model_metrics"].(map[string]interface{})
			if !ok {
				t.Error("model_metrics should be a map")
			}

			if metrics["mae"] == nil || metrics["mse"] == nil || metrics["mape"] == nil {
				t.Error("Should have mae, mse, and mape metrics")
			}
			return nil
		})
	})

	t.Run("ForecastWithSeasonality", func(t *testing.T) {
		body := ForecastRequest{
			TimeSeries:      timeSeries,
			ForecastHorizon: 4,
			Method:          "linear_trend",
		}
		body.Options.Seasonality = true

		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			params, ok := resp["model_parameters"].(map[string]interface{})
			if !ok {
				t.Error("Should have model_parameters")
			}

			if params["method"] != "linear_trend" {
				t.Error("Should preserve method in parameters")
			}
			return nil
		})
	})

	t.Run("ForecastShortTimeSeries", func(t *testing.T) {
		body := ForecastRequest{
			TimeSeries:      []interface{}{100.0, 102.0},
			ForecastHorizon: 3,
			Method:          "moving_average",
		}

		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			// Should handle short time series gracefully
			if resp["forecast"] == nil {
				t.Error("Should still produce forecast for short series")
			}
			return nil
		})
	})
}

// TestStatisticsAllAnalyses tests all statistical analyses at once
func TestStatisticsAllAnalyses(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("AllAnalysesAtOnce", func(t *testing.T) {
		body := StatisticsRequest{
			Data:     data,
			Analyses: []string{"descriptive", "correlation", "regression"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			results, ok := resp["results"].(map[string]interface{})
			if !ok {
				t.Error("results should be a map")
			}

			if results["descriptive"] == nil {
				t.Error("Should have descriptive statistics")
			}

			if results["correlation"] == nil {
				t.Error("Should have correlation")
			}

			if results["regression"] == nil {
				t.Error("Should have regression")
			}

			desc, ok := results["descriptive"].(map[string]interface{})
			if !ok {
				t.Error("descriptive should be a map")
			}

			// Check all descriptive stats fields
			required := []string{"mean", "median", "mode", "std_dev", "variance", "min", "max", "count", "sum"}
			for _, field := range required {
				if desc[field] == nil {
					t.Errorf("descriptive stats should have %s", field)
				}
			}
			return nil
		})
	})
}

// TestSolveEquationComprehensive tests equation solving with various inputs
func TestSolveEquationComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ArrayEquationFormat", func(t *testing.T) {
		body := SolveRequest{
			Equations: []interface{}{"x^2 - 4 = 0"},
			Variables: []string{"x"},
			Method:    "numerical",
		}

		testEndpoint(t, server, "POST", "/api/v1/math/solve", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			solutions, ok := resp["solutions"].([]interface{})
			if !ok {
				t.Error("solutions should be an array")
			}

			if len(solutions) == 0 {
				t.Error("Should have at least one solution")
			}
			return nil
		})
	})

	t.Run("CubicEquation", func(t *testing.T) {
		body := SolveRequest{
			Equations: "x^3 - 8 = 0",
			Variables: []string{"x"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/solve", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			solutionType, ok := resp["solution_type"].(string)
			if !ok {
				t.Error("solution_type should be a string")
			}

			if solutionType != "unique" && solutionType != "multiple" {
				t.Logf("Got solution_type: %s", solutionType)
			}
			return nil
		})
	})

	t.Run("ConvergenceInfo", func(t *testing.T) {
		body := SolveRequest{
			Equations: "random",
			Variables: []string{"x"},
		}
		body.Options.Tolerance = 1e-8
		body.Options.MaxIterations = 500

		testEndpoint(t, server, "POST", "/api/v1/math/solve", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			convInfo, ok := resp["convergence_info"].(map[string]interface{})
			if !ok {
				t.Error("Should have convergence_info")
			}

			if convInfo["converged"] == nil {
				t.Error("Should have converged field")
			}

			if convInfo["iterations"] == nil {
				t.Error("Should have iterations field")
			}

			if convInfo["final_error"] == nil {
				t.Error("Should have final_error field")
			}
			return nil
		})
	})
}
