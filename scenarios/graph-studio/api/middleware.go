package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

var previewTokenFallbackOnce sync.Once

// DatabaseMiddleware injects database connection into context
func DatabaseMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs request details
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only errors and slow requests
		latency := time.Since(start)
		if c.Writer.Status() >= 400 || latency > 1*time.Second {
			if raw != "" {
				path = path + "?" + raw
			}

			log.Printf("[%s] %s %s %d %v",
				c.GetString("request_id"),
				c.Request.Method,
				path,
				c.Writer.Status(),
				latency,
			)
		}
	}
}

// UserContextMiddleware extracts user information from request
func UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from header or token
		// For now, we'll use a header-based approach
		// In production, this should validate JWT tokens
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			// Generate anonymous user ID for this session
			userID = "anon-" + uuid.New().String()[:8]
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// ErrorHandlerMiddleware handles panics and errors
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(500, ErrorResponse{
					Error:   "Internal server error",
					Details: "An unexpected error occurred",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.JSON(504, ErrorResponse{
				Error:   "Request timeout",
				Details: "The request took too long to process",
			})
			c.Abort()
		}
	}
}

// getDB extracts database connection from context
func getDB(c *gin.Context) *sql.DB {
	db, exists := c.Get("db")
	if !exists {
		panic("database not found in context")
	}
	return db.(*sql.DB)
}

// getUserID extracts user ID from context
func getUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return "system"
	}
	return userID.(string)
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks in legacy browsers
		c.Header("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		frameAncestors := allowedFrameAncestors()
		csp := fmt.Sprintf(
			"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors %s",
			strings.Join(frameAncestors, " "),
		)
		c.Header("Content-Security-Policy", csp)

		// Permissions Policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

func allowedFrameAncestors() []string {
	defaults := []string{"'self'", "http://localhost:*", "http://127.0.0.1:*", "http://[::1]:*"}
	extra := strings.Fields(os.Getenv("FRAME_ANCESTORS"))
	seen := make(map[string]struct{}, len(defaults)+len(extra))
	allow := make([]string, 0, len(defaults)+len(extra))
	for _, candidate := range append(defaults, extra...) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		allow = append(allow, trimmed)
	}
	return allow
}

// PreviewAccessMiddleware requires a valid preview token for all API requests when configured.
func PreviewAccessMiddleware() gin.HandlerFunc {
	tokens, requireToken := resolvePreviewTokens()
	if len(tokens) == 0 {
		if !requireToken {
			return func(c *gin.Context) {
				c.Next()
			}
		}

		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, ErrorResponse{
				Error:   "Preview token not configured",
				Details: "Graph Studio preview requires PREVIEW_ACCESS_TOKEN (or compatible) to be set before serving requests.",
			})
		}
	}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		if isLocalHealthRequest(c.Request) {
			c.Next()
			return
		}

		provided := strings.TrimSpace(c.GetHeader("X-Preview-Token"))
		if provided == "" {
			if cookie, err := c.Cookie("preview_token"); err == nil {
				provided = strings.TrimSpace(cookie)
			}
		}

		if !matchesPreviewToken(provided, tokens) {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error:   "Preview access denied",
				Details: "Graph Studio API requires a valid preview token (X-Preview-Token header or preview_token cookie).",
			})
			return
		}

		c.Next()
	}
}

func resolvePreviewTokens() ([]string, bool) {
	requireToken := shouldRequirePreviewToken()
	envKeys := []string{
		"PREVIEW_ACCESS_TOKEN",
		"GRAPH_STUDIO_PREVIEW_TOKEN",
		"VITE_PREVIEW_ACCESS_TOKEN",
		"APP_MONITOR_PREVIEW_TOKEN",
		"APP_MONITOR_PREVIEW_TOKENS",
	}

	unique := make(map[string]struct{})
	for _, key := range envKeys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		for _, segment := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(segment)
			if trimmed == "" {
				continue
			}
			unique[trimmed] = struct{}{}
		}
	}

	for _, path := range previewTokenFileCandidates() {
		for _, token := range readPreviewTokensFromFile(path) {
			unique[token] = struct{}{}
		}
	}

	if len(unique) == 0 {
		if requireToken {
			previewTokenFallbackOnce.Do(func() {
				log.Printf("Preview access: managed lifecycle detected but no preview token configured (set PREVIEW_ACCESS_TOKEN or APP_MONITOR_PREVIEW_TOKEN)")
			})
		}
		// Fall back to open access when no tokens are provisioned to avoid blocking local development/tests.
		return nil, false
	}

	tokens := make([]string, 0, len(unique))
	for token := range unique {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	return tokens, requireToken
}

func previewTokenFileCandidates() []string {
	candidates := make([]string, 0, 5)
	for _, key := range []string{"PREVIEW_TOKEN_FILE", "APP_MONITOR_PREVIEW_TOKEN_FILE", "GRAPH_STUDIO_PREVIEW_TOKEN_FILE"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			candidates = append(candidates, value)
		}
	}

	if root := strings.TrimSpace(os.Getenv("SCENARIO_ROOT")); root != "" {
		candidates = append(candidates, filepath.Join(root, "tmp", "preview-token"))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		scenarioRoot := filepath.Clean(filepath.Join(exeDir, ".."))
		candidates = append(candidates,
			filepath.Join(scenarioRoot, "tmp", "preview-token"),
			filepath.Join(exeDir, "preview-token"),
		)
	}

	return dedupeStrings(candidates)
}

func readPreviewTokensFromFile(path string) []string {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	tokens := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		tokens = append(tokens, trimmed)
	}
	return tokens
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			continue
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}

func isLifecycleManaged() bool {
	if truthyEnv("VROOLI_LIFECYCLE_MANAGED") {
		return true
	}

	switch strings.TrimSpace(strings.ToLower(os.Getenv("VROOLI_LIFECYCLE"))) {
	case "active", "preview", "managed", "production", "staging":
		return true
	}

	if truthyEnv("SCENARIO_MODE") {
		return true
	}

	switch strings.TrimSpace(strings.ToLower(os.Getenv("VROOLI_PHASE"))) {
	case "active", "preview", "managed", "production", "staging", "develop", "development":
		return true
	}

	for _, key := range []string{
		"SCENARIO_NAME",
		"VROOLI_SCENARIO",
		"VROOLI_SCENARIO_NAME",
		"APP_MONITOR_SCENARIO",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}

	return false
}

func shouldRequirePreviewToken() bool {
	if truthyEnv("GRAPH_STUDIO_DISABLE_PREVIEW_GUARD") {
		return false
	}
	if truthyEnv("GRAPH_STUDIO_FORCE_PREVIEW_GUARD") {
		return true
	}
	return isLifecycleManaged()
}

func truthyEnv(key string) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(key))) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

func matchesPreviewToken(provided string, tokens []string) bool {
	if provided == "" {
		return false
	}
	for _, token := range tokens {
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

func isLocalHealthRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Method != http.MethodGet {
		return false
	}
	switch r.URL.Path {
	case "/health", "/health/detailed":
	default:
		return false
	}

	remoteHost := strings.TrimSpace(r.RemoteAddr)
	if remoteHost == "" {
		return false
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		remoteHost = strings.Split(forwarded, ",")[0]
	}

	host := remoteHost
	if strings.Contains(remoteHost, ":") {
		if h, _, err := net.SplitHostPort(remoteHost); err == nil {
			host = h
		}
	}

	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// RequestSizeLimitMiddleware limits the size of incoming requests
func RequestSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = &limitedReader{
			reader:   c.Request.Body,
			maxBytes: maxBytes,
		}
		c.Next()
	}
}

// limitedReader wraps request body to enforce size limit
type limitedReader struct {
	reader interface {
		Read([]byte) (int, error)
		Close() error
	}
	maxBytes int64
	n        int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.n >= lr.maxBytes {
		return 0, &requestTooLargeError{}
	}

	// Limit read size
	maxRead := lr.maxBytes - lr.n
	if int64(len(p)) > maxRead {
		p = p[0:maxRead]
	}

	n, err = lr.reader.Read(p)
	lr.n += int64(n)

	if lr.n > lr.maxBytes {
		return n, &requestTooLargeError{}
	}

	return n, err
}

func (lr *limitedReader) Close() error {
	return lr.reader.Close()
}

type requestTooLargeError struct{}

func (e *requestTooLargeError) Error() string {
	return "request body too large"
}

// RateLimitMiddleware implements rate limiting per IP address
func RateLimitMiddleware(requestsPerSecond int, burst int) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Cleanup old clients every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
			}
		}
		clients[ip].lastSeen = time.Now()
		limiter := clients[ip].limiter
		mu.Unlock()

		if !limiter.Allow() {
			c.JSON(429, ErrorResponse{
				Error:   "Rate limit exceeded",
				Details: "Too many requests, please slow down",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
