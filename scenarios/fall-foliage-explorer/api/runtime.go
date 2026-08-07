package main

import (
	"context"
	"database/sql"
	"encoding/json"
	schema "fall-foliage-explorer/internal/foliage"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/server"
)

var db *sql.DB

func writeJSON(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logError("failed to write JSON response", "error", err)
	}
}

func logInfo(message string, fields ...interface{}) {
	writeStructuredLog("info", message, fields...)
}

func logWarn(message string, fields ...interface{}) {
	writeStructuredLog("warn", message, fields...)
}

func logError(message string, fields ...interface{}) {
	writeStructuredLog("error", message, fields...)
}

func writeStructuredLog(level, message string, fields ...interface{}) {
	entry := map[string]interface{}{
		"level":     level,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok || key == "" {
			continue
		}
		switch value := fields[i+1].(type) {
		case error:
			entry[key] = value.Error()
		default:
			entry[key] = value
		}
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"level":"%s","message":%q}`+"\n", level, message)
		return
	}
	fmt.Fprintln(os.Stderr, string(encoded))
}

func initDB() error {
	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(schema.Schema)); err != nil {
		return fmt.Errorf("schema initialization failed: %w", err)
	}

	logInfo("database connection established")
	return nil
}

func runServer() error {
	if err := initDB(); err != nil {
		logWarn("database initialization failed; running in mock data mode", "error", err)
	}

	return server.Run(server.Config{
		Handler: buildMux(),
		Cleanup: closeDB,
	})
}

func buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", enableCORS(healthHandler))
	mux.HandleFunc("/api/regions", enableCORS(regionsHandler))
	mux.HandleFunc("/api/foliage", enableCORS(foliageHandler))
	mux.HandleFunc("/api/predict", enableCORS(predictHandler))
	mux.HandleFunc("/api/weather", enableCORS(weatherHandler))
	mux.HandleFunc("/api/reports", enableCORS(reportsHandler))
	mux.HandleFunc("/api/trips", enableCORS(tripsHandler))
	return mux
}

func closeDB(ctx context.Context) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := allowedCORSOrigin(r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "0")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func allowedCORSOrigin(origin string) string {
	if origin == "" {
		return ""
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}

	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return origin
	default:
		return ""
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health.New().Version("1.0.0").Check(health.DB(db), health.Optional).Handler()(w, r)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
