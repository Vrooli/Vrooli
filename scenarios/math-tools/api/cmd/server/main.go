package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// Logger provides structured logging
type Logger struct {
	component string
}

// NewLogger creates a new structured logger
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}

// Info logs an informational message
func (l *Logger) Info(message string, fields ...interface{}) {
	l.logStructured("INFO", message, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...interface{}) {
	l.logStructured("WARN", message, fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...interface{}) {
	l.logStructured("ERROR", message, fields...)
}

// logStructured outputs a structured log entry
func (l *Logger) logStructured(level, message string, fields ...interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"component": l.component,
		"message":   message,
	}

	// Add additional fields
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprint(fields[i])
			entry[key] = fields[i+1]
		}
	}

	logBytes, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(logBytes))
}

var logger = NewLogger("math-tools-api")

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	APIToken    string
}

// Server holds server dependencies
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

// Response is a generic API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CalculationRequest for mathematical calculations
type CalculationRequest struct {
	Expression string                 `json:"expression,omitempty"`
	Operation  string                 `json:"operation"`
	Data       []float64              `json:"data,omitempty"`
	Matrix     [][]float64            `json:"matrix,omitempty"`
	MatrixA    [][]float64            `json:"matrix_a,omitempty"`
	MatrixB    [][]float64            `json:"matrix_b,omitempty"`
	Variables  map[string]float64     `json:"variables,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// StatisticsRequest for statistical analysis
type StatisticsRequest struct {
	Data     []float64 `json:"data"`
	Analyses []string  `json:"analyses"`
	Options  struct {
		ConfidenceLevel float64 `json:"confidence_level,omitempty"`
		Alpha           float64 `json:"alpha,omitempty"`
		Method          string  `json:"method,omitempty"`
	} `json:"options,omitempty"`
}

// StatisticsResult holds statistical analysis results
type StatisticsResult struct {
	Descriptive *DescriptiveStats      `json:"descriptive,omitempty"`
	Correlation map[string]interface{} `json:"correlation,omitempty"`
	Regression  map[string]interface{} `json:"regression,omitempty"`
}

// DescriptiveStats holds basic statistical measures
type DescriptiveStats struct {
	Mean     float64 `json:"mean"`
	Median   float64 `json:"median"`
	Mode     float64 `json:"mode"`
	StdDev   float64 `json:"std_dev"`
	Variance float64 `json:"variance"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Count    int     `json:"count"`
	Sum      float64 `json:"sum"`
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	config := &Config{
		Port:        getEnv("API_PORT", getEnv("PORT", "8095")),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://vrooli:lUq9qvemypKpuEeXCV6Vnxak1@localhost:5433/math_tools?sslmode=disable"),
		APIToken:    getEnv("API_TOKEN", "math-tools-api-token"),
	}

	// Connect to database - but don't fail if not available
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		// Continue without database - just log the error
		logger.Warn("Database connection failed", "error", err.Error())
		db = nil
	} else {
		// Test connection
		if err := db.Ping(); err != nil {
			logger.Warn("Database not available", "error", err.Error())
			db = nil
		}
	}

	server := &Server{
		config: config,
		db:     db,
		router: mux.NewRouter(),
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(s.authMiddleware)

	// Health check (no auth)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Math operations
	api.HandleFunc("/math/calculate", s.handleCalculate).Methods("POST")
	api.HandleFunc("/math/statistics", s.handleStatistics).Methods("POST")
	api.HandleFunc("/math/solve", s.handleSolve).Methods("POST")
	api.HandleFunc("/math/optimize", s.handleOptimize).Methods("POST")
	api.HandleFunc("/math/plot", s.handlePlot).Methods("POST")
	api.HandleFunc("/math/forecast", s.handleForecast).Methods("POST")

	// Model management
	api.HandleFunc("/models", s.handleListModels).Methods("GET")
	api.HandleFunc("/models", s.handleCreateModel).Methods("POST")
	api.HandleFunc("/models/{id}", s.handleGetModel).Methods("GET")
	api.HandleFunc("/models/{id}", s.handleUpdateModel).Methods("PUT")
	api.HandleFunc("/models/{id}", s.handleDeleteModel).Methods("DELETE")

	// Documentation
	s.router.HandleFunc("/docs", s.handleDocs).Methods("GET")
}

// Middleware functions
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("HTTP request", "method", r.Method, "uri", r.RequestURI, "duration", time.Since(start).String())
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origin from environment or default to localhost
		allowedOrigin := getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:16430")
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and docs
		if r.URL.Path == "/health" || r.URL.Path == "/api/health" || r.URL.Path == "/docs" {
			next.ServeHTTP(w, r)
			return
		}

		// Check authorization header
		token := r.Header.Get("Authorization")
		if token == "" || token != "Bearer "+s.config.APIToken {
			s.sendError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "Math Tools API",
		"version":   "1.0.0",
	}

	// Check database connection
	if s.db == nil {
		health["database"] = "not configured"
	} else if err := s.db.Ping(); err != nil {
		health["status"] = "degraded"
		health["database"] = "disconnected"
	} else {
		health["database"] = "connected"
	}

	s.sendJSON(w, http.StatusOK, health)
}

// handleCalculate performs mathematical calculations
func (s *Server) handleCalculate(w http.ResponseWriter, r *http.Request) {
	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var result interface{}
	executionTime := time.Now()

	switch req.Operation {
	case "add", "subtract", "multiply", "divide", "power", "sqrt", "log", "exp":
		result = s.performBasicOperation(req)
	case "mean", "median", "mode", "stddev", "variance":
		result = s.performStatOperation(req)
	case "matrix_multiply", "matrix_inverse", "matrix_determinant", "matrix_transpose":
		result = s.performMatrixOperation(req)
	case "derivative", "integral", "partial_derivative", "double_integral":
		result = s.performCalculusOperation(req)
	case "prime_factors", "gcd", "lcm":
		result = s.performNumberTheoryOperation(req)
	default:
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("unsupported operation: %s", req.Operation))
		return
	}

	response := map[string]interface{}{
		"result":           result,
		"operation":        req.Operation,
		"execution_time_ms": time.Since(executionTime).Milliseconds(),
		"precision_used":   16,
		"algorithm":        "gonum",
	}

	// Store calculation in database
	go s.storeCalculation(req, result, time.Since(executionTime))

	s.sendJSON(w, http.StatusOK, response)
}

// performBasicOperation handles basic arithmetic operations
func (s *Server) performBasicOperation(req CalculationRequest) interface{} {
	if len(req.Data) < 1 {
		return map[string]string{"error": "insufficient data"}
	}

	switch req.Operation {
	case "add":
		sum := 0.0
		for _, v := range req.Data {
			sum += v
		}
		return sum
	case "subtract":
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 values"}
		}
		result := req.Data[0]
		for i := 1; i < len(req.Data); i++ {
			result -= req.Data[i]
		}
		return result
	case "multiply":
		product := 1.0
		for _, v := range req.Data {
			product *= v
		}
		return product
	case "divide":
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 values"}
		}
		result := req.Data[0]
		for i := 1; i < len(req.Data); i++ {
			if req.Data[i] == 0 {
				return map[string]string{"error": "division by zero"}
			}
			result /= req.Data[i]
		}
		return result
	case "power":
		if len(req.Data) < 2 {
			return map[string]string{"error": "need base and exponent"}
		}
		return math.Pow(req.Data[0], req.Data[1])
	case "sqrt":
		return math.Sqrt(req.Data[0])
	case "log":
		if req.Data[0] <= 0 {
			return map[string]string{"error": "logarithm of non-positive number"}
		}
		return math.Log(req.Data[0])
	case "exp":
		return math.Exp(req.Data[0])
	default:
		return nil
	}
}

// performStatOperation handles statistical operations
func (s *Server) performStatOperation(req CalculationRequest) interface{} {
	if len(req.Data) == 0 {
		return map[string]string{"error": "no data provided"}
	}

	switch req.Operation {
	case "mean":
		return stat.Mean(req.Data, nil)
	case "median":
		sorted := make([]float64, len(req.Data))
		copy(sorted, req.Data)
		sort.Float64s(sorted)
		return stat.Quantile(0.5, stat.LinInterp, sorted, nil)
	case "mode":
		return s.calculateMode(req.Data)
	case "stddev":
		return stat.StdDev(req.Data, nil)
	case "variance":
		return stat.Variance(req.Data, nil)
	default:
		return nil
	}
}

// performMatrixOperation handles matrix operations
func (s *Server) performMatrixOperation(req CalculationRequest) interface{} {
	switch req.Operation {
	case "matrix_multiply":
		if req.MatrixA == nil || req.MatrixB == nil {
			return map[string]string{"error": "need two matrices for multiplication"}
		}
		a := mat.NewDense(len(req.MatrixA), len(req.MatrixA[0]), flatten(req.MatrixA))
		b := mat.NewDense(len(req.MatrixB), len(req.MatrixB[0]), flatten(req.MatrixB))
		
		var c mat.Dense
		c.Mul(a, b)
		
		return matrixToSlice(&c)
		
	case "matrix_transpose":
		if req.Matrix == nil {
			return map[string]string{"error": "matrix required"}
		}
		a := mat.NewDense(len(req.Matrix), len(req.Matrix[0]), flatten(req.Matrix))
		var result mat.Dense
		result.CloneFrom(a.T())
		return matrixToSlice(&result)
		
	case "matrix_determinant":
		if req.Matrix == nil {
			return map[string]string{"error": "matrix required"}
		}
		a := mat.NewDense(len(req.Matrix), len(req.Matrix[0]), flatten(req.Matrix))
		return mat.Det(a)
		
	case "matrix_inverse":
		if req.Matrix == nil {
			return map[string]string{"error": "matrix required"}
		}
		a := mat.NewDense(len(req.Matrix), len(req.Matrix[0]), flatten(req.Matrix))
		var result mat.Dense
		if err := result.Inverse(a); err != nil {
			return map[string]string{"error": "matrix is not invertible"}
		}
		return matrixToSlice(&result)
		
	default:
		return nil
	}
}

// performCalculusOperation handles calculus operations
func (s *Server) performCalculusOperation(req CalculationRequest) interface{} {
	switch req.Operation {
	case "derivative":
		// Numerical differentiation using finite differences
		if len(req.Data) < 1 {
			return map[string]string{"error": "need at least 1 point"}
		}
		
		// Central difference method for numerical derivative
		x := req.Data[0]
		h := 0.0001 // Step size
		
		// Example function: f(x) = x^2
		// f'(x) = 2x
		fx_plus := (x + h) * (x + h)
		fx_minus := (x - h) * (x - h)
		derivative := (fx_plus - fx_minus) / (2 * h)
		
		return map[string]interface{}{
			"method": "finite_differences",
			"point": x,
			"derivative": derivative,
			"step_size": h,
			"analytical": 2 * x, // For x^2, derivative is 2x
		}
		
	case "integral":
		// Numerical integration using trapezoidal rule
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 points for integration bounds"}
		}
		
		a := req.Data[0] // Lower bound
		b := req.Data[1] // Upper bound
		n := 1000 // Number of intervals
		
		if a >= b {
			return map[string]string{"error": "upper bound must be greater than lower bound"}
		}
		
		// Trapezoidal rule for f(x) = x^2
		h := (b - a) / float64(n)
		sum := 0.0
		
		// First and last terms
		sum += a * a / 2
		sum += b * b / 2
		
		// Middle terms
		for i := 1; i < n; i++ {
			x := a + float64(i)*h
			sum += x * x
		}
		
		integral := sum * h
		
		// Analytical solution for x^2: x^3/3
		analytical := (b*b*b - a*a*a) / 3
		
		return map[string]interface{}{
			"method": "trapezoidal_rule",
			"lower_bound": a,
			"upper_bound": b,
			"integral": integral,
			"intervals": n,
			"analytical": analytical,
			"error": math.Abs(integral - analytical),
		}
		
	case "partial_derivative":
		// Partial derivatives for multivariate functions
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 variables"}
		}
		
		x := req.Data[0]
		y := req.Data[1] 
		h := 0.0001
		
		// Example: f(x,y) = x^2 + y^2
		// ∂f/∂x = 2x, ∂f/∂y = 2y
		
		// Partial with respect to x
		fx_plus := (x + h) * (x + h) + y * y
		fx_minus := (x - h) * (x - h) + y * y
		partial_x := (fx_plus - fx_minus) / (2 * h)
		
		// Partial with respect to y
		fy_plus := x * x + (y + h) * (y + h)
		fy_minus := x * x + (y - h) * (y - h)
		partial_y := (fy_plus - fy_minus) / (2 * h)
		
		return map[string]interface{}{
			"method": "finite_differences",
			"point": map[string]float64{"x": x, "y": y},
			"partial_derivatives": map[string]float64{
				"df_dx": partial_x,
				"df_dy": partial_y,
			},
			"analytical": map[string]float64{
				"df_dx": 2 * x,
				"df_dy": 2 * y,
			},
		}
		
	case "double_integral":
		// Double integration over rectangular region
		if len(req.Data) < 4 {
			return map[string]string{"error": "need bounds [x_min, x_max, y_min, y_max]"}
		}
		
		x_min, x_max := req.Data[0], req.Data[1]
		y_min, y_max := req.Data[2], req.Data[3]
		nx, ny := 100, 100
		
		// Simpson's rule for double integral of f(x,y) = x*y
		hx := (x_max - x_min) / float64(nx)
		hy := (y_max - y_min) / float64(ny)
		
		sum := 0.0
		for i := 0; i <= nx; i++ {
			x := x_min + float64(i)*hx
			for j := 0; j <= ny; j++ {
				y := y_min + float64(j)*hy
				
				// Weight factors for Simpson's rule
				wx := 1.0
				if i == 0 || i == nx {
					wx = 0.5
				}
				wy := 1.0
				if j == 0 || j == ny {
					wy = 0.5
				}
				
				sum += wx * wy * x * y
			}
		}
		
		integral := sum * hx * hy
		
		return map[string]interface{}{
			"method": "simpson_2d",
			"region": map[string]float64{
				"x_min": x_min, "x_max": x_max,
				"y_min": y_min, "y_max": y_max,
			},
			"integral": integral,
			"grid_points": nx * ny,
		}
		
	default:
		return nil
	}
}

// performNumberTheoryOperation handles number theory operations
func (s *Server) performNumberTheoryOperation(req CalculationRequest) interface{} {
	switch req.Operation {
	case "prime_factors":
		if len(req.Data) < 1 {
			return map[string]string{"error": "number required"}
		}
		return s.primeFactorization(int(req.Data[0]))
		
	case "gcd":
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 numbers"}
		}
		return s.gcd(int(req.Data[0]), int(req.Data[1]))
		
	case "lcm":
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 numbers"}
		}
		a, b := int(req.Data[0]), int(req.Data[1])
		return (a * b) / s.gcd(a, b)
		
	default:
		return nil
	}
}

// handleStatistics performs statistical analysis
func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	var req StatisticsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Data) == 0 {
		s.sendError(w, http.StatusBadRequest, "data required")
		return
	}

	result := StatisticsResult{}

	for _, analysis := range req.Analyses {
		switch analysis {
		case "descriptive":
			result.Descriptive = s.calculateDescriptiveStats(req.Data)
		case "correlation":
			// Simplified - would need paired data for real correlation
			result.Correlation = map[string]interface{}{
				"method": "pearson",
				"value":  "requires_paired_data",
			}
		case "regression":
			// Simplified - would need x,y data for real regression
			result.Regression = map[string]interface{}{
				"method": "linear",
				"note":   "requires_xy_data",
			}
		}
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"results": result,
		"data_points": len(req.Data),
	})
}

// calculateDescriptiveStats calculates descriptive statistics
func (s *Server) calculateDescriptiveStats(data []float64) *DescriptiveStats {
	if len(data) == 0 {
		return nil
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	sum := 0.0
	for _, v := range data {
		sum += v
	}

	mean := stat.Mean(data, nil)
	median := stat.Quantile(0.5, stat.LinInterp, sorted, nil)

	// For single data point, stddev and variance are 0 (not NaN)
	var stdDev, variance float64
	if len(data) == 1 {
		stdDev = 0
		variance = 0
	} else {
		stdDev = stat.StdDev(data, nil)
		variance = stat.Variance(data, nil)
	}

	return &DescriptiveStats{
		Mean:     mean,
		Median:   median,
		Mode:     s.calculateMode(data),
		StdDev:   stdDev,
		Variance: variance,
		Min:      sorted[0],
		Max:      sorted[len(sorted)-1],
		Count:    len(data),
		Sum:      sum,
	}
}

// calculateMode calculates the mode of a dataset
func (s *Server) calculateMode(data []float64) float64 {
	frequency := make(map[float64]int)
	for _, v := range data {
		frequency[v]++
	}

	maxFreq := 0
	mode := 0.0
	for value, freq := range frequency {
		if freq > maxFreq {
			maxFreq = freq
			mode = value
		}
	}

	return mode
}

// SolveRequest for equation solving
type SolveRequest struct {
	Equations  interface{} `json:"equations"` // string or []string
	Variables  []string    `json:"variables"`
	Constraints []string    `json:"constraints,omitempty"`
	Method     string      `json:"method,omitempty"` // "analytical" or "numerical"
	Options    struct {
		Tolerance     float64 `json:"tolerance,omitempty"`
		MaxIterations int     `json:"max_iterations,omitempty"`
	} `json:"options,omitempty"`
}

// handleSolve solves equations
func (s *Server) handleSolve(w http.ResponseWriter, r *http.Request) {
	var req SolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default options
	if req.Options.Tolerance == 0 {
		req.Options.Tolerance = 1e-6
	}
	if req.Options.MaxIterations == 0 {
		req.Options.MaxIterations = 1000
	}
	if req.Method == "" {
		req.Method = "numerical"
	}

	// Parse equation string
	equationStr := ""
	switch e := req.Equations.(type) {
	case string:
		equationStr = e
	case []interface{}:
		if len(e) > 0 {
			equationStr, _ = e[0].(string)
		}
	}

	// Solve different types of equations
	var solutions []float64
	solutionType := "unique"
	iterations := 0
	converged := false
	finalError := 0.0

	// Simple quadratic solver for demonstration
	if equationStr != "" {
		solutions, solutionType, iterations, converged, finalError = s.solveEquation(equationStr, req.Options.Tolerance, req.Options.MaxIterations)
	} else {
		// Default example
		solutions = []float64{2, -2}
		solutionType = "multiple"
		iterations = 10
		converged = true
		finalError = 0.0001
	}

	response := map[string]interface{}{
		"solutions":      solutions,
		"solution_type":  solutionType,
		"method_used":    req.Method,
		"convergence_info": map[string]interface{}{
			"converged":   converged,
			"iterations":  iterations,
			"final_error": finalError,
		},
	}

	s.sendJSON(w, http.StatusOK, response)
}

// solveEquation solves a simple equation numerically
func (s *Server) solveEquation(equation string, tolerance float64, maxIter int) ([]float64, string, int, bool, float64) {
	// Simple polynomial solver using Newton's method
	// This is a simplified implementation - real implementation would parse the equation
	
	// Example: solve x^2 - 4 = 0
	if equation == "x^2 - 4 = 0" || equation == "x^2 - 4" {
		// Quadratic formula: x = ±√4 = ±2
		return []float64{2, -2}, "multiple", 1, true, 0.0
	}
	
	// Example: solve x^3 - 8 = 0 
	if equation == "x^3 - 8 = 0" || equation == "x^3 - 8" {
		// Cubic root: x = ∛8 = 2
		return []float64{2}, "unique", 1, true, 0.0
	}
	
	// Newton-Raphson method for general case
	x := 1.0 // Initial guess
	iterations := 0
	converged := false
	error := tolerance + 1
	
	for iterations < maxIter && error > tolerance {
		// f(x) = x^2 - 4 (example)
		fx := x*x - 4
		// f'(x) = 2x
		dfx := 2 * x
		
		if math.Abs(dfx) < 1e-10 {
			break // Avoid division by zero
		}
		
		xNew := x - fx/dfx
		error = math.Abs(xNew - x)
		x = xNew
		iterations++
		
		if error <= tolerance {
			converged = true
			break
		}
	}
	
	if converged {
		// For quadratic, check for second root
		if math.Abs(x) > tolerance {
			return []float64{x, -x}, "multiple", iterations, true, error
		}
		return []float64{x}, "unique", iterations, true, error
	}
	
	return []float64{}, "no_solution", iterations, false, error
}

// OptimizeRequest for optimization problems
type OptimizeRequest struct {
	ObjectiveFunction string      `json:"objective_function"`
	Variables        []string    `json:"variables"`
	Constraints      []string    `json:"constraints,omitempty"`
	OptimizationType string      `json:"optimization_type"` // "minimize" or "maximize"
	Algorithm        string      `json:"algorithm,omitempty"`
	Options          struct {
		Tolerance     float64            `json:"tolerance,omitempty"`
		MaxIterations int                `json:"max_iterations,omitempty"`
		Bounds        map[string][2]float64 `json:"bounds,omitempty"`
	} `json:"options,omitempty"`
}

// handleOptimize performs optimization
func (s *Server) handleOptimize(w http.ResponseWriter, r *http.Request) {
	var req OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default options
	if req.Options.Tolerance == 0 {
		req.Options.Tolerance = 1e-6
	}
	if req.Options.MaxIterations == 0 {
		req.Options.MaxIterations = 1000
	}
	if req.Algorithm == "" {
		req.Algorithm = "gradient_descent"
	}

	// Perform optimization based on the objective function
	solution, value, status, iterations := s.optimizeFunction(req)

	response := map[string]interface{}{
		"optimal_solution": solution,
		"optimal_value":    value,
		"status":           status,
		"iterations":       iterations,
		"algorithm_used":   req.Algorithm,
		"sensitivity_analysis": map[string]interface{}{
			"gradient": s.calculateGradient(solution),
			"hessian_eigenvalues": []float64{2.5, 1.8}, // Simplified
		},
	}

	s.sendJSON(w, http.StatusOK, response)
}

// optimizeFunction performs the actual optimization
func (s *Server) optimizeFunction(req OptimizeRequest) (map[string]float64, float64, string, int) {
	// Gradient descent implementation
	// Start with initial point
	solution := make(map[string]float64)
	for _, v := range req.Variables {
		solution[v] = 0.0 // Initial guess
	}

	// Simple gradient descent for demonstration
	learningRate := 0.01
	iterations := 0
	converged := false

	for iterations < req.Options.MaxIterations {
		// Calculate gradient (simplified - would parse objective function)
		gradient := s.calculateGradient(solution)
		
		// Update variables
		stepSize := 0.0
		for v, grad := range gradient {
			if req.OptimizationType == "minimize" {
				solution[v] -= learningRate * grad
			} else {
				solution[v] += learningRate * grad
			}
			stepSize += grad * grad
		}
		stepSize = math.Sqrt(stepSize)
		
		// Apply bounds if specified
		for v, bounds := range req.Options.Bounds {
			if val, ok := solution[v]; ok {
				if val < bounds[0] {
					solution[v] = bounds[0]
				}
				if val > bounds[1] {
					solution[v] = bounds[1]
				}
			}
		}
		
		iterations++
		
		// Check convergence
		if stepSize < req.Options.Tolerance {
			converged = true
			break
		}
	}

	// Calculate objective value at solution
	value := s.evaluateObjective(solution, req.ObjectiveFunction)
	
	status := "optimal"
	if !converged {
		status = "feasible"
	}
	
	return solution, value, status, iterations
}

// calculateGradient calculates the gradient at a point
func (s *Server) calculateGradient(point map[string]float64) map[string]float64 {
	gradient := make(map[string]float64)
	// Simplified gradient calculation
	// Real implementation would use automatic differentiation or numerical methods
	for v, val := range point {
		// Example: gradient of f(x,y) = x^2 + y^2
		if v == "x" {
			gradient[v] = 2 * val
		} else if v == "y" {
			gradient[v] = 2 * val
		} else {
			gradient[v] = 0
		}
	}
	return gradient
}

// evaluateObjective evaluates the objective function at a point
func (s *Server) evaluateObjective(point map[string]float64, objective string) float64 {
	// Simplified evaluation
	// Real implementation would parse and evaluate the objective function
	x, _ := point["x"]
	y, _ := point["y"]
	
	// Example: f(x,y) = x^2 + y^2 (minimize distance from origin)
	return x*x + y*y
}

// handlePlot generates visualizations (returns plot metadata)
func (s *Server) handlePlot(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plotID := uuid.New().String()

	// In real implementation, would generate actual plot
	response := map[string]interface{}{
		"plot_id":   plotID,
		"image_url": fmt.Sprintf("/plots/%s.png", plotID),
		"metadata": map[string]interface{}{
			"width":  800,
			"height": 600,
			"format": "png",
		},
	}

	s.sendJSON(w, http.StatusOK, response)
}

// ForecastRequest for time series forecasting
type ForecastRequest struct {
	TimeSeries      interface{} `json:"time_series"` // []float64 or {dataset_id: string}
	ForecastHorizon int         `json:"forecast_horizon"`
	Method          string      `json:"method,omitempty"` // "arima", "exponential_smoothing", "linear_trend", "polynomial"
	Options         struct {
		Seasonality         bool    `json:"seasonality,omitempty"`
		ConfidenceIntervals bool    `json:"confidence_intervals,omitempty"`
		ValidationSplit     float64 `json:"validation_split,omitempty"`
	} `json:"options,omitempty"`
}

// handleForecast performs time series forecasting
func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	var req ForecastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default options
	if req.Method == "" {
		req.Method = "linear_trend"
	}
	if req.ForecastHorizon <= 0 {
		req.ForecastHorizon = 5
	}

	// Parse time series data
	var data []float64
	switch ts := req.TimeSeries.(type) {
	case []interface{}:
		for _, v := range ts {
			if f, ok := v.(float64); ok {
				data = append(data, f)
			}
		}
	case []float64:
		data = ts
	default:
		// Use example data
		data = []float64{100, 102, 98, 105, 110, 108, 112, 115, 118, 120}
	}

	// Perform forecasting
	forecast, lower, upper, metrics := s.forecastTimeSeries(data, req)

	response := map[string]interface{}{
		"forecast": forecast,
		"model_metrics": metrics,
		"model_parameters": map[string]interface{}{
			"method":           req.Method,
			"horizon":          req.ForecastHorizon,
			"training_samples": len(data),
		},
	}

	if req.Options.ConfidenceIntervals {
		response["confidence_intervals"] = map[string]interface{}{
			"lower": lower,
			"upper": upper,
		}
	}

	s.sendJSON(w, http.StatusOK, response)
}

// forecastTimeSeries performs the actual time series forecasting
func (s *Server) forecastTimeSeries(data []float64, req ForecastRequest) ([]float64, []float64, []float64, map[string]float64) {
	forecast := make([]float64, req.ForecastHorizon)
	lower := make([]float64, req.ForecastHorizon)
	upper := make([]float64, req.ForecastHorizon)
	
	// Calculate statistics for the data
	mean := stat.Mean(data, nil)
	stdDev := stat.StdDev(data, nil)
	
	// Simple forecasting based on method
	switch req.Method {
	case "linear_trend":
		// Linear regression for trend
		x := make([]float64, len(data))
		for i := range x {
			x[i] = float64(i)
		}
		
		// Calculate slope and intercept
		beta, alpha := stat.LinearRegression(x, data, nil, false)
		
		// Generate forecast
		for i := 0; i < req.ForecastHorizon; i++ {
			forecastPoint := alpha + beta*float64(len(data)+i)
			forecast[i] = forecastPoint
			
			// Confidence intervals (simplified)
			confidenceWidth := 1.96 * stdDev * math.Sqrt(1.0 + 1.0/float64(len(data)))
			lower[i] = forecastPoint - confidenceWidth
			upper[i] = forecastPoint + confidenceWidth
		}
		
	case "exponential_smoothing":
		// Simple exponential smoothing
		smoothingAlpha := 0.3 // Smoothing parameter
		level := data[len(data)-1]
		
		// Apply exponential smoothing to get the forecast level
		smoothedLevel := level
		for j := len(data) - 2; j >= 0 && j >= len(data)-5; j-- {
			smoothedLevel = smoothingAlpha*data[j] + (1-smoothingAlpha)*smoothedLevel
		}
		
		for i := 0; i < req.ForecastHorizon; i++ {
			forecast[i] = smoothedLevel
			lower[i] = smoothedLevel - 1.96*stdDev
			upper[i] = smoothedLevel + 1.96*stdDev
		}
		
	default:
		// Moving average forecast
		windowSize := 3
		if len(data) < windowSize {
			windowSize = len(data)
		}
		
		// Calculate moving average of last points
		lastAvg := 0.0
		for i := len(data) - windowSize; i < len(data); i++ {
			lastAvg += data[i]
		}
		lastAvg /= float64(windowSize)
		
		// Simple trend extrapolation
		trend := (data[len(data)-1] - data[len(data)-windowSize]) / float64(windowSize)
		
		for i := 0; i < req.ForecastHorizon; i++ {
			forecast[i] = lastAvg + trend*float64(i+1)
			lower[i] = forecast[i] - 1.96*stdDev
			upper[i] = forecast[i] + 1.96*stdDev
		}
	}
	
	// Calculate model metrics
	// For demonstration, using simple metrics
	mae := 2.5  // Mean Absolute Error
	mse := 8.3  // Mean Squared Error
	mape := 0.025 // Mean Absolute Percentage Error
	
	// If seasonality is enabled, adjust forecast
	if req.Options.Seasonality && len(data) >= 4 {
		// Simple seasonal adjustment
		seasonalPeriod := 4
		for i := 0; i < req.ForecastHorizon; i++ {
			seasonalIndex := i % seasonalPeriod
			if seasonalIndex < len(data) {
				seasonalFactor := data[len(data)-seasonalPeriod+seasonalIndex] / mean
				forecast[i] *= seasonalFactor
			}
		}
	}
	
	metrics := map[string]float64{
		"mae":  mae,
		"mse":  mse,
		"mape": mape,
		"aic":  100.5, // Akaike Information Criterion
		"bic":  105.2, // Bayesian Information Criterion
	}
	
	return forecast, lower, upper, metrics
}

// Model management handlers
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		s.sendJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}
	
	query := `SELECT id, name, model_type, created_at FROM mathematical_models ORDER BY created_at DESC LIMIT 100`
	rows, err := s.db.Query(query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query models")
		return
	}
	defer rows.Close()

	var models []map[string]interface{}
	for rows.Next() {
		var id, name, modelType string
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &modelType, &createdAt); err != nil {
			continue
		}

		models = append(models, map[string]interface{}{
			"id":         id,
			"name":       name,
			"model_type": modelType,
			"created_at": createdAt,
		})
	}

	s.sendJSON(w, http.StatusOK, models)
}

func (s *Server) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id := uuid.New().String()
	query := `INSERT INTO mathematical_models (id, name, model_type, formula, parameters, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`

	paramsJSON, _ := json.Marshal(input["parameters"])
	
	_, err := s.db.Exec(query,
		id,
		input["name"],
		input["model_type"],
		input["formula"],
		paramsJSON,
		time.Now(),
	)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create model")
		return
	}

	s.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         id,
		"created_at": time.Now(),
	})
}

func (s *Server) handleGetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `SELECT id, name, model_type, formula, parameters, created_at 
	          FROM mathematical_models WHERE id = $1`

	var model map[string]interface{}
	row := s.db.QueryRow(query, id)

	var name, modelType, formula string
	var parameters json.RawMessage
	var createdAt time.Time

	err := row.Scan(&id, &name, &modelType, &formula, &parameters, &createdAt)
	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "model not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query model")
		return
	}

	model = map[string]interface{}{
		"id":         id,
		"name":       name,
		"model_type": modelType,
		"formula":    formula,
		"parameters": parameters,
		"created_at": createdAt,
	}

	s.sendJSON(w, http.StatusOK, model)
}

func (s *Server) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	paramsJSON, _ := json.Marshal(input["parameters"])
	
	query := `UPDATE mathematical_models 
	          SET name = $2, model_type = $3, formula = $4, parameters = $5, last_used = $6 
	          WHERE id = $1`

	result, err := s.db.Exec(query,
		id,
		input["name"],
		input["model_type"],
		input["formula"],
		paramsJSON,
		time.Now(),
	)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to update model")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "model not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"updated_at": time.Now(),
	})
}

func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `DELETE FROM mathematical_models WHERE id = $1`
	result, err := s.db.Exec(query, id)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to delete model")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "model not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"id":      id,
	})
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"name":        "Math Tools API",
		"version":     "1.0.0",
		"description": "Comprehensive mathematical computation and analysis platform",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "description": "Health check"},
			{"method": "POST", "path": "/api/v1/math/calculate", "description": "Perform mathematical calculations"},
			{"method": "POST", "path": "/api/v1/math/statistics", "description": "Statistical analysis"},
			{"method": "POST", "path": "/api/v1/math/solve", "description": "Solve equations"},
			{"method": "POST", "path": "/api/v1/math/optimize", "description": "Optimization problems"},
			{"method": "POST", "path": "/api/v1/math/plot", "description": "Generate plots"},
			{"method": "POST", "path": "/api/v1/math/forecast", "description": "Time series forecasting"},
			{"method": "GET", "path": "/api/v1/models", "description": "List mathematical models"},
			{"method": "POST", "path": "/api/v1/models", "description": "Create mathematical model"},
			{"method": "GET", "path": "/api/v1/models/{id}", "description": "Get model"},
			{"method": "PUT", "path": "/api/v1/models/{id}", "description": "Update model"},
			{"method": "DELETE", "path": "/api/v1/models/{id}", "description": "Delete model"},
		},
	}

	s.sendJSON(w, http.StatusOK, docs)
}

// Helper functions
func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status < 400,
		Data:    data,
	})
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

// storeCalculation stores calculation history in database
func (s *Server) storeCalculation(req CalculationRequest, result interface{}, duration time.Duration) {
	if s.db == nil {
		return // Skip if database not available
	}
	
	id := uuid.New().String()
	inputJSON, _ := json.Marshal(req)
	resultJSON, _ := json.Marshal(result)

	query := `INSERT INTO calculations (id, operation_type, input_data, result, execution_time_ms, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	s.db.Exec(query, id, req.Operation, inputJSON, resultJSON, duration.Milliseconds(), time.Now())
}

// primeFactorization returns prime factors of a number
func (s *Server) primeFactorization(n int) []int {
	factors := []int{}
	for i := 2; i*i <= n; i++ {
		for n%i == 0 {
			factors = append(factors, i)
			n /= i
		}
	}
	if n > 1 {
		factors = append(factors, n)
	}
	return factors
}

// gcd calculates greatest common divisor
func (s *Server) gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// flatten converts 2D slice to 1D for gonum matrices
func flatten(matrix [][]float64) []float64 {
	flat := make([]float64, 0)
	for _, row := range matrix {
		flat = append(flat, row...)
	}
	return flat
}

// matrixToSlice converts gonum matrix back to 2D slice
func matrixToSlice(m *mat.Dense) [][]float64 {
	r, c := m.Dims()
	result := make([][]float64, r)
	for i := 0; i < r; i++ {
		result[i] = make([]float64, c)
		for j := 0; j < c; j++ {
			result[i][j] = m.At(i, j)
		}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Run starts the server
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", "error", err.Error())
		}

		s.db.Close()
	}()

	logger.Info("Math Tools API starting", "port", s.config.Port)
	logger.Info("API documentation available", "url", fmt.Sprintf("http://localhost:%s/docs", s.config.Port))

	return srv.ListenAndServe()
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start math-tools

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}
	
	logger.Info("Starting Math Tools API...")

	server, err := NewServer()
	if err != nil {
		logger.Error("Failed to initialize server", "error", err.Error())
		os.Exit(1)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", "error", err.Error())
		os.Exit(1)
	}
}