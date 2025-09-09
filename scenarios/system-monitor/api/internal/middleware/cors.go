package middleware

import (
	"net/http"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORS creates a configurable CORS middleware
func NewCORS(config CORSConfig) func(http.Handler) http.Handler {
	// Set defaults if not provided
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Content-Type", "Authorization", "X-Requested-With", "Accept"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if allowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			}
			
			// Set other CORS headers
			w.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods))
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders))
			
			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", joinStrings(config.ExposedHeaders))
			}
			
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			
			w.Header().Set("Access-Control-Max-Age", string(config.MaxAge))
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// joinStrings joins a slice of strings with comma separator
func joinStrings(strs []string) string {
	result := ""
	for i, str := range strs {
		if i > 0 {
			result += ", "
		}
		result += str
	}
	return result
}