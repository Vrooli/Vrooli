package main

import (
	"agent-inbox/handlers"
	"agent-inbox/middleware"

	"github.com/gorilla/mux"
)

// registerRoutes sets up the HTTP router with all middleware and route handlers.
func registerRoutes(h *handlers.Handlers, uploadHandlers *handlers.UploadHandlers) *mux.Router {
	router := mux.NewRouter()
	router.Use(middleware.RequestID) // Request ID first for tracing
	router.Use(middleware.Logging)
	router.Use(middleware.CORS)
	h.RegisterRoutes(router)
	uploadHandlers.RegisterRoutes(router)
	return router
}
