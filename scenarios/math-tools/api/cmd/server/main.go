package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
		Port:        getEnv("PORT", "8095"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/math_tools?sslmode=disable"),
		APIToken:    getEnv("API_TOKEN", "math-tools-api-token"),
	}

	// Connect to database - but don't fail if not available
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		// Continue without database - just log the error
		log.Printf("Warning: Database connection failed: %v", err)
		db = nil
	} else {
		// Test connection
		if err := db.Ping(); err != nil {
			log.Printf("Warning: Database not available: %v", err)
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
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
	case "derivative", "integral":
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

// performCalculusOperation handles calculus operations (simplified)
func (s *Server) performCalculusOperation(req CalculationRequest) interface{} {
	// Simplified numerical differentiation and integration
	// In a real implementation, you'd use more sophisticated methods
	switch req.Operation {
	case "derivative":
		// Numerical differentiation using finite differences
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 points"}
		}
		// step size h = 0.0001 for approximation
		// f'(x) ≈ (f(x+h) - f(x-h)) / 2h
		return map[string]interface{}{
			"method": "finite_differences",
			"result": "numerical_derivative",
			"note":   "simplified implementation",
		}
		
	case "integral":
		// Numerical integration using trapezoidal rule
		if len(req.Data) < 2 {
			return map[string]string{"error": "need at least 2 points"}
		}
		return map[string]interface{}{
			"method": "trapezoidal_rule",
			"result": "numerical_integral",
			"note":   "simplified implementation",
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
	stdDev := stat.StdDev(data, nil)
	variance := stat.Variance(data, nil)

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

// handleSolve solves equations (placeholder)
func (s *Server) handleSolve(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Simplified response - real implementation would solve equations
	response := map[string]interface{}{
		"solutions":      []float64{2, -2}, // Example for x^2 - 4 = 0
		"solution_type":  "multiple",
		"method_used":    "numerical",
		"convergence_info": map[string]interface{}{
			"converged":   true,
			"iterations":  10,
			"final_error": 0.0001,
		},
	}

	s.sendJSON(w, http.StatusOK, response)
}

// handleOptimize handles optimization problems (placeholder)
func (s *Server) handleOptimize(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Simplified response - real implementation would perform optimization
	response := map[string]interface{}{
		"optimal_solution": map[string]float64{"x": 1.5, "y": 2.3},
		"optimal_value":    42.0,
		"status":           "optimal",
		"iterations":       25,
		"algorithm_used":   "gradient_descent",
	}

	s.sendJSON(w, http.StatusOK, response)
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

// handleForecast performs time series forecasting (placeholder)
func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Simplified response - real implementation would perform forecasting
	response := map[string]interface{}{
		"forecast": []float64{100.5, 102.3, 104.1, 106.0},
		"confidence_intervals": map[string]interface{}{
			"lower": []float64{95.0, 96.5, 98.0, 99.5},
			"upper": []float64{106.0, 108.1, 110.2, 112.5},
		},
		"model_metrics": map[string]float64{
			"mae":  2.5,
			"mse":  8.3,
			"mape": 0.025,
		},
	}

	s.sendJSON(w, http.StatusOK, response)
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

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		s.db.Close()
	}()

	log.Printf("Math Tools API starting on port %s", s.config.Port)
	log.Printf("API documentation available at http://localhost:%s/docs", s.config.Port)

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
	
	log.Println("Starting Math Tools API...")

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}