package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"metareasoning-api/internal/container"
	httpRouter "metareasoning-api/internal/interfaces/http"
)

func main() {
	log.Println("Starting Metareasoning API v3.0.0...")
	
	// Initialize dependency container
	c, err := container.NewContainer()
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}
	defer func() {
		if closeErr := c.Close(); closeErr != nil {
			log.Printf("Error closing container: %v", closeErr)
		}
	}()
	
	log.Printf("Configuration loaded - Port: %s", c.Config.Port)
	log.Println("Database connected")
	log.Println("Services initialized")
	
	// Setup routes with clean handler architecture
	router := httpRouter.GetRouter(c)
	
	// Start server
	server := &http.Server{
		Addr:         ":" + c.Config.Port,
		Handler:      router,
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
		
		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during graceful shutdown: %v", err)
			if closeErr := server.Close(); closeErr != nil {
				log.Printf("Error during force close: %v", closeErr)
			}
		} else {
			log.Println("Server shutdown gracefully")
		}
	}()
	
	// Start listening
	log.Printf("Server listening on port %s", c.Config.Port)
	log.Printf("API documentation available at http://localhost:%s/docs", c.Config.Port)
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

