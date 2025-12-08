package isolation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Config controls how isolation resources are provisioned for Playbooks.
type Config struct {
	ScenarioName    string
	RequirePostgres bool
	RequireRedis    bool
	Retain          bool
	LogWriter       io.Writer
	Timeout         time.Duration
}

// ResourceInfo describes a provisioned resource for observability and retention.
type ResourceInfo struct {
	Name            string
	Endpoint        string
	InspectCommands []string
}

// Result captures the env overrides and cleanup logic produced by Prepare.
type Result struct {
	RunID     string
	Env       map[string]string
	Resources []ResourceInfo
	Cleanup   func(ctx context.Context) error
}

type Manager struct {
	cfg Config
}

func NewManager(cfg Config) *Manager {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 90 * time.Second
	}
	return &Manager{cfg: cfg}
}

// ShouldRetainFromEnv returns true when the retain flag is set for debugging.
func ShouldRetainFromEnv() bool {
	return os.Getenv("TEST_GENIE_PLAYBOOKS_RETAIN") == "1"
}

// Prepare provisions ephemeral Postgres + Redis endpoints and returns the env
// vars needed to target them. Containers are stopped unless Retain is true.
func (m *Manager) Prepare(ctx context.Context) (*Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, m.cfg.Timeout)
	defer cancel()

	runID := uuid.NewString()
	env := map[string]string{
		"TEST_GENIE_PLAYBOOKS":        "1",
		"TEST_GENIE_RUN_ID":           runID,
		"TEST_GENIE_PLAYBOOKS_RETAIN": boolToString(m.cfg.Retain),
	}
	resources := make([]ResourceInfo, 0, 2)
	cleanups := make([]func(context.Context) error, 0, 2)

	if m.cfg.RequirePostgres {
		pg, pgCleanup, err := m.startPostgres(runCtx, runID)
		if err != nil {
			return nil, fmt.Errorf("postgres isolation failed: %w", err)
		}
		resources = append(resources, pg.info)
		env = merge(env, pg.env)
		cleanups = append(cleanups, pgCleanup)
	}

	if m.cfg.RequireRedis {
		rd, redisCleanup, err := m.startRedis(runCtx, runID)
		if err != nil {
			// Ensure postgres is cleaned if redis fails
			for _, fn := range cleanups {
				_ = fn(context.Background())
			}
			return nil, fmt.Errorf("redis isolation failed: %w", err)
		}
		resources = append(resources, rd.info)
		env = merge(env, rd.env)
		cleanups = append(cleanups, redisCleanup)
	}

	cleanupAll := func(c context.Context) error {
		if m.cfg.Retain {
			return nil
		}
		var combinedErr error
		for _, fn := range cleanups {
			if err := fn(c); err != nil && combinedErr == nil {
				combinedErr = err
			}
		}
		return combinedErr
	}

	return &Result{
		RunID:     runID,
		Env:       env,
		Resources: resources,
		Cleanup:   cleanupAll,
	}, nil
}

type startResult struct {
	env  map[string]string
	info ResourceInfo
}

func (m *Manager) startPostgres(ctx context.Context, runID string) (*startResult, func(context.Context) error, error) {
	password, err := randomHex(12)
	if err != nil {
		return nil, nil, err
	}
	dbName := buildDBName(m.cfg.ScenarioName, runID)
	user := "testgenie"

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		Env:          map[string]string{"POSTGRES_USER": user, "POSTGRES_PASSWORD": password, "POSTGRES_DB": dbName},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(45 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return m.startPostgresFallback(ctx, runID, fmt.Errorf("start postgres container: %w", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(context.Background())
		return m.startPostgresFallback(ctx, runID, fmt.Errorf("resolve postgres host: %w", err))
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(context.Background())
		return m.startPostgresFallback(ctx, runID, fmt.Errorf("resolve postgres port: %w", err))
	}

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port.Port(), dbName)
	env := map[string]string{
		"DATABASE_URL":           url,
		"POSTGRES_URL":           url,
		"PGHOST":                 host,
		"POSTGRES_HOST":          host,
		"PGPORT":                 port.Port(),
		"POSTGRES_PORT":          port.Port(),
		"PGUSER":                 user,
		"POSTGRES_USER":          user,
		"PGPASSWORD":             password,
		"POSTGRES_PASSWORD":      password,
		"PGDATABASE":             dbName,
		"POSTGRES_DB":            dbName,
		"POSTGRES_SSLMODE":       "disable",
		"PLAYBOOKS_DATABASE_URL": url,
	}

	info := ResourceInfo{
		Name:     "postgres",
		Endpoint: url,
		InspectCommands: []string{
			fmt.Sprintf("psql %s", url),
		},
	}

	cleanup := func(c context.Context) error {
		return container.Terminate(c)
	}

	if m.cfg.Retain {
		cleanup = func(context.Context) error { return nil }
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

func (m *Manager) startRedis(ctx context.Context, runID string) (*startResult, func(context.Context) error, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return m.startRedisFallback(ctx, runID, fmt.Errorf("start redis container: %w", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(context.Background())
		return m.startRedisFallback(ctx, runID, fmt.Errorf("resolve redis host: %w", err))
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(context.Background())
		return m.startRedisFallback(ctx, runID, fmt.Errorf("resolve redis port: %w", err))
	}

	url := fmt.Sprintf("redis://%s:%s", host, port.Port())
	env := map[string]string{
		"REDIS_HOST":          host,
		"REDIS_PORT":          port.Port(),
		"REDIS_URL":           url,
		"REDIS_DB":            "0",
		"PLAYBOOKS_REDIS_URL": url,
	}

	info := ResourceInfo{
		Name:     "redis",
		Endpoint: url,
		InspectCommands: []string{
			fmt.Sprintf("redis-cli -u %s", url),
		},
	}

	cleanup := func(c context.Context) error {
		return container.Terminate(c)
	}

	if m.cfg.Retain {
		cleanup = func(context.Context) error { return nil }
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func merge(dst, src map[string]string) map[string]string {
	out := make(map[string]string, len(dst)+len(src))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		out[k] = v
	}
	return out
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "scenario"
	}
	builder := strings.Builder{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	return builder.String()
}

func buildDBName(scenario, runID string) string {
	base := fmt.Sprintf("tg_pb_%s_%s", sanitize(scenario), sanitize(runID))
	if len(base) <= 60 {
		return base
	}

	hash := randomSuffix(runID)
	maxBase := 60 - len(hash) - 1
	if maxBase < 1 {
		maxBase = 1
	}
	trimmed := strings.TrimRight(base[:maxBase], "_-")
	return fmt.Sprintf("%s_%s", trimmed, hash)
}

func randomSuffix(source string) string {
	safe := sanitize(source)
	if len(safe) >= 6 {
		return safe[:6]
	}
	h, _ := randomHex(3)
	return h
}

// startPostgresFallback creates an ephemeral database on an existing Postgres host
// when testcontainers is unavailable. It requires DATABASE_URL or POSTGRES_URL in env.
func (m *Manager) startPostgresFallback(ctx context.Context, runID string, rootErr error) (*startResult, func(context.Context) error, error) {
	baseURL := strings.TrimSpace(firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_URL")))
	if baseURL == "" {
		return nil, nil, fmt.Errorf("no postgres base URL available for fallback: %w", rootErr)
	}

	parsed, err := parsePostgresURL(baseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse postgres url: %w", err)
	}

	dbName := buildDBName(m.cfg.ScenarioName, runID)
	createCmd := execCommandContext(m.cfg.LogWriter, ctx, "psql", parsed.withDatabase("postgres"), "-v", "ON_ERROR_STOP=1", "-c", fmt.Sprintf("CREATE DATABASE %q TEMPLATE template0", dbName))
	if createCmd != nil {
		if err := createCmd(); err != nil {
			return nil, nil, fmt.Errorf("create temp database via psql: %w", err)
		}
	}

	url := parsed.withDatabase(dbName)
	env := map[string]string{
		"DATABASE_URL":           url,
		"POSTGRES_URL":           url,
		"PGHOST":                 parsed.host,
		"POSTGRES_HOST":          parsed.host,
		"PGPORT":                 parsed.port,
		"POSTGRES_PORT":          parsed.port,
		"PGUSER":                 parsed.user,
		"POSTGRES_USER":          parsed.user,
		"PGPASSWORD":             parsed.password,
		"POSTGRES_PASSWORD":      parsed.password,
		"PGDATABASE":             dbName,
		"POSTGRES_DB":            dbName,
		"POSTGRES_SSLMODE":       parsed.sslmode,
		"PLAYBOOKS_DATABASE_URL": url,
	}

	info := ResourceInfo{
		Name:     "postgres",
		Endpoint: url,
		InspectCommands: []string{
			fmt.Sprintf("psql %s", url),
		},
	}

	cleanup := func(c context.Context) error {
		if m.cfg.Retain {
			return nil
		}
		dropCmd := execCommandContext(m.cfg.LogWriter, c, "psql", parsed.withDatabase("postgres"), "-v", "ON_ERROR_STOP=1", "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %q", dbName))
		if dropCmd != nil {
			return dropCmd()
		}
		return nil
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

// startRedisFallback targets an existing Redis by carving out a dedicated DB index.
func (m *Manager) startRedisFallback(ctx context.Context, runID string, rootErr error) (*startResult, func(context.Context) error, error) {
	baseURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if baseURL == "" {
		host := firstNonEmpty(os.Getenv("REDIS_HOST"), "127.0.0.1")
		port := firstNonEmpty(os.Getenv("REDIS_PORT"), "6379")
		baseURL = fmt.Sprintf("redis://%s:%s", host, port)
	}
	// Derive a DB index from runID (0-15)
	dbIdx := deriveRedisDB(runID)
	baseHost, basePort := parseRedisHostPort(baseURL)
	redisURL := fmt.Sprintf("redis://%s:%s/%d", baseHost, basePort, dbIdx)

	env := map[string]string{
		"REDIS_URL":           redisURL,
		"REDIS_HOST":          baseHost,
		"REDIS_PORT":          basePort,
		"REDIS_DB":            fmt.Sprintf("%d", dbIdx),
		"PLAYBOOKS_REDIS_URL": redisURL,
	}

	info := ResourceInfo{
		Name:     "redis",
		Endpoint: redisURL,
		InspectCommands: []string{
			fmt.Sprintf("redis-cli -u %s", redisURL),
		},
	}

	cleanup := func(c context.Context) error {
		if m.cfg.Retain {
			return nil
		}
		return runRedisCLI(m.cfg.LogWriter, c, redisURL, "FLUSHDB")
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

type parsedPostgres struct {
	host     string
	port     string
	user     string
	password string
	db       string
	sslmode  string
}

func (p parsedPostgres) withDatabase(db string) string {
	userInfo := p.user
	if p.password != "" {
		userInfo = fmt.Sprintf("%s:%s", p.user, p.password)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", userInfo, p.host, p.port, db, p.sslmode)
}

func parsePostgresURL(raw string) (parsedPostgres, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return parsedPostgres{}, err
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	user := u.User.Username()
	pw, _ := u.User.Password()
	db := strings.TrimPrefix(u.Path, "/")
	if db == "" {
		db = "postgres"
	}
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}
	return parsedPostgres{host: host, port: port, user: user, password: pw, db: db, sslmode: sslmode}, nil
}

func deriveRedisDB(runID string) int {
	sum := 0
	for _, r := range runID {
		sum += int(r)
	}
	return sum%15 + 1 // avoid DB 0 in shared environments
}

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func parsePortFromRedisURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "6379"
	}
	if port := u.Port(); port != "" {
		return port
	}
	return "6379"
}

func parseRedisHostPort(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return "127.0.0.1", "6379"
	}
	host := u.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}
	port := u.Port()
	if port == "" {
		port = "6379"
	}
	return host, port
}

// execCommandContext returns a closure that runs the command with env preserved
// rather than importing extra dependencies.
func execCommandContext(logWriter io.Writer, ctx context.Context, binary string, databaseURL string, args ...string) func() error {
	allArgs := append([]string{"-d", databaseURL}, args...)
	return func() error {
		cmd := exec.CommandContext(ctx, binary, allArgs...)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		return cmd.Run()
	}
}

func runRedisCLI(logWriter io.Writer, ctx context.Context, url string, cmd string) error {
	c := exec.CommandContext(ctx, "redis-cli", "-u", url, cmd)
	c.Stdout = logWriter
	c.Stderr = logWriter
	return c.Run()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
