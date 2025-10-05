package main

import (
	"net/http"
	"testing"
)

// TestNewServerInitialization tests server initialization
func TestNewServerInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ServerInitialization", func(t *testing.T) {
		server, cleanup := setupTestServer(t, nil)
		defer cleanup()

		if server == nil {
			t.Fatal("Server should not be nil")
		}
		if server.router == nil {
			t.Fatal("Router should be initialized")
		}
		if server.config == nil {
			t.Fatal("Config should be initialized")
		}
	})
}

// TestHelperUtilities tests helper utility functions
func TestHelperUtilities(t *testing.T) {
	t.Run("getEnv_WithValue", func(t *testing.T) {
		t.Setenv("TEST_VAR", "test-value")
		result := getEnv("TEST_VAR", "default")
		if result != "test-value" {
			t.Errorf("Expected test-value, got %s", result)
		}
	})

	t.Run("getEnv_WithDefault", func(t *testing.T) {
		result := getEnv("NON_EXISTENT_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("Expected default-value, got %s", result)
		}
	})

	t.Run("flatten", func(t *testing.T) {
		matrix := [][]float64{{1, 2}, {3, 4}}
		flat := flatten(matrix)
		expected := []float64{1, 2, 3, 4}
		if len(flat) != len(expected) {
			t.Errorf("Expected length %d, got %d", len(expected), len(flat))
		}
		for i, v := range expected {
			if !floatEquals(flat[i], v) {
				t.Errorf("Expected %v at index %d, got %v", v, i, flat[i])
			}
		}
	})

	t.Run("gcd", func(t *testing.T) {
		s := &Server{}
		result := s.gcd(48, 18)
		if result != 6 {
			t.Errorf("Expected gcd(48,18)=6, got %d", result)
		}
	})

	t.Run("primeFactorization", func(t *testing.T) {
		s := &Server{}
		factors := s.primeFactorization(60)
		// 60 = 2 * 2 * 3 * 5
		if len(factors) != 4 {
			t.Errorf("Expected 4 factors, got %d: %v", len(factors), factors)
		}
	})

	t.Run("calculateMode", func(t *testing.T) {
		s := &Server{}
		data := []float64{1, 2, 2, 3, 3, 3, 4}
		mode := s.calculateMode(data)
		if mode != 3 {
			t.Errorf("Expected mode=3, got %v", mode)
		}
	})
}

// TestCalculusExtended tests extended calculus operations
func TestCalculusExtended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("Derivative_SinglePoint", func(t *testing.T) {
		body := createCalculationRequest("derivative", []float64{3.5})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("Integral_ValidBounds", func(t *testing.T) {
		body := createCalculationRequest("integral", []float64{1, 5})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("Integral_SinglePoint", func(t *testing.T) {
		body := createCalculationRequest("integral", []float64{0})
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})
}

// TestCalculationStorage tests calculation storage
func TestCalculationStorage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, nil)
	defer cleanup()

	t.Run("StoreCalculation_NoDatabase", func(t *testing.T) {
		req := CalculationRequest{
			Operation: "add",
			Data:      []float64{1, 2, 3},
		}
		// Should not panic with nil database
		server.storeCalculation(req, 6.0, 0)
	})
}

// TestHealthComplete tests health endpoint thoroughly
func TestHealthComplete(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, nil)
	defer cleanup()

	t.Run("Health_AllFields", func(t *testing.T) {
		testEndpoint(t, server, "GET", "/health", nil, "", http.StatusOK, func(resp map[string]interface{}) error {
			data, ok := resp["data"].(map[string]interface{})
			if !ok {
				t.Errorf("Expected data object")
				return nil
			}
			
			// Check all expected fields
			expectedFields := []string{"status", "timestamp", "service", "version", "database"}
			for _, field := range expectedFields {
				if _, ok := data[field]; !ok {
					t.Errorf("Missing field: %s", field)
				}
			}
			return nil
		})
	})
}

// TestOptimizationUtilities tests optimization helper functions
func TestOptimizationUtilities(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("evaluateObjective", func(t *testing.T) {
		s := &Server{}
		point := map[string]float64{"x": 3, "y": 4}
		value := s.evaluateObjective(point, "x^2 + y^2")
		// For x=3, y=4: 3^2 + 4^2 = 9 + 16 = 25
		if !floatEquals(value, 25.0) {
			t.Errorf("Expected 25.0, got %v", value)
		}
	})

	t.Run("calculateGradient", func(t *testing.T) {
		s := &Server{}
		point := map[string]float64{"x": 5, "y": 3}
		gradient := s.calculateGradient(point)
		
		// For f(x,y) = x^2 + y^2, gradient is [2x, 2y]
		if gx, ok := gradient["x"]; !ok || !floatEquals(gx, 10.0) {
			t.Errorf("Expected gradient x=10.0, got %v", gradient["x"])
		}
		if gy, ok := gradient["y"]; !ok || !floatEquals(gy, 6.0) {
			t.Errorf("Expected gradient y=6.0, got %v", gradient["y"])
		}
	})
}

// TestSolveEquationVariants tests equation solving edge cases
func TestSolveEquationVariants(t *testing.T) {
	s := &Server{}

	t.Run("QuadraticEquation", func(t *testing.T) {
		solutions, solutionType, iterations, converged, err := s.solveEquation("x^2 - 4 = 0", 1e-6, 1000)
		if !converged {
			t.Errorf("Expected convergence")
		}
		if len(solutions) == 0 {
			t.Errorf("Expected solutions")
		}
		if iterations == 0 {
			t.Errorf("Expected non-zero iterations")
		}
		if err < 0 {
			t.Errorf("Error should be non-negative")
		}
		if solutionType == "" {
			t.Errorf("Expected solution type")
		}
	})

	t.Run("CubicEquation", func(t *testing.T) {
		solutions, _, _, _, _ := s.solveEquation("x^3 - 8 = 0", 1e-6, 1000)
		if len(solutions) == 0 {
			t.Errorf("Expected at least one solution")
		}
	})

	t.Run("UnknownEquation", func(t *testing.T) {
		_, _, iterations, _, _ := s.solveEquation("unknown equation", 1e-6, 100)
		if iterations == 0 {
			t.Errorf("Should attempt iterations")
		}
	})
}

// TestForecastingVariants tests forecasting edge cases
func TestForecastingVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ShortTimeSeries", func(t *testing.T) {
		body := createForecastRequest([]float64{100, 102}, 3, "linear_trend")
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})

	t.Run("MovingAverage", func(t *testing.T) {
		body := createForecastRequest([]float64{100, 105, 110}, 5, "moving_average")
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})

	t.Run("WithSeasonality", func(t *testing.T) {
		body := createForecastRequest([]float64{100, 105, 110, 115, 120}, 4, "linear_trend")
		body.Options.Seasonality = true
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})

	t.Run("WithSeasonality_SmallDataset", func(t *testing.T) {
		body := createForecastRequest([]float64{100, 102}, 2, "linear_trend")
		body.Options.Seasonality = true // Should skip seasonality with < 4 data points
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})
}

// TestOptimizationVariants tests optimization edge cases
func TestOptimizationVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("WithBounds", func(t *testing.T) {
		body := createOptimizeRequest("x^2 + y^2", []string{"x", "y"}, "minimize")
		body.Options.Bounds = map[string][2]float64{
			"x": {-10, 10},
			"y": {-10, 10},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("BoundsEnforcement", func(t *testing.T) {
		body := createOptimizeRequest("x + y", []string{"x", "y"}, "minimize")
		body.Options.Bounds = map[string][2]float64{
			"x": {-5, 5},
			"y": {-5, 5},
		}
		body.Options.MaxIterations = 10
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("Maximize", func(t *testing.T) {
		body := createOptimizeRequest("-(x^2 + y^2)", []string{"x", "y"}, "maximize")
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})
}

// TestStatisticsExtended tests extended statistics
func TestStatisticsExtended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("CorrelationAnalysis", func(t *testing.T) {
		body := createStatisticsRequest([]float64{1, 2, 3, 4, 5}, []string{"correlation"})
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, nil)
	})

	t.Run("RegressionAnalysis", func(t *testing.T) {
		body := createStatisticsRequest([]float64{1, 2, 3, 4, 5}, []string{"regression"})
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, nil)
	})

	t.Run("MultipleAnalyses", func(t *testing.T) {
		body := createStatisticsRequest([]float64{1, 2, 3, 4, 5}, []string{"descriptive", "correlation", "regression"})
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, nil)
	})
}
