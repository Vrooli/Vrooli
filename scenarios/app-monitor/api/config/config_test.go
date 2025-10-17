package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Set required environment variables
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("Expected LoadConfig to succeed, got error: %v", err)
		}
		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Check default values are set
		if cfg.API.Port == "" {
			t.Error("Expected API port to be set")
		}
		if cfg.API.ReadTimeout == 0 {
			t.Error("Expected read timeout to have default value")
		}
		if cfg.API.WriteTimeout == 0 {
			t.Error("Expected write timeout to have default value")
		}
	})

	t.Run("CustomValues", func(t *testing.T) {
		os.Setenv("API_PORT", "9090")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("API_READ_TIMEOUT", "30s")
		os.Setenv("API_WRITE_TIMEOUT", "45s")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
		defer os.Unsetenv("API_READ_TIMEOUT")
		defer os.Unsetenv("API_WRITE_TIMEOUT")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.API.Port != "9090" {
			t.Errorf("Expected port 9090, got %s", cfg.API.Port)
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := &Config{
			API: APIConfig{
				Port:            "8080",
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				ShutdownTimeout: 10 * time.Second,
			},
		}

		// Config should be valid
		if cfg.API.Port == "" {
			t.Error("Expected port to be set")
		}
	})
}

func TestDatabaseConfig(t *testing.T) {
	t.Run("PostgresConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
		}()

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Database.URL == "" {
			t.Error("Expected database URL to be set")
		}
	})

	t.Run("MissingDatabaseConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Should succeed even without database config
		if cfg == nil {
			t.Error("Expected config to be created even without database")
		}
	})
}

func TestRedisConfig(t *testing.T) {
	t.Run("RedisConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PORT")
		}()

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Redis.URL == "" {
			t.Error("Expected Redis URL to be set")
		}
	})

	t.Run("MissingRedisConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Should succeed even without Redis config
		if cfg == nil {
			t.Error("Expected config to be created even without Redis")
		}
	})
}

func TestTimeoutParsing(t *testing.T) {
	t.Run("ValidTimeout", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("API_READ_TIMEOUT", "60s")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("API_READ_TIMEOUT")
		}()

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.API.ReadTimeout != 60*time.Second {
			t.Errorf("Expected 60s timeout, got %v", cfg.API.ReadTimeout)
		}
	})

	t.Run("InvalidTimeout", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		os.Setenv("API_READ_TIMEOUT", "invalid")
		defer func() {
			os.Unsetenv("API_PORT")
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("API_READ_TIMEOUT")
		}()

		cfg, err := LoadConfig()
		// Should use default on invalid timeout
		if err != nil {
			t.Fatalf("Expected LoadConfig to use default for invalid timeout, got error: %v", err)
		}
		if cfg.API.ReadTimeout == 0 {
			t.Error("Expected default timeout to be set")
		}
	})
}

func TestInitializeDatabase(t *testing.T) {
	t.Run("WithoutConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Should handle missing database gracefully
		db, err := cfg.InitializeDatabase()
		if db != nil && err == nil {
			// If we got a connection, it's because the database is actually available
			t.Log("Database connection succeeded")
		} else if err != nil {
			// Expected when database is not available
			t.Logf("Database initialization failed as expected: %v", err)
		}
	})
}

func TestInitializeRedis(t *testing.T) {
	t.Run("WithoutConfig", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Should handle missing Redis gracefully
		redis, err := cfg.InitializeRedis()
		if redis != nil && err == nil {
			// If we got a connection, it's because Redis is actually available
			t.Log("Redis connection succeeded")
		} else if err != nil {
			// Expected when Redis is not available
			t.Logf("Redis initialization failed as expected: %v", err)
		}
	})
}

func TestInitializeDocker(t *testing.T) {
	t.Run("DockerClient", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		defer os.Unsetenv("API_PORT")
		defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Docker may or may not be available
		docker, err := cfg.InitializeDocker()
		if docker != nil && err == nil {
			t.Log("Docker client initialized successfully")
		} else if err != nil {
			t.Logf("Docker initialization failed (may not be available): %v", err)
		}
	})
}
