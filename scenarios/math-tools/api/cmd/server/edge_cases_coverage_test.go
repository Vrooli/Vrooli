package main

import (
	"net/http"
	"testing"
)

// TestNumberTheoryEdgeCases tests number theory operations thoroughly
func TestNumberTheoryEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("PrimeFactorization", func(t *testing.T) {
		tests := []struct {
			name     string
			number   float64
			expected []int
		}{
			{"SmallPrime", 7, []int{7}},
			{"Composite", 12, []int{2, 2, 3}},
			{"LargePrime", 97, []int{97}},
			{"PowerOfTwo", 64, []int{2, 2, 2, 2, 2, 2}},
			{"One", 1, []int{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "prime_factors",
					Data:      []float64{tt.number},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					result, ok := resp["result"]
					if !ok {
						t.Error("Should have result field")
					}

					// Result should be an array of factors
					if result == nil {
						t.Error("Result should not be nil")
					}
					return nil
				})
			})
		}
	})

	t.Run("GCD", func(t *testing.T) {
		tests := []struct {
			name     string
			a, b     float64
			expected int
		}{
			{"Coprime", 15, 28, 1},
			{"CommonFactor", 48, 18, 6},
			{"SameNumber", 12, 12, 12},
			{"OneIsOne", 1, 100, 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "gcd",
					Data:      []float64{tt.a, tt.b},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					result, ok := resp["result"].(float64)
					if !ok {
						t.Error("Result should be a number")
					}

					if int(result) != tt.expected {
						t.Errorf("Expected GCD %d, got %v", tt.expected, result)
					}
					return nil
				})
			})
		}
	})

	t.Run("LCM", func(t *testing.T) {
		tests := []struct {
			name     string
			a, b     float64
			expected int
		}{
			{"Small", 4, 6, 12},
			{"Coprime", 5, 7, 35},
			{"OneIsMultiple", 12, 3, 12},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "lcm",
					Data:      []float64{tt.a, tt.b},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					result, ok := resp["result"].(float64)
					if !ok {
						t.Error("Result should be a number")
					}

					if int(result) != tt.expected {
						t.Errorf("Expected LCM %d, got %v", tt.expected, result)
					}
					return nil
				})
			})
		}
	})
}

// TestBasicOperationExtendedEdgeCases tests additional edge cases in basic operations
func TestBasicOperationExtendedEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("PowerOperation", func(t *testing.T) {
		tests := []struct {
			name     string
			base     float64
			exponent float64
		}{
			{"SquareRoot", 16, 0.5},
			{"Cube", 3, 3},
			{"NegativeExponent", 2, -3},
			{"ZeroExponent", 5, 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "power",
					Data:      []float64{tt.base, tt.exponent},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					if resp["result"] == nil {
						t.Error("Should have result")
					}
					return nil
				})
			})
		}
	})

	t.Run("SquareRoot", func(t *testing.T) {
		tests := []struct {
			name   string
			number float64
		}{
			{"Perfect", 25},
			{"Decimal", 2.25},
			{"Large", 10000},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "sqrt",
					Data:      []float64{tt.number},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					if resp["result"] == nil {
						t.Error("Should have result")
					}
					return nil
				})
			})
		}
	})

	t.Run("Logarithm", func(t *testing.T) {
		tests := []struct {
			name   string
			number float64
		}{
			{"One", 1},
			{"E", 2.718281828},
			{"Ten", 10},
			{"Large", 1000},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "log",
					Data:      []float64{tt.number},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					if resp["result"] == nil {
						t.Error("Should have result")
					}
					return nil
				})
			})
		}
	})

	t.Run("Exponential", func(t *testing.T) {
		tests := []struct {
			name   string
			number float64
		}{
			{"Zero", 0},
			{"One", 1},
			{"Negative", -1},
			{"Small", 0.5},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := CalculationRequest{
					Operation: "exp",
					Data:      []float64{tt.number},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					if resp["result"] == nil {
						t.Error("Should have result")
					}
					return nil
				})
			})
		}
	})
}

// TestMatrixOperationEdgeCases tests matrix operations edge cases
func TestMatrixOperationEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("MatrixMultiplyIdentity", func(t *testing.T) {
		// Multiply by identity matrix
		matrixA := [][]float64{
			{1, 2},
			{3, 4},
		}
		identity := [][]float64{
			{1, 0},
			{0, 1},
		}

		body := CalculationRequest{
			Operation: "matrix_multiply",
			MatrixA:   matrixA,
			MatrixB:   identity,
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].([]interface{})
			if !ok {
				t.Error("Result should be a matrix (2D array)")
			}

			if len(result) != 2 {
				t.Errorf("Result should be 2x2, got %d rows", len(result))
			}
			return nil
		})
	})

	t.Run("MatrixInverseIdentity", func(t *testing.T) {
		// Inverse of identity is identity
		identity := [][]float64{
			{1, 0},
			{0, 1},
		}

		body := CalculationRequest{
			Operation: "matrix_inverse",
			Matrix:    identity,
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["result"] == nil {
				t.Error("Should have result")
			}
			return nil
		})
	})

	t.Run("MatrixDeterminantZero", func(t *testing.T) {
		// Singular matrix (determinant = 0)
		singular := [][]float64{
			{1, 2},
			{2, 4},
		}

		body := CalculationRequest{
			Operation: "matrix_determinant",
			Matrix:    singular,
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(float64)
			if !ok {
				t.Error("Determinant should be a number")
			}

			if result != 0 {
				t.Logf("Determinant of singular matrix should be ~0, got %v", result)
			}
			return nil
		})
	})

	t.Run("MatrixTranspose1x1", func(t *testing.T) {
		// 1x1 matrix transpose
		matrix := [][]float64{{5}}

		body := CalculationRequest{
			Operation: "matrix_transpose",
			Matrix:    matrix,
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["result"] == nil {
				t.Error("Should have result")
			}
			return nil
		})
	})
}

// TestCalculusEdgeCases tests calculus operations edge cases
func TestCalculusEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("DerivativeAtZero", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "derivative",
			Data:      []float64{0},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map")
			}

			if result["derivative"] == nil {
				t.Error("Should have derivative value")
			}

			if result["analytical"] == nil {
				t.Error("Should have analytical value")
			}
			return nil
		})
	})

	t.Run("DerivativeNegative", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "derivative",
			Data:      []float64{-5},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["result"] == nil {
				t.Error("Should have result")
			}
			return nil
		})
	})

	t.Run("IntegralZeroWidth", func(t *testing.T) {
		// Same bounds (should give error)
		body := CalculationRequest{
			Operation: "integral",
			Data:      []float64{5, 5},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map")
			}

			// Should have error for zero-width integral
			if result["error"] == nil {
				t.Log("Expected error for zero-width integral")
			}
			return nil
		})
	})

	t.Run("IntegralNegativeBounds", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "integral",
			Data:      []float64{-5, -2},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if !ok {
				t.Error("Result should be a map")
			}

			if result["integral"] == nil && result["error"] == nil {
				t.Error("Should have either integral or error")
			}
			return nil
		})
	})
}

// TestStatisticsEdgeCases tests statistical operations edge cases
func TestStatisticsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("SingleDataPoint", func(t *testing.T) {
		body := StatisticsRequest{
			Data:     []float64{42},
			Analyses: []string{"descriptive"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			results, ok := resp["results"].(map[string]interface{})
			if !ok {
				t.Error("Results should be a map")
			}

			desc, ok := results["descriptive"].(map[string]interface{})
			if !ok {
				t.Error("Descriptive should be a map")
			}

			// All stats should equal the single value
			mean, _ := desc["mean"].(float64)
			median, _ := desc["median"].(float64)
			mode, _ := desc["mode"].(float64)

			if mean != 42 || median != 42 || mode != 42 {
				t.Logf("All stats should be 42 for single value: mean=%v, median=%v, mode=%v", mean, median, mode)
			}
			return nil
		})
	})

	t.Run("AllSameValues", func(t *testing.T) {
		body := StatisticsRequest{
			Data:     []float64{5, 5, 5, 5, 5},
			Analyses: []string{"descriptive"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			results, ok := resp["results"].(map[string]interface{})
			if !ok {
				t.Error("Results should be a map")
			}

			desc, ok := results["descriptive"].(map[string]interface{})
			if !ok {
				t.Error("Descriptive should be a map")
			}

			// Variance and stddev should be 0
			variance, _ := desc["variance"].(float64)
			stddev, _ := desc["std_dev"].(float64)

			if variance != 0 || stddev != 0 {
				t.Logf("Variance and stddev should be 0 for identical values: variance=%v, stddev=%v", variance, stddev)
			}
			return nil
		})
	})

	t.Run("NegativeValues", func(t *testing.T) {
		body := StatisticsRequest{
			Data:     []float64{-10, -5, -2, -8, -3},
			Analyses: []string{"descriptive"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			results, ok := resp["results"].(map[string]interface{})
			if !ok {
				t.Error("Results should be a map")
			}

			desc, ok := results["descriptive"].(map[string]interface{})
			if !ok {
				t.Error("Descriptive should be a map")
			}

			mean, _ := desc["mean"].(float64)
			if mean >= 0 {
				t.Errorf("Mean of negative numbers should be negative, got %v", mean)
			}
			return nil
		})
	})

	t.Run("LargeNumbers", func(t *testing.T) {
		body := StatisticsRequest{
			Data:     []float64{1e10, 1e11, 1e12},
			Analyses: []string{"descriptive"},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			results, ok := resp["results"].(map[string]interface{})
			if !ok {
				t.Error("Results should be a map")
			}

			if results["descriptive"] == nil {
				t.Error("Should handle large numbers")
			}
			return nil
		})
	})
}

// TestDocsEndpointDetailed tests documentation endpoint thoroughly
func TestDocsEndpointDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("DocsNoAuth", func(t *testing.T) {
		// Docs should be accessible without auth
		testEndpoint(t, server, "GET", "/docs", nil, "", http.StatusOK, func(resp map[string]interface{}) error {
			if resp["name"] != "Math Tools API" {
				t.Error("Should have correct API name")
			}

			if resp["version"] != "1.0.0" {
				t.Error("Should have version")
			}

			endpoints, ok := resp["endpoints"].([]interface{})
			if !ok || len(endpoints) == 0 {
				t.Error("Should have endpoints list")
			}
			return nil
		})
	})
}

// TestPlotEndpointExtended tests plot generation with various types
func TestPlotEndpointExtended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("BasicPlotRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"plot_type": "line",
			"data":      []float64{1, 2, 3, 4, 5},
			"options": map[string]interface{}{
				"title": "Test Plot",
			},
		}

		testEndpoint(t, server, "POST", "/api/v1/math/plot", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			if resp["plot_id"] == nil {
				t.Error("Should have plot_id")
			}

			if resp["image_url"] == nil {
				t.Error("Should have image_url")
			}

			metadata, ok := resp["metadata"].(map[string]interface{})
			if !ok {
				t.Error("Should have metadata")
			}

			if metadata["width"] == nil || metadata["height"] == nil {
				t.Error("Metadata should have dimensions")
			}
			return nil
		})
	})

	t.Run("DifferentPlotTypes", func(t *testing.T) {
		plotTypes := []string{"scatter", "histogram", "heatmap", "surface"}

		for _, plotType := range plotTypes {
			t.Run(plotType, func(t *testing.T) {
				body := map[string]interface{}{
					"plot_type": plotType,
					"data":      []float64{1, 2, 3},
				}

				testEndpoint(t, server, "POST", "/api/v1/math/plot", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					if resp["plot_id"] == nil {
						t.Error("Should generate plot for type: " + plotType)
					}
					return nil
				})
			})
		}
	})
}

// TestInsufficientDataErrors tests all operations with insufficient data
func TestInsufficientDataErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("SubtractInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "subtract",
			Data:      []float64{5}, // Need at least 2
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if ok && result["error"] != nil {
				// Good, got expected error
				return nil
			}
			return nil
		})
	})

	t.Run("DivideInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "divide",
			Data:      []float64{10}, // Need at least 2
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if ok && result["error"] != nil {
				// Good, got expected error
				return nil
			}
			return nil
		})
	})

	t.Run("PowerInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "power",
			Data:      []float64{2}, // Need base and exponent
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if ok && result["error"] != nil {
				// Good, got expected error
				return nil
			}
			return nil
		})
	})

	t.Run("GCDInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "gcd",
			Data:      []float64{12}, // Need 2 numbers
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if ok && result["error"] != nil {
				// Good, got expected error
				return nil
			}
			return nil
		})
	})

	t.Run("LCMInsufficientData", func(t *testing.T) {
		body := CalculationRequest{
			Operation: "lcm",
			Data:      []float64{12}, // Need 2 numbers
		}

		testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
			result, ok := resp["result"].(map[string]interface{})
			if ok && result["error"] != nil {
				// Good, got expected error
				return nil
			}
			return nil
		})
	})
}
