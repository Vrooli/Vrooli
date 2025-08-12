package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"metareasoning-api/internal/container"
	"metareasoning-api/internal/interfaces/http/handlers"
	"metareasoning-api/internal/interfaces/http/middleware"
	"metareasoning-api/internal/pkg/errors"
)

// Router handles HTTP routing and middleware setup
type Router struct {
	container      *container.Container
	errorHandler   *middleware.ErrorHandler
	
	// Handlers
	workflowHandler    *handlers.WorkflowHandler
	executionHandler   *handlers.ExecutionHandler
	systemHandler      *handlers.SystemHandler
	aiHandler          *handlers.AIHandler
	integrationHandler *handlers.IntegrationHandler
}

// NewRouter creates a new HTTP router with all dependencies
func NewRouter(c *container.Container) *Router {
	// Create error handler
	errorHandler := middleware.NewErrorHandler(true) // Include stack traces in development
	
	// Create handlers
	workflowHandler := handlers.NewWorkflowHandler(
		c.WorkflowService,
		c.RequestBinder,
		errorHandler,
	)
	
	executionHandler := handlers.NewExecutionHandler(
		c.WorkflowService,
		c.RequestBinder,
		errorHandler,
	)
	
	systemHandler := handlers.NewSystemHandler(
		c,
		c.WorkflowService,
		c.AIGenerationService,
		c.ExecutionEngine,
		errorHandler,
	)
	
	aiHandler := handlers.NewAIHandler(
		c.AIGenerationService,
		c.WorkflowService,
		c.RequestBinder,
		errorHandler,
	)
	
	integrationHandler := handlers.NewIntegrationHandler(
		c.IntegrationService,
		c.WorkflowService,
		c.RequestBinder,
		errorHandler,
	)
	
	return &Router{
		container:          c,
		errorHandler:       errorHandler,
		workflowHandler:    workflowHandler,
		executionHandler:   executionHandler,
		systemHandler:      systemHandler,
		aiHandler:          aiHandler,
		integrationHandler: integrationHandler,
	}
}

// SetupRoutes creates and configures the HTTP router with all routes
func (r *Router) SetupRoutes() *mux.Router {
	router := mux.NewRouter()
	
	// Apply middleware in optimal order
	router.Use(r.recoveryMiddleware)              // First: catch panics
	router.Use(middleware.CORSMiddleware)         // Second: handle CORS early
	router.Use(middleware.AuthMiddleware)         // Third: authenticate requests
	router.Use(middleware.LoggingMiddleware)      // Fourth: log authenticated requests
	router.Use(middleware.PerformanceMiddleware)  // Fifth: add performance timing
	router.Use(middleware.CacheMiddleware(r.container.Cache, 300)) // Sixth: cache responses
	router.Use(middleware.CompressionMiddleware)  // Last: compress responses
	
	// Register routes
	r.registerSystemRoutes(router)
	r.registerWorkflowRoutes(router)
	r.registerExecutionRoutes(router)
	r.registerAIRoutes(router)
	r.registerIntegrationRoutes(router)
	
	return router
}

// System routes
func (r *Router) registerSystemRoutes(router *mux.Router) {
	// Health and status
	router.HandleFunc("/health", r.wrap(r.systemHandler.HealthCheck)).Methods("GET")
	router.HandleFunc("/health/full", r.wrap(r.systemHandler.HealthCheckFull)).Methods("GET")
	router.HandleFunc("/version", r.wrap(r.systemHandler.GetVersion)).Methods("GET")
	
	// System information
	router.HandleFunc("/platforms", r.wrap(r.systemHandler.GetPlatforms)).Methods("GET")
	router.HandleFunc("/models", r.wrap(r.systemHandler.GetModels)).Methods("GET")
	router.HandleFunc("/stats", r.wrap(r.systemHandler.GetStats)).Methods("GET")
	
	// System monitoring
	router.HandleFunc("/circuit-breakers", r.wrap(r.systemHandler.GetCircuitBreakerStats)).Methods("GET")
	router.HandleFunc("/cache", r.wrap(r.systemHandler.GetCacheStats)).Methods("GET")
	
	// Documentation
	router.HandleFunc("/openapi.json", r.wrap(r.systemHandler.GetOpenAPISpec)).Methods("GET")
	router.HandleFunc("/docs", r.redirectToDocs).Methods("GET")
}

// Workflow management routes
func (r *Router) registerWorkflowRoutes(router *mux.Router) {
	// CRUD operations
	router.HandleFunc("/workflows", r.wrap(r.workflowHandler.ListWorkflows)).Methods("GET")
	router.HandleFunc("/workflows", r.wrap(r.workflowHandler.CreateWorkflow)).Methods("POST")
	router.HandleFunc("/workflows/{id}", r.wrap(r.workflowHandler.GetWorkflow)).Methods("GET")
	router.HandleFunc("/workflows/{id}", r.wrap(r.workflowHandler.UpdateWorkflow)).Methods("PUT")
	router.HandleFunc("/workflows/{id}", r.wrap(r.workflowHandler.DeleteWorkflow)).Methods("DELETE")
	
	// Workflow operations
	router.HandleFunc("/workflows/search", r.wrap(r.workflowHandler.SearchWorkflows)).Methods("GET")
	router.HandleFunc("/workflows/{id}/clone", r.wrap(r.workflowHandler.CloneWorkflow)).Methods("POST")
	
	// Workflow metrics and history
	router.HandleFunc("/workflows/{id}/metrics", r.wrap(r.workflowHandler.GetWorkflowMetrics)).Methods("GET")
	router.HandleFunc("/workflows/{id}/history", r.wrap(r.workflowHandler.GetExecutionHistory)).Methods("GET")
}

// Execution routes
func (r *Router) registerExecutionRoutes(router *mux.Router) {
	// Workflow execution
	router.HandleFunc("/workflows/{id}/execute", r.wrap(r.executionHandler.ExecuteWorkflow)).Methods("POST")
	router.HandleFunc("/workflows/{id}/execute/async", r.wrap(r.executionHandler.ExecuteWorkflowAsync)).Methods("POST")
	router.HandleFunc("/workflows/{id}/executions/{execution_id}", r.wrap(r.executionHandler.GetExecutionStatus)).Methods("GET")
	
	// Workflow validation and estimation
	router.HandleFunc("/workflows/{id}/validate", r.wrap(r.executionHandler.ValidateWorkflow)).Methods("POST")
	router.HandleFunc("/workflows/{id}/estimate", r.wrap(r.executionHandler.EstimateExecutionTime)).Methods("POST")
}

// AI-powered routes
func (r *Router) registerAIRoutes(router *mux.Router) {
	// AI generation
	router.HandleFunc("/ai/generate", r.wrap(r.aiHandler.GenerateWorkflow)).Methods("POST")
	router.HandleFunc("/ai/generate/batch", r.wrap(r.aiHandler.GenerateWorkflowBatch)).Methods("POST")
	
	// Model operations
	router.HandleFunc("/ai/models", r.wrap(r.aiHandler.ListModels)).Methods("GET")
	router.HandleFunc("/ai/models/{model}/validate", r.wrap(r.aiHandler.ValidateModel)).Methods("POST")
	
	// Advanced AI features (placeholders for future)
	router.HandleFunc("/ai/improve/{id}", r.wrap(r.aiHandler.ImproveWorkflow)).Methods("POST")
	router.HandleFunc("/ai/explain/{id}", r.wrap(r.aiHandler.ExplainWorkflow)).Methods("POST")
}

// Integration routes
func (r *Router) registerIntegrationRoutes(router *mux.Router) {
	// Import/Export
	router.HandleFunc("/integration/import", r.wrap(r.integrationHandler.ImportWorkflow)).Methods("POST")
	router.HandleFunc("/integration/import/batch", r.wrap(r.integrationHandler.BatchImport)).Methods("POST")
	router.HandleFunc("/workflows/{id}/export", r.wrap(r.integrationHandler.ExportWorkflow)).Methods("GET")
	
	// Platform integration
	router.HandleFunc("/integration/platforms", r.wrap(r.integrationHandler.GetPlatformStatus)).Methods("GET")
	router.HandleFunc("/integration/validate", r.wrap(r.integrationHandler.ValidateImportData)).Methods("POST")
	router.HandleFunc("/integration/export/formats", r.wrap(r.integrationHandler.GetExportFormats)).Methods("GET")
}

// Middleware and helper methods

// wrap wraps handlers with error handling
func (r *Router) wrap(handler handlers.ErrorHandlerFunc) http.HandlerFunc {
	return r.errorHandler.WrapGenericErrorHandler(handler)
}

// recoveryMiddleware wraps the error handler's recovery middleware
func (r *Router) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				var err error
				if e, ok := recovered.(error); ok {
					err = errors.NewInternalError("panic recovered", e)
				} else {
					err = errors.NewInternalError("panic recovered", nil).
						WithDetail("panic_value", recovered)
				}
				r.errorHandler.HandleError(w, req, err)
			}
		}()
		
		next.ServeHTTP(w, req)
	})
}

// redirectToDocs handles documentation redirect
func (r *Router) redirectToDocs(w http.ResponseWriter, req *http.Request) {
	// In a real implementation, this would redirect to Swagger UI or similar
	response := map[string]interface{}{
		"message": "API Documentation",
		"links": map[string]string{
			"openapi_spec": "/openapi.json",
			"health":       "/health",
			"version":      "/version",
		},
		"endpoints": map[string]interface{}{
			"workflows":   "/workflows",
			"execution":   "/workflows/{id}/execute",
			"ai":          "/ai/generate",
			"integration": "/integration/import",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	handlers.WriteJSON(w, http.StatusOK, response)
}

// GetRouter returns a configured router (convenience method)
func GetRouter(c *container.Container) *mux.Router {
	r := NewRouter(c)
	return r.SetupRoutes()
}