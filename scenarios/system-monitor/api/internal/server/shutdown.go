package server

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"system-monitor-api/internal/services"
)

func waitForShutdown(monitorSvc *services.MonitorService, investigationSvc *services.InvestigationService, srv *http.Server, repo io.Closer) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	investigationSvc.Shutdown()
	monitorSvc.Stop()

	if err := srv.Close(); err != nil {
		slog.Error("Error closing server", "error", err)
	}

	if repo != nil {
		if err := repo.Close(); err != nil {
			slog.Error("Error closing repository", "error", err)
		}
	}

	slog.Info("Server shutdown complete")
}
