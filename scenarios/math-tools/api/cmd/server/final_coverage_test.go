package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCalculusComprehensive tests all calculus operation paths
func TestCalculusComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("PartialDerivativeOperation", func(t *testing.T) {
		// This exercises the partial_derivative case in performCalculusOperation
		body := CalculationRequest{
			Operation: "partial_derivative",
			Data:      []float64{1, 2},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusBadRequest, nil)
	})

	t.Run("DoubleIntegralOperation", func(t *testing.T) {
		// This exercises the double_integral case
		body := CalculationRequest{
			Operation: "double_integral",
			Data:      []float64{0, 1, 0, 1},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusBadRequest, nil)
	})

	t.Run("DerivativeNil", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "derivative",
			Data:      nil,
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("IntegralSingleValue", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "integral",
			Data:      []float64{5},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("IntegralReversedBounds", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "integral",
			Data:      []float64{10, 5},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
	})

	t.Run("UnknownCalculusOperation", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "laplacian",
			Data:      []float64{1, 2},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusBadRequest, nil)
	})
}

// TestOptimizeComprehensive tests all optimize endpoint paths
func TestOptimizeComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("OptimizeInvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(t, "POST", "/api/v1/math/optimize", "bad-json", testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for bad JSON, got %d", recorder.Code)
		}
	})

	t.Run("OptimizeDefaultValues", func(t *testing.T) {
		body := OptimizeRequest{
			ObjectiveFunction: "x^2",
			Variables:        []string{"x"},
			OptimizationType: "minimize",
			// Leave Options at defaults
		}
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("OptimizeWithAlgorithm", func(t *testing.T) {
		body := OptimizeRequest{
			ObjectiveFunction: "x^2 + y^2",
			Variables:        []string{"x", "y"},
			OptimizationType: "maximize",
			Algorithm:        "custom_algo",
		}
		body.Options.Tolerance = 0.001
		body.Options.MaxIterations = 50
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("OptimizeSingleVariable", func(t *testing.T) {
		body := createOptimizeRequest("x^2", []string{"x"}, "minimize")
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("OptimizeMultipleVariablesWithBounds", func(t *testing.T) {
		body := createOptimizeRequest("x^2 + y^2 + z^2", []string{"x", "y", "z"}, "minimize")
		body.Options.Bounds = map[string][2]float64{
			"x": {-2.0, 2.0},
			"y": {-3.0, 3.0},
			"z": {-1.0, 1.0},
		}
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})

	t.Run("OptimizeConvergence", func(t *testing.T) {
		body := createOptimizeRequest("x^2", []string{"x"}, "minimize")
		body.Options.Tolerance = 0.000001
		body.Options.MaxIterations = 1000
		testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
	})
}

// TestForecastComprehensive tests all forecast paths
func TestForecastComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ForecastInvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(t, "POST", "/api/v1/math/forecast", "invalid", testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for bad JSON, got %d", recorder.Code)
		}
	})

	t.Run("ForecastDifferentDataFormats", func(t *testing.T) {
		// Test with interface{} data
		req := ForecastRequest{
			TimeSeries:      map[string]string{"dataset_id": "test"},
			ForecastHorizon: 3,
			Method:          "linear_trend",
		}
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", req, testToken, http.StatusOK, nil)
	})

	t.Run("ForecastWithAllOptions", func(t *testing.T) {
		data := []float64{100, 102, 98, 105, 110, 108, 112, 115}
		req := createForecastRequest(data, 5, "exponential_smoothing")
		req.Options.Seasonality = true
		req.Options.ConfidenceIntervals = true
		req.Options.ValidationSplit = 0.2
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", req, testToken, http.StatusOK, nil)
	})

	t.Run("ForecastLongTimeSeries", func(t *testing.T) {
		data := make([]float64, 50)
		for i := range data {
			data[i] = float64(i) + 100
		}
		body := createForecastRequest(data, 10, "linear_trend")
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})

	t.Run("ForecastExponentialSmoothing", func(t *testing.T) {
		data := []float64{100, 105, 103, 108, 110}
		body := createForecastRequest(data, 3, "exponential_smoothing")
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})

	t.Run("ForecastMovingAverage", func(t *testing.T) {
		data := []float64{100, 105, 103, 108}
		body := createForecastRequest(data, 2, "moving_average")
		testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
	})
}

// TestStatisticsComprehensive tests statistics endpoint thoroughly
func TestStatisticsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("StatisticsWithOptions", func(t *testing.T) {
		req := StatisticsRequest{
			Data:     []float64{1, 2, 3, 4, 5},
			Analyses: []string{"descriptive", "correlation"},
		}
		req.Options.ConfidenceLevel = 0.95
		req.Options.Alpha = 0.05
		req.Options.Method = "pearson"
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", req, testToken, http.StatusOK, nil)
	})

	t.Run("StatisticsAllAnalyses", func(t *testing.T) {
		body := createStatisticsRequest([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, []string{"descriptive", "correlation", "regression"})
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, nil)
	})

	t.Run("StatisticsLargeDataset", func(t *testing.T) {
		data := make([]float64, 100)
		for i := range data {
			data[i] = float64(i)
		}
		body := createStatisticsRequest(data, []string{"descriptive"})
		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, nil)
	})
}

// TestSolveComprehensive tests solve endpoint thoroughly
func TestSolveComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("SolveInvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(t, "POST", "/api/v1/math/solve", "invalid", testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for bad JSON, got %d", recorder.Code)
		}
	})

	t.Run("SolveWithCustomOptions", func(t *testing.T) {
		req := createSolveRequest("x^2 - 9 = 0", []string{"x"}, "numerical")
		req.Options.Tolerance = 0.0001
		req.Options.MaxIterations = 500
		testEndpoint(t, server, "POST", "/api/v1/math/solve", req, testToken, http.StatusOK, nil)
	})

	t.Run("SolveWithConstraints", func(t *testing.T) {
		req := createSolveRequest("x^2 - 4", []string{"x"}, "analytical")
		req.Constraints = []string{"x > 0"}
		testEndpoint(t, server, "POST", "/api/v1/math/solve", req, testToken, http.StatusOK, nil)
	})
}

// TestPlotComprehensive tests plot endpoint thoroughly
func TestPlotComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("PlotInvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(t, "POST", "/api/v1/math/plot", "invalid", testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for bad JSON, got %d", recorder.Code)
		}
	})

	t.Run("PlotWithMetadata", func(t *testing.T) {
		body := map[string]interface{}{
			"type":   "scatter",
			"data":   []float64{1, 4, 9, 16, 25},
			"title":  "Test Plot",
			"xlabel": "X Axis",
			"ylabel": "Y Axis",
		}
		testEndpoint(t, server, "POST", "/api/v1/math/plot", body, testToken, http.StatusOK, nil)
	})
}

// TestUnhandledOperations tests operations that fall through to default case
func TestUnhandledOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("UnknownOperation", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "unknown_op",
			Data:      []float64{1, 2, 3},
		}
		testEndpointError(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusBadRequest, "unsupported operation")
	})

	t.Run("MatrixUnknownOperation", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "matrix_unknown",
			Matrix:    [][]float64{{1, 2}, {3, 4}},
		}
		testEndpointError(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusBadRequest, "unsupported operation")
	})
}

// TestCORSAndMiddleware tests CORS and middleware more thoroughly
func TestCORSAndMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, nil)
	defer cleanup()

	t.Run("CORSOnPOST", func(t *testing.T) {
		body := createCalculationRequest("add", []float64{1, 2})
		req := makeHTTPRequest(t, "POST", "/api/v1/math/calculate", body, testToken)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)

		if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS header *, got %s", origin)
		}
	})

	t.Run("CORSAllowMethods", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)

		if methods := recorder.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Errorf("Expected CORS allow methods header")
		}
	})
}
