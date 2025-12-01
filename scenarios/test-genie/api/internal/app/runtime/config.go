package runtime

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config captures runtime parameters that should not be hard-coded inside HTTP handlers.
type Config struct {
	Port          string
	DatabaseURL   string
	ScenariosRoot string
}

// LoadConfig gathers lifecycle-provided environment variables and resolves derived paths.
func LoadConfig() (*Config, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	scenariosRoot, err := resolveScenariosRoot()
	if err != nil {
		return nil, err
	}

	port, err := requireEnv("API_PORT")
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		ScenariosRoot: scenariosRoot,
	}, nil
}

func requireEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value, nil
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func resolveScenariosRoot() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("SCENARIOS_ROOT")); raw != "" {
		return filepath.Abs(raw)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	scenarioDir := filepath.Dir(wd)
	root := filepath.Dir(scenarioDir)
	return root, nil
}
