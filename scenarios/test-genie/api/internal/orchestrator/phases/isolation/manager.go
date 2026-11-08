package isolation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
)

// Config controls how isolation resources are provisioned for Playbooks.
type Config struct {
	ScenarioName    string
	RequirePostgres bool
	RequireRedis    bool
	RequireSQLite   bool
	SQLiteEnvVars   []string
	Retain          bool
	LogWriter       io.Writer
	Timeout         time.Duration
	RunID           string
	OnAllocation    func(Allocation) error
	AllocationRoot  string
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

// Prepare provisions ephemeral backing services and returns the env vars needed
// to target them. Containers/files are cleaned up unless Retain is true.
func (m *Manager) Prepare(ctx context.Context) (*Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, m.cfg.Timeout)
	defer cancel()

	runID := m.cfg.RunID
	if runID == "" {
		runID = uuid.NewString()
	}
	if filepath.Base(runID) != runID || strings.ContainsAny(runID, "/\\") || runID == "." || runID == ".." {
		return nil, errors.New("invalid seed allocation identity")
	}
	env := map[string]string{
		"TEST_GENIE_PLAYBOOKS":        "1",
		"TEST_GENIE_RUN_ID":           runID,
		"TEST_GENIE_PLAYBOOKS_RETAIN": boolToString(m.cfg.Retain),
	}
	resources := make([]ResourceInfo, 0, 3)
	cleanups := make([]func(context.Context) error, 0, 3)
	result := &Result{RunID: runID, Env: env}
	result.Cleanup = func(c context.Context) error {
		if m.cfg.Retain {
			return nil
		}
		var combinedErr error
		for i, fn := range cleanups {
			if fn == nil {
				continue
			}
			if err := fn(c); err != nil {
				combinedErr = errors.Join(combinedErr, err)
			} else {
				cleanups[i] = nil
			}
		}
		return combinedErr
	}
	add := func(resource *startResult, cleanup func(context.Context) error) {
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
		if resource != nil {
			resources = append(resources, resource.info)
			env = merge(env, resource.env)
			result.Resources, result.Env = resources, env
		}
	}

	if m.cfg.RequirePostgres {
		pg, pgCleanup, err := m.startPostgres(runCtx, runID)
		add(pg, pgCleanup)
		if err != nil {
			return result, fmt.Errorf("postgres isolation failed: %w", err)
		}
	}

	if m.cfg.RequireRedis {
		rd, redisCleanup, err := m.startRedis(runCtx, runID)
		add(rd, redisCleanup)
		if err != nil {
			return result, fmt.Errorf("redis isolation failed: %w", err)
		}
	}

	if m.cfg.RequireSQLite {
		sqliteResult, sqliteCleanup, err := m.startSQLite(runCtx, runID)
		add(sqliteResult, sqliteCleanup)
		if err != nil {
			return result, fmt.Errorf("sqlite isolation failed: %w", err)
		}
	}
	return result, nil
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

	host, port, cleanup, err := m.startPrivateContainer(ctx, runID, "postgres", "postgres:16-alpine", "5432/tcp", []string{"POSTGRES_USER=" + user, "POSTGRES_PASSWORD=" + password, "POSTGRES_DB=" + dbName})
	if err != nil {
		if cleanup != nil {
			return nil, cleanup, err
		}
		return m.startPostgresFallback(ctx, runID, err)
	}

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
	env := map[string]string{
		"DATABASE_URL":           url,
		"POSTGRES_URL":           url,
		"PGHOST":                 host,
		"POSTGRES_HOST":          host,
		"PGPORT":                 port,
		"POSTGRES_PORT":          port,
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

	if m.cfg.Retain {
		cleanup = func(context.Context) error { return nil }
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

func (m *Manager) startRedis(ctx context.Context, runID string) (*startResult, func(context.Context) error, error) {
	host, port, cleanup, err := m.startPrivateContainer(ctx, runID, "redis", "redis:7-alpine", "6379/tcp", nil)
	if err != nil {
		return nil, cleanup, fmt.Errorf("private redis container unavailable (shared Redis DB indexes cannot prove isolation): %w", err)
	}

	url := fmt.Sprintf("redis://%s:%s", host, port)
	env := map[string]string{
		"REDIS_HOST":          host,
		"REDIS_PORT":          port,
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

	if m.cfg.Retain {
		cleanup = func(context.Context) error { return nil }
	}

	return &startResult{env: env, info: info}, cleanup, nil
}

// startSQLite leases the scenario under test an isolated storage tree for the
// duration of a run.
//
// Isolation is expressed as a storage ROOT rather than a database PATH, and the
// difference is the whole point. Injecting SQLITE_PATH and SQLITE_DB named one
// specific file, which meant (a) every scenario that inherited the pair opened
// that one file, and (b) the mechanism only isolated SQLite, leaving the
// scenario's other storage classes pointed at production. VROOLI_STORAGE_ROOT
// is scenario-agnostic: every scenario beneath it resolves to its OWN path
// within the tree, so a leaked value isolates rather than collides, and the
// whole storage surface moves together.
func (m *Manager) startSQLite(ctx context.Context, runID string) (*startResult, func(context.Context) error, error) {
	parent := m.cfg.AllocationRoot
	temporaryParent := parent == ""
	if parent == "" {
		var err error
		parent, err = os.MkdirTemp("", "test-genie-sqlite-owner-")
		if err != nil {
			return nil, nil, err
		}
	} else if err := os.MkdirAll(parent, 0700); err != nil {
		return nil, nil, err
	}
	transferred := false
	defer func() {
		if temporaryParent && !transferred {
			// No allocation was published. Remove only empty directories created
			// for this invocation; never recursively remove an unexpected entry.
			_ = os.Remove(filepath.Join(parent, ".staging"))
			_ = os.Remove(parent)
		}
	}()
	rootDir := filepath.Join(parent, "test-genie-playbooks-"+runID)
	if _, err := os.Lstat(rootDir); err == nil {
		return nil, nil, errors.New("private sqlite allocation name already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	staging := filepath.Join(parent, ".staging", runID)
	allocation := Allocation{Kind: "sqlite", RunID: runID, Root: rootDir, StagingRoot: staging}
	if err := ensureSQLiteStagingNamespace(parent); err != nil {
		return nil, nil, err
	}
	if _, err := os.Lstat(staging); err == nil {
		return nil, nil, errors.New("private sqlite staging identity already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	if err := m.track(allocation); err != nil {
		return nil, nil, err
	}
	cleanup := func(context.Context) error {
		if m.cfg.Retain {
			return nil
		}
		if temporaryParent {
			if _, err := os.Lstat(parent); errors.Is(err, os.ErrNotExist) {
				return nil
			}
		}
		if err := RetireAllocation(context.Background(), allocation); err != nil {
			return err
		}
		if temporaryParent {
			if err := os.Remove(filepath.Join(parent, ".staging")); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return os.Remove(parent)
		}
		return nil
	}
	transferred = true
	if err := prepareSQLiteDirectory(allocation, nil); err != nil {
		return nil, cleanup, err
	}

	// Resolve the path the scenario under test will itself resolve, using the
	// same seam it uses. Reporting a path derived any other way would let the
	// two drift, and an inspect command that opens the wrong file is worse than
	// no inspect command.
	//
	// ScenarioName is sanitized first: it may arrive as a display name
	// ("Test Genie"), and the storage seam requires a scenario identifier —
	// alphanumerics plus '-', '_' and '.'. sanitize is the same normalization
	// used for the temp directory above, so the two agree.
	dbPath, err := storage.SQLitePath(storage.SQLiteConfig{
		Scenario:     sanitize(m.cfg.ScenarioName),
		RootOverride: rootDir,
	})
	if err != nil {
		return nil, cleanup, fmt.Errorf("resolve isolated sqlite path: %w", err)
	}
	dsn, err := storage.SQLiteDSNAt(dbPath, storage.SQLiteTuning{})
	if err != nil {
		return nil, cleanup, fmt.Errorf("build isolated sqlite dsn: %w", err)
	}

	env := map[string]string{
		"VROOLI_STORAGE_ROOT":   rootDir,
		"SQLITE_DATABASE_PATH":  filepath.Dir(dbPath),
		"PLAYBOOKS_SQLITE_PATH": dbPath,
		"PLAYBOOKS_SQLITE_DSN":  dsn,
	}
	if !m.cfg.RequirePostgres {
		env["DATABASE_URL"] = dsn
		env["PLAYBOOKS_DATABASE_URL"] = dsn
	}
	// A scenario may declare its own explicitly named path variable. Those are
	// scenario-scoped by construction, so they cannot collide across scenarios
	// the way the generic pair did.
	for _, key := range m.cfg.SQLiteEnvVars {
		env[key] = dbPath
	}

	info := ResourceInfo{
		Name:     "sqlite",
		Endpoint: dbPath,
		InspectCommands: []string{
			fmt.Sprintf("sqlite3 %s", dbPath),
		},
	}

	select {
	case <-ctx.Done():
		return &startResult{env: env, info: info}, cleanup, ctx.Err()
	default:
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
	allocation := Allocation{Kind: "postgres-database", RunID: runID, Name: dbName, AdminURL: parsed.withDatabase("postgres")}
	if err := m.track(allocation); err != nil {
		return nil, nil, err
	}
	createCmd := execCommandContext(ctx, m.cfg.LogWriter, "psql", parsed.withDatabase("postgres"), "-v", "ON_ERROR_STOP=1", "-c", fmt.Sprintf("CREATE DATABASE %q TEMPLATE template0", dbName))
	if createCmd != nil {
		if err := createCmd(); err != nil {
			return nil, func(c context.Context) error { return RetireAllocation(c, allocation) }, fmt.Errorf("create temp database via psql: %w", err)
		}
	}
	markerCmd := execCommandContext(ctx, m.cfg.LogWriter, "psql", parsed.withDatabase("postgres"), "-v", "ON_ERROR_STOP=1", "-c", fmt.Sprintf("COMMENT ON DATABASE %q IS 'test-genie-seed:%s'", dbName, runID))
	if err := markerCmd(); err != nil {
		return nil, func(c context.Context) error { return RetireAllocation(c, allocation) }, fmt.Errorf("mark private postgres owner: %w", err)
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
		return RetireAllocation(c, allocation)
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

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// execCommandContext returns a closure that runs the command with env preserved
// rather than importing extra dependencies.
func execCommandContext(ctx context.Context, logWriter io.Writer, binary string, databaseURL string, args ...string) func() error {
	allArgs := append([]string{"-d", databaseURL}, args...)
	return func() error {
		cmd := exec.CommandContext(ctx, binary, allArgs...)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		return cmd.Run()
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
