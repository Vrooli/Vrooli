package server

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"system-monitor-api/internal/services"
)

func waitForShutdown(monitorSvc *services.MonitorService, srv *http.Server, db *sql.DB) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	monitorSvc.Stop()

	if err := srv.Close(); err != nil {
		log.Printf("Error closing server: %v", err)
	}

	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	log.Println("Server shutdown complete")
}
